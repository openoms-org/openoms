package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// VariantRepository implements VariantRepo.
type VariantRepository struct{}

// NewVariantRepository creates a new VariantRepository.
func NewVariantRepository() *VariantRepository {
	return &VariantRepository{}
}

// variantColumns is the canonical column list for SELECTing a product variant row.
const variantColumns = "id, tenant_id, product_id, sku, ean, name, attributes, price_override, stock_quantity, weight, image_url, position, active, created_at, updated_at"

// scanVariant scans a single product variant row in variantColumns order.
func scanVariant(row pgx.Row) (model.ProductVariant, error) {
	var v model.ProductVariant
	err := row.Scan(
		&v.ID, &v.TenantID, &v.ProductID, &v.SKU, &v.EAN, &v.Name,
		&v.Attributes, &v.PriceOverride, &v.StockQuantity,
		&v.Weight, &v.ImageURL, &v.Position, &v.Active,
		&v.CreatedAt, &v.UpdatedAt,
	)
	return v, err
}

// List returns a paginated list of product variants matching the filter.
func (r *VariantRepository) List(ctx context.Context, tx pgx.Tx, filter model.VariantListFilter) ([]model.ProductVariant, int, error) {
	qb := NewQueryBuilder()

	qb.Add("product_id = $%d", filter.ProductID)
	if filter.Active != nil {
		qb.Add("active = $%d", *filter.Active)
	}

	where := qb.WhereClause()

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM product_variants %s", where)
	var total int
	if err := tx.QueryRow(ctx, countQuery, qb.Args()...).Scan(&total); err != nil {
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

	argIdx := qb.AddArgs(filter.Limit, filter.Offset)
	query := fmt.Sprintf(
		"SELECT "+variantColumns+" FROM product_variants %s %s LIMIT $%d OFFSET $%d",
		where, orderByClause, argIdx, argIdx+1,
	)

	rows, err := tx.Query(ctx, query, qb.Args()...)
	if err != nil {
		return nil, 0, fmt.Errorf("list variants: %w", err)
	}
	defer rows.Close()

	var variants []model.ProductVariant
	for rows.Next() {
		v, err := scanVariant(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan variant: %w", err)
		}
		variants = append(variants, v)
	}
	return variants, total, rows.Err()
}

// FindByID returns a product variant by its ID.
func (r *VariantRepository) FindByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.ProductVariant, error) {
	v, err := scanVariant(tx.QueryRow(ctx,
		"SELECT "+variantColumns+" FROM product_variants WHERE id = $1", id,
	))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find variant by id: %w", err)
	}
	return &v, nil
}

// FindBySKU returns all product variants with the given SKU.
func (r *VariantRepository) FindBySKU(ctx context.Context, tx pgx.Tx, sku string) ([]model.ProductVariant, error) {
	rows, err := tx.Query(ctx,
		"SELECT "+variantColumns+" FROM product_variants WHERE sku = $1", sku,
	)
	if err != nil {
		return nil, fmt.Errorf("find variants by sku: %w", err)
	}
	defer rows.Close()

	var variants []model.ProductVariant
	for rows.Next() {
		v, err := scanVariant(rows)
		if err != nil {
			return nil, fmt.Errorf("scan variant by sku: %w", err)
		}
		variants = append(variants, v)
	}
	return variants, rows.Err()
}

// FindByEAN returns all product variants with the given EAN barcode.
func (r *VariantRepository) FindByEAN(ctx context.Context, tx pgx.Tx, ean string) ([]model.ProductVariant, error) {
	rows, err := tx.Query(ctx,
		"SELECT "+variantColumns+" FROM product_variants WHERE ean = $1", ean,
	)
	if err != nil {
		return nil, fmt.Errorf("find variants by ean: %w", err)
	}
	defer rows.Close()

	var variants []model.ProductVariant
	for rows.Next() {
		v, err := scanVariant(rows)
		if err != nil {
			return nil, fmt.Errorf("scan variant by ean: %w", err)
		}
		variants = append(variants, v)
	}
	return variants, rows.Err()
}

// Create inserts a new product variant.
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

// Update applies partial updates to a product variant.
func (r *VariantRepository) Update(ctx context.Context, tx pgx.Tx, id uuid.UUID, req model.UpdateVariantRequest) error {
	ub := NewUpdateBuilder()
	SetPtr(ub, "sku", req.SKU)
	SetPtr(ub, "ean", req.EAN)
	SetPtr(ub, "name", req.Name)
	SetPtr(ub, "attributes", req.Attributes)
	SetPtr(ub, "price_override", req.PriceOverride)
	SetPtr(ub, "stock_quantity", req.StockQuantity)
	SetPtr(ub, "weight", req.Weight)
	SetPtr(ub, "image_url", req.ImageURL)
	SetPtr(ub, "position", req.Position)
	SetPtr(ub, "active", req.Active)

	if ub.IsEmpty() {
		return nil
	}

	ub.SetRaw("updated_at = NOW()")
	query := fmt.Sprintf("UPDATE product_variants SET %s WHERE id = $%d",
		ub.SetClause(), ub.NextArgIdx())
	args := append(ub.Args(), id)

	ct, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update variant: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("variant not found")
	}
	return nil
}

// Delete removes a product variant by its ID.
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

// CountByProductID returns the number of variants for the given product.
func (r *VariantRepository) CountByProductID(ctx context.Context, tx pgx.Tx, productID uuid.UUID) (int, error) {
	var count int
	err := tx.QueryRow(ctx, "SELECT COUNT(*) FROM product_variants WHERE product_id = $1", productID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count variants by product: %w", err)
	}
	return count, nil
}
