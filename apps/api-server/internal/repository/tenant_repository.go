package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

type TenantRepository struct {
	pool *pgxpool.Pool
}

func NewTenantRepository(pool *pgxpool.Pool) *TenantRepository {
	return &TenantRepository{pool: pool}
}

// FindBySlug uses SECURITY DEFINER function (bypasses RLS).
func (r *TenantRepository) FindBySlug(ctx context.Context, slug string) (*model.Tenant, error) {
	var t model.Tenant
	err := r.pool.QueryRow(ctx,
		"SELECT id, name, slug, plan FROM find_tenant_by_slug($1)", slug,
	).Scan(&t.ID, &t.Name, &t.Slug, &t.Plan)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find tenant by slug: %w", err)
	}
	return &t, nil
}

// FindByID finds a tenant by ID within a WithTenant transaction.
func (r *TenantRepository) FindByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.Tenant, error) {
	var t model.Tenant
	err := tx.QueryRow(ctx,
		`SELECT id, name, slug, plan, settings, created_at, updated_at
		 FROM tenants WHERE id = $1`, id,
	).Scan(&t.ID, &t.Name, &t.Slug, &t.Plan, &t.Settings, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find tenant by id: %w", err)
	}
	return &t, nil
}

// SlugExists checks if a tenant slug is already taken.
func (r *TenantRepository) SlugExists(ctx context.Context, slug string) (bool, error) {
	t, err := r.FindBySlug(ctx, slug)
	if err != nil {
		return false, err
	}
	return t != nil, nil
}

// Create inserts a new tenant within a WithTenant transaction.
func (r *TenantRepository) Create(ctx context.Context, tx pgx.Tx, tenant *model.Tenant) error {
	return tx.QueryRow(ctx,
		`INSERT INTO tenants (id, name, slug, plan)
		 VALUES ($1, $2, $3, $4)
		 RETURNING created_at, updated_at`,
		tenant.ID, tenant.Name, tenant.Slug, tenant.Plan,
	).Scan(&tenant.CreatedAt, &tenant.UpdatedAt)
}

func (r *TenantRepository) GetSettings(ctx context.Context, tx pgx.Tx, id uuid.UUID) (json.RawMessage, error) {
	var settings json.RawMessage
	err := tx.QueryRow(ctx, "SELECT settings FROM tenants WHERE id = $1", id).Scan(&settings)
	if err != nil {
		return nil, fmt.Errorf("get tenant settings: %w", err)
	}
	return settings, nil
}

func (r *TenantRepository) UpdateSettings(ctx context.Context, tx pgx.Tx, id uuid.UUID, settings json.RawMessage) error {
	ct, err := tx.Exec(ctx, "UPDATE tenants SET settings = $1, updated_at = NOW() WHERE id = $2", settings, id)
	if err != nil {
		return fmt.Errorf("update tenant settings: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("tenant not found")
	}
	return nil
}
