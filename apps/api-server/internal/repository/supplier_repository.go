package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// SupplierRepository handles persistence for suppliers.
type SupplierRepository struct{}

// NewSupplierRepository creates a new SupplierRepository.
func NewSupplierRepository() *SupplierRepository {
	return &SupplierRepository{}
}

// List returns a paginated list of suppliers matching the filter.
func (r *SupplierRepository) List(ctx context.Context, tx pgx.Tx, filter model.SupplierListFilter) ([]model.Supplier, int, error) {
	qb := NewQueryBuilder()
	if filter.Status != nil {
		qb.Add("status = $%d", *filter.Status)
	}
	where := qb.WhereClause()

	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM suppliers %s", where)
	if err := tx.QueryRow(ctx, countQuery, qb.Args()...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count suppliers: %w", err)
	}

	allowedSortColumns := map[string]string{
		"created_at": "created_at",
		"name":       "name",
		"status":     "status",
	}
	orderByClause := model.BuildOrderByClause(filter.SortBy, filter.SortOrder, allowedSortColumns)

	limitIdx := qb.AddArgs(filter.Limit, filter.Offset)
	query := fmt.Sprintf(
		`SELECT id, tenant_id, name, code, feed_url, feed_format, status, settings,
		        sync_interval_minutes, last_sync_at, error_message, portal_enabled, integration_id, default_category_id, created_at, updated_at
		 FROM suppliers %s %s LIMIT $%d OFFSET $%d`,
		where, orderByClause, limitIdx, limitIdx+1,
	)

	rows, err := tx.Query(ctx, query, qb.Args()...)
	if err != nil {
		return nil, 0, fmt.Errorf("list suppliers: %w", err)
	}
	defer rows.Close()

	var suppliers []model.Supplier
	for rows.Next() {
		var s model.Supplier
		if err := rows.Scan(
			&s.ID, &s.TenantID, &s.Name, &s.Code, &s.FeedURL, &s.FeedFormat,
			&s.Status, &s.Settings, &s.SyncIntervalMinutes, &s.LastSyncAt, &s.ErrorMessage,
			&s.PortalEnabled, &s.IntegrationID, &s.DefaultCategoryID, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan supplier: %w", err)
		}
		suppliers = append(suppliers, s)
	}
	return suppliers, total, rows.Err()
}

// FindByID returns a supplier by its ID.
func (r *SupplierRepository) FindByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.Supplier, error) {
	var s model.Supplier
	err := tx.QueryRow(ctx,
		`SELECT id, tenant_id, name, code, feed_url, feed_format, status, settings,
		        sync_interval_minutes, last_sync_at, error_message, portal_enabled, integration_id, default_category_id, created_at, updated_at
		 FROM suppliers WHERE id = $1`, id,
	).Scan(
		&s.ID, &s.TenantID, &s.Name, &s.Code, &s.FeedURL, &s.FeedFormat,
		&s.Status, &s.Settings, &s.SyncIntervalMinutes, &s.LastSyncAt, &s.ErrorMessage,
		&s.PortalEnabled, &s.IntegrationID, &s.DefaultCategoryID, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find supplier by id: %w", err)
	}
	return &s, nil
}

// Create inserts a new supplier.
func (r *SupplierRepository) Create(ctx context.Context, tx pgx.Tx, supplier *model.Supplier) error {
	return tx.QueryRow(ctx,
		`INSERT INTO suppliers (id, tenant_id, name, code, feed_url, feed_format, status, settings, sync_interval_minutes, integration_id)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		 RETURNING created_at, updated_at`,
		supplier.ID, supplier.TenantID, supplier.Name, supplier.Code,
		supplier.FeedURL, supplier.FeedFormat, supplier.Status, supplier.Settings,
		supplier.SyncIntervalMinutes, supplier.IntegrationID,
	).Scan(&supplier.CreatedAt, &supplier.UpdatedAt)
}

// Update applies partial updates to a supplier.
func (r *SupplierRepository) Update(ctx context.Context, tx pgx.Tx, id uuid.UUID, req model.UpdateSupplierRequest) error {
	ub := NewUpdateBuilder()
	SetPtr(ub, "name", req.Name)
	SetPtr(ub, "code", req.Code)
	SetPtr(ub, "feed_url", req.FeedURL)
	SetPtr(ub, "feed_format", req.FeedFormat)
	SetPtr(ub, "status", req.Status)
	SetPtr(ub, "settings", req.Settings)
	SetPtr(ub, "error_message", req.ErrorMessage)
	SetPtr(ub, "sync_interval_minutes", req.SyncIntervalMinutes)
	SetPtr(ub, "portal_enabled", req.PortalEnabled)
	SetPtr(ub, "integration_id", req.IntegrationID)
	SetPtr(ub, "default_category_id", req.DefaultCategoryID)

	if ub.IsEmpty() {
		return nil
	}

	ub.SetRaw("updated_at = NOW()")
	query := fmt.Sprintf("UPDATE suppliers SET %s WHERE id = $%d",
		ub.SetClause(), ub.NextArgIdx())
	args := append(ub.Args(), id)

	ct, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update supplier: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("supplier not found")
	}
	return nil
}

// Delete removes a supplier by its ID.
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

// UpdateSyncStatus records the last sync time and any error for the supplier.
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

// UpdateLastFullSync records the last full product sync timestamp in the supplier's JSONB settings.
func (r *SupplierRepository) UpdateLastFullSync(ctx context.Context, tx pgx.Tx, id uuid.UUID, t time.Time) error {
	_, err := tx.Exec(ctx,
		`UPDATE suppliers SET settings = COALESCE(settings, '{}'::jsonb) || jsonb_build_object('last_full_sync_at', $1::text), updated_at = NOW() WHERE id = $2`,
		t.Format(time.RFC3339), id,
	)
	if err != nil {
		return fmt.Errorf("update supplier last full sync: %w", err)
	}
	return nil
}

// UpdateSettingsKeys merges specific keys into the supplier's settings JSONB.
func (r *SupplierRepository) UpdateSettingsKeys(ctx context.Context, tx pgx.Tx, id uuid.UUID, keys map[string]any) error {
	keysJSON, err := json.Marshal(keys)
	if err != nil {
		return fmt.Errorf("marshal settings keys: %w", err)
	}
	ct, err := tx.Exec(ctx,
		`UPDATE suppliers SET settings = COALESCE(settings, '{}'::jsonb) || $1::jsonb, updated_at = NOW() WHERE id = $2`,
		string(keysJSON), id,
	)
	if err != nil {
		return fmt.Errorf("update supplier settings keys: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("supplier not found")
	}
	return nil
}

// SupplierProductRepository handles persistence for supplier products.
type SupplierProductRepository struct{}

// NewSupplierProductRepository creates a new SupplierProductRepository.
func NewSupplierProductRepository() *SupplierProductRepository {
	return &SupplierProductRepository{}
}

var supplierProductColumns = `id, tenant_id, supplier_id, product_id, external_id, name, ean, sku,
	price, stock_quantity, source_category, metadata, last_synced_at, created_at, updated_at`

func scanSupplierProduct(row interface{ Scan(dest ...any) error }) (*model.SupplierProduct, error) {
	var sp model.SupplierProduct
	err := row.Scan(
		&sp.ID, &sp.TenantID, &sp.SupplierID, &sp.ProductID, &sp.ExternalID,
		&sp.Name, &sp.EAN, &sp.SKU, &sp.Price, &sp.StockQuantity,
		&sp.SourceCategory, &sp.Metadata, &sp.LastSyncedAt, &sp.CreatedAt, &sp.UpdatedAt,
	)
	return &sp, err
}

// List returns a paginated list of supplier products matching the filter.
func (r *SupplierProductRepository) List(ctx context.Context, tx pgx.Tx, filter model.SupplierProductListFilter) ([]model.SupplierProduct, int, error) {
	qb := NewQueryBuilder()
	if filter.SupplierID != nil {
		qb.Add("supplier_id = $%d", *filter.SupplierID)
	}
	if filter.EAN != nil {
		qb.Add("ean = $%d", *filter.EAN)
	}
	if filter.Linked != nil {
		if *filter.Linked {
			qb.AddRaw("product_id IS NOT NULL")
		} else {
			qb.AddRaw("product_id IS NULL")
		}
	}
	if filter.Search != nil && *filter.Search != "" {
		qb.AddMultiRef("(name ILIKE $%d OR ean ILIKE $%d OR sku ILIKE $%d)", 3, "%"+*filter.Search+"%")
	}
	if filter.SourceCategory != nil && *filter.SourceCategory != "" {
		qb.Add("source_category = $%d", *filter.SourceCategory)
	}
	where := qb.WhereClause()

	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM supplier_products %s", where)
	if err := tx.QueryRow(ctx, countQuery, qb.Args()...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count supplier products: %w", err)
	}

	allowedSortColumns := map[string]string{
		"created_at":     "created_at",
		"name":           "name",
		"price":          "price",
		"stock_quantity": "stock_quantity",
	}
	orderByClause := model.BuildOrderByClause(filter.SortBy, filter.SortOrder, allowedSortColumns)

	limitIdx := qb.AddArgs(filter.Limit, filter.Offset)
	query := fmt.Sprintf(
		`SELECT %s FROM supplier_products %s %s LIMIT $%d OFFSET $%d`,
		supplierProductColumns, where, orderByClause, limitIdx, limitIdx+1,
	)

	rows, err := tx.Query(ctx, query, qb.Args()...)
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

// FindByID returns a supplier product by its ID.
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

// Create inserts a new supplier product.
func (r *SupplierProductRepository) Create(ctx context.Context, tx pgx.Tx, sp *model.SupplierProduct) error {
	return tx.QueryRow(ctx,
		`INSERT INTO supplier_products (id, tenant_id, supplier_id, product_id, external_id, name, ean, sku, price, stock_quantity, source_category, metadata, last_synced_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		 RETURNING created_at, updated_at`,
		sp.ID, sp.TenantID, sp.SupplierID, sp.ProductID, sp.ExternalID,
		sp.Name, sp.EAN, sp.SKU, sp.Price, sp.StockQuantity, sp.SourceCategory, sp.Metadata, sp.LastSyncedAt,
	).Scan(&sp.CreatedAt, &sp.UpdatedAt)
}

// Update overwrites mutable fields on a supplier product.
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

// Delete removes a supplier product by its ID.
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

// FindByEAN returns the first supplier product with the given EAN.
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

// FindBySupplierAndExternalID returns the supplier product matching supplier and external ID.
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

// FindBySupplierAndProductID returns the supplier's catalogue entry linked to the given
// internal product (the supplier_products mapping). When more than one mapping exists the
// most recently updated one wins (deterministic). Returns nil when the product has no
// mapping at this supplier — the caller must then fall back to EAN-only identity, NEVER the
// tenant's internal SKU (OPE-516).
func (r *SupplierProductRepository) FindBySupplierAndProductID(ctx context.Context, tx pgx.Tx, supplierID, productID uuid.UUID) (*model.SupplierProduct, error) {
	sp, err := scanSupplierProduct(tx.QueryRow(ctx,
		fmt.Sprintf("SELECT %s FROM supplier_products WHERE supplier_id = $1 AND product_id = $2 ORDER BY updated_at DESC LIMIT 1", supplierProductColumns),
		supplierID, productID,
	))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find supplier product by supplier and product id: %w", err)
	}
	return sp, nil
}

// UpsertByExternalID inserts or updates a supplier product matched by external ID.
func (r *SupplierProductRepository) UpsertByExternalID(ctx context.Context, tx pgx.Tx, sp *model.SupplierProduct) error {
	return tx.QueryRow(ctx,
		`INSERT INTO supplier_products (id, tenant_id, supplier_id, product_id, external_id, name, ean, sku, price, stock_quantity, source_category, metadata, last_synced_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		 ON CONFLICT (tenant_id, supplier_id, external_id)
		 DO UPDATE SET name = COALESCE(NULLIF(EXCLUDED.name, ''), supplier_products.name),
		              ean = COALESCE(EXCLUDED.ean, supplier_products.ean),
		              sku = COALESCE(EXCLUDED.sku, supplier_products.sku),
		              price = COALESCE(EXCLUDED.price, supplier_products.price),
		              stock_quantity = EXCLUDED.stock_quantity,
		              source_category = COALESCE(EXCLUDED.source_category, supplier_products.source_category),
		              metadata = COALESCE(supplier_products.metadata, '{}'::jsonb) || EXCLUDED.metadata,
		              last_synced_at = EXCLUDED.last_synced_at,
		              updated_at = NOW()
		 RETURNING id, created_at, updated_at`,
		sp.ID, sp.TenantID, sp.SupplierID, sp.ProductID, sp.ExternalID,
		sp.Name, sp.EAN, sp.SKU, sp.Price, sp.StockQuantity, sp.SourceCategory, sp.Metadata, sp.LastSyncedAt,
	).Scan(&sp.ID, &sp.CreatedAt, &sp.UpdatedAt)
}

// supplierProductUpsertChunk bounds rows per multi-row upsert. 13 bind params per row;
// PostgreSQL caps a statement at 65535 params (max ~5041 rows), so 1000 stays well under.
const supplierProductUpsertChunk = 1000

// buildSupplierProductsUpsert builds a multi-row INSERT … ON CONFLICT … DO UPDATE for one
// chunk, returning the query and positional args. The DO UPDATE column list is identical to
// the single-row UpsertByExternalID (merge semantics preserved); product_id is intentionally
// not in DO UPDATE so an existing link survives. RETURNING id, external_id lets the caller map
// the DB row id (the existing id on conflict) back onto each struct. Pure (no I/O).
func buildSupplierProductsUpsert(chunk []*model.SupplierProduct) (string, []any) {
	var b strings.Builder
	b.WriteString(`INSERT INTO supplier_products (id, tenant_id, supplier_id, product_id, external_id, name, ean, sku, price, stock_quantity, source_category, metadata, last_synced_at) VALUES `)
	args := make([]any, 0, len(chunk)*13)
	for i, sp := range chunk {
		if i > 0 {
			b.WriteString(", ")
		}
		n := i * 13
		fmt.Fprintf(&b, "($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)",
			n+1, n+2, n+3, n+4, n+5, n+6, n+7, n+8, n+9, n+10, n+11, n+12, n+13)
		args = append(args, sp.ID, sp.TenantID, sp.SupplierID, sp.ProductID, sp.ExternalID,
			sp.Name, sp.EAN, sp.SKU, sp.Price, sp.StockQuantity, sp.SourceCategory, sp.Metadata, sp.LastSyncedAt)
	}
	b.WriteString(`
		 ON CONFLICT (tenant_id, supplier_id, external_id)
		 DO UPDATE SET name = COALESCE(NULLIF(EXCLUDED.name, ''), supplier_products.name),
		              ean = COALESCE(EXCLUDED.ean, supplier_products.ean),
		              sku = COALESCE(EXCLUDED.sku, supplier_products.sku),
		              price = COALESCE(EXCLUDED.price, supplier_products.price),
		              stock_quantity = EXCLUDED.stock_quantity,
		              source_category = COALESCE(EXCLUDED.source_category, supplier_products.source_category),
		              metadata = COALESCE(supplier_products.metadata, '{}'::jsonb) || EXCLUDED.metadata,
		              last_synced_at = EXCLUDED.last_synced_at,
		              updated_at = NOW()
		 RETURNING id, external_id`)
	return b.String(), args
}

// UpsertBatchByExternalID upserts many supplier products in chunked multi-row statements
// (one round-trip per chunk) instead of one per row. Each input must have a unique external_id
// within the slice (caller dedupes) — a duplicate would trip "ON CONFLICT cannot affect row a
// second time". After upsert each sp.ID is set to the DB row id (the existing id on conflict),
// matching the single-row UpsertByExternalID. Runs inside the caller's tenant-scoped tx.
func (r *SupplierProductRepository) UpsertBatchByExternalID(ctx context.Context, tx pgx.Tx, sps []*model.SupplierProduct) error {
	byExternalID := make(map[string]*model.SupplierProduct, len(sps))
	for _, sp := range sps {
		byExternalID[sp.ExternalID] = sp
	}
	type returnedRow struct {
		id  uuid.UUID
		ext string
	}
	for start := 0; start < len(sps); start += supplierProductUpsertChunk {
		end := min(start+supplierProductUpsertChunk, len(sps))
		query, args := buildSupplierProductsUpsert(sps[start:end])
		rows, err := tx.Query(ctx, query, args...)
		if err != nil {
			return fmt.Errorf("batch upsert supplier products: %w", err)
		}
		// Drain fully before the next tx.Query (pgx single-connection rule).
		var returned []returnedRow
		for rows.Next() {
			var rr returnedRow
			if err := rows.Scan(&rr.id, &rr.ext); err != nil {
				rows.Close()
				return fmt.Errorf("scan batch upsert result: %w", err)
			}
			returned = append(returned, rr)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return fmt.Errorf("batch upsert supplier products rows: %w", err)
		}
		rows.Close()
		for _, rr := range returned {
			if sp, ok := byExternalID[rr.ext]; ok {
				sp.ID = rr.id
			}
		}
	}
	return nil
}

// FindByIDs returns supplier products matching the given IDs.
func (r *SupplierProductRepository) FindByIDs(ctx context.Context, tx pgx.Tx, ids []uuid.UUID) ([]model.SupplierProduct, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}
	query := fmt.Sprintf("SELECT %s FROM supplier_products WHERE id IN (%s)",
		supplierProductColumns, strings.Join(placeholders, ", "))
	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("find supplier products by ids: %w", err)
	}
	defer rows.Close()
	var products []model.SupplierProduct
	for rows.Next() {
		sp, err := scanSupplierProduct(rows)
		if err != nil {
			return nil, fmt.Errorf("scan supplier product: %w", err)
		}
		products = append(products, *sp)
	}
	return products, rows.Err()
}

// LinkToProduct associates a supplier product with an internal product.
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

// UnlinkProduct removes the link between a supplier product and an OMS product.
func (r *SupplierProductRepository) UnlinkProduct(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	ct, err := tx.Exec(ctx,
		`UPDATE supplier_products SET product_id = NULL, updated_at = NOW() WHERE id = $1`,
		id,
	)
	if err != nil {
		return fmt.Errorf("unlink supplier product: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("supplier product not found")
	}
	return nil
}

// BulkDelete deletes multiple supplier products by their IDs and returns the count deleted.
func (r *SupplierProductRepository) BulkDelete(ctx context.Context, tx pgx.Tx, ids []uuid.UUID) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}
	ct, err := tx.Exec(ctx,
		fmt.Sprintf("DELETE FROM supplier_products WHERE id IN (%s)", strings.Join(placeholders, ", ")),
		args...,
	)
	if err != nil {
		return 0, fmt.Errorf("bulk delete supplier products: %w", err)
	}
	return int(ct.RowsAffected()), nil
}

// ListSourceCategories returns distinct source_category values for a supplier.
func (r *SupplierProductRepository) ListSourceCategories(ctx context.Context, tx pgx.Tx, supplierID uuid.UUID) ([]string, error) {
	rows, err := tx.Query(ctx,
		`SELECT DISTINCT source_category FROM supplier_products
		 WHERE supplier_id = $1 AND source_category IS NOT NULL AND source_category != ''
		 ORDER BY source_category`,
		supplierID,
	)
	if err != nil {
		return nil, fmt.Errorf("list source categories: %w", err)
	}
	defer rows.Close()
	var categories []string
	for rows.Next() {
		var cat string
		if err := rows.Scan(&cat); err != nil {
			return nil, fmt.Errorf("scan source category: %w", err)
		}
		categories = append(categories, cat)
	}
	return categories, rows.Err()
}

// ListAttributes returns distinct attribute names from supplier_products.metadata.attributes for a supplier.
func (r *SupplierProductRepository) ListAttributes(ctx context.Context, tx pgx.Tx, supplierID uuid.UUID) ([]string, error) {
	rows, err := tx.Query(ctx,
		`SELECT DISTINCT key
		 FROM supplier_products, jsonb_object_keys(metadata->'attributes') AS key
		 WHERE supplier_id = $1
		 ORDER BY key`,
		supplierID,
	)
	if err != nil {
		return nil, fmt.Errorf("list supplier attributes: %w", err)
	}
	defer rows.Close()
	var attrs []string
	for rows.Next() {
		var attr string
		if err := rows.Scan(&attr); err != nil {
			return nil, fmt.Errorf("scan supplier attribute: %w", err)
		}
		attrs = append(attrs, attr)
	}
	return attrs, rows.Err()
}

// ListExternalIDsBySupplier returns all external_ids for a supplier's products.
func (r *SupplierProductRepository) ListExternalIDsBySupplier(ctx context.Context, tx pgx.Tx, supplierID uuid.UUID) ([]string, error) {
	rows, err := tx.Query(ctx,
		`SELECT external_id FROM supplier_products WHERE supplier_id = $1`,
		supplierID,
	)
	if err != nil {
		return nil, fmt.Errorf("list external ids: %w", err)
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan external id: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// DeleteStaleByExternalIDs deletes supplier products not present in the given external IDs
// and returns the linked product_ids (OMS products) that were affected.
func (r *SupplierProductRepository) DeleteStaleByExternalIDs(ctx context.Context, tx pgx.Tx, supplierID uuid.UUID, keepExternalIDs []string) ([]uuid.UUID, error) {
	if len(keepExternalIDs) == 0 {
		return nil, nil
	}
	// First get linked product IDs that will be affected
	placeholders := make([]string, len(keepExternalIDs))
	args := make([]any, len(keepExternalIDs)+1)
	args[0] = supplierID
	for i, eid := range keepExternalIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+2)
		args[i+1] = eid
	}
	notInClause := strings.Join(placeholders, ", ")

	rows, err := tx.Query(ctx,
		fmt.Sprintf(`SELECT product_id FROM supplier_products
			WHERE supplier_id = $1 AND external_id NOT IN (%s) AND product_id IS NOT NULL`, notInClause),
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("list stale linked products: %w", err)
	}
	defer rows.Close()
	var linkedProductIDs []uuid.UUID
	for rows.Next() {
		var pid uuid.UUID
		if err := rows.Scan(&pid); err != nil {
			return nil, fmt.Errorf("scan linked product id: %w", err)
		}
		linkedProductIDs = append(linkedProductIDs, pid)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Delete stale supplier products
	_, err = tx.Exec(ctx,
		fmt.Sprintf(`DELETE FROM supplier_products WHERE supplier_id = $1 AND external_id NOT IN (%s)`, notInClause),
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("delete stale supplier products: %w", err)
	}
	return linkedProductIDs, nil
}

// FindSupplierIDByProductID returns the supplier_id for a given linked product_id.
func (r *SupplierProductRepository) FindSupplierIDByProductID(ctx context.Context, tx pgx.Tx, productID uuid.UUID) (*uuid.UUID, error) {
	var supplierID uuid.UUID
	err := tx.QueryRow(ctx,
		`SELECT supplier_id FROM supplier_products WHERE product_id = $1 LIMIT 1`,
		productID,
	).Scan(&supplierID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find supplier by product: %w", err)
	}
	return &supplierID, nil
}

// ListAll returns supplier products across all suppliers with the supplier name included.
func (r *SupplierProductRepository) ListAll(ctx context.Context, tx pgx.Tx, params model.SupplierProductListAllParams) ([]model.SupplierProductWithSupplier, int, error) {
	qb := NewQueryBuilder()
	if params.SupplierID != nil {
		qb.Add("sp.supplier_id = $%d", *params.SupplierID)
	}
	if params.Search != "" {
		qb.AddMultiRef("(sp.name ILIKE $%d OR sp.ean ILIKE $%d OR sp.sku ILIKE $%d)", 3, "%"+params.Search+"%")
	}
	if params.Category != "" {
		qb.Add("sp.source_category = $%d", params.Category)
	}
	switch params.Linked {
	case "linked":
		qb.AddRaw("sp.product_id IS NOT NULL")
	case "unlinked":
		qb.AddRaw("sp.product_id IS NULL")
	}
	where := qb.WhereClause()

	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM supplier_products sp %s", where)
	if err := tx.QueryRow(ctx, countQuery, qb.Args()...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count all supplier products: %w", err)
	}

	allowedSortColumns := map[string]string{
		"created_at":     "sp.created_at",
		"name":           "sp.name",
		"price":          "sp.price",
		"stock_quantity": "sp.stock_quantity",
	}
	orderCol, ok := allowedSortColumns[params.SortBy]
	if !ok {
		orderCol = "sp.created_at"
	}
	direction := "DESC"
	if params.SortOrder == "asc" {
		direction = "ASC"
	}
	orderByClause := fmt.Sprintf("ORDER BY %s %s", orderCol, direction)

	limitIdx := qb.AddArgs(params.Limit, params.Offset)
	query := fmt.Sprintf(
		`SELECT sp.id, sp.tenant_id, sp.supplier_id, sp.product_id, sp.external_id, sp.name, sp.ean, sp.sku,
		        sp.price, sp.stock_quantity, sp.source_category, sp.metadata, sp.last_synced_at, sp.created_at, sp.updated_at,
		        s.name as supplier_name
		 FROM supplier_products sp
		 JOIN suppliers s ON s.id = sp.supplier_id
		 %s %s LIMIT $%d OFFSET $%d`,
		where, orderByClause, limitIdx, limitIdx+1,
	)

	rows, err := tx.Query(ctx, query, qb.Args()...)
	if err != nil {
		return nil, 0, fmt.Errorf("list all supplier products: %w", err)
	}
	defer rows.Close()

	var products []model.SupplierProductWithSupplier
	for rows.Next() {
		var sp model.SupplierProductWithSupplier
		if err := rows.Scan(
			&sp.ID, &sp.TenantID, &sp.SupplierID, &sp.ProductID, &sp.ExternalID,
			&sp.Name, &sp.EAN, &sp.SKU, &sp.Price, &sp.StockQuantity,
			&sp.SourceCategory, &sp.Metadata, &sp.LastSyncedAt, &sp.CreatedAt, &sp.UpdatedAt,
			&sp.SupplierName,
		); err != nil {
			return nil, 0, fmt.Errorf("scan supplier product with supplier: %w", err)
		}
		products = append(products, sp)
	}
	return products, total, rows.Err()
}
