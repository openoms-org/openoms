package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// KSeFStatusWorker periodically checks the status of pending KSeF invoices.
type KSeFStatusWorker struct {
	pool        *pgxpool.Pool
	ksefService *service.KSeFService
	logger      *slog.Logger
}

// NewKSeFStatusWorker creates a new KSeF status worker.
func NewKSeFStatusWorker(pool *pgxpool.Pool, ksefService *service.KSeFService, logger *slog.Logger) *KSeFStatusWorker {
	return &KSeFStatusWorker{
		pool:        pool,
		ksefService: ksefService,
		logger:      logger,
	}
}

// Name returns the worker identifier.
func (w *KSeFStatusWorker) Name() string {
	return "ksef_status_checker"
}

// Interval returns how frequently the worker should run.
func (w *KSeFStatusWorker) Interval() time.Duration {
	return 5 * time.Minute
}

// Run polls the KSeF API for invoice submission status updates.
func (w *KSeFStatusWorker) Run(ctx context.Context) error {
	// Get all tenant IDs
	rows, err := w.pool.Query(ctx, "SELECT id FROM tenants")
	if err != nil {
		return err
	}
	defer rows.Close()

	var tenantIDs []uuid.UUID
	for rows.Next() {
		if err := checkWorkerContext(ctx); err != nil {
			return err
		}
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			w.logger.Error("ksef worker: scan tenant", "error", err)
			continue
		}
		tenantIDs = append(tenantIDs, id)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	totalSynced := 0
	totalRetried := 0
	for _, tenantID := range tenantIDs {
		if err := checkWorkerContext(ctx); err != nil {
			return err
		}
		synced, err := w.ksefService.SyncPendingStatuses(ctx, tenantID)
		if err != nil {
			w.logger.Error("ksef worker: sync statuses", "tenant_id", tenantID, "error", err)
			continue
		}
		totalSynced += synced

		retried, err := w.ksefService.RetryErroredInvoices(ctx, tenantID)
		if err != nil {
			w.logger.Error("ksef worker: retry errored", "tenant_id", tenantID, "error", err)
			continue
		}
		totalRetried += retried
	}

	if totalSynced > 0 || totalRetried > 0 {
		w.logger.Info("ksef worker completed", "tenants", len(tenantIDs), "synced", totalSynced, "retried", totalRetried)
	}
	return nil
}
