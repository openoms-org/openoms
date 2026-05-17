package worker

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/automation"
	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

const (
	delayedActionBatchLimit  = 100
	delayedActionMaxAttempts = 5
	delayedActionBaseBackoff = time.Minute
)

type delayedActionFailurePlan struct {
	Retry         bool
	AttemptCount  int
	NextExecuteAt time.Time
}

// DelayedActionWorker polls for pending delayed automation actions and executes them.
type DelayedActionWorker struct {
	pool        *pgxpool.Pool
	delayedRepo repository.DelayedActionRepo
	executor    automation.ActionExecutor
	logger      *slog.Logger
}

// NewDelayedActionWorker creates a new DelayedActionWorker.
func NewDelayedActionWorker(
	pool *pgxpool.Pool,
	delayedRepo repository.DelayedActionRepo,
	executor automation.ActionExecutor,
	logger *slog.Logger,
) *DelayedActionWorker {
	return &DelayedActionWorker{
		pool:        pool,
		delayedRepo: delayedRepo,
		executor:    executor,
		logger:      logger,
	}
}

// Name returns the worker identifier.
func (w *DelayedActionWorker) Name() string {
	return "delayed_action_executor"
}

// Interval returns how frequently the worker should run.
func (w *DelayedActionWorker) Interval() time.Duration {
	return 30 * time.Second
}

// Run executes pending delayed automation actions.
func (w *DelayedActionWorker) Run(ctx context.Context) error {
	// Query pending actions directly (bypassing RLS for cross-tenant)
	var pending []model.DelayedAction
	err := func() error {
		tx, err := w.pool.Begin(ctx)
		if err != nil {
			return err
		}
		defer tx.Rollback(ctx) //nolint:errcheck

		pending, err = w.delayedRepo.ListPending(ctx, tx, delayedActionBatchLimit)
		if err != nil {
			return err
		}
		return tx.Commit(ctx)
	}()
	if err != nil {
		return err
	}

	if len(pending) == 0 {
		return nil
	}

	w.logger.Info("delayed action worker: processing pending actions", "count", len(pending))
	if len(pending) == delayedActionBatchLimit {
		w.logger.Info("delayed action worker: processing full batch", "batch_limit", delayedActionBatchLimit)
	}

	for _, da := range pending {
		if err := checkWorkerContext(ctx); err != nil {
			return err
		}
		w.executeDelayedAction(ctx, da)
	}

	return nil
}

func (w *DelayedActionWorker) executeDelayedAction(ctx context.Context, da model.DelayedAction) {
	// Unmarshal action and event data
	var action automation.Action
	if err := json.Unmarshal(da.ActionData, &action); err != nil {
		w.logger.Error("delayed action worker: unmarshal action data",
			"delayed_action_id", da.ID,
			"error", err,
		)
		errMsg := "failed to unmarshal action data: " + err.Error()
		w.markExecuted(ctx, da.TenantID, da.ID, &errMsg)
		return
	}

	var event automation.Event
	if err := json.Unmarshal(da.EventData, &event); err != nil {
		w.logger.Error("delayed action worker: unmarshal event data",
			"delayed_action_id", da.ID,
			"error", err,
		)
		errMsg := "failed to unmarshal event data: " + err.Error()
		w.markExecuted(ctx, da.TenantID, da.ID, &errMsg)
		return
	}

	// Execute the action
	if err := w.executor.ExecuteAction(ctx, da.TenantID, action, event); err != nil {
		failurePlan := planDelayedActionFailure(da, time.Now())
		if failurePlan.Retry {
			w.logger.Warn("delayed action worker: action execution failed, retry scheduled",
				"delayed_action_id", da.ID,
				"action_type", action.Type,
				"attempt", failurePlan.AttemptCount,
				"max_attempts", delayedActionMaxAttempts,
				"next_execute_at", failurePlan.NextExecuteAt,
				"error", err,
			)
			w.requeueForRetry(ctx, da.TenantID, da.ID, failurePlan.NextExecuteAt, err.Error())
			return
		}

		w.logger.Error("delayed action worker: action execution failed permanently",
			"delayed_action_id", da.ID,
			"action_type", action.Type,
			"attempt", failurePlan.AttemptCount,
			"max_attempts", delayedActionMaxAttempts,
			"error", err,
		)
		msg := err.Error()
		w.markExecuted(ctx, da.TenantID, da.ID, &msg)
		return
	}

	w.logger.Info("delayed action worker: action executed successfully",
		"delayed_action_id", da.ID,
		"action_type", action.Type,
	)
	w.markExecuted(ctx, da.TenantID, da.ID, nil)
}

func planDelayedActionFailure(da model.DelayedAction, now time.Time) delayedActionFailurePlan {
	nextAttempt := max(da.AttemptCount+1, 1)
	if nextAttempt >= delayedActionMaxAttempts {
		return delayedActionFailurePlan{AttemptCount: nextAttempt}
	}

	backoff := delayedActionBaseBackoff
	for i := 1; i < nextAttempt; i++ {
		backoff *= 2
	}
	return delayedActionFailurePlan{
		Retry:         true,
		AttemptCount:  nextAttempt,
		NextExecuteAt: now.Add(backoff),
	}
}

func (w *DelayedActionWorker) markExecuted(ctx context.Context, tenantID uuid.UUID, id uuid.UUID, errMsg *string) {
	err := database.WithTenant(ctx, w.pool, tenantID, func(tx pgx.Tx) error {
		return w.delayedRepo.MarkExecuted(ctx, tx, id, errMsg)
	})
	if err != nil {
		w.logger.Error("delayed action worker: failed to mark action as executed",
			"delayed_action_id", id,
			"error", err,
		)
	}
}

func (w *DelayedActionWorker) requeueForRetry(ctx context.Context, tenantID uuid.UUID, id uuid.UUID, nextExecuteAt time.Time, errMsg string) {
	err := database.WithTenant(ctx, w.pool, tenantID, func(tx pgx.Tx) error {
		return w.delayedRepo.RequeueForRetry(ctx, tx, id, nextExecuteAt, errMsg)
	})
	if err != nil {
		w.logger.Error("delayed action worker: failed to schedule action retry",
			"delayed_action_id", id,
			"next_execute_at", nextExecuteAt,
			"error", err,
		)
	}
}
