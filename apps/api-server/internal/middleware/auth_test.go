package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// mockValidator implements TokenValidator for testing.
type mockValidator struct {
	claims *model.AuthClaims
	err    error
}

func (m *mockValidator) ValidateToken(tokenStr string) (*model.AuthClaims, error) {
	return m.claims, m.err
}

func TestJWTAuth_ValidToken(t *testing.T) {
	userID := uuid.New()
	tenantID := uuid.New()

	validator := &mockValidator{
		claims: &model.AuthClaims{
			RegisteredClaims: jwt.RegisteredClaims{Subject: userID.String()},
			TenantID:         tenantID,
			Email:            "test@example.com",
			Role:             "admin",
		},
	}

	var capturedTenantID uuid.UUID
	var capturedUserID uuid.UUID
	var capturedClaims *model.AuthClaims

	handler := JWTAuth(validator)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedTenantID = TenantIDFromContext(r.Context())
		capturedUserID = UserIDFromContext(r.Context())
		capturedClaims = ClaimsFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	if capturedTenantID != tenantID {
		t.Errorf("tenant_id = %s, want %s", capturedTenantID, tenantID)
	}
	if capturedUserID != userID {
		t.Errorf("user_id = %s, want %s", capturedUserID, userID)
	}
	if capturedClaims == nil {
		t.Fatal("claims should not be nil")
	}
	if capturedClaims.Role != "admin" {
		t.Errorf("role = %s, want admin", capturedClaims.Role)
	}
}

func TestJWTAuth_MissingHeader(t *testing.T) {
	handler := JWTAuth(&mockValidator{})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestJWTAuth_InvalidToken(t *testing.T) {
	validator := &mockValidator{
		err: jwt.ErrTokenExpired,
	}

	handler := JWTAuth(validator)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestJWTAuth_NoBearerPrefix(t *testing.T) {
	handler := JWTAuth(&mockValidator{})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Basic abc123")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}
