package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/asyncutil"
	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

// Order service sentinel errors.
var (
	ErrOrderNotFound      = errors.New("order not found")
	ErrInvalidTransition  = errors.New("invalid status transition")
	ErrUnknownStatus      = errors.New("unknown status")
	ErrOrderLimitExceeded = errors.New("monthly order limit exceeded")
)

// OrderService handles business logic for order management.
type OrderService struct {
	orderRepo          repository.OrderRepo
	auditRepo          repository.AuditRepo
	tenantRepo         repository.TenantRepo
	warehouseStockRepo repository.WarehouseStockRepo
	pool               *pgxpool.Pool
	emailService       *EmailService
	webhookDispatch    *WebhookDispatchService
	invoiceService     *InvoiceService
	smsService         *SMSService
	automationService  *AutomationService
	shipmentService    *ShipmentService
	allegroSync        *AllegroSyncService
	stockSyncService   *StockSyncService
	fulfillment        *FulfillmentService
}

// NewOrderService creates a new OrderService. fulfillment may be nil (or a
// disabled FulfillmentService), in which case order creation does not spin up a
// fulfillment process — preserving the pre-OPE-416 behavior.
func NewOrderService(
	orderRepo repository.OrderRepo,
	auditRepo repository.AuditRepo,
	tenantRepo repository.TenantRepo,
	pool *pgxpool.Pool,
	emailService *EmailService,
	webhookDispatch *WebhookDispatchService,
	fulfillment *FulfillmentService,
) *OrderService {
	return &OrderService{
		orderRepo:       orderRepo,
		auditRepo:       auditRepo,
		tenantRepo:      tenantRepo,
		pool:            pool,
		emailService:    emailService,
		webhookDispatch: webhookDispatch,
		fulfillment:     fulfillment,
	}
}

// OrderRepo returns the underlying order repository for direct access.
func (s *OrderService) OrderRepo() repository.OrderRepo {
	return s.orderRepo
}

// AuditRepo returns the underlying audit repository for direct access.
func (s *OrderService) AuditRepo() repository.AuditRepo {
	return s.auditRepo
}

// MonthlyOrderCounter provides the order count needed for plan-limit enforcement.
type MonthlyOrderCounter interface {
	CountThisMonth(ctx context.Context, tx pgx.Tx) (int, error)
}

// EnforceMonthlyOrderLimit serializes per-tenant monthly order limit checks inside
// the caller's transaction. pendingCreatesBeforeCurrent is the number of creates
// already in flight but not visible to CountThisMonth; it excludes the create
// the caller is about to perform.
func EnforceMonthlyOrderLimit(ctx context.Context, tx pgx.Tx, orderCounter MonthlyOrderCounter, tenantID uuid.UUID, maxOrdersMonthly, pendingCreatesBeforeCurrent int) error {
	if maxOrdersMonthly <= 0 {
		return nil
	}
	if pendingCreatesBeforeCurrent < 0 {
		pendingCreatesBeforeCurrent = 0
	}

	if _, err := tx.Exec(ctx, "SELECT 1 FROM tenants WHERE id = $1 FOR UPDATE", tenantID); err != nil {
		return fmt.Errorf("lock tenant for order limit check: %w", err)
	}

	count, err := orderCounter.CountThisMonth(ctx, tx)
	if err != nil {
		return fmt.Errorf("count orders for limit check: %w", err)
	}
	if count+pendingCreatesBeforeCurrent >= maxOrdersMonthly {
		return ErrOrderLimitExceeded
	}
	return nil
}

// WebhookDispatch returns the webhook dispatch service for direct access.
func (s *OrderService) WebhookDispatch() *WebhookDispatchService {
	return s.webhookDispatch
}

// SetInvoiceService sets the invoice service for auto-invoicing on status change.
// Called after both services are constructed to avoid circular dependency.
func (s *OrderService) SetInvoiceService(invoiceSvc *InvoiceService) {
	s.invoiceService = invoiceSvc
}

// SetAutomationService sets the automation service for rule processing.
// Called after construction to avoid circular dependency.
func (s *OrderService) SetAutomationService(automationSvc *AutomationService) {
	s.automationService = automationSvc
}

// SetSMSService sets the SMS service for sending SMS notifications on status change.
func (s *OrderService) SetSMSService(smsSvc *SMSService) {
	s.smsService = smsSvc
}

// SetShipmentService sets the shipment service for auto-creating shipments with orders.
func (s *OrderService) SetShipmentService(shipmentSvc *ShipmentService) {
	s.shipmentService = shipmentSvc
}

// SetAllegroSyncService sets the Allegro sync service for auto-syncing fulfillment status.
// Called after construction to avoid circular dependency.
func (s *OrderService) SetAllegroSyncService(allegroSync *AllegroSyncService) {
	s.allegroSync = allegroSync
}

// SetStockSyncService sets the stock sync service for real-time stock synchronization.
// Called after construction to avoid circular dependency.
func (s *OrderService) SetStockSyncService(stockSync *StockSyncService) {
	s.stockSyncService = stockSync
}

// SetWarehouseStockRepo sets the warehouse stock repo for stock reservation/decrement.
func (s *OrderService) SetWarehouseStockRepo(repo repository.WarehouseStockRepo) {
	s.warehouseStockRepo = repo
}

func (s *OrderService) loadStatusConfig(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID) (*model.OrderStatusConfig, error) {
	settings, err := s.tenantRepo.GetSettings(ctx, tx, tenantID)
	if err != nil {
		return nil, err
	}

	if settings != nil {
		var allSettings map[string]json.RawMessage
		if err := json.Unmarshal(settings, &allSettings); err == nil {
			if raw, ok := allSettings["order_statuses"]; ok {
				var config model.OrderStatusConfig
				if err := json.Unmarshal(raw, &config); err != nil {
					slog.Warn("failed to unmarshal order status config", "error", err)
				} else if len(config.Statuses) > 0 {
					return &config, nil
				}
			}
		}
	}

	cfg := model.DefaultOrderStatusConfig()
	return &cfg, nil
}

// List returns a paginated list of orders for a tenant.
func (s *OrderService) List(ctx context.Context, tenantID uuid.UUID, filter model.OrderListFilter) (model.ListResponse[model.Order], error) {
	var resp model.ListResponse[model.Order]
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		orders, total, err := s.orderRepo.List(ctx, tx, filter)
		if err != nil {
			return err
		}
		if orders == nil {
			orders = []model.Order{}
		}
		resp = model.ListResponse[model.Order]{
			Items:  orders,
			Total:  total,
			Limit:  filter.Limit,
			Offset: filter.Offset,
		}
		return nil
	})
	return resp, err
}

// Get returns a single order by ID.
func (s *OrderService) Get(ctx context.Context, tenantID, orderID uuid.UUID) (*model.Order, error) {
	var order *model.Order
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var err error
		order, err = s.orderRepo.FindByID(ctx, tx, orderID)
		return err
	})
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, ErrOrderNotFound
	}
	return order, nil
}

// Create inserts a new order.
func (s *OrderService) Create(ctx context.Context, tenantID uuid.UUID, req model.CreateOrderRequest, actorID uuid.UUID, ip string) (*model.Order, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	// Sanitize user-facing text fields to prevent stored XSS
	req.CustomerName = model.StripHTMLTags(req.CustomerName)
	if req.Notes != nil {
		sanitized := model.StripHTMLTags(*req.Notes)
		req.Notes = &sanitized
	}

	// Default NOT NULL jsonb fields to avoid inserting NULL
	shippingAddr := req.ShippingAddress
	if shippingAddr == nil {
		shippingAddr = json.RawMessage("{}")
	}
	items := req.Items
	if items == nil {
		items = json.RawMessage("[]")
	}
	metadata := req.Metadata
	if metadata == nil {
		metadata = json.RawMessage("{}")
	}
	orderedAt := req.OrderedAt
	if orderedAt == nil {
		now := time.Now()
		orderedAt = &now
	}

	tags := req.Tags
	if tags == nil {
		tags = []string{}
	}

	order := &model.Order{
		ID:              uuid.New(),
		TenantID:        tenantID,
		ExternalID:      req.ExternalID,
		Source:          req.Source,
		IntegrationID:   req.IntegrationID,
		Status:          "new",
		CustomerName:    req.CustomerName,
		CustomerEmail:   req.CustomerEmail,
		CustomerPhone:   req.CustomerPhone,
		ShippingAddress: shippingAddr,
		BillingAddress:  req.BillingAddress,
		Items:           items,
		TotalAmount:     req.TotalAmount,
		Currency:        req.Currency,
		Notes:           req.Notes,
		Metadata:        metadata,
		Tags:            tags,
		DeliveryMethod:  req.DeliveryMethod,
		PickupPointID:   req.PickupPointID,
		OrderedAt:       orderedAt,
		InternalNotes:   req.InternalNotes,
		Priority:        req.Priority,
	}

	if req.PaymentStatus != nil {
		order.PaymentStatus = *req.PaymentStatus
	} else {
		order.PaymentStatus = "pending"
	}
	if req.PaymentMethod != nil {
		order.PaymentMethod = req.PaymentMethod
	}

	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		if err := EnforceMonthlyOrderLimit(ctx, tx, s.orderRepo, tenantID, req.MaxOrdersMonthly, 0); err != nil {
			return err
		}

		if err := s.orderRepo.Create(ctx, tx, order); err != nil {
			return err
		}
		// Route the new order through the fulfillment commands (OPE-416): create
		// its fulfillment process + enqueue the orchestration event in the SAME
		// transaction. No-op when the gated service is nil/disabled.
		if s.fulfillment != nil {
			if _, err := s.fulfillment.EnsureProcessForOrder(ctx, tx, tenantID, order.ID); err != nil {
				return err
			}
		}
		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "order.created",
			EntityType: "order",
			EntityID:   order.ID,
			Changes:    map[string]string{"source": req.Source, "customer_name": req.CustomerName},
			IPAddress:  ip,
		})
	})
	if err != nil {
		return nil, err
	}

	// Reserve stock for order items (best-effort, async stock sync)
	if s.warehouseStockRepo != nil {
		productQtys := extractProductQuantities(order.Items)
		if len(productQtys) > 0 {
			bgCtx := context.Background()
			if err := database.WithTenant(bgCtx, s.pool, tenantID, func(tx pgx.Tx) error {
				for productID, qty := range productQtys {
					stocks, err := s.warehouseStockRepo.ListByProduct(bgCtx, tx, productID)
					if err != nil || len(stocks) == 0 {
						continue
					}
					remaining := qty
					for _, stock := range stocks {
						if remaining <= 0 {
							break
						}
						available := stock.Quantity - stock.Reserved
						if available <= 0 {
							continue
						}
						reserveQty := min(remaining, available)
						if _, err := tx.Exec(bgCtx,
							`UPDATE warehouse_stock SET reserved = reserved + $1, updated_at = NOW()
							 WHERE warehouse_id = $2 AND product_id = $3 AND variant_id IS NULL`,
							reserveQty, stock.WarehouseID, productID); err != nil {
							slog.Error("failed to reserve stock", "error", err, "product_id", productID, "warehouse_id", stock.WarehouseID)
						}
						remaining -= reserveQty
					}
				}
				return nil
			}); err != nil {
				slog.Error("failed to reserve stock for order", "error", err, "tenant_id", tenantID)
			}
			s.triggerStockSync(tenantID, productQtys, "order_placed")
		}
	}

	if s.webhookDispatch != nil {
		asyncutil.SafeGo(func() { s.webhookDispatch.Dispatch(context.Background(), tenantID, "order.created", order) })
	}
	FireAutomationEvent(s.automationService, tenantID, "order", "order.created", order.ID, map[string]any{
		"status": order.Status, "source": order.Source,
		"customer_name": order.CustomerName, "total_amount": order.TotalAmount,
		"currency": order.Currency, "payment_status": order.PaymentStatus,
	})

	// Auto-create shipment if requested (best effort — never fails order creation)
	if req.AutoCreateShipment && req.ShipmentProvider != nil && *req.ShipmentProvider != "" && s.shipmentService != nil {
		asyncutil.SafeGo(func() {
			shipReq := model.CreateShipmentRequest{
				OrderID:  order.ID,
				Provider: *req.ShipmentProvider,
			}
			// Include pickup_point_id as carrier_data.target_point if present
			if req.PickupPointID != nil && *req.PickupPointID != "" {
				cd, _ := json.Marshal(map[string]string{"target_point": *req.PickupPointID})
				shipReq.CarrierData = cd
			}
			if _, err := s.shipmentService.Create(context.Background(), tenantID, shipReq, actorID, ip); err != nil {
				slog.Error("auto-create shipment failed", "order_id", order.ID, "provider", *req.ShipmentProvider, "error", err)
			}
		})
	}

	return order, nil
}

// Update modifies an existing order.
func (s *OrderService) Update(ctx context.Context, tenantID, orderID uuid.UUID, req model.UpdateOrderRequest, actorID uuid.UUID, ip string) (*model.Order, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	var order *model.Order
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		existing, err := s.orderRepo.FindByID(ctx, tx, orderID)
		if err != nil {
			return err
		}
		if existing == nil {
			return ErrOrderNotFound
		}

		if err := s.orderRepo.Update(ctx, tx, orderID, req); err != nil {
			return err
		}

		order, err = s.orderRepo.FindByID(ctx, tx, orderID)
		if err != nil {
			return err
		}

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "order.updated",
			EntityType: "order",
			EntityID:   orderID,
			IPAddress:  ip,
		})
	})
	if err == nil && order != nil {
		if s.webhookDispatch != nil {
			asyncutil.SafeGo(func() { s.webhookDispatch.Dispatch(context.Background(), tenantID, "order.updated", order) })
		}
		FireAutomationEvent(s.automationService, tenantID, "order", "order.updated", order.ID, map[string]any{
			"status": order.Status, "source": order.Source,
			"customer_name": order.CustomerName, "total_amount": order.TotalAmount,
			"currency": order.Currency, "payment_status": order.PaymentStatus,
		})
	}
	return order, err
}

// Delete removes an order by ID.
func (s *OrderService) Delete(ctx context.Context, tenantID, orderID, actorID uuid.UUID, ip string) error {
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		order, err := s.orderRepo.FindByID(ctx, tx, orderID)
		if err != nil {
			return err
		}
		if order == nil {
			return ErrOrderNotFound
		}

		if err := s.orderRepo.Delete(ctx, tx, orderID); err != nil {
			return err
		}

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "order.deleted",
			EntityType: "order",
			EntityID:   orderID,
			Changes:    map[string]string{"external_id": stringOrEmpty(order.ExternalID)},
			IPAddress:  ip,
		})
	})
	if err == nil {
		if s.webhookDispatch != nil {
			asyncutil.SafeGo(func() {
				s.webhookDispatch.Dispatch(context.Background(), tenantID, "order.deleted", map[string]any{"order_id": orderID.String()})
			})
		}
	}
	return err
}

// TransitionStatus moves an order to a new status, enforcing allowed transitions.
func (s *OrderService) TransitionStatus(ctx context.Context, tenantID, orderID uuid.UUID, req model.StatusTransitionRequest, actorID uuid.UUID, ip string) (*model.Order, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	var order *model.Order
	var oldStatus string
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		existing, err := s.orderRepo.FindByID(ctx, tx, orderID)
		if err != nil {
			return err
		}
		if existing == nil {
			return ErrOrderNotFound
		}
		oldStatus = existing.Status

		config, err := s.loadStatusConfig(ctx, tx, tenantID)
		if err != nil {
			return fmt.Errorf("load status config: %w", err)
		}

		if !config.IsValidStatus(req.Status) {
			return fmt.Errorf("%w: %q", ErrUnknownStatus, req.Status)
		}

		var setShippedAt, setDeliveredAt *time.Time

		if !req.Force {
			if !config.IsValidStatus(existing.Status) {
				return fmt.Errorf("%w: current %q", ErrUnknownStatus, existing.Status)
			}
			if !config.CanTransition(existing.Status, req.Status) {
				return fmt.Errorf("%w: %s -> %s", ErrInvalidTransition, existing.Status, req.Status)
			}
		}

		// Set timestamps for special statuses
		if req.Status == "shipped" {
			now := time.Now()
			setShippedAt = &now
		}
		if req.Status == "delivered" {
			now := time.Now()
			setDeliveredAt = &now
		}

		if err := s.orderRepo.UpdateStatus(ctx, tx, orderID, req.Status, setShippedAt, setDeliveredAt); err != nil {
			return err
		}

		order, err = s.orderRepo.FindByID(ctx, tx, orderID)
		if err != nil {
			return err
		}

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "order.status_changed",
			EntityType: "order",
			EntityID:   orderID,
			Changes:    map[string]string{"from": existing.Status, "to": req.Status},
			IPAddress:  ip,
		})
	})
	if err == nil && order != nil {
		if s.emailService != nil {
			asyncutil.SafeGo(func() {
				s.emailService.SendOrderStatusEmail(context.Background(), tenantID, order, oldStatus, req.Status)
			})
		}
		if s.webhookDispatch != nil {
			asyncutil.SafeGo(func() {
				s.webhookDispatch.Dispatch(context.Background(), tenantID, "order.status_changed", map[string]any{"order_id": orderID.String(), "from": oldStatus, "to": req.Status})
			})
		}
		if s.invoiceService != nil {
			asyncutil.SafeGo(func() { s.invoiceService.HandleOrderStatusChange(context.Background(), tenantID, order) })
		}
		if s.smsService != nil {
			asyncutil.SafeGo(func() { s.smsService.SendOrderStatusSMS(context.Background(), tenantID, order, oldStatus, req.Status) })
		}
		FireAutomationEvent(s.automationService, tenantID, "order", "order.status_changed", order.ID, map[string]any{
			"status": order.Status, "old_status": oldStatus, "new_status": req.Status,
			"source": order.Source, "customer_name": order.CustomerName,
			"total_amount": order.TotalAmount, "currency": order.Currency,
			"payment_status": order.PaymentStatus,
		})
		// Auto-sync fulfillment status to Allegro (async, best-effort)
		if s.allegroSync != nil && order.Source == "allegro" {
			asyncutil.SafeGo(func() { s.allegroSync.SyncFulfillmentStatus(context.Background(), tenantID, order, req.Status) })
		}
		// Stock sync on status change
		switch req.Status {
		case "shipped":
			asyncutil.SafeGo(func() { s.handleStockOnShip(context.Background(), tenantID, order) })
		case "cancelled":
			asyncutil.SafeGo(func() { s.handleStockOnCancel(context.Background(), tenantID, order) })
		}
	}
	return order, err
}

// BulkTransitionStatus transitions multiple orders to a new status.
func (s *OrderService) BulkTransitionStatus(ctx context.Context, tenantID uuid.UUID, req model.BulkStatusTransitionRequest, actorID uuid.UUID, ip string) (*model.BulkStatusTransitionResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	resp := &model.BulkStatusTransitionResponse{
		Results: make([]model.BulkStatusResult, 0, len(req.OrderIDs)),
	}

	// Collect notifications to dispatch after the transaction commits
	type emailNotification struct {
		order     *model.Order
		oldStatus string
		newStatus string
	}
	type webhookNotification struct {
		orderID   uuid.UUID
		oldStatus string
		newStatus string
	}
	var pendingEmails []emailNotification
	var pendingWebhooks []webhookNotification

	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		config, err := s.loadStatusConfig(ctx, tx, tenantID)
		if err != nil {
			return fmt.Errorf("load status config: %w", err)
		}

		if !config.IsValidStatus(req.Status) {
			return fmt.Errorf("%w: %q", ErrUnknownStatus, req.Status)
		}

		// Batch fetch all orders in one query
		ordersMap, err := s.orderRepo.FindByIDs(ctx, tx, req.OrderIDs)
		if err != nil {
			return fmt.Errorf("batch fetch orders: %w", err)
		}

		for _, orderID := range req.OrderIDs {
			result := model.BulkStatusResult{OrderID: orderID}

			existing := ordersMap[orderID]
			if existing == nil {
				result.Error = "order not found"
				resp.Results = append(resp.Results, result)
				resp.Failed++
				continue
			}

			var setShippedAt, setDeliveredAt *time.Time

			if !req.Force && !config.CanTransition(existing.Status, req.Status) {
				result.Error = fmt.Sprintf("invalid transition: %s -> %s", existing.Status, req.Status)
				resp.Results = append(resp.Results, result)
				resp.Failed++
				continue
			}

			if req.Status == "shipped" {
				now := time.Now()
				setShippedAt = &now
			}
			if req.Status == "delivered" {
				now := time.Now()
				setDeliveredAt = &now
			}

			oldStatus := existing.Status

			if err := s.orderRepo.UpdateStatus(ctx, tx, orderID, req.Status, setShippedAt, setDeliveredAt); err != nil {
				result.Error = "failed to update status"
				resp.Results = append(resp.Results, result)
				resp.Failed++
				continue
			}

			if err := s.auditRepo.Log(ctx, tx, model.AuditEntry{
				TenantID:   tenantID,
				UserID:     actorID,
				Action:     "order.status_changed",
				EntityType: "order",
				EntityID:   orderID,
				Changes:    map[string]string{"from": existing.Status, "to": req.Status},
				IPAddress:  ip,
			}); err != nil {
				slog.Warn("bulk status transition: audit log failed, status update succeeded without audit record",
					"order_id", orderID, "error", err)
				resp.AuditFailures = append(resp.AuditFailures, orderID.String())
			}

			updated, err := s.orderRepo.FindByID(ctx, tx, orderID)
			if err == nil && updated != nil {
				pendingEmails = append(pendingEmails, emailNotification{
					order: updated, oldStatus: oldStatus, newStatus: req.Status,
				})
			}

			result.Success = true
			resp.Results = append(resp.Results, result)
			resp.Succeeded++

			pendingWebhooks = append(pendingWebhooks, webhookNotification{
				orderID: orderID, oldStatus: oldStatus, newStatus: req.Status,
			})
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Dispatch notifications outside the transaction
	for _, n := range pendingEmails {
		if s.emailService != nil {
			asyncutil.SafeGo(func() {
				s.emailService.SendOrderStatusEmail(context.Background(), tenantID, n.order, n.oldStatus, n.newStatus)
			})
		}
		if s.smsService != nil {
			asyncutil.SafeGo(func() {
				s.smsService.SendOrderStatusSMS(context.Background(), tenantID, n.order, n.oldStatus, n.newStatus)
			})
		}
		// Auto-sync fulfillment status to Allegro (async, best-effort)
		if s.allegroSync != nil && n.order.Source == "allegro" {
			asyncutil.SafeGo(func() { s.allegroSync.SyncFulfillmentStatus(context.Background(), tenantID, n.order, n.newStatus) })
		}
		// Stock sync on status change
		switch n.newStatus {
		case "shipped":
			asyncutil.SafeGo(func() { s.handleStockOnShip(context.Background(), tenantID, n.order) })
		case "cancelled":
			asyncutil.SafeGo(func() { s.handleStockOnCancel(context.Background(), tenantID, n.order) })
		}
	}
	for _, n := range pendingWebhooks {
		if s.webhookDispatch != nil {
			asyncutil.SafeGo(func() {
				s.webhookDispatch.Dispatch(context.Background(), tenantID, "order.status_changed", map[string]any{"order_id": n.orderID.String(), "from": n.oldStatus, "to": n.newStatus})
			})
		}
	}

	return resp, nil
}

// GetAudit returns the audit log for an order.
func (s *OrderService) GetAudit(ctx context.Context, tenantID, orderID uuid.UUID) ([]model.AuditLogEntry, error) {
	var entries []model.AuditLogEntry
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var err error
		entries, err = s.auditRepo.ListByEntity(ctx, tx, "order", orderID)
		return err
	})
	return entries, err
}

// extractProductQuantities parses order items JSON to extract product_id -> quantity map.
func extractProductQuantities(items json.RawMessage) map[uuid.UUID]int {
	result := make(map[uuid.UUID]int)
	if len(items) == 0 {
		return result
	}

	var parsed []struct {
		ProductID *uuid.UUID `json:"product_id"`
		Quantity  int        `json:"quantity"`
	}
	if err := json.Unmarshal(items, &parsed); err != nil {
		return result
	}
	for _, item := range parsed {
		if item.ProductID != nil && *item.ProductID != uuid.Nil && item.Quantity > 0 {
			result[*item.ProductID] += item.Quantity
		}
	}
	return result
}

// triggerStockSync fires OnStockChange for each product in the given map.
func (s *OrderService) triggerStockSync(tenantID uuid.UUID, productQtys map[uuid.UUID]int, trigger string) {
	if s.stockSyncService == nil {
		return
	}
	for productID, qty := range productQtys {
		asyncutil.SafeGo(func() { s.stockSyncService.OnStockChange(context.Background(), tenantID, productID, trigger, qty, qty) })
	}
}

// handleStockOnShip decrements quantity and reserved in warehouse_stock for shipped orders.
func (s *OrderService) handleStockOnShip(ctx context.Context, tenantID uuid.UUID, order *model.Order) {
	if s.warehouseStockRepo == nil {
		return
	}
	productQtys := extractProductQuantities(order.Items)
	if err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		for productID, qty := range productQtys {
			stocks, err := s.warehouseStockRepo.ListByProduct(ctx, tx, productID)
			if err != nil || len(stocks) == 0 {
				continue
			}
			remaining := qty
			for _, stock := range stocks {
				if remaining <= 0 {
					break
				}
				deduct := min(remaining, stock.Quantity)
				// Decrement both quantity and reserved
				if _, execErr := tx.Exec(ctx,
					`UPDATE warehouse_stock
					 SET quantity = GREATEST(quantity - $1, 0),
					     reserved = GREATEST(reserved - $1, 0),
					     updated_at = NOW()
					 WHERE warehouse_id = $2 AND product_id = $3 AND variant_id IS NULL`,
					deduct, stock.WarehouseID, productID); execErr != nil {
					slog.Error("failed to adjust warehouse stock", "warehouse_id", stock.WarehouseID, "product_id", productID, "error", execErr)
				}
				remaining -= deduct
			}
		}
		return nil
	}); err != nil {
		slog.Error("handleStockOnShip: failed to adjust stock", "error", err, "order_id", order.ID, "tenant_id", tenantID)
	}
	s.triggerStockSync(tenantID, productQtys, "order_shipped")
}

// handleStockOnCancel releases reserved stock for cancelled orders.
func (s *OrderService) handleStockOnCancel(ctx context.Context, tenantID uuid.UUID, order *model.Order) {
	if s.warehouseStockRepo == nil {
		return
	}
	productQtys := extractProductQuantities(order.Items)
	if err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		for productID, qty := range productQtys {
			stocks, err := s.warehouseStockRepo.ListByProduct(ctx, tx, productID)
			if err != nil || len(stocks) == 0 {
				continue
			}
			remaining := qty
			for _, stock := range stocks {
				if remaining <= 0 {
					break
				}
				release := min(remaining, stock.Reserved)
				if _, execErr := tx.Exec(ctx,
					`UPDATE warehouse_stock SET reserved = GREATEST(reserved - $1, 0), updated_at = NOW()
					 WHERE warehouse_id = $2 AND product_id = $3 AND variant_id IS NULL`,
					release, stock.WarehouseID, productID); execErr != nil {
					slog.Error("failed to release warehouse stock reservation", "warehouse_id", stock.WarehouseID, "product_id", productID, "error", execErr)
				}
				remaining -= release
			}
		}
		return nil
	}); err != nil {
		slog.Error("handleStockOnCancel: failed to release reserved stock", "error", err, "order_id", order.ID, "tenant_id", tenantID)
	}
	s.triggerStockSync(tenantID, productQtys, "order_cancelled")
}

// CountOrdersThisMonth returns the number of orders created in the current calendar month.
func (s *OrderService) CountOrdersThisMonth(ctx context.Context, tenantID uuid.UUID) (int, error) {
	var count int
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var e error
		count, e = s.orderRepo.CountThisMonth(ctx, tx)
		return e
	})
	return count, err
}

func stringOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
