package middleware

import (
	"context"

	"github.com/google/uuid"
)

type contextKey string

// TenantIDKey is the context key for the authenticated tenant ID.
const TenantIDKey contextKey = "tenant_id"

// TenantIDFromContext extracts the tenant UUID from the context.
func TenantIDFromContext(ctx context.Context) uuid.UUID {
	if id, ok := ctx.Value(TenantIDKey).(uuid.UUID); ok {
		return id
	}
	return uuid.Nil
}
