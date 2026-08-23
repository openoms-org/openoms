package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	engine "github.com/openoms-org/openoms/packages/order-engine"

	"github.com/openoms-org/openoms/apps/api-server/internal/asyncutil"
	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
	"github.com/openoms-org/openoms/apps/api-server/internal/storage"
)

var (
	// ErrShipmentNotFound is returned when a shipment does not exist.
	ErrShipmentNotFound = errors.New("shipment not found")
	// ErrOrderNotFoundForShipment is returned when a shipment's associated order cannot be found.
	ErrOrderNotFoundForShipment = errors.New("order not found for shipment")
	// ErrLabelNotAvailable is returned when a shipment has no stored label file.
	ErrLabelNotAvailable = errors.New("label not available")
)

type shipmentLookupQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// orderStatusSideEffectApplier is the OrderService surface ShipmentService needs to fan
// out the post-commit side effects of an order status change that ShipmentService wrote
// itself (carrier reported the package picked up, in transit, or delivered). Implemented
// by *OrderService.
type orderStatusSideEffectApplier interface {
	applyStatusChangeSideEffects(tenantID uuid.UUID, c orderStatusChange)
}

// ShipmentService handles business logic for shipment management.
type ShipmentService struct {
	shipmentRepo      repository.ShipmentRepo
	orderRepo         repository.OrderRepo
	productRepo       repository.ProductRepo
	auditRepo         repository.AuditRepo
	tenantRepo        repository.TenantRepo
	pool              *pgxpool.Pool
	workerQuerier     shipmentLookupQuerier
	webhookDispatch   *WebhookDispatchService
	smsService        *SMSService
	automationService *AutomationService
	allegroSync       *AllegroSyncService
	fulfillment       *FulfillmentService
	objectStorage     storage.ObjectStorage
	orderStatusFanout orderStatusSideEffectApplier
}

// SetOrderStatusSideEffects wires the OrderService whose status-change fan-out applies
// to orders this service moves on a carrier's behalf. Called after construction to
// avoid a circular dependency. When left unwired the order status is still synced, but
// its side effects (email, webhook, invoice, SMS, automation, stock on first ship) do
// not fire — which is exactly the gap this wiring closes, so production must set it.
func (s *ShipmentService) SetOrderStatusSideEffects(applier orderStatusSideEffectApplier) {
	s.orderStatusFanout = applier
}

// SetFulfillmentService wires the gated fulfillment service used for best-effort
// provider-attempt recording + fulfillment-step emission (OPE-417). Nil-safe and a
// complete no-op until FULFILLMENT_PROCESS_ENABLED is set.
func (s *ShipmentService) SetFulfillmentService(f *FulfillmentService) {
	s.fulfillment = f
}

// SetWorkerPool sets the privileged database pool used for cross-tenant webhook lookups.
func (s *ShipmentService) SetWorkerPool(workerPool *pgxpool.Pool) {
	s.workerQuerier = workerPool
}

// SetSMSService sets the SMS service for sending SMS notifications on shipment status change.
func (s *ShipmentService) SetSMSService(smsSvc *SMSService) {
	s.smsService = smsSvc
}

// SetAutomationService sets the automation service for rule processing.
func (s *ShipmentService) SetAutomationService(automationSvc *AutomationService) {
	s.automationService = automationSvc
}

// SetAllegroSyncService sets the Allegro sync service for auto-syncing tracking info.
// Called after construction to avoid circular dependency.
func (s *ShipmentService) SetAllegroSyncService(allegroSync *AllegroSyncService) {
	s.allegroSync = allegroSync
}

// NewShipmentService creates a new ShipmentService.
func NewShipmentService(
	shipmentRepo repository.ShipmentRepo,
	orderRepo repository.OrderRepo,
	productRepo repository.ProductRepo,
	auditRepo repository.AuditRepo,
	tenantRepo repository.TenantRepo,
	pool *pgxpool.Pool,
	webhookDispatch *WebhookDispatchService,
	objectStorage storage.ObjectStorage,
) *ShipmentService {
	return &ShipmentService{
		shipmentRepo:    shipmentRepo,
		orderRepo:       orderRepo,
		productRepo:     productRepo,
		auditRepo:       auditRepo,
		tenantRepo:      tenantRepo,
		pool:            pool,
		webhookDispatch: webhookDispatch,
		objectStorage:   objectStorage,
	}
}

// ListByOrder returns all shipments for a given order, sorted by package_number.
func (s *ShipmentService) ListByOrder(ctx context.Context, tenantID, orderID uuid.UUID) ([]model.Shipment, error) {
	var shipments []model.Shipment
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		filter := model.ShipmentListFilter{
			OrderID:          &orderID,
			PaginationParams: model.PaginationParams{Limit: 1000, Offset: 0, SortBy: "package_number", SortOrder: "asc"},
		}
		result, _, err := s.shipmentRepo.List(ctx, tx, filter)
		if err != nil {
			return err
		}
		if result == nil {
			result = []model.Shipment{}
		}
		shipments = result
		return nil
	})
	return shipments, err
}

// List returns a paginated list of shipments for a tenant.
func (s *ShipmentService) List(ctx context.Context, tenantID uuid.UUID, filter model.ShipmentListFilter) (model.ListResponse[model.Shipment], error) {
	var resp model.ListResponse[model.Shipment]
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		shipments, total, err := s.shipmentRepo.List(ctx, tx, filter)
		if err != nil {
			return err
		}
		resp = model.NewListResponse(shipments, total, filter.Limit, filter.Offset)
		return nil
	})
	return resp, err
}

// Get returns a single shipment by ID.
func (s *ShipmentService) Get(ctx context.Context, tenantID, shipmentID uuid.UUID) (*model.Shipment, error) {
	var shipment *model.Shipment
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var err error
		shipment, err = s.shipmentRepo.FindByID(ctx, tx, shipmentID)
		return err
	})
	if err != nil {
		return nil, err
	}
	if shipment == nil {
		return nil, ErrShipmentNotFound
	}
	return shipment, nil
}

// Create inserts a new shipment.
func (s *ShipmentService) Create(ctx context.Context, tenantID uuid.UUID, req model.CreateShipmentRequest, actorID uuid.UUID, ip string) (*model.Shipment, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	carrierData := req.CarrierData
	if carrierData == nil {
		carrierData = json.RawMessage("{}")
	}

	shipment := &model.Shipment{
		ID:             uuid.New(),
		TenantID:       tenantID,
		OrderID:        req.OrderID,
		Provider:       req.Provider,
		IntegrationID:  req.IntegrationID,
		ExternalID:     req.ExternalID,
		TrackingNumber: req.TrackingNumber,
		Status:         "created",
		LabelURL:       req.LabelURL,
		CarrierData:    carrierData,
		WarehouseID:    req.WarehouseID,
		Weight:         req.Weight,
		Length:         req.Length,
		Width:          req.Width,
		Height:         req.Height,
		Notes:          &req.Notes,
	}

	var associatedOrder *model.Order
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		order, err := s.orderRepo.FindByID(ctx, tx, req.OrderID)
		if err != nil {
			return err
		}
		if order == nil {
			return ErrOrderNotFoundForShipment
		}
		associatedOrder = order

		// Auto-fill weight from order products if not provided
		if shipment.Weight == nil {
			if w := s.calculateOrderWeight(ctx, tx, order); w > 0 {
				shipment.Weight = &w
			}
		}

		// Auto-assign package_number: count existing shipments for this order + 1
		count, err := s.shipmentRepo.CountByOrder(ctx, tx, req.OrderID)
		if err != nil {
			return fmt.Errorf("count existing shipments: %w", err)
		}
		shipment.PackageNumber = count + 1

		// Auto-calculate carbon footprint estimate
		s.estimateCarbon(ctx, tx, tenantID, shipment, order)

		if err := s.shipmentRepo.Create(ctx, tx, shipment); err != nil {
			return err
		}
		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "shipment.created",
			EntityType: "shipment",
			EntityID:   shipment.ID,
			Changes:    map[string]string{"order_id": req.OrderID.String(), "provider": req.Provider},
			IPAddress:  ip,
		})
	})
	if err != nil {
		return nil, err
	}
	DispatchWebhookAsync(s.webhookDispatch, tenantID, "shipment.created", shipment)
	FireAutomationEvent(s.automationService, tenantID, "shipment", "shipment.created", shipment.ID, map[string]any{
		"status": shipment.Status, "provider": shipment.Provider, "order_id": shipment.OrderID.String(),
	})
	// OPE-417: record the create_shipment provider attempt + emit the canonical
	// fulfillment step (gated, best-effort — never affects the result above).
	if s.fulfillment.Enabled() {
		s.fulfillment.RecordProviderAttempt(ctx, ProviderAttemptInput{
			TenantID:      tenantID,
			OrderID:       shipment.OrderID,
			Provider:      shipment.Provider,
			Operation:     model.ProviderOpCreateShipment,
			Status:        model.ProviderAttemptSucceeded,
			CorrelationID: shipment.ID.String(),
			RequestID:     externalIDOrEmpty(shipment),
			ResultRedacted: map[string]any{
				"shipment_status": shipment.Status,
				"package_number":  shipment.PackageNumber,
			},
		})
		s.fulfillment.EmitFulfillmentStep(ctx, tenantID, shipment.OrderID,
			model.StepCreateShipment, model.FulfillmentStatusSucceeded, shipment.ID.String(),
			map[string]any{"provider": shipment.Provider})
	}
	// Auto-sync tracking to Allegro if shipment has a tracking number (async, best-effort)
	if s.allegroSync != nil && shipment.TrackingNumber != nil && *shipment.TrackingNumber != "" && associatedOrder != nil {
		asyncutil.SafeGo(func() {
			s.allegroSync.SyncTracking(context.Background(), tenantID, associatedOrder, shipment.Provider, *shipment.TrackingNumber)
		})
	}
	return shipment, nil
}

// Update modifies an existing shipment.
func (s *ShipmentService) Update(ctx context.Context, tenantID, shipmentID uuid.UUID, req model.UpdateShipmentRequest, actorID uuid.UUID, ip string) (*model.Shipment, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	var shipment *model.Shipment
	var associatedOrder *model.Order
	trackingChanged := false
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		existing, err := s.shipmentRepo.FindByID(ctx, tx, shipmentID)
		if err != nil {
			return err
		}
		if existing == nil {
			return ErrShipmentNotFound
		}

		// Detect if tracking number is being set or changed
		if req.TrackingNumber != nil && *req.TrackingNumber != "" {
			oldTracking := ""
			if existing.TrackingNumber != nil {
				oldTracking = *existing.TrackingNumber
			}
			if *req.TrackingNumber != oldTracking {
				trackingChanged = true
			}
		}

		if err := s.shipmentRepo.Update(ctx, tx, shipmentID, req); err != nil {
			return err
		}

		shipment, err = s.shipmentRepo.FindByID(ctx, tx, shipmentID)
		if err != nil {
			return err
		}

		// Fetch associated order for Allegro sync if tracking changed
		if trackingChanged && s.allegroSync != nil {
			order, err := s.orderRepo.FindByID(ctx, tx, shipment.OrderID)
			if err == nil && order != nil {
				associatedOrder = order
			}
		}

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "shipment.updated",
			EntityType: "shipment",
			EntityID:   shipmentID,
			IPAddress:  ip,
		})
	})
	if err == nil && shipment != nil {
		DispatchWebhookAsync(s.webhookDispatch, tenantID, "shipment.updated", shipment)
		// Auto-sync tracking to Allegro when tracking number is set/changed (async, best-effort)
		if trackingChanged && s.allegroSync != nil && shipment.TrackingNumber != nil && *shipment.TrackingNumber != "" && associatedOrder != nil {
			asyncutil.SafeGo(func() {
				s.allegroSync.SyncTracking(context.Background(), tenantID, associatedOrder, shipment.Provider, *shipment.TrackingNumber)
			})
		}
	}
	return shipment, err
}

// Delete removes a shipment by ID.
func (s *ShipmentService) Delete(ctx context.Context, tenantID, shipmentID, actorID uuid.UUID, ip string) error {
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		shipment, err := s.shipmentRepo.FindByID(ctx, tx, shipmentID)
		if err != nil {
			return err
		}
		if shipment == nil {
			return ErrShipmentNotFound
		}

		if err := s.shipmentRepo.Delete(ctx, tx, shipmentID); err != nil {
			return err
		}

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "shipment.deleted",
			EntityType: "shipment",
			EntityID:   shipmentID,
			Changes:    map[string]string{"order_id": shipment.OrderID.String()},
			IPAddress:  ip,
		})
	})
	if err == nil {
		DispatchWebhookAsync(s.webhookDispatch, tenantID, "shipment.deleted", map[string]any{"shipment_id": shipmentID.String()})
	}
	return err
}

// TransitionStatus moves a shipment to a new carrier status.
func (s *ShipmentService) TransitionStatus(ctx context.Context, tenantID, shipmentID uuid.UUID, req model.ShipmentStatusTransitionRequest, actorID uuid.UUID, ip string) (*model.Shipment, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	var shipment *model.Shipment
	var orderChange *orderStatusChange
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		orderChange = nil
		existing, err := s.shipmentRepo.FindByID(ctx, tx, shipmentID)
		if err != nil {
			return err
		}
		if existing == nil {
			return ErrShipmentNotFound
		}

		currentStatus, err := engine.ParseShipmentStatus(existing.Status)
		if err != nil {
			return err
		}

		targetStatus, err := engine.ParseShipmentStatus(req.Status)
		if err != nil {
			return err
		}

		if _, err := engine.TransitionShipment(currentStatus, targetStatus, time.Now()); err != nil {
			return err
		}

		if err := s.shipmentRepo.UpdateStatus(ctx, tx, shipmentID, req.Status); err != nil {
			return err
		}

		shipment, err = s.shipmentRepo.FindByID(ctx, tx, shipmentID)
		if err != nil {
			return err
		}

		if err := s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "shipment.status_changed",
			EntityType: "shipment",
			EntityID:   shipmentID,
			Changes:    map[string]string{"from": existing.Status, "to": req.Status},
			IPAddress:  ip,
		}); err != nil {
			return err
		}

		// Order-shipment status sync (multi-package aware). The write stays in this
		// transaction; its post-commit fan-out is applied below.
		orderChange, err = s.syncOrderStatusForShipment(ctx, tx, tenantID, existing, req.Status, actorID, ip)
		return err
	})
	if err == nil && shipment != nil {
		DispatchWebhookAsync(s.webhookDispatch, tenantID, "shipment.status_changed", shipment)
		if s.smsService != nil {
			asyncutil.SafeGo(func() { s.smsService.SendShipmentStatusSMS(context.Background(), tenantID, shipment, "") })
		}
		FireAutomationEvent(s.automationService, tenantID, "shipment", "shipment.status_changed", shipment.ID, map[string]any{
			"status": shipment.Status, "provider": shipment.Provider, "order_id": shipment.OrderID.String(),
		})
		// The synced order status owes the same side effects an operator-driven
		// transition would fire — most importantly the stock decrement on first ship,
		// which used to be skipped entirely for carrier-driven ships.
		if orderChange != nil && s.orderStatusFanout != nil {
			s.orderStatusFanout.applyStatusChangeSideEffects(tenantID, *orderChange)
		}
	}
	return shipment, err
}

// syncOrderStatusForShipment mirrors a shipment's new carrier status onto its order
// inside the caller's transaction and returns the change whose side effects must fan
// out after commit, or nil when the shipment status implies no order change.
//
// The order status is written directly (not through OrderService.TransitionStatus) so
// it stays atomic with the shipment status: a failed order write rolls the shipment
// transition back. The carrier is the authority on what physically happened, so the
// tenant transition graph is not consulted — but the audit entry and the post-commit
// fan-out now match what TransitionStatus would have produced.
func (s *ShipmentService) syncOrderStatusForShipment(
	ctx context.Context,
	tx pgx.Tx,
	tenantID uuid.UUID,
	shipment *model.Shipment,
	newShipmentStatus string,
	actorID uuid.UUID,
	ip string,
) (*orderStatusChange, error) {
	order, err := s.orderRepo.FindByID(ctx, tx, shipment.OrderID)
	if err != nil {
		return nil, fmt.Errorf("load order for shipment status sync: %w", err)
	}
	if order == nil {
		return nil, nil
	}

	now := time.Now()
	var targetStatus string
	var setShippedAt, setDeliveredAt *time.Time

	switch newShipmentStatus {
	case "delivered":
		// Only mark the order delivered when ALL its shipments are delivered.
		allDelivered, err := s.allOrderShipmentsDelivered(ctx, tx, shipment)
		if err != nil {
			return nil, err
		}
		if !allDelivered {
			return nil, nil
		}
		targetStatus = model.OrderStatusDelivered
		setDeliveredAt = &now
	case "picked_up", "in_transit":
		if order.Status == model.OrderStatusShipped || order.Status == model.OrderStatusDelivered {
			return nil, nil
		}
		targetStatus = model.OrderStatusShipped
		setShippedAt = &now
	default:
		return nil, nil
	}

	if order.Status == targetStatus {
		return nil, nil // already there: no write, and no duplicate side effects
	}

	oldStatus := order.Status
	// shipped_at is only ever set (COALESCE on update), so this gates the one-time
	// stock decrement even if several packages report movement.
	firstShip := order.ShippedAt == nil

	if err := s.orderRepo.UpdateStatus(ctx, tx, order.ID, targetStatus, setShippedAt, setDeliveredAt); err != nil {
		return nil, fmt.Errorf("sync order status to %s: %w", targetStatus, err)
	}
	if err := s.auditRepo.Log(ctx, tx, model.AuditEntry{
		TenantID:   tenantID,
		UserID:     actorID,
		Action:     "order.status_changed",
		EntityType: "order",
		EntityID:   order.ID,
		Changes:    map[string]string{"from": oldStatus, "to": targetStatus, "shipment_id": shipment.ID.String()},
		IPAddress:  ip,
	}); err != nil {
		return nil, fmt.Errorf("audit synced order status: %w", err)
	}

	updated, err := s.orderRepo.FindByID(ctx, tx, order.ID)
	if err != nil {
		return nil, fmt.Errorf("reload order after shipment status sync: %w", err)
	}
	return &orderStatusChange{
		orderID:   order.ID,
		order:     updated,
		oldStatus: oldStatus,
		newStatus: targetStatus,
		firstShip: firstShip,
	}, nil
}

// allOrderShipmentsDelivered reports whether every shipment of the order is delivered,
// treating the just-updated shipment as delivered.
func (s *ShipmentService) allOrderShipmentsDelivered(ctx context.Context, tx pgx.Tx, updated *model.Shipment) (bool, error) {
	orderShipments, _, err := s.shipmentRepo.List(ctx, tx, model.ShipmentListFilter{
		OrderID:          &updated.OrderID,
		PaginationParams: model.PaginationParams{Limit: 1000, Offset: 0},
	})
	if err != nil {
		return false, fmt.Errorf("list order shipments for status sync: %w", err)
	}
	for _, os := range orderShipments {
		if os.ID == updated.ID {
			continue // the one we just updated — it is now "delivered"
		}
		if os.Status != "delivered" {
			return false, nil
		}
	}
	return true, nil
}

// GetLabelFile loads the already-stored label PDF for a single shipment.
func (s *ShipmentService) GetLabelFile(ctx context.Context, tenantID, shipmentID uuid.UUID) ([]byte, error) {
	var labelURL string
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		shipment, err := s.shipmentRepo.FindByID(ctx, tx, shipmentID)
		if err != nil {
			return err
		}
		if shipment == nil {
			return ErrShipmentNotFound
		}
		if shipment.LabelURL == nil || *shipment.LabelURL == "" {
			return ErrLabelNotAvailable
		}
		labelURL = *shipment.LabelURL
		return nil
	})
	if err != nil {
		return nil, err
	}
	return readLabelObject(ctx, s.objectStorage, labelURL)
}

// GetBatchLabelURLs loads label data for multiple shipments from ObjectStorage
// using the label_url stored on each shipment.
func (s *ShipmentService) GetBatchLabelURLs(ctx context.Context, tenantID uuid.UUID, shipmentIDs []uuid.UUID) ([]model.BatchLabelResult, error) {
	type pending struct {
		id  uuid.UUID
		url string
	}
	var toRead []pending

	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		for _, sid := range shipmentIDs {
			shipment, err := s.shipmentRepo.FindByID(ctx, tx, sid)
			if err != nil || shipment == nil || shipment.LabelURL == nil || *shipment.LabelURL == "" {
				continue
			}
			toRead = append(toRead, pending{id: sid, url: *shipment.LabelURL})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	var validResults []model.BatchLabelResult
	for _, item := range toRead {
		labelData, err := readLabelObject(ctx, s.objectStorage, item.url)
		if err != nil {
			continue
		}
		validResults = append(validResults, model.BatchLabelResult{
			ShipmentID: item.id.String(),
			Data:       labelData,
		})
	}

	return validResults, nil
}

// externalIDOrEmpty returns a shipment's carrier external id as a request
// correlation id, or "" when unset. Used only for best-effort attempt recording.
func externalIDOrEmpty(shipment *model.Shipment) string {
	if shipment == nil || shipment.ExternalID == nil {
		return ""
	}
	return *shipment.ExternalID
}

// calculateOrderWeight sums product weights for all items in the order.
// Returns 0 if any product lacks weight data or order has no items with product_id.
func (s *ShipmentService) calculateOrderWeight(ctx context.Context, tx pgx.Tx, order *model.Order) float64 {
	if s.productRepo == nil || len(order.Items) == 0 {
		return 0
	}

	type orderItem struct {
		ProductID *uuid.UUID `json:"product_id"`
		Quantity  int        `json:"quantity"`
	}
	var items []orderItem
	if err := json.Unmarshal(order.Items, &items); err != nil {
		return 0
	}

	// Collect unique product IDs and build quantity map.
	quantityByID := make(map[uuid.UUID]int)
	var productIDs []uuid.UUID
	for _, item := range items {
		if item.ProductID == nil || *item.ProductID == uuid.Nil || item.Quantity <= 0 {
			continue
		}
		if _, exists := quantityByID[*item.ProductID]; !exists {
			productIDs = append(productIDs, *item.ProductID)
		}
		quantityByID[*item.ProductID] += item.Quantity
	}
	if len(productIDs) == 0 {
		return 0
	}

	// Batch-fetch all products in a single query.
	products, err := s.productRepo.FindByIDs(ctx, tx, productIDs)
	if err != nil {
		return 0
	}

	var totalWeight float64
	for _, p := range products {
		if p.Weight == nil || *p.Weight <= 0 {
			continue
		}
		totalWeight += *p.Weight * float64(quantityByID[p.ID])
	}
	return totalWeight
}

// estimateCarbon auto-calculates carbon footprint estimate for a shipment.
// Best-effort: any errors are silently ignored (carbon is supplementary data).
func (s *ShipmentService) estimateCarbon(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID, shipment *model.Shipment, order *model.Order) {
	// Parse destination postal code from order's shipping address
	destPostalCode := ""
	if len(order.ShippingAddress) > 0 {
		var addr model.ShippingAddress
		if err := json.Unmarshal(order.ShippingAddress, &addr); err == nil {
			destPostalCode = addr.PostalCode
		}
	}

	if destPostalCode == "" {
		return // Cannot estimate without destination
	}

	// Get origin postal code from tenant's company settings
	originPostalCode := ""
	if s.tenantRepo != nil {
		tenant, err := s.tenantRepo.FindByID(ctx, tx, tenantID)
		if err == nil && tenant != nil && tenant.Settings != nil {
			var settings map[string]json.RawMessage
			if err := json.Unmarshal(tenant.Settings, &settings); err == nil {
				if companyRaw, ok := settings["company"]; ok {
					var company model.CompanySettings
					if err := json.Unmarshal(companyRaw, &company); err == nil {
						originPostalCode = company.PostCode
					}
				}
			}
		}
	}

	if originPostalCode == "" {
		originPostalCode = "00-001" // Default: Warsaw
	}

	distanceKm := EstimateDistance(originPostalCode, destPostalCode)
	weightKg := 1.0
	if shipment.Weight != nil && *shipment.Weight > 0 {
		weightKg = *shipment.Weight
	}
	carbonKg := EstimateCarbon(shipment.Provider, weightKg, distanceKm)

	shipment.DistanceKm = &distanceKm
	shipment.CarbonKg = &carbonKg
	method := "estimate"
	shipment.CarbonMethod = &method
}

// UpdateStatusByTrackingNumber finds a shipment by tracking number and provider
// (cross-tenant) and updates its status. Used by webhook handlers.
func (s *ShipmentService) UpdateStatusByTrackingNumber(ctx context.Context, trackingNumber, provider, newStatus string) error {
	if s.workerQuerier == nil {
		return errors.New("worker database pool is not configured")
	}

	// Cross-tenant lookup uses the explicit privileged worker pool. The normal
	// app pool is RLS-scoped and must not be used for public webhook lookups.
	// Scoped by both tracking_number AND provider to avoid ambiguity when
	// different carriers reuse the same tracking number format.
	var shipmentID, tenantID uuid.UUID
	var oldStatus string
	err := s.workerQuerier.QueryRow(ctx,
		"SELECT id, tenant_id, status FROM shipments WHERE tracking_number = $1 AND provider = $2 LIMIT 1",
		trackingNumber, provider,
	).Scan(&shipmentID, &tenantID, &oldStatus)
	if err != nil {
		return fmt.Errorf("find shipment by tracking number %q provider %q: %w", trackingNumber, provider, err)
	}

	return s.applyShipmentStatusChange(ctx, tenantID, shipmentID, oldStatus, newStatus)
}

// UpdateStatusByTrackingNumberForTenant updates a shipment status inside an already verified tenant scope.
func (s *ShipmentService) UpdateStatusByTrackingNumberForTenant(ctx context.Context, tenantID uuid.UUID, trackingNumber, provider, newStatus string) error {
	var shipmentID uuid.UUID
	var oldStatus string
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx,
			"SELECT id, status FROM shipments WHERE tracking_number = $1 AND provider = $2 LIMIT 1",
			trackingNumber,
			provider,
		).Scan(&shipmentID, &oldStatus)
	})
	if err != nil {
		return fmt.Errorf("find tenant shipment by tracking number %q provider %q: %w", trackingNumber, provider, err)
	}

	return s.applyShipmentStatusChange(ctx, tenantID, shipmentID, oldStatus, newStatus)
}

func (s *ShipmentService) applyShipmentStatusChange(ctx context.Context, tenantID, shipmentID uuid.UUID, oldStatus, newStatus string) error {
	if oldStatus == newStatus {
		return nil
	}

	if err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx,
			"UPDATE shipments SET status = $1, updated_at = NOW() WHERE id = $2",
			newStatus, shipmentID)
		return err
	}); err != nil {
		return fmt.Errorf("update shipment status: %w", err)
	}

	// Phase 2: Side effects AFTER commit — automation and webhooks
	eventData := map[string]any{
		"shipment_id": shipmentID,
		"old_status":  oldStatus,
		"status":      newStatus,
	}
	if s.automationService != nil {
		FireAutomationEvent(s.automationService, tenantID, "shipment", "shipment.status_changed", shipmentID, eventData)
	}
	DispatchWebhookAsync(s.webhookDispatch, tenantID, "shipment.status_changed", eventData)

	return nil
}
