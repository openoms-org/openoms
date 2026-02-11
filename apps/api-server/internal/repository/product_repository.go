package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

type ProductRepository struct{}

func NewProductRepository() *ProductRepository {
	return &ProductRepository{}
}

func (r *ProductRepository) List(ctx context.Context, tx pgx.Tx, filter model.ProductListFilter) ([]model.Product, int, error) {
	var conditions []string
	var args []any
	argIdx := 1

	if filter.Name != nil {
		conditions = append(conditions, fmt.Sprintf("name ILIKE '%%' || $%d || '%%'", argIdx))
		args = append(args, *filter.Name)
		argIdx++
	}
	if filter.SKU != nil {
		conditions = append(conditions, fmt.Sprintf("sku ILIKE '%%' || $%d || '%%'", argIdx))
		args = append(args, *filter.SKU)
		argIdx++
	}
	if filter.Tag != nil {
		conditions = append(conditions, fmt.Sprintf("tags @> ARRAY[$%d]::text[]", argIdx))
		args = append(args, *filter.Tag)
		argIdx++
	}
	if filter.Category != nil {
		conditions = append(conditions, fmt.Sprintf("category = $%d", argIdx))
		args = append(args, *filter.Category)
		argIdx++
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM products %s", where)
	var total int
	if err := tx.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count products: %w", err)
	}

	allowedSortColumns := map[string]string{
		"created_at":     "created_at",
		"name":           "name",
		"sku":            "sku",
		"price":          "price",
		"stock_quantity": "stock_quantity",
	}
	orderByClause := model.BuildOrderByClause(filter.SortBy, filter.SortOrder, allowedSortColumns)

	query := fmt.Sprintf(
		`SELECT id, tenant_id, external_id, source, name, sku, ean, price, stock_quantity, metadata, tags, description_short, description_long, weight, width, height, depth, category, image_url, images, has_variants, created_at, updated_at
		 FROM products %s %s LIMIT $%d OFFSET $%d`,
		where, orderByClause, argIdx, argIdx+1,
	)
	args = append(args, filter.Limit, filter.Offset)

	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list products: %w", err)
	}
	defer rows.Close()

	var products []model.Product
	for rows.Next() {
		var p model.Product
		if err := rows.Scan(&p.ID, &p.TenantID, &p.ExternalID, &p.Source, &p.Name,
			&p.SKU, &p.EAN, &p.Price, &p.StockQuantity, &p.Metadata, &p.Tags,
			&p.DescriptionShort, &p.DescriptionLong,
			&p.Weight, &p.Width, &p.Height, &p.Depth, &p.Category,
			&p.ImageURL, &p.Images, &p.HasVariants, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan product: %w", err)
		}
		products = append(products, p)
	}
	return products, total, rows.Err()
}

func (r *ProductRepository) FindByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.Product, error) {
	var p model.Product
	err := tx.QueryRow(ctx,
		`SELECT id, tenant_id, external_id, source, name, sku, ean, price, stock_quantity, metadata, tags, description_short, description_long, weight, width, height, depth, category, image_url, images, has_variants, created_at, updated_at
		 FROM products WHERE id = $1`, id,
	).Scan(&p.ID, &p.TenantID, &p.ExternalID, &p.Source, &p.Name,
		&p.SKU, &p.EAN, &p.Price, &p.StockQuantity, &p.Metadata, &p.Tags,
		&p.DescriptionShort, &p.DescriptionLong,
		&p.Weight, &p.Width, &p.Height, &p.Depth, &p.Category,
		&p.ImageURL, &p.Images, &p.HasVariants, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find product by id: %w", err)
	}
	return &p, nil
}

func (r *ProductRepository) Create(ctx context.Context, tx pgx.Tx, product *model.Product) error {
	tags := product.Tags
	if tags == nil {
		tags = []string{}
	}
	return tx.QueryRow(ctx,
		`INSERT INTO products (id, tenant_id, external_id, source, name, sku, ean, price, stock_quantity, metadata, tags, description_short, description_long, weight, width, height, depth, category, image_url, images)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)
		 RETURNING created_at, updated_at`,
		product.ID, product.TenantID, product.ExternalID, product.Source, product.Name,
		product.SKU, product.EAN, product.Price, product.StockQuantity, product.Metadata, tags,
		product.DescriptionShort, product.DescriptionLong,
		product.Weight, product.Width, product.Height, product.Depth, product.Category,
		product.ImageURL, product.Images,
	).Scan(&product.CreatedAt, &product.UpdatedAt)
}

func (r *ProductRepository) Update(ctx context.Context, tx pgx.Tx, id uuid.UUID, req model.UpdateProductRequest) error {
	var setClauses []string
	var args []any
	argIdx := 1

	if req.ExternalID != nil {
		setClauses = append(setClauses, fmt.Sprintf("external_id = $%d", argIdx))
		args = append(args, *req.ExternalID)
		argIdx++
	}
	if req.Source != nil {
		setClauses = append(setClauses, fmt.Sprintf("source = $%d", argIdx))
		args = append(args, *req.Source)
		argIdx++
	}
	if req.Name != nil {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, *req.Name)
		argIdx++
	}
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
	if req.Price != nil {
		setClauses = append(setClauses, fmt.Sprintf("price = $%d", argIdx))
		args = append(args, *req.Price)
		argIdx++
	}
	if req.StockQuantity != nil {
		setClauses = append(setClauses, fmt.Sprintf("stock_quantity = $%d", argIdx))
		args = append(args, *req.StockQuantity)
		argIdx++
	}
	if req.Metadata != nil {
		setClauses = append(setClauses, fmt.Sprintf("metadata = $%d", argIdx))
		args = append(args, *req.Metadata)
		argIdx++
	}
	if req.Tags != nil {
		setClauses = append(setClauses, fmt.Sprintf("tags = $%d", argIdx))
		args = append(args, *req.Tags)
		argIdx++
	}
	if req.DescriptionShort != nil {
		setClauses = append(setClauses, fmt.Sprintf("description_short = $%d", argIdx))
		args = append(args, *req.DescriptionShort)
		argIdx++
	}
	if req.DescriptionLong != nil {
		setClauses = append(setClauses, fmt.Sprintf("description_long = $%d", argIdx))
		args = append(args, *req.DescriptionLong)
		argIdx++
	}
	if req.Weight != nil {
		setClauses = append(setClauses, fmt.Sprintf("weight = $%d", argIdx))
		args = append(args, *req.Weight)
		argIdx++
	}
	if req.Width != nil {
		setClauses = append(setClauses, fmt.Sprintf("width = $%d", argIdx))
		args = append(args, *req.Width)
		argIdx++
	}
	if req.Height != nil {
		setClauses = append(setClauses, fmt.Sprintf("height = $%d", argIdx))
		args = append(args, *req.Height)
		argIdx++
	}
	if req.Depth != nil {
		setClauses = append(setClauses, fmt.Sprintf("depth = $%d", argIdx))
		args = append(args, *req.Depth)
		argIdx++
	}
	if req.Category != nil {
		setClauses = append(setClauses, fmt.Sprintf("category = $%d", argIdx))
		args = append(args, *req.Category)
		argIdx++
	}
	if req.ImageURL != nil {
		setClauses = append(setClauses, fmt.Sprintf("image_url = $%d", argIdx))
		args = append(args, *req.ImageURL)
		argIdx++
	}
	if req.Images != nil {
		setClauses = append(setClauses, fmt.Sprintf("images = $%d", argIdx))
		args = append(args, *req.Images)
		argIdx++
	}

	if len(setClauses) == 0 {
		return nil
	}

	setClauses = append(setClauses, "updated_at = NOW()")
	query := fmt.Sprintf("UPDATE products SET %s WHERE id = $%d",
		strings.Join(setClauses, ", "), argIdx)
	args = append(args, id)

	ct, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update product: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("product not found")
	}
	return nil
}

func (r *ProductRepository) Delete(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	ct, err := tx.Exec(ctx, "DELETE FROM products WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete product: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("product not found")
	}
	return nil
}
