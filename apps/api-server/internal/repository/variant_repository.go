package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// VariantRepository implements VariantRepo.
type VariantRepository struct{}

func NewVariantRepository() *VariantRepository {
	return &VariantRepository{}
}

func (r *VariantRepository) List(ctx context.Context, tx pgx.Tx, filter model.VariantListFilter) ([]model.ProductVariant, int, error) {
	var conditions []string
	var args []any
	argIdx := 1

	conditions = append(conditions, fmt.Sprintf("product_id = $%d", argIdx))
	args = append(args, filter.ProductID)
	argIdx++

	if filter.Active != nil {
		conditions = append(conditions, fmt.Sprintf("active = $%d", argIdx))
		args = append(args, *filter.Active)
		argIdx++
	}

	where := "WHERE " + strings.Join(conditions, " AND ")

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM product_variants %s", where)
	var total int
	if err := tx.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count variants: %w", err)
	}

	allowedSortColumns := map[string]string{
		"created_at": "created_at",
		"name":       "name",
		"position":   "position",
		"sku":        "sku",
	}
	orderByClause := model.BuildOrderByClause(filter.SortBy, filter.SortOrder, allowedSortColumns)
	// Default sort by position for variants
	if filter.SortBy == "" {
		orderByClause = "ORDER BY position ASC, created_at ASC"
	}

	query := fmt.Sprintf(
		`SELECT id, tenant_id, product_id, sku, ean, name, attributes, price_override, stock_quantity,
		        weight, image_url, position, active, created_at, updated_at
		 FROM product_variants %s %s LIMIT $%d OFFSET $%d`,
		where, orderByClause, argIdx, argIdx+1,
	)
	args = append(args, filter.Limit, filter.Offset)

	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list variants: %w", err)
	}
	defer rows.Close()

	var variants []model.ProductVariant
	for rows.Next() {
		var v model.ProductVariant
		if err := rows.Scan(
			&v.ID, &v.TenantID, &v.ProductID, &v.SKU, &v.EAN, &v.Name,
			&v.Attributes, &v.PriceOverride, &v.StockQuantity,
			&v.Weight, &v.ImageURL, &v.Position, &v.Active,
			&v.CreatedAt, &v.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan variant: %w", err)
		}
		variants = append(variants, v)
	}
	return variants, total, rows.Err()
}

func (r *VariantRepository) FindByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.ProductVariant, error) {
	var v model.ProductVariant
	err := tx.QueryRow(ctx,
		`SELECT id, tenant_id, product_id, sku, ean, name, attributes, price_override, stock_quantity,
		        weight, image_url, position, active, created_at, updated_at
		 FROM product_variants WHERE id = $1`, id,
	).Scan(
		&v.ID, &v.TenantID, &v.ProductID, &v.SKU, &v.EAN, &v.Name,
		&v.Attributes, &v.PriceOverride, &v.StockQuantity,
		&v.Weight, &v.ImageURL, &v.Position, &v.Active,
		&v.CreatedAt, &v.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find variant by id: %w", err)
	}
	return &v, nil
}

func (r *VariantRepository) FindBySKU(ctx context.Context, tx pgx.Tx, sku string) ([]model.ProductVariant, error) {
	rows, err := tx.Query(ctx,
		`SELECT id, tenant_id, product_id, sku, ean, name, attributes, price_override, stock_quantity,
		        weight, image_url, position, active, created_at, updated_at
		 FROM product_variants WHERE sku = $1`, sku,
	)
	if err != nil {
		return nil, fmt.Errorf("find variants by sku: %w", err)
	}
	defer rows.Close()

	var variants []model.ProductVariant
	for rows.Next() {
		var v model.ProductVariant
		if err := rows.Scan(
			&v.ID, &v.TenantID, &v.ProductID, &v.SKU, &v.EAN, &v.Name,
			&v.Attributes, &v.PriceOverride, &v.StockQuantity,
			&v.Weight, &v.ImageURL, &v.Position, &v.Active,
			&v.CreatedAt, &v.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan variant by sku: %w", err)
		}
		variants = append(variants, v)
	}
	return variants, rows.Err()
}

func (r *VariantRepository) FindByEAN(ctx context.Context, tx pgx.Tx, ean string) ([]model.ProductVariant, error) {
	rows, err := tx.Query(ctx,
		`SELECT id, tenant_id, product_id, sku, ean, name, attributes, price_override, stock_quantity,
		        weight, image_url, position, active, created_at, updated_at
		 FROM product_variants WHERE ean = $1`, ean,
	)
	if err != nil {
		return nil, fmt.Errorf("find variants by ean: %w", err)
	}
	defer rows.Close()

	var variants []model.ProductVariant
	for rows.Next() {
		var v model.ProductVariant
		if err := rows.Scan(
			&v.ID, &v.TenantID, &v.ProductID, &v.SKU, &v.EAN, &v.Name,
			&v.Attributes, &v.PriceOverride, &v.StockQuantity,
			&v.Weight, &v.ImageURL, &v.Position, &v.Active,
			&v.CreatedAt, &v.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan variant by ean: %w", err)
		}
		variants = append(variants, v)
	}
	return variants, rows.Err()
}

func (r *VariantRepository) Create(ctx context.Context, tx pgx.Tx, variant *model.ProductVariant) error {
	return tx.QueryRow(ctx,
		`INSERT INTO product_variants (id, tenant_id, product_id, sku, ean, name, attributes, price_override, stock_quantity, weight, image_url, position, active)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		 RETURNING created_at, updated_at`,
		variant.ID, variant.TenantID, variant.ProductID, variant.SKU, variant.EAN,
		variant.Name, variant.Attributes, variant.PriceOverride, variant.StockQuantity,
		variant.Weight, variant.ImageURL, variant.Position, variant.Active,
	).Scan(&variant.CreatedAt, &variant.UpdatedAt)
}

func (r *VariantRepository) Update(ctx context.Context, tx pgx.Tx, id uuid.UUID, req model.UpdateVariantRequest) error {
	var setClauses []string
	var args []any
	argIdx := 1

	if req.SKU != nil {
		setClauses = append(setClauses, fmt.Sprintf("sku = $%d", argIdx))
		args = append(args, *req.SKU)
		argIdx++
	}
	if req.EAN != nil {
		setClauses = append(setClauses, fmt.Sprintf("ean = $%d", argIdx))
		args = append(args, *req.EAN)
		argIdx++
	}
	if req.Name != nil {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, *req.Name)
		argIdx++
	}
	if req.Attributes != nil {
		setClauses = append(setClauses, fmt.Sprintf("attributes = $%d", argIdx))
		args = append(args, *req.Attributes)
		argIdx++
	}
	if req.PriceOverride != nil {
		setClauses = append(setClauses, fmt.Sprintf("price_override = $%d", argIdx))
		args = append(args, *req.PriceOverride)
		argIdx++
	}
	if req.StockQuantity != nil {
		setClauses = append(setClauses, fmt.Sprintf("stock_quantity = $%d", argIdx))
		args = append(args, *req.StockQuantity)
		argIdx++
	}
	if req.Weight != nil {
		setClauses = append(setClauses, fmt.Sprintf("weight = $%d", argIdx))
		args = append(args, *req.Weight)
		argIdx++
	}
	if req.ImageURL != nil {
		setClauses = append(setClauses, fmt.Sprintf("image_url = $%d", argIdx))
		args = append(args, *req.ImageURL)
		argIdx++
	}
	if req.Position != nil {
		setClauses = append(setClauses, fmt.Sprintf("position = $%d", argIdx))
		args = append(args, *req.Position)
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
	query := fmt.Sprintf("UPDATE product_variants SET %s WHERE id = $%d",
		strings.Join(setClauses, ", "), argIdx)
	args = append(args, id)

	ct, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update variant: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("variant not found")
	}
	return nil
}

func (r *VariantRepository) Delete(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	ct, err := tx.Exec(ctx, "DELETE FROM product_variants WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete variant: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("variant not found")
	}
	return nil
}

func (r *VariantRepository) CountByProductID(ctx context.Context, tx pgx.Tx, productID uuid.UUID) (int, error) {
	var count int
	err := tx.QueryRow(ctx, "SELECT COUNT(*) FROM product_variants WHERE product_id = $1", productID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count variants by product: %w", err)
	}
	return count, nil
}
