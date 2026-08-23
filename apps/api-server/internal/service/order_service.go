package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"strings"
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
	customerRepo       repository.CustomerRepo
	priceListService   *PriceListService
	loyaltyService     *LoyaltyService
	bundleService      *BundleService
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

// SetCustomerRepo sets the customer repository so order creation can link orders to
// CRM customers (resolve-or-create by email) and keep customer order statistics.
func (s *OrderService) SetCustomerRepo(repo repository.CustomerRepo) {
	s.customerRepo = repo
}

// SetPriceListService sets the price list service so order creation can apply a linked
// B2B customer's price list to the order line items. Called after construction to avoid
// a circular dependency.
func (s *OrderService) SetPriceListService(priceListSvc *PriceListService) {
	s.priceListService = priceListSvc
}

// SetLoyaltyService sets the loyalty service so order completion accrues loyalty points
// for the linked customer. Called after construction to avoid a circular dependency.
func (s *OrderService) SetLoyaltyService(loyaltySvc *LoyaltyService) {
	s.loyaltyService = loyaltySvc
}

// SetBundleService sets the bundle service so order stock reserve/ship/cancel can expand
// bundle line items into their component products. Called after construction to avoid a
// circular dependency.
func (s *OrderService) SetBundleService(bundleSvc *BundleService) {
	s.bundleService = bundleSvc
}

// expandBundleQuantities rewrites productQtys in place: bundle product entries are removed
// (bundles are virtual and hold no warehouse_stock of their own) and replaced by their
// component product quantities, so a single warehouse_stock walk covers components AND
// normal products. No-op when the bundle service is unwired; on expansion failure it logs
// and leaves productQtys unchanged (best-effort, matching the surrounding side-effects).
func (s *OrderService) expandBundleQuantities(ctx context.Context, tx pgx.Tx, productQtys map[uuid.UUID]int) {
	if s.bundleService == nil || len(productQtys) == 0 {
		return
	}
	componentQtys, bundleIDs, err := s.bundleService.ExpandBundleComponents(ctx, tx, productQtys)
	if err != nil {
		slog.Error("failed to expand bundle components for stock adjustment", "error", err)
		return
	}
	for id := range bundleIDs {
		delete(productQtys, id)
	}
	for productID, qty := range componentQtys {
		productQtys[productID] += qty
	}
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

// findCustomerByOrderEmail resolves an existing CRM customer for the order by email,
// in a short read-only tenant transaction. Returns nil when customerRepo is unwired,
// the order carries no email, or no customer matches. Used to link orders to customers
// and to resolve a B2B price list before the order-create transaction.
func (s *OrderService) findCustomerByOrderEmail(ctx context.Context, tenantID uuid.UUID, req model.CreateOrderRequest) (*model.Customer, error) {
	if s.customerRepo == nil || req.CustomerEmail == nil {
		return nil, nil
	}
	email := strings.TrimSpace(*req.CustomerEmail)
	if email == "" {
		return nil, nil
	}
	var customer *model.Customer
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var e error
		customer, e = s.customerRepo.FindByEmail(ctx, tx, email)
		return e
	})
	return customer, err
}

// ensureOrderCustomer returns the CRM customer to link to a new order: the pre-resolved
// existing match when present, otherwise a freshly created minimal customer (email + name
// from the order). Returns nil when customerRepo is unwired or the order has no usable
// email. Runs inside the caller's order-create transaction so the customer and order rows
// commit atomically. Dedup is best-effort by email (mirrors the BaseLinker import); there
// is no unique email constraint, matching existing behavior.
func (s *OrderService) ensureOrderCustomer(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID, req model.CreateOrderRequest, existing *model.Customer) (*model.Customer, error) {
	if s.customerRepo == nil {
		return nil, nil
	}
	if existing != nil {
		return existing, nil
	}
	if req.CustomerEmail == nil {
		return nil, nil
	}
	email := strings.TrimSpace(*req.CustomerEmail)
	if email == "" {
		return nil, nil
	}
	newCustomer := &model.Customer{
		ID:       uuid.New(),
		TenantID: tenantID,
		Name:     req.CustomerName,
		Email:    &email,
		Phone:    req.CustomerPhone,
		Tags:     []string{},
	}
	if err := s.customerRepo.Create(ctx, tx, newCustomer); err != nil {
		return nil, fmt.Errorf("link order to customer: %w", err)
	}
	return newCustomer, nil
}

// applyCustomerPriceList re-prices the order's line items from the linked B2B
// customer's price list and recomputes the order total. It is a no-op unless the price
// list service is wired, the customer has a price list, and that list is active and in
// the order's currency. Each line with a product_id is repriced via CalculatePrice
// (which derives the effective price from the product's base price, never from the
// inbound line price — so the transform is idempotent and never compounds discounts);
// lines whose price cannot be resolved keep their original price (fail-soft). The order
// total is recomputed only when at least one line was actually repriced. Unknown line
// keys (name, sku, weight, custom fields) are preserved via a map round-trip.
func (s *OrderService) applyCustomerPriceList(ctx context.Context, tenantID uuid.UUID, customer *model.Customer, req *model.CreateOrderRequest) {
	if s.priceListService == nil || customer == nil || customer.PriceListID == nil {
		return
	}
	priceListID := *customer.PriceListID

	pl, err := s.priceListService.Get(ctx, tenantID, priceListID)
	if err != nil || pl == nil {
		slog.Warn("B2B pricing: price list lookup failed", "error", err, "price_list_id", priceListID, "tenant_id", tenantID)
		return
	}
	if !pl.Active {
		return
	}
	// Currency guard: never price an order's lines with a price list in another currency.
	if pl.Currency != "" && req.Currency != "" && !strings.EqualFold(pl.Currency, req.Currency) {
		slog.Warn("B2B pricing: skipped, price list currency differs from order currency",
			"price_list_currency", pl.Currency, "order_currency", req.Currency, "tenant_id", tenantID)
		return
	}

	var lines []map[string]json.RawMessage
	if err := json.Unmarshal(req.Items, &lines); err != nil || len(lines) == 0 {
		return
	}

	var total float64
	var anyPriced bool
	for _, line := range lines {
		qty := rawMessageInt(line["quantity"])
		price, hasPrice := rawMessageFloat(line["price"])
		productID := rawMessageUUID(line["product_id"])

		if productID != nil && *productID != uuid.Nil && qty > 0 {
			resp, perr := s.priceListService.CalculatePrice(ctx, tenantID, *productID, nil, qty, priceListID)
			if perr == nil {
				if pj, mErr := json.Marshal(resp.EffectivePrice); mErr == nil {
					line["price"] = pj
					price = resp.EffectivePrice
					hasPrice = true
					anyPriced = true
				}
			} else {
				slog.Warn("B2B pricing: CalculatePrice failed, keeping original line price",
					"error", perr, "product_id", *productID, "tenant_id", tenantID)
			}
		}
		if hasPrice && qty > 0 {
			total += price * float64(qty)
		}
	}

	if !anyPriced {
		return
	}
	repacked, err := json.Marshal(lines)
	if err != nil {
		return
	}
	req.Items = repacked
	req.TotalAmount = math.Round(total*100) / 100
}

// rawMessageInt parses a JSON number RawMessage into an int (truncating), returning 0
// when absent or not a number.
func rawMessageInt(raw json.RawMessage) int {
	if len(raw) == 0 {
		return 0
	}
	var f float64
	if err := json.Unmarshal(raw, &f); err != nil {
		return 0
	}
	return int(f)
}

// rawMessageFloat parses a JSON number RawMessage into a float64; the bool reports
// whether a number was present.
func rawMessageFloat(raw json.RawMessage) (float64, bool) {
	if len(raw) == 0 {
		return 0, false
	}
	var f float64
	if err := json.Unmarshal(raw, &f); err != nil {
		return 0, false
	}
	return f, true
}

// rawMessageUUID parses a JSON string RawMessage into a *uuid.UUID, returning nil when
// absent, null, the nil UUID, or not a valid UUID.
func rawMessageUUID(raw json.RawMessage) *uuid.UUID {
	if len(raw) == 0 {
		return nil
	}
	var id uuid.UUID
	if err := json.Unmarshal(raw, &id); err != nil || id == uuid.Nil {
		return nil
	}
	return &id
}

// List returns a paginated list of orders for a tenant.
func (s *OrderService) List(ctx context.Context, tenantID uuid.UUID, filter model.OrderListFilter) (model.ListResponse[model.Order], error) {
	var resp model.ListResponse[model.Order]
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		orders, total, err := s.orderRepo.List(ctx, tx, filter)
		if err != nil {
			return err
		}
		resp = model.NewListResponse(orders, total, filter.Limit, filter.Offset)
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

	// Resolve an existing CRM customer by email (read-only) before the order tx. The
	// match (when any) links the order to the customer and carries the B2B price list.
	// A read failure is non-fatal: the order is still created, just unlinked.
	linkedCustomer, custErr := s.findCustomerByOrderEmail(ctx, tenantID, req)
	if custErr != nil {
		slog.Warn("failed to resolve order customer by email", "error", custErr, "tenant_id", tenantID)
		linkedCustomer = nil
	}

	// Apply the linked B2B customer's price list to the line items + total (no-op when
	// the customer has no price list). Runs before the order is built so the adjusted
	// items/total are what gets persisted.
	s.applyCustomerPriceList(ctx, tenantID, linkedCustomer, &req)

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

	// Products whose reservation the create transaction actually applied, used to
	// trigger the marketplace stock sync once the order is committed.
	var reservedQtys map[uuid.UUID]int
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		if err := EnforceMonthlyOrderLimit(ctx, tx, s.orderRepo, tenantID, req.MaxOrdersMonthly, 0); err != nil {
			return err
		}

		// Link the order to a CRM customer (resolve-or-create by email) so downstream
		// per-customer features (stats, loyalty) have a customer to act on.
		customer, err := s.ensureOrderCustomer(ctx, tx, tenantID, req, linkedCustomer)
		if err != nil {
			return err
		}
		if customer != nil {
			order.CustomerID = &customer.ID
		}

		if err := s.orderRepo.Create(ctx, tx, order); err != nil {
			return err
		}
		// Keep the linked customer's denormalized order stats in sync (count on
		// placement; atomic with the order insert). No-op for unlinked orders.
		if s.customerRepo != nil && order.CustomerID != nil {
			if err := s.customerRepo.IncrementOrderStats(ctx, tx, *order.CustomerID, order.TotalAmount); err != nil {
				return err
			}
		}
		// Route the new order through the fulfillment commands (OPE-416): create
		// its fulfillment process + enqueue the orchestration event in the SAME
		// transaction. No-op when the gated service is nil/disabled.
		if s.fulfillment.Enabled() {
			if _, err := s.fulfillment.EnsureProcessForOrder(ctx, tx, tenantID, order.ID); err != nil {
				return err
			}
		}
		if err := s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "order.created",
			EntityType: "order",
			EntityID:   order.ID,
			Changes:    map[string]string{"source": req.Source, "customer_name": req.CustomerName},
			IPAddress:  ip,
		}); err != nil {
			return err
		}
		reservedQtys, err = s.reserveStockInTx(ctx, tx, order.Items)
		return err
	})
	if err != nil {
		return nil, err
	}

	if len(reservedQtys) > 0 {
		s.triggerStockSync(tenantID, reservedQtys, "order_placed")
	}

	DispatchWebhookAsync(s.webhookDispatch, tenantID, "order.created", order)
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

// Duplicate creates a copy of an existing order as a fresh "new" order, with a new ID
// and no external reference. It enforces the monthly order limit inside the same
// transaction (mirroring Create), increments the linked customer's denormalized order
// stats, reserves stock for the copied line items, and writes an "order.duplicated"
// audit entry, then dispatches an order.created webhook after commit. It fires no
// automation event and creates no fulfillment process, and is intentionally
// non-idempotent — each call mints a new order.
func (s *OrderService) Duplicate(ctx context.Context, tenantID, orderID uuid.UUID, maxOrdersMonthly int, actorID uuid.UUID, ip string) (*model.Order, error) {
	var newOrder *model.Order
	var reservedQtys map[uuid.UUID]int
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		if err := EnforceMonthlyOrderLimit(ctx, tx, s.orderRepo, tenantID, maxOrdersMonthly, 0); err != nil {
			return err
		}

		existing, err := s.orderRepo.FindByID(ctx, tx, orderID)
		if err != nil {
			return err
		}
		if existing == nil {
			return ErrOrderNotFound
		}

		// Sanitize user-facing text fields to prevent stored XSS
		customerName := model.StripHTMLTags(existing.CustomerName)
		var notes *string
		if existing.Notes != nil {
			sanitized := model.StripHTMLTags(*existing.Notes)
			notes = &sanitized
		}

		newOrder = &model.Order{
			ID:              uuid.New(),
			TenantID:        existing.TenantID,
			ExternalID:      nil, // Clear ExternalID to avoid duplicate external references
			Source:          existing.Source,
			IntegrationID:   existing.IntegrationID,
			Status:          "new",
			CustomerName:    customerName,
			CustomerEmail:   existing.CustomerEmail,
			CustomerPhone:   existing.CustomerPhone,
			ShippingAddress: existing.ShippingAddress,
			BillingAddress:  existing.BillingAddress,
			Items:           existing.Items,
			TotalAmount:     existing.TotalAmount,
			Currency:        existing.Currency,
			Notes:           notes,
			Metadata:        existing.Metadata,
			Tags:            existing.Tags,
			OrderedAt:       existing.OrderedAt,
			DeliveryMethod:  existing.DeliveryMethod,
			PickupPointID:   existing.PickupPointID,
			PaymentStatus:   existing.PaymentStatus,
			PaymentMethod:   existing.PaymentMethod,
			CustomerID:      existing.CustomerID,
			InternalNotes:   existing.InternalNotes,
			Priority:        existing.Priority,
		}

		if err := s.orderRepo.Create(ctx, tx, newOrder); err != nil {
			return fmt.Errorf("create duplicated order: %w", err)
		}

		// The duplicate persists the source order's customer_id, so the RFM live aggregate
		// (COUNT/SUM over orders by customer_id) counts it. Increment the denormalized
		// customer stats in the same tx — exactly like Create — so the counters stay equal
		// to that aggregate. No-op for unlinked source orders (customer_id NULL).
		if s.customerRepo != nil && newOrder.CustomerID != nil {
			if err := s.customerRepo.IncrementOrderStats(ctx, tx, *newOrder.CustomerID, newOrder.TotalAmount); err != nil {
				return err
			}
		}

		if err := s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "order.duplicated",
			EntityType: "order",
			EntityID:   newOrder.ID,
			Changes:    map[string]string{"source_order_id": orderID.String()},
			IPAddress:  ip,
		}); err != nil {
			return err
		}

		// A duplicate is a real, fulfillable order, so its lines have to hold stock like
		// any other. Without this its quantities stayed invisible to every availability
		// calculation until someone shipped it.
		reservedQtys, err = s.reserveStockInTx(ctx, tx, newOrder.Items)
		return err
	})
	if err != nil {
		return nil, err
	}

	if len(reservedQtys) > 0 {
		s.triggerStockSync(tenantID, reservedQtys, "order_placed")
	}
	// Dispatch webhook for the duplicated order (async, best-effort)
	DispatchWebhookAsync(s.webhookDispatch, tenantID, "order.created", newOrder)

	return newOrder, nil
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
		DispatchWebhookAsync(s.webhookDispatch, tenantID, "order.updated", order)
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
		DispatchWebhookAsync(s.webhookDispatch, tenantID, "order.deleted", map[string]any{"order_id": orderID.String()})
	}
	return err
}

// OrderStatusChange is the record of one applied order status write, handed from the
// transactional writer to the post-commit side effects. It exists so a caller that
// owns its own transaction (ShipmentService, reacting to carrier events) can write the
// status atomically with its own rows and still run the full side-effect set.
type OrderStatusChange struct {
	Order     *model.Order
	OldStatus string
	NewStatus string
	// FirstShip reports that the order had not shipped before this change. shipped_at
	// is stamped once (COALESCE on update, never cleared), so it gates the one-time
	// stock decrement and makes a repeated ship a no-op.
	FirstShip bool
}

// OrderStatusSyncer is the narrow OrderService surface used by services that change an
// order's status as a side effect of their own domain event. Splitting the write from
// the effects is what keeps it safe: the write joins the caller's transaction, and the
// effects run after that transaction commits. Calling OrderService.TransitionStatus
// instead would open a second transaction against a row the caller still has locked.
type OrderStatusSyncer interface {
	WriteOrderStatusInTx(ctx context.Context, tx pgx.Tx, tenantID, orderID uuid.UUID, newStatus string, actorID uuid.UUID, ip string) (*OrderStatusChange, error)
	FireOrderStatusEffects(tenantID uuid.UUID, change *OrderStatusChange)
}

// WriteOrderStatusInTx applies an order status change inside the caller's transaction:
// the status-specific timestamps, the status update, and the order.status_changed audit
// entry. It deliberately does NOT consult the tenant transition graph — validation is
// the caller's policy decision (the API path validates, carrier events do not, because
// a delivered parcel is a fact rather than a request). The returned change must be
// passed to FireOrderStatusEffects once the transaction commits, and only then.
func (s *OrderService) WriteOrderStatusInTx(ctx context.Context, tx pgx.Tx, tenantID, orderID uuid.UUID, newStatus string, actorID uuid.UUID, ip string) (*OrderStatusChange, error) {
	existing, err := s.orderRepo.FindByID(ctx, tx, orderID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, ErrOrderNotFound
	}

	var setShippedAt, setDeliveredAt *time.Time
	if newStatus == model.OrderStatusShipped {
		now := time.Now()
		setShippedAt = &now
	}
	if newStatus == model.OrderStatusDelivered {
		now := time.Now()
		setDeliveredAt = &now
	}

	if err := s.orderRepo.UpdateStatus(ctx, tx, orderID, newStatus, setShippedAt, setDeliveredAt); err != nil {
		return nil, err
	}

	updated, err := s.orderRepo.FindByID(ctx, tx, orderID)
	if err != nil {
		return nil, err
	}

	if err := s.auditRepo.Log(ctx, tx, model.AuditEntry{
		TenantID:   tenantID,
		UserID:     actorID,
		Action:     "order.status_changed",
		EntityType: "order",
		EntityID:   orderID,
		Changes:    map[string]string{"from": existing.Status, "to": newStatus},
		IPAddress:  ip,
	}); err != nil {
		return nil, err
	}

	return &OrderStatusChange{
		Order:     updated,
		OldStatus: existing.Status,
		NewStatus: newStatus,
		FirstShip: existing.ShippedAt == nil,
	}, nil
}

// FireOrderStatusEffects runs every post-commit consequence of an order status change:
// customer email/SMS, the status_changed webhook, invoice auto-create, the automation
// event, Allegro fulfillment sync, the one-time stock decrement on ship, the
// reservation release on cancel, and loyalty accrual on completion. Every branch is
// best-effort and asynchronous, so no side effect can fail the change that caused it.
//
// This is the only place those effects live. Any writer of order status must call it,
// otherwise the status moves while the stock, invoice and notifications do not.
func (s *OrderService) FireOrderStatusEffects(tenantID uuid.UUID, change *OrderStatusChange) {
	if change == nil || change.Order == nil {
		return
	}
	order := change.Order
	oldStatus, newStatus := change.OldStatus, change.NewStatus

	if s.emailService != nil {
		asyncutil.SafeGo(func() {
			s.emailService.SendOrderStatusEmail(context.Background(), tenantID, order, oldStatus, newStatus)
		})
	}
	DispatchWebhookAsync(s.webhookDispatch, tenantID, "order.status_changed", map[string]any{"order_id": order.ID.String(), "from": oldStatus, "to": newStatus})
	if s.invoiceService != nil {
		asyncutil.SafeGo(func() { s.invoiceService.HandleOrderStatusChange(context.Background(), tenantID, order) })
	}
	if s.smsService != nil {
		asyncutil.SafeGo(func() { s.smsService.SendOrderStatusSMS(context.Background(), tenantID, order, oldStatus, newStatus) })
	}
	FireAutomationEvent(s.automationService, tenantID, "order", "order.status_changed", order.ID, map[string]any{
		"status": order.Status, "old_status": oldStatus, "new_status": newStatus,
		"source": order.Source, "customer_name": order.CustomerName,
		"total_amount": order.TotalAmount, "currency": order.Currency,
		"payment_status": order.PaymentStatus,
	})
	// Auto-sync fulfillment status to Allegro (async, best-effort)
	if s.allegroSync != nil && order.Source == "allegro" {
		asyncutil.SafeGo(func() { s.allegroSync.SyncFulfillmentStatus(context.Background(), tenantID, order, newStatus) })
	}
	// Stock sync on status change
	switch newStatus {
	case model.OrderStatusShipped:
		// Decrement component/product stock exactly once per order (FirstShip gate),
		// so a re-ship — or a second parcel of the same order — does not double-decrement.
		if change.FirstShip {
			asyncutil.SafeGo(func() { s.handleStockOnShip(context.Background(), tenantID, order) })
		}
	case model.OrderStatusCancelled:
		// Release reservations once; skip when the order was already terminal
		// (cancelled/refunded) to avoid double-releasing.
		if !isTerminalStockStatus(oldStatus) {
			asyncutil.SafeGo(func() { s.handleStockOnCancel(context.Background(), tenantID, order) })
		}
	case model.OrderStatusCompleted:
		s.awardLoyaltyOnComplete(tenantID, order)
	}
}

// TransitionStatus moves an order to a new status, enforcing allowed transitions.
func (s *OrderService) TransitionStatus(ctx context.Context, tenantID, orderID uuid.UUID, req model.StatusTransitionRequest, actorID uuid.UUID, ip string) (*model.Order, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	var change *OrderStatusChange
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		existing, err := s.orderRepo.FindByID(ctx, tx, orderID)
		if err != nil {
			return err
		}
		if existing == nil {
			return ErrOrderNotFound
		}

		config, err := s.loadStatusConfig(ctx, tx, tenantID)
		if err != nil {
			return fmt.Errorf("load status config: %w", err)
		}

		if !config.IsValidStatus(req.Status) {
			return fmt.Errorf("%w: %q", ErrUnknownStatus, req.Status)
		}

		if !req.Force {
			if !config.IsValidStatus(existing.Status) {
				return fmt.Errorf("%w: current %q", ErrUnknownStatus, existing.Status)
			}
			if !config.CanTransition(existing.Status, req.Status) {
				return fmt.Errorf("%w: %s -> %s", ErrInvalidTransition, existing.Status, req.Status)
			}
		}

		change, err = s.WriteOrderStatusInTx(ctx, tx, tenantID, orderID, req.Status, actorID, ip)
		return err
	})
	if err != nil {
		return nil, err
	}
	s.FireOrderStatusEffects(tenantID, change)
	return change.Order, nil
}

// Compile-time assertion that OrderService is the shared order-status writer that
// ShipmentService depends on.
var _ OrderStatusSyncer = (*OrderService)(nil)

// isTerminalStockStatus reports whether an order status already released/closed out its
// stock reservations, so a transition into cancelled from such a status must not release
// reservations again.
func isTerminalStockStatus(status string) bool {
	return status == model.OrderStatusCancelled || status == model.OrderStatusRefunded
}

// awardLoyaltyOnComplete fires order-completion loyalty accrual asynchronously for the
// order's linked customer. It is a no-op when the loyalty service is unwired or the order
// has no linked customer. Accrual is exactly-once per (order, program) via the ledger
// guard inside AwardPointsForOrder, so re-entry into "completed" (incl. forced
// re-transitions) never double-awards. Runs in a background context like the other
// post-commit side-effects so it cannot fail or block the status transition.
func (s *OrderService) awardLoyaltyOnComplete(tenantID uuid.UUID, order *model.Order) {
	if s.loyaltyService == nil || order.CustomerID == nil {
		return
	}
	customerID := *order.CustomerID
	orderID := order.ID
	amount := order.TotalAmount
	asyncutil.SafeGo(func() {
		s.loyaltyService.AwardPointsForOrder(context.Background(), tenantID, customerID, orderID, amount)
	})
}

// dedupeOrderIDs returns ids with duplicates removed, preserving first-seen order.
func dedupeOrderIDs(ids []uuid.UUID) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{}, len(ids))
	out := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

// BulkTransitionStatus transitions multiple orders to a new status.
func (s *OrderService) BulkTransitionStatus(ctx context.Context, tenantID uuid.UUID, req model.BulkStatusTransitionRequest, actorID uuid.UUID, ip string) (*model.BulkStatusTransitionResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	// Collapse repeated order IDs so each order is processed exactly once. The firstShip
	// gate reads a single ordersMap snapshot that UpdateStatus never refreshes, so a
	// repeated ID would re-pass the gate and fire handleStockOnShip (plus emails, webhooks,
	// audit, and the success counter) once per occurrence — double-decrementing stock for a
	// single ship. req is passed by value, so this mutation stays local to the call.
	req.OrderIDs = dedupeOrderIDs(req.OrderIDs)

	resp := &model.BulkStatusTransitionResponse{
		Results: make([]model.BulkStatusResult, 0, len(req.OrderIDs)),
	}

	// Side effects are collected and replayed after the transaction commits, through
	// the same helper the single-order path uses.
	var pendingChanges []*OrderStatusChange

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

			if !req.Force && !config.CanTransition(existing.Status, req.Status) {
				result.Error = fmt.Sprintf("invalid transition: %s -> %s", existing.Status, req.Status)
				resp.Results = append(resp.Results, result)
				resp.Failed++
				continue
			}

			change, err := s.WriteOrderStatusInTx(ctx, tx, tenantID, orderID, req.Status, actorID, ip)
			if err != nil {
				// A failed audit insert aborts the surrounding transaction, so unlike the
				// pre-shared-writer code this cannot leave a status change unaudited.
				slog.Warn("bulk status transition: status write failed",
					"order_id", orderID, "error", err)
				result.Error = "failed to update status"
				resp.Results = append(resp.Results, result)
				resp.Failed++
				return err
			}

			pendingChanges = append(pendingChanges, change)
			result.Success = true
			resp.Results = append(resp.Results, result)
			resp.Succeeded++
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	for _, change := range pendingChanges {
		s.FireOrderStatusEffects(tenantID, change)
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

// reserveStockInTx reserves warehouse stock for an order's line items inside the
// caller's transaction, so an order and the reservation backing it commit together or
// not at all. It returns the per-product quantities it walked, for the post-commit
// marketplace sync, and nil when there is nothing to reserve.
//
// Reserving less than requested is not an error: a line with no warehouse rows, or
// with fewer on hand than ordered, reserves what exists and lets the order through —
// the pre-existing overselling tolerance. A failure to *apply* a reservation is an
// error and fails the order, which is the point: the previous code reserved in a
// second transaction after the order had committed, so a failure there left a live
// order holding no stock and nothing reported it.
func (s *OrderService) reserveStockInTx(ctx context.Context, tx pgx.Tx, items json.RawMessage) (map[uuid.UUID]int, error) {
	if s.warehouseStockRepo == nil {
		return nil, nil
	}
	productQtys := extractProductQuantities(items)
	if len(productQtys) == 0 {
		return nil, nil
	}
	s.expandBundleQuantities(ctx, tx, productQtys)
	if err := s.adjustStockPerProduct(ctx, tx, productQtys, func(ctx context.Context, tx pgx.Tx, productID uuid.UUID, stock model.WarehouseStock, remaining int) (int, error) {
		available := stock.Quantity - stock.Reserved
		if available <= 0 {
			return 0, nil
		}
		reserveQty := min(remaining, available)
		if _, err := tx.Exec(ctx,
			`UPDATE warehouse_stock SET reserved = reserved + $1, updated_at = NOW()
			 WHERE id = $2`,
			reserveQty, stock.ID); err != nil {
			return 0, fmt.Errorf("reserve stock for product %s in warehouse %s: %w", productID, stock.WarehouseID, err)
		}
		return reserveQty, nil
	}); err != nil {
		return nil, err
	}
	return productQtys, nil
}

// adjustStockPerProduct walks each product in productQtys and, for products that
// have stock rows, applies perStock to successive rows until the requested
// quantity is consumed. perStock performs the row-level UPDATE and returns how
// much of remaining it consumed (0 when it skips the row). It factors out the
// reserve/ship/cancel iteration skeleton; callers must already run inside a
// tenant-scoped transaction.
//
// A perStock error aborts the whole walk: a failed statement has already poisoned the
// transaction, so continuing would only pile up "current transaction is aborted"
// failures on top of the real cause.
func (s *OrderService) adjustStockPerProduct(
	ctx context.Context,
	tx pgx.Tx,
	productQtys map[uuid.UUID]int,
	perStock func(ctx context.Context, tx pgx.Tx, productID uuid.UUID, stock model.WarehouseStock, remaining int) (int, error),
) error {
	for productID, qty := range productQtys {
		stocks, err := s.warehouseStockRepo.ListByProduct(ctx, tx, productID)
		if err != nil {
			return fmt.Errorf("list warehouse stock for product %s: %w", productID, err)
		}
		remaining := qty
		for _, stock := range stocks {
			if remaining <= 0 {
				break
			}
			consumed, err := perStock(ctx, tx, productID, stock, remaining)
			if err != nil {
				return err
			}
			remaining -= consumed
		}
	}
	return nil
}

// handleStockOnShip decrements quantity and reserved in warehouse_stock for shipped orders.
func (s *OrderService) handleStockOnShip(ctx context.Context, tenantID uuid.UUID, order *model.Order) {
	if s.warehouseStockRepo == nil {
		return
	}
	productQtys := extractProductQuantities(order.Items)
	if err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		s.expandBundleQuantities(ctx, tx, productQtys)
		return s.adjustStockPerProduct(ctx, tx, productQtys, func(ctx context.Context, tx pgx.Tx, productID uuid.UUID, stock model.WarehouseStock, remaining int) (int, error) {
			deduct := min(remaining, stock.Quantity)
			// Decrement both quantity and reserved
			if _, execErr := tx.Exec(ctx,
				`UPDATE warehouse_stock
				 SET quantity = GREATEST(quantity - $1, 0),
				     reserved = GREATEST(reserved - $1, 0),
				     updated_at = NOW()
				 WHERE id = $2`,
				deduct, stock.ID); execErr != nil {
				return 0, fmt.Errorf("adjust warehouse stock for product %s in warehouse %s: %w", productID, stock.WarehouseID, execErr)
			}
			return deduct, nil
		})
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
		s.expandBundleQuantities(ctx, tx, productQtys)
		return s.adjustStockPerProduct(ctx, tx, productQtys, func(ctx context.Context, tx pgx.Tx, productID uuid.UUID, stock model.WarehouseStock, remaining int) (int, error) {
			release := min(remaining, stock.Reserved)
			if _, execErr := tx.Exec(ctx,
				`UPDATE warehouse_stock SET reserved = GREATEST(reserved - $1, 0), updated_at = NOW()
				 WHERE id = $2`,
				release, stock.ID); execErr != nil {
				return 0, fmt.Errorf("release stock reservation for product %s in warehouse %s: %w", productID, stock.WarehouseID, execErr)
			}
			return release, nil
		})
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
