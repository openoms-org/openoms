package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// UserWithPassword includes the password hash (for auth only, never in API responses).
type UserWithPassword struct {
	model.User
	PasswordHash string
	TOTPSecret   *string
	TOTPEnabled  bool
}

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

// FindForAuth uses SECURITY DEFINER function (bypasses RLS).
func (r *UserRepository) FindForAuth(ctx context.Context, email string, tenantID uuid.UUID) (*UserWithPassword, error) {
	var u UserWithPassword
	err := r.pool.QueryRow(ctx,
		"SELECT id, tenant_id, email, name, password_hash, role, role_id, created_at, updated_at, totp_secret, totp_enabled FROM find_user_for_auth($1, $2)",
		email, tenantID,
	).Scan(&u.ID, &u.TenantID, &u.Email, &u.Name, &u.PasswordHash, &u.Role, &u.RoleID, &u.CreatedAt, &u.UpdatedAt, &u.TOTPSecret, &u.TOTPEnabled)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find user for auth: %w", err)
	}

	return &u, nil
}

// FindByID finds a user by ID within a WithTenant transaction.
func (r *UserRepository) FindByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.User, error) {
	var u model.User
	err := tx.QueryRow(ctx,
		`SELECT id, tenant_id, email, name, role, role_id, last_login_at, last_logout_at, created_at, updated_at
		 FROM users WHERE id = $1`, id,
	).Scan(&u.ID, &u.TenantID, &u.Email, &u.Name, &u.Role, &u.RoleID, &u.LastLoginAt, &u.LastLogoutAt, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find user by id: %w", err)
	}
	return &u, nil
}

// List returns all users for the current tenant.
func (r *UserRepository) List(ctx context.Context, tx pgx.Tx) ([]model.User, error) {
	rows, err := tx.Query(ctx,
		`SELECT id, tenant_id, email, name, role, role_id, last_login_at, last_logout_at, created_at, updated_at
		 FROM users ORDER BY created_at`)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var users []model.User
	for rows.Next() {
		var u model.User
		if err := rows.Scan(&u.ID, &u.TenantID, &u.Email, &u.Name, &u.Role, &u.RoleID, &u.LastLoginAt, &u.LastLogoutAt, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// Create inserts a new user with password hash.
func (r *UserRepository) Create(ctx context.Context, tx pgx.Tx, user *model.User, passwordHash string) error {
	return tx.QueryRow(ctx,
		`INSERT INTO users (id, tenant_id, email, name, password_hash, role)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING created_at, updated_at`,
		user.ID, user.TenantID, user.Email, user.Name, passwordHash, user.Role,
	).Scan(&user.CreatedAt, &user.UpdatedAt)
}

// UpdateRole updates a user's role.
func (r *UserRepository) UpdateRole(ctx context.Context, tx pgx.Tx, id uuid.UUID, role string) error {
	ct, err := tx.Exec(ctx, "UPDATE users SET role = $1 WHERE id = $2", role, id)
	if err != nil {
		return fmt.Errorf("update user role: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

// UpdateName updates a user's name.
func (r *UserRepository) UpdateName(ctx context.Context, tx pgx.Tx, id uuid.UUID, name string) error {
	ct, err := tx.Exec(ctx, "UPDATE users SET name = $1 WHERE id = $2", name, id)
	if err != nil {
		return fmt.Errorf("update user name: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

// UpdateRoleID updates a user's RBAC role_id.
func (r *UserRepository) UpdateRoleID(ctx context.Context, tx pgx.Tx, id uuid.UUID, roleID *uuid.UUID) error {
	ct, err := tx.Exec(ctx, "UPDATE users SET role_id = $1 WHERE id = $2", roleID, id)
	if err != nil {
		return fmt.Errorf("update user role_id: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

// UpdateLastLogin sets last_login_at to now.
func (r *UserRepository) UpdateLastLogin(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	_, err := tx.Exec(ctx, "UPDATE users SET last_login_at = NOW() WHERE id = $1", id)
	return err
}

// UpdateLastLogout sets last_logout_at to now.
func (r *UserRepository) UpdateLastLogout(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	_, err := tx.Exec(ctx, "UPDATE users SET last_logout_at = NOW() WHERE id = $1", id)
	return err
}

// Delete removes a user.
func (r *UserRepository) Delete(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	ct, err := tx.Exec(ctx, "DELETE FROM users WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

// CountByRole counts users with the given role in the current tenant.
func (r *UserRepository) CountByRole(ctx context.Context, tx pgx.Tx, role string) (int, error) {
	var count int
	err := tx.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE role = $1", role).Scan(&count)
	return count, err
}

// SetTOTPSecret sets the encrypted TOTP secret for a user.
func (r *UserRepository) SetTOTPSecret(ctx context.Context, tx pgx.Tx, id uuid.UUID, encryptedSecret string) error {
	_, err := tx.Exec(ctx, "UPDATE users SET totp_secret = $1 WHERE id = $2", encryptedSecret, id)
	return err
}

// EnableTOTP enables TOTP for a user and sets the verified_at timestamp.
func (r *UserRepository) EnableTOTP(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	_, err := tx.Exec(ctx, "UPDATE users SET totp_enabled = TRUE, totp_verified_at = NOW() WHERE id = $1", id)
	return err
}

// DisableTOTP disables TOTP for a user and clears the secret.
func (r *UserRepository) DisableTOTP(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	_, err := tx.Exec(ctx, "UPDATE users SET totp_enabled = FALSE, totp_secret = NULL, totp_verified_at = NULL WHERE id = $1", id)
	return err
}

// GetTOTPStatus returns the TOTP enabled state and verified_at timestamp.
func (r *UserRepository) GetTOTPStatus(ctx context.Context, tx pgx.Tx, id uuid.UUID) (bool, *string, error) {
	var enabled bool
	var verifiedAt *string
	err := tx.QueryRow(ctx,
		"SELECT totp_enabled, totp_verified_at::text FROM users WHERE id = $1", id,
	).Scan(&enabled, &verifiedAt)
	if err != nil {
		return false, nil, fmt.Errorf("get totp status: %w", err)
	}
	return enabled, verifiedAt, nil
}

// GetTOTPSecret returns the encrypted TOTP secret for a user.
func (r *UserRepository) GetTOTPSecret(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*string, error) {
	var secret *string
	err := tx.QueryRow(ctx, "SELECT totp_secret FROM users WHERE id = $1", id).Scan(&secret)
	if err != nil {
		return nil, fmt.Errorf("get totp secret: %w", err)
	}
	return secret, nil
}
