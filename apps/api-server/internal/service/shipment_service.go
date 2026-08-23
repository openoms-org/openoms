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
	orderStatusSyncer OrderStatusSyncer
}

// SetOrderStatusSyncer wires the shared order-status writer (OrderService) so a carrier
// event that moves the order also runs the order's own side effects — stock, invoice,
// notifications, automation, loyalty. Called after construction to avoid a circular
// dependency. Left unwired, shipment-driven status changes fall back to a bare status
// write and those effects do not happen.
func (s *ShipmentService) SetOrderStatusSyncer(syncer OrderStatusSyncer) {
	s.orderStatusSyncer = syncer
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
	// Carrier-driven order status change, applied in this transaction and replayed as
	// side effects after it commits.
	var orderChange *OrderStatusChange
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
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

		// Order-shipment status sync (multi-package aware)
		switch req.Status {
		case "delivered":
			// Only mark order as delivered if ALL shipments for this order are delivered
			allDelivered := true
			orderShipments, _, err := s.shipmentRepo.List(ctx, tx, model.ShipmentListFilter{
				OrderID:          &existing.OrderID,
				PaginationParams: model.PaginationParams{Limit: 1000, Offset: 0},
			})
			if err != nil {
				return fmt.Errorf("list order shipments for status sync: %w", err)
			}
			for _, os := range orderShipments {
				if os.ID == shipmentID {
					continue // skip the one we just updated — it's now "delivered"
				}
				if os.Status != "delivered" {
					allDelivered = false
					break
				}
			}
			if allDelivered {
				orderChange, err = s.syncOrderStatus(ctx, tx, tenantID, existing.OrderID, model.OrderStatusDelivered, actorID, ip)
				if err != nil {
					return fmt.Errorf("sync order status to delivered: %w", err)
				}
			}
		case "picked_up", "in_transit":
			order, err := s.orderRepo.FindByID(ctx, tx, existing.OrderID)
			if err == nil && order != nil && order.Status != model.OrderStatusShipped && order.Status != model.OrderStatusDelivered {
				orderChange, err = s.syncOrderStatus(ctx, tx, tenantID, existing.OrderID, model.OrderStatusShipped, actorID, ip)
				if err != nil {
					return fmt.Errorf("sync order status to shipped: %w", err)
				}
			}
		}

		return nil
	})
	if err == nil && shipment != nil {
		// After commit, so the order's own consequences (stock decrement on first ship,
		// invoice, notifications, automation) cannot roll back the carrier event.
		if s.orderStatusSyncer != nil && orderChange != nil {
			s.orderStatusSyncer.FireOrderStatusEffects(tenantID, orderChange)
		}
		DispatchWebhookAsync(s.webhookDispatch, tenantID, "shipment.status_changed", shipment)
		if s.smsService != nil {
			asyncutil.SafeGo(func() { s.smsService.SendShipmentStatusSMS(context.Background(), tenantID, shipment, "") })
		}
		FireAutomationEvent(s.automationService, tenantID, "shipment", "shipment.status_changed", shipment.ID, map[string]any{
			"status": shipment.Status, "provider": shipment.Provider, "order_id": shipment.OrderID.String(),
		})
	}
	return shipment, err
}

// syncOrderStatus applies a carrier-driven order status change through the shared
// order-status writer, inside the shipment's own transaction so the two rows move
// together. The returned change carries the post-commit side effects; a nil change
// means the caller has nothing to fire.
//
// The tenant transition graph is deliberately not consulted: a parcel that has been
// picked up or delivered is a fact reported by the carrier, not a request that the
// order's current status may veto. What used to be wrong is that the fact was recorded
// without any of its consequences.
func (s *ShipmentService) syncOrderStatus(ctx context.Context, tx pgx.Tx, tenantID, orderID uuid.UUID, newStatus string, actorID uuid.UUID, ip string) (*OrderStatusChange, error) {
	if s.orderStatusSyncer != nil {
		return s.orderStatusSyncer.WriteOrderStatusInTx(ctx, tx, tenantID, orderID, newStatus, actorID, ip)
	}
	// Unwired fallback: preserve the historical bare status write rather than dropping
	// the carrier event entirely.
	var setShippedAt, setDeliveredAt *time.Time
	now := time.Now()
	switch newStatus {
	case model.OrderStatusShipped:
		setShippedAt = &now
	case model.OrderStatusDelivered:
		setDeliveredAt = &now
	}
	return nil, s.orderRepo.UpdateStatus(ctx, tx, orderID, newStatus, setShippedAt, setDeliveredAt)
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
