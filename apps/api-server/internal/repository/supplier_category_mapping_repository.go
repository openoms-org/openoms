package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

type SupplierCategoryMappingRepository struct{}

func NewSupplierCategoryMappingRepository() *SupplierCategoryMappingRepository {
	return &SupplierCategoryMappingRepository{}
}

func (r *SupplierCategoryMappingRepository) ListBySupplier(ctx context.Context, tx pgx.Tx, supplierID uuid.UUID) ([]model.SupplierCategoryMapping, error) {
	rows, err := tx.Query(ctx,
		`SELECT id, tenant_id, supplier_id, source_category, category_id, auto_matched, confirmed, created_at
		 FROM supplier_category_mappings
		 WHERE supplier_id = $1
		 ORDER BY source_category`, supplierID,
	)
	if err != nil {
		return nil, fmt.Errorf("list category mappings: %w", err)
	}
	defer rows.Close()

	var mappings []model.SupplierCategoryMapping
	for rows.Next() {
		var m model.SupplierCategoryMapping
		if err := rows.Scan(&m.ID, &m.TenantID, &m.SupplierID, &m.SourceCategory,
			&m.CategoryID, &m.AutoMatched, &m.Confirmed, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan category mapping: %w", err)
		}
		mappings = append(mappings, m)
	}
	return mappings, rows.Err()
}

func (r *SupplierCategoryMappingRepository) FindBySourceCategory(ctx context.Context, tx pgx.Tx, supplierID uuid.UUID, sourceCategory string) (*model.SupplierCategoryMapping, error) {
	var m model.SupplierCategoryMapping
	err := tx.QueryRow(ctx,
		`SELECT id, tenant_id, supplier_id, source_category, category_id, auto_matched, confirmed, created_at
		 FROM supplier_category_mappings
		 WHERE supplier_id = $1 AND source_category = $2`, supplierID, sourceCategory,
	).Scan(&m.ID, &m.TenantID, &m.SupplierID, &m.SourceCategory,
		&m.CategoryID, &m.AutoMatched, &m.Confirmed, &m.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find category mapping: %w", err)
	}
	return &m, nil
}

func (r *SupplierCategoryMappingRepository) Upsert(ctx context.Context, tx pgx.Tx, m *model.SupplierCategoryMapping) error {
	return tx.QueryRow(ctx,
		`INSERT INTO supplier_category_mappings (id, tenant_id, supplier_id, source_category, category_id, auto_matched, confirmed)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 ON CONFLICT (tenant_id, supplier_id, source_category)
		 DO UPDATE SET category_id = EXCLUDED.category_id, auto_matched = EXCLUDED.auto_matched, confirmed = EXCLUDED.confirmed
		 RETURNING id, created_at`,
		m.ID, m.TenantID, m.SupplierID, m.SourceCategory, m.CategoryID, m.AutoMatched, m.Confirmed,
	).Scan(&m.ID, &m.CreatedAt)
}

func (r *SupplierCategoryMappingRepository) Delete(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	ct, err := tx.Exec(ctx, "DELETE FROM supplier_category_mappings WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete category mapping: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("category mapping not found")
	}
	return nil
}
