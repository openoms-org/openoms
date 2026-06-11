package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// ProductListingRepository handles persistence for product marketplace listings.
type ProductListingRepository struct{}

// NewProductListingRepository creates a new ProductListingRepository.
func NewProductListingRepository() *ProductListingRepository {
	return &ProductListingRepository{}
}

// Create inserts a new product listing.
func (r *ProductListingRepository) Create(ctx context.Context, tx pgx.Tx, listing *model.ProductListing) error {
	return tx.QueryRow(ctx,
		`INSERT INTO product_listings (
			id, tenant_id, product_id, integration_id, external_id,
			status, url, price_override, stock_override,
			sync_status, last_synced_at, error_message, stock_sync_mode, description_html, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING created_at, updated_at`,
		listing.ID, listing.TenantID, listing.ProductID, listing.IntegrationID, listing.ExternalID,
		listing.Status, listing.URL, listing.PriceOverride, listing.StockOverride,
		listing.SyncStatus, listing.LastSyncedAt, listing.ErrorMessage, listing.StockSyncMode, listing.DescriptionHTML, listing.Metadata,
	).Scan(&listing.CreatedAt, &listing.UpdatedAt)
}

// Update applies partial updates to a product listing.
func (r *ProductListingRepository) Update(ctx context.Context, tx pgx.Tx, id uuid.UUID, req *model.UpdateProductListingRequest) error {
	ub := NewUpdateBuilder()
	SetPtr(ub, "external_id", req.ExternalID)
	SetPtr(ub, "status", req.Status)
	SetPtr(ub, "url", req.URL)
	SetPtr(ub, "price_override", req.PriceOverride)
	SetPtr(ub, "stock_override", req.StockOverride)
	SetPtr(ub, "sync_status", req.SyncStatus)
	SetPtr(ub, "error_message", req.ErrorMessage)
	SetPtr(ub, "stock_sync_mode", req.StockSyncMode)
	SetPtr(ub, "description_html", req.DescriptionHTML)
	SetPtr(ub, "metadata", req.Metadata)

	if ub.IsEmpty() {
		return nil
	}

	ub.SetRaw("updated_at = NOW()")
	query := fmt.Sprintf("UPDATE product_listings SET %s WHERE id = $%d",
		ub.SetClause(), ub.NextArgIdx())
	args := append(ub.Args(), id)

	ct, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update product listing: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("product listing not found")
	}
	return nil
}

// GetByID returns a product listing by its ID.
func (r *ProductListingRepository) GetByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.ProductListing, error) {
	var l model.ProductListing
	err := tx.QueryRow(ctx,
		`SELECT id, tenant_id, product_id, integration_id, external_id,
		        status, url, price_override, stock_override,
		        sync_status, last_synced_at, error_message, stock_sync_mode, description_html, metadata,
		        created_at, updated_at
		 FROM product_listings WHERE id = $1`, id,
	).Scan(
		&l.ID, &l.TenantID, &l.ProductID, &l.IntegrationID, &l.ExternalID,
		&l.Status, &l.URL, &l.PriceOverride, &l.StockOverride,
		&l.SyncStatus, &l.LastSyncedAt, &l.ErrorMessage, &l.StockSyncMode, &l.DescriptionHTML, &l.Metadata,
		&l.CreatedAt, &l.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find product listing by id: %w", err)
	}
	return &l, nil
}

// FindByProductAndIntegration returns the listing for a product on a given integration.
func (r *ProductListingRepository) FindByProductAndIntegration(ctx context.Context, tx pgx.Tx, productID, integrationID uuid.UUID) (*model.ProductListing, error) {
	var l model.ProductListing
	err := tx.QueryRow(ctx,
		`SELECT id, tenant_id, product_id, integration_id, external_id,
		        status, url, price_override, stock_override,
		        sync_status, last_synced_at, error_message, stock_sync_mode, description_html, metadata,
		        created_at, updated_at
		 FROM product_listings WHERE product_id = $1 AND integration_id = $2`, productID, integrationID,
	).Scan(
		&l.ID, &l.TenantID, &l.ProductID, &l.IntegrationID, &l.ExternalID,
		&l.Status, &l.URL, &l.PriceOverride, &l.StockOverride,
		&l.SyncStatus, &l.LastSyncedAt, &l.ErrorMessage, &l.StockSyncMode, &l.DescriptionHTML, &l.Metadata,
		&l.CreatedAt, &l.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find product listing by product and integration: %w", err)
	}
	return &l, nil
}

// FindByExternalIDAndIntegration returns the listing matching the given external ID and integration.
func (r *ProductListingRepository) FindByExternalIDAndIntegration(ctx context.Context, tx pgx.Tx, externalID string, integrationID uuid.UUID) (*model.ProductListing, error) {
	var l model.ProductListing
	err := tx.QueryRow(ctx,
		`SELECT id, tenant_id, product_id, integration_id, external_id,
		        status, url, price_override, stock_override,
		        sync_status, last_synced_at, error_message, stock_sync_mode, description_html, metadata,
		        created_at, updated_at
		 FROM product_listings WHERE external_id = $1 AND integration_id = $2`, externalID, integrationID,
	).Scan(
		&l.ID, &l.TenantID, &l.ProductID, &l.IntegrationID, &l.ExternalID,
		&l.Status, &l.URL, &l.PriceOverride, &l.StockOverride,
		&l.SyncStatus, &l.LastSyncedAt, &l.ErrorMessage, &l.StockSyncMode, &l.DescriptionHTML, &l.Metadata,
		&l.CreatedAt, &l.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find product listing by external id and integration: %w", err)
	}
	return &l, nil
}

// ListByProduct returns all listings for the given product.
func (r *ProductListingRepository) ListByProduct(ctx context.Context, tx pgx.Tx, productID uuid.UUID) ([]*model.ProductListing, error) {
	rows, err := tx.Query(ctx,
		`SELECT id, tenant_id, product_id, integration_id, external_id,
		        status, url, price_override, stock_override,
		        sync_status, last_synced_at, error_message, stock_sync_mode, description_html, metadata,
		        created_at, updated_at
		 FROM product_listings WHERE product_id = $1 ORDER BY created_at`, productID,
	)
	if err != nil {
		return nil, fmt.Errorf("list product listings by product: %w", err)
	}
	defer rows.Close()

	var listings []*model.ProductListing
	for rows.Next() {
		var l model.ProductListing
		if err := rows.Scan(
			&l.ID, &l.TenantID, &l.ProductID, &l.IntegrationID, &l.ExternalID,
			&l.Status, &l.URL, &l.PriceOverride, &l.StockOverride,
			&l.SyncStatus, &l.LastSyncedAt, &l.ErrorMessage, &l.StockSyncMode, &l.DescriptionHTML, &l.Metadata,
			&l.CreatedAt, &l.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan product listing: %w", err)
		}
		listings = append(listings, &l)
	}
	return listings, rows.Err()
}

// ListByIntegration returns all listings for the given integration.
func (r *ProductListingRepository) ListByIntegration(ctx context.Context, tx pgx.Tx, integrationID uuid.UUID) ([]*model.ProductListing, error) {
	rows, err := tx.Query(ctx,
		`SELECT id, tenant_id, product_id, integration_id, external_id,
		        status, url, price_override, stock_override,
		        sync_status, last_synced_at, error_message, stock_sync_mode, description_html, metadata,
		        created_at, updated_at
		 FROM product_listings WHERE integration_id = $1 ORDER BY created_at`, integrationID,
	)
	if err != nil {
		return nil, fmt.Errorf("list product listings by integration: %w", err)
	}
	defer rows.Close()

	var listings []*model.ProductListing
	for rows.Next() {
		var l model.ProductListing
		if err := rows.Scan(
			&l.ID, &l.TenantID, &l.ProductID, &l.IntegrationID, &l.ExternalID,
			&l.Status, &l.URL, &l.PriceOverride, &l.StockOverride,
			&l.SyncStatus, &l.LastSyncedAt, &l.ErrorMessage, &l.StockSyncMode, &l.DescriptionHTML, &l.Metadata,
			&l.CreatedAt, &l.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan product listing: %w", err)
		}
		listings = append(listings, &l)
	}
	return listings, rows.Err()
}

// ListAutoSyncByProduct returns all active listings with auto stock sync for a given product.
func (r *ProductListingRepository) ListAutoSyncByProduct(ctx context.Context, tx pgx.Tx, productID uuid.UUID) ([]*model.ProductListing, error) {
	rows, err := tx.Query(ctx,
		`SELECT id, tenant_id, product_id, integration_id, external_id,
		        status, url, price_override, stock_override,
		        sync_status, last_synced_at, error_message, stock_sync_mode, description_html, metadata,
		        created_at, updated_at
		 FROM product_listings
		 WHERE product_id = $1 AND status = 'active' AND external_id IS NOT NULL AND stock_sync_mode = 'auto'
		 ORDER BY created_at`, productID,
	)
	if err != nil {
		return nil, fmt.Errorf("list auto-sync product listings: %w", err)
	}
	defer rows.Close()

	var listings []*model.ProductListing
	for rows.Next() {
		var l model.ProductListing
		if err := rows.Scan(
			&l.ID, &l.TenantID, &l.ProductID, &l.IntegrationID, &l.ExternalID,
			&l.Status, &l.URL, &l.PriceOverride, &l.StockOverride,
			&l.SyncStatus, &l.LastSyncedAt, &l.ErrorMessage, &l.StockSyncMode, &l.DescriptionHTML, &l.Metadata,
			&l.CreatedAt, &l.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan auto-sync product listing: %w", err)
		}
		listings = append(listings, &l)
	}
	return listings, rows.Err()
}

// Delete removes a product listing by its ID.
func (r *ProductListingRepository) Delete(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	ct, err := tx.Exec(ctx, "DELETE FROM product_listings WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete product listing: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("product listing not found")
	}
	return nil
}
