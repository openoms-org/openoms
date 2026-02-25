package middleware_test

import (
	"context"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestUserIDFromContext_ValidUUID(t *testing.T) {
	uid := uuid.New()
	ctx := context.WithValue(context.Background(), middleware.UserIDKey, uid)

	result := middleware.UserIDFromContext(ctx)
	assert.Equal(t, uid, result)
}

func TestUserIDFromContext_MissingKey_ReturnsNil(t *testing.T) {
	ctx := context.Background()

	result := middleware.UserIDFromContext(ctx)
	assert.Equal(t, uuid.Nil, result)
}

func TestUserIDFromContext_WrongType_ReturnsNil(t *testing.T) {
	ctx := context.WithValue(context.Background(), middleware.UserIDKey, "not-a-uuid")

	result := middleware.UserIDFromContext(ctx)
	assert.Equal(t, uuid.Nil, result)
}

func TestClaimsFromContext_ValidClaims(t *testing.T) {
	claims := &model.AuthClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject: uuid.New().String(),
		},
		TenantID: uuid.New(),
		Email:    "test@openoms.org",
		Role:     "admin",
	}
	ctx := context.WithValue(context.Background(), middleware.ClaimsKey, claims)

	result := middleware.ClaimsFromContext(ctx)
	assert.NotNil(t, result)
	assert.Equal(t, claims.Email, result.Email)
	assert.Equal(t, claims.TenantID, result.TenantID)
	assert.Equal(t, claims.Role, result.Role)
}

func TestClaimsFromContext_MissingKey_ReturnsNil(t *testing.T) {
	ctx := context.Background()

	result := middleware.ClaimsFromContext(ctx)
	assert.Nil(t, result)
}

func TestClaimsFromContext_WrongType_ReturnsNil(t *testing.T) {
	ctx := context.WithValue(context.Background(), middleware.ClaimsKey, "not-claims")

	result := middleware.ClaimsFromContext(ctx)
	assert.Nil(t, result)
}
