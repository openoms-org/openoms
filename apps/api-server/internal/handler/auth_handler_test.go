package handler

import (
	"crypto/ed25519"
	"crypto/sha512"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthHandler_Login_InvalidJSON(t *testing.T) {
	// authService can be nil because we expect to fail before calling it
	h := NewAuthHandler(nil, true)

	req := httptest.NewRequest(http.MethodPost, "/v1/auth/login", strings.NewReader("not json"))
	rr := httptest.NewRecorder()

	h.Login(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)

	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestAuthHandler_Register_InvalidJSON(t *testing.T) {
	h := NewAuthHandler(nil, true)

	req := httptest.NewRequest(http.MethodPost, "/v1/auth/register", strings.NewReader("{malformed"))
	rr := httptest.NewRecorder()

	h.Register(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)

	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestAuthHandler_Refresh_NoCookie(t *testing.T) {
	h := NewAuthHandler(nil, true)

	req := httptest.NewRequest(http.MethodPost, "/v1/auth/refresh", nil)
	rr := httptest.NewRecorder()

	h.Refresh(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)

	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "missing refresh token", resp["error"])
}

func TestAuthHandler_Logout(t *testing.T) {
	h := NewAuthHandler(nil, true)

	req := httptest.NewRequest(http.MethodPost, "/v1/auth/logout", nil)
	rr := httptest.NewRecorder()

	h.Logout(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	// Check refresh cookie is cleared
	cookies := rr.Result().Cookies()
	var found bool
	for _, c := range cookies {
		if c.Name == "refresh_token" {
			found = true
			assert.Equal(t, "", c.Value)
			assert.Equal(t, -1, c.MaxAge)
		}
	}
	assert.True(t, found, "refresh_token cookie should be set")
}

func TestAuthHandler_Logout_BlacklistsAccessTokenUntilTokenExpiry(t *testing.T) {
	const jwtSecret = "test-secret-with-enough-entropy-for-ed25519"
	tokenSvc, err := service.NewTokenService(jwtSecret)
	require.NoError(t, err)

	authSvc := service.NewAuthService(nil, nil, nil, tokenSvc, nil, nil)
	store := &captureTokenBlacklistStore{}
	h := NewAuthHandler(authSvc, true, middleware.NewTokenBlacklistWithStore(store))

	expiresAt := time.Now().Add(10 * time.Minute).Truncate(time.Second)
	tokenStr := signedAccessToken(t, jwtSecret, expiresAt)
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/logout", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	rr := httptest.NewRecorder()

	h.Logout(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	require.Len(t, store.revocations, 1)
	assert.Equal(t, middleware.HashToken(tokenStr), store.revocations[0].tokenHash)
	assert.WithinDuration(t, expiresAt, store.revocations[0].expiresAt, time.Second)
}

func TestAuthHandler_Logout_DoesNotBlacklistInvalidAccessToken(t *testing.T) {
	const jwtSecret = "test-secret-with-enough-entropy-for-ed25519"
	tokenSvc, err := service.NewTokenService(jwtSecret)
	require.NoError(t, err)

	authSvc := service.NewAuthService(nil, nil, nil, tokenSvc, nil, nil)
	store := &captureTokenBlacklistStore{}
	h := NewAuthHandler(authSvc, true, middleware.NewTokenBlacklistWithStore(store))

	req := httptest.NewRequest(http.MethodPost, "/v1/auth/logout", nil)
	req.Header.Set("Authorization", "Bearer not-a-valid-jwt")
	rr := httptest.NewRecorder()

	h.Logout(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Empty(t, store.revocations)
}

func TestClientIP(t *testing.T) {
	tests := []struct {
		remoteAddr string
		want       string
	}{
		{"192.168.1.1:8080", "192.168.1.1"},
		{"[::1]:59036", "::1"},
		{"192.168.1.1", "192.168.1.1"}, // no port
	}

	for _, tt := range tests {
		t.Run(tt.remoteAddr, func(t *testing.T) {
			r := &http.Request{RemoteAddr: tt.remoteAddr}
			assert.Equal(t, tt.want, clientIP(r))
		})
	}
}

type capturedTokenRevocation struct {
	tokenHash string
	expiresAt time.Time
}

type captureTokenBlacklistStore struct {
	revocations []capturedTokenRevocation
}

func (s *captureTokenBlacklistStore) Revoke(tokenHash string, expiresAt time.Time) {
	s.revocations = append(s.revocations, capturedTokenRevocation{
		tokenHash: tokenHash,
		expiresAt: expiresAt,
	})
}

func (s *captureTokenBlacklistStore) IsRevoked(string) bool {
	return false
}

func signedAccessToken(t *testing.T, jwtSecret string, expiresAt time.Time) string {
	t.Helper()

	hash := sha512.Sum512([]byte(jwtSecret))
	privateKey := ed25519.NewKeyFromSeed(hash[:ed25519.SeedSize])
	tenantID := uuid.New()
	userID := uuid.New()
	claims := model.AuthClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			Issuer:    "openoms",
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-time.Minute)),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
		TenantID: tenantID,
		Email:    "jan@test.pl",
		Role:     "admin",
		Type:     "access",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	tokenStr, err := token.SignedString(privateKey)
	require.NoError(t, err)
	return tokenStr
}

func TestAuthHandler_Register_LicenseToken_MissingBothTokens(t *testing.T) {
	h := &AuthHandler{registrationMode: "invite"}

	body := `{"email":"jan@test.pl","password":"test1234","name":"Jan","tenant_name":"Test","tenant_slug":"test"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.Register(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "invite_token, license_token, or checkout_session_id is required")
}

func TestIsValidationError(t *testing.T) {
	assert.True(t, isValidationError(service.NewValidationError(errors.New("email is required"))))
	assert.False(t, isValidationError(errors.New("some other error")))
	assert.False(t, isValidationError(nil))
	assert.False(t, isValidationError(errors.New("short")))
}

// --- Checkout Session Registration Tests ---

func TestAuthHandler_Register_Disabled(t *testing.T) {
	h := &AuthHandler{registrationMode: "disabled"}

	body := `{"email":"jan@test.pl","password":"test1234","name":"Jan","tenant_name":"Test","tenant_slug":"test"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/register", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.Register(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "registration is disabled", resp["error"])
}

func TestAuthHandler_Register_Closed(t *testing.T) {
	h := &AuthHandler{registrationMode: "closed"}

	body := `{"email":"jan@test.pl","password":"test1234","name":"Jan","tenant_name":"Test","tenant_slug":"test"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/register", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.Register(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "registration is disabled", resp["error"])
}

func TestAuthHandler_Register_UnknownModeFailsClosed(t *testing.T) {
	h := &AuthHandler{registrationMode: "typo"}

	body := `{"email":"jan@test.pl","password":"test1234","name":"Jan","tenant_name":"Test","tenant_slug":"test"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/register", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.Register(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "registration is disabled", resp["error"])
}

func TestAuthHandler_Register_InviteMode_CheckoutSessionWithoutService(t *testing.T) {
	// When in invite mode with checkout_session_id but no checkout service configured,
	// it should fall through to the default case requiring a valid token.
	h := &AuthHandler{registrationMode: "invite"}
	// checkoutSvc is nil

	body := `{"email":"jan@test.pl","password":"test1234","name":"Jan","tenant_name":"Test","tenant_slug":"test","checkout_session_id":"cs_test_123"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/register", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.Register(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "invite_token, license_token, or checkout_session_id is required")
}

func TestAuthHandler_Register_InviteMode_OnlyCheckoutSessionID(t *testing.T) {
	// When only checkout_session_id is provided but checkoutSvc is nil,
	// registration should fail with the generic token-required error.
	h := &AuthHandler{registrationMode: "invite"}

	body := `{"email":"jan@test.pl","password":"test1234","name":"Jan","tenant_name":"Test","tenant_slug":"test","checkout_session_id":"cs_test_abc"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/register", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.Register(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "invite_token, license_token, or checkout_session_id is required")
}
