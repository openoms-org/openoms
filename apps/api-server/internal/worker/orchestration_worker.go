package worker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/automation"
	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/obsmetrics"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
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
	// metrics is an OPTIONAL, best-effort observability handle (OPE-422). It is
	// nil-receiver safe, so every record call is a no-op when it is not wired and
	// can never affect outbox processing.
	metrics *obsmetrics.FulfillmentMetrics
}

// WithMetrics wires the optional fulfillment metrics collector for additive,
// best-effort outbox observability. Returns the worker for chaining. Safe to omit:
// when unset every metric call is a no-op.
func (w *OrchestrationWorker) WithMetrics(m *obsmetrics.FulfillmentMetrics) *OrchestrationWorker {
	w.metrics = m
	return w
}

// recordOutcome counts one processed outbox event by bounded result
// (processed | failed | claimed). Best-effort, nil-safe.
func (w *OrchestrationWorker) recordOutcome(result string) {
	w.metrics.RecordOutboxEvent(result)
}

// recordClaimed counts n newly claimed outbox events. Best-effort, nil-safe.
func (w *OrchestrationWorker) recordClaimed(n int) {
	for range n {
		w.metrics.RecordOutboxEvent("claimed")
	}
}

// setQueueDepth publishes the current pending outbox depth gauge. Best-effort, nil-safe.
func (w *OrchestrationWorker) setQueueDepth(n int) {
	w.metrics.SetOutboxQueueDepth(n)
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
	w.recordClaimed(len(events))
	for i := range events {
		w.processEvent(ctx, events[i])
	}
	// Publish the remaining queue depth as a best-effort gauge AFTER draining the
	// batch. A count error must never fail the run, so it is logged and ignored.
	if depth, derr := w.repo.CountPending(ctx, w.pool); derr != nil {
		w.logger.Warn("orchestration queue-depth gauge skipped (best-effort)", "error", derr)
	} else {
		w.setQueueDepth(depth)
	}
	return nil
}

// processEvent executes one claimed event. A panic or error in one event must
// not abort the rest of the batch.
func (w *OrchestrationWorker) processEvent(ctx context.Context, e model.OrchestrationOutboxEvent) {
	// Correlation fields shared by every log line for this event. The idempotency key
	// is the stable correlation id across enqueue -> claim -> dispatch -> mark; the
	// process id ties the event to its fulfillment process. These are LOG fields
	// (allowed to be high-cardinality) — they are NEVER used as metric labels, and no
	// secret/PII material is logged (payloads are not included).
	log := w.logger.With(
		"correlation_id", e.IdempotencyKey,
		"event_id", e.ID,
		"event_type", e.EventType,
		"process_id", e.ProcessID,
	)

	var att *model.OrchestrationAttempt
	defer func() {
		if r := recover(); r != nil {
			log.Error("orchestration worker panic", "panic", r)
			sentry.CurrentHub().Recover(r)
			if att != nil {
				_ = w.repo.FinishAttempt(ctx, w.pool, att.ID, model.AttemptStatusFailed, fmt.Sprintf("panic: %v", r))
			}
			_ = w.repo.MarkFailedRetry(ctx, w.pool, e.ID, fmt.Sprintf("panic: %v", r), time.Now().UTC().Add(model.NextOutboxBackoff(e.Attempts+1)))
			w.recordOutcome("failed")
		}
	}()

	attemptNumber := e.Attempts + 1
	var err error
	att, err = w.repo.StartAttempt(ctx, w.pool, e.TenantID, e.ID, attemptNumber)
	if err != nil {
		// Re-queue rather than leaving the row stuck as 'claimed'.
		log.Error("orchestration start attempt failed", "error", err)
		_ = w.repo.MarkFailedRetry(ctx, w.pool, e.ID, fmt.Sprintf("start attempt error: %v", err), time.Now().UTC().Add(model.NextOutboxBackoff(e.Attempts)))
		w.recordOutcome("failed")
		return
	}

	dispErr := w.dispatcher.Dispatch(ctx, e)

	// OPE-421: a DeferUntil sentinel means the dispatch SUCCEEDED but the event must stay
	// pending-callback until a deadline (next_attempt_at). The outbound POST happened, so the
	// attempt is recorded succeeded; the outbox row is re-queued (pending) at the deadline and
	// attempts is incremented so a re-dispatch past the deadline (Attempts>0) is treated as a
	// timeout by the handler. This is NOT a failure outcome.
	var deferErr *service.DeferUntil
	if errors.As(dispErr, &deferErr) {
		_ = w.repo.FinishAttempt(ctx, w.pool, att.ID, model.AttemptStatusSucceeded, "")
		if err := w.repo.MarkFailedRetry(ctx, w.pool, e.ID, "awaiting external workflow callback", deferErr.At); err != nil {
			log.Error("orchestration mark pending-callback failed", "error", err)
		}
		w.recordOutcome("claimed")
		return
	}

	if dispErr == nil {
		_ = w.repo.FinishAttempt(ctx, w.pool, att.ID, model.AttemptStatusSucceeded, "")
		if err := w.repo.MarkSucceeded(ctx, w.pool, e.ID); err != nil {
			log.Error("orchestration mark succeeded failed", "error", err)
		}
		w.recordOutcome("processed")
		return
	}

	_ = w.repo.FinishAttempt(ctx, w.pool, att.ID, model.AttemptStatusFailed, dispErr.Error())
	permanent := model.IsPermanent(dispErr) || attemptNumber >= e.MaxAttempts
	if permanent {
		if err := w.repo.MarkFailedPermanent(ctx, w.pool, e.ID, dispErr.Error()); err != nil {
			log.Error("orchestration mark permanent failed", "error", err)
		}
		w.openBlocker(ctx, e, dispErr)
		log.Warn("orchestration event failed permanently", "error", dispErr)
		w.recordOutcome("failed")
		return
	}
	next := time.Now().UTC().Add(model.NextOutboxBackoff(attemptNumber))
	if err := w.repo.MarkFailedRetry(ctx, w.pool, e.ID, dispErr.Error(), next); err != nil {
		log.Error("orchestration mark retry failed", "error", err)
	}
	w.recordOutcome("failed")
}

// blockerCodeForEvent maps an outbox event type to the fulfillment blocker code
// opened when it fails permanently. Unknown/other event types keep the historical
// default (integration_capability_missing) so existing behaviour is unchanged;
// automation.set_status maps to its own automation_action_failed code (OPE-421) so
// operators can see a failed automation distinctly from an integration gap.
func blockerCodeForEvent(eventType string) string {
	switch eventType {
	case automation.EventAutomationSetStatus:
		return model.BlockerAutomationActionFailed
	case service.EventExternalWorkflow:
		// OPE-421: a permanently failed external-workflow event is a callback timeout.
		return model.BlockerExternalWorkflowTimeout
	default:
		return model.BlockerIntegrationCapabilityMissing
	}
}

// openBlocker records a fulfillment blocker for a permanently failed event, in
// the event's own tenant context.
func (w *OrchestrationWorker) openBlocker(ctx context.Context, e model.OrchestrationOutboxEvent, cause error) {
	code := blockerCodeForEvent(e.EventType)
	err := database.WithTenant(ctx, w.pool, e.TenantID, func(tx pgx.Tx) error {
		_, err := w.fulfillment.CreateBlocker(ctx, tx, model.FulfillmentBlocker{
			TenantID:    e.TenantID,
			ProcessID:   e.ProcessID,
			UnitID:      e.UnitID,
			Code:        code,
			Description: fmt.Sprintf("orchestration event %q failed permanently: %s", e.EventType, cause.Error()),
		})
		return err
	})
	if err != nil {
		w.logger.Error("orchestration open blocker failed", "event_id", e.ID, "error", err)
		return
	}
	// Best-effort: count the blocker by its bounded category (nil-safe).
	w.metrics.RecordBlocker(model.BlockerCategory(code))
}
