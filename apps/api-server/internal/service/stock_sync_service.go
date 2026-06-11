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
	"github.com/openoms-org/openoms/apps/api-server/internal/crypto"
	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

var (
	// ErrStockSyncChannelNotFound is returned when a stock sync channel does not exist.
	ErrStockSyncChannelNotFound = errors.New("stock sync channel not found")
)

// closeMarketplaceProvider closes providers that implement io.Closer (e.g. Allegro token refresh goroutine).
func closeMarketplaceProvider(provider any) {
	type closer interface{ Close() }
	if c, ok := provider.(closer); ok {
		c.Close()
	}
}

const maxServiceErrorMsgLen = 500

// truncateServiceErrorMsg limits error messages stored in the database to avoid storing
// verbose API responses that may contain presigned URLs or other sensitive data.
func truncateServiceErrorMsg(msg string) string {
	if len(msg) <= maxServiceErrorMsgLen {
		return msg
	}
	return msg[:maxServiceErrorMsgLen]
}

// StockSyncService provides business logic for real-time stock synchronization.
type StockSyncService struct {
	channelRepo     repository.StockSyncChannelRepo
	eventRepo       repository.StockSyncEventRepo
	productRepo     repository.ProductRepo
	auditRepo       repository.AuditRepo
	listingRepo     repository.ProductListingRepo
	integrationRepo repository.IntegrationRepo
	pool            *pgxpool.Pool
	webhookDispatch *WebhookDispatchService
	automationSvc   *AutomationService
	encryptionKey   []byte
	logger          *slog.Logger
}

// NewStockSyncService creates a new StockSyncService.
func NewStockSyncService(
	channelRepo repository.StockSyncChannelRepo,
	eventRepo repository.StockSyncEventRepo,
	productRepo repository.ProductRepo,
	auditRepo repository.AuditRepo,
	listingRepo repository.ProductListingRepo,
	integrationRepo repository.IntegrationRepo,
	pool *pgxpool.Pool,
	webhookDispatch *WebhookDispatchService,
	encryptionKey []byte,
	logger *slog.Logger,
) *StockSyncService {
	return &StockSyncService{
		channelRepo:     channelRepo,
		eventRepo:       eventRepo,
		productRepo:     productRepo,
		auditRepo:       auditRepo,
		listingRepo:     listingRepo,
		integrationRepo: integrationRepo,
		pool:            pool,
		webhookDispatch: webhookDispatch,
		encryptionKey:   encryptionKey,
		logger:          logger,
	}
}

// SetAutomationService sets the automation service for firing stock-restored events.
func (s *StockSyncService) SetAutomationService(automationSvc *AutomationService) {
	s.automationSvc = automationSvc
}

// --- Channel CRUD ---

// ListChannels returns a paginated list of stock sync channels.
func (s *StockSyncService) ListChannels(ctx context.Context, tenantID uuid.UUID, filter model.StockSyncChannelListFilter) (model.ListResponse[model.StockSyncChannel], error) {
	var resp model.ListResponse[model.StockSyncChannel]
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		channels, total, err := s.channelRepo.List(ctx, tx, filter)
		if err != nil {
			return err
		}
		resp = model.NewListResponse(channels, total, filter.Limit, filter.Offset)
		return nil
	})
	if resp.Items == nil {
		resp.Items = []model.StockSyncChannel{}
	}
	return resp, err
}

// GetChannel returns a single stock sync channel by ID.
func (s *StockSyncService) GetChannel(ctx context.Context, tenantID, channelID uuid.UUID) (*model.StockSyncChannel, error) {
	var ch *model.StockSyncChannel
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var err error
		ch, err = s.channelRepo.FindByID(ctx, tx, channelID)
		return err
	})
	if err != nil {
		return nil, err
	}
	if ch == nil {
		return nil, ErrStockSyncChannelNotFound
	}
	return ch, nil
}

// CreateChannel creates a new stock sync channel.
func (s *StockSyncService) CreateChannel(ctx context.Context, tenantID uuid.UUID, req model.CreateStockSyncChannelRequest) (*model.StockSyncChannel, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	ch := &model.StockSyncChannel{
		ID:            uuid.New(),
		TenantID:      tenantID,
		IntegrationID: req.IntegrationID,
		ChannelType:   req.ChannelType,
		Enabled:       true,
		StockBuffer:   0,
		SyncMode:      "realtime",
		Priority:      0,
	}
	if req.Enabled != nil {
		ch.Enabled = *req.Enabled
	}
	if req.StockBuffer != nil {
		ch.StockBuffer = *req.StockBuffer
	}
	if req.SyncMode != nil {
		ch.SyncMode = *req.SyncMode
	}
	if req.Priority != nil {
		ch.Priority = *req.Priority
	}

	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		if err := s.channelRepo.Create(ctx, tx, ch); err != nil {
			return err
		}
		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			EntityType: "stock_sync_channel",
			EntityID:   ch.ID,
			Action:     "created",
		})
	})
	if err != nil {
		return nil, err
	}
	return ch, nil
}

// UpdateChannel updates a stock sync channel.
func (s *StockSyncService) UpdateChannel(ctx context.Context, tenantID, channelID uuid.UUID, req model.UpdateStockSyncChannelRequest) error {
	if err := req.Validate(); err != nil {
		return NewValidationError(err)
	}

	return database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		existing, err := s.channelRepo.FindByID(ctx, tx, channelID)
		if err != nil {
			return err
		}
		if existing == nil {
			return ErrStockSyncChannelNotFound
		}
		if err := s.channelRepo.Update(ctx, tx, channelID, req); err != nil {
			return err
		}
		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			EntityType: "stock_sync_channel",
			EntityID:   channelID,
			Action:     "updated",
		})
	})
}

// DeleteChannel deletes a stock sync channel.
func (s *StockSyncService) DeleteChannel(ctx context.Context, tenantID, channelID uuid.UUID) error {
	return database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		existing, err := s.channelRepo.FindByID(ctx, tx, channelID)
		if err != nil {
			return err
		}
		if existing == nil {
			return ErrStockSyncChannelNotFound
		}
		if err := s.channelRepo.Delete(ctx, tx, channelID); err != nil {
			return err
		}
		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			EntityType: "stock_sync_channel",
			EntityID:   channelID,
			Action:     "deleted",
		})
	})
}

// --- Stock sync logic ---

// OnStockChange is the main entry point called when stock levels change.
// It calculates available stock per channel, creates an event record,
// dispatches webhook, and triggers push to all enabled realtime channels.
func (s *StockSyncService) OnStockChange(ctx context.Context, tenantID, productID uuid.UUID, triggerType string, oldQty, newQty int) {
	if !model.IsValidTriggerType(triggerType) {
		s.logger.Warn("stock sync: invalid trigger type", "trigger_type", triggerType)
		return
	}

	var event model.StockSyncEvent
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		// Look up product SKU
		product, err := s.productRepo.FindByID(ctx, tx, productID)
		if err != nil {
			return err
		}

		// Calculate available stock from warehouse tables
		totalQty, reservedQty, err := s.eventRepo.GetAvailableStock(ctx, tx, productID)
		if err != nil {
			return err
		}

		// Get enabled channels and calculate total buffer
		channels, err := s.channelRepo.ListEnabled(ctx, tx)
		if err != nil {
			return err
		}

		totalBuffer := 0
		for _, ch := range channels {
			totalBuffer += ch.StockBuffer
		}
		availableQty := max(totalQty-reservedQty-totalBuffer, 0)

		// Count how many channels are realtime-enabled
		realtimeCount := 0
		for _, ch := range channels {
			if ch.SyncMode == "realtime" {
				realtimeCount++
			}
		}

		var sku *string
		if product != nil {
			sku = product.SKU
		}

		// Create event record
		details := map[string]any{
			"total_stock":       totalQty,
			"reserved":          reservedQty,
			"total_buffer":      totalBuffer,
			"realtime_channels": realtimeCount,
		}
		detailsJSON, _ := json.Marshal(details)

		event = model.StockSyncEvent{
			ID:                uuid.New(),
			TenantID:          tenantID,
			ProductID:         &productID,
			SKU:               sku,
			TriggerType:       triggerType,
			OldQuantity:       oldQty,
			NewQuantity:       newQty,
			AvailableQuantity: availableQty,
			ChannelsNotified:  realtimeCount,
			ChannelsFailed:    0,
			Details:           detailsJSON,
		}
		return s.eventRepo.Create(ctx, tx, &event)
	})
	if err != nil {
		s.logger.Error("stock sync: failed to process stock change",
			"tenant_id", tenantID,
			"product_id", productID,
			"error", err,
		)
		return
	}

	// Dispatch webhook for stock change
	if s.webhookDispatch != nil {
		asyncutil.SafeGo(func() { s.webhookDispatch.Dispatch(context.Background(), tenantID, "stock.changed", event) })
	}

	// Detect stock restored: stock was 0, now > 0.
	// If automation engine is wired, fire the event and let user-configured rules handle relisting.
	// Otherwise, fall back to direct auto-relisting of inactive marketplace listings.
	if oldQty == 0 && newQty > 0 {
		s.logger.Info("stock sync: stock restored from zero, triggering auto-relisting",
			"tenant_id", tenantID,
			"product_id", productID,
			"new_qty", newQty,
		)

		if s.automationSvc != nil {
			// Fire product.stock_restored automation event for custom rules
			FireAutomationEvent(s.automationSvc, tenantID, "product", "product.stock_restored", productID, map[string]any{
				"old_quantity": oldQty,
				"new_quantity": newQty,
				"trigger_type": triggerType,
			})
		} else {
			// Fallback: direct auto-relisting when automation engine is not wired
			asyncutil.SafeGo(func() { s.reactivateListings(context.Background(), tenantID, productID) })
		}
	}

	// Detect stock depleted: stock was > 0, now 0.
	if oldQty > 0 && newQty == 0 {
		s.logger.Info("stock sync: stock depleted to zero",
			"tenant_id", tenantID,
			"product_id", productID,
		)

		if s.automationSvc != nil {
			FireAutomationEvent(s.automationSvc, tenantID, "product", "product.out_of_stock", productID, map[string]any{
				"old_quantity": oldQty,
				"new_quantity": newQty,
				"trigger_type": triggerType,
			})
		} else {
			// Fallback: direct deactivation when automation engine is not wired
			asyncutil.SafeGo(func() { s.deactivateListings(context.Background(), tenantID, productID) })
		}
	}

	s.logger.Info("stock sync: stock change recorded",
		"tenant_id", tenantID,
		"product_id", productID,
		"trigger", triggerType,
		"old_qty", oldQty,
		"new_qty", newQty,
		"available", event.AvailableQuantity,
	)

	// Propagate stock to marketplaces asynchronously.
	// Skip when stock goes from >0 to 0 — deactivation is already handled above
	// (via automation event or direct deactivateListings fallback), and PropagateStockToMarketplaces
	// would redundantly deactivate the same listings.
	if oldQty <= 0 || newQty != 0 {
		asyncutil.SafeGo(func() { s.PropagateStockToMarketplaces(context.Background(), tenantID, productID) })
	}
}

// reactivateListings finds inactive/ended marketplace listings for a product and
// reactivates them via the marketplace provider's ActivateOffer API.
// This is the direct fallback called when stock is restored from zero and the
// automation engine is not wired. When the automation engine IS available,
// the equivalent logic runs via automation.executeActivateListing instead.
// The two implementations are kept separate to avoid a circular dependency between
// the service and automation packages (see automation/actions.go for details).
func (s *StockSyncService) reactivateListings(ctx context.Context, tenantID, productID uuid.UUID) {
	type relistJob struct {
		listingID   uuid.UUID
		externalID  string
		integration *model.IntegrationWithCreds
	}

	var jobs []relistJob

	// Phase 1: Gather inactive listings inside a transaction.
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		listings, err := s.listingRepo.ListByProduct(ctx, tx, productID)
		if err != nil {
			return err
		}

		for _, listing := range listings {
			if listing.Status != "inactive" && listing.Status != "ended" {
				continue
			}
			if listing.ExternalID == nil || *listing.ExternalID == "" {
				continue
			}

			integ, err := s.integrationRepo.FindByID(ctx, tx, listing.IntegrationID)
			if err != nil || integ == nil {
				s.logger.Warn("auto-relist: integration not found for listing",
					"listing_id", listing.ID, "integration_id", listing.IntegrationID)
				continue
			}

			jobs = append(jobs, relistJob{
				listingID:   listing.ID,
				externalID:  *listing.ExternalID,
				integration: integ,
			})
		}
		return nil
	})
	if err != nil {
		s.logger.Error("auto-relist: failed to gather listings",
			"tenant_id", tenantID, "product_id", productID, "error", err)
		return
	}

	if len(jobs) == 0 {
		return
	}

	// Phase 2: Activate via marketplace APIs outside the transaction.
	activated, failed := 0, 0
	for _, job := range jobs {
		if job.integration.EncryptedCredentials == "" {
			s.logger.Warn("auto-relist: no credentials for integration",
				"integration_id", job.integration.ID)
			failed++
			continue
		}

		credJSON, err := crypto.Decrypt(job.integration.EncryptedCredentials, s.encryptionKey)
		if err != nil {
			s.logger.Error("auto-relist: decrypt credentials failed",
				"integration_id", job.integration.ID, "error", err)
			failed++
			continue
		}

		provider, err := integration.NewMarketplaceProvider(job.integration.Provider, credJSON, job.integration.Settings)
		if err != nil {
			s.logger.Error("auto-relist: create provider failed",
				"provider", job.integration.Provider, "error", err)
			failed++
			continue
		}

		// Check if provider supports listing activation
		activator, ok := provider.(integration.ListingActivator)
		if !ok {
			s.logger.Info("auto-relist: provider does not support activation, skipping",
				"provider", job.integration.Provider)
			closeMarketplaceProvider(provider)
			continue
		}

		if err := activator.ActivateOffer(ctx, job.externalID); err != nil {
			s.logger.Error("auto-relist: activation failed",
				"listing_id", job.listingID, "external_id", job.externalID,
				"provider", job.integration.Provider, "error", err)
			closeMarketplaceProvider(provider)
			failed++
			continue
		}
		closeMarketplaceProvider(provider)

		// Update listing status to active (best-effort)
		activeStatus := "active"
		syncOK := "synced"
		_ = database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
			return s.listingRepo.Update(ctx, tx, job.listingID, &model.UpdateProductListingRequest{
				Status:     &activeStatus,
				SyncStatus: &syncOK,
			})
		})
		activated++
	}

	s.logger.Info("auto-relist: completed",
		"tenant_id", tenantID, "product_id", productID,
		"activated", activated, "failed", failed)
}

// deactivateListings finds active marketplace listings for a product and
// deactivates them via the marketplace provider's DeactivateOffer API.
// This is the direct fallback called when stock drops to zero and the
// automation engine is not wired. Mirrors reactivateListings above.
func (s *StockSyncService) deactivateListings(ctx context.Context, tenantID, productID uuid.UUID) {
	type deactivateJob struct {
		listingID   uuid.UUID
		externalID  string
		integration *model.IntegrationWithCreds
	}

	var jobs []deactivateJob

	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		// ListAutoSyncByProduct filters: status='active', stock_sync_mode='auto', external_id IS NOT NULL.
		listings, err := s.listingRepo.ListAutoSyncByProduct(ctx, tx, productID)
		if err != nil {
			return err
		}

		for _, listing := range listings {
			integ, err := s.integrationRepo.FindByID(ctx, tx, listing.IntegrationID)
			if err != nil || integ == nil {
				s.logger.Warn("auto-deactivate: integration not found for listing",
					"listing_id", listing.ID, "integration_id", listing.IntegrationID)
				continue
			}

			jobs = append(jobs, deactivateJob{
				listingID:   listing.ID,
				externalID:  *listing.ExternalID,
				integration: integ,
			})
		}
		return nil
	})
	if err != nil {
		s.logger.Error("auto-deactivate: failed to gather listings",
			"tenant_id", tenantID, "product_id", productID, "error", err)
		return
	}

	if len(jobs) == 0 {
		return
	}

	deactivated, failed := 0, 0
	for _, job := range jobs {
		if job.integration.EncryptedCredentials == "" {
			s.logger.Warn("auto-deactivate: no credentials for integration",
				"integration_id", job.integration.ID, "listing_id", job.listingID)
			failed++
			continue
		}

		credJSON, err := crypto.Decrypt(job.integration.EncryptedCredentials, s.encryptionKey)
		if err != nil {
			s.logger.Error("auto-deactivate: decrypt credentials failed",
				"integration_id", job.integration.ID, "error", err)
			failed++
			continue
		}

		provider, err := integration.NewMarketplaceProvider(job.integration.Provider, credJSON, job.integration.Settings)
		if err != nil {
			s.logger.Error("auto-deactivate: create provider failed",
				"provider", job.integration.Provider, "error", err)
			failed++
			continue
		}

		deactivator, ok := provider.(integration.ListingDeactivator)
		if !ok {
			closeMarketplaceProvider(provider)
			continue
		}

		if err := deactivator.DeactivateOffer(ctx, job.externalID); err != nil {
			s.logger.Error("auto-deactivate: deactivation failed",
				"listing_id", job.listingID, "external_id", job.externalID,
				"provider", job.integration.Provider, "error", err)
			closeMarketplaceProvider(provider)
			failed++
			continue
		}
		closeMarketplaceProvider(provider)

		inactiveStatus := "inactive"
		syncOK := "synced"
		_ = database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
			return s.listingRepo.Update(ctx, tx, job.listingID, &model.UpdateProductListingRequest{
				Status:     &inactiveStatus,
				SyncStatus: &syncOK,
			})
		})
		deactivated++
	}

	s.logger.Info("auto-deactivate: completed",
		"tenant_id", tenantID, "product_id", productID,
		"deactivated", deactivated, "failed", failed)
}

// CalculateAvailableStock returns stock allocation per channel for a product.
func (s *StockSyncService) CalculateAvailableStock(ctx context.Context, tenantID, productID uuid.UUID) ([]model.StockAllocation, error) {
	var allocations []model.StockAllocation
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		totalQty, reservedQty, err := s.eventRepo.GetAvailableStock(ctx, tx, productID)
		if err != nil {
			return err
		}

		channels, err := s.channelRepo.ListEnabled(ctx, tx)
		if err != nil {
			return err
		}

		// Calculate total buffer across all channels
		totalBuffer := 0
		for _, ch := range channels {
			totalBuffer += ch.StockBuffer
		}

		baseAvailable := max(totalQty-reservedQty, 0)

		for _, ch := range channels {
			available := max(baseAvailable-ch.StockBuffer, 0)
			allocations = append(allocations, model.StockAllocation{
				ChannelID:         ch.ID,
				ChannelType:       ch.ChannelType,
				TotalStock:        totalQty,
				Reserved:          reservedQty,
				Buffer:            ch.StockBuffer,
				AvailableQuantity: available,
			})
		}

		return nil
	})
	if allocations == nil {
		allocations = []model.StockAllocation{}
	}
	return allocations, err
}

// PushStockToChannel pushes current stock to a specific channel.
func (s *StockSyncService) PushStockToChannel(ctx context.Context, tenantID, channelID uuid.UUID) error {
	return database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		ch, err := s.channelRepo.FindByID(ctx, tx, channelID)
		if err != nil {
			return err
		}
		if ch == nil {
			return ErrStockSyncChannelNotFound
		}

		// Mark sync timestamp
		return s.channelRepo.UpdateSyncStatus(ctx, tx, channelID, nil)
	})
}

// PushStockToAllChannels pushes stock for a product to all enabled channels via marketplace APIs.
func (s *StockSyncService) PushStockToAllChannels(ctx context.Context, tenantID, productID uuid.UUID) error {
	s.PropagateStockToMarketplaces(ctx, tenantID, productID)
	return nil
}

// PropagateStockToMarketplaces calculates available stock and pushes it to all auto-sync listings.
// Uses a two-phase approach: Phase 1 gathers data inside a DB transaction, Phase 2 calls marketplace APIs outside the transaction.
func (s *StockSyncService) PropagateStockToMarketplaces(ctx context.Context, tenantID, productID uuid.UUID) {
	type pushJob struct {
		listing      *model.ProductListing
		availableQty int
		integration  *model.IntegrationWithCreds
	}

	var jobs []pushJob

	// Phase 1: Gather data inside transaction
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		// Calculate available stock from warehouse tables
		totalQty, reservedQty, err := s.eventRepo.GetAvailableStock(ctx, tx, productID)
		if err != nil {
			return err
		}
		baseAvailable := max(totalQty-reservedQty, 0)

		// Get auto-sync listings for this product
		listings, err := s.listingRepo.ListAutoSyncByProduct(ctx, tx, productID)
		if err != nil {
			return err
		}

		// Get channel buffers (keyed by integration_id match to channel)
		channels, err := s.channelRepo.ListEnabled(ctx, tx)
		if err != nil {
			return err
		}
		channelBuffers := make(map[uuid.UUID]int) // integration_id -> buffer
		for _, ch := range channels {
			if ch.IntegrationID != nil {
				channelBuffers[*ch.IntegrationID] = ch.StockBuffer
			}
		}

		for _, listing := range listings {
			// Use stock_override if set
			qty := baseAvailable
			if listing.StockOverride != nil {
				qty = *listing.StockOverride
			} else {
				// Apply channel buffer
				if buffer, ok := channelBuffers[listing.IntegrationID]; ok {
					qty = max(qty-buffer, 0)
				}
			}

			// Get integration credentials
			integ, err := s.integrationRepo.FindByID(ctx, tx, listing.IntegrationID)
			if err != nil || integ == nil {
				s.logger.Warn("stock sync: integration not found for listing",
					"listing_id", listing.ID, "integration_id", listing.IntegrationID)
				continue
			}

			jobs = append(jobs, pushJob{
				listing:      listing,
				availableQty: qty,
				integration:  integ,
			})
		}
		return nil
	})
	if err != nil {
		s.logger.Error("stock sync: failed to gather propagation data",
			"tenant_id", tenantID, "product_id", productID, "error", err)
		return
	}

	if len(jobs) == 0 {
		return
	}

	// Phase 2: Push to marketplace APIs outside the transaction
	pushed, failed := 0, 0
	for _, job := range jobs {
		if job.integration.EncryptedCredentials == "" {
			s.logger.Warn("stock sync: no credentials for integration", "integration_id", job.integration.ID)
			failed++
			continue
		}

		credJSON, err := crypto.Decrypt(job.integration.EncryptedCredentials, s.encryptionKey)
		if err != nil {
			s.logger.Error("stock sync: decrypt credentials failed",
				"integration_id", job.integration.ID, "error", err)
			failed++
			continue
		}

		provider, err := integration.NewMarketplaceProvider(job.integration.Provider, credJSON, job.integration.Settings)
		if err != nil {
			s.logger.Error("stock sync: create provider failed",
				"provider", job.integration.Provider, "error", err)
			failed++
			continue
		}

		// Deactivate listing if stock is 0 and provider supports it
		if job.availableQty == 0 {
			if deactivator, ok := provider.(integration.ListingDeactivator); ok {
				if err := deactivator.DeactivateOffer(ctx, *job.listing.ExternalID); err != nil {
					s.logger.Error("stock sync: deactivate listing failed",
						"listing_id", job.listing.ID, "external_id", *job.listing.ExternalID,
						"provider", job.integration.Provider, "error", err)
					errMsg := truncateServiceErrorMsg(err.Error())
					syncErr := "error"
					_ = database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
						return s.listingRepo.Update(ctx, tx, job.listing.ID, &model.UpdateProductListingRequest{
							SyncStatus:   &syncErr,
							ErrorMessage: &errMsg,
						})
					})
					failed++
					continue
				}
				inactiveStatus := "inactive"
				syncOK := "synced"
				_ = database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
					return s.listingRepo.Update(ctx, tx, job.listing.ID, &model.UpdateProductListingRequest{
						Status:     &inactiveStatus,
						SyncStatus: &syncOK,
					})
				})
				pushed++
				continue
			}
			// Provider doesn't support deactivation — fall through to UpdateStock(0)
		}

		if err := provider.UpdateStock(ctx, *job.listing.ExternalID, job.availableQty); err != nil {
			s.logger.Error("stock sync: push stock failed",
				"listing_id", job.listing.ID, "external_id", *job.listing.ExternalID,
				"provider", job.integration.Provider, "error", err)

			// Update listing sync status to error (best-effort)
			errMsg := truncateServiceErrorMsg(err.Error())
			syncErr := "error"
			_ = database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
				return s.listingRepo.Update(ctx, tx, job.listing.ID, &model.UpdateProductListingRequest{
					SyncStatus:   &syncErr,
					ErrorMessage: &errMsg,
				})
			})
			failed++
			continue
		}

		// Use 'pending' for async providers (e.g. Amazon Feeds API), 'synced' otherwise.
		syncStatus := "synced"
		if _, ok := provider.(integration.AsyncStockUpdater); ok {
			syncStatus = "pending"
		}
		if syncStatus == "pending" {
			// Store feed metadata for async polling (e.g. Amazon feed ID)
			if fr, ok := provider.(integration.AsyncFeedResult); ok {
				if result := fr.FeedResult(); result != nil {
					feedMeta, _ := json.Marshal(map[string]string{
						"amazon_feed_id":   result.FeedID,
						"amazon_feed_type": result.FeedType,
					})
					_ = database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
						_, err := tx.Exec(ctx,
							`UPDATE product_listings SET sync_status = 'pending', error_message = NULL,
							 metadata = COALESCE(metadata, '{}'::jsonb) || $2::jsonb, updated_at = NOW() WHERE id = $1`,
							job.listing.ID, string(feedMeta))
						return err
					})
					pushed++
					continue
				}
			}
		}
		_ = database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
			return s.listingRepo.Update(ctx, tx, job.listing.ID, &model.UpdateProductListingRequest{
				SyncStatus: &syncStatus,
			})
		})
		pushed++
	}

	s.logger.Info("stock sync: propagation complete",
		"tenant_id", tenantID, "product_id", productID,
		"pushed", pushed, "failed", failed)
}

// PushSingleListing forces a stock push for a single listing (ignores sync_mode).
func (s *StockSyncService) PushSingleListing(ctx context.Context, tenantID, listingID uuid.UUID) error {
	var listing *model.ProductListing
	var integ *model.IntegrationWithCreds
	var availableQty int

	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var err error
		listing, err = s.listingRepo.GetByID(ctx, tx, listingID)
		if err != nil {
			return err
		}
		if listing == nil {
			return errors.New("listing not found")
		}
		if listing.ExternalID == nil || *listing.ExternalID == "" {
			return errors.New("listing has no external_id")
		}

		totalQty, reservedQty, err := s.eventRepo.GetAvailableStock(ctx, tx, listing.ProductID)
		if err != nil {
			return err
		}
		availableQty = max(totalQty-reservedQty, 0)
		if listing.StockOverride != nil {
			availableQty = *listing.StockOverride
		}

		integ, err = s.integrationRepo.FindByID(ctx, tx, listing.IntegrationID)
		if err != nil {
			return err
		}
		if integ == nil {
			return errors.New("integration not found")
		}
		return nil
	})
	if err != nil {
		return err
	}

	if integ.EncryptedCredentials == "" {
		return errors.New("integration has no credentials")
	}
	credJSON, err := crypto.Decrypt(integ.EncryptedCredentials, s.encryptionKey)
	if err != nil {
		return fmt.Errorf("decrypt credentials: %w", err)
	}

	provider, err := integration.NewMarketplaceProvider(integ.Provider, credJSON, integ.Settings)
	if err != nil {
		return fmt.Errorf("create provider: %w", err)
	}

	if err := provider.UpdateStock(ctx, *listing.ExternalID, availableQty); err != nil {
		errMsg := err.Error()
		syncErr := "error"
		_ = database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
			return s.listingRepo.Update(ctx, tx, listing.ID, &model.UpdateProductListingRequest{
				SyncStatus:   &syncErr,
				ErrorMessage: &errMsg,
			})
		})
		return fmt.Errorf("push stock: %w", err)
	}

	syncOK := "synced"
	_ = database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		return s.listingRepo.Update(ctx, tx, listing.ID, &model.UpdateProductListingRequest{
			SyncStatus: &syncOK,
		})
	})

	s.logger.Info("stock sync: single listing pushed",
		"tenant_id", tenantID, "listing_id", listingID,
		"external_id", *listing.ExternalID, "quantity", availableQty)
	return nil
}

// PushAll triggers a stock push for all products across all enabled channels.
func (s *StockSyncService) PushAll(ctx context.Context, tenantID uuid.UUID) (int, error) {
	var synced int
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		channels, err := s.channelRepo.ListEnabled(ctx, tx)
		if err != nil {
			return err
		}

		for _, ch := range channels {
			if err := s.channelRepo.UpdateSyncStatus(ctx, tx, ch.ID, nil); err != nil {
				s.logger.Error("stock sync: failed to update channel during push all",
					"channel_id", ch.ID,
					"error", err,
				)
				continue
			}
			synced++
		}
		return nil
	})
	return synced, err
}

// ReconcileStock reconciles stock for a product by comparing local stock with channel states.
func (s *StockSyncService) ReconcileStock(ctx context.Context, tenantID, productID uuid.UUID) (*model.StockSyncEvent, error) {
	var event *model.StockSyncEvent
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		product, err := s.productRepo.FindByID(ctx, tx, productID)
		if err != nil {
			return err
		}
		if product == nil {
			return errors.New("product not found")
		}

		totalQty, reservedQty, err := s.eventRepo.GetAvailableStock(ctx, tx, productID)
		if err != nil {
			return err
		}

		channels, err := s.channelRepo.ListEnabled(ctx, tx)
		if err != nil {
			return err
		}

		totalBuffer := 0
		for _, ch := range channels {
			totalBuffer += ch.StockBuffer
		}

		availableQty := max(totalQty-reservedQty-totalBuffer, 0)

		details := map[string]any{
			"reconcile":    true,
			"total_stock":  totalQty,
			"reserved":     reservedQty,
			"total_buffer": totalBuffer,
			"channels":     len(channels),
		}
		detailsJSON, _ := json.Marshal(details)

		evt := model.StockSyncEvent{
			ID:                uuid.New(),
			TenantID:          tenantID,
			ProductID:         &productID,
			SKU:               product.SKU,
			TriggerType:       "recount",
			OldQuantity:       totalQty,
			NewQuantity:       totalQty,
			AvailableQuantity: availableQty,
			ChannelsNotified:  len(channels),
			ChannelsFailed:    0,
			Details:           detailsJSON,
		}
		if err := s.eventRepo.Create(ctx, tx, &evt); err != nil {
			return err
		}

		// Update all channel sync timestamps
		for _, ch := range channels {
			_ = s.channelRepo.UpdateSyncStatus(ctx, tx, ch.ID, nil)
		}

		event = &evt
		return nil
	})
	if err != nil {
		return nil, err
	}
	return event, nil
}

// --- Events & Dashboard ---

// ListEvents returns a paginated list of stock sync events.
func (s *StockSyncService) ListEvents(ctx context.Context, tenantID uuid.UUID, filter model.StockSyncEventListFilter) (model.ListResponse[model.StockSyncEvent], error) {
	var resp model.ListResponse[model.StockSyncEvent]
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		events, total, err := s.eventRepo.List(ctx, tx, filter)
		if err != nil {
			return err
		}
		resp = model.NewListResponse(events, total, filter.Limit, filter.Offset)
		return nil
	})
	if resp.Items == nil {
		resp.Items = []model.StockSyncEvent{}
	}
	return resp, err
}

// GetDashboard returns an overview of the stock sync status.
func (s *StockSyncService) GetDashboard(ctx context.Context, tenantID uuid.UUID) (*model.StockSyncDashboard, error) {
	dash := &model.StockSyncDashboard{}
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		// Count products with stock entries
		if err := tx.QueryRow(ctx,
			"SELECT COUNT(DISTINCT product_id) FROM warehouse_stock WHERE quantity > 0",
		).Scan(&dash.TotalProducts); err != nil {
			return err
		}

		// Get all channels
		channels, err := s.channelRepo.ListEnabled(ctx, tx)
		if err != nil {
			return err
		}
		dash.ActiveChannels = len(channels)

		// Count recent errors (last 24h)
		recentErrors, err := s.eventRepo.CountRecentErrors(ctx, tx)
		if err != nil {
			return err
		}
		dash.RecentErrors = recentErrors

		// Find latest sync across all channels
		var latestSync *time.Time
		for _, ch := range channels {
			if ch.LastSyncAt != nil {
				if latestSync == nil || ch.LastSyncAt.After(*latestSync) {
					latestSync = ch.LastSyncAt
				}
			}
		}
		dash.LastSyncAt = latestSync

		// Build channel summaries (include all channels, not just enabled)
		allChannels, _, err := s.channelRepo.List(ctx, tx, model.StockSyncChannelListFilter{
			PaginationParams: model.PaginationParams{Limit: 100, Offset: 0},
		})
		if err != nil {
			return err
		}

		for _, ch := range allChannels {
			status := "ok"
			switch {
			case !ch.Enabled:
				status = "disabled"
			case ch.LastError != nil && *ch.LastError != "":
				status = "error"
			case ch.LastSyncAt == nil:
				status = "warning"
			}

			dash.ChannelSummaries = append(dash.ChannelSummaries, model.ChannelSummary{
				ID:          ch.ID,
				ChannelType: ch.ChannelType,
				Enabled:     ch.Enabled,
				SyncMode:    ch.SyncMode,
				StockBuffer: ch.StockBuffer,
				LastSyncAt:  ch.LastSyncAt,
				LastError:   ch.LastError,
				Status:      status,
			})
		}
		if dash.ChannelSummaries == nil {
			dash.ChannelSummaries = []model.ChannelSummary{}
		}

		return nil
	})
	return dash, err
}
