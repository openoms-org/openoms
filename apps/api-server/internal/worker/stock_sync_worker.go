package worker

import (
	"context"
	"log/slog"
	"time"

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

		tenantErr := database.WithTenant(ctx, w.pool, ti.TenantID, func(tx pgx.Tx) error {
			// Query auto-sync product_listings with warehouse-based available stock.
			// stock_sync_mode = 'auto' gates both stock and price sync (single toggle per listing).
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

			// Collect all listings into a slice
			var listings []listingStock
			for rows.Next() {
				var listingID, externalID string
				var stockOverride *int
				var availableQty int
				if err := rows.Scan(&listingID, &externalID, &stockOverride, &availableQty); err != nil {
					w.logger.Error("stock sync: scan listing", "error", err)
					continue
				}

				// Use stock_override if set
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
			if err := rows.Err(); err != nil {
				return err
			}

			// Deactivate zero-stock listings via ListingDeactivator if supported
			deactivator, hasDeactivator := provider.(integration.ListingDeactivator)
			var nonZeroStock []listingStock
			if hasDeactivator {
				for _, l := range listings {
					if l.StockQty == 0 {
						totalSynced += w.deactivateListing(ctx, tx, ti, deactivator, l)
					} else {
						nonZeroStock = append(nonZeroStock, l)
					}
				}
			} else {
				nonZeroStock = listings
			}

			// Sync non-zero stock listings (and zero-stock if no deactivator)
			bulkProvider, hasBulk := provider.(integration.BulkStockUpdater)
			if hasBulk {
				totalSynced += w.syncBulk(ctx, tx, ti, bulkProvider, nonZeroStock)
			} else {
				totalSynced += w.syncOneByOne(ctx, tx, ti, provider, nonZeroStock)
			}

			return nil
		})
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
	tx pgx.Tx,
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
			for _, l := range batchListings {
				if _, execErr := tx.Exec(ctx,
					`UPDATE product_listings SET sync_status = 'error', error_message = $2, updated_at = NOW() WHERE id = $1`,
					l.ListingID, errMsg,
				); execErr != nil {
					w.logger.Error("stock sync: failed to update listing sync status", "listing_id", l.ListingID, "error", execErr)
				}
			}
			continue
		}

		for _, l := range batchListings {
			if _, execErr := tx.Exec(ctx,
				`UPDATE product_listings SET sync_status = 'synced', error_message = NULL, last_synced_at = NOW(), updated_at = NOW() WHERE id = $1`,
				l.ListingID,
			); execErr != nil {
				w.logger.Error("stock sync: failed to update listing sync status", "listing_id", l.ListingID, "error", execErr)
			}
		}
		w.logger.Info("worker: stock batch synced",
			"operation", "listing.stock_bulk_update",
			"tenant_id", ti.TenantID,
			"batch_size", len(chunk),
		)
		synced += len(chunk)
	}

	return synced
}

// syncOneByOne updates stock one listing at a time (fallback for providers without bulk support).
func (w *StockSyncWorker) syncOneByOne(
	ctx context.Context,
	tx pgx.Tx,
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
			if _, execErr := tx.Exec(ctx,
				`UPDATE product_listings SET sync_status = 'error', error_message = $2, updated_at = NOW() WHERE id = $1`,
				l.ListingID, truncateErrorMessage(err.Error()),
			); execErr != nil {
				w.logger.Error("stock sync: failed to update listing sync status", "listing_id", l.ListingID, "error", execErr)
			}
			continue
		}

		if _, execErr := tx.Exec(ctx,
			`UPDATE product_listings SET sync_status = 'synced', error_message = NULL, last_synced_at = NOW(), updated_at = NOW() WHERE id = $1`,
			l.ListingID,
		); execErr != nil {
			w.logger.Error("stock sync: failed to update listing sync status", "listing_id", l.ListingID, "error", execErr)
		}
		w.logger.Info("worker: stock synced",
			"operation", "listing.stock_update",
			"tenant_id", ti.TenantID,
			"entity_id", l.ListingID,
			"external_id", l.ExternalID,
			"stock_quantity", l.StockQty,
		)
		synced++
	}

	return synced
}

// deactivateListing deactivates a single listing via ListingDeactivator and updates its status.
func (w *StockSyncWorker) deactivateListing(
	ctx context.Context,
	tx pgx.Tx,
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
		if _, execErr := tx.Exec(ctx,
			`UPDATE product_listings SET sync_status = 'error', error_message = $2, updated_at = NOW() WHERE id = $1`,
			l.ListingID, truncateErrorMessage(err.Error()),
		); execErr != nil {
			w.logger.Error("stock sync: failed to update listing sync status", "listing_id", l.ListingID, "error", execErr)
		}
		return 0
	}

	if _, execErr := tx.Exec(ctx,
		`UPDATE product_listings SET status = 'inactive', sync_status = 'synced', error_message = NULL, last_synced_at = NOW(), updated_at = NOW() WHERE id = $1`,
		l.ListingID,
	); execErr != nil {
		w.logger.Error("stock sync: failed to update listing status", "listing_id", l.ListingID, "error", execErr)
	}
	w.logger.Info("worker: listing deactivated (stock=0)",
		"operation", "listing.deactivate",
		"tenant_id", ti.TenantID,
		"entity_id", l.ListingID,
		"external_id", l.ExternalID,
	)
	return 1
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
