package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

func withClaims(r *http.Request, role string) *http.Request {
	claims := &model.AuthClaims{
		RegisteredClaims: jwt.RegisteredClaims{},
		Role:             role,
	}
	ctx := context.WithValue(r.Context(), ClaimsKey, claims)
	return r.WithContext(ctx)
}

func TestRequireRole_OwnerCanAccessAdmin(t *testing.T) {
	handler := RequireRole("admin")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := withClaims(httptest.NewRequest("GET", "/test", nil), "owner")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200 (owner should pass admin check)", rec.Code)
	}
}

func TestRequireRole_AdminCanAccessAdmin(t *testing.T) {
	handler := RequireRole("admin")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := withClaims(httptest.NewRequest("GET", "/test", nil), "admin")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200 (admin should pass admin check)", rec.Code)
	}
}

func TestRequireRole_MemberCannotAccessAdmin(t *testing.T) {
	handler := RequireRole("admin")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := withClaims(httptest.NewRequest("GET", "/test", nil), "member")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403 (member should not pass admin check)", rec.Code)
	}
}

func TestRequireRole_NoClaims(t *testing.T) {
	handler := RequireRole("admin")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401 (no claims should fail)", rec.Code)
	}
}
