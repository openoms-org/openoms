package middleware

import (
	"net/http"
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
