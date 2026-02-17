package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// ListingSyncRepository handles persistence for listing sync configs and logs.
type ListingSyncRepository struct{}

func NewListingSyncRepository() *ListingSyncRepository {
	return &ListingSyncRepository{}
}

// --- Configs ---

func (r *ListingSyncRepository) CreateConfig(ctx context.Context, tx pgx.Tx, cfg *model.ListingSyncConfig) error {
	return tx.QueryRow(ctx,
		`INSERT INTO listing_sync_configs (
			id, tenant_id, integration_id, sync_direction, auto_sync,
			sync_interval_minutes, field_mapping, price_rule, price_modifier,
			stock_buffer, status
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING created_at, updated_at`,
		cfg.ID, cfg.TenantID, cfg.IntegrationID, cfg.SyncDirection, cfg.AutoSync,
		cfg.SyncIntervalMinutes, cfg.FieldMapping, cfg.PriceRule, cfg.PriceModifier,
		cfg.StockBuffer, cfg.Status,
	).Scan(&cfg.CreatedAt, &cfg.UpdatedAt)
}

func (r *ListingSyncRepository) UpdateConfig(ctx context.Context, tx pgx.Tx, id uuid.UUID, req *model.UpdateListingSyncConfigRequest) error {
	var setClauses []string
	var args []any
	argIdx := 1

	if req.SyncDirection != nil {
		setClauses = append(setClauses, fmt.Sprintf("sync_direction = $%d", argIdx))
		args = append(args, *req.SyncDirection)
		argIdx++
	}
	if req.AutoSync != nil {
		setClauses = append(setClauses, fmt.Sprintf("auto_sync = $%d", argIdx))
		args = append(args, *req.AutoSync)
		argIdx++
	}
	if req.SyncIntervalMinutes != nil {
		setClauses = append(setClauses, fmt.Sprintf("sync_interval_minutes = $%d", argIdx))
		args = append(args, *req.SyncIntervalMinutes)
		argIdx++
	}
	if req.FieldMapping != nil {
		setClauses = append(setClauses, fmt.Sprintf("field_mapping = $%d", argIdx))
		args = append(args, *req.FieldMapping)
		argIdx++
	}
	if req.PriceRule != nil {
		setClauses = append(setClauses, fmt.Sprintf("price_rule = $%d", argIdx))
		args = append(args, *req.PriceRule)
		argIdx++
	}
	if req.PriceModifier != nil {
		setClauses = append(setClauses, fmt.Sprintf("price_modifier = $%d", argIdx))
		args = append(args, *req.PriceModifier)
		argIdx++
	}
	if req.StockBuffer != nil {
		setClauses = append(setClauses, fmt.Sprintf("stock_buffer = $%d", argIdx))
		args = append(args, *req.StockBuffer)
		argIdx++
	}
	if req.Status != nil {
		setClauses = append(setClauses, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *req.Status)
		argIdx++
	}

	if len(setClauses) == 0 {
		return nil
	}

	setClauses = append(setClauses, "updated_at = NOW()")
	query := fmt.Sprintf("UPDATE listing_sync_configs SET %s WHERE id = $%d",
		strings.Join(setClauses, ", "), argIdx)
	args = append(args, id)

	ct, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update listing sync config: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("listing sync config not found")
	}
	return nil
}

func (r *ListingSyncRepository) GetConfigByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.ListingSyncConfig, error) {
	var c model.ListingSyncConfig
	err := tx.QueryRow(ctx,
		`SELECT lsc.id, lsc.tenant_id, lsc.integration_id, lsc.sync_direction,
		        lsc.auto_sync, lsc.sync_interval_minutes, lsc.field_mapping,
		        lsc.price_rule, lsc.price_modifier, lsc.stock_buffer,
		        lsc.status, lsc.last_sync_at, lsc.last_error,
		        lsc.created_at, lsc.updated_at,
		        COALESCE(i.provider, ''), i.label
		 FROM listing_sync_configs lsc
		 LEFT JOIN integrations i ON i.id = lsc.integration_id
		 WHERE lsc.id = $1`, id,
	).Scan(
		&c.ID, &c.TenantID, &c.IntegrationID, &c.SyncDirection,
		&c.AutoSync, &c.SyncIntervalMinutes, &c.FieldMapping,
		&c.PriceRule, &c.PriceModifier, &c.StockBuffer,
		&c.Status, &c.LastSyncAt, &c.LastError,
		&c.CreatedAt, &c.UpdatedAt,
		&c.IntegrationProvider, &c.IntegrationLabel,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get listing sync config: %w", err)
	}
	return &c, nil
}

func (r *ListingSyncRepository) GetConfigByIntegration(ctx context.Context, tx pgx.Tx, integrationID uuid.UUID) (*model.ListingSyncConfig, error) {
	var c model.ListingSyncConfig
	err := tx.QueryRow(ctx,
		`SELECT lsc.id, lsc.tenant_id, lsc.integration_id, lsc.sync_direction,
		        lsc.auto_sync, lsc.sync_interval_minutes, lsc.field_mapping,
		        lsc.price_rule, lsc.price_modifier, lsc.stock_buffer,
		        lsc.status, lsc.last_sync_at, lsc.last_error,
		        lsc.created_at, lsc.updated_at,
		        COALESCE(i.provider, ''), i.label
		 FROM listing_sync_configs lsc
		 LEFT JOIN integrations i ON i.id = lsc.integration_id
		 WHERE lsc.integration_id = $1 LIMIT 1`, integrationID,
	).Scan(
		&c.ID, &c.TenantID, &c.IntegrationID, &c.SyncDirection,
		&c.AutoSync, &c.SyncIntervalMinutes, &c.FieldMapping,
		&c.PriceRule, &c.PriceModifier, &c.StockBuffer,
		&c.Status, &c.LastSyncAt, &c.LastError,
		&c.CreatedAt, &c.UpdatedAt,
		&c.IntegrationProvider, &c.IntegrationLabel,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get config by integration: %w", err)
	}
	return &c, nil
}

func (r *ListingSyncRepository) ListConfigs(ctx context.Context, tx pgx.Tx, filter model.ListingSyncConfigFilter) ([]model.ListingSyncConfig, int, error) {
	var conditions []string
	var args []any
	argIdx := 1

	if filter.IntegrationID != nil {
		conditions = append(conditions, fmt.Sprintf("lsc.integration_id = $%d", argIdx))
		args = append(args, *filter.IntegrationID)
		argIdx++
	}
	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("lsc.status = $%d", argIdx))
		args = append(args, *filter.Status)
		argIdx++
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM listing_sync_configs lsc %s", where)
	var total int
	if err := tx.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count listing sync configs: %w", err)
	}

	// Items
	query := fmt.Sprintf(
		`SELECT lsc.id, lsc.tenant_id, lsc.integration_id, lsc.sync_direction,
		        lsc.auto_sync, lsc.sync_interval_minutes, lsc.field_mapping,
		        lsc.price_rule, lsc.price_modifier, lsc.stock_buffer,
		        lsc.status, lsc.last_sync_at, lsc.last_error,
		        lsc.created_at, lsc.updated_at,
		        COALESCE(i.provider, ''), i.label
		 FROM listing_sync_configs lsc
		 LEFT JOIN integrations i ON i.id = lsc.integration_id
		 %s
		 ORDER BY lsc.created_at DESC
		 LIMIT $%d OFFSET $%d`,
		where, argIdx, argIdx+1,
	)
	args = append(args, filter.Limit, filter.Offset)

	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list listing sync configs: %w", err)
	}
	defer rows.Close()

	var configs []model.ListingSyncConfig
	for rows.Next() {
		var c model.ListingSyncConfig
		if err := rows.Scan(
			&c.ID, &c.TenantID, &c.IntegrationID, &c.SyncDirection,
			&c.AutoSync, &c.SyncIntervalMinutes, &c.FieldMapping,
			&c.PriceRule, &c.PriceModifier, &c.StockBuffer,
			&c.Status, &c.LastSyncAt, &c.LastError,
			&c.CreatedAt, &c.UpdatedAt,
			&c.IntegrationProvider, &c.IntegrationLabel,
		); err != nil {
			return nil, 0, fmt.Errorf("scan listing sync config: %w", err)
		}
		configs = append(configs, c)
	}
	return configs, total, rows.Err()
}

func (r *ListingSyncRepository) DeleteConfig(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	ct, err := tx.Exec(ctx, "DELETE FROM listing_sync_configs WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete listing sync config: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("listing sync config not found")
	}
	return nil
}

func (r *ListingSyncRepository) UpdateLastSync(ctx context.Context, tx pgx.Tx, id uuid.UUID, lastError *string) error {
	_, err := tx.Exec(ctx,
		`UPDATE listing_sync_configs SET last_sync_at = NOW(), last_error = $1, updated_at = NOW() WHERE id = $2`,
		lastError, id,
	)
	return err
}

// --- Logs ---

func (r *ListingSyncRepository) CreateLog(ctx context.Context, tx pgx.Tx, log *model.ListingSyncLog) error {
	return tx.QueryRow(ctx,
		`INSERT INTO listing_sync_log (
			id, tenant_id, config_id, direction, entity_type,
			product_id, external_id, status, changes, error_message
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING created_at`,
		log.ID, log.TenantID, log.ConfigID, log.Direction, log.EntityType,
		log.ProductID, log.ExternalID, log.Status, log.Changes, log.ErrorMessage,
	).Scan(&log.CreatedAt)
}

func (r *ListingSyncRepository) ListLogs(ctx context.Context, tx pgx.Tx, filter model.ListingSyncLogFilter) ([]model.ListingSyncLog, int, error) {
	var conditions []string
	var args []any
	argIdx := 1

	conditions = append(conditions, fmt.Sprintf("config_id = $%d", argIdx))
	args = append(args, filter.ConfigID)
	argIdx++

	if filter.Direction != nil {
		conditions = append(conditions, fmt.Sprintf("direction = $%d", argIdx))
		args = append(args, *filter.Direction)
		argIdx++
	}
	if filter.EntityType != nil {
		conditions = append(conditions, fmt.Sprintf("entity_type = $%d", argIdx))
		args = append(args, *filter.EntityType)
		argIdx++
	}
	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *filter.Status)
		argIdx++
	}

	where := "WHERE " + strings.Join(conditions, " AND ")

	// Count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM listing_sync_log %s", where)
	var total int
	if err := tx.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count listing sync logs: %w", err)
	}

	// Items
	query := fmt.Sprintf(
		`SELECT id, tenant_id, config_id, direction, entity_type,
		        product_id, external_id, status, changes, error_message, created_at
		 FROM listing_sync_log
		 %s
		 ORDER BY created_at DESC
		 LIMIT $%d OFFSET $%d`,
		where, argIdx, argIdx+1,
	)
	args = append(args, filter.Limit, filter.Offset)

	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list listing sync logs: %w", err)
	}
	defer rows.Close()

	var logs []model.ListingSyncLog
	for rows.Next() {
		var l model.ListingSyncLog
		if err := rows.Scan(
			&l.ID, &l.TenantID, &l.ConfigID, &l.Direction, &l.EntityType,
			&l.ProductID, &l.ExternalID, &l.Status, &l.Changes, &l.ErrorMessage,
			&l.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan listing sync log: %w", err)
		}
		logs = append(logs, l)
	}
	return logs, total, rows.Err()
}

// --- Cross-tenant queries (for worker, bypasses RLS) ---

// ListAutoSyncConfigs returns all active configs with auto_sync=true across all tenants.
// Must run on workerPool (superuser, no RLS).
func (r *ListingSyncRepository) ListAutoSyncConfigs(ctx context.Context, tx pgx.Tx) ([]model.ListingSyncConfig, error) {
	rows, err := tx.Query(ctx,
		`SELECT lsc.id, lsc.tenant_id, lsc.integration_id, lsc.sync_direction,
		        lsc.auto_sync, lsc.sync_interval_minutes, lsc.field_mapping,
		        lsc.price_rule, lsc.price_modifier, lsc.stock_buffer,
		        lsc.status, lsc.last_sync_at, lsc.last_error,
		        lsc.created_at, lsc.updated_at,
		        COALESCE(i.provider, ''), i.label
		 FROM listing_sync_configs lsc
		 LEFT JOIN integrations i ON i.id = lsc.integration_id
		 WHERE lsc.auto_sync = true AND lsc.status = 'active'
		 ORDER BY lsc.last_sync_at ASC NULLS FIRST`)
	if err != nil {
		return nil, fmt.Errorf("list auto sync configs: %w", err)
	}
	defer rows.Close()

	var configs []model.ListingSyncConfig
	for rows.Next() {
		var c model.ListingSyncConfig
		if err := rows.Scan(
			&c.ID, &c.TenantID, &c.IntegrationID, &c.SyncDirection,
			&c.AutoSync, &c.SyncIntervalMinutes, &c.FieldMapping,
			&c.PriceRule, &c.PriceModifier, &c.StockBuffer,
			&c.Status, &c.LastSyncAt, &c.LastError,
			&c.CreatedAt, &c.UpdatedAt,
			&c.IntegrationProvider, &c.IntegrationLabel,
		); err != nil {
			return nil, fmt.Errorf("scan auto sync config: %w", err)
		}
		configs = append(configs, c)
	}
	return configs, rows.Err()
}
