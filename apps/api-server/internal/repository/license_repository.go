package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// LicenseRepository implements LicenseRepo using SECURITY DEFINER functions.
type LicenseRepository struct{}

// NewLicenseRepository creates a new license repository instance.
func NewLicenseRepository() *LicenseRepository {
	return &LicenseRepository{}
}

// IsTokenUsed checks if a license token JTI has already been consumed.
func (r *LicenseRepository) IsTokenUsed(ctx context.Context, pool *pgxpool.Pool, jti uuid.UUID) (bool, error) {
	var used bool
	err := pool.QueryRow(ctx, `SELECT check_license_token_used($1)`, jti).Scan(&used)
	if err != nil {
		return false, fmt.Errorf("check license token used: %w", err)
	}
	return used, nil
}

// MarkTokenUsed atomically claims a license token JTI. Returns true if the
// token was successfully claimed (inserted), false if it was already used.
func (r *LicenseRepository) MarkTokenUsed(ctx context.Context, pool *pgxpool.Pool, jti uuid.UUID, tenantID *uuid.UUID, email, plan string) (bool, error) {
	var claimed bool
	err := pool.QueryRow(ctx, `SELECT mark_license_token_used($1, $2, $3, $4)`, jti, tenantID, email, plan).Scan(&claimed)
	if err != nil {
		return false, fmt.Errorf("mark license token used: %w", err)
	}
	return claimed, nil
}

// UpdateClaimedTenant sets the tenant_id on a previously claimed token.
func (r *LicenseRepository) UpdateClaimedTenant(ctx context.Context, pool *pgxpool.Pool, jti, tenantID uuid.UUID) error {
	_, err := pool.Exec(ctx, `SELECT update_license_token_tenant($1, $2)`, jti, tenantID)
	if err != nil {
		return fmt.Errorf("update license token tenant: %w", err)
	}
	return nil
}
