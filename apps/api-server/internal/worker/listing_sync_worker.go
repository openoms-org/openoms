package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// ListingSyncWorker periodically runs full sync for configs with auto_sync=true.
type ListingSyncWorker struct {
	pool        *pgxpool.Pool
	syncRepo    *repository.ListingSyncRepository
	syncService *service.ListingSyncService
	logger      *slog.Logger
}

// NewListingSyncWorker creates a new ListingSyncWorker.
func NewListingSyncWorker(
	pool *pgxpool.Pool,
	syncRepo *repository.ListingSyncRepository,
	syncService *service.ListingSyncService,
	logger *slog.Logger,
) *ListingSyncWorker {
	return &ListingSyncWorker{
		pool:        pool,
		syncRepo:    syncRepo,
		syncService: syncService,
		logger:      logger,
	}
}

func (w *ListingSyncWorker) Name() string {
	return "listing_sync"
}

func (w *ListingSyncWorker) Interval() time.Duration {
	return 5 * time.Minute
}

func (w *ListingSyncWorker) Run(ctx context.Context) error {
	// Query all auto-sync configs across all tenants (bypasses RLS via workerPool).
	tx, err := w.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	configs, err := w.syncRepo.ListAutoSyncConfigs(ctx, tx)
	if err != nil {
		return err
	}
	_ = tx.Commit(ctx)

	if len(configs) == 0 {
		return nil
	}

	now := time.Now()
	syncedCount := 0

	for _, cfg := range configs {
		// Check if enough time has passed since last sync
		if cfg.LastSyncAt != nil {
			elapsed := now.Sub(*cfg.LastSyncAt)
			interval := time.Duration(cfg.SyncIntervalMinutes) * time.Minute
			if elapsed < interval {
				continue
			}
		}

		w.logger.Info("listing sync worker: running full sync",
			"config_id", cfg.ID,
			"tenant_id", cfg.TenantID,
			"integration_id", cfg.IntegrationID,
			"provider", cfg.IntegrationProvider,
		)

		result, syncErr := w.syncService.RunFullSync(ctx, cfg.TenantID, cfg.ID)
		if syncErr != nil {
			w.logger.Error("listing sync worker: sync failed",
				"config_id", cfg.ID,
				"tenant_id", cfg.TenantID,
				"error", syncErr,
			)

			// Update config with error
			errStr := syncErr.Error()
			updateTx, txErr := w.pool.Begin(ctx)
			if txErr == nil {
				defer updateTx.Rollback(ctx) //nolint:errcheck
				if _, execErr := updateTx.Exec(ctx,
					"SELECT set_config('app.current_tenant_id', $1, true)",
					cfg.TenantID.String(),
				); execErr != nil {
					w.logger.Error("listing sync worker: failed to set tenant context", "config_id", cfg.ID, "error", execErr)
				}
				_ = w.syncRepo.UpdateLastSync(ctx, updateTx, cfg.ID, &errStr)
				_ = updateTx.Commit(ctx)
			}
			continue
		}

		if result.ItemsProcessed > 0 || result.ItemsFailed > 0 {
			w.logger.Info("listing sync worker: completed",
				"config_id", cfg.ID,
				"tenant_id", cfg.TenantID,
				"items_processed", result.ItemsProcessed,
				"items_failed", result.ItemsFailed,
			)
		}
		syncedCount++
	}

	if syncedCount > 0 {
		w.logger.Info("listing sync worker: cycle complete",
			"total_configs", len(configs),
			"synced", syncedCount,
		)
	}
	return nil
}
