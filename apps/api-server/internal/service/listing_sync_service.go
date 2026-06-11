package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

var (
	// ErrListingSyncConfigNotFound is returned when a listing sync configuration does not exist.
	ErrListingSyncConfigNotFound = errors.New("listing sync config not found")
)

// ListingSyncService handles CRUD and sync orchestration for listing sync configs.
type ListingSyncService struct {
	syncRepo    *repository.ListingSyncRepository
	productRepo repository.ProductRepo
	listingRepo *repository.ProductListingRepository
	auditRepo   repository.AuditRepo
	pool        *pgxpool.Pool
	logger      *slog.Logger
}

// NewListingSyncService creates a new ListingSyncService.
func NewListingSyncService(
	syncRepo *repository.ListingSyncRepository,
	productRepo repository.ProductRepo,
	listingRepo *repository.ProductListingRepository,
	auditRepo repository.AuditRepo,
	pool *pgxpool.Pool,
	logger *slog.Logger,
) *ListingSyncService {
	return &ListingSyncService{
		syncRepo:    syncRepo,
		productRepo: productRepo,
		listingRepo: listingRepo,
		auditRepo:   auditRepo,
		pool:        pool,
		logger:      logger,
	}
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

// SyncProducts pushes local products to the marketplace as new listings or updates existing ones.
func (s *ListingSyncService) SyncProducts(ctx context.Context, tenantID uuid.UUID, configID uuid.UUID) (*SyncResult, error) {
	cfg, err := s.Get(ctx, tenantID, configID)
	if err != nil {
		return nil, err
	}

	result := &SyncResult{}
	err = database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		// Get all products for the tenant
		products, _, listErr := s.productRepo.List(ctx, tx, model.ProductListFilter{
			PaginationParams: model.PaginationParams{Limit: 1000, Offset: 0},
		})
		if listErr != nil {
			return fmt.Errorf("list products: %w", listErr)
		}

		for _, product := range products {
			// Check if a listing already exists for this product + integration
			existing, findErr := s.listingRepo.FindByProductAndIntegration(ctx, tx, product.ID, cfg.IntegrationID)
			if findErr != nil {
				s.logSyncEntry(ctx, tx, tenantID, configID, "push", "product", &product.ID, nil, "failed", findErr.Error())
				result.ItemsFailed++
				continue
			}

			if existing == nil {
				// Create a placeholder listing record (actual marketplace push is provider-specific)
				listing := &model.ProductListing{
					ID:            uuid.New(),
					TenantID:      tenantID,
					ProductID:     product.ID,
					IntegrationID: cfg.IntegrationID,
					Status:        "pending",
					SyncStatus:    "pending",
					Metadata:      json.RawMessage("{}"),
				}

				// Apply price rule
				adjustedPrice := s.applyPriceRule(cfg, product.Price)
				if adjustedPrice != product.Price {
					listing.PriceOverride = &adjustedPrice
				}

				// Apply stock buffer
				if cfg.StockBuffer > 0 {
					bufferedStock := max(product.StockQuantity-cfg.StockBuffer, 0)
					listing.StockOverride = &bufferedStock
				}

				if createErr := s.listingRepo.Create(ctx, tx, listing); createErr != nil {
					s.logSyncEntry(ctx, tx, tenantID, configID, "push", "product", &product.ID, nil, "failed", createErr.Error())
					result.ItemsFailed++
					continue
				}

				changes, _ := json.Marshal(map[string]any{
					"action":  "created_listing",
					"product": product.Name,
				})
				s.logSyncEntryWithChanges(ctx, tx, tenantID, configID, "push", "product", &product.ID, nil, "success", changes)
				result.ItemsProcessed++
			} else {
				// Listing exists — update price/stock overrides if needed
				needsUpdate := false
				updateReq := &model.UpdateProductListingRequest{}

				adjustedPrice := s.applyPriceRule(cfg, product.Price)
				if existing.PriceOverride == nil || *existing.PriceOverride != adjustedPrice {
					updateReq.PriceOverride = &adjustedPrice
					needsUpdate = true
				}

				if cfg.StockBuffer > 0 {
					bufferedStock := max(product.StockQuantity-cfg.StockBuffer, 0)
					if existing.StockOverride == nil || *existing.StockOverride != bufferedStock {
						updateReq.StockOverride = &bufferedStock
						needsUpdate = true
					}
				}

				if needsUpdate {
					if updateErr := s.listingRepo.Update(ctx, tx, existing.ID, updateReq); updateErr != nil {
						s.logSyncEntry(ctx, tx, tenantID, configID, "push", "product", &product.ID, existing.ExternalID, "failed", updateErr.Error())
						result.ItemsFailed++
						continue
					}
				}

				s.logSyncEntry(ctx, tx, tenantID, configID, "push", "product", &product.ID, existing.ExternalID, "success", "")
				result.ItemsProcessed++
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// Update last sync
	s.updateLastSync(ctx, tenantID, configID, result.ItemsFailed > 0)

	result.Message = fmt.Sprintf("Synced %d products, %d errors", result.ItemsProcessed, result.ItemsFailed)
	return result, nil
}

// PullListings imports marketplace listings as local products.
func (s *ListingSyncService) PullListings(ctx context.Context, tenantID uuid.UUID, configID uuid.UUID) (*SyncResult, error) {
	cfg, err := s.Get(ctx, tenantID, configID)
	if err != nil {
		return nil, err
	}

	result := &SyncResult{}

	err = database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		// Get all existing listings for this integration
		listings, listErr := s.listingRepo.ListByIntegration(ctx, tx, cfg.IntegrationID)
		if listErr != nil {
			return fmt.Errorf("list listings: %w", listErr)
		}

		// Batch-fetch all linked products
		productIDs := make([]uuid.UUID, 0, len(listings))
		for _, l := range listings {
			productIDs = append(productIDs, l.ProductID)
		}
		products, prodErr := s.productRepo.FindByIDs(ctx, tx, productIDs)
		if prodErr != nil {
			return fmt.Errorf("batch fetch products: %w", prodErr)
		}
		productMap := make(map[uuid.UUID]*model.Product, len(products))
		for i := range products {
			productMap[products[i].ID] = &products[i]
		}

		for _, listing := range listings {
			if listing.ExternalID == nil || *listing.ExternalID == "" {
				s.logSyncEntry(ctx, tx, tenantID, configID, "pull", "offer", nil, nil, "skipped", "no external_id")
				continue
			}

			product := productMap[listing.ProductID]
			if product == nil {
				s.logSyncEntry(ctx, tx, tenantID, configID, "pull", "offer", &listing.ProductID, listing.ExternalID, "skipped", "product not found")
				continue
			}

			// Mark as synced (actual marketplace data pull is provider-specific)
			syncStatus := "synced"
			now := time.Now()
			updateReq := &model.UpdateProductListingRequest{
				SyncStatus: &syncStatus,
			}
			if updateErr := s.listingRepo.Update(ctx, tx, listing.ID, updateReq); updateErr != nil {
				result.ItemsFailed++
				continue
			}
			// Update last_synced_at
			_, _ = tx.Exec(ctx, "UPDATE product_listings SET last_synced_at = $1 WHERE id = $2", now, listing.ID)

			s.logSyncEntry(ctx, tx, tenantID, configID, "pull", "offer", &listing.ProductID, listing.ExternalID, "success", "")
			result.ItemsProcessed++
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	s.updateLastSync(ctx, tenantID, configID, result.ItemsFailed > 0)
	result.Message = fmt.Sprintf("Fetched %d offers, %d errors", result.ItemsProcessed, result.ItemsFailed)
	return result, nil
}

// SyncPrices pushes price updates from local products to marketplace listings.
func (s *ListingSyncService) SyncPrices(ctx context.Context, tenantID uuid.UUID, configID uuid.UUID) (*SyncResult, error) {
	cfg, err := s.Get(ctx, tenantID, configID)
	if err != nil {
		return nil, err
	}

	result := &SyncResult{}
	err = database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		listings, listErr := s.listingRepo.ListByIntegration(ctx, tx, cfg.IntegrationID)
		if listErr != nil {
			return fmt.Errorf("list listings: %w", listErr)
		}

		// Batch-fetch all linked products
		productIDs := make([]uuid.UUID, 0, len(listings))
		for _, l := range listings {
			productIDs = append(productIDs, l.ProductID)
		}
		products, prodErr := s.productRepo.FindByIDs(ctx, tx, productIDs)
		if prodErr != nil {
			return fmt.Errorf("batch fetch products: %w", prodErr)
		}
		productMap := make(map[uuid.UUID]*model.Product, len(products))
		for i := range products {
			productMap[products[i].ID] = &products[i]
		}

		for _, listing := range listings {
			product := productMap[listing.ProductID]
			if product == nil {
				s.logSyncEntry(ctx, tx, tenantID, configID, "push", "price", &listing.ProductID, listing.ExternalID, "failed", "product not found")
				result.ItemsFailed++
				continue
			}

			adjustedPrice := s.applyPriceRule(cfg, product.Price)
			if listing.PriceOverride != nil && *listing.PriceOverride == adjustedPrice {
				// No change needed
				result.ItemsProcessed++
				continue
			}

			updateReq := &model.UpdateProductListingRequest{
				PriceOverride: &adjustedPrice,
			}
			if updateErr := s.listingRepo.Update(ctx, tx, listing.ID, updateReq); updateErr != nil {
				s.logSyncEntry(ctx, tx, tenantID, configID, "push", "price", &listing.ProductID, listing.ExternalID, "failed", updateErr.Error())
				result.ItemsFailed++
				continue
			}

			changes, _ := json.Marshal(map[string]any{
				"old_price": listing.PriceOverride,
				"new_price": adjustedPrice,
			})
			s.logSyncEntryWithChanges(ctx, tx, tenantID, configID, "push", "price", &listing.ProductID, listing.ExternalID, "success", changes)
			result.ItemsProcessed++
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	s.updateLastSync(ctx, tenantID, configID, result.ItemsFailed > 0)
	result.Message = fmt.Sprintf("Synced prices for %d offers, %d errors", result.ItemsProcessed, result.ItemsFailed)
	return result, nil
}

// SyncStock pushes stock levels from local products to marketplace listings.
func (s *ListingSyncService) SyncStock(ctx context.Context, tenantID uuid.UUID, configID uuid.UUID) (*SyncResult, error) {
	cfg, err := s.Get(ctx, tenantID, configID)
	if err != nil {
		return nil, err
	}

	result := &SyncResult{}
	err = database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		listings, listErr := s.listingRepo.ListByIntegration(ctx, tx, cfg.IntegrationID)
		if listErr != nil {
			return fmt.Errorf("list listings: %w", listErr)
		}

		// Batch-fetch all linked products
		productIDs := make([]uuid.UUID, 0, len(listings))
		for _, l := range listings {
			productIDs = append(productIDs, l.ProductID)
		}
		products, prodErr := s.productRepo.FindByIDs(ctx, tx, productIDs)
		if prodErr != nil {
			return fmt.Errorf("batch fetch products: %w", prodErr)
		}
		productMap := make(map[uuid.UUID]*model.Product, len(products))
		for i := range products {
			productMap[products[i].ID] = &products[i]
		}

		for _, listing := range listings {
			product := productMap[listing.ProductID]
			if product == nil {
				s.logSyncEntry(ctx, tx, tenantID, configID, "push", "stock", &listing.ProductID, listing.ExternalID, "failed", "product not found")
				result.ItemsFailed++
				continue
			}

			bufferedStock := max(product.StockQuantity-cfg.StockBuffer, 0)

			if listing.StockOverride != nil && *listing.StockOverride == bufferedStock {
				// No change needed
				result.ItemsProcessed++
				continue
			}

			updateReq := &model.UpdateProductListingRequest{
				StockOverride: &bufferedStock,
			}
			if updateErr := s.listingRepo.Update(ctx, tx, listing.ID, updateReq); updateErr != nil {
				s.logSyncEntry(ctx, tx, tenantID, configID, "push", "stock", &listing.ProductID, listing.ExternalID, "failed", updateErr.Error())
				result.ItemsFailed++
				continue
			}

			changes, _ := json.Marshal(map[string]any{
				"old_stock":    listing.StockOverride,
				"new_stock":    bufferedStock,
				"stock_buffer": cfg.StockBuffer,
			})
			s.logSyncEntryWithChanges(ctx, tx, tenantID, configID, "push", "stock", &listing.ProductID, listing.ExternalID, "success", changes)
			result.ItemsProcessed++
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	s.updateLastSync(ctx, tenantID, configID, result.ItemsFailed > 0)
	result.Message = fmt.Sprintf("Synced stock for %d offers, %d errors", result.ItemsProcessed, result.ItemsFailed)
	return result, nil
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
