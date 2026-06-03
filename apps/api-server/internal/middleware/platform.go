package middleware

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// PlatformAdminKey is the context key for the resolved platform admin.
const PlatformAdminKey contextKey = "platform_admin"

// PlatformAdminLookup resolves a platform admin by user ID. Implemented by
// repository.PlatformAdminRepository. Returns a non-nil error (e.g.
// pgx.ErrNoRows) when the user is not a platform admin.
type PlatformAdminLookup interface {
	GetByUserID(ctx context.Context, userID uuid.UUID) (*model.PlatformAdmin, error)
}

// PlatformAdminFromContext returns the resolved platform admin, or nil.
func PlatformAdminFromContext(ctx context.Context) *model.PlatformAdmin {
	if a, ok := ctx.Value(PlatformAdminKey).(*model.PlatformAdmin); ok {
		return a
	}
	return nil
}

// RequirePlatformAdmin authenticates the request as an OpenOMS platform admin.
// It MUST run after JWTAuth (which sets UserIDKey). It deliberately does NOT
// consult tenant roles or permissions — platform authorization is a separate
// governance layer. The lookup is fail-closed: any error (including a not-found
// user or a transient lookup failure) results in 403. Unauthenticated requests
// (no user in context) get 401.
func RequirePlatformAdmin(lookup PlatformAdminLookup) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := UserIDFromContext(r.Context())
			if userID == uuid.Nil {
				writePlatformError(w, http.StatusUnauthorized, "authentication required")
				return
			}

			admin, err := lookup.GetByUserID(r.Context(), userID)
			if err != nil || admin == nil {
				writePlatformError(w, http.StatusForbidden, "platform administration access required")
				return
			}

			ctx := context.WithValue(r.Context(), PlatformAdminKey, admin)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequirePlatformPermission requires a specific platform permission. It MUST
// run after RequirePlatformAdmin (which puts the admin in context).
func RequirePlatformPermission(required string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			admin := PlatformAdminFromContext(r.Context())
			if admin == nil {
				writePlatformError(w, http.StatusForbidden, "platform administration access required")
				return
			}
			if !admin.HasPermission(required) {
				writePlatformError(w, http.StatusForbidden, "insufficient platform permissions")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func writePlatformError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
