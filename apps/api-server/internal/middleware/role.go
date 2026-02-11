package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

var roleLevel = map[string]int{
	"member": 1,
	"admin":  2,
	"owner":  3,
}

// RequireRole creates middleware that checks if the authenticated user has
// at least the specified role level. Role hierarchy: member < admin < owner.
func RequireRole(minRole string) func(http.Handler) http.Handler {
	minLevel := roleLevel[minRole]

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := ClaimsFromContext(r.Context())
			if claims == nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":"authentication required"}`))
				return
			}

			userLevel := roleLevel[claims.Role]
			if userLevel < minLevel {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte(`{"error":"insufficient permissions"}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RoleFinder is an interface for looking up a role by tenant and role ID.
// This avoids importing the service package directly in middleware.
type RoleFinder interface {
	FindByID(ctx context.Context, tenantID, roleID uuid.UUID) (*model.Role, error)
}

// RequirePermission creates middleware that checks if the authenticated user's
// RBAC role has the given permission. If the user has no role_id assigned
// (legacy users), it falls back to the legacy role hierarchy check:
// owner/admin pass all permission checks, member is denied admin-only perms.
func RequirePermission(permission string, roleFinder RoleFinder) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := ClaimsFromContext(r.Context())
			if claims == nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":"authentication required"}`))
				return
			}

			// If user has a role_id, use fine-grained RBAC
			if claims.RoleID != uuid.Nil {
				tenantID := TenantIDFromContext(r.Context())
				role, err := roleFinder.FindByID(r.Context(), tenantID, claims.RoleID)
				if err != nil || role == nil {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusForbidden)
					w.Write([]byte(`{"error":"insufficient permissions"}`))
					return
				}

				if !role.HasPermission(permission) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusForbidden)
					w.Write([]byte(`{"error":"insufficient permissions"}`))
					return
				}

				next.ServeHTTP(w, r)
				return
			}

			// Fallback: legacy role check — owner/admin pass, member is denied
			userLevel := roleLevel[claims.Role]
			if userLevel >= roleLevel["admin"] {
				next.ServeHTTP(w, r)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`{"error":"insufficient permissions"}`))
		})
	}
}
