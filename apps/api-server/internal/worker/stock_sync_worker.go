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
	tis, err := ListActiveIntegrations(ctx, w.pool, "allegro")
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
			// Query product_listings with active status that have external_id
			rows, err := tx.Query(ctx,
				`SELECT pl.id, pl.external_id, p.stock_quantity
				 FROM product_listings pl
				 JOIN products p ON p.id = pl.product_id
				 WHERE pl.integration_id = $1 AND pl.status = 'active' AND pl.external_id IS NOT NULL`,
				ti.IntegrationID,
			)
			if err != nil {
				return err
			}
			defer rows.Close()

			for rows.Next() {
				var listingID, externalID string
				var stockQty int
				if err := rows.Scan(&listingID, &externalID, &stockQty); err != nil {
					w.logger.Error("stock sync: scan listing", "error", err)
					continue
				}

				if err := provider.UpdateStock(ctx, externalID, stockQty); err != nil {
					w.logger.Error("stock sync: update stock failed", "external_id", externalID, "error", err)
					// Update listing sync status to error
					_, _ = tx.Exec(ctx,
						`UPDATE product_listings SET sync_status = 'error', updated_at = NOW() WHERE id = $1`,
						listingID,
					)
					continue
				}

				// Update listing sync status
				_, _ = tx.Exec(ctx,
					`UPDATE product_listings SET sync_status = 'synced', last_synced_at = NOW(), updated_at = NOW() WHERE id = $1`,
					listingID,
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
