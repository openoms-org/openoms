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
	"github.com/pquerna/otp/totp"
	"github.com/openoms-org/openoms/apps/api-server/internal/crypto"
	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

var (
	ErrSlugTaken          = errors.New("tenant slug is already taken")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrTenantNotFound     = errors.New("tenant not found")
	ErrUserNotFound       = errors.New("user not found")
	ErrInvalid2FACode     = errors.New("invalid 2FA code")
	ErrInvalid2FAToken    = errors.New("invalid or expired 2FA token")
	Err2FANotEnabled      = errors.New("2FA is not enabled")
	Err2FAAlreadyEnabled  = errors.New("2FA is already enabled")
	Err2FANotSetup        = errors.New("2FA has not been set up yet")
)

type AuthService struct {
	userRepo      repository.UserRepo
	tenantRepo    repository.TenantRepo
	auditRepo     repository.AuditRepo
	tokenService  *TokenService
	passwordSvc   *PasswordService
	pool          *pgxpool.Pool
	encryptionKey []byte
}

func NewAuthService(
	userRepo repository.UserRepo,
	tenantRepo repository.TenantRepo,
	auditRepo repository.AuditRepo,
	tokenSvc *TokenService,
	passwordSvc *PasswordService,
	pool *pgxpool.Pool,
	encryptionKey ...[]byte,
) *AuthService {
	s := &AuthService{
		userRepo:     userRepo,
		tenantRepo:   tenantRepo,
		auditRepo:    auditRepo,
		tokenService: tokenSvc,
		passwordSvc:  passwordSvc,
		pool:         pool,
	}
	if len(encryptionKey) > 0 {
		s.encryptionKey = encryptionKey[0]
	}
	return s
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

// LoginResult holds the result of a login attempt. If Requires2FA is true,
// TempToken is set and the caller must complete the 2FA flow.
type LoginResult struct {
	TokenResponse *model.TokenResponse
	RefreshToken  string
	Requires2FA   bool
	TempToken     string
}

// Login authenticates a user and returns tokens (or a 2FA pending token).
func (s *AuthService) Login(ctx context.Context, req model.LoginRequest, ipAddress string) (*LoginResult, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	// Find tenant by slug (SECURITY DEFINER, bypasses RLS)
	tenant, err := s.tenantRepo.FindBySlug(ctx, req.TenantSlug)
	if err != nil {
		return nil, fmt.Errorf("find tenant: %w", err)
	}
	if tenant == nil {
		return nil, ErrInvalidCredentials
	}

	// Find user for auth (SECURITY DEFINER, bypasses RLS)
	userWithPwd, err := s.userRepo.FindForAuth(ctx, req.Email, tenant.ID)
	if err != nil {
		return nil, fmt.Errorf("find user: %w", err)
	}
	if userWithPwd == nil {
		return nil, ErrInvalidCredentials
	}

	if err := s.passwordSvc.Compare(userWithPwd.PasswordHash, req.Password); err != nil {
		return nil, ErrInvalidCredentials
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

	// If 2FA is enabled, return a pending token instead of full tokens
	if userWithPwd.TOTPEnabled {
		tempToken, err := s.tokenService.Generate2FAPendingToken(user)
		if err != nil {
			return nil, fmt.Errorf("generate 2fa pending token: %w", err)
		}
		return &LoginResult{
			Requires2FA: true,
			TempToken:   tempToken,
		}, nil
	}

	accessToken, err := s.tokenService.GenerateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	refreshToken, err := s.tokenService.GenerateRefreshToken(user)
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	return &LoginResult{
		TokenResponse: &model.TokenResponse{
			AccessToken: accessToken,
			ExpiresIn:   3600,
			User:        user,
			Tenant:      *tenant,
		},
		RefreshToken: refreshToken,
	}, nil
}

// Verify2FALogin validates the TOTP code from a 2FA pending token and returns full tokens.
func (s *AuthService) Verify2FALogin(ctx context.Context, tempTokenStr, code string) (*model.TokenResponse, string, error) {
	claims, err := s.tokenService.ValidateToken(tempTokenStr)
	if err != nil {
		return nil, "", ErrInvalid2FAToken
	}
	if claims.Type != "2fa_pending" {
		return nil, "", ErrInvalid2FAToken
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return nil, "", ErrInvalid2FAToken
	}

	var user *model.User
	var tenant *model.Tenant
	var encryptedSecret *string

	err = database.WithTenant(ctx, s.pool, claims.TenantID, func(tx pgx.Tx) error {
		var findErr error
		user, findErr = s.userRepo.FindByID(ctx, tx, userID)
		if findErr != nil {
			return findErr
		}
		tenant, findErr = s.tenantRepo.FindByID(ctx, tx, claims.TenantID)
		if findErr != nil {
			return findErr
		}
		encryptedSecret, findErr = s.userRepo.GetTOTPSecret(ctx, tx, userID)
		return findErr
	})
	if err != nil {
		return nil, "", fmt.Errorf("find user for 2fa: %w", err)
	}
	if user == nil || encryptedSecret == nil {
		return nil, "", ErrInvalid2FAToken
	}

	// Decrypt the TOTP secret
	secretBytes, err := crypto.Decrypt(*encryptedSecret, s.encryptionKey)
	if err != nil {
		return nil, "", fmt.Errorf("decrypt totp secret: %w", err)
	}

	// Validate the TOTP code
	if !totp.Validate(code, string(secretBytes)) {
		return nil, "", ErrInvalid2FACode
	}

	accessToken, err := s.tokenService.GenerateAccessToken(*user)
	if err != nil {
		return nil, "", fmt.Errorf("generate access token: %w", err)
	}

	refreshToken, err := s.tokenService.GenerateRefreshToken(*user)
	if err != nil {
		return nil, "", fmt.Errorf("generate refresh token: %w", err)
	}

	return &model.TokenResponse{
		AccessToken: accessToken,
		ExpiresIn:   3600,
		User:        *user,
		Tenant:      *tenant,
	}, refreshToken, nil
}

// Setup2FA generates a TOTP secret for the user and stores it encrypted.
func (s *AuthService) Setup2FA(ctx context.Context, userID, tenantID uuid.UUID, email string) (*model.TwoFASetupResponse, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "OpenOMS",
		AccountName: email,
	})
	if err != nil {
		return nil, fmt.Errorf("generate totp key: %w", err)
	}

	// Encrypt the secret before storing
	encryptedSecret, err := crypto.Encrypt([]byte(key.Secret()), s.encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("encrypt totp secret: %w", err)
	}

	err = database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		return s.userRepo.SetTOTPSecret(ctx, tx, userID, encryptedSecret)
	})
	if err != nil {
		return nil, fmt.Errorf("store totp secret: %w", err)
	}

	return &model.TwoFASetupResponse{
		Secret: key.Secret(),
		QRURL:  key.URL(),
	}, nil
}

// Verify2FA verifies a TOTP code and enables 2FA for the user.
func (s *AuthService) Verify2FA(ctx context.Context, userID, tenantID uuid.UUID, code string) error {
	var encryptedSecret *string

	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var findErr error
		encryptedSecret, findErr = s.userRepo.GetTOTPSecret(ctx, tx, userID)
		return findErr
	})
	if err != nil {
		return fmt.Errorf("get totp secret: %w", err)
	}
	if encryptedSecret == nil {
		return Err2FANotSetup
	}

	secretBytes, err := crypto.Decrypt(*encryptedSecret, s.encryptionKey)
	if err != nil {
		return fmt.Errorf("decrypt totp secret: %w", err)
	}

	if !totp.Validate(code, string(secretBytes)) {
		return ErrInvalid2FACode
	}

	return database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		return s.userRepo.EnableTOTP(ctx, tx, userID)
	})
}

// Disable2FA disables 2FA for the user after verifying password and code.
func (s *AuthService) Disable2FA(ctx context.Context, userID, tenantID uuid.UUID, password, code string) error {
	// Find user to verify password
	var user *model.User
	var encryptedSecret *string

	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var findErr error
		user, findErr = s.userRepo.FindByID(ctx, tx, userID)
		if findErr != nil {
			return findErr
		}
		encryptedSecret, findErr = s.userRepo.GetTOTPSecret(ctx, tx, userID)
		return findErr
	})
	if err != nil {
		return fmt.Errorf("find user: %w", err)
	}
	if user == nil {
		return ErrUserNotFound
	}

	// Verify password via FindForAuth (has password hash)
	userWithPwd, err := s.userRepo.FindForAuth(ctx, user.Email, tenantID)
	if err != nil || userWithPwd == nil {
		return ErrInvalidCredentials
	}
	if err := s.passwordSvc.Compare(userWithPwd.PasswordHash, password); err != nil {
		return ErrInvalidCredentials
	}

	// Verify TOTP code
	if encryptedSecret != nil {
		secretBytes, err := crypto.Decrypt(*encryptedSecret, s.encryptionKey)
		if err != nil {
			return fmt.Errorf("decrypt totp secret: %w", err)
		}
		if !totp.Validate(code, string(secretBytes)) {
			return ErrInvalid2FACode
		}
	}

	return database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		return s.userRepo.DisableTOTP(ctx, tx, userID)
	})
}

// Get2FAStatus returns the 2FA status for a user.
func (s *AuthService) Get2FAStatus(ctx context.Context, userID, tenantID uuid.UUID) (*model.TwoFAStatusResponse, error) {
	var enabled bool
	var verifiedAt *string

	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var findErr error
		enabled, verifiedAt, findErr = s.userRepo.GetTOTPStatus(ctx, tx, userID)
		return findErr
	})
	if err != nil {
		return nil, fmt.Errorf("get 2fa status: %w", err)
	}

	resp := &model.TwoFAStatusResponse{
		Enabled: enabled,
	}
	if verifiedAt != nil {
		resp.VerifiedAt = *verifiedAt
	}
	return resp, nil
}


type RefreshTokenInfo struct {
	UserID   uuid.UUID
	TenantID uuid.UUID
}

func (s *AuthService) ValidateRefreshToken(tokenStr string) (*RefreshTokenInfo, error) {
	claims, err := s.tokenService.ValidateToken(tokenStr)
	if err != nil {
		return nil, err
	}
	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return nil, err
	}
	return &RefreshTokenInfo{UserID: userID, TenantID: claims.TenantID}, nil
}

func (s *AuthService) Logout(ctx context.Context, userID, tenantID uuid.UUID) error {
	return database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		return s.userRepo.UpdateLastLogout(ctx, tx, userID)
	})
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

	if user.LastLogoutAt != nil && claims.IssuedAt != nil && claims.IssuedAt.Time.Before(*user.LastLogoutAt) {
		return nil, "", fmt.Errorf("refresh token revoked by logout")
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
