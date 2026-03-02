package middleware

import (
	"net/http"
	"time"

	"github.com/getsentry/sentry-go"
)

// SentryMiddleware captures panics and reports them to Sentry with request context.
// Should be placed early in the middleware chain (after RealIP, before Recoverer).
func SentryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hub := sentry.GetHubFromContext(r.Context())
		if hub == nil {
			hub = sentry.CurrentHub().Clone()
		}
		hub.Scope().SetRequest(r)

		ctx := sentry.SetHubOnContext(r.Context(), hub)

		defer func() {
			if err := recover(); err != nil {
				hub.RecoverWithContext(ctx, err)
				sentry.Flush(2 * time.Second)
				// Re-panic so chi's Recoverer handles the HTTP 500 response.
				panic(err)
			}
		}()

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
