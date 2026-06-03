package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// PlatformAdminRepository persists platform-admin identities. The
// platform_admins table is NOT tenant-scoped and has no RLS, so all queries
// run directly on the pool, outside any tenant context. This is safe because
// database.WithTenant sets app.current_tenant_id transaction-locally (it never
// leaks across the implicit single-statement transactions used here); do not
// switch this table to RLS or set the tenant GUC session-locally anywhere.
type PlatformAdminRepository struct {
	pool *pgxpool.Pool
}

// NewPlatformAdminRepository creates a PlatformAdminRepository backed by the pool.
func NewPlatformAdminRepository(pool *pgxpool.Pool) *PlatformAdminRepository {
	return &PlatformAdminRepository{pool: pool}
}

// GetByUserID returns the platform admin for the given user, or pgx.ErrNoRows
// when the user is not a platform admin.
func (r *PlatformAdminRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*model.PlatformAdmin, error) {
	var a model.PlatformAdmin
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, permissions, created_at, updated_at
		   FROM platform_admins
		  WHERE user_id = $1`, userID).
		Scan(&a.ID, &a.UserID, &a.Permissions, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// Create inserts a new platform admin with the given permissions.
func (r *PlatformAdminRepository) Create(ctx context.Context, userID uuid.UUID, permissions []string) (*model.PlatformAdmin, error) {
	if permissions == nil {
		permissions = []string{}
	}
	var a model.PlatformAdmin
	err := r.pool.QueryRow(ctx,
		`INSERT INTO platform_admins (user_id, permissions)
		 VALUES ($1, $2)
		 RETURNING id, user_id, permissions, created_at, updated_at`, userID, permissions).
		Scan(&a.ID, &a.UserID, &a.Permissions, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create platform admin: %w", err)
	}
	return &a, nil
}
