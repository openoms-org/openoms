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

// DelayedActionWorker polls for pending delayed automation actions and executes them.
type DelayedActionWorker struct {
	pool        *pgxpool.Pool
	delayedRepo repository.DelayedActionRepo
	executor    automation.ActionExecutor
	logger      *slog.Logger
}

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

func (w *DelayedActionWorker) Name() string {
	return "delayed_action_executor"
}

func (w *DelayedActionWorker) Interval() time.Duration {
	return 30 * time.Second
}

func (w *DelayedActionWorker) Run(ctx context.Context) error {
	// Query pending actions directly (bypassing RLS for cross-tenant)
	var pending []model.DelayedAction
	err := func() error {
		tx, err := w.pool.Begin(ctx)
		if err != nil {
			return err
		}
		defer tx.Rollback(ctx) //nolint:errcheck

		pending, err = w.delayedRepo.ListPending(ctx, tx)
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

	for _, da := range pending {
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
	var errMsg *string
	if err := w.executor.ExecuteAction(ctx, da.TenantID, action, event); err != nil {
		w.logger.Error("delayed action worker: action execution failed",
			"delayed_action_id", da.ID,
			"action_type", action.Type,
			"error", err,
		)
		msg := err.Error()
		errMsg = &msg
	} else {
		w.logger.Info("delayed action worker: action executed successfully",
			"delayed_action_id", da.ID,
			"action_type", action.Type,
		)
	}

	w.markExecuted(ctx, da.TenantID, da.ID, errMsg)
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
