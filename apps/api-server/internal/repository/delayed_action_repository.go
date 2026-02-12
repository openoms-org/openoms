package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// DelayedActionRepository implements persistence for delayed automation actions.
type DelayedActionRepository struct{}

func NewDelayedActionRepository() *DelayedActionRepository {
	return &DelayedActionRepository{}
}

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
		        executed, executed_at, error, created_at, action_data, event_data
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
func (r *DelayedActionRepository) ListPending(ctx context.Context, tx pgx.Tx) ([]model.DelayedAction, error) {
	rows, err := tx.Query(ctx,
		`SELECT id, tenant_id, rule_id, action_index, order_id, execute_at,
		        executed, executed_at, error, created_at, action_data, event_data
		 FROM automation_delayed_actions
		 WHERE execute_at <= NOW() AND NOT executed
		 ORDER BY execute_at ASC
		 LIMIT 100`,
	)
	if err != nil {
		return nil, fmt.Errorf("list pending delayed actions: %w", err)
	}
	defer rows.Close()

	return scanDelayedActions(rows)
}

func (r *DelayedActionRepository) MarkExecuted(ctx context.Context, tx pgx.Tx, id uuid.UUID, errMsg *string) error {
	_, err := tx.Exec(ctx,
		`UPDATE automation_delayed_actions
		 SET executed = true, executed_at = $2, error = $3
		 WHERE id = $1`,
		id, time.Now(), errMsg,
	)
	if err != nil {
		return fmt.Errorf("mark delayed action executed: %w", err)
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
			&da.CreatedAt, &da.ActionData, &da.EventData,
		); err != nil {
			return nil, fmt.Errorf("scan delayed action: %w", err)
		}
		actions = append(actions, da)
	}
	return actions, rows.Err()
}
