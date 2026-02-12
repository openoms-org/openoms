package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// WarehouseRepository implements WarehouseRepo.
type WarehouseRepository struct{}

// NewWarehouseRepository creates a new WarehouseRepository.
func NewWarehouseRepository() *WarehouseRepository {
	return &WarehouseRepository{}
}

func (r *WarehouseRepository) List(ctx context.Context, tx pgx.Tx, filter model.WarehouseListFilter) ([]model.Warehouse, int, error) {
	var conditions []string
	var args []any
	argIdx := 1

	if filter.Active != nil {
		conditions = append(conditions, fmt.Sprintf("active = $%d", argIdx))
		args = append(args, *filter.Active)
		argIdx++
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM warehouses %s", where)
	if err := tx.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count warehouses: %w", err)
	}

	allowedSortColumns := map[string]string{
		"created_at": "created_at",
		"name":       "name",
	}
	orderByClause := model.BuildOrderByClause(filter.SortBy, filter.SortOrder, allowedSortColumns)

	query := fmt.Sprintf(
		`SELECT id, tenant_id, name, code, address, is_default, active, created_at, updated_at
		 FROM warehouses %s %s LIMIT $%d OFFSET $%d`,
		where, orderByClause, argIdx, argIdx+1,
	)
	args = append(args, filter.Limit, filter.Offset)

	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list warehouses: %w", err)
	}
	defer rows.Close()

	var warehouses []model.Warehouse
	for rows.Next() {
		var w model.Warehouse
		if err := rows.Scan(
			&w.ID, &w.TenantID, &w.Name, &w.Code, &w.Address,
			&w.IsDefault, &w.Active, &w.CreatedAt, &w.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan warehouse: %w", err)
		}
		warehouses = append(warehouses, w)
	}
	return warehouses, total, rows.Err()
}

func (r *WarehouseRepository) FindByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.Warehouse, error) {
	var w model.Warehouse
	err := tx.QueryRow(ctx,
		`SELECT id, tenant_id, name, code, address, is_default, active, created_at, updated_at
		 FROM warehouses WHERE id = $1`, id,
	).Scan(
		&w.ID, &w.TenantID, &w.Name, &w.Code, &w.Address,
		&w.IsDefault, &w.Active, &w.CreatedAt, &w.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find warehouse by id: %w", err)
	}
	return &w, nil
}

func (r *WarehouseRepository) Create(ctx context.Context, tx pgx.Tx, warehouse *model.Warehouse) error {
	return tx.QueryRow(ctx,
		`INSERT INTO warehouses (id, tenant_id, name, code, address, is_default, active)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING created_at, updated_at`,
		warehouse.ID, warehouse.TenantID, warehouse.Name, warehouse.Code,
		warehouse.Address, warehouse.IsDefault, warehouse.Active,
	).Scan(&warehouse.CreatedAt, &warehouse.UpdatedAt)
}

func (r *WarehouseRepository) Update(ctx context.Context, tx pgx.Tx, id uuid.UUID, req model.UpdateWarehouseRequest) error {
	var setClauses []string
	var args []any
	argIdx := 1

	if req.Name != nil {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, *req.Name)
		argIdx++
	}
	if req.Code != nil {
		setClauses = append(setClauses, fmt.Sprintf("code = $%d", argIdx))
		args = append(args, *req.Code)
		argIdx++
	}
	if req.Address != nil {
		setClauses = append(setClauses, fmt.Sprintf("address = $%d", argIdx))
		args = append(args, *req.Address)
		argIdx++
	}
	if req.IsDefault != nil {
		setClauses = append(setClauses, fmt.Sprintf("is_default = $%d", argIdx))
		args = append(args, *req.IsDefault)
		argIdx++
	}
	if req.Active != nil {
		setClauses = append(setClauses, fmt.Sprintf("active = $%d", argIdx))
		args = append(args, *req.Active)
		argIdx++
	}

	if len(setClauses) == 0 {
		return nil
	}

	setClauses = append(setClauses, "updated_at = NOW()")
	query := fmt.Sprintf("UPDATE warehouses SET %s WHERE id = $%d",
		strings.Join(setClauses, ", "), argIdx)
	args = append(args, id)

	ct, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update warehouse: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("warehouse not found")
	}
	return nil
}

func (r *WarehouseRepository) Delete(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	ct, err := tx.Exec(ctx, "DELETE FROM warehouses WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete warehouse: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("warehouse not found")
	}
	return nil
}

// WarehouseStockRepository implements WarehouseStockRepo.
type WarehouseStockRepository struct{}

// NewWarehouseStockRepository creates a new WarehouseStockRepository.
func NewWarehouseStockRepository() *WarehouseStockRepository {
	return &WarehouseStockRepository{}
}

func (r *WarehouseStockRepository) ListByWarehouse(ctx context.Context, tx pgx.Tx, warehouseID uuid.UUID, filter model.WarehouseStockListFilter) ([]model.WarehouseStock, int, error) {
	var total int
	if err := tx.QueryRow(ctx,
		"SELECT COUNT(*) FROM warehouse_stock WHERE warehouse_id = $1", warehouseID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count warehouse stock: %w", err)
	}

	rows, err := tx.Query(ctx,
		`SELECT id, tenant_id, warehouse_id, product_id, variant_id, quantity, reserved, min_stock, created_at, updated_at
		 FROM warehouse_stock WHERE warehouse_id = $1
		 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		warehouseID, filter.Limit, filter.Offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list warehouse stock: %w", err)
	}
	defer rows.Close()

	var stocks []model.WarehouseStock
	for rows.Next() {
		var s model.WarehouseStock
		if err := rows.Scan(
			&s.ID, &s.TenantID, &s.WarehouseID, &s.ProductID, &s.VariantID,
			&s.Quantity, &s.Reserved, &s.MinStock, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan warehouse stock: %w", err)
		}
		stocks = append(stocks, s)
	}
	return stocks, total, rows.Err()
}

func (r *WarehouseStockRepository) ListByProduct(ctx context.Context, tx pgx.Tx, productID uuid.UUID) ([]model.WarehouseStock, error) {
	rows, err := tx.Query(ctx,
		`SELECT id, tenant_id, warehouse_id, product_id, variant_id, quantity, reserved, min_stock, created_at, updated_at
		 FROM warehouse_stock WHERE product_id = $1
		 ORDER BY created_at DESC`,
		productID,
	)
	if err != nil {
		return nil, fmt.Errorf("list stock by product: %w", err)
	}
	defer rows.Close()

	var stocks []model.WarehouseStock
	for rows.Next() {
		var s model.WarehouseStock
		if err := rows.Scan(
			&s.ID, &s.TenantID, &s.WarehouseID, &s.ProductID, &s.VariantID,
			&s.Quantity, &s.Reserved, &s.MinStock, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan warehouse stock: %w", err)
		}
		stocks = append(stocks, s)
	}
	return stocks, rows.Err()
}

func (r *WarehouseStockRepository) Upsert(ctx context.Context, tx pgx.Tx, stock *model.WarehouseStock) error {
	return tx.QueryRow(ctx,
		`INSERT INTO warehouse_stock (id, tenant_id, warehouse_id, product_id, variant_id, quantity, reserved, min_stock)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 ON CONFLICT (warehouse_id, product_id, variant_id)
		 DO UPDATE SET quantity = EXCLUDED.quantity, reserved = EXCLUDED.reserved,
		              min_stock = EXCLUDED.min_stock, updated_at = NOW()
		 RETURNING id, created_at, updated_at`,
		stock.ID, stock.TenantID, stock.WarehouseID, stock.ProductID,
		stock.VariantID, stock.Quantity, stock.Reserved, stock.MinStock,
	).Scan(&stock.ID, &stock.CreatedAt, &stock.UpdatedAt)
}

func (r *WarehouseStockRepository) AdjustQuantity(ctx context.Context, tx pgx.Tx, warehouseID, productID uuid.UUID, variantID *uuid.UUID, delta int) error {
	if variantID != nil {
		_, err := tx.Exec(ctx,
			`INSERT INTO warehouse_stock (id, tenant_id, warehouse_id, product_id, variant_id, quantity)
			 VALUES (uuid_generate_v4(), current_setting('app.current_tenant_id')::uuid, $1, $2, $3, $4)
			 ON CONFLICT (warehouse_id, product_id, variant_id)
			 DO UPDATE SET quantity = warehouse_stock.quantity + EXCLUDED.quantity, updated_at = NOW()`,
			warehouseID, productID, *variantID, delta,
		)
		if err != nil {
			return fmt.Errorf("adjust stock quantity: %w", err)
		}
	} else {
		_, err := tx.Exec(ctx,
			`INSERT INTO warehouse_stock (id, tenant_id, warehouse_id, product_id, variant_id, quantity)
			 VALUES (uuid_generate_v4(), current_setting('app.current_tenant_id')::uuid, $1, $2, NULL, $3)
			 ON CONFLICT (warehouse_id, product_id, variant_id)
			 DO UPDATE SET quantity = warehouse_stock.quantity + EXCLUDED.quantity, updated_at = NOW()`,
			warehouseID, productID, delta,
		)
		if err != nil {
			return fmt.Errorf("adjust stock quantity: %w", err)
		}
	}
	return nil
}
