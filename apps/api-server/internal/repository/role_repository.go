package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// RoleRepository implements RoleRepo.
type RoleRepository struct{}

// NewRoleRepository creates a new RoleRepository.
func NewRoleRepository() *RoleRepository {
	return &RoleRepository{}
}

func (r *RoleRepository) List(ctx context.Context, tx pgx.Tx, filter model.RoleListFilter) ([]model.Role, int, error) {
	var total int
	if err := tx.QueryRow(ctx, "SELECT COUNT(*) FROM roles").Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count roles: %w", err)
	}

	allowedSortColumns := map[string]string{
		"created_at": "created_at",
		"name":       "name",
	}
	orderByClause := model.BuildOrderByClause(filter.SortBy, filter.SortOrder, allowedSortColumns)

	query := fmt.Sprintf(
		`SELECT id, tenant_id, name, description, is_system, permissions, created_at, updated_at
		 FROM roles %s LIMIT $1 OFFSET $2`,
		orderByClause,
	)

	rows, err := tx.Query(ctx, query, filter.Limit, filter.Offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list roles: %w", err)
	}
	defer rows.Close()

	var roles []model.Role
	for rows.Next() {
		var role model.Role
		if err := rows.Scan(
			&role.ID, &role.TenantID, &role.Name, &role.Description,
			&role.IsSystem, &role.Permissions, &role.CreatedAt, &role.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan role: %w", err)
		}
		roles = append(roles, role)
	}
	return roles, total, rows.Err()
}

func (r *RoleRepository) FindByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.Role, error) {
	var role model.Role
	err := tx.QueryRow(ctx,
		`SELECT id, tenant_id, name, description, is_system, permissions, created_at, updated_at
		 FROM roles WHERE id = $1`, id,
	).Scan(
		&role.ID, &role.TenantID, &role.Name, &role.Description,
		&role.IsSystem, &role.Permissions, &role.CreatedAt, &role.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find role by id: %w", err)
	}
	return &role, nil
}

func (r *RoleRepository) FindByName(ctx context.Context, tx pgx.Tx, name string) (*model.Role, error) {
	var role model.Role
	err := tx.QueryRow(ctx,
		`SELECT id, tenant_id, name, description, is_system, permissions, created_at, updated_at
		 FROM roles WHERE name = $1`, name,
	).Scan(
		&role.ID, &role.TenantID, &role.Name, &role.Description,
		&role.IsSystem, &role.Permissions, &role.CreatedAt, &role.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find role by name: %w", err)
	}
	return &role, nil
}

func (r *RoleRepository) Create(ctx context.Context, tx pgx.Tx, role *model.Role) error {
	return tx.QueryRow(ctx,
		`INSERT INTO roles (id, tenant_id, name, description, is_system, permissions)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING created_at, updated_at`,
		role.ID, role.TenantID, role.Name, role.Description,
		role.IsSystem, role.Permissions,
	).Scan(&role.CreatedAt, &role.UpdatedAt)
}

func (r *RoleRepository) Update(ctx context.Context, tx pgx.Tx, id uuid.UUID, req model.UpdateRoleRequest) error {
	var setClauses []string
	var args []any
	argIdx := 1

	if req.Name != nil {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, *req.Name)
		argIdx++
	}
	if req.Description != nil {
		setClauses = append(setClauses, fmt.Sprintf("description = $%d", argIdx))
		args = append(args, *req.Description)
		argIdx++
	}
	if req.Permissions != nil {
		setClauses = append(setClauses, fmt.Sprintf("permissions = $%d", argIdx))
		args = append(args, req.Permissions)
		argIdx++
	}

	if len(setClauses) == 0 {
		return nil
	}

	setClauses = append(setClauses, "updated_at = NOW()")

	query := fmt.Sprintf("UPDATE roles SET %s WHERE id = $%d",
		joinStrings(setClauses, ", "), argIdx)
	args = append(args, id)

	ct, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update role: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("role not found")
	}
	return nil
}

func (r *RoleRepository) Delete(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	ct, err := tx.Exec(ctx, "DELETE FROM roles WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete role: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("role not found")
	}
	return nil
}

// joinStrings joins string slices — avoids importing strings package in this file.
func joinStrings(parts []string, sep string) string {
	result := ""
	for i, p := range parts {
		if i > 0 {
			result += sep
		}
		result += p
	}
	return result
}
