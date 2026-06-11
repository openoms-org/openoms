package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// SegmentRefreshWorker recomputes rule-based customer segment membership for all
// tenants on a fixed interval. The recompute is a full clear-then-repopulate, so a
// tick is idempotent: re-running yields identical membership and customer_count.
type SegmentRefreshWorker struct {
	pool           *pgxpool.Pool
	segmentService *service.SegmentService
	logger         *slog.Logger
}

// NewSegmentRefreshWorker creates a new SegmentRefreshWorker.
func NewSegmentRefreshWorker(
	pool *pgxpool.Pool,
	segmentService *service.SegmentService,
	logger *slog.Logger,
) *SegmentRefreshWorker {
	return &SegmentRefreshWorker{
		pool:           pool,
		segmentService: segmentService,
		logger:         logger,
	}
}

// Name returns the unique name of this worker.
func (w *SegmentRefreshWorker) Name() string {
	return "segment_refresh"
}

// Interval returns how often this worker should run.
func (w *SegmentRefreshWorker) Interval() time.Duration {
	return 1 * time.Hour
}

// Run recomputes rule-based segment membership for every tenant.
func (w *SegmentRefreshWorker) Run(ctx context.Context) error {
	// Get all tenant IDs (bypasses RLS — runs on workerPool)
	tenantIDs, err := listAllTenantIDs(ctx, w.pool, w.logger)
	if err != nil {
		return err
	}

	for _, tenantID := range tenantIDs {
		if err := checkWorkerContext(ctx); err != nil {
			return err
		}
		if err := w.segmentService.RefreshRuleBasedSegments(ctx, tenantID); err != nil {
			w.logger.Error("segment refresh worker: refresh rule-based segments",
				"tenant_id", tenantID,
				"error", err,
			)
			continue
		}
	}

	return nil
}
