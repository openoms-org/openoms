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

const priceBulkBatchSize = 100

// listingPrice holds the data needed to sync price for a single product listing.
type listingPrice struct {
	ListingID  string
	ExternalID string
	Price      float64
}

type PriceSyncWorker struct {
	pool          *pgxpool.Pool
	encryptionKey []byte
	logger        *slog.Logger
}

func NewPriceSyncWorker(pool *pgxpool.Pool, encryptionKey []byte, logger *slog.Logger) *PriceSyncWorker {
	return &PriceSyncWorker{
		pool:          pool,
		encryptionKey: encryptionKey,
		logger:        logger,
	}
}

func (w *PriceSyncWorker) Name() string {
	return "price_sync"
}

func (w *PriceSyncWorker) Interval() time.Duration {
	return 5 * time.Minute
}

func (w *PriceSyncWorker) Run(ctx context.Context) error {
	// Get all active marketplace integrations (all providers)
	tis, err := ListAllActiveMarketplaceIntegrations(ctx, w.pool)
	if err != nil {
		return err
	}

	totalSynced := 0

	for _, ti := range tis {
		credJSON, err := crypto.Decrypt(ti.Credentials, w.encryptionKey)
		if err != nil {
			w.logger.Error("price sync: failed to decrypt credentials", "integration_id", ti.IntegrationID, "error", err)
			continue
		}

		provider, err := integration.NewMarketplaceProvider(ti.Provider, credJSON, ti.Settings)
		if err != nil {
			w.logger.Error("price sync: failed to create provider", "integration_id", ti.IntegrationID, "error", err)
			continue
		}

		tenantErr := database.WithTenant(ctx, w.pool, ti.TenantID, func(tx pgx.Tx) error {
			// Query auto-sync product_listings with price data.
			// stock_sync_mode = 'auto' gates both stock and price sync (single toggle per listing).
			rows, err := tx.Query(ctx,
				`SELECT pl.id, pl.external_id, COALESCE(pl.price_override, p.price) AS sync_price
				 FROM product_listings pl
				 JOIN products p ON p.id = pl.product_id
				 WHERE pl.integration_id = $1 AND pl.status = 'active'
				   AND pl.external_id IS NOT NULL AND pl.stock_sync_mode = 'auto'`,
				ti.IntegrationID,
			)
			if err != nil {
				return err
			}
			defer rows.Close()

			// Collect all listings into a slice
			var listings []listingPrice
			for rows.Next() {
				var listingID, externalID string
				var price float64
				if err := rows.Scan(&listingID, &externalID, &price); err != nil {
					w.logger.Error("price sync: scan listing", "error", err)
					continue
				}

				// Skip rows where price <= 0
				if price <= 0 {
					continue
				}

				listings = append(listings, listingPrice{
					ListingID:  listingID,
					ExternalID: externalID,
					Price:      price,
				})
			}
			if err := rows.Err(); err != nil {
				return err
			}

			// Try bulk path if provider supports it
			bulkProvider, hasBulk := provider.(integration.BulkPriceUpdater)
			if hasBulk {
				totalSynced += w.syncBulk(ctx, tx, ti, bulkProvider, listings)
			} else {
				totalSynced += w.syncOneByOne(ctx, tx, ti, provider, listings)
			}

			return nil
		})
		closeProvider(provider)
		if tenantErr != nil {
			w.logger.Error("price sync: tenant error", "tenant_id", ti.TenantID, "error", tenantErr)
			continue
		}
	}

	w.logger.Info("price sync completed", "tenants", len(tis), "synced", totalSynced)
	return nil
}

// syncBulk sends price updates in batches of priceBulkBatchSize.
func (w *PriceSyncWorker) syncBulk(
	ctx context.Context,
	tx pgx.Tx,
	ti TenantIntegration,
	provider integration.BulkPriceUpdater,
	listings []listingPrice,
) int {
	synced := 0

	// Build PriceUpdate slice for chunking.
	// Currency hardcoded to PLN — all supported marketplaces (Allegro) operate in PLN.
	updates := make([]integration.PriceUpdate, len(listings))
	for i, l := range listings {
		updates[i] = integration.PriceUpdate{
			ExternalOfferID: l.ExternalID,
			Amount:          l.Price,
			Currency:        "PLN",
		}
	}

	chunks := chunkPriceUpdates(updates, priceBulkBatchSize)

	// Process each chunk; the listing slice is indexed in parallel with the update slice
	offset := 0
	for _, chunk := range chunks {
		batchListings := listings[offset : offset+len(chunk)]
		offset += len(chunk)

		if err := provider.BulkUpdatePrice(ctx, chunk); err != nil {
			w.logger.Error("price sync: bulk update failed",
				"operation", "listing.price_bulk_update",
				"tenant_id", ti.TenantID,
				"batch_size", len(chunk),
				"error", err,
			)
			errMsg := truncateErrorMessage(err.Error(), 500)
			for _, l := range batchListings {
				_, _ = tx.Exec(ctx,
					`UPDATE product_listings SET sync_status = 'error', error_message = $2, updated_at = NOW() WHERE id = $1`,
					l.ListingID, errMsg,
				)
			}
			continue
		}

		for _, l := range batchListings {
			_, _ = tx.Exec(ctx,
				`UPDATE product_listings SET sync_status = 'synced', error_message = NULL, last_synced_at = NOW(), updated_at = NOW() WHERE id = $1`,
				l.ListingID,
			)
		}
		w.logger.Info("worker: price batch synced",
			"operation", "listing.price_bulk_update",
			"tenant_id", ti.TenantID,
			"batch_size", len(chunk),
		)
		synced += len(chunk)
	}

	return synced
}

// syncOneByOne updates price one listing at a time (fallback for providers without bulk support).
func (w *PriceSyncWorker) syncOneByOne(
	ctx context.Context,
	tx pgx.Tx,
	ti TenantIntegration,
	provider integration.MarketplaceProvider,
	listings []listingPrice,
) int {
	synced := 0

	for _, l := range listings {
		if err := provider.UpdatePrice(ctx, l.ExternalID, l.Price); err != nil {
			w.logger.Error("price sync: update price failed",
				"operation", "listing.price_update",
				"tenant_id", ti.TenantID,
				"entity_id", l.ListingID,
				"external_id", l.ExternalID,
				"error", err,
			)
			_, _ = tx.Exec(ctx,
				`UPDATE product_listings SET sync_status = 'error', error_message = $2, updated_at = NOW() WHERE id = $1`,
				l.ListingID, truncateErrorMessage(err.Error(), 500),
			)
			continue
		}

		_, _ = tx.Exec(ctx,
			`UPDATE product_listings SET sync_status = 'synced', error_message = NULL, last_synced_at = NOW(), updated_at = NOW() WHERE id = $1`,
			l.ListingID,
		)
		w.logger.Info("worker: price synced",
			"operation", "listing.price_update",
			"tenant_id", ti.TenantID,
			"entity_id", l.ListingID,
			"external_id", l.ExternalID,
			"price", l.Price,
		)
		synced++
	}

	return synced
}

// chunkPriceUpdates splits a slice of PriceUpdate into chunks of the given size.
func chunkPriceUpdates(items []integration.PriceUpdate, size int) [][]integration.PriceUpdate {
	if len(items) == 0 {
		return nil
	}
	var chunks [][]integration.PriceUpdate
	for i := 0; i < len(items); i += size {
		end := min(i+size, len(items))
		chunks = append(chunks, items[i:end])
	}
	return chunks
}
