package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

type ProductCategoryRepository struct{}

func NewProductCategoryRepository() *ProductCategoryRepository {
	return &ProductCategoryRepository{}
}

var categoryColumns = `id, tenant_id, parent_id, name, slug, color, icon, position, depth, created_at, updated_at`

func scanCategory(row interface{ Scan(dest ...any) error }) (*model.ProductCategory, error) {
	var c model.ProductCategory
	err := row.Scan(
		&c.ID, &c.TenantID, &c.ParentID, &c.Name, &c.Slug,
		&c.Color, &c.Icon, &c.Position, &c.Depth, &c.CreatedAt, &c.UpdatedAt,
	)
	return &c, err
}

// List returns categories filtered by optional parent_id. If filter.ParentID is nil, returns root categories.
func (r *ProductCategoryRepository) List(ctx context.Context, tx pgx.Tx, filter model.CategoryListFilter) ([]model.ProductCategory, error) {
	var query string
	var args []any

	if filter.IncludeTree {
		// Return all categories for tree building
		query = fmt.Sprintf("SELECT %s FROM product_categories ORDER BY depth, position, name", categoryColumns)
	} else if filter.ParentID != nil {
		query = fmt.Sprintf("SELECT %s FROM product_categories WHERE parent_id = $1 ORDER BY position, name", categoryColumns)
		args = append(args, *filter.ParentID)
	} else {
		query = fmt.Sprintf("SELECT %s FROM product_categories WHERE parent_id IS NULL ORDER BY position, name", categoryColumns)
	}

	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list product categories: %w", err)
	}
	defer rows.Close()

	var categories []model.ProductCategory
	for rows.Next() {
		c, err := scanCategory(rows)
		if err != nil {
			return nil, fmt.Errorf("scan product category: %w", err)
		}
		categories = append(categories, *c)
	}
	return categories, rows.Err()
}

func (r *ProductCategoryRepository) FindByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.ProductCategory, error) {
	c, err := scanCategory(tx.QueryRow(ctx,
		fmt.Sprintf("SELECT %s FROM product_categories WHERE id = $1", categoryColumns), id,
	))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find product category by id: %w", err)
	}
	return c, nil
}

func (r *ProductCategoryRepository) FindBySlug(ctx context.Context, tx pgx.Tx, slug string) (*model.ProductCategory, error) {
	c, err := scanCategory(tx.QueryRow(ctx,
		fmt.Sprintf("SELECT %s FROM product_categories WHERE slug = $1", categoryColumns), slug,
	))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find product category by slug: %w", err)
	}
	return c, nil
}

func (r *ProductCategoryRepository) Create(ctx context.Context, tx pgx.Tx, c *model.ProductCategory) error {
	return tx.QueryRow(ctx,
		`INSERT INTO product_categories (id, tenant_id, parent_id, name, slug, color, icon, position, depth)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 RETURNING created_at, updated_at`,
		c.ID, c.TenantID, c.ParentID, c.Name, c.Slug, c.Color, c.Icon, c.Position, c.Depth,
	).Scan(&c.CreatedAt, &c.UpdatedAt)
}

func (r *ProductCategoryRepository) Update(ctx context.Context, tx pgx.Tx, c *model.ProductCategory) error {
	ct, err := tx.Exec(ctx,
		`UPDATE product_categories
		 SET name = $1, slug = $2, parent_id = $3, color = $4, icon = $5, position = $6, depth = $7, updated_at = NOW()
		 WHERE id = $8`,
		c.Name, c.Slug, c.ParentID, c.Color, c.Icon, c.Position, c.Depth, c.ID,
	)
	if err != nil {
		return fmt.Errorf("update product category: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("product category not found")
	}
	return nil
}

func (r *ProductCategoryRepository) Delete(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	ct, err := tx.Exec(ctx, "DELETE FROM product_categories WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete product category: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("product category not found")
	}
	return nil
}

// GetDescendantIDs returns the id of the given category plus all its descendant IDs using a recursive CTE.
func (r *ProductCategoryRepository) GetDescendantIDs(ctx context.Context, tx pgx.Tx, id uuid.UUID) ([]uuid.UUID, error) {
	rows, err := tx.Query(ctx, `
		WITH RECURSIVE descendants AS (
			SELECT id FROM product_categories WHERE id = $1
			UNION ALL
			SELECT c.id FROM product_categories c
			INNER JOIN descendants d ON c.parent_id = d.id
		)
		SELECT id FROM descendants
	`, id)
	if err != nil {
		return nil, fmt.Errorf("get descendant IDs: %w", err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan descendant id: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// FuzzyMatch finds categories whose name contains the given text (case-insensitive).
func (r *ProductCategoryRepository) FuzzyMatch(ctx context.Context, tx pgx.Tx, name string) ([]model.ProductCategory, error) {
	rows, err := tx.Query(ctx,
		fmt.Sprintf("SELECT %s FROM product_categories WHERE name ILIKE '%%' || $1 || '%%' ORDER BY depth, position LIMIT 10", categoryColumns),
		name,
	)
	if err != nil {
		return nil, fmt.Errorf("fuzzy match categories: %w", err)
	}
	defer rows.Close()

	var categories []model.ProductCategory
	for rows.Next() {
		c, err := scanCategory(rows)
		if err != nil {
			return nil, fmt.Errorf("scan category fuzzy match: %w", err)
		}
		categories = append(categories, *c)
	}
	return categories, rows.Err()
}

// CountBySlug returns how many categories have the given slug prefix (for uniqueness).
func (r *ProductCategoryRepository) CountBySlug(ctx context.Context, tx pgx.Tx, slug string) (int, error) {
	var count int
	err := tx.QueryRow(ctx,
		"SELECT COUNT(*) FROM product_categories WHERE slug = $1 OR slug LIKE $2",
		slug, slug+"-%",
	).Scan(&count)
	return count, err
}
