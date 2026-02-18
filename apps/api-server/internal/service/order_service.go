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

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

var (
	ErrOrderNotFound     = errors.New("order not found")
	ErrInvalidTransition = errors.New("invalid status transition")
	ErrUnknownStatus     = errors.New("unknown status")
)

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
}

func NewOrderService(
	orderRepo repository.OrderRepo,
	auditRepo repository.AuditRepo,
	tenantRepo repository.TenantRepo,
	pool *pgxpool.Pool,
	emailService *EmailService,
	webhookDispatch *WebhookDispatchService,
) *OrderService {
	return &OrderService{
		orderRepo:       orderRepo,
		auditRepo:       auditRepo,
		tenantRepo:      tenantRepo,
		pool:            pool,
		emailService:    emailService,
		webhookDispatch: webhookDispatch,
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
		if err := s.orderRepo.Create(ctx, tx, order); err != nil {
			return err
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
			_ = database.WithTenant(context.Background(), s.pool, tenantID, func(tx pgx.Tx) error {
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
						available := stock.Quantity - stock.Reserved
						if available <= 0 {
							continue
						}
						reserveQty := min(remaining, available)
						_, _ = tx.Exec(ctx,
							`UPDATE warehouse_stock SET reserved = reserved + $1, updated_at = NOW()
							 WHERE warehouse_id = $2 AND product_id = $3 AND variant_id IS NULL`,
							reserveQty, stock.WarehouseID, productID)
						remaining -= reserveQty
					}
				}
				return nil
			})
			s.triggerStockSync(tenantID, productQtys, "order_placed")
		}
	}

	go s.webhookDispatch.Dispatch(context.Background(), tenantID, "order.created", order)
	FireAutomationEvent(s.automationService, tenantID, "order", "order.created", order.ID, map[string]any{
		"status": order.Status, "source": order.Source,
		"customer_name": order.CustomerName, "total_amount": order.TotalAmount,
		"currency": order.Currency, "payment_status": order.PaymentStatus,
	})

	// Auto-create shipment if requested (best effort — never fails order creation)
	if req.AutoCreateShipment && req.ShipmentProvider != nil && *req.ShipmentProvider != "" && s.shipmentService != nil {
		go func() {
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
		}()
	}

	return order, nil
}

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
		go s.webhookDispatch.Dispatch(context.Background(), tenantID, "order.updated", order)
		FireAutomationEvent(s.automationService, tenantID, "order", "order.updated", order.ID, map[string]any{
			"status": order.Status, "source": order.Source,
			"customer_name": order.CustomerName, "total_amount": order.TotalAmount,
			"currency": order.Currency, "payment_status": order.PaymentStatus,
		})
	}
	return order, err
}

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
		go s.webhookDispatch.Dispatch(context.Background(), tenantID, "order.deleted", map[string]any{"order_id": orderID.String()})
	}
	return err
}

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

		if req.Force {
			// Force mode: skip transition validation
		} else {
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
		go s.emailService.SendOrderStatusEmail(context.Background(), tenantID, order, oldStatus, req.Status)
		go s.webhookDispatch.Dispatch(context.Background(), tenantID, "order.status_changed", map[string]any{"order_id": orderID.String(), "from": oldStatus, "to": req.Status})
		if s.invoiceService != nil {
			go s.invoiceService.HandleOrderStatusChange(context.Background(), tenantID, order)
		}
		if s.smsService != nil {
			go s.smsService.SendOrderStatusSMS(context.Background(), tenantID, order, oldStatus, req.Status)
		}
		FireAutomationEvent(s.automationService, tenantID, "order", "order.status_changed", order.ID, map[string]any{
			"status": order.Status, "old_status": oldStatus, "new_status": req.Status,
			"source": order.Source, "customer_name": order.CustomerName,
			"total_amount": order.TotalAmount, "currency": order.Currency,
			"payment_status": order.PaymentStatus,
		})
		// Auto-sync fulfillment status to Allegro (async, best-effort)
		if s.allegroSync != nil && order.Source == "allegro" {
			go s.allegroSync.SyncFulfillmentStatus(context.Background(), tenantID, order, req.Status)
		}
		// Stock sync on status change
		if req.Status == "shipped" {
			go s.handleStockOnShip(context.Background(), tenantID, order)
		} else if req.Status == "cancelled" {
			go s.handleStockOnCancel(context.Background(), tenantID, order)
		}
	}
	return order, err
}

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

		for _, orderID := range req.OrderIDs {
			result := model.BulkStatusResult{OrderID: orderID}

			existing, err := s.orderRepo.FindByID(ctx, tx, orderID)
			if err != nil || existing == nil {
				result.Error = "order not found"
				resp.Results = append(resp.Results, result)
				resp.Failed++
				continue
			}

			var setShippedAt, setDeliveredAt *time.Time

			if req.Force {
				// Force mode: skip transition validation
			} else {
				if !config.CanTransition(existing.Status, req.Status) {
					result.Error = fmt.Sprintf("invalid transition: %s -> %s", existing.Status, req.Status)
					resp.Results = append(resp.Results, result)
					resp.Failed++
					continue
				}
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
		go s.emailService.SendOrderStatusEmail(context.Background(), tenantID, n.order, n.oldStatus, n.newStatus)
		if s.smsService != nil {
			go s.smsService.SendOrderStatusSMS(context.Background(), tenantID, n.order, n.oldStatus, n.newStatus)
		}
		// Auto-sync fulfillment status to Allegro (async, best-effort)
		if s.allegroSync != nil && n.order.Source == "allegro" {
			go s.allegroSync.SyncFulfillmentStatus(context.Background(), tenantID, n.order, n.newStatus)
		}
		// Stock sync on status change
		if n.newStatus == "shipped" {
			go s.handleStockOnShip(context.Background(), tenantID, n.order)
		} else if n.newStatus == "cancelled" {
			go s.handleStockOnCancel(context.Background(), tenantID, n.order)
		}
	}
	for _, n := range pendingWebhooks {
		go s.webhookDispatch.Dispatch(context.Background(), tenantID, "order.status_changed", map[string]any{"order_id": n.orderID.String(), "from": n.oldStatus, "to": n.newStatus})
	}

	return resp, nil
}

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

// reserveStockForOrder increments reserved count in warehouse_stock for each product in order items.
// Best-effort: errors are logged but don't fail the order.
func (s *OrderService) reserveStockForOrder(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID, items json.RawMessage) map[uuid.UUID]int {
	if s.warehouseStockRepo == nil {
		return nil
	}
	productQtys := extractProductQuantities(items)
	for productID, qty := range productQtys {
		stocks, err := s.warehouseStockRepo.ListByProduct(ctx, tx, productID)
		if err != nil || len(stocks) == 0 {
			continue
		}
		// Reserve from first warehouse with stock
		for _, stock := range stocks {
			available := stock.Quantity - stock.Reserved
			if available <= 0 {
				continue
			}
			reserveQty := min(qty, available)
			if err := s.warehouseStockRepo.AdjustQuantity(ctx, tx, stock.WarehouseID, productID, nil, 0); err != nil {
				continue
			}
			// Direct SQL to increment reserved
			_, _ = tx.Exec(ctx,
				`UPDATE warehouse_stock SET reserved = reserved + $1, updated_at = NOW()
				 WHERE warehouse_id = $2 AND product_id = $3 AND variant_id IS NULL`,
				reserveQty, stock.WarehouseID, productID)
			qty -= reserveQty
			if qty <= 0 {
				break
			}
		}
	}
	return productQtys
}

// triggerStockSync fires OnStockChange for each product in the given map.
func (s *OrderService) triggerStockSync(tenantID uuid.UUID, productQtys map[uuid.UUID]int, trigger string) {
	if s.stockSyncService == nil {
		return
	}
	for productID, qty := range productQtys {
		go s.stockSyncService.OnStockChange(context.Background(), tenantID, productID, trigger, qty, qty)
	}
}

// handleStockOnShip decrements quantity and reserved in warehouse_stock for shipped orders.
func (s *OrderService) handleStockOnShip(ctx context.Context, tenantID uuid.UUID, order *model.Order) {
	if s.warehouseStockRepo == nil {
		return
	}
	productQtys := extractProductQuantities(order.Items)
	_ = database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
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
				_, _ = tx.Exec(ctx,
					`UPDATE warehouse_stock
					 SET quantity = GREATEST(quantity - $1, 0),
					     reserved = GREATEST(reserved - $1, 0),
					     updated_at = NOW()
					 WHERE warehouse_id = $2 AND product_id = $3 AND variant_id IS NULL`,
					deduct, stock.WarehouseID, productID)
				remaining -= deduct
			}
		}
		return nil
	})
	s.triggerStockSync(tenantID, productQtys, "order_shipped")
}

// handleStockOnCancel releases reserved stock for cancelled orders.
func (s *OrderService) handleStockOnCancel(ctx context.Context, tenantID uuid.UUID, order *model.Order) {
	if s.warehouseStockRepo == nil {
		return
	}
	productQtys := extractProductQuantities(order.Items)
	_ = database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
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
				_, _ = tx.Exec(ctx,
					`UPDATE warehouse_stock SET reserved = GREATEST(reserved - $1, 0), updated_at = NOW()
					 WHERE warehouse_id = $2 AND product_id = $3 AND variant_id IS NULL`,
					release, stock.WarehouseID, productID)
				remaining -= release
			}
		}
		return nil
	})
	s.triggerStockSync(tenantID, productQtys, "order_cancelled")
}

func stringOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
