package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/stretchr/testify/assert"
)

func TestMetricsAuth_EmptyToken_Returns404(t *testing.T) {
	handler := middleware.MetricsAuth("")(okHandler())
	req := httptest.NewRequest("GET", "/metrics", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Contains(t, rr.Body.String(), "metrics disabled")
}

func TestMetricsAuth_MissingAuthorizationHeader_Returns401(t *testing.T) {
	handler := middleware.MetricsAuth("secret-token")(okHandler())
	req := httptest.NewRequest("GET", "/metrics", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.Equal(t, "Bearer", rr.Header().Get("WWW-Authenticate"))
}

func TestMetricsAuth_NonBearerAuth_Returns401(t *testing.T) {
	handler := middleware.MetricsAuth("secret-token")(okHandler())
	req := httptest.NewRequest("GET", "/metrics", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.Equal(t, "Bearer", rr.Header().Get("WWW-Authenticate"))
}

func TestMetricsAuth_WrongBearerToken_Returns401(t *testing.T) {
	handler := middleware.MetricsAuth("correct-token")(okHandler())
	req := httptest.NewRequest("GET", "/metrics", nil)
	req.Header.Set("Authorization", "Bearer wrong-token")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestMetricsAuth_CorrectBearerToken_Returns200(t *testing.T) {
	handler := middleware.MetricsAuth("secret-token")(okHandler())
	req := httptest.NewRequest("GET", "/metrics", nil)
	req.Header.Set("Authorization", "Bearer secret-token")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestMetricsAuth_WWWAuthenticate_OnUnauthorized(t *testing.T) {
	handler := middleware.MetricsAuth("token")(okHandler())

	// No auth header at all
	req := httptest.NewRequest("GET", "/metrics", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, "Bearer", rr.Header().Get("WWW-Authenticate"))

	// Non-Bearer prefix
	req = httptest.NewRequest("GET", "/metrics", nil)
	req.Header.Set("Authorization", "Token abc")
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, "Bearer", rr.Header().Get("WWW-Authenticate"))
}
