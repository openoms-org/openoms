package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// SupplierSyncWorker periodically syncs product data from supplier feeds.
type SupplierSyncWorker struct {
	pool            *pgxpool.Pool
	supplierService *service.SupplierService
	logger          *slog.Logger
}

// NewSupplierSyncWorker creates a new SupplierSyncWorker.
func NewSupplierSyncWorker(pool *pgxpool.Pool, supplierService *service.SupplierService, logger *slog.Logger) *SupplierSyncWorker {
	return &SupplierSyncWorker{
		pool:            pool,
		supplierService: supplierService,
		logger:          logger,
	}
}

// Name returns the unique name of this worker.
func (w *SupplierSyncWorker) Name() string {
	return "supplier-sync"
}

// Interval returns how often this worker should run.
func (w *SupplierSyncWorker) Interval() time.Duration {
	return 1 * time.Minute
}

// Run syncs product data from all active supplier feeds.
func (w *SupplierSyncWorker) Run(ctx context.Context) error {
	// Query active suppliers whose sync interval has elapsed (bypasses RLS).
	// Picks up both feed-URL based suppliers (IOF) and integration-linked suppliers (BTP etc.)
	rows, err := w.pool.Query(ctx,
		`SELECT id, tenant_id FROM suppliers
		 WHERE status = 'active'
		   AND ((feed_url IS NOT NULL AND feed_url != '') OR integration_id IS NOT NULL)
		   AND (last_sync_at IS NULL OR last_sync_at + (sync_interval_minutes || ' minutes')::interval < NOW())
		 ORDER BY last_sync_at ASC NULLS FIRST
		 LIMIT 100`,
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	type supplierRef struct {
		ID       uuid.UUID
		TenantID uuid.UUID
	}
	var refs []supplierRef
	for rows.Next() {
		if err := checkWorkerContext(ctx); err != nil {
			return err
		}
		var ref supplierRef
		if err := rows.Scan(&ref.ID, &ref.TenantID); err != nil {
			return err
		}
		refs = append(refs, ref)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	synced := 0
	for _, ref := range refs {
		if err := checkWorkerContext(ctx); err != nil {
			return err
		}
		supplierID := ref.ID
		tenantID := ref.TenantID

		if err := w.supplierService.SyncFeed(ctx, tenantID, supplierID); err != nil {
			w.logger.Error("worker: supplier sync failed",
				"operation", "supplier.sync_feed",
				"tenant_id", tenantID,
				"entity_id", supplierID,
				"error", err,
			)

			// Record error on supplier
			errMsg := err.Error()
			if dbErr := database.WithTenant(ctx, w.pool, tenantID, func(tx pgx.Tx) error {
				_, err := tx.Exec(ctx,
					`UPDATE suppliers SET error_message = $1, updated_at = NOW() WHERE id = $2`,
					errMsg, supplierID,
				)
				return err
			}); dbErr != nil {
				w.logger.Error("failed to record supplier sync error", "supplier_id", supplierID, "error", dbErr)
			}
			continue
		}
		w.logger.Info("worker: supplier synced",
			"operation", "supplier.sync_feed",
			"tenant_id", tenantID,
			"entity_id", supplierID,
		)
		synced++
	}

	w.logger.Info("supplier sync worker completed",
		"total", len(refs), "synced", synced)
	return nil
}
