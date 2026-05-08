package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// RepricingWorker runs repricing rules for all tenants on a schedule.
type RepricingWorker struct {
	pool             *pgxpool.Pool
	repricingService *service.RepricingService
	logger           *slog.Logger
}

// NewRepricingWorker creates a new RepricingWorker.
func NewRepricingWorker(
	pool *pgxpool.Pool,
	repricingService *service.RepricingService,
	logger *slog.Logger,
) *RepricingWorker {
	return &RepricingWorker{
		pool:             pool,
		repricingService: repricingService,
		logger:           logger,
	}
}

// Name returns the unique name of this worker.
func (w *RepricingWorker) Name() string {
	return "repricing_engine"
}

// Interval returns how often this worker should run.
func (w *RepricingWorker) Interval() time.Duration {
	return 15 * time.Minute
}

// Run applies repricing rules across all active tenants.
func (w *RepricingWorker) Run(ctx context.Context) error {
	// Get all tenant IDs (bypasses RLS — runs on workerPool)
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
			w.logger.Error("repricing worker: scan tenant", "error", err)
			continue
		}
		tenantIDs = append(tenantIDs, id)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	totalChanges := 0
	for _, tenantID := range tenantIDs {
		if err := checkWorkerContext(ctx); err != nil {
			return err
		}
		result, err := w.repricingService.ApplyRules(ctx, tenantID)
		if err != nil {
			w.logger.Error("repricing worker: apply rules",
				"tenant_id", tenantID,
				"error", err,
			)
			continue
		}
		if result.PriceChanges > 0 {
			w.logger.Info("repricing worker: applied",
				"tenant_id", tenantID,
				"rules_evaluated", result.RulesEvaluated,
				"products_affected", result.ProductsAffected,
				"price_changes", result.PriceChanges,
			)
		}
		totalChanges += result.PriceChanges
	}

	if totalChanges > 0 {
		w.logger.Info("repricing worker completed",
			"tenants", len(tenantIDs),
			"total_price_changes", totalChanges,
		)
	}
	return nil
}
