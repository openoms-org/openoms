package middleware

import (
	"encoding/json"
	"net/http"
	"slices"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// RequirePermission creates middleware that checks for one of the required granular permissions.
// Access tokens issued before permissions existed fall back to legacy system-role defaults.
func RequirePermission(required string, alternatives ...string) func(http.Handler) http.Handler {
	requiredPermissions := append([]string{required}, alternatives...)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := ClaimsFromContext(r.Context())
			if claims == nil {
				writePermissionError(w, http.StatusUnauthorized, "authentication required")
				return
			}

			for _, permission := range requiredPermissions {
				if claimsHasPermission(claims, permission) {
					next.ServeHTTP(w, r)
					return
				}
			}

			writePermissionError(w, http.StatusForbidden, "insufficient permissions")
		})
	}
}

func claimsHasPermission(claims *model.AuthClaims, permission string) bool {
	permissions := claims.Permissions
	if permissions == nil {
		permissions = model.SystemPermissionsForRole(claims.Role)
	}
	return slices.Contains(permissions, permission)
}

func writePermissionError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
