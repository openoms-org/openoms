package repository

import (
	"context"
	"fmt"

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

// warehouseColumns is the canonical column list for SELECTing a warehouse row.
const warehouseColumns = "id, tenant_id, name, code, address, is_default, active, created_at, updated_at"

// scanWarehouse scans a single warehouse row in warehouseColumns order.
func scanWarehouse(row pgx.Row) (model.Warehouse, error) {
	var w model.Warehouse
	err := row.Scan(
		&w.ID, &w.TenantID, &w.Name, &w.Code, &w.Address,
		&w.IsDefault, &w.Active, &w.CreatedAt, &w.UpdatedAt,
	)
	return w, err
}

// List returns a paginated list of warehouses matching the filter.
func (r *WarehouseRepository) List(ctx context.Context, tx pgx.Tx, filter model.WarehouseListFilter) ([]model.Warehouse, int, error) {
	qb := NewQueryBuilder()
	if filter.Active != nil {
		qb.Add("active = $%d", *filter.Active)
	}
	where := qb.WhereClause()

	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM warehouses %s", where)
	if err := tx.QueryRow(ctx, countQuery, qb.Args()...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count warehouses: %w", err)
	}

	allowedSortColumns := map[string]string{
		"created_at": "created_at",
		"name":       "name",
	}
	orderByClause := model.BuildOrderByClause(filter.SortBy, filter.SortOrder, allowedSortColumns)

	limitIdx := qb.AddArgs(filter.Limit, filter.Offset)
	query := fmt.Sprintf(
		"SELECT "+warehouseColumns+" FROM warehouses %s %s LIMIT $%d OFFSET $%d",
		where, orderByClause, limitIdx, limitIdx+1,
	)

	rows, err := tx.Query(ctx, query, qb.Args()...)
	if err != nil {
		return nil, 0, fmt.Errorf("list warehouses: %w", err)
	}
	defer rows.Close()

	var warehouses []model.Warehouse
	for rows.Next() {
		w, err := scanWarehouse(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan warehouse: %w", err)
		}
		warehouses = append(warehouses, w)
	}
	return warehouses, total, rows.Err()
}

// FindByID returns a warehouse by its ID.
func (r *WarehouseRepository) FindByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.Warehouse, error) {
	w, err := scanWarehouse(tx.QueryRow(ctx,
		"SELECT "+warehouseColumns+" FROM warehouses WHERE id = $1", id,
	))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find warehouse by id: %w", err)
	}
	return &w, nil
}

// FindDefault returns the default active warehouse for the current tenant.
func (r *WarehouseRepository) FindDefault(ctx context.Context, tx pgx.Tx) (*model.Warehouse, error) {
	w, err := scanWarehouse(tx.QueryRow(ctx,
		"SELECT "+warehouseColumns+" FROM warehouses WHERE is_default = true AND active = true ORDER BY created_at ASC, id ASC LIMIT 1",
	))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find default warehouse: %w", err)
	}
	return &w, nil
}

// Create inserts a new warehouse.
func (r *WarehouseRepository) Create(ctx context.Context, tx pgx.Tx, warehouse *model.Warehouse) error {
	return tx.QueryRow(ctx,
		`INSERT INTO warehouses (id, tenant_id, name, code, address, is_default, active)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING created_at, updated_at`,
		warehouse.ID, warehouse.TenantID, warehouse.Name, warehouse.Code,
		warehouse.Address, warehouse.IsDefault, warehouse.Active,
	).Scan(&warehouse.CreatedAt, &warehouse.UpdatedAt)
}

// Update applies partial updates to a warehouse.
func (r *WarehouseRepository) Update(ctx context.Context, tx pgx.Tx, id uuid.UUID, req model.UpdateWarehouseRequest) error {
	ub := NewUpdateBuilder()
	SetPtr(ub, "name", req.Name)
	SetPtr(ub, "code", req.Code)
	SetPtr(ub, "address", req.Address)
	SetPtr(ub, "is_default", req.IsDefault)
	SetPtr(ub, "active", req.Active)

	if ub.IsEmpty() {
		return nil
	}

	ub.SetRaw("updated_at = NOW()")
	query := fmt.Sprintf("UPDATE warehouses SET %s WHERE id = $%d",
		ub.SetClause(), ub.NextArgIdx())
	args := append(ub.Args(), id)

	ct, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update warehouse: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("warehouse not found")
	}
	return nil
}

// Delete removes a warehouse by its ID.
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

// warehouseStockColumns is the canonical column list for SELECTing a warehouse stock row.
const warehouseStockColumns = "id, tenant_id, warehouse_id, product_id, variant_id, quantity, reserved, min_stock, created_at, updated_at"

// scanWarehouseStock scans a single warehouse stock row in warehouseStockColumns order.
func scanWarehouseStock(row pgx.Row) (model.WarehouseStock, error) {
	var s model.WarehouseStock
	err := row.Scan(
		&s.ID, &s.TenantID, &s.WarehouseID, &s.ProductID, &s.VariantID,
		&s.Quantity, &s.Reserved, &s.MinStock, &s.CreatedAt, &s.UpdatedAt,
	)
	return s, err
}

// ListByWarehouse returns paginated stock records for the given warehouse.
func (r *WarehouseStockRepository) ListByWarehouse(ctx context.Context, tx pgx.Tx, warehouseID uuid.UUID, filter model.WarehouseStockListFilter) ([]model.WarehouseStock, int, error) {
	var total int
	if err := tx.QueryRow(ctx,
		"SELECT COUNT(*) FROM warehouse_stock WHERE warehouse_id = $1", warehouseID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count warehouse stock: %w", err)
	}

	rows, err := tx.Query(ctx,
		"SELECT "+warehouseStockColumns+" FROM warehouse_stock WHERE warehouse_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3",
		warehouseID, filter.Limit, filter.Offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list warehouse stock: %w", err)
	}
	defer rows.Close()

	var stocks []model.WarehouseStock
	for rows.Next() {
		s, err := scanWarehouseStock(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan warehouse stock: %w", err)
		}
		stocks = append(stocks, s)
	}
	return stocks, total, rows.Err()
}

// ListByProduct returns all stock records for the given product.
func (r *WarehouseStockRepository) ListByProduct(ctx context.Context, tx pgx.Tx, productID uuid.UUID) ([]model.WarehouseStock, error) {
	rows, err := tx.Query(ctx,
		"SELECT "+warehouseStockColumns+" FROM warehouse_stock WHERE product_id = $1 ORDER BY created_at DESC",
		productID,
	)
	if err != nil {
		return nil, fmt.Errorf("list stock by product: %w", err)
	}
	defer rows.Close()

	var stocks []model.WarehouseStock
	for rows.Next() {
		s, err := scanWarehouseStock(rows)
		if err != nil {
			return nil, fmt.Errorf("scan warehouse stock: %w", err)
		}
		stocks = append(stocks, s)
	}
	return stocks, rows.Err()
}

// Upsert inserts or updates a warehouse stock record.
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

// AdjustQuantity adjusts the stock quantity for a product in a warehouse by the given delta.
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
