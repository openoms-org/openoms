package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

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
	assert.Contains(t, err.Error(), "tenant_slug")
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

func TestAuthService_Refresh_InvalidToken(t *testing.T) {
	tokenSvc, err := NewTokenService("test-secret-key-for-unit-tests-32chars!")
	require.NoError(t, err)
	svc := NewAuthService(nil, nil, nil, tokenSvc, nil, nil)

	_, _, err = svc.Refresh(context.Background(), "invalid-token")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid refresh token")
}
