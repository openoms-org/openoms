package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// AutomationRuleRepository implements AutomationRuleRepo.
type AutomationRuleRepository struct{}

func NewAutomationRuleRepository() *AutomationRuleRepository {
	return &AutomationRuleRepository{}
}

func (r *AutomationRuleRepository) List(ctx context.Context, tx pgx.Tx, filter model.AutomationRuleListFilter) ([]model.AutomationRule, int, error) {
	where := "WHERE 1=1"
	args := []any{}
	argIdx := 1

	if filter.TriggerEvent != nil {
		where += fmt.Sprintf(" AND trigger_event = $%d", argIdx)
		args = append(args, *filter.TriggerEvent)
		argIdx++
	}
	if filter.Enabled != nil {
		where += fmt.Sprintf(" AND enabled = $%d", argIdx)
		args = append(args, *filter.Enabled)
		argIdx++
	}

	var total int
	countQuery := "SELECT COUNT(*) FROM automation_rules " + where
	if err := tx.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count automation rules: %w", err)
	}

	allowedSortColumns := map[string]string{
		"created_at": "created_at",
		"name":       "name",
		"priority":   "priority",
		"fire_count": "fire_count",
	}
	orderByClause := model.BuildOrderByClause(filter.SortBy, filter.SortOrder, allowedSortColumns)

	query := fmt.Sprintf(
		`SELECT id, tenant_id, name, description, enabled, priority,
		        trigger_event, conditions, actions, last_fired_at, fire_count,
		        created_at, updated_at
		 FROM automation_rules %s
		 %s
		 LIMIT $%d OFFSET $%d`,
		where, orderByClause, argIdx, argIdx+1,
	)
	args = append(args, filter.Limit, filter.Offset)

	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list automation rules: %w", err)
	}
	defer rows.Close()

	var rules []model.AutomationRule
	for rows.Next() {
		var rule model.AutomationRule
		if err := rows.Scan(
			&rule.ID, &rule.TenantID, &rule.Name, &rule.Description,
			&rule.Enabled, &rule.Priority, &rule.TriggerEvent,
			&rule.Conditions, &rule.Actions, &rule.LastFiredAt, &rule.FireCount,
			&rule.CreatedAt, &rule.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan automation rule: %w", err)
		}
		rules = append(rules, rule)
	}
	return rules, total, rows.Err()
}

func (r *AutomationRuleRepository) FindByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.AutomationRule, error) {
	var rule model.AutomationRule
	err := tx.QueryRow(ctx,
		`SELECT id, tenant_id, name, description, enabled, priority,
		        trigger_event, conditions, actions, last_fired_at, fire_count,
		        created_at, updated_at
		 FROM automation_rules WHERE id = $1`, id,
	).Scan(
		&rule.ID, &rule.TenantID, &rule.Name, &rule.Description,
		&rule.Enabled, &rule.Priority, &rule.TriggerEvent,
		&rule.Conditions, &rule.Actions, &rule.LastFiredAt, &rule.FireCount,
		&rule.CreatedAt, &rule.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find automation rule by id: %w", err)
	}
	return &rule, nil
}

func (r *AutomationRuleRepository) FindByTenantAndEvent(ctx context.Context, tx pgx.Tx, event string) ([]model.AutomationRule, error) {
	rows, err := tx.Query(ctx,
		`SELECT id, tenant_id, name, description, enabled, priority,
		        trigger_event, conditions, actions, last_fired_at, fire_count,
		        created_at, updated_at
		 FROM automation_rules
		 WHERE trigger_event = $1 AND enabled = true
		 ORDER BY priority DESC, created_at ASC`, event,
	)
	if err != nil {
		return nil, fmt.Errorf("find automation rules by event: %w", err)
	}
	defer rows.Close()

	var rules []model.AutomationRule
	for rows.Next() {
		var rule model.AutomationRule
		if err := rows.Scan(
			&rule.ID, &rule.TenantID, &rule.Name, &rule.Description,
			&rule.Enabled, &rule.Priority, &rule.TriggerEvent,
			&rule.Conditions, &rule.Actions, &rule.LastFiredAt, &rule.FireCount,
			&rule.CreatedAt, &rule.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan automation rule: %w", err)
		}
		rules = append(rules, rule)
	}
	return rules, rows.Err()
}

func (r *AutomationRuleRepository) Create(ctx context.Context, tx pgx.Tx, rule *model.AutomationRule) error {
	return tx.QueryRow(ctx,
		`INSERT INTO automation_rules (
			id, tenant_id, name, description, enabled, priority,
			trigger_event, conditions, actions
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING created_at, updated_at`,
		rule.ID, rule.TenantID, rule.Name, rule.Description,
		rule.Enabled, rule.Priority, rule.TriggerEvent,
		rule.Conditions, rule.Actions,
	).Scan(&rule.CreatedAt, &rule.UpdatedAt)
}

func (r *AutomationRuleRepository) Update(ctx context.Context, tx pgx.Tx, id uuid.UUID, req model.UpdateAutomationRuleRequest) error {
	setClauses := []string{}
	args := []any{}
	argIdx := 1

	if req.Name != nil {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, *req.Name)
		argIdx++
	}
	if req.Description != nil {
		setClauses = append(setClauses, fmt.Sprintf("description = $%d", argIdx))
		args = append(args, *req.Description)
		argIdx++
	}
	if req.Enabled != nil {
		setClauses = append(setClauses, fmt.Sprintf("enabled = $%d", argIdx))
		args = append(args, *req.Enabled)
		argIdx++
	}
	if req.Priority != nil {
		setClauses = append(setClauses, fmt.Sprintf("priority = $%d", argIdx))
		args = append(args, *req.Priority)
		argIdx++
	}
	if req.TriggerEvent != nil {
		setClauses = append(setClauses, fmt.Sprintf("trigger_event = $%d", argIdx))
		args = append(args, *req.TriggerEvent)
		argIdx++
	}
	if req.Conditions != nil {
		setClauses = append(setClauses, fmt.Sprintf("conditions = $%d", argIdx))
		args = append(args, req.Conditions)
		argIdx++
	}
	if req.Actions != nil {
		setClauses = append(setClauses, fmt.Sprintf("actions = $%d", argIdx))
		args = append(args, req.Actions)
		argIdx++
	}

	if len(setClauses) == 0 {
		return nil
	}

	var query strings.Builder
	query.WriteString("UPDATE automation_rules SET ")
	for i, clause := range setClauses {
		if i > 0 {
			query.WriteString(", ")
		}
		query.WriteString(clause)
	}
	query.WriteString(fmt.Sprintf(" WHERE id = $%d", argIdx))
	args = append(args, id)

	ct, err := tx.Exec(ctx, query.String(), args...)
	if err != nil {
		return fmt.Errorf("update automation rule: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("automation rule not found")
	}
	return nil
}

func (r *AutomationRuleRepository) Delete(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	ct, err := tx.Exec(ctx, "DELETE FROM automation_rules WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete automation rule: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("automation rule not found")
	}
	return nil
}

func (r *AutomationRuleRepository) IncrementFireCount(ctx context.Context, tx pgx.Tx, id uuid.UUID, firedAt time.Time) error {
	_, err := tx.Exec(ctx,
		`UPDATE automation_rules SET fire_count = fire_count + 1, last_fired_at = $2 WHERE id = $1`,
		id, firedAt,
	)
	if err != nil {
		return fmt.Errorf("increment fire count: %w", err)
	}
	return nil
}

// AutomationRuleLogRepository implements AutomationRuleLogRepo.
type AutomationRuleLogRepository struct{}

func NewAutomationRuleLogRepository() *AutomationRuleLogRepository {
	return &AutomationRuleLogRepository{}
}

func (r *AutomationRuleLogRepository) Create(ctx context.Context, tx pgx.Tx, log *model.AutomationRuleLog) error {
	return tx.QueryRow(ctx,
		`INSERT INTO automation_rule_logs (
			id, tenant_id, rule_id, trigger_event, entity_type, entity_id,
			conditions_met, actions_executed, error_message
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING executed_at`,
		log.ID, log.TenantID, log.RuleID, log.TriggerEvent,
		log.EntityType, log.EntityID, log.ConditionsMet,
		log.ActionsExecuted, log.ErrorMessage,
	).Scan(&log.ExecutedAt)
}

func (r *AutomationRuleLogRepository) ListByRuleID(ctx context.Context, tx pgx.Tx, ruleID uuid.UUID, limit, offset int) ([]model.AutomationRuleLog, int, error) {
	var total int
	if err := tx.QueryRow(ctx, "SELECT COUNT(*) FROM automation_rule_logs WHERE rule_id = $1", ruleID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count automation rule logs: %w", err)
	}

	rows, err := tx.Query(ctx,
		`SELECT id, tenant_id, rule_id, trigger_event, entity_type, entity_id,
		        conditions_met, actions_executed, error_message, executed_at
		 FROM automation_rule_logs
		 WHERE rule_id = $1
		 ORDER BY executed_at DESC
		 LIMIT $2 OFFSET $3`, ruleID, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list automation rule logs: %w", err)
	}
	defer rows.Close()

	var logs []model.AutomationRuleLog
	for rows.Next() {
		var log model.AutomationRuleLog
		if err := rows.Scan(
			&log.ID, &log.TenantID, &log.RuleID, &log.TriggerEvent,
			&log.EntityType, &log.EntityID, &log.ConditionsMet,
			&log.ActionsExecuted, &log.ErrorMessage, &log.ExecutedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan automation rule log: %w", err)
		}
		logs = append(logs, log)
	}
	return logs, total, rows.Err()
}
