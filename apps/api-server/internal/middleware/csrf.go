package middleware

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
)

// CSRF validates CSRF tokens on state-changing requests.
// The `secure` parameter controls the Secure flag on the cookie.
func CSRF(secure bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Exempt paths
			path := r.URL.Path
			if isCSRFExempt(path) {
				next.ServeHTTP(w, r)
				return
			}

			// For safe methods, set the cookie if not present and proceed
			if r.Method == "GET" || r.Method == "HEAD" || r.Method == "OPTIONS" {
				setCSRFCookieIfMissing(w, r, secure)
				next.ServeHTTP(w, r)
				return
			}

			// State-changing methods: validate token
			cookie, err := r.Cookie("csrf_token")
			if err != nil || cookie.Value == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]string{"error": "missing CSRF token"})
				return
			}

			headerToken := r.Header.Get("X-CSRF-Token")
			if subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(headerToken)) != 1 {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]string{"error": "invalid CSRF token"})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func isCSRFExempt(path string) bool {
	exemptPrefixes := []string{
		"/v1/auth/login",
		"/v1/auth/register",
		"/v1/public/",
		"/v1/webhooks/",
		"/health",
		"/metrics",
	}
	for _, prefix := range exemptPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

func setCSRFCookieIfMissing(w http.ResponseWriter, r *http.Request, secure bool) {
	if _, err := r.Cookie("csrf_token"); err == nil {
		return // already has cookie
	}

	token := generateCSRFToken()
	http.SetCookie(w, &http.Cookie{
		Name:     "csrf_token",
		Value:    token,
		Path:     "/",
		SameSite: http.SameSiteStrictMode,
		Secure:   secure,
		HttpOnly: false, // JS must read this
	})
}

func generateCSRFToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}
