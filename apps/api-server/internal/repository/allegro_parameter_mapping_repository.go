package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

type AllegroParameterMappingRepository struct{}

func NewAllegroParameterMappingRepository() *AllegroParameterMappingRepository {
	return &AllegroParameterMappingRepository{}
}

func (r *AllegroParameterMappingRepository) ListBySupplierAndCategory(ctx context.Context, tx pgx.Tx, supplierID uuid.UUID, allegroCategoryID string) ([]model.AllegroParameterMapping, error) {
	rows, err := tx.Query(ctx,
		`SELECT id, tenant_id, supplier_id, allegro_category_id, allegro_param_id, allegro_param_name,
		        source_type, source_key, value_mapping, created_at, updated_at
		 FROM allegro_parameter_mappings
		 WHERE supplier_id = $1 AND allegro_category_id = $2
		 ORDER BY allegro_param_name`, supplierID, allegroCategoryID,
	)
	if err != nil {
		return nil, fmt.Errorf("list allegro parameter mappings: %w", err)
	}
	defer rows.Close()

	var mappings []model.AllegroParameterMapping
	for rows.Next() {
		var m model.AllegroParameterMapping
		if err := rows.Scan(&m.ID, &m.TenantID, &m.SupplierID, &m.AllegroCategoryID,
			&m.AllegroParamID, &m.AllegroParamName, &m.SourceType, &m.SourceKey,
			&m.ValueMapping, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan allegro parameter mapping: %w", err)
		}
		mappings = append(mappings, m)
	}
	return mappings, rows.Err()
}

func (r *AllegroParameterMappingRepository) BulkUpsert(ctx context.Context, tx pgx.Tx, mappings []*model.AllegroParameterMapping) error {
	for _, m := range mappings {
		if err := tx.QueryRow(ctx,
			`INSERT INTO allegro_parameter_mappings (id, tenant_id, supplier_id, allegro_category_id, allegro_param_id, allegro_param_name, source_type, source_key, value_mapping)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			 ON CONFLICT (tenant_id, supplier_id, allegro_category_id, allegro_param_id)
			 DO UPDATE SET allegro_param_name = EXCLUDED.allegro_param_name,
			              source_type = EXCLUDED.source_type,
			              source_key = EXCLUDED.source_key,
			              value_mapping = EXCLUDED.value_mapping,
			              updated_at = NOW()
			 RETURNING id, created_at, updated_at`,
			m.ID, m.TenantID, m.SupplierID, m.AllegroCategoryID, m.AllegroParamID,
			m.AllegroParamName, m.SourceType, m.SourceKey, m.ValueMapping,
		).Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return fmt.Errorf("upsert allegro parameter mapping: %w", err)
		}
	}
	return nil
}

func (r *AllegroParameterMappingRepository) Delete(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	ct, err := tx.Exec(ctx, "DELETE FROM allegro_parameter_mappings WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete allegro parameter mapping: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("allegro parameter mapping not found")
	}
	return nil
}

func (r *AllegroParameterMappingRepository) ListCategoriesForSupplier(ctx context.Context, tx pgx.Tx, supplierID uuid.UUID) ([]string, error) {
	rows, err := tx.Query(ctx,
		`SELECT DISTINCT allegro_category_id FROM allegro_parameter_mappings
		 WHERE supplier_id = $1
		 ORDER BY allegro_category_id`,
		supplierID,
	)
	if err != nil {
		return nil, fmt.Errorf("list allegro mapping categories: %w", err)
	}
	defer rows.Close()

	var categories []string
	for rows.Next() {
		var cat string
		if err := rows.Scan(&cat); err != nil {
			return nil, fmt.Errorf("scan allegro mapping category: %w", err)
		}
		categories = append(categories, cat)
	}
	return categories, rows.Err()
}
