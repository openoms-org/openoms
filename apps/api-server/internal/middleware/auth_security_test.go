package middleware_test

import (
	"crypto/ed25519"
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
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

const testSecret = "test-secret-32-chars-long-enough" //nolint:gosec // test credential

func newTestTokenService(t *testing.T) *service.TokenService {
	t.Helper()
	svc, err := service.NewTokenService(testSecret)
	require.NoError(t, err)
	return svc
}

func TestSecurity_Auth_RejectsEmptyBearer(t *testing.T) {
	svc := newTestTokenService(t)
	handler := middleware.JWTAuth(svc)(okHandler())
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer ")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestSecurity_Auth_RejectsMalformedToken(t *testing.T) {
	svc := newTestTokenService(t)
	handler := middleware.JWTAuth(svc)(okHandler())
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer not.a.jwt")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestSecurity_Auth_RejectsExpiredToken(t *testing.T) {
	svc := newTestTokenService(t)
	hash := sha512.Sum512([]byte(testSecret))
	privKey := ed25519.NewKeyFromSeed(hash[:32])

	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, &model.AuthClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "550e8400-e29b-41d4-a716-446655440000",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Second)),
		},
		TenantID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"),
		Type:     "access",
	})
	tokenStr, _ := token.SignedString(privKey)

	handler := middleware.JWTAuth(svc)(okHandler())
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestSecurity_Auth_RejectsRefreshTokenAsAccess(t *testing.T) {
	svc := newTestTokenService(t)
	hash := sha512.Sum512([]byte(testSecret))
	privKey := ed25519.NewKeyFromSeed(hash[:32])

	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, &model.AuthClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "550e8400-e29b-41d4-a716-446655440000",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		},
		TenantID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"),
		Type:     "refresh",
	})
	tokenStr, _ := token.SignedString(privKey)

	handler := middleware.JWTAuth(svc)(okHandler())
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestSecurity_Auth_RejectsAlgNoneToken(t *testing.T) {
	svc := newTestTokenService(t)
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	payload := base64.RawURLEncoding.EncodeToString(fmt.Appendf(nil,
		`{"sub":"550e8400-e29b-41d4-a716-446655440000","exp":%d,"tid":"550e8400-e29b-41d4-a716-446655440001","type":"access"}`,
		time.Now().Add(1*time.Hour).Unix(),
	))
	tokenStr := header + "." + payload + "."

	handler := middleware.JWTAuth(svc)(okHandler())
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestSecurity_Auth_BlacklistedTokenRejected(t *testing.T) {
	svc := newTestTokenService(t)
	hash := sha512.Sum512([]byte(testSecret))
	privKey := ed25519.NewKeyFromSeed(hash[:32])

	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, &model.AuthClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "550e8400-e29b-41d4-a716-446655440000",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		},
		TenantID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"),
		Type:     "access",
	})
	tokenStr, _ := token.SignedString(privKey)

	bl := middleware.NewTokenBlacklist()
	bl.Revoke(middleware.HashToken(tokenStr), time.Now().Add(1*time.Hour))

	handler := middleware.JWTAuth(svc, bl)(okHandler())
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}
