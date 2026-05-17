package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

const defaultDelayedActionPendingLimit = 100

// DelayedActionRepository implements persistence for delayed automation actions.
type DelayedActionRepository struct{}

// NewDelayedActionRepository creates a new DelayedActionRepository.
func NewDelayedActionRepository() *DelayedActionRepository {
	return &DelayedActionRepository{}
}

// Create inserts a new delayed action.
func (r *DelayedActionRepository) Create(ctx context.Context, tx pgx.Tx, da *model.DelayedAction) error {
	return tx.QueryRow(ctx,
		`INSERT INTO automation_delayed_actions (
			id, tenant_id, rule_id, action_index, order_id, execute_at,
			action_data, event_data
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING created_at`,
		da.ID, da.TenantID, da.RuleID, da.ActionIndex, da.OrderID,
		da.ExecuteAt, da.ActionData, da.EventData,
	).Scan(&da.CreatedAt)
}

// ListPendingByTenant returns pending delayed actions for the current tenant (RLS-scoped).
func (r *DelayedActionRepository) ListPendingByTenant(ctx context.Context, tx pgx.Tx) ([]model.DelayedAction, error) {
	rows, err := tx.Query(ctx,
		`SELECT id, tenant_id, rule_id, action_index, order_id, execute_at,
		        executed, executed_at, error, attempt_count, last_attempt_at,
		        created_at, action_data, event_data
		 FROM automation_delayed_actions
		 WHERE NOT executed
		 ORDER BY execute_at ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list pending delayed actions: %w", err)
	}
	defer rows.Close()

	return scanDelayedActions(rows)
}

// ListPending returns delayed actions that are ready to execute.
// This is called from the worker which bypasses RLS.
func (r *DelayedActionRepository) ListPending(ctx context.Context, tx pgx.Tx, limit int) ([]model.DelayedAction, error) {
	if limit <= 0 {
		limit = defaultDelayedActionPendingLimit
	}

	rows, err := tx.Query(ctx,
		`SELECT id, tenant_id, rule_id, action_index, order_id, execute_at,
		        executed, executed_at, error, attempt_count, last_attempt_at,
		        created_at, action_data, event_data
		 FROM automation_delayed_actions
		 WHERE execute_at <= NOW() AND NOT executed
		 ORDER BY execute_at ASC, id ASC
		 LIMIT $1`,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list pending delayed actions: %w", err)
	}
	defer rows.Close()

	return scanDelayedActions(rows)
}

// MarkExecuted marks a delayed action as executed, recording any error message.
func (r *DelayedActionRepository) MarkExecuted(ctx context.Context, tx pgx.Tx, id uuid.UUID, errMsg *string) error {
	now := time.Now()
	_, err := tx.Exec(ctx,
		`UPDATE automation_delayed_actions
		 SET executed = true,
		     executed_at = $2,
		     last_attempt_at = $2,
		     attempt_count = attempt_count + 1,
		     error = $3
		 WHERE id = $1`,
		id, now, errMsg,
	)
	if err != nil {
		return fmt.Errorf("mark delayed action executed: %w", err)
	}
	return nil
}

// RequeueForRetry records a failed attempt and schedules the delayed action for another try.
func (r *DelayedActionRepository) RequeueForRetry(ctx context.Context, tx pgx.Tx, id uuid.UUID, nextExecuteAt time.Time, errMsg string) error {
	now := time.Now()
	_, err := tx.Exec(ctx,
		`UPDATE automation_delayed_actions
		 SET execute_at = $2,
		     executed = false,
		     executed_at = NULL,
		     last_attempt_at = $3,
		     attempt_count = attempt_count + 1,
		     error = $4
		 WHERE id = $1`,
		id, nextExecuteAt, now, errMsg,
	)
	if err != nil {
		return fmt.Errorf("requeue delayed action for retry: %w", err)
	}
	return nil
}

func scanDelayedActions(rows pgx.Rows) ([]model.DelayedAction, error) {
	var actions []model.DelayedAction
	for rows.Next() {
		var da model.DelayedAction
		if err := rows.Scan(
			&da.ID, &da.TenantID, &da.RuleID, &da.ActionIndex, &da.OrderID,
			&da.ExecuteAt, &da.Executed, &da.ExecutedAt, &da.Error,
			&da.AttemptCount, &da.LastAttemptAt, &da.CreatedAt, &da.ActionData,
			&da.EventData,
		); err != nil {
			return nil, fmt.Errorf("scan delayed action: %w", err)
		}
		actions = append(actions, da)
	}
	return actions, rows.Err()
}
