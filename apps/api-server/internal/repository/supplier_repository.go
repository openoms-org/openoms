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

type SupplierRepository struct{}

func NewSupplierRepository() *SupplierRepository {
	return &SupplierRepository{}
}

func (r *SupplierRepository) List(ctx context.Context, tx pgx.Tx, filter model.SupplierListFilter) ([]model.Supplier, int, error) {
	var conditions []string
	var args []any
	argIdx := 1

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
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM suppliers %s", where)
	if err := tx.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count suppliers: %w", err)
	}

	allowedSortColumns := map[string]string{
		"created_at": "created_at",
		"name":       "name",
		"status":     "status",
	}
	orderByClause := model.BuildOrderByClause(filter.SortBy, filter.SortOrder, allowedSortColumns)

	query := fmt.Sprintf(
		`SELECT id, tenant_id, name, code, feed_url, feed_format, status, settings,
		        last_sync_at, error_message, created_at, updated_at
		 FROM suppliers %s %s LIMIT $%d OFFSET $%d`,
		where, orderByClause, argIdx, argIdx+1,
	)
	args = append(args, filter.Limit, filter.Offset)

	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list suppliers: %w", err)
	}
	defer rows.Close()

	var suppliers []model.Supplier
	for rows.Next() {
		var s model.Supplier
		if err := rows.Scan(
			&s.ID, &s.TenantID, &s.Name, &s.Code, &s.FeedURL, &s.FeedFormat,
			&s.Status, &s.Settings, &s.LastSyncAt, &s.ErrorMessage,
			&s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan supplier: %w", err)
		}
		suppliers = append(suppliers, s)
	}
	return suppliers, total, rows.Err()
}

func (r *SupplierRepository) FindByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.Supplier, error) {
	var s model.Supplier
	err := tx.QueryRow(ctx,
		`SELECT id, tenant_id, name, code, feed_url, feed_format, status, settings,
		        last_sync_at, error_message, created_at, updated_at
		 FROM suppliers WHERE id = $1`, id,
	).Scan(
		&s.ID, &s.TenantID, &s.Name, &s.Code, &s.FeedURL, &s.FeedFormat,
		&s.Status, &s.Settings, &s.LastSyncAt, &s.ErrorMessage,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find supplier by id: %w", err)
	}
	return &s, nil
}

func (r *SupplierRepository) Create(ctx context.Context, tx pgx.Tx, supplier *model.Supplier) error {
	return tx.QueryRow(ctx,
		`INSERT INTO suppliers (id, tenant_id, name, code, feed_url, feed_format, status, settings)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 RETURNING created_at, updated_at`,
		supplier.ID, supplier.TenantID, supplier.Name, supplier.Code,
		supplier.FeedURL, supplier.FeedFormat, supplier.Status, supplier.Settings,
	).Scan(&supplier.CreatedAt, &supplier.UpdatedAt)
}

func (r *SupplierRepository) Update(ctx context.Context, tx pgx.Tx, id uuid.UUID, req model.UpdateSupplierRequest) error {
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
	if req.FeedURL != nil {
		setClauses = append(setClauses, fmt.Sprintf("feed_url = $%d", argIdx))
		args = append(args, *req.FeedURL)
		argIdx++
	}
	if req.FeedFormat != nil {
		setClauses = append(setClauses, fmt.Sprintf("feed_format = $%d", argIdx))
		args = append(args, *req.FeedFormat)
		argIdx++
	}
	if req.Status != nil {
		setClauses = append(setClauses, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *req.Status)
		argIdx++
	}
	if req.Settings != nil {
		setClauses = append(setClauses, fmt.Sprintf("settings = $%d", argIdx))
		args = append(args, *req.Settings)
		argIdx++
	}
	if req.ErrorMessage != nil {
		setClauses = append(setClauses, fmt.Sprintf("error_message = $%d", argIdx))
		args = append(args, *req.ErrorMessage)
		argIdx++
	}

	if len(setClauses) == 0 {
		return nil
	}

	setClauses = append(setClauses, "updated_at = NOW()")
	query := fmt.Sprintf("UPDATE suppliers SET %s WHERE id = $%d",
		strings.Join(setClauses, ", "), argIdx)
	args = append(args, id)

	ct, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update supplier: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("supplier not found")
	}
	return nil
}

func (r *SupplierRepository) Delete(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	ct, err := tx.Exec(ctx, "DELETE FROM suppliers WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete supplier: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("supplier not found")
	}
	return nil
}

func (r *SupplierRepository) UpdateSyncStatus(ctx context.Context, tx pgx.Tx, id uuid.UUID, lastSyncAt time.Time, errorMessage *string) error {
	_, err := tx.Exec(ctx,
		`UPDATE suppliers SET last_sync_at = $1, error_message = $2, updated_at = NOW() WHERE id = $3`,
		lastSyncAt, errorMessage, id,
	)
	if err != nil {
		return fmt.Errorf("update supplier sync status: %w", err)
	}
	return nil
}

// SupplierProductRepository

type SupplierProductRepository struct{}

func NewSupplierProductRepository() *SupplierProductRepository {
	return &SupplierProductRepository{}
}

var supplierProductColumns = `id, tenant_id, supplier_id, product_id, external_id, name, ean, sku,
	price, stock_quantity, metadata, last_synced_at, created_at, updated_at`

func scanSupplierProduct(row interface{ Scan(dest ...any) error }) (*model.SupplierProduct, error) {
	var sp model.SupplierProduct
	err := row.Scan(
		&sp.ID, &sp.TenantID, &sp.SupplierID, &sp.ProductID, &sp.ExternalID,
		&sp.Name, &sp.EAN, &sp.SKU, &sp.Price, &sp.StockQuantity,
		&sp.Metadata, &sp.LastSyncedAt, &sp.CreatedAt, &sp.UpdatedAt,
	)
	return &sp, err
}

func (r *SupplierProductRepository) List(ctx context.Context, tx pgx.Tx, filter model.SupplierProductListFilter) ([]model.SupplierProduct, int, error) {
	var conditions []string
	var args []any
	argIdx := 1

	if filter.SupplierID != nil {
		conditions = append(conditions, fmt.Sprintf("supplier_id = $%d", argIdx))
		args = append(args, *filter.SupplierID)
		argIdx++
	}
	if filter.EAN != nil {
		conditions = append(conditions, fmt.Sprintf("ean = $%d", argIdx))
		args = append(args, *filter.EAN)
		argIdx++
	}
	if filter.Linked != nil {
		if *filter.Linked {
			conditions = append(conditions, "product_id IS NOT NULL")
		} else {
			conditions = append(conditions, "product_id IS NULL")
		}
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM supplier_products %s", where)
	if err := tx.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count supplier products: %w", err)
	}

	allowedSortColumns := map[string]string{
		"created_at":     "created_at",
		"name":           "name",
		"price":          "price",
		"stock_quantity": "stock_quantity",
	}
	orderByClause := model.BuildOrderByClause(filter.SortBy, filter.SortOrder, allowedSortColumns)

	query := fmt.Sprintf(
		`SELECT %s FROM supplier_products %s %s LIMIT $%d OFFSET $%d`,
		supplierProductColumns, where, orderByClause, argIdx, argIdx+1,
	)
	args = append(args, filter.Limit, filter.Offset)

	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list supplier products: %w", err)
	}
	defer rows.Close()

	var products []model.SupplierProduct
	for rows.Next() {
		sp, err := scanSupplierProduct(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan supplier product: %w", err)
		}
		products = append(products, *sp)
	}
	return products, total, rows.Err()
}

func (r *SupplierProductRepository) FindByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.SupplierProduct, error) {
	sp, err := scanSupplierProduct(tx.QueryRow(ctx,
		fmt.Sprintf("SELECT %s FROM supplier_products WHERE id = $1", supplierProductColumns), id,
	))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find supplier product by id: %w", err)
	}
	return sp, nil
}

func (r *SupplierProductRepository) Create(ctx context.Context, tx pgx.Tx, sp *model.SupplierProduct) error {
	return tx.QueryRow(ctx,
		`INSERT INTO supplier_products (id, tenant_id, supplier_id, product_id, external_id, name, ean, sku, price, stock_quantity, metadata, last_synced_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		 RETURNING created_at, updated_at`,
		sp.ID, sp.TenantID, sp.SupplierID, sp.ProductID, sp.ExternalID,
		sp.Name, sp.EAN, sp.SKU, sp.Price, sp.StockQuantity, sp.Metadata, sp.LastSyncedAt,
	).Scan(&sp.CreatedAt, &sp.UpdatedAt)
}

func (r *SupplierProductRepository) Update(ctx context.Context, tx pgx.Tx, id uuid.UUID, name string, ean, sku *string, price *float64, stock int, metadata []byte, syncedAt *time.Time) error {
	ct, err := tx.Exec(ctx,
		`UPDATE supplier_products SET name = $1, ean = $2, sku = $3, price = $4,
		 stock_quantity = $5, metadata = $6, last_synced_at = $7, updated_at = NOW()
		 WHERE id = $8`,
		name, ean, sku, price, stock, metadata, syncedAt, id,
	)
	if err != nil {
		return fmt.Errorf("update supplier product: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("supplier product not found")
	}
	return nil
}

func (r *SupplierProductRepository) Delete(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	ct, err := tx.Exec(ctx, "DELETE FROM supplier_products WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete supplier product: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("supplier product not found")
	}
	return nil
}

func (r *SupplierProductRepository) FindByEAN(ctx context.Context, tx pgx.Tx, ean string) (*model.SupplierProduct, error) {
	sp, err := scanSupplierProduct(tx.QueryRow(ctx,
		fmt.Sprintf("SELECT %s FROM supplier_products WHERE ean = $1 LIMIT 1", supplierProductColumns), ean,
	))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find supplier product by ean: %w", err)
	}
	return sp, nil
}

func (r *SupplierProductRepository) FindBySupplierAndExternalID(ctx context.Context, tx pgx.Tx, supplierID uuid.UUID, externalID string) (*model.SupplierProduct, error) {
	sp, err := scanSupplierProduct(tx.QueryRow(ctx,
		fmt.Sprintf("SELECT %s FROM supplier_products WHERE supplier_id = $1 AND external_id = $2", supplierProductColumns),
		supplierID, externalID,
	))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find supplier product by supplier and external id: %w", err)
	}
	return sp, nil
}

func (r *SupplierProductRepository) UpsertByExternalID(ctx context.Context, tx pgx.Tx, sp *model.SupplierProduct) error {
	return tx.QueryRow(ctx,
		`INSERT INTO supplier_products (id, tenant_id, supplier_id, product_id, external_id, name, ean, sku, price, stock_quantity, metadata, last_synced_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		 ON CONFLICT (tenant_id, supplier_id, external_id)
		 DO UPDATE SET name = EXCLUDED.name, ean = EXCLUDED.ean, sku = EXCLUDED.sku,
		              price = EXCLUDED.price, stock_quantity = EXCLUDED.stock_quantity,
		              metadata = EXCLUDED.metadata, last_synced_at = EXCLUDED.last_synced_at,
		              updated_at = NOW()
		 RETURNING id, created_at, updated_at`,
		sp.ID, sp.TenantID, sp.SupplierID, sp.ProductID, sp.ExternalID,
		sp.Name, sp.EAN, sp.SKU, sp.Price, sp.StockQuantity, sp.Metadata, sp.LastSyncedAt,
	).Scan(&sp.ID, &sp.CreatedAt, &sp.UpdatedAt)
}

func (r *SupplierProductRepository) LinkToProduct(ctx context.Context, tx pgx.Tx, id uuid.UUID, productID uuid.UUID) error {
	ct, err := tx.Exec(ctx,
		`UPDATE supplier_products SET product_id = $1, updated_at = NOW() WHERE id = $2`,
		productID, id,
	)
	if err != nil {
		return fmt.Errorf("link supplier product: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("supplier product not found")
	}
	return nil
}
