package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// RepricingRepository handles persistence for repricing rules and logs.
type RepricingRepository struct{}

// NewRepricingRepository creates a new RepricingRepository.
func NewRepricingRepository() *RepricingRepository {
	return &RepricingRepository{}
}

// --- Rules CRUD ---

// ListRules returns a paginated list of repricing rules matching the filter.
func (r *RepricingRepository) ListRules(ctx context.Context, tx pgx.Tx, filter model.RepricingRuleListFilter) ([]model.RepricingRule, int, error) {
	qb := NewQueryBuilder()
	if filter.Status != nil {
		qb.Add("status = $%d", *filter.Status)
	}
	if filter.Strategy != nil {
		qb.Add("strategy = $%d", *filter.Strategy)
	}
	where := qb.WhereClause()

	var total int
	countQuery := "SELECT COUNT(*) FROM repricing_rules " + where
	if err := tx.QueryRow(ctx, countQuery, qb.Args()...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count repricing rules: %w", err)
	}

	allowedSortColumns := map[string]string{
		"created_at": "created_at",
		"priority":   "priority",
		"name":       "name",
		"status":     "status",
		"strategy":   "strategy",
	}
	orderByClause := model.BuildOrderByClause(filter.SortBy, filter.SortOrder, allowedSortColumns)

	limitIdx := qb.AddArgs(filter.Limit, filter.Offset)
	query := fmt.Sprintf(
		`SELECT id, tenant_id, name, status, strategy, priority, scope_type, scope_value,
		        params, min_price, max_price, channels, last_applied_at, products_affected,
		        created_by, created_at, updated_at
		 FROM repricing_rules %s
		 %s
		 LIMIT $%d OFFSET $%d`,
		where, orderByClause, limitIdx, limitIdx+1,
	)

	rows, err := tx.Query(ctx, query, qb.Args()...)
	if err != nil {
		return nil, 0, fmt.Errorf("list repricing rules: %w", err)
	}
	defer rows.Close()

	var rules []model.RepricingRule
	for rows.Next() {
		var rule model.RepricingRule
		if err := rows.Scan(
			&rule.ID, &rule.TenantID, &rule.Name, &rule.Status, &rule.Strategy,
			&rule.Priority, &rule.ScopeType, &rule.ScopeValue,
			&rule.Params, &rule.MinPrice, &rule.MaxPrice, &rule.Channels,
			&rule.LastAppliedAt, &rule.ProductsAffected,
			&rule.CreatedBy, &rule.CreatedAt, &rule.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan repricing rule: %w", err)
		}
		rules = append(rules, rule)
	}
	return rules, total, rows.Err()
}

// FindRuleByID returns a repricing rule by its ID.
func (r *RepricingRepository) FindRuleByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.RepricingRule, error) {
	var rule model.RepricingRule
	err := tx.QueryRow(ctx,
		`SELECT id, tenant_id, name, status, strategy, priority, scope_type, scope_value,
		        params, min_price, max_price, channels, last_applied_at, products_affected,
		        created_by, created_at, updated_at
		 FROM repricing_rules WHERE id = $1`, id,
	).Scan(
		&rule.ID, &rule.TenantID, &rule.Name, &rule.Status, &rule.Strategy,
		&rule.Priority, &rule.ScopeType, &rule.ScopeValue,
		&rule.Params, &rule.MinPrice, &rule.MaxPrice, &rule.Channels,
		&rule.LastAppliedAt, &rule.ProductsAffected,
		&rule.CreatedBy, &rule.CreatedAt, &rule.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find repricing rule by id: %w", err)
	}
	return &rule, nil
}

// CreateRule inserts a new repricing rule.
func (r *RepricingRepository) CreateRule(ctx context.Context, tx pgx.Tx, rule *model.RepricingRule) error {
	return tx.QueryRow(ctx,
		`INSERT INTO repricing_rules (
			id, tenant_id, name, status, strategy, priority, scope_type, scope_value,
			params, min_price, max_price, channels, created_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING created_at, updated_at`,
		rule.ID, rule.TenantID, rule.Name, rule.Status, rule.Strategy,
		rule.Priority, rule.ScopeType, rule.ScopeValue,
		rule.Params, rule.MinPrice, rule.MaxPrice, rule.Channels, rule.CreatedBy,
	).Scan(&rule.CreatedAt, &rule.UpdatedAt)
}

// UpdateRule applies partial updates to a repricing rule.
func (r *RepricingRepository) UpdateRule(ctx context.Context, tx pgx.Tx, id uuid.UUID, req model.UpdateRepricingRuleRequest) error {
	ub := NewUpdateBuilder()
	SetPtr(ub, "name", req.Name)
	SetPtr(ub, "status", req.Status)
	SetPtr(ub, "strategy", req.Strategy)
	SetPtr(ub, "priority", req.Priority)
	SetPtr(ub, "scope_type", req.ScopeType)
	SetPtr(ub, "scope_value", req.ScopeValue)
	SetPtr(ub, "params", req.Params)
	SetPtr(ub, "min_price", req.MinPrice)
	SetPtr(ub, "max_price", req.MaxPrice)
	SetPtr(ub, "channels", req.Channels)

	if ub.IsEmpty() {
		return nil
	}

	ub.SetRaw("updated_at = NOW()")
	args := append(ub.Args(), id)

	query := fmt.Sprintf("UPDATE repricing_rules SET %s WHERE id = $%d",
		ub.SetClause(), ub.NextArgIdx())

	ct, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update repricing rule: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("repricing rule not found")
	}
	return nil
}

// DeleteRule removes a repricing rule by its ID.
func (r *RepricingRepository) DeleteRule(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	ct, err := tx.Exec(ctx, "DELETE FROM repricing_rules WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete repricing rule: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("repricing rule not found")
	}
	return nil
}

// ListActiveRules returns all active repricing rules ordered by priority.
func (r *RepricingRepository) ListActiveRules(ctx context.Context, tx pgx.Tx) ([]model.RepricingRule, error) {
	rows, err := tx.Query(ctx,
		`SELECT id, tenant_id, name, status, strategy, priority, scope_type, scope_value,
		        params, min_price, max_price, channels, last_applied_at, products_affected,
		        created_by, created_at, updated_at
		 FROM repricing_rules
		 WHERE status = 'active'
		 ORDER BY priority DESC, created_at ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list active repricing rules: %w", err)
	}
	defer rows.Close()

	var rules []model.RepricingRule
	for rows.Next() {
		var rule model.RepricingRule
		if err := rows.Scan(
			&rule.ID, &rule.TenantID, &rule.Name, &rule.Status, &rule.Strategy,
			&rule.Priority, &rule.ScopeType, &rule.ScopeValue,
			&rule.Params, &rule.MinPrice, &rule.MaxPrice, &rule.Channels,
			&rule.LastAppliedAt, &rule.ProductsAffected,
			&rule.CreatedBy, &rule.CreatedAt, &rule.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan active repricing rule: %w", err)
		}
		rules = append(rules, rule)
	}
	return rules, rows.Err()
}

// UpdateRuleApplied records the last applied timestamp and products affected count.
func (r *RepricingRepository) UpdateRuleApplied(ctx context.Context, tx pgx.Tx, id uuid.UUID, productsAffected int) error {
	_, err := tx.Exec(ctx,
		"UPDATE repricing_rules SET last_applied_at = NOW(), products_affected = $1, updated_at = NOW() WHERE id = $2",
		productsAffected, id,
	)
	return err
}

// --- Log entries ---

// CreateLog inserts a new repricing log entry.
func (r *RepricingRepository) CreateLog(ctx context.Context, tx pgx.Tx, log *model.RepricingLog) error {
	return tx.QueryRow(ctx,
		`INSERT INTO repricing_log (id, tenant_id, rule_id, product_id, old_price, new_price, reason, channel)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 RETURNING applied_at`,
		log.ID, log.TenantID, log.RuleID, log.ProductID,
		log.OldPrice, log.NewPrice, log.Reason, log.Channel,
	).Scan(&log.AppliedAt)
}

// ListLogByRule returns repricing log entries for a given rule.
func (r *RepricingRepository) ListLogByRule(ctx context.Context, tx pgx.Tx, ruleID uuid.UUID, limit, offset int) ([]model.RepricingLog, int, error) {
	var total int
	if err := tx.QueryRow(ctx, "SELECT COUNT(*) FROM repricing_log WHERE rule_id = $1", ruleID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count repricing log: %w", err)
	}

	rows, err := tx.Query(ctx,
		`SELECT id, tenant_id, rule_id, product_id, old_price, new_price, reason, channel, applied_at
		 FROM repricing_log
		 WHERE rule_id = $1
		 ORDER BY applied_at DESC
		 LIMIT $2 OFFSET $3`,
		ruleID, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list repricing log by rule: %w", err)
	}
	defer rows.Close()

	var logs []model.RepricingLog
	for rows.Next() {
		var l model.RepricingLog
		if err := rows.Scan(
			&l.ID, &l.TenantID, &l.RuleID, &l.ProductID,
			&l.OldPrice, &l.NewPrice, &l.Reason, &l.Channel, &l.AppliedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan repricing log: %w", err)
		}
		logs = append(logs, l)
	}
	return logs, total, rows.Err()
}

// ListLogByProduct returns repricing log entries for a given product.
func (r *RepricingRepository) ListLogByProduct(ctx context.Context, tx pgx.Tx, productID uuid.UUID, limit, offset int) ([]model.RepricingLog, int, error) {
	var total int
	if err := tx.QueryRow(ctx, "SELECT COUNT(*) FROM repricing_log WHERE product_id = $1", productID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count repricing log by product: %w", err)
	}

	rows, err := tx.Query(ctx,
		`SELECT id, tenant_id, rule_id, product_id, old_price, new_price, reason, channel, applied_at
		 FROM repricing_log
		 WHERE product_id = $1
		 ORDER BY applied_at DESC
		 LIMIT $2 OFFSET $3`,
		productID, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list repricing log by product: %w", err)
	}
	defer rows.Close()

	var logs []model.RepricingLog
	for rows.Next() {
		var l model.RepricingLog
		if err := rows.Scan(
			&l.ID, &l.TenantID, &l.RuleID, &l.ProductID,
			&l.OldPrice, &l.NewPrice, &l.Reason, &l.Channel, &l.AppliedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan repricing log: %w", err)
		}
		logs = append(logs, l)
	}
	return logs, total, rows.Err()
}

// ListLog returns a paginated list of all repricing log entries.
func (r *RepricingRepository) ListLog(ctx context.Context, tx pgx.Tx, limit, offset int) ([]model.RepricingLog, int, error) {
	var total int
	if err := tx.QueryRow(ctx, "SELECT COUNT(*) FROM repricing_log").Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count repricing log: %w", err)
	}

	rows, err := tx.Query(ctx,
		`SELECT id, tenant_id, rule_id, product_id, old_price, new_price, reason, channel, applied_at
		 FROM repricing_log
		 ORDER BY applied_at DESC
		 LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list repricing log: %w", err)
	}
	defer rows.Close()

	var logs []model.RepricingLog
	for rows.Next() {
		var l model.RepricingLog
		if err := rows.Scan(
			&l.ID, &l.TenantID, &l.RuleID, &l.ProductID,
			&l.OldPrice, &l.NewPrice, &l.Reason, &l.Channel, &l.AppliedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan repricing log: %w", err)
		}
		logs = append(logs, l)
	}
	return logs, total, rows.Err()
}

// GetSummary returns aggregated repricing statistics for the current tenant.
func (r *RepricingRepository) GetSummary(ctx context.Context, tx pgx.Tx) (*model.RepricingSummary, error) {
	var s model.RepricingSummary

	// Count active and paused rules
	if err := tx.QueryRow(ctx, "SELECT COUNT(*) FROM repricing_rules WHERE status = 'active'").Scan(&s.ActiveRules); err != nil {
		return nil, fmt.Errorf("count active rules: %w", err)
	}
	if err := tx.QueryRow(ctx, "SELECT COUNT(*) FROM repricing_rules WHERE status = 'paused'").Scan(&s.PausedRules); err != nil {
		return nil, fmt.Errorf("count paused rules: %w", err)
	}

	// Changes today
	if err := tx.QueryRow(ctx,
		"SELECT COUNT(*) FROM repricing_log WHERE applied_at >= CURRENT_DATE",
	).Scan(&s.ChangesToday); err != nil {
		return nil, fmt.Errorf("count changes today: %w", err)
	}

	// Changes this week
	if err := tx.QueryRow(ctx,
		"SELECT COUNT(*) FROM repricing_log WHERE applied_at >= CURRENT_DATE - INTERVAL '7 days'",
	).Scan(&s.ChangesWeek); err != nil {
		return nil, fmt.Errorf("count changes week: %w", err)
	}

	// Average change percentage (this week)
	if err := tx.QueryRow(ctx,
		`SELECT COALESCE(AVG(ABS((new_price - old_price) / NULLIF(old_price, 0) * 100)), 0)
		 FROM repricing_log
		 WHERE applied_at >= CURRENT_DATE - INTERVAL '7 days'`,
	).Scan(&s.AvgChangePct); err != nil {
		return nil, fmt.Errorf("avg change pct: %w", err)
	}

	// Total affected products
	if err := tx.QueryRow(ctx,
		"SELECT COALESCE(SUM(products_affected), 0) FROM repricing_rules WHERE status = 'active'",
	).Scan(&s.TotalAffected); err != nil {
		return nil, fmt.Errorf("total affected: %w", err)
	}

	return &s, nil
}
