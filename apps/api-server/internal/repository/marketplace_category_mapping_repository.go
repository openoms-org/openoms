package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// MarketplaceCategoryMappingRepository implements MarketplaceCategoryMappingRepo.
type MarketplaceCategoryMappingRepository struct{}

// NewMarketplaceCategoryMappingRepository creates a new repository instance.
func NewMarketplaceCategoryMappingRepository() *MarketplaceCategoryMappingRepository {
	return &MarketplaceCategoryMappingRepository{}
}

// ListByIntegration returns all mappings for the given integration.
func (r *MarketplaceCategoryMappingRepository) ListByIntegration(ctx context.Context, tx pgx.Tx, integrationID uuid.UUID) ([]model.MarketplaceCategoryMapping, error) {
	rows, err := tx.Query(ctx,
		`SELECT id, tenant_id, integration_id, external_category_id, external_category_name, category_id, auto_created, confirmed, created_at, updated_at
		 FROM marketplace_category_mappings
		 WHERE integration_id = $1
		 ORDER BY external_category_name`, integrationID,
	)
	if err != nil {
		return nil, fmt.Errorf("list marketplace category mappings: %w", err)
	}
	defer rows.Close()

	var mappings []model.MarketplaceCategoryMapping
	for rows.Next() {
		var m model.MarketplaceCategoryMapping
		if err := rows.Scan(&m.ID, &m.TenantID, &m.IntegrationID, &m.ExternalCategoryID,
			&m.ExternalCategoryName, &m.CategoryID, &m.AutoCreated, &m.Confirmed, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan marketplace category mapping: %w", err)
		}
		mappings = append(mappings, m)
	}
	return mappings, rows.Err()
}

// FindByExternalID finds a mapping by external category ID within an integration.
func (r *MarketplaceCategoryMappingRepository) FindByExternalID(ctx context.Context, tx pgx.Tx, integrationID uuid.UUID, externalCategoryID string) (*model.MarketplaceCategoryMapping, error) {
	var m model.MarketplaceCategoryMapping
	err := tx.QueryRow(ctx,
		`SELECT id, tenant_id, integration_id, external_category_id, external_category_name, category_id, auto_created, confirmed, created_at, updated_at
		 FROM marketplace_category_mappings
		 WHERE integration_id = $1 AND external_category_id = $2`, integrationID, externalCategoryID,
	).Scan(&m.ID, &m.TenantID, &m.IntegrationID, &m.ExternalCategoryID,
		&m.ExternalCategoryName, &m.CategoryID, &m.AutoCreated, &m.Confirmed, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find marketplace category mapping: %w", err)
	}
	return &m, nil
}

// Upsert creates or updates a mapping, preserving confirmed flags on existing confirmed rows.
func (r *MarketplaceCategoryMappingRepository) Upsert(ctx context.Context, tx pgx.Tx, m *model.MarketplaceCategoryMapping) error {
	return tx.QueryRow(ctx,
		`INSERT INTO marketplace_category_mappings (id, tenant_id, integration_id, external_category_id, external_category_name, category_id, auto_created, confirmed)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 ON CONFLICT (tenant_id, integration_id, external_category_id)
		 DO UPDATE SET
		     external_category_name = COALESCE(NULLIF(EXCLUDED.external_category_name, ''), marketplace_category_mappings.external_category_name),
		     category_id = COALESCE(EXCLUDED.category_id, marketplace_category_mappings.category_id),
		     auto_created = CASE WHEN marketplace_category_mappings.confirmed THEN marketplace_category_mappings.auto_created ELSE EXCLUDED.auto_created END,
		     confirmed = CASE WHEN marketplace_category_mappings.confirmed THEN true ELSE EXCLUDED.confirmed END,
		     updated_at = NOW()
		 RETURNING id, created_at, updated_at`,
		m.ID, m.TenantID, m.IntegrationID, m.ExternalCategoryID, m.ExternalCategoryName, m.CategoryID, m.AutoCreated, m.Confirmed,
	).Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt)
}

// Delete removes a mapping by ID scoped to the given integration.
func (r *MarketplaceCategoryMappingRepository) Delete(ctx context.Context, tx pgx.Tx, integrationID, id uuid.UUID) error {
	ct, err := tx.Exec(ctx, "DELETE FROM marketplace_category_mappings WHERE id = $1 AND integration_id = $2", id, integrationID)
	if err != nil {
		return fmt.Errorf("delete marketplace category mapping: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("marketplace category mapping not found")
	}
	return nil
}
