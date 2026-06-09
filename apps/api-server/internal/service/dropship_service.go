package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/asyncutil"
	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

var (
	// ErrDropshipOrderNotFound is returned when a dropship order does not exist.
	ErrDropshipOrderNotFound = errors.New("dropship order not found")
	// ErrDropshipAlreadyCancelled is returned when a dropship order has already been cancelled.
	ErrDropshipAlreadyCancelled = errors.New("dropship order is already cancelled")
	// ErrDropshipInvalidTransition is returned for an invalid dropship order status transition.
	ErrDropshipInvalidTransition = errors.New("invalid dropship status transition")
	// ErrNoDropshipItems is returned when no dropship items exist for an order.
	ErrNoDropshipItems = errors.New("no dropship items found for this order")
)

// DropshipService handles business logic for dropship orders.
type DropshipService struct {
	dropshipRepo     repository.DropshipOrderRepo
	dropshipItemRepo repository.DropshipOrderItemRepo
	orderRepo        repository.OrderRepo
	productRepo      repository.ProductRepo
	supplierRepo     repository.SupplierRepo
	auditRepo        repository.AuditRepo
	integrationSvc   *IntegrationService
	pool             *pgxpool.Pool
	webhookDispatch  *WebhookDispatchService
	logger           *slog.Logger
	fulfillment      *FulfillmentService
	availability     *SupplierAvailabilityService
	supplierOrder    *SupplierOrderService
}

// NewDropshipService creates a new DropshipService.
func NewDropshipService(
	dropshipRepo repository.DropshipOrderRepo,
	dropshipItemRepo repository.DropshipOrderItemRepo,
	orderRepo repository.OrderRepo,
	productRepo repository.ProductRepo,
	supplierRepo repository.SupplierRepo,
	auditRepo repository.AuditRepo,
	integrationSvc *IntegrationService,
	pool *pgxpool.Pool,
	webhookDispatch *WebhookDispatchService,
	logger *slog.Logger,
) *DropshipService {
	return &DropshipService{
		dropshipRepo:     dropshipRepo,
		dropshipItemRepo: dropshipItemRepo,
		orderRepo:        orderRepo,
		productRepo:      productRepo,
		supplierRepo:     supplierRepo,
		auditRepo:        auditRepo,
		integrationSvc:   integrationSvc,
		pool:             pool,
		webhookDispatch:  webhookDispatch,
		logger:           logger,
	}
}

// SetFulfillmentService wires the gated fulfillment service used for best-effort
// dropship-unit + supplier-capability recording (OPE-418). Nil-safe and a complete
// no-op until FULFILLMENT_PROCESS_ENABLED is set.
func (s *DropshipService) SetFulfillmentService(f *FulfillmentService) {
	s.fulfillment = f
}

// SetAvailabilityService wires the gated supplier-availability resolver (OPE-418) used to
// gate auto-routing of dropship units. Nil-safe and a complete no-op until
// SUPPLIER_AVAILABILITY_ENABLED is set.
func (s *DropshipService) SetAvailabilityService(svc *SupplierAvailabilityService) {
	s.availability = svc
}

// SetSupplierOrderService wires the gated supplier-order engine (OPE-418/Phase-7). Nil-safe
// and a complete no-op until SUPPLIER_ORDER_ENABLED is set: the gate's API branch keeps its
// current behavior (mark the create_dropship_order step ready, no auto-submit).
func (s *DropshipService) SetSupplierOrderService(svc *SupplierOrderService) {
	s.supplierOrder = svc
}

// recordDropshipUnit ensures a dropship fulfillment unit for the order/supplier and
// records the create_dropship_order step + supplier capability (OPE-418, gated
// best-effort). When the supplier cannot accept API orders it records a manual or
// portal preflight step; when the supplier is configured with an integration whose
// format has no provider it raises a typed capability-missing blocker. It never
// affects the dropship operation: all FulfillmentService calls are best-effort.
func (s *DropshipService) recordDropshipUnit(ctx context.Context, tenantID, orderID, supplierID uuid.UUID, supplierName string, items []model.DropshipOrderItem) {
	if !s.fulfillment.Enabled() {
		return
	}
	capability := s.supplierCapability(ctx, tenantID, supplierID)

	unit := s.fulfillment.EnsureUnit(ctx, tenantID, orderID, model.UnitTypeDropship, supplierID.String(),
		map[string]any{"supplier_id": supplierID.String(), "supplier_name": supplierName, "capability": string(capability)})
	if unit == nil {
		return
	}

	switch capability {
	case SupplierCapAPI:
		// OPE-418 (gated): before treating the auto-submit path as ready, consult the
		// supplier-availability resolver. When availability is stale/unknown/insufficient
		// (or a backorder ETA applies) the unit is held with a typed blocker / backorder
		// instead of being marked ready for auto-submit. Off by default -> the legacy
		// always-ready behaviour is unchanged.
		if s.availability.Enabled() {
			if s.gateDropshipAvailability(ctx, tenantID, orderID, supplierID, unit.ID, items) {
				s.fulfillment.AggregateMixedOrder(ctx, tenantID, orderID)
				return
			}
		}
		// API-capable: the create_dropship_order step is ready to run programmatically.
		s.fulfillment.RecordStep(ctx, tenantID, unit.ID, model.StepCreateDropshipOrder, model.FulfillmentStatusReady,
			map[string]any{"capability": string(capability), "supplier_id": supplierID.String()})
		// OPE-418/Phase-7: when the supplier-order engine is enabled, enqueue the submit
		// event so the SupplierOrderHandler runs prepare->preflight->submit. Off by default
		// -> the step stays "ready" with no auto-submit (current behavior).
		if s.supplierOrder.Enabled() {
			s.supplierOrder.EnqueueSubmit(ctx, tenantID, orderID, supplierID, unit.ID, unit.ProcessID)
		}
	case SupplierCapPortal, SupplierCapManual:
		// No API: a human must place the order via portal/out-of-band — surface a
		// preflight step waiting on the operator.
		s.fulfillment.RecordStep(ctx, tenantID, unit.ID, model.StepPreflightSupplierOrder, model.FulfillmentStatusWaitingExternal,
			map[string]any{"capability": string(capability), "supplier_id": supplierID.String()})
		s.fulfillment.RecordUnitTransition(ctx, tenantID, unit.ID, model.FulfillmentStatusWaitingExternal)
		// OPE-418/Phase-7: when the engine is enabled, surface an actionable operator
		// blocker so portal/manual suppliers are an explicit step, not a silent waiting unit.
		if s.supplierOrder.Enabled() {
			s.fulfillment.CreateSupplierBlocker(ctx, tenantID, orderID, &unit.ID,
				model.BlockerSupplierManualSubmissionRequired,
				fmt.Sprintf("supplier %q requires manual order submission (capability %s)", supplierName, capability))
		}
	case SupplierCapUnsupported:
		// Integration linked but no provider for its format — capability missing.
		s.fulfillment.RecordStep(ctx, tenantID, unit.ID, model.StepPreflightSupplierOrder, model.FulfillmentStatusBlocked,
			map[string]any{"capability": string(capability), "supplier_id": supplierID.String()})
		s.fulfillment.RecordUnitTransition(ctx, tenantID, unit.ID, model.FulfillmentStatusBlocked)
		s.fulfillment.CreateSupplierBlocker(ctx, tenantID, orderID, &unit.ID,
			model.BlockerIntegrationCapabilityMissing,
			fmt.Sprintf("supplier %q has an integration with no registered order provider", supplierName))
	}
	s.fulfillment.AggregateMixedOrder(ctx, tenantID, orderID)
}

// gateDropshipAvailability consults the supplier-availability resolver for each dropship
// line (OPE-418, gated best-effort). It returns true when the unit was HELD — either a
// typed blocker was opened (stale / unknown / insufficient availability) or a backorder
// unit was surfaced (insufficient now but an ETA is known) — so the caller does NOT mark
// the create_dropship_order step ready for auto-submit. It returns false when every line
// is trusted + sufficient (auto_routable), so the caller proceeds as before. Any resolver
// error or missing product link is treated as "do not gate" so the legacy path stands.
func (s *DropshipService) gateDropshipAvailability(ctx context.Context, tenantID, orderID, supplierID, unitID uuid.UUID, items []model.DropshipOrderItem) bool {
	now := time.Now()
	for _, item := range items {
		if item.ProductID == nil {
			continue // unmapped line — nothing to resolve against
		}
		decision, ok, err := s.availability.ResolveForProduct(ctx, tenantID, supplierID, *item.ProductID, nil, nil, item.Quantity, false, now)
		if err != nil || !ok {
			continue // best-effort: a resolver failure never blocks routing
		}
		if decision.BlockerCode != nil {
			// stale / unknown / insufficient -> typed blocker on the unit, do NOT auto-submit.
			s.fulfillment.RecordStep(ctx, tenantID, unitID, model.StepCreateDropshipOrder, model.FulfillmentStatusBlocked,
				map[string]any{"supplier_id": supplierID.String(), "reason": *decision.BlockerCode})
			s.fulfillment.RecordUnitTransition(ctx, tenantID, unitID, model.FulfillmentStatusBlocked)
			s.fulfillment.CreateSupplierBlocker(ctx, tenantID, orderID, &unitID, *decision.BlockerCode,
				"supplier availability not safe for auto-routing")
			return true
		}
		if decision.Backorder {
			// Insufficient now but an ETA is known -> backorder unit carrying the ETA.
			s.fulfillment.EnsureUnit(ctx, tenantID, orderID, model.UnitTypeBackorder, supplierID.String(),
				map[string]any{"supplier_id": supplierID.String(), "reason": "supplier_backorder"})
			s.fulfillment.RecordStep(ctx, tenantID, unitID, model.StepCreateDropshipOrder, model.FulfillmentStatusWaitingExternal,
				map[string]any{"supplier_id": supplierID.String(), "reason": "supplier_backorder"})
			s.fulfillment.RecordUnitTransition(ctx, tenantID, unitID, model.FulfillmentStatusWaitingExternal)
			return true
		}
	}
	return false
}

// supplierCapability loads a supplier and classifies its fulfillment capability
// from its existing config (feed format + linked integration). Best-effort: on any
// lookup error it falls back to manual so a unit is still recorded.
func (s *DropshipService) supplierCapability(ctx context.Context, tenantID, supplierID uuid.UUID) SupplierFulfillmentCapability {
	capability := SupplierCapManual // best-effort fallback if the lookup yields nothing
	_ = database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		supplier, err := s.supplierRepo.FindByID(ctx, tx, supplierID)
		if err != nil {
			return err // logged-and-ignored by the caller; fallback stays manual
		}
		if supplier == nil {
			return nil // keep manual fallback
		}
		providerName := supplier.FeedFormat
		if providerName == "xml" {
			providerName = "btp"
		}
		capability = ClassifySupplierCapability(supplier, supplier.IntegrationID != nil && integration.HasSupplierProvider(providerName))
		return nil
	})
	return capability
}

// recordDropshipTransition maps a dropship status change onto the order's dropship
// unit + supplier steps (OPE-418, gated best-effort). It looks up the dropship
// order to resolve the customer order + supplier, then records the appropriate
// step/unit transition and re-aggregates the order's process.
func (s *DropshipService) recordDropshipTransition(ctx context.Context, tenantID uuid.UUID, d *model.DropshipOrder) {
	if !s.fulfillment.Enabled() || d == nil {
		return
	}
	unit := s.fulfillment.EnsureUnit(ctx, tenantID, d.OrderID, model.UnitTypeDropship, d.SupplierID.String(),
		map[string]any{"supplier_id": d.SupplierID.String(), "supplier_name": d.SupplierName})
	if unit == nil {
		return
	}
	switch d.Status {
	case "sent":
		s.fulfillment.RecordStep(ctx, tenantID, unit.ID, model.StepCreateDropshipOrder, model.FulfillmentStatusSucceeded,
			map[string]any{"supplier_id": d.SupplierID.String()})
		s.fulfillment.RecordUnitTransition(ctx, tenantID, unit.ID, model.FulfillmentStatusWaitingExternal)
	case "confirmed":
		s.fulfillment.RecordStep(ctx, tenantID, unit.ID, model.StepConfirmSupplierOrder, model.FulfillmentStatusSucceeded,
			map[string]any{"supplier_id": d.SupplierID.String()})
		s.fulfillment.RecordUnitTransition(ctx, tenantID, unit.ID, model.FulfillmentStatusRunning)
	case "shipped", "delivered":
		s.fulfillment.RecordStep(ctx, tenantID, unit.ID, model.StepAwaitTracking, model.FulfillmentStatusSucceeded,
			map[string]any{"supplier_id": d.SupplierID.String(), "dropship_status": d.Status})
		s.fulfillment.RecordUnitTransition(ctx, tenantID, unit.ID, model.FulfillmentStatusSucceeded)
	case "cancelled":
		s.fulfillment.RecordUnitTransition(ctx, tenantID, unit.ID, model.FulfillmentStatusCancelled)
	}
	s.fulfillment.AggregateMixedOrder(ctx, tenantID, d.OrderID)
}

// List returns a paginated list of dropship orders.
func (s *DropshipService) List(ctx context.Context, tenantID uuid.UUID, filter model.DropshipOrderListFilter) (model.ListResponse[model.DropshipOrder], error) {
	var resp model.ListResponse[model.DropshipOrder]
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		orders, total, err := s.dropshipRepo.List(ctx, tx, filter)
		if err != nil {
			return err
		}
		if orders == nil {
			orders = []model.DropshipOrder{}
		}
		resp = model.ListResponse[model.DropshipOrder]{
			Items:  orders,
			Total:  total,
			Limit:  filter.Limit,
			Offset: filter.Offset,
		}
		return nil
	})
	return resp, err
}

// Get returns a single dropship order with its items.
func (s *DropshipService) Get(ctx context.Context, tenantID, id uuid.UUID) (*model.DropshipOrder, error) {
	var d *model.DropshipOrder
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var err error
		d, err = s.dropshipRepo.FindByID(ctx, tx, id)
		if err != nil {
			return err
		}
		if d == nil {
			return ErrDropshipOrderNotFound
		}
		items, err := s.dropshipItemRepo.ListByDropshipOrderID(ctx, tx, id)
		if err != nil {
			return err
		}
		if items == nil {
			items = []model.DropshipOrderItem{}
		}
		d.Items = items
		return nil
	})
	if err != nil {
		return nil, err
	}
	return d, nil
}

// GetByOrderID returns all dropship orders for a customer order.
func (s *DropshipService) GetByOrderID(ctx context.Context, tenantID, orderID uuid.UUID) ([]model.DropshipOrder, error) {
	var orders []model.DropshipOrder
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var err error
		orders, err = s.dropshipRepo.FindByOrderID(ctx, tx, orderID)
		if err != nil {
			return err
		}
		if orders == nil {
			orders = []model.DropshipOrder{}
		}
		// Load items for each dropship order
		for i := range orders {
			items, err := s.dropshipItemRepo.ListByDropshipOrderID(ctx, tx, orders[i].ID)
			if err != nil {
				return err
			}
			if items == nil {
				items = []model.DropshipOrderItem{}
			}
			orders[i].Items = items
		}
		return nil
	})
	return orders, err
}

// dropshipOrderItemJSON is used internally for parsing order items from JSONB.
type dropshipOrderItemJSON struct {
	ProductID *string `json:"product_id"`
	SKU       string  `json:"sku"`
	Name      string  `json:"name"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"`
}

// AutoRouteOrder parses order items, checks which are dropship products,
// groups them by supplier, and creates one dropship order per supplier.
func (s *DropshipService) AutoRouteOrder(ctx context.Context, tenantID, orderID, actorID uuid.UUID, ip string) ([]model.DropshipOrder, error) {
	var result []model.DropshipOrder

	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		// 1. Get the customer order
		order, err := s.orderRepo.FindByID(ctx, tx, orderID)
		if err != nil {
			return err
		}
		if order == nil {
			return NewValidationError(fmt.Errorf("order not found"))
		}

		// 2. Parse items from order JSONB
		var items []dropshipOrderItemJSON
		if order.Items != nil {
			if err := json.Unmarshal(order.Items, &items); err != nil {
				return NewValidationError(fmt.Errorf("failed to parse order items: %w", err))
			}
		}
		if len(items) == 0 {
			return NewValidationError(fmt.Errorf("order has no items"))
		}

		// 3. For each item, check if the product is a dropship product
		type supplierGroup struct {
			supplierID   uuid.UUID
			supplierName string
			items        []model.DropshipOrderItem
			totalCost    float64
		}
		groups := make(map[uuid.UUID]*supplierGroup)

		for _, item := range items {
			if item.ProductID == nil || *item.ProductID == "" {
				continue
			}
			productID, err := uuid.Parse(*item.ProductID)
			if err != nil {
				continue
			}

			product, err := s.productRepo.FindByID(ctx, tx, productID)
			if err != nil || product == nil {
				continue
			}

			if !product.IsDropship || product.DropshipSupplierID == nil {
				continue
			}

			supplierID := *product.DropshipSupplierID

			// Lookup supplier name if not already in group
			if _, ok := groups[supplierID]; !ok {
				supplier, err := s.supplierRepo.FindByID(ctx, tx, supplierID)
				if err != nil || supplier == nil {
					s.logger.Warn("dropship supplier not found", "supplier_id", supplierID)
					continue
				}
				groups[supplierID] = &supplierGroup{
					supplierID:   supplierID,
					supplierName: supplier.Name,
				}
			}

			// Use supplier_products cost price if available, otherwise use order item price
			unitCost := item.Price
			sku := item.SKU
			if product.SKU != nil && *product.SKU != "" {
				sku = *product.SKU
			}

			dsItem := model.DropshipOrderItem{
				ID:          uuid.New(),
				TenantID:    tenantID,
				ProductID:   &productID,
				SKU:         sku,
				ProductName: item.Name,
				Quantity:    item.Quantity,
				UnitCost:    unitCost,
			}

			groups[supplierID].items = append(groups[supplierID].items, dsItem)
			groups[supplierID].totalCost += float64(item.Quantity) * unitCost
		}

		if len(groups) == 0 {
			return ErrNoDropshipItems
		}

		// 4. Create one dropship order per supplier
		for _, group := range groups {
			dsOrder := &model.DropshipOrder{
				ID:           uuid.New(),
				TenantID:     tenantID,
				OrderID:      orderID,
				SupplierID:   group.supplierID,
				SupplierName: group.supplierName,
				Status:       "pending",
				TotalCost:    group.totalCost,
				Currency:     order.Currency,
			}

			if err := s.dropshipRepo.Create(ctx, tx, dsOrder); err != nil {
				return fmt.Errorf("create dropship order: %w", err)
			}

			var createdItems []model.DropshipOrderItem
			for _, item := range group.items {
				item.DropshipOrderID = dsOrder.ID
				if err := s.dropshipItemRepo.CreateItem(ctx, tx, &item); err != nil {
					return fmt.Errorf("create dropship item: %w", err)
				}
				createdItems = append(createdItems, item)
			}
			dsOrder.Items = createdItems

			if err := s.auditRepo.Log(ctx, tx, model.AuditEntry{
				TenantID:   tenantID,
				UserID:     actorID,
				Action:     "dropship_order.created",
				EntityType: "dropship_order",
				EntityID:   dsOrder.ID,
				Changes:    map[string]string{"supplier": group.supplierName, "order_id": orderID.String()},
				IPAddress:  ip,
			}); err != nil {
				return err
			}

			result = append(result, *dsOrder)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	if s.webhookDispatch != nil {
		for _, ds := range result {
			asyncutil.SafeGo(func() { s.webhookDispatch.Dispatch(context.Background(), tenantID, "dropship_order.created", ds) })
		}
	}
	// OPE-418: record a dropship unit per supplier + supplier capability (gated,
	// best-effort — never affects the routing result above).
	for _, ds := range result {
		s.recordDropshipUnit(ctx, tenantID, ds.OrderID, ds.SupplierID, ds.SupplierName, ds.Items)
	}
	return result, nil
}

// Create creates a new dropship order manually.
func (s *DropshipService) Create(ctx context.Context, tenantID uuid.UUID, req model.CreateDropshipOrderRequest, actorID uuid.UUID, ip string) (*model.DropshipOrder, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	var d *model.DropshipOrder
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		// Verify order exists
		order, err := s.orderRepo.FindByID(ctx, tx, req.OrderID)
		if err != nil {
			return err
		}
		if order == nil {
			return NewValidationError(fmt.Errorf("order not found"))
		}

		// Verify supplier exists
		supplier, err := s.supplierRepo.FindByID(ctx, tx, req.SupplierID)
		if err != nil {
			return err
		}
		if supplier == nil {
			return NewValidationError(fmt.Errorf("supplier not found"))
		}

		// Calculate total
		var totalCost float64
		for _, item := range req.Items {
			totalCost += float64(item.Quantity) * item.UnitCost
		}

		d = &model.DropshipOrder{
			ID:           uuid.New(),
			TenantID:     tenantID,
			OrderID:      req.OrderID,
			SupplierID:   req.SupplierID,
			SupplierName: supplier.Name,
			Status:       "pending",
			TotalCost:    totalCost,
			Currency:     req.Currency,
			Notes:        req.Notes,
		}

		if err := s.dropshipRepo.Create(ctx, tx, d); err != nil {
			return fmt.Errorf("create dropship order: %w", err)
		}

		var items []model.DropshipOrderItem
		for _, itemReq := range req.Items {
			item := &model.DropshipOrderItem{
				ID:              uuid.New(),
				TenantID:        tenantID,
				DropshipOrderID: d.ID,
				ProductID:       itemReq.ProductID,
				SKU:             itemReq.SKU,
				ProductName:     model.StripHTMLTags(itemReq.ProductName),
				Quantity:        itemReq.Quantity,
				UnitCost:        itemReq.UnitCost,
			}
			if err := s.dropshipItemRepo.CreateItem(ctx, tx, item); err != nil {
				return fmt.Errorf("create dropship item: %w", err)
			}
			items = append(items, *item)
		}
		d.Items = items

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "dropship_order.created",
			EntityType: "dropship_order",
			EntityID:   d.ID,
			Changes:    map[string]string{"supplier": supplier.Name, "order_id": req.OrderID.String()},
			IPAddress:  ip,
		})
	})
	if err != nil {
		return nil, err
	}

	if s.webhookDispatch != nil {
		asyncutil.SafeGo(func() { s.webhookDispatch.Dispatch(context.Background(), tenantID, "dropship_order.created", d) })
	}
	// OPE-418: record the dropship unit + supplier capability (gated, best-effort).
	s.recordDropshipUnit(ctx, tenantID, d.OrderID, d.SupplierID, d.SupplierName, d.Items)
	return d, nil
}

// UpdateStatus updates the status of a dropship order with optional tracking info.
// When transitioning to "sent" and the supplier has a linked integration,
// the order is automatically submitted to the supplier's API.
func (s *DropshipService) UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, req model.UpdateDropshipStatusRequest, actorID uuid.UUID, ip string) (*model.DropshipOrder, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	var d *model.DropshipOrder
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		existing, err := s.dropshipRepo.FindByID(ctx, tx, id)
		if err != nil {
			return err
		}
		if existing == nil {
			return ErrDropshipOrderNotFound
		}

		// Validate status transition
		if !isValidDropshipTransition(existing.Status, req.Status) {
			return NewValidationError(fmt.Errorf("cannot transition from %s to %s", existing.Status, req.Status))
		}

		// Auto-submit to supplier API when transitioning to "sent" — but never re-submit an
		// order that already carries a supplier reference. This idempotency guard prevents a
		// double-submit when the OPE-418/Phase-7 supplier-order engine (supplier.order.submit
		// handler) has already placed the order and set the reference on this row.
		alreadySubmitted := existing.SupplierReference != nil && strings.TrimSpace(*existing.SupplierReference) != ""
		if req.Status == "sent" && existing.Status == "pending" && !alreadySubmitted {
			if result, err := s.submitToSupplierAPI(ctx, tx, tenantID, existing); err != nil {
				s.logger.Error("failed to submit dropship order to supplier API",
					"dropship_order_id", id, "supplier_id", existing.SupplierID, "error", err)
				return fmt.Errorf("submit to supplier: %w", err)
			} else if result != nil {
				ref := result.ExternalOrderID
				if result.OrderNumber != "" {
					ref = result.OrderNumber
				}
				req.SupplierReference = &ref
			}
		}

		if err := s.dropshipRepo.UpdateFields(ctx, tx, id, req); err != nil {
			return err
		}

		d, err = s.dropshipRepo.FindByID(ctx, tx, id)
		if err != nil {
			return err
		}
		items, err := s.dropshipItemRepo.ListByDropshipOrderID(ctx, tx, id)
		if err != nil {
			return err
		}
		if items == nil {
			items = []model.DropshipOrderItem{}
		}
		d.Items = items

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "dropship_order.status_updated",
			EntityType: "dropship_order",
			EntityID:   id,
			Changes:    map[string]string{"from": existing.Status, "to": req.Status},
			IPAddress:  ip,
		})
	})
	if err != nil {
		return nil, err
	}

	if s.webhookDispatch != nil {
		asyncutil.SafeGo(func() { s.webhookDispatch.Dispatch(context.Background(), tenantID, "dropship_order.status_updated", d) })
	}
	// OPE-418: map the dropship status change onto the dropship unit + supplier
	// steps (gated, best-effort — never affects the result above).
	s.recordDropshipTransition(ctx, tenantID, d)
	return d, nil
}

// Cancel cancels a dropship order.
func (s *DropshipService) Cancel(ctx context.Context, tenantID, id uuid.UUID, actorID uuid.UUID, ip string) (*model.DropshipOrder, error) {
	var d *model.DropshipOrder
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		existing, err := s.dropshipRepo.FindByID(ctx, tx, id)
		if err != nil {
			return err
		}
		if existing == nil {
			return ErrDropshipOrderNotFound
		}
		if existing.Status == "cancelled" {
			return ErrDropshipAlreadyCancelled
		}
		if existing.Status == "delivered" {
			return NewValidationError(fmt.Errorf("cannot cancel a delivered dropship order"))
		}

		if err := s.dropshipRepo.UpdateStatus(ctx, tx, id, "cancelled"); err != nil {
			return err
		}

		d, err = s.dropshipRepo.FindByID(ctx, tx, id)
		if err != nil {
			return err
		}
		items, err := s.dropshipItemRepo.ListByDropshipOrderID(ctx, tx, id)
		if err != nil {
			return err
		}
		if items == nil {
			items = []model.DropshipOrderItem{}
		}
		d.Items = items

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "dropship_order.cancelled",
			EntityType: "dropship_order",
			EntityID:   id,
			Changes:    map[string]string{"status": "cancelled"},
			IPAddress:  ip,
		})
	})
	if err != nil {
		return nil, err
	}

	if s.webhookDispatch != nil {
		asyncutil.SafeGo(func() { s.webhookDispatch.Dispatch(context.Background(), tenantID, "dropship_order.cancelled", d) })
	}
	// OPE-418: cancel the dropship unit (gated, best-effort).
	s.recordDropshipTransition(ctx, tenantID, d)
	return d, nil
}

// submitToSupplierAPI submits a dropship order to the supplier's API (e.g. BTP).
// Returns nil result if the supplier has no linked integration (manual process).
func (s *DropshipService) submitToSupplierAPI(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID, order *model.DropshipOrder) (*integration.SupplierOrderResult, error) {
	// Load supplier to check for integration
	supplier, err := s.supplierRepo.FindByID(ctx, tx, order.SupplierID)
	if err != nil {
		return nil, fmt.Errorf("load supplier: %w", err)
	}
	if supplier == nil || supplier.IntegrationID == nil {
		return nil, nil // No integration — manual process
	}

	// Determine provider name: xml-format suppliers use "btp" provider
	providerName := supplier.FeedFormat
	if providerName == "xml" {
		providerName = "btp"
	}
	if !integration.HasSupplierProvider(providerName) {
		return nil, nil // No provider registered for this format
	}

	// Decrypt credentials
	credJSON, err := s.integrationSvc.GetDecryptedCredentialsByID(ctx, tenantID, *supplier.IntegrationID)
	if err != nil {
		return nil, fmt.Errorf("decrypt credentials: %w", err)
	}

	provider, err := integration.NewSupplierProvider(providerName, credJSON, supplier.Settings)
	if err != nil {
		return nil, fmt.Errorf("create provider: %w", err)
	}

	// Load items
	items, err := s.dropshipItemRepo.ListByDropshipOrderID(ctx, tx, order.ID)
	if err != nil {
		return nil, fmt.Errorf("load items: %w", err)
	}

	// Build order request
	req := integration.SupplierOrderRequest{
		ClientOrderNumber: order.ID.String(),
	}
	if order.Notes != nil {
		req.Note = *order.Notes
	}

	for _, item := range items {
		line := integration.SupplierOrderLine{
			Quantity: float64(item.Quantity),
		}

		// Look up product EAN for BarCode identification
		if item.ProductID != nil {
			product, err := s.productRepo.FindByID(ctx, tx, *item.ProductID)
			if err == nil && product != nil {
				if product.EAN != nil && *product.EAN != "" {
					line.EAN = *product.EAN
				}
				if product.SKU != nil && *product.SKU != "" {
					line.ItemID = *product.SKU
				}
			}
		}

		// Fallback to dropship item SKU
		if line.EAN == "" && line.ItemID == "" && item.SKU != "" {
			line.ItemID = item.SKU
		}

		req.Lines = append(req.Lines, line)
	}

	if len(req.Lines) == 0 {
		return nil, fmt.Errorf("no order lines could be built")
	}

	result, err := provider.CreateOrder(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("API call: %w", err)
	}

	s.logger.Info("dropship order submitted to supplier API",
		"dropship_order_id", order.ID,
		"supplier", supplier.Name,
		"external_order_id", result.ExternalOrderID,
		"order_number", result.OrderNumber)

	return result, nil
}

// isValidDropshipTransition checks if a status transition is valid.
func isValidDropshipTransition(from, to string) bool {
	transitions := map[string][]string{
		"pending":   {"sent", "cancelled"},
		"sent":      {"confirmed", "shipped", "cancelled"},
		"confirmed": {"shipped", "cancelled"},
		"shipped":   {"delivered", "cancelled"},
		"delivered": {},
		"cancelled": {},
	}

	allowed, ok := transitions[from]
	if !ok {
		return false
	}
	return slices.Contains(allowed, to)
}
