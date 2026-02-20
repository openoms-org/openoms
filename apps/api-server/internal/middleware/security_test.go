package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/stretchr/testify/assert"
)

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func TestSecurity_RateLimit_ExceedLimitReturns429(t *testing.T) {
	limiter := middleware.NewMemoryRateLimiter()
	handler := middleware.RateLimitWith(limiter, 3, 1*time.Minute)(okHandler())

	for i := range 3 {
		req := httptest.NewRequest("POST", "/v1/auth/login", nil)
		req.RemoteAddr = "1.2.3.4:1234"
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code, "request %d should pass", i+1)
	}

	req := httptest.NewRequest("POST", "/v1/auth/login", nil)
	req.RemoteAddr = "1.2.3.4:1234"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusTooManyRequests, rr.Code)
}

func TestSecurity_RateLimit_RetryAfterHeader(t *testing.T) {
	limiter := middleware.NewMemoryRateLimiter()
	handler := middleware.RateLimitWith(limiter, 1, 1*time.Minute)(okHandler())

	// Exhaust limit
	req := httptest.NewRequest("POST", "/", nil)
	req.RemoteAddr = "5.5.5.5:1234"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	// Next request should be rate limited
	req = httptest.NewRequest("POST", "/", nil)
	req.RemoteAddr = "5.5.5.5:1234"
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusTooManyRequests, rr.Code)
	assert.NotEmpty(t, rr.Header().Get("Retry-After"))
}

func TestSecurity_RateLimit_DifferentIPsIndependent(t *testing.T) {
	limiter := middleware.NewMemoryRateLimiter()
	handler := middleware.RateLimitWith(limiter, 1, 1*time.Minute)(okHandler())

	req1 := httptest.NewRequest("POST", "/", nil)
	req1.RemoteAddr = "1.1.1.1:1234"
	rr1 := httptest.NewRecorder()
	handler.ServeHTTP(rr1, req1)
	assert.Equal(t, http.StatusOK, rr1.Code)

	req2 := httptest.NewRequest("POST", "/", nil)
	req2.RemoteAddr = "2.2.2.2:1234"
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)
	assert.Equal(t, http.StatusOK, rr2.Code)
}

func TestSecurity_Blacklist_RevokedTokenIsBlocked(t *testing.T) {
	bl := middleware.NewMemoryTokenBlacklist()
	bl.Revoke("hash123", time.Now().Add(1*time.Hour))
	assert.True(t, bl.IsRevoked("hash123"))
}

func TestSecurity_Blacklist_NonRevokedTokenPasses(t *testing.T) {
	bl := middleware.NewMemoryTokenBlacklist()
	assert.False(t, bl.IsRevoked("hash999"))
}

func TestSecurity_Blacklist_ExpiredEntryNotRevoked(t *testing.T) {
	bl := middleware.NewMemoryTokenBlacklist()
	bl.Revoke("hashexpired", time.Now().Add(-1*time.Second))
	assert.False(t, bl.IsRevoked("hashexpired"))
}

func TestSecurity_Headers_AllPresent(t *testing.T) {
	handler := middleware.SecurityHeaders(okHandler())
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, "nosniff", rr.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", rr.Header().Get("X-Frame-Options"))
	assert.Equal(t, "strict-origin-when-cross-origin", rr.Header().Get("Referrer-Policy"))
	assert.Contains(t, rr.Header().Get("Permissions-Policy"), "camera=()")
	assert.Equal(t, "frame-ancestors 'none'", rr.Header().Get("Content-Security-Policy"))
	assert.Contains(t, rr.Header().Get("Strict-Transport-Security"), "max-age=")
}

func TestSecurity_Headers_HSTSOnlyOnHTTPS(t *testing.T) {
	handler := middleware.SecurityHeaders(okHandler())
	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Empty(t, rr.Header().Get("Strict-Transport-Security"))
}

func TestSecurity_Headers_NoServerHeader(t *testing.T) {
	handler := middleware.SecurityHeaders(okHandler())
	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Empty(t, rr.Header().Get("Server"))
	assert.Empty(t, rr.Header().Get("X-Powered-By"))
}

func TestSecurity_CSRF_GETSetsCookie(t *testing.T) {
	handler := middleware.CSRF(false, "")(okHandler())
	req := httptest.NewRequest("GET", "/v1/orders", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	cookies := rr.Result().Cookies()
	var csrfCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "csrf_token" {
			csrfCookie = c
		}
	}
	assert.NotNil(t, csrfCookie, "csrf_token cookie should be set")
	assert.Equal(t, http.SameSiteLaxMode, csrfCookie.SameSite)
	assert.Equal(t, "/", csrfCookie.Path)
	assert.False(t, csrfCookie.HttpOnly, "JS must be able to read csrf_token")
}

func TestSecurity_CSRF_POSTWithoutTokenReturns403(t *testing.T) {
	handler := middleware.CSRF(false, "")(okHandler())
	req := httptest.NewRequest("POST", "/v1/orders", nil)
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: "test-token"})
	// No X-CSRF-Token header
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestSecurity_CSRF_POSTWithValidTokenPasses(t *testing.T) {
	handler := middleware.CSRF(false, "")(okHandler())
	req := httptest.NewRequest("POST", "/v1/orders", nil)
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: "test-token"})
	req.Header.Set("X-CSRF-Token", "test-token")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestSecurity_CSRF_POSTWithInvalidTokenReturns403(t *testing.T) {
	handler := middleware.CSRF(false, "")(okHandler())
	req := httptest.NewRequest("POST", "/v1/orders", nil)
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: "correct-token"})
	req.Header.Set("X-CSRF-Token", "wrong-token")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestSecurity_CSRF_ExemptEndpointPassesWithoutToken(t *testing.T) {
	handler := middleware.CSRF(false, "")(okHandler())

	endpoints := []string{"/v1/auth/login", "/v1/auth/register", "/v1/auth/refresh", "/v1/auth/ws-ticket", "/v1/public/returns/submit", "/v1/webhooks/allegro", "/health", "/metrics"}
	for _, ep := range endpoints {
		req := httptest.NewRequest("POST", ep, nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code, "exempt endpoint %s should pass", ep)
	}
}

func TestSecurity_CORS_AllowedOriginReflected(t *testing.T) {
	handler := middleware.CORS([]string{"https://app.openoms.org"})(okHandler())
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://app.openoms.org")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, "https://app.openoms.org", rr.Header().Get("Access-Control-Allow-Origin"))
}

func TestSecurity_CORS_DisallowedOriginBlocked(t *testing.T) {
	handler := middleware.CORS([]string{"https://app.openoms.org"})(okHandler())
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://evil.com")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Empty(t, rr.Header().Get("Access-Control-Allow-Origin"))
}

func TestSecurity_CORS_NullOriginBlocked(t *testing.T) {
	handler := middleware.CORS([]string{"https://app.openoms.org"})(okHandler())
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "null")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Empty(t, rr.Header().Get("Access-Control-Allow-Origin"))
}

func TestSecurity_CORS_PreflightReturns204(t *testing.T) {
	handler := middleware.CORS([]string{"https://app.openoms.org"})(okHandler())
	req := httptest.NewRequest("OPTIONS", "/v1/orders", nil)
	req.Header.Set("Origin", "https://app.openoms.org")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusNoContent, rr.Code)
	assert.Contains(t, rr.Header().Get("Access-Control-Allow-Headers"), "X-CSRF-Token")
}
