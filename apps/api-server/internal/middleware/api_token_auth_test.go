package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

type stubAPITokenAuth struct {
	claims *model.AuthClaims
	err    error
}

func (s stubAPITokenAuth) AuthenticateAPIToken(_ context.Context, _ string) (*model.AuthClaims, error) {
	return s.claims, s.err
}

func TestJWTAuth_APITokenSetsSameContextAsSession(t *testing.T) {
	userID := uuid.New()
	tenantID := uuid.New()
	lookup := stubAPITokenAuth{
		claims: &model.AuthClaims{
			RegisteredClaims: jwt.RegisteredClaims{Subject: userID.String()},
			TenantID:         tenantID,
			Email:            "rafal@example.com",
			Role:             "owner",
			Permissions:      model.SystemPermissionsForRole("owner"),
			Type:             "access",
		},
	}

	var capturedTenantID uuid.UUID
	var capturedUserID uuid.UUID
	var capturedRole string

	handler := JWTAuthWithAPITokens(&mockValidator{err: errors.New("not a jwt")}, lookup)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedTenantID = TenantIDFromContext(r.Context())
		capturedUserID = UserIDFromContext(r.Context())
		claims := ClaimsFromContext(r.Context())
		if claims != nil {
			capturedRole = claims.Role
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/v1/orders", nil)
	req.Header.Set("Authorization", "Bearer oms_test-token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if capturedTenantID != tenantID {
		t.Fatalf("tenant_id = %s, want %s", capturedTenantID, tenantID)
	}
	if capturedUserID != userID {
		t.Fatalf("user_id = %s, want %s", capturedUserID, userID)
	}
	if capturedRole != "owner" {
		t.Fatalf("role = %s, want owner", capturedRole)
	}
}

func TestJWTAuth_UnauthenticatedStillFailsWhenAPITokensConfigured(t *testing.T) {
	handler := JWTAuthWithAPITokens(&mockValidator{err: errors.New("not a jwt")}, stubAPITokenAuth{err: ErrLookupFailed})(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/v1/orders", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestJWTAuth_UnknownBearerStillFails(t *testing.T) {
	handler := JWTAuthWithAPITokens(&mockValidator{err: errors.New("not a jwt")}, stubAPITokenAuth{err: ErrLookupFailed})(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/v1/orders", nil)
	req.Header.Set("Authorization", "Bearer oms_unknown")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

var ErrLookupFailed = errors.New("api token not found")
