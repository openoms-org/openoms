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
		        sync_interval_minutes, last_sync_at, error_message, portal_enabled, integration_id, default_category_id, created_at, updated_at
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
			&s.Status, &s.Settings, &s.SyncIntervalMinutes, &s.LastSyncAt, &s.ErrorMessage,
			&s.PortalEnabled, &s.IntegrationID, &s.DefaultCategoryID, &s.CreatedAt, &s.UpdatedAt,
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
	if req.SyncIntervalMinutes != nil {
		setClauses = append(setClauses, fmt.Sprintf("sync_interval_minutes = $%d", argIdx))
		args = append(args, *req.SyncIntervalMinutes)
		argIdx++
	}
	if req.PortalEnabled != nil {
		setClauses = append(setClauses, fmt.Sprintf("portal_enabled = $%d", argIdx))
		args = append(args, *req.PortalEnabled)
		argIdx++
	}
	if req.IntegrationID != nil {
		setClauses = append(setClauses, fmt.Sprintf("integration_id = $%d", argIdx))
		args = append(args, *req.IntegrationID)
		argIdx++
	}
	if req.DefaultCategoryID != nil {
		setClauses = append(setClauses, fmt.Sprintf("default_category_id = $%d", argIdx))
		args = append(args, *req.DefaultCategoryID)
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
		keysJSON, id,
	)
	if err != nil {
		return fmt.Errorf("update supplier settings keys: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("supplier not found")
	}
	return nil
}

// SupplierProductRepository

type SupplierProductRepository struct{}

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
	if filter.Search != nil && *filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(name ILIKE $%d OR ean ILIKE $%d OR sku ILIKE $%d)", argIdx, argIdx, argIdx))
		args = append(args, "%"+*filter.Search+"%")
		argIdx++
	}
	if filter.SourceCategory != nil && *filter.SourceCategory != "" {
		conditions = append(conditions, fmt.Sprintf("source_category = $%d", argIdx))
		args = append(args, *filter.SourceCategory)
		argIdx++
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
		`INSERT INTO supplier_products (id, tenant_id, supplier_id, product_id, external_id, name, ean, sku, price, stock_quantity, source_category, metadata, last_synced_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		 RETURNING created_at, updated_at`,
		sp.ID, sp.TenantID, sp.SupplierID, sp.ProductID, sp.ExternalID,
		sp.Name, sp.EAN, sp.SKU, sp.Price, sp.StockQuantity, sp.SourceCategory, sp.Metadata, sp.LastSyncedAt,
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
	var conditions []string
	var args []any
	argIdx := 1

	if params.SupplierID != nil {
		conditions = append(conditions, fmt.Sprintf("sp.supplier_id = $%d", argIdx))
		args = append(args, *params.SupplierID)
		argIdx++
	}
	if params.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(sp.name ILIKE $%d OR sp.ean ILIKE $%d OR sp.sku ILIKE $%d)", argIdx, argIdx, argIdx))
		args = append(args, "%"+params.Search+"%")
		argIdx++
	}
	if params.Category != "" {
		conditions = append(conditions, fmt.Sprintf("sp.source_category = $%d", argIdx))
		args = append(args, params.Category)
		argIdx++
	}
	switch params.Linked {
	case "linked":
		conditions = append(conditions, "sp.product_id IS NOT NULL")
	case "unlinked":
		conditions = append(conditions, "sp.product_id IS NULL")
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM supplier_products sp %s", where)
	if err := tx.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
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

	query := fmt.Sprintf(
		`SELECT sp.id, sp.tenant_id, sp.supplier_id, sp.product_id, sp.external_id, sp.name, sp.ean, sp.sku,
		        sp.price, sp.stock_quantity, sp.source_category, sp.metadata, sp.last_synced_at, sp.created_at, sp.updated_at,
		        s.name as supplier_name
		 FROM supplier_products sp
		 JOIN suppliers s ON s.id = sp.supplier_id
		 %s %s LIMIT $%d OFFSET $%d`,
		where, orderByClause, argIdx, argIdx+1,
	)
	args = append(args, params.Limit, params.Offset)

	rows, err := tx.Query(ctx, query, args...)
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
