package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// PickPackRepository implements persistence for pick-pack sessions and items.
type PickPackRepository struct{}

// NewPickPackRepository creates a new PickPackRepository.
func NewPickPackRepository() *PickPackRepository {
	return &PickPackRepository{}
}

// CreateSession inserts a new pick-pack session.
func (r *PickPackRepository) CreateSession(ctx context.Context, tx pgx.Tx, session *model.PickPackSession) error {
	return tx.QueryRow(ctx,
		`INSERT INTO pick_pack_sessions (id, tenant_id, session_type, status, assigned_to, notes)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING started_at, created_at, updated_at`,
		session.ID, session.TenantID, session.SessionType, session.Status,
		session.AssignedTo, session.Notes,
	).Scan(&session.StartedAt, &session.CreatedAt, &session.UpdatedAt)
}

// FindSessionByID retrieves a session by ID.
func (r *PickPackRepository) FindSessionByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.PickPackSession, error) {
	var s model.PickPackSession
	err := tx.QueryRow(ctx,
		`SELECT id, tenant_id, session_type, status, assigned_to, started_at,
		        completed_at, notes, created_at, updated_at
		 FROM pick_pack_sessions WHERE id = $1`, id,
	).Scan(
		&s.ID, &s.TenantID, &s.SessionType, &s.Status, &s.AssignedTo,
		&s.StartedAt, &s.CompletedAt, &s.Notes, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find pick_pack_session by id: %w", err)
	}
	return &s, nil
}

// ListSessions lists pick-pack sessions with filtering and pagination.
func (r *PickPackRepository) ListSessions(ctx context.Context, tx pgx.Tx, filter model.PickPackSessionListFilter) ([]model.PickPackSession, int, error) {
	qb := NewQueryBuilder()
	if filter.Status != nil {
		qb.Add("status = $%d", *filter.Status)
	}
	if filter.AssignedTo != nil {
		qb.Add("assigned_to = $%d", *filter.AssignedTo)
	}
	where := qb.WhereClause()

	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM pick_pack_sessions %s", where)
	if err := tx.QueryRow(ctx, countQuery, qb.Args()...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count pick_pack_sessions: %w", err)
	}

	allowedSortColumns := map[string]string{
		"created_at": "created_at",
		"started_at": "started_at",
		"status":     "status",
	}
	orderByClause := model.BuildOrderByClause(filter.SortBy, filter.SortOrder, allowedSortColumns)

	limitIdx := qb.AddArgs(filter.Limit, filter.Offset)
	query := fmt.Sprintf(
		`SELECT id, tenant_id, session_type, status, assigned_to, started_at,
		        completed_at, notes, created_at, updated_at
		 FROM pick_pack_sessions %s %s LIMIT $%d OFFSET $%d`,
		where, orderByClause, limitIdx, limitIdx+1,
	)

	rows, err := tx.Query(ctx, query, qb.Args()...)
	if err != nil {
		return nil, 0, fmt.Errorf("list pick_pack_sessions: %w", err)
	}
	defer rows.Close()

	var sessions []model.PickPackSession
	for rows.Next() {
		var s model.PickPackSession
		if err := rows.Scan(
			&s.ID, &s.TenantID, &s.SessionType, &s.Status, &s.AssignedTo,
			&s.StartedAt, &s.CompletedAt, &s.Notes, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan pick_pack_session: %w", err)
		}
		sessions = append(sessions, s)
	}
	return sessions, total, rows.Err()
}

// UpdateSessionStatus updates the status of a session.
func (r *PickPackRepository) UpdateSessionStatus(ctx context.Context, tx pgx.Tx, id uuid.UUID, status string) error {
	ct, err := tx.Exec(ctx,
		`UPDATE pick_pack_sessions SET status = $1, updated_at = NOW() WHERE id = $2`,
		status, id,
	)
	if err != nil {
		return fmt.Errorf("update pick_pack_session status: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("pick_pack_session not found")
	}
	return nil
}

// CompleteSession marks a session as completed with a timestamp.
func (r *PickPackRepository) CompleteSession(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	ct, err := tx.Exec(ctx,
		`UPDATE pick_pack_sessions SET status = 'completed', completed_at = NOW(), updated_at = NOW() WHERE id = $1`,
		id,
	)
	if err != nil {
		return fmt.Errorf("complete pick_pack_session: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("pick_pack_session not found")
	}
	return nil
}

// CreateItem inserts a new pick-pack item.
func (r *PickPackRepository) CreateItem(ctx context.Context, tx pgx.Tx, item *model.PickPackItem) error {
	return tx.QueryRow(ctx,
		`INSERT INTO pick_pack_items (id, tenant_id, session_id, order_id, product_id, sku, product_name, quantity_required, pick_location)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 RETURNING created_at`,
		item.ID, item.TenantID, item.SessionID, item.OrderID,
		item.ProductID, item.SKU, item.ProductName, item.QuantityRequired, item.PickLocation,
	).Scan(&item.CreatedAt)
}

// ListItemsBySession returns all items for a session, sorted by pick_location for optimal walking order.
func (r *PickPackRepository) ListItemsBySession(ctx context.Context, tx pgx.Tx, sessionID uuid.UUID) ([]model.PickPackItem, error) {
	rows, err := tx.Query(ctx,
		`SELECT id, tenant_id, session_id, order_id, product_id, sku, product_name,
		        quantity_required, quantity_picked, quantity_packed,
		        pick_location, picked_at, packed_at, created_at
		 FROM pick_pack_items
		 WHERE session_id = $1
		 ORDER BY pick_location ASC NULLS LAST, product_name ASC`,
		sessionID,
	)
	if err != nil {
		return nil, fmt.Errorf("list pick_pack_items by session: %w", err)
	}
	defer rows.Close()

	var items []model.PickPackItem
	for rows.Next() {
		var item model.PickPackItem
		if err := rows.Scan(
			&item.ID, &item.TenantID, &item.SessionID, &item.OrderID,
			&item.ProductID, &item.SKU, &item.ProductName,
			&item.QuantityRequired, &item.QuantityPicked, &item.QuantityPacked,
			&item.PickLocation, &item.PickedAt, &item.PackedAt, &item.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan pick_pack_item: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// FindItemByID retrieves a single item by ID.
func (r *PickPackRepository) FindItemByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.PickPackItem, error) {
	var item model.PickPackItem
	err := tx.QueryRow(ctx,
		`SELECT id, tenant_id, session_id, order_id, product_id, sku, product_name,
		        quantity_required, quantity_picked, quantity_packed,
		        pick_location, picked_at, packed_at, created_at
		 FROM pick_pack_items WHERE id = $1`, id,
	).Scan(
		&item.ID, &item.TenantID, &item.SessionID, &item.OrderID,
		&item.ProductID, &item.SKU, &item.ProductName,
		&item.QuantityRequired, &item.QuantityPicked, &item.QuantityPacked,
		&item.PickLocation, &item.PickedAt, &item.PackedAt, &item.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find pick_pack_item by id: %w", err)
	}
	return &item, nil
}

// FindItemBySKUInSession finds items matching a SKU in a session (may return multiple for batch sessions).
func (r *PickPackRepository) FindItemBySKUInSession(ctx context.Context, tx pgx.Tx, sessionID uuid.UUID, sku string) ([]model.PickPackItem, error) {
	rows, err := tx.Query(ctx,
		`SELECT id, tenant_id, session_id, order_id, product_id, sku, product_name,
		        quantity_required, quantity_picked, quantity_packed,
		        pick_location, picked_at, packed_at, created_at
		 FROM pick_pack_items
		 WHERE session_id = $1 AND sku = $2
		 ORDER BY quantity_picked ASC, created_at ASC`,
		sessionID, sku,
	)
	if err != nil {
		return nil, fmt.Errorf("find pick_pack_items by sku in session: %w", err)
	}
	defer rows.Close()

	var items []model.PickPackItem
	for rows.Next() {
		var item model.PickPackItem
		if err := rows.Scan(
			&item.ID, &item.TenantID, &item.SessionID, &item.OrderID,
			&item.ProductID, &item.SKU, &item.ProductName,
			&item.QuantityRequired, &item.QuantityPicked, &item.QuantityPacked,
			&item.PickLocation, &item.PickedAt, &item.PackedAt, &item.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan pick_pack_item: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// IncrementPicked increments the quantity_picked of an item and sets picked_at when fully picked.
func (r *PickPackRepository) IncrementPicked(ctx context.Context, tx pgx.Tx, id uuid.UUID, delta int) (*model.PickPackItem, error) {
	var item model.PickPackItem
	err := tx.QueryRow(ctx,
		`UPDATE pick_pack_items
		 SET quantity_picked = quantity_picked + $1,
		     picked_at = CASE WHEN quantity_picked + $1 >= quantity_required THEN COALESCE(picked_at, NOW()) ELSE picked_at END
		 WHERE id = $2
		 RETURNING id, tenant_id, session_id, order_id, product_id, sku, product_name,
		           quantity_required, quantity_picked, quantity_packed,
		           pick_location, picked_at, packed_at, created_at`,
		delta, id,
	).Scan(
		&item.ID, &item.TenantID, &item.SessionID, &item.OrderID,
		&item.ProductID, &item.SKU, &item.ProductName,
		&item.QuantityRequired, &item.QuantityPicked, &item.QuantityPacked,
		&item.PickLocation, &item.PickedAt, &item.PackedAt, &item.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("increment pick_pack_item picked: %w", err)
	}
	return &item, nil
}

// IncrementPacked increments the quantity_packed of an item and sets packed_at when fully packed.
func (r *PickPackRepository) IncrementPacked(ctx context.Context, tx pgx.Tx, id uuid.UUID, delta int) (*model.PickPackItem, error) {
	var item model.PickPackItem
	err := tx.QueryRow(ctx,
		`UPDATE pick_pack_items
		 SET quantity_packed = quantity_packed + $1,
		     packed_at = CASE WHEN quantity_packed + $1 >= quantity_required THEN COALESCE(packed_at, NOW()) ELSE packed_at END
		 WHERE id = $2
		 RETURNING id, tenant_id, session_id, order_id, product_id, sku, product_name,
		           quantity_required, quantity_picked, quantity_packed,
		           pick_location, picked_at, packed_at, created_at`,
		delta, id,
	).Scan(
		&item.ID, &item.TenantID, &item.SessionID, &item.OrderID,
		&item.ProductID, &item.SKU, &item.ProductName,
		&item.QuantityRequired, &item.QuantityPicked, &item.QuantityPacked,
		&item.PickLocation, &item.PickedAt, &item.PackedAt, &item.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("increment pick_pack_item packed: %w", err)
	}
	return &item, nil
}

// GetSessionStats computes aggregate statistics for a session.
func (r *PickPackRepository) GetSessionStats(ctx context.Context, tx pgx.Tx, sessionID uuid.UUID) (*model.PickPackStats, error) {
	var stats model.PickPackStats
	err := tx.QueryRow(ctx,
		`SELECT
			COUNT(*) AS total_items,
			COALESCE(SUM(quantity_picked), 0) AS total_picked,
			COALESCE(SUM(quantity_packed), 0) AS total_packed,
			COALESCE(SUM(quantity_required), 0) AS total_required,
			COUNT(DISTINCT order_id) AS order_count,
			COALESCE(BOOL_AND(quantity_picked >= quantity_required), false) AS all_picked,
			COALESCE(BOOL_AND(quantity_packed >= quantity_required), false) AS all_packed
		 FROM pick_pack_items
		 WHERE session_id = $1`,
		sessionID,
	).Scan(
		&stats.TotalItems, &stats.TotalPicked, &stats.TotalPacked,
		&stats.TotalRequired, &stats.OrderCount,
		&stats.AllPicked, &stats.AllPacked,
	)
	if err != nil {
		return nil, fmt.Errorf("get pick_pack_session stats: %w", err)
	}
	return &stats, nil
}

// GetDistinctOrderIDs returns the distinct order IDs for a session.
func (r *PickPackRepository) GetDistinctOrderIDs(ctx context.Context, tx pgx.Tx, sessionID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := tx.Query(ctx,
		`SELECT DISTINCT order_id FROM pick_pack_items WHERE session_id = $1`,
		sessionID,
	)
	if err != nil {
		return nil, fmt.Errorf("get distinct order_ids: %w", err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan order_id: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
