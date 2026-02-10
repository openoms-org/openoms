package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

var (
	ErrCannotDeleteSelf      = errors.New("cannot delete your own account")
	ErrCannotDeleteLastOwner = errors.New("cannot delete the last owner of the tenant")
	ErrDuplicateEmail        = errors.New("email already exists in this tenant")
)

type UserService struct {
	userRepo    repository.UserRepo
	auditRepo   repository.AuditRepo
	passwordSvc *PasswordService
	pool        *pgxpool.Pool
}

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

func (s *UserService) ListUsers(ctx context.Context, tenantID uuid.UUID) ([]model.User, error) {
	var users []model.User
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var err error
		users, err = s.userRepo.List(ctx, tx)
		return err
	})
	return users, err
}

func (s *UserService) CreateUser(ctx context.Context, tenantID uuid.UUID, req model.CreateUserRequest, actorID uuid.UUID, ip string) (*model.User, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation: %w", err)
	}

	// Generate temp password
	tempPassBytes := make([]byte, 16)
	if _, err := rand.Read(tempPassBytes); err != nil {
		return nil, fmt.Errorf("generate temp password: %w", err)
	}
	tempPass := hex.EncodeToString(tempPassBytes)

	hash, err := s.passwordSvc.Hash(tempPass)
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
		"email", user.Email,
	)

	return user, nil
}

func (s *UserService) UpdateUser(ctx context.Context, tenantID, userID uuid.UUID, req model.UpdateUserRequest, actorID uuid.UUID, ip string) (*model.User, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation: %w", err)
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

// isDuplicateKeyError checks for PostgreSQL unique constraint violation (code 23505).
func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "23505") || strings.Contains(err.Error(), "duplicate key")
}
