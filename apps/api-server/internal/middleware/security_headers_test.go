package middleware_test

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/stretchr/testify/assert"
)

func TestSecurityHeaders_SetsXContentTypeOptions(t *testing.T) {
	handler := middleware.SecurityHeaders(okHandler())
	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, "nosniff", rr.Header().Get("X-Content-Type-Options"))
}

func TestSecurityHeaders_SetsXFrameOptionsDENY(t *testing.T) {
	handler := middleware.SecurityHeaders(okHandler())
	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, "DENY", rr.Header().Get("X-Frame-Options"))
}

func TestSecurityHeaders_SetsReferrerPolicy(t *testing.T) {
	handler := middleware.SecurityHeaders(okHandler())
	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, "strict-origin-when-cross-origin", rr.Header().Get("Referrer-Policy"))
}

func TestSecurityHeaders_SetsPermissionsPolicy(t *testing.T) {
	handler := middleware.SecurityHeaders(okHandler())
	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	pp := rr.Header().Get("Permissions-Policy")
	assert.Contains(t, pp, "camera=()")
	assert.Contains(t, pp, "microphone=()")
	assert.Contains(t, pp, "geolocation=(self)")
}

func TestSecurityHeaders_SetsXXSSProtectionToZero(t *testing.T) {
	handler := middleware.SecurityHeaders(okHandler())
	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, "0", rr.Header().Get("X-XSS-Protection"))
}

func TestSecurityHeaders_SetsCSPFrameAncestorsNone(t *testing.T) {
	handler := middleware.SecurityHeaders(okHandler())
	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, "frame-ancestors 'none'", rr.Header().Get("Content-Security-Policy"))
}

func TestSecurityHeaders_HSTS_SetWhenXForwardedProtoHTTPS(t *testing.T) {
	handler := middleware.SecurityHeaders(okHandler())
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	hsts := rr.Header().Get("Strict-Transport-Security")
	assert.Contains(t, hsts, "max-age=63072000")
	assert.Contains(t, hsts, "includeSubDomains")
}

func TestSecurityHeaders_HSTS_SetWhenTLSConnection(t *testing.T) {
	handler := middleware.SecurityHeaders(okHandler())
	req := httptest.NewRequest("GET", "/", nil)
	req.TLS = &tls.ConnectionState{}
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	hsts := rr.Header().Get("Strict-Transport-Security")
	assert.Contains(t, hsts, "max-age=63072000")
	assert.Contains(t, hsts, "includeSubDomains")
}

func TestSecurityHeaders_HSTS_NotSetOnPlainHTTP(t *testing.T) {
	handler := middleware.SecurityHeaders(okHandler())
	req := httptest.NewRequest("GET", "/", nil)
	// No X-Forwarded-Proto, no TLS
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Empty(t, rr.Header().Get("Strict-Transport-Security"))
}

func TestSecurityHeaders_HSTS_NotSetWhenXForwardedProtoHTTP(t *testing.T) {
	handler := middleware.SecurityHeaders(okHandler())
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Forwarded-Proto", "http")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Empty(t, rr.Header().Get("Strict-Transport-Security"))
}

func TestSecurityHeaders_CallsNextHandler(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	handler := middleware.SecurityHeaders(next)
	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.True(t, called, "next handler should be called")
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestSecurityHeaders_HeadersPresentOnNonGET(t *testing.T) {
	methods := []string{"POST", "PUT", "DELETE", "PATCH"}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			handler := middleware.SecurityHeaders(okHandler())
			req := httptest.NewRequest(method, "/", nil)
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			assert.Equal(t, "nosniff", rr.Header().Get("X-Content-Type-Options"))
			assert.Equal(t, "DENY", rr.Header().Get("X-Frame-Options"))
			assert.Equal(t, "strict-origin-when-cross-origin", rr.Header().Get("Referrer-Policy"))
		})
	}
}

func TestSecurityHeaders_PreservesHandlerResponseBody(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":1}`))
	})
	handler := middleware.SecurityHeaders(next)
	req := httptest.NewRequest("POST", "/v1/orders", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)
	assert.Equal(t, `{"id":1}`, rr.Body.String())
	// Headers should still be present
	assert.Equal(t, "nosniff", rr.Header().Get("X-Content-Type-Options"))
}

func TestSecurityHeaders_AllHeadersPresentTogether(t *testing.T) {
	handler := middleware.SecurityHeaders(okHandler())
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Verify all 7 headers are set simultaneously
	assert.NotEmpty(t, rr.Header().Get("X-Content-Type-Options"))
	assert.NotEmpty(t, rr.Header().Get("X-Frame-Options"))
	assert.NotEmpty(t, rr.Header().Get("Referrer-Policy"))
	assert.NotEmpty(t, rr.Header().Get("Permissions-Policy"))
	assert.NotEmpty(t, rr.Header().Get("X-XSS-Protection"))
	assert.NotEmpty(t, rr.Header().Get("Content-Security-Policy"))
	assert.NotEmpty(t, rr.Header().Get("Strict-Transport-Security"))
}
