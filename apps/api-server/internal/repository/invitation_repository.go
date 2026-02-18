package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

type InvitationRepository struct{}

func NewInvitationRepository() *InvitationRepository {
	return &InvitationRepository{}
}

func (r *InvitationRepository) Create(ctx context.Context, tx pgx.Tx, inv *model.Invitation) error {
	_, err := tx.Exec(ctx,
		`INSERT INTO invitations (id, tenant_id, email, token, role, expires_at, created_by)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		inv.ID, inv.TenantID, inv.Email, inv.Token, inv.Role, inv.ExpiresAt, inv.CreatedBy,
	)
	return err
}

func (r *InvitationRepository) List(ctx context.Context, tx pgx.Tx, limit, offset int) ([]model.Invitation, int, error) {
	var total int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM invitations`).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := tx.Query(ctx,
		`SELECT id, tenant_id, email, role, expires_at, used_at, created_by, created_at
		 FROM invitations ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var invitations []model.Invitation
	for rows.Next() {
		var inv model.Invitation
		if err := rows.Scan(&inv.ID, &inv.TenantID, &inv.Email, &inv.Role, &inv.ExpiresAt, &inv.UsedAt, &inv.CreatedBy, &inv.CreatedAt); err != nil {
			return nil, 0, err
		}
		invitations = append(invitations, inv)
	}
	return invitations, total, rows.Err()
}

// FindByToken uses SECURITY DEFINER function (bypasses RLS) since registration has no tenant context.
func (r *InvitationRepository) FindByToken(ctx context.Context, pool *pgxpool.Pool, token string) (*model.Invitation, error) {
	row := pool.QueryRow(ctx, `SELECT id, tenant_id, email, role, expires_at, used_at FROM find_invitation_by_token($1)`, token)
	var inv model.Invitation
	err := row.Scan(&inv.ID, &inv.TenantID, &inv.Email, &inv.Role, &inv.ExpiresAt, &inv.UsedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	inv.Token = token
	return &inv, nil
}

// MarkUsed uses SECURITY DEFINER function (bypasses RLS).
func (r *InvitationRepository) MarkUsed(ctx context.Context, pool *pgxpool.Pool, token string) error {
	_, err := pool.Exec(ctx, `SELECT use_invitation($1)`, token)
	return err
}

func (r *InvitationRepository) Delete(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	_, err := tx.Exec(ctx, `DELETE FROM invitations WHERE id = $1`, id)
	return err
}
