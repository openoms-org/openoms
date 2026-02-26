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

// MarkTokenUsed records a license token JTI as consumed after successful registration.
func (r *LicenseRepository) MarkTokenUsed(ctx context.Context, pool *pgxpool.Pool, jti, tenantID uuid.UUID, email, plan string) error {
	_, err := pool.Exec(ctx, `SELECT mark_license_token_used($1, $2, $3, $4)`, jti, tenantID, email, plan)
	if err != nil {
		return fmt.Errorf("mark license token used: %w", err)
	}
	return nil
}
