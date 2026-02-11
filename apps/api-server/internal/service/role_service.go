package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

var (
	ErrRoleNotFound     = errors.New("role not found")
	ErrRoleIsSystem     = errors.New("system roles cannot be deleted")
	ErrRoleDuplicateName = errors.New("role with this name already exists")
)

// RoleService provides business logic for roles.
type RoleService struct {
	roleRepo  repository.RoleRepo
	auditRepo repository.AuditRepo
	pool      *pgxpool.Pool
}

// NewRoleService creates a new RoleService.
func NewRoleService(roleRepo repository.RoleRepo, auditRepo repository.AuditRepo, pool *pgxpool.Pool) *RoleService {
	return &RoleService{
		roleRepo:  roleRepo,
		auditRepo: auditRepo,
		pool:      pool,
	}
}

// EnsureSystemRoles creates the default system roles for a tenant if they don't exist.
func (s *RoleService) EnsureSystemRoles(ctx context.Context, tenantID uuid.UUID) error {
	return database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		systemRoles := []struct {
			Name        string
			Description string
			Permissions []string
		}{
			{
				Name:        "Właściciel",
				Description: "Pełen dostęp do systemu",
				Permissions: model.SystemRoleOwnerPermissions,
			},
			{
				Name:        "Administrator",
				Description: "Zarządzanie systemem i danymi",
				Permissions: model.SystemRoleAdminPermissions,
			},
			{
				Name:        "Pracownik",
				Description: "Podstawowy dostęp do operacji",
				Permissions: model.SystemRoleMemberPermissions,
			},
		}

		for _, sr := range systemRoles {
			existing, err := s.roleRepo.FindByName(ctx, tx, sr.Name)
			if err != nil {
				return err
			}
			if existing != nil {
				continue
			}

			desc := sr.Description
			role := &model.Role{
				ID:          uuid.New(),
				TenantID:    tenantID,
				Name:        sr.Name,
				Description: &desc,
				IsSystem:    true,
				Permissions: sr.Permissions,
			}
			if err := s.roleRepo.Create(ctx, tx, role); err != nil {
				return err
			}
		}
		return nil
	})
}

// List lists all roles for a tenant.
func (s *RoleService) List(ctx context.Context, tenantID uuid.UUID, filter model.RoleListFilter) (model.ListResponse[model.Role], error) {
	var resp model.ListResponse[model.Role]
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		roles, total, err := s.roleRepo.List(ctx, tx, filter)
		if err != nil {
			return err
		}
		if roles == nil {
			roles = []model.Role{}
		}
		resp = model.ListResponse[model.Role]{
			Items:  roles,
			Total:  total,
			Limit:  filter.Limit,
			Offset: filter.Offset,
		}
		return nil
	})
	return resp, err
}

// Get retrieves a single role by ID.
func (s *RoleService) Get(ctx context.Context, tenantID, roleID uuid.UUID) (*model.Role, error) {
	var role *model.Role
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var err error
		role, err = s.roleRepo.FindByID(ctx, tx, roleID)
		return err
	})
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, ErrRoleNotFound
	}
	return role, nil
}

// Create creates a new custom role.
func (s *RoleService) Create(ctx context.Context, tenantID uuid.UUID, req model.CreateRoleRequest, actorID uuid.UUID, ip string) (*model.Role, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	req.Name = model.StripHTMLTags(req.Name)

	role := &model.Role{
		ID:          uuid.New(),
		TenantID:    tenantID,
		Name:        req.Name,
		Description: req.Description,
		IsSystem:    false,
		Permissions: req.Permissions,
	}
	if role.Permissions == nil {
		role.Permissions = []string{}
	}

	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		if err := s.roleRepo.Create(ctx, tx, role); err != nil {
			if isDuplicateKeyError(err) {
				return ErrRoleDuplicateName
			}
			return err
		}
		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "role.created",
			EntityType: "role",
			EntityID:   role.ID,
			Changes:    map[string]string{"name": req.Name},
			IPAddress:  ip,
		})
	})
	if err != nil {
		return nil, err
	}
	return role, nil
}

// Update updates an existing role.
func (s *RoleService) Update(ctx context.Context, tenantID, roleID uuid.UUID, req model.UpdateRoleRequest, actorID uuid.UUID, ip string) (*model.Role, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	var role *model.Role
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		existing, err := s.roleRepo.FindByID(ctx, tx, roleID)
		if err != nil {
			return err
		}
		if existing == nil {
			return ErrRoleNotFound
		}

		if err := s.roleRepo.Update(ctx, tx, roleID, req); err != nil {
			if isDuplicateKeyError(err) {
				return ErrRoleDuplicateName
			}
			return err
		}

		role, err = s.roleRepo.FindByID(ctx, tx, roleID)
		if err != nil {
			return err
		}

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "role.updated",
			EntityType: "role",
			EntityID:   roleID,
			IPAddress:  ip,
		})
	})
	if err != nil {
		return nil, err
	}
	return role, err
}

// Delete removes a role.
func (s *RoleService) Delete(ctx context.Context, tenantID, roleID uuid.UUID, actorID uuid.UUID, ip string) error {
	return database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		role, err := s.roleRepo.FindByID(ctx, tx, roleID)
		if err != nil {
			return err
		}
		if role == nil {
			return ErrRoleNotFound
		}
		if role.IsSystem {
			return ErrRoleIsSystem
		}

		if err := s.roleRepo.Delete(ctx, tx, roleID); err != nil {
			return err
		}

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "role.deleted",
			EntityType: "role",
			EntityID:   roleID,
			Changes:    map[string]string{"name": role.Name},
			IPAddress:  ip,
		})
	})
}

// FindByID retrieves a role by ID (used by middleware for permission checks).
func (s *RoleService) FindByID(ctx context.Context, tenantID, roleID uuid.UUID) (*model.Role, error) {
	return s.Get(ctx, tenantID, roleID)
}
