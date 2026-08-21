package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// #nosec G101 -- SQL column list, not credential material (token_hash is a column name).
const apiTokenColumns = `id, tenant_id, user_id, name, token_hash, last_used_at, revoked_at, created_at`

// APITokenRepository persists hashed owner API tokens.
type APITokenRepository struct {
	pool *pgxpool.Pool
}

// NewAPITokenRepository constructs a repository backed by the app pool.
func NewAPITokenRepository(pool *pgxpool.Pool) *APITokenRepository {
	return &APITokenRepository{pool: pool}
}

// Insert stores a hashed token under the tenant RLS scope.
func (r *APITokenRepository) Insert(ctx context.Context, tenantID uuid.UUID, token *model.APIToken) error {
	return database.WithTenant(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx,
			`INSERT INTO api_tokens (id, tenant_id, user_id, name, token_hash)
			 VALUES ($1, $2, $3, $4, $5)
			 RETURNING created_at`,
			token.ID, token.TenantID, token.UserID, token.Name, token.TokenHash,
		).Scan(&token.CreatedAt)
	})
}

// ListByUser returns active tokens for the owning user.
func (r *APITokenRepository) ListByUser(ctx context.Context, tenantID, userID uuid.UUID) ([]model.APIToken, error) {
	var tokens []model.APIToken
	err := database.WithTenant(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx,
			`SELECT `+apiTokenColumns+`
			 FROM api_tokens
			 WHERE user_id = $1 AND revoked_at IS NULL
			 ORDER BY created_at DESC`,
			userID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			tok, scanErr := scanAPIToken(rows)
			if scanErr != nil {
				return scanErr
			}
			tokens = append(tokens, *tok)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, fmt.Errorf("list API tokens: %w", err)
	}
	if tokens == nil {
		tokens = []model.APIToken{}
	}
	return tokens, nil
}

// Revoke sets revoked_at. Returns false when the row is missing or already revoked.
func (r *APITokenRepository) Revoke(ctx context.Context, tenantID, userID, id uuid.UUID) (bool, error) {
	var revoked bool
	err := database.WithTenant(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx,
			`UPDATE api_tokens
			 SET revoked_at = NOW()
			 WHERE id = $1 AND user_id = $2 AND revoked_at IS NULL`,
			id, userID)
		if err != nil {
			return err
		}
		revoked = tag.RowsAffected() > 0
		return nil
	})
	if err != nil {
		return false, fmt.Errorf("revoke API token: %w", err)
	}
	return revoked, nil
}

// FindActiveByHash uses the SECURITY DEFINER auth function so lookup works
// before tenant context is known — same pattern as find_user_for_auth.
func (r *APITokenRepository) FindActiveByHash(ctx context.Context, tokenHash string) (*model.APIToken, error) {
	tok, err := scanAPIToken(r.pool.QueryRow(ctx,
		`SELECT `+apiTokenColumns+` FROM find_api_token_for_auth($1)`, tokenHash))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find API token: %w", err)
	}
	return tok, nil
}

// TouchLastUsed records the most recent successful authentication.
func (r *APITokenRepository) TouchLastUsed(ctx context.Context, tenantID, id uuid.UUID) error {
	return database.WithTenant(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `UPDATE api_tokens SET last_used_at = NOW() WHERE id = $1 AND revoked_at IS NULL`, id)
		return err
	})
}

func scanAPIToken(row pgx.Row) (*model.APIToken, error) {
	var t model.APIToken
	if err := row.Scan(&t.ID, &t.TenantID, &t.UserID, &t.Name, &t.TokenHash, &t.LastUsedAt, &t.RevokedAt, &t.CreatedAt); err != nil {
		return nil, err
	}
	return &t, nil
}

// TenantUserLookup loads a user inside WithTenant (RLS-scoped).
type TenantUserLookup struct {
	Pool  *pgxpool.Pool
	Users UserRepo
}

// FindByID implements service.APITokenUserLookup.
func (l TenantUserLookup) FindByID(ctx context.Context, tenantID, id uuid.UUID) (*model.User, error) {
	var user *model.User
	err := database.WithTenant(ctx, l.Pool, tenantID, func(tx pgx.Tx) error {
		var findErr error
		user, findErr = l.Users.FindByID(ctx, tx, id)
		return findErr
	})
	if err != nil {
		return nil, err
	}
	return user, nil
}

// TenantRoleLookup loads a custom role inside WithTenant (RLS-scoped).
type TenantRoleLookup struct {
	Pool  *pgxpool.Pool
	Roles RoleRepo
}

// FindByID implements service.APITokenRoleLookup.
func (l TenantRoleLookup) FindByID(ctx context.Context, tenantID, id uuid.UUID) (*model.Role, error) {
	var role *model.Role
	err := database.WithTenant(ctx, l.Pool, tenantID, func(tx pgx.Tx) error {
		var findErr error
		role, findErr = l.Roles.FindByID(ctx, tx, id)
		return findErr
	})
	if err != nil {
		return nil, err
	}
	return role, nil
}
