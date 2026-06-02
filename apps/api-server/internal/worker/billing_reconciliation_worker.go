package worker

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// reconcileBatchLimit caps how many unreconciled checkout sessions are healed per run.
const reconcileBatchLimit = 50

// checkoutReconciler re-runs the idempotent billing finalization for checkout sessions
// whose finalization partially failed. Implemented by *service.CheckoutService.
type checkoutReconciler interface {
	ReconcileCheckoutClaims(ctx context.Context, scanPool *pgxpool.Pool, limit int) (reconciled int, failed int, err error)
}

// BillingReconciliationWorker periodically heals checkout sessions that registered but
// whose billing customer/subscription records were never created (a partial finalization
// failure leaves a paying tenant without billing state). It re-runs the idempotent
// finalization so the gap self-heals, and surfaces persistent failures for alerting.
type BillingReconciliationWorker struct {
	pool       *pgxpool.Pool
	reconciler checkoutReconciler
	logger     *slog.Logger
}

// NewBillingReconciliationWorker creates a new BillingReconciliationWorker. The pool must
// bypass RLS (the worker pool) because the reconciliation scan spans tenants.
func NewBillingReconciliationWorker(pool *pgxpool.Pool, reconciler checkoutReconciler, logger *slog.Logger) *BillingReconciliationWorker {
	return &BillingReconciliationWorker{pool: pool, reconciler: reconciler, logger: logger}
}

// Name returns the worker identifier.
func (w *BillingReconciliationWorker) Name() string { return "billing_reconciliation" }

// Interval returns how frequently the worker should run.
func (w *BillingReconciliationWorker) Interval() time.Duration { return 15 * time.Minute }

// Run heals checkout sessions whose billing finalization partially failed.
func (w *BillingReconciliationWorker) Run(ctx context.Context) error {
	reconciled, failed, err := w.reconciler.ReconcileCheckoutClaims(ctx, w.pool, reconcileBatchLimit)
	if reconciled > 0 || failed > 0 {
		w.logger.Info("billing reconciliation processed checkout sessions",
			"reconciled", reconciled, "failed", failed)
	}
	if err != nil {
		// Returned to the worker manager, which logs and reports it to Sentry, so a
		// persistent gap (a paying tenant left without billing records) keeps alerting
		// until an operator intervenes.
		return fmt.Errorf("billing reconciliation: %w", err)
	}
	return nil
}
