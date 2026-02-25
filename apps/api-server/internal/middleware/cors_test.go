package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/stretchr/testify/assert"
)

func TestCORS_AllowedOrigin_SetsHeaders(t *testing.T) {
	handler := middleware.CORS([]string{"https://app.openoms.org"})(okHandler())
	req := httptest.NewRequest("GET", "/v1/orders", nil)
	req.Header.Set("Origin", "https://app.openoms.org")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "https://app.openoms.org", rr.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORS_NonAllowedOrigin_NoCORSHeaders(t *testing.T) {
	handler := middleware.CORS([]string{"https://app.openoms.org"})(okHandler())
	req := httptest.NewRequest("GET", "/v1/orders", nil)
	req.Header.Set("Origin", "https://evil.com")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Empty(t, rr.Header().Get("Access-Control-Allow-Origin"))
	assert.Empty(t, rr.Header().Get("Access-Control-Allow-Methods"))
	assert.Empty(t, rr.Header().Get("Access-Control-Allow-Credentials"))
	// Vary: Origin must still be present
	assert.Contains(t, rr.Header().Values("Vary"), "Origin")
}

func TestCORS_Preflight_Returns204WithMethods(t *testing.T) {
	handler := middleware.CORS([]string{"https://app.openoms.org"})(okHandler())
	req := httptest.NewRequest("OPTIONS", "/v1/orders", nil)
	req.Header.Set("Origin", "https://app.openoms.org")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNoContent, rr.Code)
	assert.Contains(t, rr.Header().Get("Access-Control-Allow-Methods"), "GET")
	assert.Contains(t, rr.Header().Get("Access-Control-Allow-Methods"), "POST")
	assert.Contains(t, rr.Header().Get("Access-Control-Allow-Methods"), "DELETE")
}

func TestCORS_Preflight_NonAllowedOrigin_Returns204NoCORSHeaders(t *testing.T) {
	handler := middleware.CORS([]string{"https://app.openoms.org"})(okHandler())
	req := httptest.NewRequest("OPTIONS", "/v1/orders", nil)
	req.Header.Set("Origin", "https://evil.com")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNoContent, rr.Code)
	assert.Empty(t, rr.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORS_TrailingSlashInAllowedOrigins_Trimmed(t *testing.T) {
	handler := middleware.CORS([]string{"https://app.openoms.org/"})(okHandler())
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://app.openoms.org")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, "https://app.openoms.org", rr.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORS_VaryOrigin_AlwaysPresent(t *testing.T) {
	handler := middleware.CORS([]string{"https://app.openoms.org"})(okHandler())

	// Without Origin header
	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Contains(t, rr.Header().Values("Vary"), "Origin")

	// With non-matching Origin
	req = httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://other.com")
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Contains(t, rr.Header().Values("Vary"), "Origin")

	// With matching Origin
	req = httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://app.openoms.org")
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Contains(t, rr.Header().Values("Vary"), "Origin")
}

func TestCORS_AllowCredentials_TrueWhenOriginMatches(t *testing.T) {
	handler := middleware.CORS([]string{"https://app.openoms.org"})(okHandler())
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://app.openoms.org")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, "true", rr.Header().Get("Access-Control-Allow-Credentials"))
}

func TestCORS_MultipleAllowedOrigins(t *testing.T) {
	origins := []string{"https://app.openoms.org", "https://localhost:3000"}
	handler := middleware.CORS(origins)(okHandler())

	for _, origin := range origins {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Origin", origin)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		assert.Equal(t, origin, rr.Header().Get("Access-Control-Allow-Origin"), "origin %s should be allowed", origin)
	}
}
