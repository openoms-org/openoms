package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

type BundleRepository struct{}

func NewBundleRepository() *BundleRepository {
	return &BundleRepository{}
}

func (r *BundleRepository) Create(ctx context.Context, tx pgx.Tx, bundle *model.ProductBundle) error {
	return tx.QueryRow(ctx,
		`INSERT INTO product_bundles (id, tenant_id, bundle_product_id, component_product_id, component_variant_id, quantity, position)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING created_at, updated_at`,
		bundle.ID, bundle.TenantID, bundle.BundleProductID, bundle.ComponentProductID,
		bundle.ComponentVariantID, bundle.Quantity, bundle.Position,
	).Scan(&bundle.CreatedAt, &bundle.UpdatedAt)
}

func (r *BundleRepository) FindByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.ProductBundle, error) {
	var b model.ProductBundle
	err := tx.QueryRow(ctx,
		`SELECT pb.id, pb.tenant_id, pb.bundle_product_id, pb.component_product_id,
		        pb.component_variant_id, pb.quantity, pb.position, pb.created_at, pb.updated_at,
		        p.name, p.sku, p.stock_quantity
		 FROM product_bundles pb
		 JOIN products p ON p.id = pb.component_product_id
		 WHERE pb.id = $1`, id,
	).Scan(&b.ID, &b.TenantID, &b.BundleProductID, &b.ComponentProductID,
		&b.ComponentVariantID, &b.Quantity, &b.Position, &b.CreatedAt, &b.UpdatedAt,
		&b.ComponentName, &b.ComponentSKU, &b.ComponentStock)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find bundle by id: %w", err)
	}
	return &b, nil
}

func (r *BundleRepository) ListByBundleProduct(ctx context.Context, tx pgx.Tx, bundleProductID uuid.UUID) ([]model.ProductBundle, error) {
	rows, err := tx.Query(ctx,
		`SELECT pb.id, pb.tenant_id, pb.bundle_product_id, pb.component_product_id,
		        pb.component_variant_id, pb.quantity, pb.position, pb.created_at, pb.updated_at,
		        p.name, p.sku, p.stock_quantity
		 FROM product_bundles pb
		 JOIN products p ON p.id = pb.component_product_id
		 WHERE pb.bundle_product_id = $1
		 ORDER BY pb.position ASC, pb.created_at ASC`, bundleProductID,
	)
	if err != nil {
		return nil, fmt.Errorf("list bundle components: %w", err)
	}
	defer rows.Close()

	var bundles []model.ProductBundle
	for rows.Next() {
		var b model.ProductBundle
		if err := rows.Scan(&b.ID, &b.TenantID, &b.BundleProductID, &b.ComponentProductID,
			&b.ComponentVariantID, &b.Quantity, &b.Position, &b.CreatedAt, &b.UpdatedAt,
			&b.ComponentName, &b.ComponentSKU, &b.ComponentStock); err != nil {
			return nil, fmt.Errorf("scan bundle: %w", err)
		}
		bundles = append(bundles, b)
	}
	return bundles, rows.Err()
}

func (r *BundleRepository) ListByComponentProduct(ctx context.Context, tx pgx.Tx, componentProductID uuid.UUID) ([]model.ProductBundle, error) {
	rows, err := tx.Query(ctx,
		`SELECT pb.id, pb.tenant_id, pb.bundle_product_id, pb.component_product_id,
		        pb.component_variant_id, pb.quantity, pb.position, pb.created_at, pb.updated_at,
		        p.name, p.sku, p.stock_quantity
		 FROM product_bundles pb
		 JOIN products p ON p.id = pb.component_product_id
		 WHERE pb.component_product_id = $1
		 ORDER BY pb.position ASC`, componentProductID,
	)
	if err != nil {
		return nil, fmt.Errorf("list bundles by component: %w", err)
	}
	defer rows.Close()

	var bundles []model.ProductBundle
	for rows.Next() {
		var b model.ProductBundle
		if err := rows.Scan(&b.ID, &b.TenantID, &b.BundleProductID, &b.ComponentProductID,
			&b.ComponentVariantID, &b.Quantity, &b.Position, &b.CreatedAt, &b.UpdatedAt,
			&b.ComponentName, &b.ComponentSKU, &b.ComponentStock); err != nil {
			return nil, fmt.Errorf("scan bundle: %w", err)
		}
		bundles = append(bundles, b)
	}
	return bundles, rows.Err()
}

func (r *BundleRepository) Update(ctx context.Context, tx pgx.Tx, id uuid.UUID, req model.UpdateBundleComponentRequest) error {
	var setClauses []string
	var args []any
	argIdx := 1

	if req.Quantity != nil {
		setClauses = append(setClauses, fmt.Sprintf("quantity = $%d", argIdx))
		args = append(args, *req.Quantity)
		argIdx++
	}
	if req.Position != nil {
		setClauses = append(setClauses, fmt.Sprintf("position = $%d", argIdx))
		args = append(args, *req.Position)
		argIdx++
	}

	if len(setClauses) == 0 {
		return nil
	}

	setClauses = append(setClauses, "updated_at = NOW()")
	args = append(args, id)

	query := fmt.Sprintf("UPDATE product_bundles SET %s WHERE id = $%d",
		strings.Join(setClauses, ", "), argIdx)

	ct, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update bundle component: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("bundle component not found")
	}
	return nil
}

func (r *BundleRepository) Delete(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	ct, err := tx.Exec(ctx, "DELETE FROM product_bundles WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete bundle component: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("bundle component not found")
	}
	return nil
}
