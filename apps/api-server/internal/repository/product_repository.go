package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// ProductRepository handles persistence for products.
type ProductRepository struct{}

// NewProductRepository creates a new ProductRepository.
func NewProductRepository() *ProductRepository {
	return &ProductRepository{}
}

// productSelectColumns is the canonical list of columns selected from products.
const productSelectColumns = `id, tenant_id, external_id, source, name, sku, ean, price, stock_quantity,
		        metadata, tags, description_short, description_long, weight, width, height, depth,
		        category, category_id, image_url, images, has_variants, is_bundle, is_dropship, dropship_supplier_id,
		        created_at, updated_at`

// scanProduct scans a row into a model.Product using the productSelectColumns column order.
func scanProduct(row pgx.Row) (model.Product, error) {
	var p model.Product
	err := row.Scan(
		&p.ID, &p.TenantID, &p.ExternalID, &p.Source, &p.Name,
		&p.SKU, &p.EAN, &p.Price, &p.StockQuantity, &p.Metadata, &p.Tags,
		&p.DescriptionShort, &p.DescriptionLong,
		&p.Weight, &p.Width, &p.Height, &p.Depth, &p.Category, &p.CategoryID,
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

// List returns a paginated list of products matching the filter.
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
		        p.category, p.category_id, p.image_url, p.images, p.has_variants, p.is_bundle, p.is_dropship, p.dropship_supplier_id,
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
			&p.Weight, &p.Width, &p.Height, &p.Depth, &p.Category, &p.CategoryID,
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

// FindByID returns a product by its ID.
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

// FindByIDs returns products matching the given IDs in a single query. It uses
// scalar IN ($1,$2,...) placeholders rather than = ANY($1) with a []uuid.UUID
// argument: under the simple query protocol production runs behind the Supabase
// transaction pooler, pgx cannot encode a []uuid.UUID parameter for an unknown
// OID, so = ANY($1) fails at runtime. Scalar placeholders encode fine under both
// protocols. Mirrors SupplierProductRepository.FindByIDs.
func (r *ProductRepository) FindByIDs(ctx context.Context, tx pgx.Tx, ids []uuid.UUID) ([]model.Product, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}
	rows, err := tx.Query(ctx,
		fmt.Sprintf(`SELECT %s FROM products WHERE id IN (%s)`, productSelectColumns, strings.Join(placeholders, ", ")), args...,
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

// Create inserts a new product.
func (r *ProductRepository) Create(ctx context.Context, tx pgx.Tx, product *model.Product) error {
	tags := product.Tags
	if tags == nil {
		tags = []string{}
	}
	return tx.QueryRow(ctx,
		`INSERT INTO products (id, tenant_id, external_id, source, name, sku, ean, price, stock_quantity, metadata, tags, description_short, description_long, weight, width, height, depth, category, category_id, image_url, images, is_bundle, is_dropship, dropship_supplier_id)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24)
		 RETURNING created_at, updated_at`,
		product.ID, product.TenantID, product.ExternalID, product.Source, product.Name,
		product.SKU, product.EAN, product.Price, product.StockQuantity, product.Metadata, tags,
		product.DescriptionShort, product.DescriptionLong,
		product.Weight, product.Width, product.Height, product.Depth, product.Category, product.CategoryID,
		product.ImageURL, product.Images, product.IsBundle, product.IsDropship, product.DropshipSupplierID,
	).Scan(&product.CreatedAt, &product.UpdatedAt)
}

// Update applies partial updates to a product.
func (r *ProductRepository) Update(ctx context.Context, tx pgx.Tx, id uuid.UUID, req model.UpdateProductRequest) error {
	ub := NewUpdateBuilder()
	SetPtr(ub, "external_id", req.ExternalID)
	SetPtr(ub, "source", req.Source)
	SetPtr(ub, "name", req.Name)
	SetPtr(ub, "sku", req.SKU)
	SetPtr(ub, "ean", req.EAN)
	SetPtr(ub, "price", req.Price)
	SetPtr(ub, "stock_quantity", req.StockQuantity)
	SetPtr(ub, "metadata", req.Metadata)
	SetPtr(ub, "tags", req.Tags)
	SetPtr(ub, "description_short", req.DescriptionShort)
	SetPtr(ub, "description_long", req.DescriptionLong)
	SetPtr(ub, "weight", req.Weight)
	SetPtr(ub, "width", req.Width)
	SetPtr(ub, "height", req.Height)
	SetPtr(ub, "depth", req.Depth)
	SetPtr(ub, "category", req.Category)
	SetPtr(ub, "category_id", req.CategoryID)
	SetPtr(ub, "image_url", req.ImageURL)
	SetPtr(ub, "images", req.Images)
	SetPtr(ub, "is_bundle", req.IsBundle)
	SetPtr(ub, "is_dropship", req.IsDropship)
	SetPtr(ub, "dropship_supplier_id", req.DropshipSupplierID)

	if ub.IsEmpty() {
		return nil
	}

	ub.SetRaw("updated_at = NOW()")
	query := fmt.Sprintf("UPDATE products SET %s WHERE id = $%d",
		ub.SetClause(), ub.NextArgIdx())
	args := append(ub.Args(), id)

	ct, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update product: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("product not found")
	}
	return nil
}

// FindBySKU returns the first product with the given SKU.
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

// FindByEAN returns the first product with the given EAN barcode.
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

// FindIDsByEANs resolves a set of EANs to product ids in one query, returning ean -> product id.
// DISTINCT ON picks one product per EAN deterministically, matching the single-row
// "WHERE ean = $1 LIMIT 1" lookup (any product with that EAN is an acceptable auto-link target).
func (r *ProductRepository) FindIDsByEANs(ctx context.Context, tx pgx.Tx, eans []string) (map[string]uuid.UUID, error) {
	result := make(map[string]uuid.UUID)
	if len(eans) == 0 {
		return result, nil
	}
	rows, err := tx.Query(ctx,
		`SELECT DISTINCT ON (ean) ean, id FROM products WHERE ean = ANY($1) ORDER BY ean, id`, eans)
	if err != nil {
		return nil, fmt.Errorf("find product ids by eans: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var ean string
		var id uuid.UUID
		if err := rows.Scan(&ean, &id); err != nil {
			return nil, fmt.Errorf("scan product id by ean: %w", err)
		}
		result[ean] = id
	}
	return result, rows.Err()
}

// Delete removes a product by its ID.
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

// AvailableStockBatch returns the canonical available stock per product id: the sum of
// warehouse_stock (quantity - reserved) across warehouses, falling back to the legacy
// products.stock_quantity only for products that have no warehouse rows (e.g. supplier/import-
// managed products that are not warehouse-tracked). This is the stock that order fulfillment
// actually draws from and that marketplace sync reports, unlike the raw products.stock_quantity
// column which is never decremented on shipment. Relies on RLS for tenant scoping.
func (r *ProductRepository) AvailableStockBatch(ctx context.Context, tx pgx.Tx, productIDs []uuid.UUID) (map[uuid.UUID]int, error) {
	result := make(map[uuid.UUID]int, len(productIDs))
	if len(productIDs) == 0 {
		return result, nil
	}
	rows, err := tx.Query(ctx,
		`SELECT p.id,
		        GREATEST(COALESCE(ws.available, p.stock_quantity), 0)
		 FROM products p
		 LEFT JOIN (
		     SELECT product_id, SUM(quantity) - SUM(reserved) AS available
		     FROM warehouse_stock
		     GROUP BY product_id
		 ) ws ON ws.product_id = p.id
		 WHERE p.id = ANY($1)`, productIDs)
	if err != nil {
		return nil, fmt.Errorf("available stock batch: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id uuid.UUID
		var available int
		if err := rows.Scan(&id, &available); err != nil {
			return nil, fmt.Errorf("scan available stock: %w", err)
		}
		result[id] = available
	}
	return result, rows.Err()
}
