package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"math"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/crypto"
	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

var (
	// ErrListingSyncConfigNotFound is returned when a listing sync configuration does not exist.
	ErrListingSyncConfigNotFound = errors.New("listing sync config not found")
)

// ListingSyncService handles CRUD and sync orchestration for listing sync configs.
type ListingSyncService struct {
	syncRepo        *repository.ListingSyncRepository
	productRepo     repository.ProductRepo
	listingRepo     *repository.ProductListingRepository
	auditRepo       repository.AuditRepo
	integrationRepo repository.IntegrationRepo
	pool            *pgxpool.Pool
	encryptionKey   []byte
	stockSyncSvc    *StockSyncService
	logger          *slog.Logger
}

// NewListingSyncService creates a new ListingSyncService.
func NewListingSyncService(
	syncRepo *repository.ListingSyncRepository,
	productRepo repository.ProductRepo,
	listingRepo *repository.ProductListingRepository,
	auditRepo repository.AuditRepo,
	integrationRepo repository.IntegrationRepo,
	pool *pgxpool.Pool,
	encryptionKey []byte,
	logger *slog.Logger,
) *ListingSyncService {
	return &ListingSyncService{
		syncRepo:        syncRepo,
		productRepo:     productRepo,
		listingRepo:     listingRepo,
		auditRepo:       auditRepo,
		integrationRepo: integrationRepo,
		pool:            pool,
		encryptionKey:   encryptionKey,
		logger:          logger,
	}
}

// SetStockSyncService injects the canonical stock owner. SyncStock delegates stock
// propagation to it (reading warehouse_stock) rather than pushing a second, stale
// stock value derived from products.stock_quantity.
func (s *ListingSyncService) SetStockSyncService(stockSyncSvc *StockSyncService) {
	s.stockSyncSvc = stockSyncSvc
}

// List returns paginated listing sync configs for a tenant.
func (s *ListingSyncService) List(ctx context.Context, tenantID uuid.UUID, filter model.ListingSyncConfigFilter) (model.ListResponse[model.ListingSyncConfig], error) {
	var resp model.ListResponse[model.ListingSyncConfig]
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		configs, total, listErr := s.syncRepo.ListConfigs(ctx, tx, filter)
		if listErr != nil {
			return listErr
		}
		resp = model.NewListResponse(configs, total, filter.Limit, filter.Offset)
		return nil
	})
	return resp, err
}

// Get returns a single listing sync config by ID.
func (s *ListingSyncService) Get(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*model.ListingSyncConfig, error) {
	var cfg *model.ListingSyncConfig
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var getErr error
		cfg, getErr = s.syncRepo.GetConfigByID(ctx, tx, id)
		return getErr
	})
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		return nil, ErrListingSyncConfigNotFound
	}
	return cfg, nil
}

// Create creates a new listing sync config.
func (s *ListingSyncService) Create(ctx context.Context, tenantID uuid.UUID, req model.CreateListingSyncConfigRequest) (*model.ListingSyncConfig, error) {
	cfg := &model.ListingSyncConfig{
		ID:                  uuid.New(),
		TenantID:            tenantID,
		IntegrationID:       req.IntegrationID,
		SyncDirection:       req.SyncDirection,
		AutoSync:            false,
		SyncIntervalMinutes: 30,
		FieldMapping:        json.RawMessage("{}"),
		PriceRule:           req.PriceRule,
		PriceModifier:       0,
		StockBuffer:         0,
		Status:              "active",
	}

	if req.AutoSync != nil {
		cfg.AutoSync = *req.AutoSync
	}
	if req.SyncIntervalMinutes != nil {
		cfg.SyncIntervalMinutes = *req.SyncIntervalMinutes
	}
	if len(req.FieldMapping) > 0 {
		cfg.FieldMapping = req.FieldMapping
	}
	if req.PriceModifier != nil {
		cfg.PriceModifier = *req.PriceModifier
	}
	if req.StockBuffer != nil {
		cfg.StockBuffer = *req.StockBuffer
	}

	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		if createErr := s.syncRepo.CreateConfig(ctx, tx, cfg); createErr != nil {
			return createErr
		}

		// Audit log
		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			EntityType: "listing_sync_config",
			EntityID:   cfg.ID,
			Action:     "created",
			Changes: map[string]any{
				"integration_id": req.IntegrationID,
				"sync_direction": cfg.SyncDirection,
				"auto_sync":      cfg.AutoSync,
				"price_rule":     cfg.PriceRule,
			},
		})
	})
	if err != nil {
		return nil, err
	}

	// Re-fetch to get joined integration data
	return s.Get(ctx, tenantID, cfg.ID)
}

// Update updates a listing sync config.
func (s *ListingSyncService) Update(ctx context.Context, tenantID uuid.UUID, id uuid.UUID, req model.UpdateListingSyncConfigRequest) (*model.ListingSyncConfig, error) {
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		existing, getErr := s.syncRepo.GetConfigByID(ctx, tx, id)
		if getErr != nil {
			return getErr
		}
		if existing == nil {
			return ErrListingSyncConfigNotFound
		}

		if updateErr := s.syncRepo.UpdateConfig(ctx, tx, id, &req); updateErr != nil {
			return updateErr
		}

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			EntityType: "listing_sync_config",
			EntityID:   id,
			Action:     "updated",
			Changes:    req,
		})
	})
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, tenantID, id)
}

// Delete deletes a listing sync config.
func (s *ListingSyncService) Delete(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) error {
	return database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		existing, getErr := s.syncRepo.GetConfigByID(ctx, tx, id)
		if getErr != nil {
			return getErr
		}
		if existing == nil {
			return ErrListingSyncConfigNotFound
		}

		if deleteErr := s.syncRepo.DeleteConfig(ctx, tx, id); deleteErr != nil {
			return deleteErr
		}

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			EntityType: "listing_sync_config",
			EntityID:   id,
			Action:     "deleted",
			Changes:    map[string]any{"integration_id": existing.IntegrationID},
		})
	})
}

// ListLogs returns paginated sync logs for a config.
func (s *ListingSyncService) ListLogs(ctx context.Context, tenantID uuid.UUID, filter model.ListingSyncLogFilter) (model.ListResponse[model.ListingSyncLog], error) {
	var resp model.ListResponse[model.ListingSyncLog]
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		logs, total, listErr := s.syncRepo.ListLogs(ctx, tx, filter)
		if listErr != nil {
			return listErr
		}
		resp = model.NewListResponse(logs, total, filter.Limit, filter.Offset)
		return nil
	})
	return resp, err
}

// --- Sync Operations ---

// SyncResult is returned by sync operations.
type SyncResult struct {
	ItemsProcessed int    `json:"items_processed"`
	ItemsFailed    int    `json:"items_failed"`
	Message        string `json:"message"`
}

// listingPushJob couples a newly-created local listing with the product to publish.
// It is gathered inside a transaction and pushed outside it.
type listingPushJob struct {
	listing *model.ProductListing
	product model.Product
}

// SyncProducts creates marketplace offers for products without a listing. It follows the
// same three-phase pattern as SyncPrices: gather and create pending local rows in a short
// transaction, call the marketplace without a transaction, then persist each outcome.
func (s *ListingSyncService) SyncProducts(ctx context.Context, tenantID uuid.UUID, configID uuid.UUID) (*SyncResult, error) {
	cfg, err := s.Get(ctx, tenantID, configID)
	if err != nil {
		return nil, err
	}

	result := &SyncResult{}
	var jobs []listingPushJob
	var integ *model.IntegrationWithCreds

	// Phase 1: create local pending rows and gather the provider credentials.
	err = database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		products, _, listErr := s.productRepo.List(ctx, tx, model.ProductListFilter{
			PaginationParams: model.PaginationParams{Limit: 1000, Offset: 0},
		})
		if listErr != nil {
			return fmt.Errorf("list products: %w", listErr)
		}

		for _, product := range products {
			existing, findErr := s.listingRepo.FindByProductAndIntegration(ctx, tx, product.ID, cfg.IntegrationID)
			if findErr != nil {
				s.logSyncEntry(ctx, tx, tenantID, configID, "push", "product", &product.ID, nil, "failed", findErr.Error())
				result.ItemsFailed++
				continue
			}
			if existing != nil {
				// PushOffer creates an offer, so reusing it for an existing listing could duplicate it.
				s.logSyncEntry(ctx, tx, tenantID, configID, "push", "product", &product.ID, existing.ExternalID, "skipped", "listing already exists")
				continue
			}

			listing := &model.ProductListing{
				ID:            uuid.New(),
				TenantID:      tenantID,
				ProductID:     product.ID,
				IntegrationID: cfg.IntegrationID,
				Status:        "pending",
				SyncStatus:    "pending",
				Metadata:      json.RawMessage("{}"),
			}
			adjustedPrice := s.applyPriceRule(cfg, product.Price)
			if adjustedPrice != product.Price {
				listing.PriceOverride = &adjustedPrice
			}
			if createErr := s.listingRepo.Create(ctx, tx, listing); createErr != nil {
				s.logSyncEntry(ctx, tx, tenantID, configID, "push", "product", &product.ID, nil, "failed", createErr.Error())
				result.ItemsFailed++
				continue
			}
			jobs = append(jobs, listingPushJob{listing: listing, product: product})
		}

		if len(jobs) == 0 {
			return nil
		}
		var findErr error
		integ, findErr = s.integrationRepo.FindByID(ctx, tx, cfg.IntegrationID)
		if findErr != nil {
			return fmt.Errorf("find integration: %w", findErr)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Phase 2: PushOffer runs without a database transaction. Phase 3 persists each outcome.
	if len(jobs) > 0 {
		switch {
		case integ == nil || integ.EncryptedCredentials == "":
			for _, job := range jobs {
				s.persistProductPushError(ctx, tenantID, configID, job.listing, "integration has no credentials")
				result.ItemsFailed++
			}
		default:
			provider, providerErr := s.marketplaceProvider(integ)
			if providerErr != nil {
				errMsg := truncateServiceErrorMsg(providerErr.Error())
				for _, job := range jobs {
					s.persistProductPushError(ctx, tenantID, configID, job.listing, errMsg)
					result.ItemsFailed++
				}
			} else {
				s.dispatchProductOffers(ctx, integ, provider, cfg, jobs, result,
					func(listing *model.ProductListing, externalID string) {
						s.persistProductPush(ctx, tenantID, configID, listing, externalID)
					},
					func(listing *model.ProductListing, errMsg string) {
						s.persistProductPushError(ctx, tenantID, configID, listing, errMsg)
					},
				)
				closeMarketplaceProvider(provider)
			}
		}
	}

	s.updateLastSync(ctx, tenantID, configID, result.ItemsFailed > 0)
	result.Message = fmt.Sprintf("Synced %d products, %d errors", result.ItemsProcessed, result.ItemsFailed)
	return result, nil
}

// dispatchProductOffers calls PushOffer for each new listing outside a transaction and
// reports each provider outcome to the supplied persistence callbacks. A failed offer
// does not prevent later jobs from being pushed.
func (s *ListingSyncService) dispatchProductOffers(
	ctx context.Context,
	integ *model.IntegrationWithCreds,
	provider integration.MarketplaceProvider,
	cfg *model.ListingSyncConfig,
	jobs []listingPushJob,
	result *SyncResult,
	onSuccess func(*model.ProductListing, string),
	onFailure func(*model.ProductListing, string),
) {
	for _, job := range jobs {
		externalID, pushErr := provider.PushOffer(ctx, &job.product, s.listingData(cfg, job.listing, job.product))
		if pushErr != nil || externalID == "" {
			errMsg := "provider returned an empty external id"
			if pushErr != nil {
				errMsg = truncateServiceErrorMsg(pushErr.Error())
			}
			s.logger.Error("listing sync: push offer failed", "listing_id", job.listing.ID, "provider", integ.Provider, "error", errMsg)
			onFailure(job.listing, errMsg)
			result.ItemsFailed++
			continue
		}
		onSuccess(job.listing, externalID)
		result.ItemsProcessed++
	}
}

// listingData returns the common product fields expected by marketplace providers, overlaid
// with the config's provider-specific field mapping. It deliberately excludes stock: the
// canonical stock owner is StockSyncService, which reads warehouse_stock.
func (s *ListingSyncService) listingData(cfg *model.ListingSyncConfig, listing *model.ProductListing, product model.Product) map[string]any {
	data := map[string]any{
		"title":          product.Name,
		"name":           product.Name,
		"price":          product.Price,
		"price_override": listing.PriceOverride,
		"sku":            product.SKU,
		"ean":            product.EAN,
		"description":    product.DescriptionLong,
	}
	var mapped map[string]any
	if json.Unmarshal(cfg.FieldMapping, &mapped) == nil {
		maps.Copy(data, mapped)
	}
	return data
}

// persistProductPush records a successful PushOffer in its own short transaction.
func (s *ListingSyncService) persistProductPush(ctx context.Context, tenantID, configID uuid.UUID, listing *model.ProductListing, externalID string) {
	if err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		if _, execErr := tx.Exec(ctx, `UPDATE product_listings
			SET external_id = $2, status = 'active', stock_sync_mode = 'auto', sync_status = 'synced',
				error_message = NULL, last_synced_at = NOW(), updated_at = NOW()
			WHERE id = $1`, listing.ID, externalID); execErr != nil {
			return execErr
		}
		s.logSyncEntry(ctx, tx, tenantID, configID, "push", "product", &listing.ProductID, &externalID, "success", "")
		return nil
	}); err != nil {
		s.logger.Error("listing sync: persist product push failed", "listing_id", listing.ID, "error", err)
	}
}

// persistProductPushError marks one offer creation as failed and records its sync log.
func (s *ListingSyncService) persistProductPushError(ctx context.Context, tenantID, configID uuid.UUID, listing *model.ProductListing, errMsg string) {
	if err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		if _, execErr := tx.Exec(ctx, `UPDATE product_listings
			SET sync_status = 'failed', error_message = $2, updated_at = NOW() WHERE id = $1`, listing.ID, errMsg); execErr != nil {
			return execErr
		}
		s.logSyncEntry(ctx, tx, tenantID, configID, "push", "product", &listing.ProductID, nil, "failed", errMsg)
		return nil
	}); err != nil {
		s.logger.Error("listing sync: persist product push error failed", "listing_id", listing.ID, "error", err)
	}
}

const listingPullUnsupportedReason = "provider does not expose offer listing"

// listingPullSkip is one honest skip for a listing we cannot fetch from the provider.
type listingPullSkip struct {
	ProductID  *uuid.UUID
	ExternalID *string
	LogStatus  string
	Reason     string
	MarkSynced bool
}

func planUnsupportedListingPull(listings []*model.ProductListing) []listingPullSkip {
	actions := make([]listingPullSkip, 0, len(listings))
	for _, listing := range listings {
		var productID *uuid.UUID
		if listing.ProductID != uuid.Nil {
			id := listing.ProductID
			productID = &id
		}
		actions = append(actions, listingPullSkip{
			ProductID:  productID,
			ExternalID: listing.ExternalID,
			LogStatus:  "skipped",
			Reason:     listingPullUnsupportedReason,
			MarkSynced: false,
		})
	}
	return actions
}

func syncResultForUnsupportedPull(skipped int) *SyncResult {
	return &SyncResult{
		Message: fmt.Sprintf("Offer pull is not supported by the provider; %d listings skipped, 0 fetched", skipped),
	}
}

func lastErrorForUnsupportedPull() *string {
	msg := listingPullUnsupportedReason
	return &msg
}

func recordCleanLastSyncForUnsupportedPull() bool {
	return false
}

// PullListings does not fetch marketplace offers: MarketplaceProvider has no
// list-offers API. It records an honest skip per existing listing and does
// not write a successful pull status for work that did not happen.
func (s *ListingSyncService) PullListings(ctx context.Context, tenantID uuid.UUID, configID uuid.UUID) (*SyncResult, error) {
	cfg, err := s.Get(ctx, tenantID, configID)
	if err != nil {
		return nil, err
	}

	var skipped int
	err = database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		listings, listErr := s.listingRepo.ListByIntegration(ctx, tx, cfg.IntegrationID)
		if listErr != nil {
			return fmt.Errorf("list listings: %w", listErr)
		}

		for _, action := range planUnsupportedListingPull(listings) {
			s.logSyncEntry(ctx, tx, tenantID, configID, "pull", "offer", action.ProductID, action.ExternalID, action.LogStatus, action.Reason)
			skipped++
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	if err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		return s.syncRepo.UpdateLastSync(ctx, tx, configID, lastErrorForUnsupportedPull())
	}); err != nil {
		slog.Error("failed to update last sync timestamp", "error", err, "tenant_id", tenantID, "config_id", configID)
	}
	return syncResultForUnsupportedPull(skipped), nil
}

// listingPriceJob couples a push-eligible listing with the markup-applied price
// to push for it. Gathered inside the DB transaction, consumed outside it.
type listingPriceJob struct {
	listing       *model.ProductListing
	adjustedPrice float64
}

// SyncPrices pushes price updates from local products to the marketplace via the
// provider's UpdatePrice API. It follows the two-phase pattern mandated after
// OPE-479 (no marketplace HTTP call held inside a DB transaction): Phase 1 gathers
// the eligible listings, their markup-applied prices and the integration
// credentials inside a short transaction; Phase 2 constructs the provider once and
// pushes with no open transaction; Phase 3 persists each listing's sync status in
// its own short transaction. Only active, auto-sync listings with an external_id
// are pushed — the same gate the periodic PriceSyncWorker uses — so manual or
// inactive offers are never touched. The markup-applied price (applyPriceRule) is
// written to price_override, which is exactly the per-config value the periodic
// PriceSyncWorker also pushes, so the two pushers stay consistent.
func (s *ListingSyncService) SyncPrices(ctx context.Context, tenantID uuid.UUID, configID uuid.UUID) (*SyncResult, error) {
	cfg, err := s.Get(ctx, tenantID, configID)
	if err != nil {
		return nil, err
	}

	result := &SyncResult{}
	var jobs []listingPriceJob
	var integ *model.IntegrationWithCreds

	// Phase 1: gather eligible listings, adjusted prices and integration credentials.
	err = database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		listings, listErr := s.listingRepo.ListByIntegration(ctx, tx, cfg.IntegrationID)
		if listErr != nil {
			return fmt.Errorf("list listings: %w", listErr)
		}

		productMap, mapErr := s.productMapForListings(ctx, tx, listings)
		if mapErr != nil {
			return mapErr
		}

		for _, listing := range listings {
			if !isPushEligible(listing) {
				continue
			}
			product := productMap[listing.ProductID]
			if product == nil {
				s.logSyncEntry(ctx, tx, tenantID, configID, "push", "price", &listing.ProductID, listing.ExternalID, "failed", "product not found")
				result.ItemsFailed++
				continue
			}

			adjustedPrice := s.applyPriceRule(cfg, product.Price)
			// Idempotency / no-duplicate-feed guard: an offer already at the target
			// price is not re-pushed (each Amazon feed submission is a new feed).
			if listing.PriceOverride != nil && *listing.PriceOverride == adjustedPrice {
				result.ItemsProcessed++
				continue
			}
			jobs = append(jobs, listingPriceJob{listing: listing, adjustedPrice: adjustedPrice})
		}

		if len(jobs) == 0 {
			return nil
		}

		// One config maps to one integration; load its credentials for the push phase.
		var findErr error
		integ, findErr = s.integrationRepo.FindByID(ctx, tx, cfg.IntegrationID)
		if findErr != nil {
			return fmt.Errorf("find integration: %w", findErr)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Phase 2 + 3: push to the marketplace API outside any transaction, then persist.
	if len(jobs) > 0 {
		switch {
		case integ == nil || integ.EncryptedCredentials == "":
			for _, job := range jobs {
				s.persistPriceError(ctx, tenantID, configID, job.listing, "integration has no credentials")
				result.ItemsFailed++
			}
		default:
			provider, provErr := s.marketplaceProvider(integ)
			if provErr != nil {
				errMsg := truncateServiceErrorMsg(provErr.Error())
				for _, job := range jobs {
					s.persistPriceError(ctx, tenantID, configID, job.listing, errMsg)
					result.ItemsFailed++
				}
			} else {
				s.dispatchPriceUpdates(ctx, tenantID, configID, integ, provider, jobs, result)
				closeMarketplaceProvider(provider)
			}
		}
	}

	s.updateLastSync(ctx, tenantID, configID, result.ItemsFailed > 0)
	result.Message = fmt.Sprintf("Synced prices for %d offers, %d errors", result.ItemsProcessed, result.ItemsFailed)
	return result, nil
}

// dispatchPriceUpdates performs the marketplace UpdatePrice call for each job (no DB
// transaction open) and persists the resulting sync status per listing. Async feed
// providers (e.g. Amazon Feeds API) yield sync_status='pending' plus feed metadata;
// synchronous providers yield 'synced'.
func (s *ListingSyncService) dispatchPriceUpdates(ctx context.Context, tenantID, configID uuid.UUID, integ *model.IntegrationWithCreds, provider integration.MarketplaceProvider, jobs []listingPriceJob, result *SyncResult) {
	syncStatus := "synced"
	if _, ok := provider.(integration.AsyncPriceUpdater); ok {
		syncStatus = "pending"
	}

	for _, job := range jobs {
		if pushErr := provider.UpdatePrice(ctx, *job.listing.ExternalID, job.adjustedPrice); pushErr != nil {
			s.logger.Error("listing sync: push price failed",
				"listing_id", job.listing.ID,
				"external_id", *job.listing.ExternalID,
				"provider", integ.Provider,
				"error", pushErr,
			)
			s.persistPriceError(ctx, tenantID, configID, job.listing, truncateServiceErrorMsg(pushErr.Error()))
			result.ItemsFailed++
			continue
		}
		// Capture feed metadata AFTER each successful push: an async feed provider
		// (Amazon Feeds API) populates FeedResult() only when UpdatePrice submits a
		// feed, and each push yields its own feed id. Hoisting this above the loop
		// would persist a nil/stale feed id and leave pending listings unreconcilable
		// by the feed-status worker. Mirrors PriceSyncWorker.syncOneByOne.
		feedMeta := feedMetaForProvider(provider)
		s.persistPricePush(ctx, tenantID, configID, job.listing, job.adjustedPrice, syncStatus, feedMeta)
		result.ItemsProcessed++
	}
}

// SyncStock delegates stock propagation to the canonical owner, StockSyncService.
//
// Stock is owned end-to-end by StockSyncWorker/StockSyncService, which read
// warehouse_stock (the canonical source) and push via the provider's UpdateStock API
// with async-feed and zero-stock-deactivation handling. ListingSyncService must NOT
// add a second stock pusher: the previous implementation wrote stock_override from
// products.stock_quantity — the wrong, stale source — which actively corrupted the
// override the stock owner reads. Here we only resolve the eligible products and hand
// each to the stock owner.
func (s *ListingSyncService) SyncStock(ctx context.Context, tenantID uuid.UUID, configID uuid.UUID) (*SyncResult, error) {
	cfg, err := s.Get(ctx, tenantID, configID)
	if err != nil {
		return nil, err
	}

	result := &SyncResult{}

	if s.stockSyncSvc == nil {
		// Stock is delegated; without the stock owner wired there is nothing to push.
		result.Message = "stock sync service not configured; skipped"
		return result, nil
	}

	// Phase 1: gather the distinct eligible product ids inside a short transaction.
	var productIDs []uuid.UUID
	err = database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		listings, listErr := s.listingRepo.ListByIntegration(ctx, tx, cfg.IntegrationID)
		if listErr != nil {
			return fmt.Errorf("list listings: %w", listErr)
		}
		seen := make(map[uuid.UUID]struct{}, len(listings))
		for _, listing := range listings {
			if !isPushEligible(listing) {
				continue
			}
			if _, ok := seen[listing.ProductID]; ok {
				continue
			}
			seen[listing.ProductID] = struct{}{}
			productIDs = append(productIDs, listing.ProductID)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Phase 2: delegate each product to the stock owner (warehouse_stock -> UpdateStock).
	for _, productID := range productIDs {
		if pushErr := s.stockSyncSvc.PushStockToAllChannels(ctx, tenantID, productID); pushErr != nil {
			s.logStockDelegation(ctx, tenantID, configID, productID, "failed", truncateServiceErrorMsg(pushErr.Error()))
			result.ItemsFailed++
			continue
		}
		s.logStockDelegation(ctx, tenantID, configID, productID, "success", "")
		result.ItemsProcessed++
	}

	s.updateLastSync(ctx, tenantID, configID, result.ItemsFailed > 0)
	result.Message = fmt.Sprintf("Delegated stock sync for %d products, %d errors", result.ItemsProcessed, result.ItemsFailed)
	return result, nil
}

// logStockDelegation records a stock-delegation sync-log entry in its own short tx.
func (s *ListingSyncService) logStockDelegation(ctx context.Context, tenantID, configID, productID uuid.UUID, status, errMsg string) {
	if err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		s.logSyncEntry(ctx, tx, tenantID, configID, "push", "stock", &productID, nil, status, errMsg)
		return nil
	}); err != nil {
		s.logger.Error("listing sync: persist stock delegation log failed", "product_id", productID, "error", err)
	}
}

// RunFullSync orchestrates a full sync: products, prices, and stock.
func (s *ListingSyncService) RunFullSync(ctx context.Context, tenantID uuid.UUID, configID uuid.UUID) (*SyncResult, error) {
	cfg, err := s.Get(ctx, tenantID, configID)
	if err != nil {
		return nil, err
	}

	totalResult := &SyncResult{}

	// Push products if direction allows
	if cfg.SyncDirection == "push" || cfg.SyncDirection == "bidirectional" {
		if prodResult, prodErr := s.SyncProducts(ctx, tenantID, configID); prodErr == nil {
			totalResult.ItemsProcessed += prodResult.ItemsProcessed
			totalResult.ItemsFailed += prodResult.ItemsFailed
		} else {
			s.logger.Error("listing sync: product sync failed", "config_id", configID, "error", prodErr)
		}

		if priceResult, priceErr := s.SyncPrices(ctx, tenantID, configID); priceErr == nil {
			totalResult.ItemsProcessed += priceResult.ItemsProcessed
			totalResult.ItemsFailed += priceResult.ItemsFailed
		} else {
			s.logger.Error("listing sync: price sync failed", "config_id", configID, "error", priceErr)
		}

		if stockResult, stockErr := s.SyncStock(ctx, tenantID, configID); stockErr == nil {
			totalResult.ItemsProcessed += stockResult.ItemsProcessed
			totalResult.ItemsFailed += stockResult.ItemsFailed
		} else {
			s.logger.Error("listing sync: stock sync failed", "config_id", configID, "error", stockErr)
		}
	}

	// Pull listings if direction allows
	if cfg.SyncDirection == "pull" || cfg.SyncDirection == "bidirectional" {
		if pullResult, pullErr := s.PullListings(ctx, tenantID, configID); pullErr == nil {
			totalResult.ItemsProcessed += pullResult.ItemsProcessed
			totalResult.ItemsFailed += pullResult.ItemsFailed
		} else {
			s.logger.Error("listing sync: pull sync failed", "config_id", configID, "error", pullErr)
		}
	}

	totalResult.Message = fmt.Sprintf("Full sync: %d operations, %d errors", totalResult.ItemsProcessed, totalResult.ItemsFailed)
	return totalResult, nil
}

// --- Helpers ---

// productMapForListings batch-fetches the products linked to the given listings
// and returns them keyed by product ID.
func (s *ListingSyncService) productMapForListings(ctx context.Context, tx pgx.Tx, listings []*model.ProductListing) (map[uuid.UUID]*model.Product, error) {
	productIDs := make([]uuid.UUID, 0, len(listings))
	seen := make(map[uuid.UUID]struct{}, len(listings))
	for _, l := range listings {
		if _, ok := seen[l.ProductID]; ok {
			continue
		}
		seen[l.ProductID] = struct{}{}
		productIDs = append(productIDs, l.ProductID)
	}
	products, err := s.productRepo.FindByIDs(ctx, tx, productIDs)
	if err != nil {
		return nil, fmt.Errorf("batch fetch products: %w", err)
	}
	productMap := make(map[uuid.UUID]*model.Product, len(products))
	for i := range products {
		productMap[products[i].ID] = &products[i]
	}
	return productMap, nil
}

func (s *ListingSyncService) applyPriceRule(cfg *model.ListingSyncConfig, basePrice float64) float64 {
	switch cfg.PriceRule {
	case "markup_pct":
		return math.Round((basePrice*(1+cfg.PriceModifier/100))*100) / 100
	case "markup_fixed":
		return math.Round((basePrice+cfg.PriceModifier)*100) / 100
	default:
		return basePrice
	}
}

// isPushEligible reports whether a listing may be pushed to the marketplace: it must
// have a non-empty external offer id, be active, and be in auto sync mode. This is
// the same gate the wired StockSyncWorker / PriceSyncWorker apply, so listings an
// operator set to manual or inactive are never pushed.
func isPushEligible(listing *model.ProductListing) bool {
	return listing.ExternalID != nil && *listing.ExternalID != "" &&
		listing.Status == "active" && listing.StockSyncMode == "auto"
}

// marketplaceProvider decrypts the integration credentials and constructs its
// marketplace provider. The caller owns the provider lifecycle and must call
// closeMarketplaceProvider when done.
func (s *ListingSyncService) marketplaceProvider(integ *model.IntegrationWithCreds) (integration.MarketplaceProvider, error) {
	credJSON, err := crypto.Decrypt(integ.EncryptedCredentials, s.encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("decrypt credentials: %w", err)
	}
	provider, err := integration.NewMarketplaceProvider(integ.Provider, credJSON, integ.Settings)
	if err != nil {
		return nil, fmt.Errorf("create provider: %w", err)
	}
	return provider, nil
}

// feedMetaForProvider extracts async feed metadata (e.g. an Amazon feed id) from a
// provider implementing AsyncFeedResult, or nil when there is none. Mirrors the
// worker's buildFeedMeta so pending listings carry the feed id for status polling.
func feedMetaForProvider(provider any) []byte {
	if fr, ok := provider.(integration.AsyncFeedResult); ok {
		if result := fr.FeedResult(); result != nil {
			data, _ := json.Marshal(map[string]string{
				"amazon_feed_id":   result.FeedID,
				"amazon_feed_type": result.FeedType,
			})
			return data
		}
	}
	return nil
}

// persistPricePush records a successful price push in its own short transaction: it
// writes the markup-applied price to price_override (the per-config value the
// periodic PriceSyncWorker also pushes), sets sync_status ('pending' + feed metadata
// for async feed providers, 'synced' + last_synced_at otherwise) and writes a
// success sync-log entry.
func (s *ListingSyncService) persistPricePush(ctx context.Context, tenantID, configID uuid.UUID, listing *model.ProductListing, adjustedPrice float64, syncStatus string, feedMeta []byte) {
	if err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var execErr error
		switch {
		case syncStatus == "pending" && feedMeta != nil:
			_, execErr = tx.Exec(ctx,
				`UPDATE product_listings SET price_override = $2, sync_status = 'pending', error_message = NULL,
				 metadata = COALESCE(metadata, '{}'::jsonb) || $3::jsonb, updated_at = NOW() WHERE id = $1`,
				listing.ID, adjustedPrice, string(feedMeta))
		case syncStatus == "pending":
			_, execErr = tx.Exec(ctx,
				`UPDATE product_listings SET price_override = $2, sync_status = 'pending', error_message = NULL, updated_at = NOW() WHERE id = $1`,
				listing.ID, adjustedPrice)
		default:
			_, execErr = tx.Exec(ctx,
				`UPDATE product_listings SET price_override = $2, sync_status = 'synced', error_message = NULL, last_synced_at = NOW(), updated_at = NOW() WHERE id = $1`,
				listing.ID, adjustedPrice)
		}
		if execErr != nil {
			return execErr
		}
		changes, _ := json.Marshal(map[string]any{
			"old_price": listing.PriceOverride,
			"new_price": adjustedPrice,
		})
		s.logSyncEntryWithChanges(ctx, tx, tenantID, configID, "push", "price", &listing.ProductID, listing.ExternalID, "success", changes)
		return nil
	}); err != nil {
		s.logger.Error("listing sync: persist price push failed", "listing_id", listing.ID, "error", err)
	}
}

// persistPriceError marks a listing's price push as failed in its own short
// transaction and writes a failed sync-log entry.
func (s *ListingSyncService) persistPriceError(ctx context.Context, tenantID, configID uuid.UUID, listing *model.ProductListing, errMsg string) {
	if err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		if _, execErr := tx.Exec(ctx,
			`UPDATE product_listings SET sync_status = 'error', error_message = $2, updated_at = NOW() WHERE id = $1`,
			listing.ID, errMsg); execErr != nil {
			return execErr
		}
		s.logSyncEntry(ctx, tx, tenantID, configID, "push", "price", &listing.ProductID, listing.ExternalID, "failed", errMsg)
		return nil
	}); err != nil {
		s.logger.Error("listing sync: persist price error failed", "listing_id", listing.ID, "error", err)
	}
}

func (s *ListingSyncService) logSyncEntry(ctx context.Context, tx pgx.Tx, tenantID, configID uuid.UUID, direction, entityType string, productID *uuid.UUID, externalID *string, status, errMsg string) {
	var errMsgPtr *string
	if errMsg != "" {
		errMsgPtr = &errMsg
	}
	log := &model.ListingSyncLog{
		ID:           uuid.New(),
		TenantID:     tenantID,
		ConfigID:     configID,
		Direction:    direction,
		EntityType:   entityType,
		ProductID:    productID,
		ExternalID:   externalID,
		Status:       status,
		ErrorMessage: errMsgPtr,
	}
	if createErr := s.syncRepo.CreateLog(ctx, tx, log); createErr != nil {
		s.logger.Error("listing sync: failed to create log", "error", createErr)
	}
}

func (s *ListingSyncService) logSyncEntryWithChanges(ctx context.Context, tx pgx.Tx, tenantID, configID uuid.UUID, direction, entityType string, productID *uuid.UUID, externalID *string, status string, changes json.RawMessage) {
	log := &model.ListingSyncLog{
		ID:         uuid.New(),
		TenantID:   tenantID,
		ConfigID:   configID,
		Direction:  direction,
		EntityType: entityType,
		ProductID:  productID,
		ExternalID: externalID,
		Status:     status,
		Changes:    changes,
	}
	if createErr := s.syncRepo.CreateLog(ctx, tx, log); createErr != nil {
		s.logger.Error("listing sync: failed to create log", "error", createErr)
	}
}

func (s *ListingSyncService) updateLastSync(ctx context.Context, tenantID uuid.UUID, configID uuid.UUID, hadErrors bool) {
	var lastError *string
	if hadErrors {
		errMsg := "Some items were not synchronized"
		lastError = &errMsg
	}
	if err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		return s.syncRepo.UpdateLastSync(ctx, tx, configID, lastError)
	}); err != nil {
		slog.Error("failed to update last sync timestamp", "error", err, "tenant_id", tenantID, "config_id", configID)
	}
}
