package middleware

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/getsentry/sentry-go"
)

const sentryFilteredValue = "[Filtered]"

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
		hub.Scope().SetRequest(SanitizeSentryRequest(r))

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

// SanitizeSentryRequest returns a request copy safe to attach to Sentry scope.
// The original request is left untouched for downstream handlers.
func SanitizeSentryRequest(r *http.Request) *http.Request {
	if r == nil {
		return nil
	}

	sanitized := r.Clone(r.Context())
	sanitized.Header = r.Header.Clone()
	scrubSentryHeaders(sanitized.Header)

	if sanitized.URL != nil {
		u := *sanitized.URL
		u.RawQuery = scrubSentryRawQuery(u.RawQuery)
		sanitized.URL = &u
	}

	sanitized.RemoteAddr = ""
	sanitized.Body = http.NoBody
	sanitized.ContentLength = 0
	sanitized.GetBody = nil

	return sanitized
}

// ScrubSentryEvent is a BeforeSend hook that removes sensitive request data.
func ScrubSentryEvent(event *sentry.Event, _ *sentry.EventHint) *sentry.Event {
	if event == nil || event.Request == nil {
		return event
	}

	event.Request.QueryString = scrubSentryRawQuery(event.Request.QueryString)
	event.Request.Cookies = ""
	event.Request.Data = ""
	scrubSentryEventHeaders(event.Request.Headers)
	if event.Request.Env != nil {
		delete(event.Request.Env, "REMOTE_ADDR")
		delete(event.Request.Env, "REMOTE_PORT")
	}

	return event
}

func scrubSentryHeaders(headers http.Header) {
	for key := range headers {
		if isSentrySensitiveHeader(key) {
			delete(headers, key)
		}
	}
}

func scrubSentryEventHeaders(headers map[string]string) {
	for key := range headers {
		if isSentrySensitiveHeader(key) {
			delete(headers, key)
		}
	}
}

func isSentrySensitiveHeader(key string) bool {
	switch strings.ToLower(key) {
	case "authorization", "cookie", "set-cookie", "x-api-key", "x-csrf-token", "x-forwarded-for", "x-real-ip", "x-xsrf-token":
		return true
	default:
		return false
	}
}

func scrubSentryRawQuery(rawQuery string) string {
	if rawQuery == "" {
		return ""
	}

	values, err := url.ParseQuery(rawQuery)
	if err != nil {
		return ""
	}
	for key := range values {
		if isSentrySensitiveQueryKey(key) {
			values[key] = []string{sentryFilteredValue}
		}
	}
	return values.Encode()
}

func isSentrySensitiveQueryKey(key string) bool {
	normalized := strings.ToLower(key)
	switch normalized {
	case "api_key", "authorization", "code", "email", "id_token", "key", "password", "refresh_token", "secret", "token":
		return true
	default:
		return strings.Contains(normalized, "token") ||
			strings.Contains(normalized, "secret") ||
			strings.Contains(normalized, "password")
	}
}
