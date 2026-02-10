package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

type ProductListingRepository struct{}

func NewProductListingRepository() *ProductListingRepository {
	return &ProductListingRepository{}
}

func (r *ProductListingRepository) Create(ctx context.Context, tx pgx.Tx, listing *model.ProductListing) error {
	return tx.QueryRow(ctx,
		`INSERT INTO product_listings (
			id, tenant_id, product_id, integration_id, external_id,
			status, url, price_override, stock_override,
			sync_status, last_synced_at, error_message, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING created_at, updated_at`,
		listing.ID, listing.TenantID, listing.ProductID, listing.IntegrationID, listing.ExternalID,
		listing.Status, listing.URL, listing.PriceOverride, listing.StockOverride,
		listing.SyncStatus, listing.LastSyncedAt, listing.ErrorMessage, listing.Metadata,
	).Scan(&listing.CreatedAt, &listing.UpdatedAt)
}

func (r *ProductListingRepository) Update(ctx context.Context, tx pgx.Tx, id uuid.UUID, req *model.UpdateProductListingRequest) error {
	var setClauses []string
	var args []any
	argIdx := 1

	if req.ExternalID != nil {
		setClauses = append(setClauses, fmt.Sprintf("external_id = $%d", argIdx))
		args = append(args, *req.ExternalID)
		argIdx++
	}
	if req.Status != nil {
		setClauses = append(setClauses, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *req.Status)
		argIdx++
	}
	if req.URL != nil {
		setClauses = append(setClauses, fmt.Sprintf("url = $%d", argIdx))
		args = append(args, *req.URL)
		argIdx++
	}
	if req.PriceOverride != nil {
		setClauses = append(setClauses, fmt.Sprintf("price_override = $%d", argIdx))
		args = append(args, *req.PriceOverride)
		argIdx++
	}
	if req.StockOverride != nil {
		setClauses = append(setClauses, fmt.Sprintf("stock_override = $%d", argIdx))
		args = append(args, *req.StockOverride)
		argIdx++
	}
	if req.SyncStatus != nil {
		setClauses = append(setClauses, fmt.Sprintf("sync_status = $%d", argIdx))
		args = append(args, *req.SyncStatus)
		argIdx++
	}
	if req.ErrorMessage != nil {
		setClauses = append(setClauses, fmt.Sprintf("error_message = $%d", argIdx))
		args = append(args, *req.ErrorMessage)
		argIdx++
	}
	if req.Metadata != nil {
		setClauses = append(setClauses, fmt.Sprintf("metadata = $%d", argIdx))
		args = append(args, *req.Metadata)
		argIdx++
	}

	if len(setClauses) == 0 {
		return nil
	}

	setClauses = append(setClauses, "updated_at = NOW()")
	query := fmt.Sprintf("UPDATE product_listings SET %s WHERE id = $%d",
		strings.Join(setClauses, ", "), argIdx)
	args = append(args, id)

	ct, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update product listing: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("product listing not found")
	}
	return nil
}

func (r *ProductListingRepository) GetByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.ProductListing, error) {
	var l model.ProductListing
	err := tx.QueryRow(ctx,
		`SELECT id, tenant_id, product_id, integration_id, external_id,
		        status, url, price_override, stock_override,
		        sync_status, last_synced_at, error_message, metadata,
		        created_at, updated_at
		 FROM product_listings WHERE id = $1`, id,
	).Scan(
		&l.ID, &l.TenantID, &l.ProductID, &l.IntegrationID, &l.ExternalID,
		&l.Status, &l.URL, &l.PriceOverride, &l.StockOverride,
		&l.SyncStatus, &l.LastSyncedAt, &l.ErrorMessage, &l.Metadata,
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

func (r *ProductListingRepository) FindByProductAndIntegration(ctx context.Context, tx pgx.Tx, productID, integrationID uuid.UUID) (*model.ProductListing, error) {
	var l model.ProductListing
	err := tx.QueryRow(ctx,
		`SELECT id, tenant_id, product_id, integration_id, external_id,
		        status, url, price_override, stock_override,
		        sync_status, last_synced_at, error_message, metadata,
		        created_at, updated_at
		 FROM product_listings WHERE product_id = $1 AND integration_id = $2`, productID, integrationID,
	).Scan(
		&l.ID, &l.TenantID, &l.ProductID, &l.IntegrationID, &l.ExternalID,
		&l.Status, &l.URL, &l.PriceOverride, &l.StockOverride,
		&l.SyncStatus, &l.LastSyncedAt, &l.ErrorMessage, &l.Metadata,
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

func (r *ProductListingRepository) ListByProduct(ctx context.Context, tx pgx.Tx, productID uuid.UUID) ([]*model.ProductListing, error) {
	rows, err := tx.Query(ctx,
		`SELECT id, tenant_id, product_id, integration_id, external_id,
		        status, url, price_override, stock_override,
		        sync_status, last_synced_at, error_message, metadata,
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
			&l.SyncStatus, &l.LastSyncedAt, &l.ErrorMessage, &l.Metadata,
			&l.CreatedAt, &l.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan product listing: %w", err)
		}
		listings = append(listings, &l)
	}
	return listings, rows.Err()
}

func (r *ProductListingRepository) ListByIntegration(ctx context.Context, tx pgx.Tx, integrationID uuid.UUID) ([]*model.ProductListing, error) {
	rows, err := tx.Query(ctx,
		`SELECT id, tenant_id, product_id, integration_id, external_id,
		        status, url, price_override, stock_override,
		        sync_status, last_synced_at, error_message, metadata,
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
			&l.SyncStatus, &l.LastSyncedAt, &l.ErrorMessage, &l.Metadata,
			&l.CreatedAt, &l.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan product listing: %w", err)
		}
		listings = append(listings, &l)
	}
	return listings, rows.Err()
}

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
