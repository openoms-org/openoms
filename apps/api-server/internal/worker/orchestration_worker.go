package worker

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

// orchestrationBatchLimit caps how many due outbox rows are processed per run.
const orchestrationBatchLimit = 50

// orchestrationDispatcher routes an outbox event to its handler.
// Implemented by *service.OrchestrationDispatcher.
type orchestrationDispatcher interface {
	Dispatch(ctx context.Context, event model.OrchestrationOutboxEvent) error
}

// OrchestrationWorker drains the transactional outbox: it claims due side
// effects across tenants (privileged pool, SKIP LOCKED), executes each through
// the dispatcher with retry/backoff, and opens a fulfillment blocker on
// permanent or exhausted failure.
type OrchestrationWorker struct {
	pool        *pgxpool.Pool
	repo        *repository.OrchestrationRepository
	dispatcher  orchestrationDispatcher
	fulfillment *repository.FulfillmentRepository
	logger      *slog.Logger
	interval    time.Duration
	batchLimit  int
}

// NewOrchestrationWorker creates the worker. pool MUST be the privileged worker
// pool (cross-tenant claims span all tenants).
func NewOrchestrationWorker(pool *pgxpool.Pool, repo *repository.OrchestrationRepository, dispatcher orchestrationDispatcher, fulfillment *repository.FulfillmentRepository, interval time.Duration, logger *slog.Logger) *OrchestrationWorker {
	if interval <= 0 {
		interval = 15 * time.Second
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &OrchestrationWorker{
		pool: pool, repo: repo, dispatcher: dispatcher, fulfillment: fulfillment,
		logger: logger, interval: interval, batchLimit: orchestrationBatchLimit,
	}
}

// Name returns the worker identifier.
func (w *OrchestrationWorker) Name() string { return "orchestration" }

// Interval returns how frequently the worker drains the outbox.
func (w *OrchestrationWorker) Interval() time.Duration { return w.interval }

// Run claims and processes one batch of due outbox events.
func (w *OrchestrationWorker) Run(ctx context.Context) error {
	events, err := w.repo.ClaimDue(ctx, w.pool, w.batchLimit)
	if err != nil {
		return fmt.Errorf("claim due outbox events: %w", err)
	}
	for i := range events {
		w.processEvent(ctx, events[i])
	}
	return nil
}

// processEvent executes one claimed event. A panic or error in one event must
// not abort the rest of the batch.
func (w *OrchestrationWorker) processEvent(ctx context.Context, e model.OrchestrationOutboxEvent) {
	defer func() {
		if r := recover(); r != nil {
			w.logger.Error("orchestration worker panic", "event_id", e.ID, "event_type", e.EventType, "panic", r)
			sentry.CurrentHub().Recover(r)
			_ = w.repo.MarkFailedRetry(ctx, w.pool, e.ID, fmt.Sprintf("panic: %v", r), time.Now().UTC().Add(model.NextOutboxBackoff(e.Attempts+1)))
		}
	}()

	attemptNumber := e.Attempts + 1
	att, err := w.repo.StartAttempt(ctx, w.pool, e.TenantID, e.ID, attemptNumber)
	if err != nil {
		w.logger.Error("orchestration start attempt failed", "event_id", e.ID, "error", err)
		return
	}

	dispErr := w.dispatcher.Dispatch(ctx, e)
	if dispErr == nil {
		_ = w.repo.FinishAttempt(ctx, w.pool, att.ID, model.AttemptStatusSucceeded, "")
		if err := w.repo.MarkSucceeded(ctx, w.pool, e.ID); err != nil {
			w.logger.Error("orchestration mark succeeded failed", "event_id", e.ID, "error", err)
		}
		return
	}

	_ = w.repo.FinishAttempt(ctx, w.pool, att.ID, model.AttemptStatusFailed, dispErr.Error())
	permanent := model.IsPermanent(dispErr) || attemptNumber >= e.MaxAttempts
	if permanent {
		if err := w.repo.MarkFailedPermanent(ctx, w.pool, e.ID, dispErr.Error()); err != nil {
			w.logger.Error("orchestration mark permanent failed", "event_id", e.ID, "error", err)
		}
		w.openBlocker(ctx, e, dispErr)
		w.logger.Warn("orchestration event failed permanently", "event_id", e.ID, "event_type", e.EventType, "error", dispErr)
		return
	}
	next := time.Now().UTC().Add(model.NextOutboxBackoff(attemptNumber))
	if err := w.repo.MarkFailedRetry(ctx, w.pool, e.ID, dispErr.Error(), next); err != nil {
		w.logger.Error("orchestration mark retry failed", "event_id", e.ID, "error", err)
	}
}

// openBlocker records a fulfillment blocker for a permanently failed event, in
// the event's own tenant context.
func (w *OrchestrationWorker) openBlocker(ctx context.Context, e model.OrchestrationOutboxEvent, cause error) {
	err := database.WithTenant(ctx, w.pool, e.TenantID, func(tx pgx.Tx) error {
		_, err := w.fulfillment.CreateBlocker(ctx, tx, model.FulfillmentBlocker{
			TenantID:    e.TenantID,
			ProcessID:   e.ProcessID,
			UnitID:      e.UnitID,
			Code:        model.BlockerIntegrationCapabilityMissing,
			Description: fmt.Sprintf("orchestration event %q failed permanently: %s", e.EventType, cause.Error()),
		})
		return err
	})
	if err != nil {
		w.logger.Error("orchestration open blocker failed", "event_id", e.ID, "error", err)
	}
}
