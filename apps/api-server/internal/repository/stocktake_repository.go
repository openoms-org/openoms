package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// StocktakeRepository implements StocktakeRepo.
type StocktakeRepository struct{}

// NewStocktakeRepository creates a new StocktakeRepository.
func NewStocktakeRepository() *StocktakeRepository {
	return &StocktakeRepository{}
}

func (r *StocktakeRepository) Create(ctx context.Context, tx pgx.Tx, stocktake *model.Stocktake) error {
	return tx.QueryRow(ctx,
		`INSERT INTO stocktakes (id, tenant_id, warehouse_id, name, status, notes, created_by)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING created_at, updated_at`,
		stocktake.ID, stocktake.TenantID, stocktake.WarehouseID,
		stocktake.Name, stocktake.Status, stocktake.Notes, stocktake.CreatedBy,
	).Scan(&stocktake.CreatedAt, &stocktake.UpdatedAt)
}

func (r *StocktakeRepository) FindByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.Stocktake, error) {
	var s model.Stocktake
	err := tx.QueryRow(ctx,
		`SELECT id, tenant_id, warehouse_id, name, status, started_at, completed_at,
		        notes, created_by, created_at, updated_at
		 FROM stocktakes WHERE id = $1`, id,
	).Scan(
		&s.ID, &s.TenantID, &s.WarehouseID, &s.Name, &s.Status,
		&s.StartedAt, &s.CompletedAt, &s.Notes, &s.CreatedBy,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find stocktake by id: %w", err)
	}
	return &s, nil
}

func (r *StocktakeRepository) List(ctx context.Context, tx pgx.Tx, filter model.StocktakeListFilter) ([]model.Stocktake, int, error) {
	var conditions []string
	var args []any
	argIdx := 1

	if filter.WarehouseID != nil {
		conditions = append(conditions, fmt.Sprintf("warehouse_id = $%d", argIdx))
		args = append(args, *filter.WarehouseID)
		argIdx++
	}
	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *filter.Status)
		argIdx++
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM stocktakes %s", where)
	if err := tx.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count stocktakes: %w", err)
	}

	allowedSortColumns := map[string]string{
		"created_at": "created_at",
		"name":       "name",
		"status":     "status",
	}
	orderByClause := model.BuildOrderByClause(filter.SortBy, filter.SortOrder, allowedSortColumns)

	query := fmt.Sprintf(
		`SELECT id, tenant_id, warehouse_id, name, status, started_at, completed_at,
		        notes, created_by, created_at, updated_at
		 FROM stocktakes %s %s LIMIT $%d OFFSET $%d`,
		where, orderByClause, argIdx, argIdx+1,
	)
	args = append(args, filter.Limit, filter.Offset)

	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list stocktakes: %w", err)
	}
	defer rows.Close()

	var stocktakes []model.Stocktake
	for rows.Next() {
		var s model.Stocktake
		if err := rows.Scan(
			&s.ID, &s.TenantID, &s.WarehouseID, &s.Name, &s.Status,
			&s.StartedAt, &s.CompletedAt, &s.Notes, &s.CreatedBy,
			&s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan stocktake: %w", err)
		}
		stocktakes = append(stocktakes, s)
	}
	return stocktakes, total, rows.Err()
}

func (r *StocktakeRepository) UpdateStatus(ctx context.Context, tx pgx.Tx, id uuid.UUID, status string) error {
	ct, err := tx.Exec(ctx,
		`UPDATE stocktakes SET status = $1, updated_at = NOW() WHERE id = $2`,
		status, id,
	)
	if err != nil {
		return fmt.Errorf("update stocktake status: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("stocktake not found")
	}
	return nil
}

func (r *StocktakeRepository) SetStartedAt(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	ct, err := tx.Exec(ctx,
		`UPDATE stocktakes SET started_at = NOW(), status = 'in_progress', updated_at = NOW() WHERE id = $1`,
		id,
	)
	if err != nil {
		return fmt.Errorf("set stocktake started_at: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("stocktake not found")
	}
	return nil
}

func (r *StocktakeRepository) SetCompletedAt(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	ct, err := tx.Exec(ctx,
		`UPDATE stocktakes SET completed_at = NOW(), status = 'completed', updated_at = NOW() WHERE id = $1`,
		id,
	)
	if err != nil {
		return fmt.Errorf("set stocktake completed_at: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("stocktake not found")
	}
	return nil
}

func (r *StocktakeRepository) Delete(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	ct, err := tx.Exec(ctx, "DELETE FROM stocktakes WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete stocktake: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("stocktake not found")
	}
	return nil
}

// StocktakeItemRepository implements StocktakeItemRepo.
type StocktakeItemRepository struct{}

// NewStocktakeItemRepository creates a new StocktakeItemRepository.
func NewStocktakeItemRepository() *StocktakeItemRepository {
	return &StocktakeItemRepository{}
}

func (r *StocktakeItemRepository) CreateBulk(ctx context.Context, tx pgx.Tx, items []model.StocktakeItem) error {
	if len(items) == 0 {
		return nil
	}

	for _, item := range items {
		_, err := tx.Exec(ctx,
			`INSERT INTO stocktake_items (id, tenant_id, stocktake_id, product_id, expected_quantity)
			 VALUES ($1, $2, $3, $4, $5)`,
			item.ID, item.TenantID, item.StocktakeID, item.ProductID, item.ExpectedQuantity,
		)
		if err != nil {
			return fmt.Errorf("insert stocktake_item: %w", err)
		}
	}
	return nil
}

func (r *StocktakeItemRepository) List(ctx context.Context, tx pgx.Tx, stocktakeID uuid.UUID, filter model.StocktakeItemListFilter) ([]model.StocktakeItem, int, error) {
	var conditions []string
	var args []any
	argIdx := 1

	conditions = append(conditions, fmt.Sprintf("si.stocktake_id = $%d", argIdx))
	args = append(args, stocktakeID)
	argIdx++

	switch filter.Filter {
	case "uncounted":
		conditions = append(conditions, "si.counted_quantity IS NULL")
	case "discrepancies":
		conditions = append(conditions, "si.counted_quantity IS NOT NULL AND si.difference != 0")
	}

	where := "WHERE " + strings.Join(conditions, " AND ")

	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM stocktake_items si %s", where)
	if err := tx.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count stocktake_items: %w", err)
	}

	query := fmt.Sprintf(
		`SELECT si.id, si.tenant_id, si.stocktake_id, si.product_id,
		        si.expected_quantity, si.counted_quantity, si.difference,
		        si.notes, si.counted_at, si.counted_by, si.created_at,
		        p.name, p.sku
		 FROM stocktake_items si
		 LEFT JOIN products p ON p.id = si.product_id
		 %s
		 ORDER BY p.name ASC NULLS LAST
		 LIMIT $%d OFFSET $%d`,
		where, argIdx, argIdx+1,
	)
	args = append(args, filter.Limit, filter.Offset)

	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list stocktake_items: %w", err)
	}
	defer rows.Close()

	var items []model.StocktakeItem
	for rows.Next() {
		var item model.StocktakeItem
		if err := rows.Scan(
			&item.ID, &item.TenantID, &item.StocktakeID, &item.ProductID,
			&item.ExpectedQuantity, &item.CountedQuantity, &item.Difference,
			&item.Notes, &item.CountedAt, &item.CountedBy, &item.CreatedAt,
			&item.ProductName, &item.ProductSKU,
		); err != nil {
			return nil, 0, fmt.Errorf("scan stocktake_item: %w", err)
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (r *StocktakeItemRepository) FindByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.StocktakeItem, error) {
	var item model.StocktakeItem
	err := tx.QueryRow(ctx,
		`SELECT id, tenant_id, stocktake_id, product_id,
		        expected_quantity, counted_quantity, difference,
		        notes, counted_at, counted_by, created_at
		 FROM stocktake_items WHERE id = $1`, id,
	).Scan(
		&item.ID, &item.TenantID, &item.StocktakeID, &item.ProductID,
		&item.ExpectedQuantity, &item.CountedQuantity, &item.Difference,
		&item.Notes, &item.CountedAt, &item.CountedBy, &item.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find stocktake_item by id: %w", err)
	}
	return &item, nil
}

func (r *StocktakeItemRepository) UpdateCount(ctx context.Context, tx pgx.Tx, itemID uuid.UUID, countedQty int, notes *string, countedBy uuid.UUID) error {
	ct, err := tx.Exec(ctx,
		`UPDATE stocktake_items
		 SET counted_quantity = $1, notes = $2, counted_at = NOW(), counted_by = $3
		 WHERE id = $4`,
		countedQty, notes, countedBy, itemID,
	)
	if err != nil {
		return fmt.Errorf("update stocktake_item count: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("stocktake_item not found")
	}
	return nil
}

func (r *StocktakeItemRepository) GetStats(ctx context.Context, tx pgx.Tx, stocktakeID uuid.UUID) (*model.StocktakeStats, error) {
	var stats model.StocktakeStats
	err := tx.QueryRow(ctx,
		`SELECT
			COUNT(*) AS total_items,
			COUNT(counted_quantity) AS counted_items,
			COUNT(*) FILTER (WHERE counted_quantity IS NOT NULL AND difference != 0) AS discrepancies,
			COUNT(*) FILTER (WHERE counted_quantity IS NOT NULL AND difference > 0) AS surplus_count,
			COUNT(*) FILTER (WHERE counted_quantity IS NOT NULL AND difference < 0) AS shortage_count
		 FROM stocktake_items
		 WHERE stocktake_id = $1`,
		stocktakeID,
	).Scan(&stats.TotalItems, &stats.CountedItems, &stats.Discrepancies, &stats.SurplusCount, &stats.ShortageCount)
	if err != nil {
		return nil, fmt.Errorf("get stocktake stats: %w", err)
	}
	return &stats, nil
}

func (r *StocktakeItemRepository) ListDiscrepancies(ctx context.Context, tx pgx.Tx, stocktakeID uuid.UUID) ([]model.StocktakeItem, error) {
	rows, err := tx.Query(ctx,
		`SELECT si.id, si.tenant_id, si.stocktake_id, si.product_id,
		        si.expected_quantity, si.counted_quantity, si.difference,
		        si.notes, si.counted_at, si.counted_by, si.created_at,
		        p.name, p.sku
		 FROM stocktake_items si
		 LEFT JOIN products p ON p.id = si.product_id
		 WHERE si.stocktake_id = $1 AND si.counted_quantity IS NOT NULL AND si.difference != 0
		 ORDER BY p.name ASC NULLS LAST`,
		stocktakeID,
	)
	if err != nil {
		return nil, fmt.Errorf("list stocktake discrepancies: %w", err)
	}
	defer rows.Close()

	var items []model.StocktakeItem
	for rows.Next() {
		var item model.StocktakeItem
		if err := rows.Scan(
			&item.ID, &item.TenantID, &item.StocktakeID, &item.ProductID,
			&item.ExpectedQuantity, &item.CountedQuantity, &item.Difference,
			&item.Notes, &item.CountedAt, &item.CountedBy, &item.CreatedAt,
			&item.ProductName, &item.ProductSKU,
		); err != nil {
			return nil, fmt.Errorf("scan stocktake_item: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
