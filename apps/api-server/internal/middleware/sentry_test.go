package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/getsentry/sentry-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
)

func TestSentryMiddleware_NormalRequest(t *testing.T) {
	err := sentry.Init(sentry.ClientOptions{
		Dsn: "",
	})
	assert.NoError(t, err)

	handler := middleware.SentryMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestSentryMiddleware_PanicRecovery(t *testing.T) {
	err := sentry.Init(sentry.ClientOptions{
		Dsn: "",
	})
	assert.NoError(t, err)

	handler := middleware.SentryMiddleware(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		panic("test panic")
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	// SentryMiddleware re-panics after capturing
	assert.Panics(t, func() {
		handler.ServeHTTP(rec, req)
	})
}

func TestSentryMiddleware_ScrubsSensitiveRequestData(t *testing.T) {
	client, err := sentry.NewClient(sentry.ClientOptions{
		Dsn:            "",
		SendDefaultPII: true,
	})
	require.NoError(t, err)
	hub := sentry.NewHub(client, sentry.NewScope())

	var capturedRequest *sentry.Request
	handler := middleware.SentryMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestHub := sentry.GetHubFromContext(r.Context())
		require.NotNil(t, requestHub)

		event := requestHub.Scope().ApplyToEvent(sentry.NewEvent(), nil, requestHub.Client())
		require.NotNil(t, event)
		require.NotNil(t, event.Request)
		capturedRequest = event.Request

		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(
		http.MethodGet,
		"https://app.example.test/v1/orders?customer_email=customer@example.com&userEmail=buyer@example.com&email=owner@example.com&token=secret-token&q=orders",
		nil,
	)
	req.Header.Set("Authorization", "Bearer access-token")
	req.Header.Set("Cookie", "refresh_token=secret-refresh-token")
	req.Header.Set("X-CSRF-Token", "csrf-token")
	req.Header.Set("X-API-Key", "api-key")
	req = req.WithContext(sentry.SetHubOnContext(req.Context(), hub))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	require.NotNil(t, capturedRequest)
	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Equal(t, "GET", capturedRequest.Method)
	assert.Equal(t, "https://app.example.test/v1/orders", capturedRequest.URL)
	assert.Contains(t, capturedRequest.QueryString, "q=orders")
	assert.Equal(t, "[Filtered]", sentryQueryValue(t, capturedRequest.QueryString, "customer_email"))
	assert.Equal(t, "[Filtered]", sentryQueryValue(t, capturedRequest.QueryString, "userEmail"))
	assert.Equal(t, "[Filtered]", sentryQueryValue(t, capturedRequest.QueryString, "email"))
	assert.NotContains(t, capturedRequest.QueryString, "customer@example.com")
	assert.NotContains(t, capturedRequest.QueryString, "buyer@example.com")
	assert.NotContains(t, capturedRequest.QueryString, "owner@example.com")
	assert.NotContains(t, capturedRequest.QueryString, "secret-token")
	assert.Empty(t, capturedRequest.Cookies)
	assertNoSentryHeader(t, capturedRequest.Headers, "Authorization")
	assertNoSentryHeader(t, capturedRequest.Headers, "Cookie")
	assertNoSentryHeader(t, capturedRequest.Headers, "X-CSRF-Token")
	assertNoSentryHeader(t, capturedRequest.Headers, "X-API-Key")
}

func TestScrubSentryEvent_RemovesSensitiveRequestData(t *testing.T) {
	event := &sentry.Event{
		Request: &sentry.Request{
			QueryString: "customer_email=customer@example.com&userEmail=buyer@example.com&email=owner@example.com&token=secret-token&q=orders",
			Cookies:     "refresh_token=secret-refresh-token",
			Data:        `{"password":"secret"}`,
			Headers: map[string]string{
				"Authorization": "Bearer access-token",
				"Safe-Header":   "safe-value",
				"X-Csrf-Token":  "csrf-token",
			},
			Env: map[string]string{
				"REMOTE_ADDR": "203.0.113.10",
				"REMOTE_PORT": "41234",
			},
		},
	}

	scrubbed := middleware.ScrubSentryEvent(event, nil)

	require.NotNil(t, scrubbed)
	require.NotNil(t, scrubbed.Request)
	assert.Contains(t, scrubbed.Request.QueryString, "q=orders")
	assert.Equal(t, "[Filtered]", sentryQueryValue(t, scrubbed.Request.QueryString, "customer_email"))
	assert.Equal(t, "[Filtered]", sentryQueryValue(t, scrubbed.Request.QueryString, "userEmail"))
	assert.Equal(t, "[Filtered]", sentryQueryValue(t, scrubbed.Request.QueryString, "email"))
	assert.NotContains(t, scrubbed.Request.QueryString, "customer@example.com")
	assert.NotContains(t, scrubbed.Request.QueryString, "buyer@example.com")
	assert.NotContains(t, scrubbed.Request.QueryString, "owner@example.com")
	assert.NotContains(t, scrubbed.Request.QueryString, "secret-token")
	assert.Empty(t, scrubbed.Request.Cookies)
	assert.Empty(t, scrubbed.Request.Data)
	assert.Equal(t, "safe-value", scrubbed.Request.Headers["Safe-Header"])
	assertNoSentryHeader(t, scrubbed.Request.Headers, "Authorization")
	assertNoSentryHeader(t, scrubbed.Request.Headers, "X-CSRF-Token")
	assert.NotContains(t, scrubbed.Request.Env, "REMOTE_ADDR")
	assert.NotContains(t, scrubbed.Request.Env, "REMOTE_PORT")
}

func assertNoSentryHeader(t *testing.T, headers map[string]string, name string) {
	t.Helper()
	for key := range headers {
		assert.Falsef(t, strings.EqualFold(key, name), "header %q should have been scrubbed", key)
	}
}

func sentryQueryValue(t *testing.T, rawQuery, key string) string {
	t.Helper()

	values, err := url.ParseQuery(rawQuery)
	require.NoError(t, err)
	return values.Get(key)
}
