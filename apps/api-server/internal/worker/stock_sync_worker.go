package worker

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/crypto"
	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
)

const stockBulkBatchSize = 100

// listingStock holds the data needed to sync stock for a single product listing.
type listingStock struct {
	ListingID  string
	ExternalID string
	StockQty   int
}

type StockSyncWorker struct {
	pool          *pgxpool.Pool
	encryptionKey []byte
	logger        *slog.Logger
}

func NewStockSyncWorker(pool *pgxpool.Pool, encryptionKey []byte, logger *slog.Logger) *StockSyncWorker {
	return &StockSyncWorker{
		pool:          pool,
		encryptionKey: encryptionKey,
		logger:        logger,
	}
}

func (w *StockSyncWorker) Name() string {
	return "stock_sync"
}

func (w *StockSyncWorker) Interval() time.Duration {
	return 5 * time.Minute
}

func (w *StockSyncWorker) Run(ctx context.Context) error {
	// Get all active marketplace integrations (all providers)
	tis, err := ListAllActiveMarketplaceIntegrations(ctx, w.pool)
	if err != nil {
		return err
	}

	totalSynced := 0

	for _, ti := range tis {
		credJSON, err := crypto.Decrypt(ti.Credentials, w.encryptionKey)
		if err != nil {
			w.logger.Error("stock sync: failed to decrypt credentials", "integration_id", ti.IntegrationID, "error", err)
			continue
		}

		provider, err := integration.NewMarketplaceProvider(ti.Provider, credJSON, ti.Settings)
		if err != nil {
			w.logger.Error("stock sync: failed to create provider", "integration_id", ti.IntegrationID, "error", err)
			continue
		}

		// Phase 1: Gather listings inside a short transaction.
		var listings []listingStock
		gatherErr := database.WithTenant(ctx, w.pool, ti.TenantID, func(tx pgx.Tx) error {
			rows, err := tx.Query(ctx,
				`SELECT pl.id, pl.external_id, pl.stock_override,
				        GREATEST(COALESCE(SUM(ws.quantity), 0) - COALESCE(SUM(ws.reserved), 0), 0) AS available_qty
				 FROM product_listings pl
				 LEFT JOIN warehouse_stock ws ON ws.product_id = pl.product_id AND ws.variant_id IS NULL
				 WHERE pl.integration_id = $1 AND pl.status = 'active'
				   AND pl.external_id IS NOT NULL AND pl.stock_sync_mode = 'auto'
				 GROUP BY pl.id, pl.external_id, pl.stock_override
				 LIMIT 5000`,
				ti.IntegrationID,
			)
			if err != nil {
				return err
			}
			defer rows.Close()

			for rows.Next() {
				var listingID, externalID string
				var stockOverride *int
				var availableQty int
				if err := rows.Scan(&listingID, &externalID, &stockOverride, &availableQty); err != nil {
					w.logger.Error("stock sync: scan listing", "error", err)
					continue
				}

				stockQty := availableQty
				if stockOverride != nil {
					stockQty = *stockOverride
				}

				listings = append(listings, listingStock{
					ListingID:  listingID,
					ExternalID: externalID,
					StockQty:   stockQty,
				})
			}
			return rows.Err()
		})
		if gatherErr != nil {
			w.logger.Error("stock sync: gather listings failed", "tenant_id", ti.TenantID, "error", gatherErr)
			closeProvider(provider)
			continue
		}

		// Phase 2: Call marketplace APIs outside the transaction.
		// Deactivate zero-stock listings via ListingDeactivator if supported.
		deactivator, hasDeactivator := provider.(integration.ListingDeactivator)
		var nonZeroStock []listingStock
		if hasDeactivator {
			for _, l := range listings {
				if l.StockQty == 0 {
					totalSynced += w.deactivateListing(ctx, ti, deactivator, l)
				} else {
					nonZeroStock = append(nonZeroStock, l)
				}
			}
		} else {
			nonZeroStock = listings
		}

		// Sync non-zero stock listings (and zero-stock if no deactivator).
		bulkProvider, hasBulk := provider.(integration.BulkStockUpdater)
		if hasBulk {
			totalSynced += w.syncBulk(ctx, ti, bulkProvider, nonZeroStock)
		} else {
			totalSynced += w.syncOneByOne(ctx, ti, provider, nonZeroStock)
		}

		tenantErr := gatherErr
		closeProvider(provider)
		if tenantErr != nil {
			w.logger.Error("stock sync: tenant error", "tenant_id", ti.TenantID, "error", tenantErr)
			continue
		}
	}

	w.logger.Info("stock sync completed", "tenants", len(tis), "synced", totalSynced)
	return nil
}

// syncBulk sends stock updates in batches of stockBulkBatchSize.
func (w *StockSyncWorker) syncBulk(
	ctx context.Context,
	ti TenantIntegration,
	provider integration.BulkStockUpdater,
	listings []listingStock,
) int {
	synced := 0

	// Build StockUpdate slice for chunking
	updates := make([]integration.StockUpdate, len(listings))
	for i, l := range listings {
		updates[i] = integration.StockUpdate{
			ExternalOfferID: l.ExternalID,
			Quantity:        l.StockQty,
		}
	}

	chunks := chunkStockUpdates(updates, stockBulkBatchSize)

	// Process each chunk; the listing slice is indexed in parallel with the update slice
	offset := 0
	for _, chunk := range chunks {
		batchListings := listings[offset : offset+len(chunk)]
		offset += len(chunk)

		if err := provider.BulkUpdateStock(ctx, chunk); err != nil {
			w.logger.Error("stock sync: bulk update failed",
				"operation", "listing.stock_bulk_update",
				"tenant_id", ti.TenantID,
				"batch_size", len(chunk),
				"error", err,
			)
			errMsg := truncateErrorMessage(err.Error())
			w.updateListingsStatus(ctx, ti.TenantID, batchListings, "error", &errMsg, nil)
			continue
		}

		syncStatus := "synced"
		if _, ok := provider.(integration.AsyncStockUpdater); ok {
			syncStatus = "pending"
		}
		var feedMeta []byte
		if syncStatus == "pending" {
			if fr, ok := provider.(integration.AsyncFeedResult); ok {
				if result := fr.FeedResult(); result != nil {
					feedMeta, _ = json.Marshal(map[string]string{
						"amazon_feed_id":   result.FeedID,
						"amazon_feed_type": result.FeedType,
					})
				}
			}
		}
		w.updateListingsStatus(ctx, ti.TenantID, batchListings, syncStatus, nil, feedMeta)
		w.logger.Info("worker: stock batch synced",
			"operation", "listing.stock_bulk_update",
			"tenant_id", ti.TenantID,
			"batch_size", len(chunk),
			"sync_status", syncStatus,
		)
		synced += len(chunk)
	}

	return synced
}

// syncOneByOne updates stock one listing at a time (fallback for providers without bulk support).
func (w *StockSyncWorker) syncOneByOne(
	ctx context.Context,
	ti TenantIntegration,
	provider integration.MarketplaceProvider,
	listings []listingStock,
) int {
	synced := 0

	for _, l := range listings {
		if err := provider.UpdateStock(ctx, l.ExternalID, l.StockQty); err != nil {
			w.logger.Error("stock sync: update stock failed",
				"operation", "listing.stock_update",
				"tenant_id", ti.TenantID,
				"entity_id", l.ListingID,
				"external_id", l.ExternalID,
				"error", err,
			)
			errMsg := truncateErrorMessage(err.Error())
			w.updateListingsStatus(ctx, ti.TenantID, []listingStock{l}, "error", &errMsg, nil)
			continue
		}

		syncStatus := "synced"
		if _, ok := provider.(integration.AsyncStockUpdater); ok {
			syncStatus = "pending"
		}
		var feedMeta []byte
		if syncStatus == "pending" {
			if fr, ok := provider.(integration.AsyncFeedResult); ok {
				if result := fr.FeedResult(); result != nil {
					feedMeta, _ = json.Marshal(map[string]string{
						"amazon_feed_id":   result.FeedID,
						"amazon_feed_type": result.FeedType,
					})
				}
			}
		}
		w.updateListingsStatus(ctx, ti.TenantID, []listingStock{l}, syncStatus, nil, feedMeta)
		w.logger.Info("worker: stock synced",
			"operation", "listing.stock_update",
			"tenant_id", ti.TenantID,
			"entity_id", l.ListingID,
			"external_id", l.ExternalID,
			"stock_quantity", l.StockQty,
			"sync_status", syncStatus,
		)
		synced++
	}

	return synced
}

// deactivateListing deactivates a single listing via ListingDeactivator and updates its status.
func (w *StockSyncWorker) deactivateListing(
	ctx context.Context,
	ti TenantIntegration,
	deactivator integration.ListingDeactivator,
	l listingStock,
) int {
	if err := deactivator.DeactivateOffer(ctx, l.ExternalID); err != nil {
		w.logger.Error("stock sync: deactivate listing failed",
			"operation", "listing.deactivate",
			"tenant_id", ti.TenantID,
			"entity_id", l.ListingID,
			"external_id", l.ExternalID,
			"error", err,
		)
		errMsg := truncateErrorMessage(err.Error())
		w.updateListingsStatus(ctx, ti.TenantID, []listingStock{l}, "error", &errMsg, nil)
		return 0
	}

	// Mark listing as inactive + synced
	_ = database.WithTenant(ctx, w.pool, ti.TenantID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx,
			`UPDATE product_listings SET status = 'inactive', sync_status = 'synced', error_message = NULL, last_synced_at = NOW(), updated_at = NOW() WHERE id = $1`,
			l.ListingID,
		)
		return err
	})
	w.logger.Info("worker: listing deactivated (stock=0)",
		"operation", "listing.deactivate",
		"tenant_id", ti.TenantID,
		"entity_id", l.ListingID,
		"external_id", l.ExternalID,
	)
	return 1
}

// updateListingsStatus updates sync_status for listings in a short transaction (Phase 2 DB writes).
// feedMeta is optional JSONB to merge into metadata (used for Amazon feed IDs on "pending" status).
func (w *StockSyncWorker) updateListingsStatus(ctx context.Context, tenantID uuid.UUID, listings []listingStock, status string, errMsg *string, feedMeta []byte) {
	_ = database.WithTenant(ctx, w.pool, tenantID, func(tx pgx.Tx) error {
		for _, l := range listings {
			var execErr error
			switch status {
			case "synced":
				_, execErr = tx.Exec(ctx,
					`UPDATE product_listings SET sync_status = 'synced', error_message = NULL, last_synced_at = NOW(), updated_at = NOW() WHERE id = $1`,
					l.ListingID,
				)
			case "pending":
				if feedMeta != nil {
					_, execErr = tx.Exec(ctx,
						`UPDATE product_listings SET sync_status = 'pending', error_message = NULL, metadata = COALESCE(metadata, '{}'::jsonb) || $2::jsonb, updated_at = NOW() WHERE id = $1`,
						l.ListingID, feedMeta,
					)
				} else {
					_, execErr = tx.Exec(ctx,
						`UPDATE product_listings SET sync_status = 'pending', error_message = NULL, updated_at = NOW() WHERE id = $1`,
						l.ListingID,
					)
				}
			default: // "error"
				msg := ""
				if errMsg != nil {
					msg = *errMsg
				}
				_, execErr = tx.Exec(ctx,
					`UPDATE product_listings SET sync_status = 'error', error_message = $2, updated_at = NOW() WHERE id = $1`,
					l.ListingID, msg,
				)
			}
			if execErr != nil {
				w.logger.Error("stock sync: failed to update listing sync status", "listing_id", l.ListingID, "error", execErr)
			}
		}
		return nil
	})
}

// chunkStockUpdates splits a slice of StockUpdate into chunks of the given size.
func chunkStockUpdates(items []integration.StockUpdate, size int) [][]integration.StockUpdate {
	if len(items) == 0 {
		return nil
	}
	var chunks [][]integration.StockUpdate
	for i := 0; i < len(items); i += size {
		end := min(i+size, len(items))
		chunks = append(chunks, items[i:end])
	}
	return chunks
}
