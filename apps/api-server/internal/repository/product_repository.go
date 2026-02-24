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

// productSelectColumns is the canonical list of columns selected from products.
const productSelectColumns = `id, tenant_id, external_id, source, name, sku, ean, price, stock_quantity,
		        metadata, tags, description_short, description_long, weight, width, height, depth,
		        category, image_url, images, has_variants, is_bundle, is_dropship, dropship_supplier_id,
		        created_at, updated_at`

// scanProduct scans a row into a model.Product using the productSelectColumns column order.
func scanProduct(row pgx.Row) (model.Product, error) {
	var p model.Product
	err := row.Scan(
		&p.ID, &p.TenantID, &p.ExternalID, &p.Source, &p.Name,
		&p.SKU, &p.EAN, &p.Price, &p.StockQuantity, &p.Metadata, &p.Tags,
		&p.DescriptionShort, &p.DescriptionLong,
		&p.Weight, &p.Width, &p.Height, &p.Depth, &p.Category,
		&p.ImageURL, &p.Images, &p.HasVariants, &p.IsBundle, &p.IsDropship, &p.DropshipSupplierID,
		&p.CreatedAt, &p.UpdatedAt,
	)
	return p, err
}

// scanProducts scans multiple rows into a slice of model.Product using the productSelectColumns column order.
func scanProducts(rows pgx.Rows) ([]model.Product, error) {
	var products []model.Product
	for rows.Next() {
		p, err := scanProduct(rows)
		if err != nil {
			return nil, err
		}
		products = append(products, p)
	}
	return products, rows.Err()
}

func (r *ProductRepository) List(ctx context.Context, tx pgx.Tx, filter model.ProductListFilter) ([]model.Product, int, error) {
	qb := NewQueryBuilder()

	if filter.Name != nil {
		qb.Add("p.name ILIKE '%%' || $%d || '%%'", *filter.Name)
	}
	if filter.SKU != nil {
		qb.Add("p.sku ILIKE '%%' || $%d || '%%'", *filter.SKU)
	}
	if filter.Tag != nil {
		qb.Add("p.tags @> ARRAY[$%d]::text[]", *filter.Tag)
	}
	if filter.Category != nil {
		qb.Add("p.category = $%d", *filter.Category)
	}
	if len(filter.CategoryIDs) > 0 {
		qb.Add("p.category_id = ANY($%d)", filter.CategoryIDs)
	}
	if filter.SupplierID != nil {
		qb.Add("p.dropship_supplier_id = $%d", *filter.SupplierID)
	}
	if filter.Source != nil {
		qb.Add("p.source = $%d", *filter.Source)
	}
	if filter.Search != nil {
		qb.AddMultiRef("(p.name ILIKE '%%' || $%d || '%%' OR p.sku ILIKE '%%' || $%d || '%%' OR p.ean ILIKE '%%' || $%d || '%%')", 3, *filter.Search)
	}
	if filter.Marketplace != nil {
		if *filter.Marketplace == "none" {
			qb.AddRaw("NOT EXISTS (SELECT 1 FROM product_listings pl WHERE pl.product_id = p.id)")
		} else {
			qb.Add(
				"EXISTS (SELECT 1 FROM product_listings pl JOIN integrations i ON pl.integration_id = i.id WHERE pl.product_id = p.id AND i.provider = $%d)", *filter.Marketplace)
		}
	}

	where := qb.WhereClause()
	fromClause := "FROM products p"

	countQuery := fmt.Sprintf("SELECT COUNT(*) %s %s", fromClause, where)
	var total int
	if err := tx.QueryRow(ctx, countQuery, qb.Args()...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count products: %w", err)
	}

	allowedSortColumns := map[string]string{
		"created_at":     "p.created_at",
		"name":           "p.name",
		"sku":            "p.sku",
		"price":          "p.price",
		"stock_quantity": "p.stock_quantity",
	}
	orderByClause := model.BuildOrderByClause(filter.SortBy, filter.SortOrder, allowedSortColumns)

	argIdx := qb.AddArgs(filter.Limit, filter.Offset)
	query := fmt.Sprintf(
		`SELECT p.id, p.tenant_id, p.external_id, p.source, p.name, p.sku, p.ean, p.price, p.stock_quantity,
		        p.metadata, p.tags, p.description_short, p.description_long, p.weight, p.width, p.height, p.depth,
		        p.category, p.image_url, p.images, p.has_variants, p.is_bundle, p.is_dropship, p.dropship_supplier_id,
		        (SELECT s.name FROM supplier_products sp JOIN suppliers s ON sp.supplier_id = s.id
		                  WHERE sp.product_id = p.id LIMIT 1) AS supplier_name,
		        COALESCE((SELECT array_agg(DISTINCT i.provider ORDER BY i.provider)
		                  FROM product_listings pl
		                  JOIN integrations i ON pl.integration_id = i.id
		                  WHERE pl.product_id = p.id), ARRAY[]::text[]) AS marketplace_providers,
		        p.created_at, p.updated_at
		 %s %s %s LIMIT $%d OFFSET $%d`,
		fromClause, where, orderByClause, argIdx, argIdx+1,
	)

	rows, err := tx.Query(ctx, query, qb.Args()...)
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
			&p.ImageURL, &p.Images, &p.HasVariants, &p.IsBundle, &p.IsDropship, &p.DropshipSupplierID,
			&p.SupplierName, &p.MarketplaceProviders, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan product: %w", err)
		}
		if p.MarketplaceProviders == nil {
			p.MarketplaceProviders = []string{}
		}
		products = append(products, p)
	}
	return products, total, rows.Err()
}

func (r *ProductRepository) FindByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.Product, error) {
	p, err := scanProduct(tx.QueryRow(ctx,
		fmt.Sprintf(`SELECT %s FROM products WHERE id = $1`, productSelectColumns), id,
	))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find product by id: %w", err)
	}
	return &p, nil
}

// FindByIDs returns products matching the given IDs in a single query.
func (r *ProductRepository) FindByIDs(ctx context.Context, tx pgx.Tx, ids []uuid.UUID) ([]model.Product, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := tx.Query(ctx,
		fmt.Sprintf(`SELECT %s FROM products WHERE id = ANY($1)`, productSelectColumns), ids,
	)
	if err != nil {
		return nil, fmt.Errorf("find products by ids: %w", err)
	}
	defer rows.Close()

	products, err := scanProducts(rows)
	if err != nil {
		return nil, fmt.Errorf("scan product: %w", err)
	}
	return products, nil
}

func (r *ProductRepository) Create(ctx context.Context, tx pgx.Tx, product *model.Product) error {
	tags := product.Tags
	if tags == nil {
		tags = []string{}
	}
	return tx.QueryRow(ctx,
		`INSERT INTO products (id, tenant_id, external_id, source, name, sku, ean, price, stock_quantity, metadata, tags, description_short, description_long, weight, width, height, depth, category, image_url, images, is_bundle, is_dropship, dropship_supplier_id)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23)
		 RETURNING created_at, updated_at`,
		product.ID, product.TenantID, product.ExternalID, product.Source, product.Name,
		product.SKU, product.EAN, product.Price, product.StockQuantity, product.Metadata, tags,
		product.DescriptionShort, product.DescriptionLong,
		product.Weight, product.Width, product.Height, product.Depth, product.Category,
		product.ImageURL, product.Images, product.IsBundle, product.IsDropship, product.DropshipSupplierID,
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
	if req.IsBundle != nil {
		setClauses = append(setClauses, fmt.Sprintf("is_bundle = $%d", argIdx))
		args = append(args, *req.IsBundle)
		argIdx++
	}
	if req.IsDropship != nil {
		setClauses = append(setClauses, fmt.Sprintf("is_dropship = $%d", argIdx))
		args = append(args, *req.IsDropship)
		argIdx++
	}
	if req.DropshipSupplierID != nil {
		setClauses = append(setClauses, fmt.Sprintf("dropship_supplier_id = $%d", argIdx))
		args = append(args, *req.DropshipSupplierID)
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

func (r *ProductRepository) FindBySKU(ctx context.Context, tx pgx.Tx, sku string) (*model.Product, error) {
	p, err := scanProduct(tx.QueryRow(ctx,
		fmt.Sprintf(`SELECT %s FROM products WHERE sku = $1 LIMIT 1`, productSelectColumns), sku,
	))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find product by sku: %w", err)
	}
	return &p, nil
}

func (r *ProductRepository) FindByEAN(ctx context.Context, tx pgx.Tx, ean string) (*model.Product, error) {
	p, err := scanProduct(tx.QueryRow(ctx,
		fmt.Sprintf(`SELECT %s FROM products WHERE ean = $1 LIMIT 1`, productSelectColumns), ean,
	))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find product by ean: %w", err)
	}
	return &p, nil
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
