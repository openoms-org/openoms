package middleware

import (
	"net/http"

	"github.com/getsentry/sentry-go"
)

// SentryMiddleware captures panics and reports them to Sentry with request context.
// Should be placed early in the middleware chain (after RealIP, before Recoverer).
//
// Intentionally does NOT call sentry.Flush on recovery — during a panic storm,
// a per-panic 2s flush serializes goroutines and causes a latency cascade.
// Events are delivered asynchronously by the Sentry SDK transport; a final
// sentry.Flush runs on process shutdown in cmd/server/main.go.
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
				// Re-panic so chi's Recoverer handles the HTTP 500 response.
				panic(err)
			}
		}()

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
