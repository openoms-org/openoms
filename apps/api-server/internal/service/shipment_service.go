package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	engine "github.com/openoms-org/openoms/packages/order-engine"

	"github.com/openoms-org/openoms/apps/api-server/internal/asyncutil"
	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

var (
	// ErrShipmentNotFound is returned when a shipment does not exist.
	ErrShipmentNotFound = errors.New("shipment not found")
	// ErrOrderNotFoundForShipment is returned when a shipment's associated order cannot be found.
	ErrOrderNotFoundForShipment = errors.New("order not found for shipment")
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
) *ShipmentService {
	return &ShipmentService{
		shipmentRepo:    shipmentRepo,
		orderRepo:       orderRepo,
		productRepo:     productRepo,
		auditRepo:       auditRepo,
		tenantRepo:      tenantRepo,
		pool:            pool,
		webhookDispatch: webhookDispatch,
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
		if shipments == nil {
			shipments = []model.Shipment{}
		}
		resp = model.ListResponse[model.Shipment]{
			Items:  shipments,
			Total:  total,
			Limit:  filter.Limit,
			Offset: filter.Offset,
		}
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
	if s.webhookDispatch != nil {
		asyncutil.SafeGo(func() { s.webhookDispatch.Dispatch(context.Background(), tenantID, "shipment.created", shipment) })
	}
	FireAutomationEvent(s.automationService, tenantID, "shipment", "shipment.created", shipment.ID, map[string]any{
		"status": shipment.Status, "provider": shipment.Provider, "order_id": shipment.OrderID.String(),
	})
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
		if s.webhookDispatch != nil {
			asyncutil.SafeGo(func() { s.webhookDispatch.Dispatch(context.Background(), tenantID, "shipment.updated", shipment) })
		}
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
		if s.webhookDispatch != nil {
			asyncutil.SafeGo(func() {
				s.webhookDispatch.Dispatch(context.Background(), tenantID, "shipment.deleted", map[string]any{"shipment_id": shipmentID.String()})
			})
		}
	}
	return err
}

// TransitionStatus moves a shipment to a new carrier status.
func (s *ShipmentService) TransitionStatus(ctx context.Context, tenantID, shipmentID uuid.UUID, req model.ShipmentStatusTransitionRequest, actorID uuid.UUID, ip string) (*model.Shipment, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	var shipment *model.Shipment
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
				if err := s.orderRepo.UpdateStatus(ctx, tx, existing.OrderID, "delivered", nil, func() *time.Time { t := time.Now(); return &t }()); err != nil {
					return fmt.Errorf("sync order status to delivered: %w", err)
				}
			}
		case "picked_up", "in_transit":
			order, err := s.orderRepo.FindByID(ctx, tx, existing.OrderID)
			if err == nil && order != nil && order.Status != "shipped" && order.Status != "delivered" {
				now := time.Now()
				if err := s.orderRepo.UpdateStatus(ctx, tx, existing.OrderID, "shipped", &now, nil); err != nil {
					return fmt.Errorf("sync order status to shipped: %w", err)
				}
			}
		}

		return nil
	})
	if err == nil && shipment != nil {
		if s.webhookDispatch != nil {
			asyncutil.SafeGo(func() {
				s.webhookDispatch.Dispatch(context.Background(), tenantID, "shipment.status_changed", shipment)
			})
		}
		if s.smsService != nil {
			asyncutil.SafeGo(func() { s.smsService.SendShipmentStatusSMS(context.Background(), tenantID, shipment, "") })
		}
		FireAutomationEvent(s.automationService, tenantID, "shipment", "shipment.status_changed", shipment.ID, map[string]any{
			"status": shipment.Status, "provider": shipment.Provider, "order_id": shipment.OrderID.String(),
		})
	}
	return shipment, err
}

// GetBatchLabelURLs loads label data for multiple shipments.
// It reads label files from disk based on the label_url stored in each shipment.
func (s *ShipmentService) GetBatchLabelURLs(ctx context.Context, tenantID uuid.UUID, shipmentIDs []uuid.UUID) ([]model.BatchLabelResult, error) {
	var results []model.BatchLabelResult

	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		for _, sid := range shipmentIDs {
			shipment, err := s.shipmentRepo.FindByID(ctx, tx, sid)
			if err != nil {
				results = append(results, model.BatchLabelResult{
					ShipmentID: sid.String(),
					Error:      "failed to load shipment",
				})
				continue
			}
			if shipment == nil {
				results = append(results, model.BatchLabelResult{
					ShipmentID: sid.String(),
					Error:      "shipment not found",
				})
				continue
			}
			if shipment.LabelURL == nil || *shipment.LabelURL == "" {
				results = append(results, model.BatchLabelResult{
					ShipmentID: sid.String(),
					Error:      "no label available",
				})
				continue
			}

			labelData, err := readLabelFile(*shipment.LabelURL)
			if err != nil {
				results = append(results, model.BatchLabelResult{
					ShipmentID: sid.String(),
					Error:      "label file not found",
				})
				continue
			}
			results = append(results, model.BatchLabelResult{
				ShipmentID: sid.String(),
				Data:       labelData,
			})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Filter to only results with data
	var validResults []model.BatchLabelResult
	for _, r := range results {
		if r.Data != nil {
			validResults = append(validResults, r)
		}
	}

	return validResults, nil
}

// readLabelFile reads a label file from disk. The URL is in the form:
// {baseURL}/uploads/{tenantID}/{filename}
// We extract the path from the URL and read the file.
func readLabelFile(labelURL string) ([]byte, error) {
	// The label URL looks like: http://localhost:8080/uploads/tenant-uuid/filename.pdf
	// We need to extract the path after /uploads/ and resolve it to disk.
	idx := 0
	const marker = "/uploads/"
	for i := range labelURL {
		if i+len(marker) <= len(labelURL) && labelURL[i:i+len(marker)] == marker {
			idx = i + len(marker)
			break
		}
	}
	if idx == 0 {
		return nil, fmt.Errorf("invalid label URL format: %s", labelURL)
	}

	relPath := filepath.Clean(labelURL[idx:])
	// Prevent path traversal
	if strings.HasPrefix(relPath, "..") || filepath.IsAbs(relPath) {
		return nil, fmt.Errorf("invalid label path")
	}

	// Try common upload directories
	for _, base := range []string{"uploads", "./uploads", "/tmp/uploads"} {
		fullPath := filepath.Join(base, relPath)
		data, err := os.ReadFile(fullPath) //nolint:gosec // path validated: Clean + traversal check above
		if err == nil {
			return data, nil
		}
	}

	return nil, fmt.Errorf("label file not found for URL: %s", labelURL)
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

	// Skip if status hasn't changed
	if oldStatus == newStatus {
		return nil
	}

	// Phase 1: Update status in DB within tenant context
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
	if s.webhookDispatch != nil {
		s.webhookDispatch.Dispatch(ctx, tenantID, "shipment.status_changed", eventData)
	}

	return nil
}
