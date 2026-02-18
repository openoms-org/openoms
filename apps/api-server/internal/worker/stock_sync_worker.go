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

		if err := database.WithTenant(ctx, w.pool, ti.TenantID, func(tx pgx.Tx) error {
			// Query auto-sync product_listings with warehouse-based available stock
			rows, err := tx.Query(ctx,
				`SELECT pl.id, pl.external_id, pl.stock_override,
				        GREATEST(COALESCE(SUM(ws.quantity), 0) - COALESCE(SUM(ws.reserved), 0), 0) AS available_qty
				 FROM product_listings pl
				 LEFT JOIN warehouse_stock ws ON ws.product_id = pl.product_id AND ws.variant_id IS NULL
				 WHERE pl.integration_id = $1 AND pl.status = 'active'
				   AND pl.external_id IS NOT NULL AND pl.stock_sync_mode = 'auto'
				 GROUP BY pl.id, pl.external_id, pl.stock_override`,
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

				// Use stock_override if set
				stockQty := availableQty
				if stockOverride != nil {
					stockQty = *stockOverride
				}

				if err := provider.UpdateStock(ctx, externalID, stockQty); err != nil {
					w.logger.Error("stock sync: update stock failed",
						"operation", "listing.stock_update",
						"tenant_id", ti.TenantID,
						"entity_id", listingID,
						"external_id", externalID,
						"error", err,
					)
					// Update listing sync status to error
					_, _ = tx.Exec(ctx,
						`UPDATE product_listings SET sync_status = 'error', error_message = $2, updated_at = NOW() WHERE id = $1`,
						listingID, err.Error(),
					)
					continue
				}

				// Update listing sync status
				_, _ = tx.Exec(ctx,
					`UPDATE product_listings SET sync_status = 'synced', error_message = NULL, last_synced_at = NOW(), updated_at = NOW() WHERE id = $1`,
					listingID,
				)
				w.logger.Info("worker: stock synced",
					"operation", "listing.stock_update",
					"tenant_id", ti.TenantID,
					"entity_id", listingID,
					"external_id", externalID,
					"stock_quantity", stockQty,
				)
				totalSynced++
			}
			return rows.Err()
		}); err != nil {
			w.logger.Error("stock sync: tenant error", "tenant_id", ti.TenantID, "error", err)
			continue
		}
	}

	w.logger.Info("stock sync completed", "tenants", len(tis), "synced", totalSynced)
	return nil
}
