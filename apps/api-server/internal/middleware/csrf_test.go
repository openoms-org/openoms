package middleware

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func csrfOkHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

// --- isCSRFExempt tests ---

func TestIsCSRFExempt_ExemptPaths(t *testing.T) {
	exemptPaths := []struct {
		path string
		desc string
	}{
		{"/v1/auth/login", "login"},
		{"/v1/auth/register", "register"},
		{"/v1/auth/refresh", "refresh"},
		{"/v1/auth/ws-ticket", "ws-ticket"},
		{"/v1/billing/checkout", "billing checkout"},
		{"/v1/billing/plans", "billing plans"},
		{"/v1/public/returns/submit", "public returns submit"},
		{"/v1/public/anything", "public wildcard"},
		{"/v1/webhooks/allegro", "webhooks allegro"},
		{"/v1/webhooks/inpost", "webhooks inpost"},
		{"/health", "health"},
		{"/metrics", "metrics"},
	}

	for _, tc := range exemptPaths {
		t.Run(tc.desc, func(t *testing.T) {
			assert.True(t, isCSRFExempt(tc.path), "path %s should be exempt", tc.path)
		})
	}
}

func TestIsCSRFExempt_NonExemptPaths(t *testing.T) {
	nonExemptPaths := []struct {
		path string
		desc string
	}{
		{"/v1/orders", "orders"},
		{"/v1/products", "products"},
		{"/v1/auth/logout", "logout should require CSRF"},
		{"/v1/auth/2fa/enable", "2fa enable (not exempt prefix)"},
		{"/v1/users/me", "users me"},
		{"/v1/settings/company", "settings"},
		{"/", "root"},
		{"/v1/shipments", "shipments"},
	}

	for _, tc := range nonExemptPaths {
		t.Run(tc.desc, func(t *testing.T) {
			assert.False(t, isCSRFExempt(tc.path), "path %s should NOT be exempt", tc.path)
		})
	}
}

// --- generateCSRFToken tests ---

func TestGenerateCSRFToken_ReturnsBase64URL(t *testing.T) {
	token := generateCSRFToken()
	assert.NotEmpty(t, token)

	// Should be valid base64 RawURL encoding of 32 bytes
	decoded, err := base64.RawURLEncoding.DecodeString(token)
	require.NoError(t, err, "token should be valid base64url")
	assert.Len(t, decoded, 32, "decoded token should be 32 bytes")
}

func TestGenerateCSRFToken_UniqueTokens(t *testing.T) {
	token1 := generateCSRFToken()
	token2 := generateCSRFToken()
	assert.NotEqual(t, token1, token2, "consecutive tokens should differ")
}

func TestGenerateCSRFToken_ExpectedLength(t *testing.T) {
	token := generateCSRFToken()
	// 32 bytes in base64url without padding = ceil(32*4/3) = 43 chars
	assert.Len(t, token, 43, "base64url of 32 bytes without padding should be 43 chars")
}

// --- CSRF middleware: safe methods ---

func TestCSRF_SafeMethods_SetCookieIfMissing(t *testing.T) {
	safeMethods := []string{"GET", "HEAD", "OPTIONS"}

	for _, method := range safeMethods {
		t.Run(method, func(t *testing.T) {
			handler := CSRF(false, "")(csrfOkHandler())
			req := httptest.NewRequest(method, "/v1/orders", nil)
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			assert.Equal(t, http.StatusOK, rr.Code)

			cookies := rr.Result().Cookies()
			var found bool
			for _, c := range cookies {
				if c.Name == "csrf_token" {
					found = true
					assert.NotEmpty(t, c.Value)
				}
			}
			assert.True(t, found, "%s should set csrf_token cookie", method)
		})
	}
}

func TestCSRF_SafeMethods_DoNotRequireHeader(t *testing.T) {
	handler := CSRF(false, "")(csrfOkHandler())

	// GET without any CSRF header or cookie should still pass
	req := httptest.NewRequest("GET", "/v1/orders", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

// --- CSRF middleware: state-changing methods ---

func TestCSRF_POST_MissingCookie_Returns403(t *testing.T) {
	handler := CSRF(false, "")(csrfOkHandler())
	req := httptest.NewRequest("POST", "/v1/orders", nil)
	// No cookie, no header
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)

	var body map[string]string
	err := json.NewDecoder(rr.Body).Decode(&body)
	require.NoError(t, err)
	assert.Equal(t, "missing CSRF token", body["error"])
}

func TestCSRF_POST_EmptyCookieValue_Returns403(t *testing.T) {
	handler := CSRF(false, "")(csrfOkHandler())
	req := httptest.NewRequest("POST", "/v1/orders", nil)
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: ""})
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestCSRF_POST_MissingHeader_Returns403(t *testing.T) {
	handler := CSRF(false, "")(csrfOkHandler())
	req := httptest.NewRequest("POST", "/v1/orders", nil)
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: "valid-token"})
	// No X-CSRF-Token header
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)

	var body map[string]string
	err := json.NewDecoder(rr.Body).Decode(&body)
	require.NoError(t, err)
	assert.Equal(t, "invalid CSRF token", body["error"])
}

func TestCSRF_POST_WrongHeader_Returns403(t *testing.T) {
	handler := CSRF(false, "")(csrfOkHandler())
	req := httptest.NewRequest("POST", "/v1/orders", nil)
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: "correct-token"})
	req.Header.Set("X-CSRF-Token", "wrong-token")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)

	var body map[string]string
	err := json.NewDecoder(rr.Body).Decode(&body)
	require.NoError(t, err)
	assert.Equal(t, "invalid CSRF token", body["error"])
}

func TestCSRF_POST_MatchingTokenPasses(t *testing.T) {
	handler := CSRF(false, "")(csrfOkHandler())
	req := httptest.NewRequest("POST", "/v1/orders", nil)
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: "my-secret-token"})
	req.Header.Set("X-CSRF-Token", "my-secret-token")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestCSRF_AllStateChangingMethods_RequireValidation(t *testing.T) {
	methods := []string{"POST", "PUT", "PATCH", "DELETE"}

	for _, method := range methods {
		t.Run(method+"_valid", func(t *testing.T) {
			handler := CSRF(false, "")(csrfOkHandler())
			req := httptest.NewRequest(method, "/v1/orders", nil)
			req.AddCookie(&http.Cookie{Name: "csrf_token", Value: "tok"})
			req.Header.Set("X-CSRF-Token", "tok")
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
			assert.Equal(t, http.StatusOK, rr.Code)
		})

		t.Run(method+"_invalid", func(t *testing.T) {
			handler := CSRF(false, "")(csrfOkHandler())
			req := httptest.NewRequest(method, "/v1/orders", nil)
			req.AddCookie(&http.Cookie{Name: "csrf_token", Value: "tok"})
			req.Header.Set("X-CSRF-Token", "bad")
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
			assert.Equal(t, http.StatusForbidden, rr.Code)
		})
	}
}

// --- CSRF middleware: exempt paths ---

func TestCSRF_ExemptPaths_BypassValidation(t *testing.T) {
	handler := CSRF(false, "")(csrfOkHandler())

	exemptPaths := []string{
		"/v1/auth/login",
		"/v1/auth/register",
		"/v1/auth/refresh",
		"/v1/auth/ws-ticket",
		"/v1/billing/checkout",
		"/v1/billing/plans",
		"/v1/public/returns/submit",
		"/v1/public/returns/status",
		"/v1/webhooks/allegro",
		"/v1/webhooks/inpost",
		"/health",
		"/metrics",
	}

	for _, path := range exemptPaths {
		t.Run(path, func(t *testing.T) {
			// POST without any CSRF token should still pass on exempt paths
			req := httptest.NewRequest("POST", path, nil)
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
			assert.Equal(t, http.StatusOK, rr.Code, "exempt path %s should bypass CSRF", path)
		})
	}
}

func TestCSRF_NonExemptPath_RequiresValidation(t *testing.T) {
	handler := CSRF(false, "")(csrfOkHandler())

	nonExemptPaths := []string{
		"/v1/orders",
		"/v1/products",
		"/v1/shipments",
		"/v1/auth/logout",
		"/v1/settings/company",
	}

	for _, path := range nonExemptPaths {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest("POST", path, nil)
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
			assert.Equal(t, http.StatusForbidden, rr.Code, "non-exempt path %s should require CSRF", path)
		})
	}
}

// --- CSRF cookie attributes ---

func TestCSRF_CookieAttributes_DevMode(t *testing.T) {
	handler := CSRF(false, "")(csrfOkHandler())
	req := httptest.NewRequest("GET", "/v1/orders", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	cookies := rr.Result().Cookies()
	var csrfCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "csrf_token" && c.Value != "" {
			csrfCookie = c
		}
	}
	require.NotNil(t, csrfCookie)
	assert.False(t, csrfCookie.HttpOnly, "JS must be able to read the cookie")
	assert.Equal(t, http.SameSiteLaxMode, csrfCookie.SameSite)
	assert.Equal(t, "/", csrfCookie.Path)
	assert.False(t, csrfCookie.Secure, "secure should be false in dev mode")
}

func TestCSRF_CookieAttributes_SecureMode(t *testing.T) {
	handler := CSRF(true, ".openoms.org")(csrfOkHandler())
	req := httptest.NewRequest("GET", "/v1/orders", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	cookies := rr.Result().Cookies()
	var csrfCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "csrf_token" && c.Value != "" {
			csrfCookie = c
		}
	}
	require.NotNil(t, csrfCookie)
	assert.True(t, csrfCookie.Secure, "secure should be true in production")
	assert.Equal(t, "openoms.org", csrfCookie.Domain)
}

// --- CSRF: cookie domain migration ---

func TestCSRF_GET_ExistingCookie_DevMode_NoCookieRewrite(t *testing.T) {
	handler := CSRF(false, "")(csrfOkHandler())
	req := httptest.NewRequest("GET", "/v1/orders", nil)
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: "existing-token"})
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	// In dev mode (empty cookieDomain), when cookie already exists, the function
	// returns early -- so it still sets a new cookie because the code always
	// generates a new token after the early-return block. Actually looking at
	// the code: if cookieDomain == "" and cookie exists, it returns -- so no
	// Set-Cookie at all.
	cookies := rr.Result().Cookies()
	assert.Empty(t, cookies, "dev mode should not rewrite existing cookie")
}

func TestCSRF_GET_ExistingCookie_WithDomain_ExpiresThenSetsNew(t *testing.T) {
	handler := CSRF(false, ".openoms.org")(csrfOkHandler())
	req := httptest.NewRequest("GET", "/v1/orders", nil)
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: "old-host-only-token"})
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	cookies := rr.Result().Cookies()
	// Should have 2 cookies: one expiring old, one setting new
	var expireCookie, newCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "csrf_token" && c.MaxAge < 0 {
			expireCookie = c
		} else if c.Name == "csrf_token" && c.Value != "" {
			newCookie = c
		}
	}
	require.NotNil(t, expireCookie, "should expire old host-only cookie")
	require.NotNil(t, newCookie, "should set new domain-scoped cookie")
	assert.NotEqual(t, "old-host-only-token", newCookie.Value)
}

// --- CSRF: JSON response format ---

func TestCSRF_ErrorResponse_ContentTypeJSON(t *testing.T) {
	handler := CSRF(false, "")(csrfOkHandler())

	// Missing cookie entirely
	req := httptest.NewRequest("POST", "/v1/orders", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
}
