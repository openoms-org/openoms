package middleware

import (
	"net/http"

	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
)

// SentryContext enriches Sentry events with tenant and user context.
// Must be placed after JWTAuth middleware (needs claims in context).
func SentryContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hub := sentry.GetHubFromContext(r.Context())
		if hub == nil {
			next.ServeHTTP(w, r)
			return
		}

		// Use the same context keys as auth middleware
		if tenantID := TenantIDFromContext(r.Context()); tenantID != uuid.Nil {
			hub.Scope().SetTag("tenant_id", tenantID.String())
		}
		if userID := UserIDFromContext(r.Context()); userID != uuid.Nil {
			hub.Scope().SetUser(sentry.User{ID: userID.String()})
		}

		next.ServeHTTP(w, r)
	})
}
