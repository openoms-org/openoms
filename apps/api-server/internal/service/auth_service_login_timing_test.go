package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

const minInvalidLoginBcryptDuration = 10 * time.Millisecond

type loginTimingTenantRepo struct {
	tenant *model.Tenant
}

func (r loginTimingTenantRepo) FindBySlug(_ context.Context, _ string) (*model.Tenant, error) {
	return r.tenant, nil
}

func (r loginTimingTenantRepo) FindByID(_ context.Context, _ pgx.Tx, _ uuid.UUID) (*model.Tenant, error) {
	return r.tenant, nil
}

func (r loginTimingTenantRepo) SlugExists(context.Context, string) (bool, error)    { return false, nil }
func (r loginTimingTenantRepo) Create(context.Context, pgx.Tx, *model.Tenant) error { return nil }
func (r loginTimingTenantRepo) GetSettings(context.Context, pgx.Tx, uuid.UUID) (json.RawMessage, error) {
	return nil, nil
}
func (r loginTimingTenantRepo) GetSettingsForUpdate(context.Context, pgx.Tx, uuid.UUID) (json.RawMessage, error) {
	return nil, nil
}
func (r loginTimingTenantRepo) ListAllTenantIDs(context.Context, *pgxpool.Pool) ([]uuid.UUID, error) {
	return nil, nil
}
func (r loginTimingTenantRepo) UpdateSettings(context.Context, pgx.Tx, uuid.UUID, json.RawMessage) error {
	return nil
}

type loginTimingUserRepo struct{}

func (r loginTimingUserRepo) FindForAuth(context.Context, string, uuid.UUID) (*repository.UserWithPassword, error) {
	return nil, nil
}

func (r loginTimingUserRepo) FindByID(context.Context, pgx.Tx, uuid.UUID) (*model.User, error) {
	return nil, nil
}
func (r loginTimingUserRepo) List(context.Context, pgx.Tx) ([]model.User, error)        { return nil, nil }
func (r loginTimingUserRepo) Count(context.Context, pgx.Tx) (int, error)                { return 0, nil }
func (r loginTimingUserRepo) Create(context.Context, pgx.Tx, *model.User, string) error { return nil }
func (r loginTimingUserRepo) FindPasswordHashByID(context.Context, pgx.Tx, uuid.UUID) (*string, error) {
	return nil, nil
}
func (r loginTimingUserRepo) UpdatePassword(context.Context, pgx.Tx, uuid.UUID, string) error {
	return nil
}
func (r loginTimingUserRepo) UpdateRole(context.Context, pgx.Tx, uuid.UUID, string) error { return nil }
func (r loginTimingUserRepo) UpdateRoleID(context.Context, pgx.Tx, uuid.UUID, *uuid.UUID) error {
	return nil
}
func (r loginTimingUserRepo) UpdateName(context.Context, pgx.Tx, uuid.UUID, string) error { return nil }
func (r loginTimingUserRepo) UpdateLanguage(context.Context, pgx.Tx, uuid.UUID, *string) error {
	return nil
}
func (r loginTimingUserRepo) UpdateLastLogin(context.Context, pgx.Tx, uuid.UUID) error  { return nil }
func (r loginTimingUserRepo) UpdateLastLogout(context.Context, pgx.Tx, uuid.UUID) error { return nil }
func (r loginTimingUserRepo) Delete(context.Context, pgx.Tx, uuid.UUID) error           { return nil }
func (r loginTimingUserRepo) CountByRole(context.Context, pgx.Tx, string) (int, error)  { return 0, nil }
func (r loginTimingUserRepo) SetTOTPSecret(context.Context, pgx.Tx, uuid.UUID, string) error {
	return nil
}
func (r loginTimingUserRepo) EnableTOTP(context.Context, pgx.Tx, uuid.UUID) error  { return nil }
func (r loginTimingUserRepo) DisableTOTP(context.Context, pgx.Tx, uuid.UUID) error { return nil }
func (r loginTimingUserRepo) GetTOTPStatus(context.Context, pgx.Tx, uuid.UUID) (bool, *string, error) {
	return false, nil, nil
}
func (r loginTimingUserRepo) GetTOTPSecret(context.Context, pgx.Tx, uuid.UUID) (*string, error) {
	return nil, nil
}
func (r loginTimingUserRepo) GetTOTPLastUsedStep(context.Context, pgx.Tx, uuid.UUID) (*int64, error) {
	return nil, nil
}
func (r loginTimingUserRepo) SetTOTPLastUsedStep(context.Context, pgx.Tx, uuid.UUID, int64) (bool, error) {
	return true, nil
}

func TestAuthService_Login_MissingTenantBurnsBcryptCompare(t *testing.T) {
	svc := NewAuthService(nil, loginTimingTenantRepo{}, nil, nil, NewPasswordService(), nil)

	elapsed := measureInvalidLogin(t, svc, "missing-tenant")

	require.GreaterOrEqual(t, elapsed, minInvalidLoginBcryptDuration)
}

func TestAuthService_Login_MissingUserBurnsBcryptCompare(t *testing.T) {
	tenantID := uuid.New()
	svc := NewAuthService(loginTimingUserRepo{}, loginTimingTenantRepo{tenant: &model.Tenant{
		ID:   tenantID,
		Slug: "tenant",
		Name: "Tenant",
	}}, nil, nil, NewPasswordService(), nil)

	elapsed := measureInvalidLogin(t, svc, "tenant")

	require.GreaterOrEqual(t, elapsed, minInvalidLoginBcryptDuration)
}

func measureInvalidLogin(t *testing.T, svc *AuthService, tenantSlug string) time.Duration {
	t.Helper()
	started := time.Now()
	_, err := svc.Login(context.Background(), model.LoginRequest{
		TenantSlug: tenantSlug,
		Email:      "missing@example.com",
		Password:   "wrong-password",
	}, "127.0.0.1")
	elapsed := time.Since(started)
	require.ErrorIs(t, err, ErrInvalidCredentials)
	return elapsed
}
