package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

var (
	// ErrCannotDeleteSelf is returned when a user attempts to delete their own account.
	ErrCannotDeleteSelf = errors.New("cannot delete your own account")
	// ErrCannotDeleteLastOwner is returned when deleting the sole remaining owner of a tenant.
	ErrCannotDeleteLastOwner = errors.New("cannot delete the last owner of the tenant")
	// ErrDuplicateEmail is returned when a user with the same email already exists.
	ErrDuplicateEmail = errors.New("email already exists in this tenant")
	// ErrOwnerRoleEscalation is returned when a non-owner attempts to assign the owner role.
	ErrOwnerRoleEscalation = errors.New("only owners can assign the owner role")
	// ErrUserLimitExceeded is returned when a tenant has reached its maximum number of users.
	ErrUserLimitExceeded = errors.New("user limit exceeded")
	// ErrInvalidCurrentPassword is returned when a password change uses the wrong current password.
	ErrInvalidCurrentPassword = errors.New("invalid current password")
)

// UserService handles user management within a tenant.
type UserService struct {
	userRepo    repository.UserRepo
	auditRepo   repository.AuditRepo
	passwordSvc *PasswordService
	pool        *pgxpool.Pool
}

// NewUserService creates a new UserService.
func NewUserService(
	userRepo repository.UserRepo,
	auditRepo repository.AuditRepo,
	passwordSvc *PasswordService,
	pool *pgxpool.Pool,
) *UserService {
	return &UserService{
		userRepo:    userRepo,
		auditRepo:   auditRepo,
		passwordSvc: passwordSvc,
		pool:        pool,
	}
}

// GetCurrentUser returns the authenticated user's profile.
func (s *UserService) GetCurrentUser(ctx context.Context, tenantID, userID uuid.UUID) (*model.User, error) {
	var user *model.User
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var err error
		user, err = s.userRepo.FindByID(ctx, tx, userID)
		return err
	})
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	return user, nil
}

// ListUsers returns all users belonging to a tenant.
func (s *UserService) ListUsers(ctx context.Context, tenantID uuid.UUID) ([]model.User, error) {
	var users []model.User
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var err error
		users, err = s.userRepo.List(ctx, tx)
		return err
	})
	return users, err
}

// CreateUser adds a new user to a tenant.
func (s *UserService) CreateUser(ctx context.Context, tenantID uuid.UUID, req model.CreateUserRequest, actorID uuid.UUID, ip string) (*model.User, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	if err := s.passwordSvc.ValidateStrength(req.Password); err != nil {
		return nil, NewValidationError(err)
	}

	hash, err := s.passwordSvc.Hash(req.Password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user := &model.User{
		ID:       uuid.New(),
		TenantID: tenantID,
		Email:    req.Email,
		Name:     req.Name,
		Role:     req.Role,
	}

	err = database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		// Atomic plan limit check inside transaction to prevent TOCTOU race
		if req.MaxUsers > 0 {
			count, err := s.userRepo.Count(ctx, tx)
			if err != nil {
				return fmt.Errorf("count users for limit check: %w", err)
			}
			if count >= req.MaxUsers {
				return ErrUserLimitExceeded
			}
		}

		if err := s.userRepo.Create(ctx, tx, user, hash); err != nil {
			return err
		}
		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "user.created",
			EntityType: "user",
			EntityID:   user.ID,
			Changes:    map[string]string{"email": req.Email, "role": req.Role},
			IPAddress:  ip,
		})
	})
	if err != nil {
		if isDuplicateKeyError(err) {
			return nil, ErrDuplicateEmail
		}
		return nil, err
	}

	slog.Info("new user created",
		"user_id", user.ID,
	)

	return user, nil
}

// ChangePassword updates the current user's password after verifying the current password.
func (s *UserService) ChangePassword(ctx context.Context, tenantID, userID uuid.UUID, req model.ChangePasswordRequest, actorID uuid.UUID, ip string) error {
	if err := req.Validate(); err != nil {
		return NewValidationError(err)
	}

	if err := s.passwordSvc.ValidateStrength(req.NewPassword); err != nil {
		return NewValidationError(err)
	}

	return database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		return s.changePasswordInTx(ctx, tx, tenantID, userID, req, actorID, ip)
	})
}

func (s *UserService) changePasswordInTx(ctx context.Context, tx pgx.Tx, tenantID, userID uuid.UUID, req model.ChangePasswordRequest, actorID uuid.UUID, ip string) error {
	passwordHash, err := s.userRepo.FindPasswordHashByID(ctx, tx, userID)
	if err != nil {
		return err
	}
	if passwordHash == nil {
		return ErrUserNotFound
	}

	if err := s.passwordSvc.Compare(*passwordHash, req.CurrentPassword); err != nil {
		return ErrInvalidCurrentPassword
	}

	newHash, err := s.passwordSvc.Hash(req.NewPassword)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	if err := s.userRepo.UpdatePassword(ctx, tx, userID, newHash); err != nil {
		return err
	}

	return s.auditRepo.Log(ctx, tx, model.AuditEntry{
		TenantID:   tenantID,
		UserID:     actorID,
		Action:     "user.password_changed",
		EntityType: "user",
		EntityID:   userID,
		Changes:    map[string]string{"password": "changed"},
		IPAddress:  ip,
	})
}

// UpdateUser modifies an existing user's profile and role.
func (s *UserService) UpdateUser(ctx context.Context, tenantID, userID uuid.UUID, req model.UpdateUserRequest, actorID uuid.UUID, actorRole string, ip string) (*model.User, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	// Only owners can assign the owner role
	if req.Role != nil && *req.Role == "owner" && actorRole != "owner" {
		return nil, ErrOwnerRoleEscalation
	}

	var user *model.User
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var err error
		user, err = s.userRepo.FindByID(ctx, tx, userID)
		if err != nil {
			return err
		}
		if user == nil {
			return ErrUserNotFound
		}

		changes := map[string]any{}
		if req.Name != nil {
			changes["name"] = map[string]string{"old": user.Name, "new": *req.Name}
			if err := s.userRepo.UpdateName(ctx, tx, userID, *req.Name); err != nil {
				return err
			}
			user.Name = *req.Name
		}
		if req.Role != nil {
			changes["role"] = map[string]string{"old": user.Role, "new": *req.Role}
			if err := s.userRepo.UpdateRole(ctx, tx, userID, *req.Role); err != nil {
				return err
			}
			user.Role = *req.Role
		}
		if req.RoleID != nil {
			changes["role_id"] = *req.RoleID
			if err := s.userRepo.UpdateRoleID(ctx, tx, userID, req.RoleID); err != nil {
				return err
			}
			user.RoleID = req.RoleID
		}
		if req.Language != nil {
			changes["language"] = map[string]any{"old": user.Language, "new": *req.Language}
			if err := s.userRepo.UpdateLanguage(ctx, tx, userID, req.Language); err != nil {
				return fmt.Errorf("update language: %w", err)
			}
			user.Language = req.Language
		}

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "user.updated",
			EntityType: "user",
			EntityID:   userID,
			Changes:    changes,
			IPAddress:  ip,
		})
	})
	return user, err
}

// DeleteUser removes a user from a tenant.
func (s *UserService) DeleteUser(ctx context.Context, tenantID, targetID, actorID uuid.UUID, ip string) error {
	if targetID == actorID {
		return ErrCannotDeleteSelf
	}

	return database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		user, err := s.userRepo.FindByID(ctx, tx, targetID)
		if err != nil {
			return err
		}
		if user == nil {
			return ErrUserNotFound
		}

		if user.Role == "owner" {
			count, err := s.userRepo.CountByRole(ctx, tx, "owner")
			if err != nil {
				return err
			}
			if count <= 1 {
				return ErrCannotDeleteLastOwner
			}
		}

		if err := s.userRepo.Delete(ctx, tx, targetID); err != nil {
			return err
		}

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "user.deleted",
			EntityType: "user",
			EntityID:   targetID,
			Changes:    map[string]string{"email": user.Email},
			IPAddress:  ip,
		})
	})
}
