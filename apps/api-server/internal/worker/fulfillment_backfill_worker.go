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

// fulfillmentBackfillInterval is how often the backfill worker drains another
// batch of still-missing orders per tenant. It is deliberately slow: backfill is a
// one-time migration that converges over a handful of passes, so there is no need to
// run it aggressively, and a slow cadence keeps it well clear of the hot path.
const fulfillmentBackfillInterval = 5 * time.Minute

// fulfillmentBackfillBatchSize bounds how many process-less orders are backfilled
// per tenant per run. Each run's per-tenant work commits in a single transaction, so
// this also bounds that transaction's size.
const fulfillmentBackfillBatchSize = 200

// FulfillmentBackfillWorker backfills fulfillment processes for existing
// non-terminal orders that lack one (OPE-423a), tenant by tenant. It is DOUBLE
// gated:
//   - enabled (FULFILLMENT_BACKFILL_ENABLED) — when false the worker is a complete
//     no-op and nothing is ever read or written; this is the default.
//   - dryRun (FULFILLMENT_BACKFILL_DRY_RUN, default true) — even when enabled, the
//     worker only COUNTS what it would backfill unless dryRun is explicitly false.
//
// Writing therefore requires BOTH enabled=true AND dryRun=false. The work is
// resumable + idempotent: each run drains up to a batch of the still-missing set per
// tenant, so re-runs finish the rest and an already-backfilled tenant is a no-op.
type FulfillmentBackfillWorker struct {
	pool    *pgxpool.Pool
	svc     *service.FulfillmentBackfillService
	enabled bool
	dryRun  bool
	logger  *slog.Logger
}

// NewFulfillmentBackfillWorker creates the worker. pool MUST be the privileged
// worker pool: the tenant list is read cross-tenant (bypassing RLS), then each
// tenant's backfill runs inside database.WithTenant on that same pool so RLS applies
// to the per-tenant work.
func NewFulfillmentBackfillWorker(pool *pgxpool.Pool, svc *service.FulfillmentBackfillService, enabled, dryRun bool, logger *slog.Logger) *FulfillmentBackfillWorker {
	if logger == nil {
		logger = slog.Default()
	}
	return &FulfillmentBackfillWorker{
		pool:    pool,
		svc:     svc,
		enabled: enabled,
		dryRun:  dryRun,
		logger:  logger,
	}
}

// Name returns the worker identifier.
func (w *FulfillmentBackfillWorker) Name() string { return "fulfillment_backfill" }

// Interval returns how frequently the worker drains another backfill batch.
func (w *FulfillmentBackfillWorker) Interval() time.Duration { return fulfillmentBackfillInterval }

// Run backfills one batch of process-less non-terminal orders for every tenant. It
// is a complete no-op when the worker is disabled. Per-tenant failures are logged
// and isolated so one tenant never blocks the rest.
func (w *FulfillmentBackfillWorker) Run(ctx context.Context) error {
	if !w.enabled {
		return nil
	}

	// Tenant list read cross-tenant on the privileged pool (bypasses RLS), mirroring
	// the other cross-tenant workers.
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
			w.logger.Error("fulfillment backfill: scan tenant", "error", err)
			continue
		}
		tenantIDs = append(tenantIDs, id)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	opts := service.BackfillOptions{DryRun: w.dryRun, BatchSize: fulfillmentBackfillBatchSize}
	totalCreated, totalNeeding, totalRemaining := 0, 0, 0
	for _, tenantID := range tenantIDs {
		if err := checkWorkerContext(ctx); err != nil {
			return err
		}
		var report service.BackfillReport
		err := database.WithTenant(ctx, w.pool, tenantID, func(tx pgx.Tx) error {
			r, e := w.svc.BackfillActiveOrderProcesses(ctx, tx, tenantID, opts)
			report = r
			return e
		})
		if err != nil {
			w.logger.Error("fulfillment backfill: tenant pass failed",
				"tenant_id", tenantID, "dry_run", w.dryRun, "error", err)
			continue
		}
		totalCreated += report.Created
		totalNeeding += report.NeedingProcess
		totalRemaining += report.NeedingProcess - report.Created
		if report.NeedingProcess > 0 || report.Created > 0 || report.Errors > 0 {
			w.logger.Info("fulfillment backfill: tenant pass",
				"tenant_id", tenantID,
				"dry_run", w.dryRun,
				"scanned", report.Scanned,
				"needing_process", report.NeedingProcess,
				"created", report.Created,
				"skipped", report.Skipped,
				"errors", report.Errors,
			)
		}
	}

	if totalNeeding > 0 || totalCreated > 0 {
		w.logger.Info("fulfillment backfill: run complete",
			"tenants", len(tenantIDs),
			"dry_run", w.dryRun,
			"needing_process", totalNeeding,
			"created", totalCreated,
			"remaining_after_run", totalRemaining,
		)
	}
	return nil
}
