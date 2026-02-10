package middleware

import (
	"context"

	"github.com/google/uuid"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// Context keys for authenticated user data.
// TenantIDKey is already defined in tenant.go.
const (
	UserIDKey contextKey = "user_id"
	ClaimsKey contextKey = "auth_claims"
)

// UserIDFromContext returns the authenticated user's ID from context.
func UserIDFromContext(ctx context.Context) uuid.UUID {
	if id, ok := ctx.Value(UserIDKey).(uuid.UUID); ok {
		return id
	}
	return uuid.Nil
}

// ClaimsFromContext returns the full JWT claims from context.
func ClaimsFromContext(ctx context.Context) *model.AuthClaims {
	if claims, ok := ctx.Value(ClaimsKey).(*model.AuthClaims); ok {
		return claims
	}
	return nil
}
