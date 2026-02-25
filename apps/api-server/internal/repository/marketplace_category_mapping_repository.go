package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

type MarketplaceCategoryMappingRepository struct{}

func NewMarketplaceCategoryMappingRepository() *MarketplaceCategoryMappingRepository {
	return &MarketplaceCategoryMappingRepository{}
}

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

func (r *MarketplaceCategoryMappingRepository) Upsert(ctx context.Context, tx pgx.Tx, m *model.MarketplaceCategoryMapping) error {
	return tx.QueryRow(ctx,
		`INSERT INTO marketplace_category_mappings (id, tenant_id, integration_id, external_category_id, external_category_name, category_id, auto_created, confirmed)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 ON CONFLICT (tenant_id, integration_id, external_category_id)
		 DO UPDATE SET external_category_name = EXCLUDED.external_category_name, category_id = EXCLUDED.category_id, auto_created = EXCLUDED.auto_created, confirmed = EXCLUDED.confirmed
		 RETURNING id, created_at, updated_at`,
		m.ID, m.TenantID, m.IntegrationID, m.ExternalCategoryID, m.ExternalCategoryName, m.CategoryID, m.AutoCreated, m.Confirmed,
	).Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt)
}

func (r *MarketplaceCategoryMappingRepository) Delete(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	ct, err := tx.Exec(ctx, "DELETE FROM marketplace_category_mappings WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete marketplace category mapping: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("marketplace category mapping not found")
	}
	return nil
}
