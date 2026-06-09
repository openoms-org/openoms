package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// ErrExternalWorkflowTokenNotFound is returned when a token hash does not resolve.
var ErrExternalWorkflowTokenNotFound = errors.New("external workflow token not found")

// ExternalWorkflowTokenRepository is tenant-scoped (write methods take a pgx.Tx from
// database.WithTenant). The callback path resolves a token cross-tenant on the privileged
// pool first (FindByHashCrossTenant) to learn the tenant, then re-scopes via WithTenant.
type ExternalWorkflowTokenRepository struct{}

// NewExternalWorkflowTokenRepository constructs the repository.
func NewExternalWorkflowTokenRepository() *ExternalWorkflowTokenRepository {
	return &ExternalWorkflowTokenRepository{}
}

// #nosec G101 -- SQL column list, not credential material (token_hash is a column name).
const externalWorkflowTokenColumns = `id, tenant_id, integration_id, token_hash, scopes, expires_at, last_used_at, created_at`

// Issue stores a new token (the caller hashes the raw token first) and returns it.
func (r *ExternalWorkflowTokenRepository) Issue(ctx context.Context, tx pgx.Tx, t model.ExternalWorkflowToken) (*model.ExternalWorkflowToken, error) {
	// The scopes column is NOT NULL DEFAULT '{}': a nil slice would send SQL NULL and violate
	// the constraint, so coalesce to an empty slice (resolve-only token).
	scopes := t.Scopes
	if scopes == nil {
		scopes = []string{}
	}
	out, err := scanExternalWorkflowToken(tx.QueryRow(ctx,
		`INSERT INTO external_workflow_tokens (tenant_id, integration_id, token_hash, scopes, expires_at)
		 VALUES ($1,$2,$3,$4,$5) RETURNING `+externalWorkflowTokenColumns,
		t.TenantID, t.IntegrationID, t.TokenHash, scopes, t.ExpiresAt))
	if err != nil {
		return nil, fmt.Errorf("issue external workflow token: %w", err)
	}
	return out, nil
}

// FindByHashCrossTenant resolves a token hash WITHOUT a tenant context — it runs on the
// privileged pool (bypassing RLS) so the callback (which has no JWT/tenant) can learn which
// tenant a token belongs to. The caller then re-scopes all further work via WithTenant.
func (r *ExternalWorkflowTokenRepository) FindByHashCrossTenant(ctx context.Context, q Querier, tokenHash string) (*model.ExternalWorkflowToken, error) {
	out, err := scanExternalWorkflowToken(q.QueryRow(ctx,
		`SELECT `+externalWorkflowTokenColumns+` FROM external_workflow_tokens WHERE token_hash = $1`, tokenHash))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrExternalWorkflowTokenNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find external workflow token: %w", err)
	}
	return out, nil
}

// TouchLastUsed records that a token was just used (tenant-scoped).
func (r *ExternalWorkflowTokenRepository) TouchLastUsed(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	_, err := tx.Exec(ctx, `UPDATE external_workflow_tokens SET last_used_at = now() WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("touch external workflow token: %w", err)
	}
	return nil
}

func scanExternalWorkflowToken(row pgx.Row) (*model.ExternalWorkflowToken, error) {
	var t model.ExternalWorkflowToken
	if err := row.Scan(&t.ID, &t.TenantID, &t.IntegrationID, &t.TokenHash, &t.Scopes,
		&t.ExpiresAt, &t.LastUsedAt, &t.CreatedAt); err != nil {
		return nil, err
	}
	return &t, nil
}
