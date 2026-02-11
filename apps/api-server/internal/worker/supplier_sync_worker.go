package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

type SupplierSyncWorker struct {
	pool            *pgxpool.Pool
	supplierService *service.SupplierService
	logger          *slog.Logger
}

func NewSupplierSyncWorker(pool *pgxpool.Pool, supplierService *service.SupplierService, logger *slog.Logger) *SupplierSyncWorker {
	return &SupplierSyncWorker{
		pool:            pool,
		supplierService: supplierService,
		logger:          logger,
	}
}

func (w *SupplierSyncWorker) Name() string {
	return "supplier-sync"
}

func (w *SupplierSyncWorker) Interval() time.Duration {
	return 1 * time.Hour
}

func (w *SupplierSyncWorker) Run(ctx context.Context) error {
	// Query all active suppliers with feed URLs across all tenants (bypasses RLS)
	rows, err := w.pool.Query(ctx,
		`SELECT id, tenant_id FROM suppliers
		 WHERE status = 'active' AND feed_url IS NOT NULL AND feed_url != ''`,
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
		supplierID := ref.ID
		tenantID := ref.TenantID

		if err := w.supplierService.SyncFeed(ctx, tenantID, supplierID); err != nil {
			w.logger.Error("supplier sync failed",
				"supplier_id", supplierID, "tenant_id", tenantID, "error", err)

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
		synced++
	}

	w.logger.Info("supplier sync worker completed",
		"total", len(refs), "synced", synced)
	return nil
}

// ListActiveSuppliers queries all active suppliers across tenants for the worker.
// This bypasses RLS.
func ListActiveSuppliers(ctx context.Context, pool *pgxpool.Pool) ([]model.Supplier, error) {
	rows, err := pool.Query(ctx,
		`SELECT id, tenant_id, name, code, feed_url, feed_format, status, settings,
		        last_sync_at, error_message, created_at, updated_at
		 FROM suppliers WHERE status = 'active' AND feed_url IS NOT NULL AND feed_url != ''`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var suppliers []model.Supplier
	for rows.Next() {
		var s model.Supplier
		if err := rows.Scan(
			&s.ID, &s.TenantID, &s.Name, &s.Code, &s.FeedURL, &s.FeedFormat,
			&s.Status, &s.Settings, &s.LastSyncAt, &s.ErrorMessage,
			&s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, err
		}
		suppliers = append(suppliers, s)
	}
	return suppliers, rows.Err()
}
