package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/getsentry/sentry-go"
	"github.com/stretchr/testify/assert"

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
