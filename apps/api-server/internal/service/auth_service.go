package service

import (
	"context"
	"encoding/json"
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
	ErrSlugTaken          = errors.New("tenant slug is already taken")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrTenantNotFound     = errors.New("tenant not found")
	ErrUserNotFound       = errors.New("user not found")
)

type AuthService struct {
	userRepo     repository.UserRepo
	tenantRepo   repository.TenantRepo
	auditRepo    repository.AuditRepo
	tokenService *TokenService
	passwordSvc  *PasswordService
	pool         *pgxpool.Pool
}

func NewAuthService(
	userRepo repository.UserRepo,
	tenantRepo repository.TenantRepo,
	auditRepo repository.AuditRepo,
	tokenSvc *TokenService,
	passwordSvc *PasswordService,
	pool *pgxpool.Pool,
) *AuthService {
	return &AuthService{
		userRepo:     userRepo,
		tenantRepo:   tenantRepo,
		auditRepo:    auditRepo,
		tokenService: tokenSvc,
		passwordSvc:  passwordSvc,
		pool:         pool,
	}
}

// Register creates a new tenant and owner user, returns tokens.
func (s *AuthService) Register(ctx context.Context, req model.RegisterRequest, ipAddress string) (*model.TokenResponse, string, error) {
	if err := req.Validate(); err != nil {
		return nil, "", NewValidationError(err)
	}

	if err := s.passwordSvc.ValidateStrength(req.Password); err != nil {
		return nil, "", NewValidationError(err)
	}

	exists, err := s.tenantRepo.SlugExists(ctx, req.TenantSlug)
	if err != nil {
		return nil, "", fmt.Errorf("check slug: %w", err)
	}
	if exists {
		return nil, "", ErrSlugTaken
	}

	tenantID := uuid.New()
	userID := uuid.New()

	hash, err := s.passwordSvc.Hash(req.Password)
	if err != nil {
		return nil, "", fmt.Errorf("hash password: %w", err)
	}

	tenant := &model.Tenant{
		ID:   tenantID,
		Name: req.TenantName,
		Slug: req.TenantSlug,
		Plan: "free",
	}

	user := &model.User{
		ID:       userID,
		TenantID: tenantID,
		Email:    req.Email,
		Name:     req.Name,
		Role:     "owner",
	}

	err = database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		if err := s.tenantRepo.Create(ctx, tx, tenant); err != nil {
			return fmt.Errorf("create tenant: %w", err)
		}

		// Save default order status config so new tenant has working transitions
		defaultCfg := model.DefaultOrderStatusConfig()
		cfgJSON, err := json.Marshal(defaultCfg)
		if err != nil {
			return fmt.Errorf("marshal default order status config: %w", err)
		}
		initialSettings, err := json.Marshal(map[string]json.RawMessage{
			"order_statuses": cfgJSON,
		})
		if err != nil {
			return fmt.Errorf("marshal initial settings: %w", err)
		}
		if err := s.tenantRepo.UpdateSettings(ctx, tx, tenantID, initialSettings); err != nil {
			return fmt.Errorf("set default settings: %w", err)
		}

		if err := s.userRepo.Create(ctx, tx, user, hash); err != nil {
			return fmt.Errorf("create user: %w", err)
		}
		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     userID,
			Action:     "user.registered",
			EntityType: "user",
			EntityID:   userID,
			IPAddress:  ipAddress,
		})
	})
	if err != nil {
		return nil, "", err
	}

	accessToken, err := s.tokenService.GenerateAccessToken(*user)
	if err != nil {
		return nil, "", fmt.Errorf("generate access token: %w", err)
	}

	refreshToken, err := s.tokenService.GenerateRefreshToken(*user)
	if err != nil {
		return nil, "", fmt.Errorf("generate refresh token: %w", err)
	}

	slog.Info("new tenant registered", "tenant_id", tenantID, "slug", req.TenantSlug, "user_email", req.Email)

	return &model.TokenResponse{
		AccessToken: accessToken,
		ExpiresIn:   3600,
		User:        *user,
		Tenant:      *tenant,
	}, refreshToken, nil
}

// Login authenticates a user and returns tokens.
func (s *AuthService) Login(ctx context.Context, req model.LoginRequest, ipAddress string) (*model.TokenResponse, string, error) {
	if err := req.Validate(); err != nil {
		return nil, "", NewValidationError(err)
	}

	// Find tenant by slug (SECURITY DEFINER, bypasses RLS)
	tenant, err := s.tenantRepo.FindBySlug(ctx, req.TenantSlug)
	if err != nil {
		return nil, "", fmt.Errorf("find tenant: %w", err)
	}
	if tenant == nil {
		return nil, "", ErrInvalidCredentials
	}

	// Find user for auth (SECURITY DEFINER, bypasses RLS)
	userWithPwd, err := s.userRepo.FindForAuth(ctx, req.Email, tenant.ID)
	if err != nil {
		return nil, "", fmt.Errorf("find user: %w", err)
	}
	if userWithPwd == nil {
		return nil, "", ErrInvalidCredentials
	}

	if err := s.passwordSvc.Compare(userWithPwd.PasswordHash, req.Password); err != nil {
		return nil, "", ErrInvalidCredentials
	}

	// Update last_login_at and create audit entry
	err = database.WithTenant(ctx, s.pool, tenant.ID, func(tx pgx.Tx) error {
		if err := s.userRepo.UpdateLastLogin(ctx, tx, userWithPwd.ID); err != nil {
			slog.Warn("failed to update last_login_at", "error", err)
		}
		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenant.ID,
			UserID:     userWithPwd.ID,
			Action:     "user.login",
			EntityType: "user",
			EntityID:   userWithPwd.ID,
			IPAddress:  ipAddress,
		})
	})
	if err != nil {
		slog.Warn("failed to log login", "error", err)
	}

	user := userWithPwd.User
	accessToken, err := s.tokenService.GenerateAccessToken(user)
	if err != nil {
		return nil, "", fmt.Errorf("generate access token: %w", err)
	}

	refreshToken, err := s.tokenService.GenerateRefreshToken(user)
	if err != nil {
		return nil, "", fmt.Errorf("generate refresh token: %w", err)
	}

	return &model.TokenResponse{
		AccessToken: accessToken,
		ExpiresIn:   3600,
		User:        user,
		Tenant:      *tenant,
	}, refreshToken, nil
}

// Refresh validates a refresh token and issues new tokens.
func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*model.TokenResponse, string, error) {
	claims, err := s.tokenService.ValidateToken(refreshToken)
	if err != nil {
		return nil, "", fmt.Errorf("invalid refresh token: %w", err)
	}

	if claims.Type != "refresh" {
		return nil, "", fmt.Errorf("invalid refresh token: token is not a refresh token")
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return nil, "", fmt.Errorf("invalid user ID in token: %w", err)
	}

	var user *model.User
	var tenant *model.Tenant
	err = database.WithTenant(ctx, s.pool, claims.TenantID, func(tx pgx.Tx) error {
		var findErr error
		user, findErr = s.userRepo.FindByID(ctx, tx, userID)
		if findErr != nil {
			return findErr
		}
		tenant, findErr = s.tenantRepo.FindByID(ctx, tx, claims.TenantID)
		return findErr
	})
	if err != nil {
		return nil, "", fmt.Errorf("find user: %w", err)
	}
	if user == nil {
		return nil, "", ErrUserNotFound
	}

	accessToken, err := s.tokenService.GenerateAccessToken(*user)
	if err != nil {
		return nil, "", fmt.Errorf("generate access token: %w", err)
	}

	newRefreshToken, err := s.tokenService.GenerateRefreshToken(*user)
	if err != nil {
		return nil, "", fmt.Errorf("generate refresh token: %w", err)
	}

	resp := &model.TokenResponse{
		AccessToken: accessToken,
		ExpiresIn:   3600,
		User:        *user,
	}
	if tenant != nil {
		resp.Tenant = *tenant
	}

	return resp, newRefreshToken, nil
}
