package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// fulfillmentGaugeSweepInterval is how often the global stuck/blocked process gauges
// are recomputed. The gauges are an at-a-glance ops signal, not a precise real-time
// metric, so a slow cadence keeps the cross-tenant aggregate query well clear of the
// hot path.
const fulfillmentGaugeSweepInterval = 1 * time.Minute

// FulfillmentGaugeSweeper periodically publishes the GLOBAL (cross-tenant) stuck and
// blocked fulfillment-process gauges (OPE-422 followup). It exists because those gauges
// cannot be set correctly from the per-tenant read path — a per-tenant value with no
// tenant label flaps to whichever tenant last read its summary. Instead this sweeper
// counts processes across ALL tenants on the privileged worker pool (which bypasses RLS)
// once per interval and sets a single, stable, label-free aggregate.
//
// Gated: it is only registered when fulfillment recording is enabled (there are
// processes to count); when disabled it is not registered at all. Read-only +
// best-effort — a sweep error is logged and the next tick retries.
type FulfillmentGaugeSweeper struct {
	pool   *pgxpool.Pool
	svc    *service.FulfillmentReadService
	logger *slog.Logger
}

// NewFulfillmentGaugeSweeper creates the sweeper. pool MUST be the privileged worker
// pool: SweepGlobalProcessGauges counts every tenant's processes and relies on that pool
// bypassing RLS (the RLS-scoped app pool would return zero).
func NewFulfillmentGaugeSweeper(pool *pgxpool.Pool, svc *service.FulfillmentReadService, logger *slog.Logger) *FulfillmentGaugeSweeper {
	if logger == nil {
		logger = slog.Default()
	}
	return &FulfillmentGaugeSweeper{pool: pool, svc: svc, logger: logger}
}

// Name returns the worker identifier.
func (w *FulfillmentGaugeSweeper) Name() string { return "fulfillment_gauge_sweeper" }

// Interval returns how frequently the gauges are recomputed.
func (w *FulfillmentGaugeSweeper) Interval() time.Duration { return fulfillmentGaugeSweepInterval }

// Run recomputes the global stuck/blocked process gauges once. Best-effort: on error it
// logs and returns nil so the worker manager keeps scheduling the next tick.
func (w *FulfillmentGaugeSweeper) Run(ctx context.Context) error {
	if w.svc == nil {
		return nil
	}
	if err := w.svc.SweepGlobalProcessGauges(ctx, w.pool); err != nil {
		w.logger.Error("fulfillment gauge sweep failed (best-effort)", "error", err)
	}
	return nil
}
