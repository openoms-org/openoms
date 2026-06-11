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

// roleColumns is the canonical column list for SELECTing a role row.
const roleColumns = "id, tenant_id, name, description, is_system, permissions, created_at, updated_at"

// scanRole scans a single role row in roleColumns order.
func scanRole(row pgx.Row) (model.Role, error) {
	var role model.Role
	err := row.Scan(
		&role.ID, &role.TenantID, &role.Name, &role.Description,
		&role.IsSystem, &role.Permissions, &role.CreatedAt, &role.UpdatedAt,
	)
	return role, err
}

// List returns a paginated list of roles matching the filter.
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
		"SELECT "+roleColumns+" FROM roles %s LIMIT $1 OFFSET $2",
		orderByClause,
	)

	rows, err := tx.Query(ctx, query, filter.Limit, filter.Offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list roles: %w", err)
	}
	defer rows.Close()

	var roles []model.Role
	for rows.Next() {
		role, err := scanRole(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan role: %w", err)
		}
		roles = append(roles, role)
	}
	return roles, total, rows.Err()
}

// FindByID returns a role by its ID.
func (r *RoleRepository) FindByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.Role, error) {
	role, err := scanRole(tx.QueryRow(ctx,
		"SELECT "+roleColumns+" FROM roles WHERE id = $1", id,
	))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find role by id: %w", err)
	}
	return &role, nil
}

// FindByName returns a role by its name.
func (r *RoleRepository) FindByName(ctx context.Context, tx pgx.Tx, name string) (*model.Role, error) {
	role, err := scanRole(tx.QueryRow(ctx,
		"SELECT "+roleColumns+" FROM roles WHERE name = $1", name,
	))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find role by name: %w", err)
	}
	return &role, nil
}

// Create inserts a new role.
func (r *RoleRepository) Create(ctx context.Context, tx pgx.Tx, role *model.Role) error {
	return tx.QueryRow(ctx,
		`INSERT INTO roles (id, tenant_id, name, description, is_system, permissions)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING created_at, updated_at`,
		role.ID, role.TenantID, role.Name, role.Description,
		role.IsSystem, role.Permissions,
	).Scan(&role.CreatedAt, &role.UpdatedAt)
}

// Update applies partial updates to a role.
func (r *RoleRepository) Update(ctx context.Context, tx pgx.Tx, id uuid.UUID, req model.UpdateRoleRequest) error {
	ub := NewUpdateBuilder()
	SetPtr(ub, "name", req.Name)
	SetPtr(ub, "description", req.Description)
	if req.Permissions != nil {
		ub.Set("permissions", req.Permissions)
	}

	if ub.IsEmpty() {
		return nil
	}

	ub.SetRaw("updated_at = NOW()")

	query := fmt.Sprintf("UPDATE roles SET %s WHERE id = $%d",
		ub.SetClause(), ub.NextArgIdx())
	args := append(ub.Args(), id)

	ct, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update role: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("role not found")
	}
	return nil
}

// Delete removes a role by its ID.
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
