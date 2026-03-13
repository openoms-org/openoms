package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// StockSyncChannelRepository implements StockSyncChannelRepo.
type StockSyncChannelRepository struct{}

// NewStockSyncChannelRepository creates a new StockSyncChannelRepository.
func NewStockSyncChannelRepository() *StockSyncChannelRepository {
	return &StockSyncChannelRepository{}
}

func scanStockSyncChannel(row pgx.Row) (model.StockSyncChannel, error) {
	var ch model.StockSyncChannel
	err := row.Scan(
		&ch.ID, &ch.TenantID, &ch.IntegrationID, &ch.ChannelType,
		&ch.Enabled, &ch.StockBuffer, &ch.SyncMode, &ch.Priority,
		&ch.LastSyncAt, &ch.LastError, &ch.CreatedAt, &ch.UpdatedAt,
	)
	return ch, err
}

const stockSyncChannelColumns = `id, tenant_id, integration_id, channel_type,
	enabled, stock_buffer, sync_mode, priority,
	last_sync_at, last_error, created_at, updated_at`

// List returns a paginated list of stock sync channels matching the filter.
func (r *StockSyncChannelRepository) List(ctx context.Context, tx pgx.Tx, filter model.StockSyncChannelListFilter) ([]model.StockSyncChannel, int, error) {
	var conditions []string
	var args []any
	argIdx := 1

	if filter.Enabled != nil {
		conditions = append(conditions, fmt.Sprintf("enabled = $%d", argIdx))
		args = append(args, *filter.Enabled)
		argIdx++
	}
	if filter.ChannelType != nil {
		conditions = append(conditions, fmt.Sprintf("channel_type = $%d", argIdx))
		args = append(args, *filter.ChannelType)
		argIdx++
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM stock_sync_channels %s", where)
	if err := tx.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count stock sync channels: %w", err)
	}

	allowedSortColumns := map[string]string{
		"created_at":   "created_at",
		"channel_type": "channel_type",
		"priority":     "priority",
		"last_sync_at": "last_sync_at",
	}
	orderByClause := model.BuildOrderByClause(filter.SortBy, filter.SortOrder, allowedSortColumns)

	query := fmt.Sprintf(
		`SELECT %s FROM stock_sync_channels %s %s LIMIT $%d OFFSET $%d`,
		stockSyncChannelColumns, where, orderByClause, argIdx, argIdx+1,
	)
	args = append(args, filter.Limit, filter.Offset)

	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list stock sync channels: %w", err)
	}
	defer rows.Close()

	var channels []model.StockSyncChannel
	for rows.Next() {
		var ch model.StockSyncChannel
		if err := rows.Scan(
			&ch.ID, &ch.TenantID, &ch.IntegrationID, &ch.ChannelType,
			&ch.Enabled, &ch.StockBuffer, &ch.SyncMode, &ch.Priority,
			&ch.LastSyncAt, &ch.LastError, &ch.CreatedAt, &ch.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan stock sync channel: %w", err)
		}
		channels = append(channels, ch)
	}
	return channels, total, rows.Err()
}

// FindByID returns a stock sync channel by its ID.
func (r *StockSyncChannelRepository) FindByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.StockSyncChannel, error) {
	ch, err := scanStockSyncChannel(tx.QueryRow(ctx,
		fmt.Sprintf("SELECT %s FROM stock_sync_channels WHERE id = $1", stockSyncChannelColumns), id,
	))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find stock sync channel by id: %w", err)
	}
	return &ch, nil
}

// ListEnabled returns all enabled stock sync channels ordered by priority.
func (r *StockSyncChannelRepository) ListEnabled(ctx context.Context, tx pgx.Tx) ([]model.StockSyncChannel, error) {
	rows, err := tx.Query(ctx,
		fmt.Sprintf("SELECT %s FROM stock_sync_channels WHERE enabled = true ORDER BY priority DESC, created_at ASC", stockSyncChannelColumns),
	)
	if err != nil {
		return nil, fmt.Errorf("list enabled stock sync channels: %w", err)
	}
	defer rows.Close()

	var channels []model.StockSyncChannel
	for rows.Next() {
		var ch model.StockSyncChannel
		if err := rows.Scan(
			&ch.ID, &ch.TenantID, &ch.IntegrationID, &ch.ChannelType,
			&ch.Enabled, &ch.StockBuffer, &ch.SyncMode, &ch.Priority,
			&ch.LastSyncAt, &ch.LastError, &ch.CreatedAt, &ch.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan enabled stock sync channel: %w", err)
		}
		channels = append(channels, ch)
	}
	return channels, rows.Err()
}

// Create inserts a new stock sync channel.
func (r *StockSyncChannelRepository) Create(ctx context.Context, tx pgx.Tx, ch *model.StockSyncChannel) error {
	return tx.QueryRow(ctx,
		`INSERT INTO stock_sync_channels (id, tenant_id, integration_id, channel_type, enabled, stock_buffer, sync_mode, priority)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 RETURNING created_at, updated_at`,
		ch.ID, ch.TenantID, ch.IntegrationID, ch.ChannelType,
		ch.Enabled, ch.StockBuffer, ch.SyncMode, ch.Priority,
	).Scan(&ch.CreatedAt, &ch.UpdatedAt)
}

// Update applies partial updates to a stock sync channel.
func (r *StockSyncChannelRepository) Update(ctx context.Context, tx pgx.Tx, id uuid.UUID, req model.UpdateStockSyncChannelRequest) error {
	var setClauses []string
	var args []any
	argIdx := 1

	if req.Enabled != nil {
		setClauses = append(setClauses, fmt.Sprintf("enabled = $%d", argIdx))
		args = append(args, *req.Enabled)
		argIdx++
	}
	if req.StockBuffer != nil {
		setClauses = append(setClauses, fmt.Sprintf("stock_buffer = $%d", argIdx))
		args = append(args, *req.StockBuffer)
		argIdx++
	}
	if req.SyncMode != nil {
		setClauses = append(setClauses, fmt.Sprintf("sync_mode = $%d", argIdx))
		args = append(args, *req.SyncMode)
		argIdx++
	}
	if req.Priority != nil {
		setClauses = append(setClauses, fmt.Sprintf("priority = $%d", argIdx))
		args = append(args, *req.Priority)
		argIdx++
	}

	if len(setClauses) == 0 {
		return nil
	}

	setClauses = append(setClauses, "updated_at = NOW()")
	query := fmt.Sprintf("UPDATE stock_sync_channels SET %s WHERE id = $%d",
		strings.Join(setClauses, ", "), argIdx)
	args = append(args, id)

	ct, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update stock sync channel: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("stock sync channel not found")
	}
	return nil
}

// Delete removes a stock sync channel by its ID.
func (r *StockSyncChannelRepository) Delete(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	ct, err := tx.Exec(ctx, "DELETE FROM stock_sync_channels WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete stock sync channel: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("stock sync channel not found")
	}
	return nil
}

// UpdateSyncStatus records the latest sync timestamp and error for the channel.
func (r *StockSyncChannelRepository) UpdateSyncStatus(ctx context.Context, tx pgx.Tx, id uuid.UUID, lastError *string) error {
	if lastError != nil {
		_, err := tx.Exec(ctx,
			"UPDATE stock_sync_channels SET last_sync_at = NOW(), last_error = $1, updated_at = NOW() WHERE id = $2",
			*lastError, id,
		)
		return err
	}
	_, err := tx.Exec(ctx,
		"UPDATE stock_sync_channels SET last_sync_at = NOW(), last_error = NULL, updated_at = NOW() WHERE id = $1",
		id,
	)
	return err
}

// StockSyncEventRepository implements StockSyncEventRepo.
type StockSyncEventRepository struct{}

// NewStockSyncEventRepository creates a new StockSyncEventRepository.
func NewStockSyncEventRepository() *StockSyncEventRepository {
	return &StockSyncEventRepository{}
}

// Create inserts a new stock sync event.
func (r *StockSyncEventRepository) Create(ctx context.Context, tx pgx.Tx, event *model.StockSyncEvent) error {
	return tx.QueryRow(ctx,
		`INSERT INTO stock_sync_events (id, tenant_id, product_id, sku, trigger_type,
		 old_quantity, new_quantity, available_quantity, channels_notified, channels_failed, details)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		 RETURNING created_at`,
		event.ID, event.TenantID, event.ProductID, event.SKU, event.TriggerType,
		event.OldQuantity, event.NewQuantity, event.AvailableQuantity,
		event.ChannelsNotified, event.ChannelsFailed, event.Details,
	).Scan(&event.CreatedAt)
}

// List returns a paginated list of stock sync events matching the filter.
func (r *StockSyncEventRepository) List(ctx context.Context, tx pgx.Tx, filter model.StockSyncEventListFilter) ([]model.StockSyncEvent, int, error) {
	var conditions []string
	var args []any
	argIdx := 1

	if filter.ProductID != nil {
		conditions = append(conditions, fmt.Sprintf("product_id = $%d", argIdx))
		args = append(args, *filter.ProductID)
		argIdx++
	}
	if filter.TriggerType != nil {
		conditions = append(conditions, fmt.Sprintf("trigger_type = $%d", argIdx))
		args = append(args, *filter.TriggerType)
		argIdx++
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM stock_sync_events %s", where)
	if err := tx.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count stock sync events: %w", err)
	}

	query := fmt.Sprintf(
		`SELECT id, tenant_id, product_id, sku, trigger_type,
		 old_quantity, new_quantity, available_quantity,
		 channels_notified, channels_failed, details, created_at
		 FROM stock_sync_events %s
		 ORDER BY created_at DESC
		 LIMIT $%d OFFSET $%d`,
		where, argIdx, argIdx+1,
	)
	args = append(args, filter.Limit, filter.Offset)

	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list stock sync events: %w", err)
	}
	defer rows.Close()

	var events []model.StockSyncEvent
	for rows.Next() {
		var e model.StockSyncEvent
		if err := rows.Scan(
			&e.ID, &e.TenantID, &e.ProductID, &e.SKU, &e.TriggerType,
			&e.OldQuantity, &e.NewQuantity, &e.AvailableQuantity,
			&e.ChannelsNotified, &e.ChannelsFailed, &e.Details, &e.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan stock sync event: %w", err)
		}
		events = append(events, e)
	}
	return events, total, rows.Err()
}

// CountRecentErrors returns the number of sync events with failures in the last 24 hours.
func (r *StockSyncEventRepository) CountRecentErrors(ctx context.Context, tx pgx.Tx) (int, error) {
	var count int
	err := tx.QueryRow(ctx,
		`SELECT COUNT(*) FROM stock_sync_events
		 WHERE channels_failed > 0 AND created_at > NOW() - INTERVAL '24 hours'`,
	).Scan(&count)
	return count, err
}

// GetAvailableStock calculates total stock minus reserved across all warehouses for a product.
func (r *StockSyncEventRepository) GetAvailableStock(ctx context.Context, tx pgx.Tx, productID uuid.UUID) (totalQty int, reservedQty int, err error) {
	err = tx.QueryRow(ctx,
		`SELECT COALESCE(SUM(quantity), 0), COALESCE(SUM(reserved), 0)
		 FROM warehouse_stock WHERE product_id = $1`,
		productID,
	).Scan(&totalQty, &reservedQty)
	return
}
