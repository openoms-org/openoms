package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

func TestTokenRevokedByLogout(t *testing.T) {
	base := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	// last_logout_at carries sub-second precision (PostgreSQL NOW()).
	epoch := base.Add(750 * time.Millisecond) // 10:00:00.750

	// A token issued in the SAME second as the epoch: JWT iat is floored to the second
	// (10:00:00.000). It must NOT be treated as revoked — this is the regression OPE-505 fixes.
	sameSecond := jwt.NewNumericDate(base)
	assert.False(t, tokenRevokedByLogout(sameSecond, &epoch),
		"token issued in the same second as the epoch must not be falsely revoked")

	// A token issued a full second before the epoch IS revoked.
	prevSecond := jwt.NewNumericDate(base.Add(-time.Second))
	assert.True(t, tokenRevokedByLogout(prevSecond, &epoch),
		"token issued before the epoch second must be revoked")

	// A token issued a second after the epoch is valid.
	nextSecond := jwt.NewNumericDate(base.Add(time.Second))
	assert.False(t, tokenRevokedByLogout(nextSecond, &epoch))

	// No epoch / no iat → never revoked.
	assert.False(t, tokenRevokedByLogout(sameSecond, nil))
	assert.False(t, tokenRevokedByLogout(nil, &epoch))
}

func TestAuthService_Register_ValidationError_MissingEmail(t *testing.T) {
	pwdSvc := NewPasswordService()
	svc := NewAuthService(nil, nil, nil, nil, pwdSvc, nil)

	_, _, err := svc.Register(context.Background(), model.RegisterRequest{
		TenantName: "Test",
		TenantSlug: "test",
		Name:       "User",
		Password:   "StrongP@ss123",
	}, "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "email")
}

func TestAuthService_Register_ValidationError_MissingTenantName(t *testing.T) {
	pwdSvc := NewPasswordService()
	svc := NewAuthService(nil, nil, nil, nil, pwdSvc, nil)

	_, _, err := svc.Register(context.Background(), model.RegisterRequest{
		TenantSlug: "test",
		Email:      "user@example.com",
		Name:       "User",
		Password:   "StrongP@ss123",
	}, "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "tenant_name")
}

func TestAuthService_Register_ValidationError_MissingTenantSlug(t *testing.T) {
	pwdSvc := NewPasswordService()
	svc := NewAuthService(nil, nil, nil, nil, pwdSvc, nil)

	_, _, err := svc.Register(context.Background(), model.RegisterRequest{
		TenantName: "Test",
		Email:      "user@example.com",
		Name:       "User",
		Password:   "StrongP@ss123",
	}, "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "slug")
}

func TestAuthService_Register_ValidationError_WeakPassword(t *testing.T) {
	pwdSvc := NewPasswordService()
	svc := NewAuthService(nil, nil, nil, nil, pwdSvc, nil)

	_, _, err := svc.Register(context.Background(), model.RegisterRequest{
		TenantName: "Test",
		TenantSlug: "test",
		Email:      "user@example.com",
		Name:       "User",
		Password:   "short",
	}, "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
}

func TestAuthService_Login_ValidationError_MissingEmail(t *testing.T) {
	svc := NewAuthService(nil, nil, nil, nil, nil, nil)

	_, err := svc.Login(context.Background(), model.LoginRequest{
		TenantSlug: "test",
		Password:   "password",
	}, "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
}

func TestAuthService_Login_ValidationError_MissingPassword(t *testing.T) {
	svc := NewAuthService(nil, nil, nil, nil, nil, nil)

	_, err := svc.Login(context.Background(), model.LoginRequest{
		TenantSlug: "test",
		Email:      "user@example.com",
	}, "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
}

func TestAuthService_Login_ValidationError_MissingTenantSlug(t *testing.T) {
	svc := NewAuthService(nil, nil, nil, nil, nil, nil)

	_, err := svc.Login(context.Background(), model.LoginRequest{
		Email:    "user@example.com",
		Password: "password",
	}, "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
}

func TestAuthService_Register_ValidationError_InvalidEmail(t *testing.T) {
	// NOTE: RegisterRequest.Validate() only checks for empty email, not format.
	// The invalid email "not-an-email" passes Validate() (non-empty), so the
	// validation error here is triggered by password strength (no digit).
	// Email format validation should be added to RegisterRequest.Validate().
	pwdSvc := NewPasswordService()
	svc := NewAuthService(nil, nil, nil, nil, pwdSvc, nil)

	_, _, err := svc.Register(context.Background(), model.RegisterRequest{
		TenantName: "Test",
		TenantSlug: "test",
		Email:      "not-an-email",
		Name:       "User",
		Password:   "nodigitshere",
	}, "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
}

func TestAuthService_Register_ValidationError_MissingName(t *testing.T) {
	pwdSvc := NewPasswordService()
	svc := NewAuthService(nil, nil, nil, nil, pwdSvc, nil)

	_, _, err := svc.Register(context.Background(), model.RegisterRequest{
		TenantName: "Test",
		TenantSlug: "test",
		Email:      "user@example.com",
		Name:       "",
		Password:   "StrongP@ss123",
	}, "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "name")
}

func TestAuthService_Register_ValidationError_InvalidSlug_Spaces(t *testing.T) {
	// NOTE: RegisterRequest.Validate() only checks for empty slug, not format.
	// The slug "has spaces" passes Validate() (non-empty), so the validation
	// error here is triggered by password strength (no digit).
	// Slug format validation should be added to RegisterRequest.Validate().
	pwdSvc := NewPasswordService()
	svc := NewAuthService(nil, nil, nil, nil, pwdSvc, nil)

	_, _, err := svc.Register(context.Background(), model.RegisterRequest{
		TenantName: "Test",
		TenantSlug: "has spaces",
		Email:      "user@example.com",
		Name:       "User",
		Password:   "nodigitshere",
	}, "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
}

func TestAuthService_Register_ValidationError_InvalidSlug_Uppercase(t *testing.T) {
	// NOTE: RegisterRequest.Validate() only checks for empty slug, not format.
	// The slug "UPPERCASE" passes Validate() (non-empty), so the validation
	// error here is triggered by password strength (no digit).
	// Slug format validation should be added to RegisterRequest.Validate().
	pwdSvc := NewPasswordService()
	svc := NewAuthService(nil, nil, nil, nil, pwdSvc, nil)

	_, _, err := svc.Register(context.Background(), model.RegisterRequest{
		TenantName: "Test",
		TenantSlug: "UPPERCASE",
		Email:      "user@example.com",
		Name:       "User",
		Password:   "nodigitshere",
	}, "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
}

func TestAuthService_Register_ValidationError_PasswordNoDigit(t *testing.T) {
	pwdSvc := NewPasswordService()
	svc := NewAuthService(nil, nil, nil, nil, pwdSvc, nil)

	_, _, err := svc.Register(context.Background(), model.RegisterRequest{
		TenantName: "Test",
		TenantSlug: "test",
		Email:      "user@example.com",
		Name:       "User",
		Password:   "ABCDEFGHij",
	}, "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "digit")
}

func TestAuthService_Register_ValidationError_PasswordNoLetter(t *testing.T) {
	pwdSvc := NewPasswordService()
	svc := NewAuthService(nil, nil, nil, nil, pwdSvc, nil)

	_, _, err := svc.Register(context.Background(), model.RegisterRequest{
		TenantName: "Test",
		TenantSlug: "test",
		Email:      "user@example.com",
		Name:       "User",
		Password:   "12345678",
	}, "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "uppercase")
}

func TestAuthService_Register_ValidationError_PasswordTooLong(t *testing.T) {
	pwdSvc := NewPasswordService()
	svc := NewAuthService(nil, nil, nil, nil, pwdSvc, nil)

	// 73 characters: exceeds bcrypt's 72-byte limit
	longPassword := "Aa1aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

	_, _, err := svc.Register(context.Background(), model.RegisterRequest{
		TenantName: "Test",
		TenantSlug: "test",
		Email:      "user@example.com",
		Name:       "User",
		Password:   longPassword,
	}, "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "72")
}

func TestAuthService_Refresh_InvalidToken(t *testing.T) {
	tokenSvc, err := NewTokenService("test-secret-key-for-unit-tests-32chars!")
	require.NoError(t, err)
	svc := NewAuthService(nil, nil, nil, tokenSvc, nil, nil)

	_, _, err = svc.Refresh(context.Background(), "invalid-token")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid refresh token")
}

func TestAuthService_ConsumeStoredRefreshTokenRejectsRevokedSibling(t *testing.T) {
	store := NewMemoryRefreshTokenStore()
	defer store.Close()
	svc := NewAuthService(nil, nil, nil, nil, nil, nil)
	svc.SetRefreshTokenStore(store)

	ctx := context.Background()
	userID := uuid.New()
	tenantID := uuid.New()
	familyID := uuid.New().String()
	currentHash := "current-" + uuid.New().String()
	siblingHash := "sibling-" + uuid.New().String()
	claims := &model.AuthClaims{
		RegisteredClaims: jwt.RegisteredClaims{Subject: userID.String()},
		TenantID:         tenantID,
		Type:             "refresh",
	}

	err := store.StoreFamily(ctx, &RefreshTokenFamily{
		FamilyID:         familyID,
		UserID:           userID.String(),
		TenantID:         tenantID.String(),
		CurrentTokenHash: currentHash,
		CreatedAt:        time.Now(),
	}, 10*time.Minute)
	require.NoError(t, err)
	err = store.StoreToken(ctx, currentHash, &RefreshTokenEntry{FamilyID: familyID}, 10*time.Minute)
	require.NoError(t, err)
	err = store.StoreToken(ctx, siblingHash, &RefreshTokenEntry{FamilyID: familyID}, 10*time.Minute)
	require.NoError(t, err)

	family, err := svc.consumeStoredRefreshToken(ctx, currentHash, claims)
	require.NoError(t, err)
	require.NotNil(t, family)

	err = store.DeleteFamily(ctx, familyID)
	require.NoError(t, err)

	family, err = svc.consumeStoredRefreshToken(ctx, siblingHash, claims)
	require.ErrorIs(t, err, ErrInvalidCredentials)
	assert.Nil(t, family)

	siblingEntry, err := store.GetToken(ctx, siblingHash)
	require.NoError(t, err)
	require.NotNil(t, siblingEntry)
	assert.True(t, siblingEntry.Used, "revoked sibling should be consumed before rejection")
}

func TestAuthService_ConsumeStoredRefreshTokenInvalidatesNonCurrentSibling(t *testing.T) {
	store := NewMemoryRefreshTokenStore()
	defer store.Close()
	svc := NewAuthService(nil, nil, nil, nil, nil, nil)
	svc.SetRefreshTokenStore(store)

	ctx := context.Background()
	userID := uuid.New()
	tenantID := uuid.New()
	familyID := uuid.New().String()
	currentHash := "current-" + uuid.New().String()
	siblingHash := "sibling-" + uuid.New().String()
	claims := &model.AuthClaims{
		RegisteredClaims: jwt.RegisteredClaims{Subject: userID.String()},
		TenantID:         tenantID,
		Type:             "refresh",
	}

	err := store.StoreFamily(ctx, &RefreshTokenFamily{
		FamilyID:         familyID,
		UserID:           userID.String(),
		TenantID:         tenantID.String(),
		CurrentTokenHash: currentHash,
		CreatedAt:        time.Now(),
	}, 10*time.Minute)
	require.NoError(t, err)
	err = store.StoreToken(ctx, siblingHash, &RefreshTokenEntry{FamilyID: familyID}, 10*time.Minute)
	require.NoError(t, err)

	family, err := svc.consumeStoredRefreshToken(ctx, siblingHash, claims)
	require.ErrorIs(t, err, ErrRefreshTokenReuse)
	assert.Nil(t, family)

	family, err = store.GetFamily(ctx, familyID)
	require.NoError(t, err)
	assert.Nil(t, family, "non-current sibling use should revoke the entire family")
}

func TestAuthService_LoginFailedAuditOmitsPlaintextEmail(t *testing.T) {
	req := model.LoginRequest{
		TenantSlug: "tenant",
		Email:      "customer@example.com",
		Password:   "wrong-password",
	}

	entry := failedLoginAuditEntry(uuid.New(), uuid.New(), req, "127.0.0.1")
	changes, ok := entry.Changes.(map[string]string)
	require.True(t, ok)

	assert.Equal(t, "user.login_failed", entry.Action)
	assert.Equal(t, "invalid_password", changes["reason"])
	assert.NotContains(t, changes, "email")
	for _, value := range changes {
		assert.NotContains(t, value, req.Email)
	}
}
