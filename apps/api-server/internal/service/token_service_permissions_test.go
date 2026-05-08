package service

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

func TestTokenService_GenerateAccessTokenIncludesRoleIDAndPermissions(t *testing.T) {
	svc, err := NewTokenService("0123456789abcdef0123456789abcdef")
	require.NoError(t, err)

	roleID := uuid.New()
	user := model.User{
		ID:          uuid.New(),
		TenantID:    uuid.New(),
		Email:       "user@example.com",
		Role:        "member",
		RoleID:      &roleID,
		Permissions: []string{model.PermOrdersView},
	}

	token, err := svc.GenerateAccessToken(user)
	require.NoError(t, err)

	claims, err := svc.ValidateToken(token)
	require.NoError(t, err)
	assert.Equal(t, roleID, claims.RoleID)
	assert.Equal(t, []string{model.PermOrdersView}, claims.Permissions)
}

func TestTokenService_GenerateAccessTokenPreservesEmptyCustomPermissions(t *testing.T) {
	svc, err := NewTokenService("0123456789abcdef0123456789abcdef")
	require.NoError(t, err)

	roleID := uuid.New()
	user := model.User{
		ID:          uuid.New(),
		TenantID:    uuid.New(),
		Email:       "user@example.com",
		Role:        "member",
		RoleID:      &roleID,
		Permissions: []string{},
	}

	token, err := svc.GenerateAccessToken(user)
	require.NoError(t, err)

	claims, err := svc.ValidateToken(token)
	require.NoError(t, err)
	assert.NotNil(t, claims.Permissions)
	assert.Empty(t, claims.Permissions)
}
