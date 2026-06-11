package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// RecurringOrderWorker processes due recurring orders for all tenants.
type RecurringOrderWorker struct {
	pool                  *pgxpool.Pool
	recurringOrderService *service.RecurringOrderService
	logger                *slog.Logger
}

// NewRecurringOrderWorker creates a new RecurringOrderWorker.
func NewRecurringOrderWorker(
	pool *pgxpool.Pool,
	recurringOrderService *service.RecurringOrderService,
	logger *slog.Logger,
) *RecurringOrderWorker {
	return &RecurringOrderWorker{
		pool:                  pool,
		recurringOrderService: recurringOrderService,
		logger:                logger,
	}
}

// Name returns the unique name of this worker.
func (w *RecurringOrderWorker) Name() string {
	return "recurring_order_processor"
}

// Interval returns how often this worker should run.
func (w *RecurringOrderWorker) Interval() time.Duration {
	return 1 * time.Hour
}

// Run processes due recurring orders and creates new order instances.
func (w *RecurringOrderWorker) Run(ctx context.Context) error {
	// Get all tenant IDs (bypasses RLS — runs on workerPool)
	tenantIDs, err := listAllTenantIDs(ctx, w.pool, w.logger)
	if err != nil {
		return err
	}

	totalCreated := 0
	for _, tenantID := range tenantIDs {
		if err := checkWorkerContext(ctx); err != nil {
			return err
		}
		count, err := w.recurringOrderService.ProcessDueOrders(ctx, tenantID)
		if err != nil {
			w.logger.Error("recurring order worker: process due orders",
				"tenant_id", tenantID,
				"error", err,
			)
			continue
		}
		if count > 0 {
			w.logger.Info("recurring order worker: created orders",
				"tenant_id", tenantID,
				"count", count,
			)
		}
		totalCreated += count
	}

	if totalCreated > 0 {
		w.logger.Info("recurring order worker completed",
			"tenants", len(tenantIDs),
			"orders_created", totalCreated,
		)
	}
	return nil
}
