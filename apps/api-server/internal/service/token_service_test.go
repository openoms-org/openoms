package service

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

func TestNewTokenService_Deterministic(t *testing.T) {
	ts1, err := NewTokenService("test-secret-that-is-at-least-32-characters-long")
	if err != nil {
		t.Fatalf("NewTokenService: %v", err)
	}
	ts2, err := NewTokenService("test-secret-that-is-at-least-32-characters-long")
	if err != nil {
		t.Fatalf("NewTokenService: %v", err)
	}
	if !ts1.publicKey.Equal(ts2.publicKey) {
		t.Error("same secret should produce same keypair")
	}
}

func TestNewTokenService_TooShort(t *testing.T) {
	_, err := NewTokenService("short")
	if err == nil {
		t.Error("expected error for short secret")
	}
}

func TestGenerateAndValidateAccessToken(t *testing.T) {
	ts, _ := NewTokenService("test-secret-that-is-at-least-32-characters-long")
	user := model.User{
		ID:       uuid.New(),
		TenantID: uuid.New(),
		Email:    "test@example.com",
		Role:     "admin",
	}
	tokenStr, err := ts.GenerateAccessToken(user)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}
	claims, err := ts.ValidateToken(tokenStr)
	if err != nil {
		t.Fatalf("ValidateToken: %v", err)
	}
	if claims.Subject != user.ID.String() {
		t.Errorf("subject = %s, want %s", claims.Subject, user.ID)
	}
	if claims.TenantID != user.TenantID {
		t.Errorf("tenant_id = %s, want %s", claims.TenantID, user.TenantID)
	}
	if claims.Email != user.Email {
		t.Errorf("email = %s, want %s", claims.Email, user.Email)
	}
	if claims.Role != user.Role {
		t.Errorf("role = %s, want %s", claims.Role, user.Role)
	}
}

func TestGenerateAndValidateRefreshToken(t *testing.T) {
	ts, _ := NewTokenService("test-secret-that-is-at-least-32-characters-long")
	user := model.User{
		ID:       uuid.New(),
		TenantID: uuid.New(),
		Email:    "test@example.com",
		Role:     "owner",
	}
	tokenStr, err := ts.GenerateRefreshToken(user)
	if err != nil {
		t.Fatalf("GenerateRefreshToken: %v", err)
	}
	claims, err := ts.ValidateToken(tokenStr)
	if err != nil {
		t.Fatalf("ValidateToken: %v", err)
	}
	if claims.Subject != user.ID.String() {
		t.Errorf("subject = %s, want %s", claims.Subject, user.ID)
	}
	if claims.TenantID != user.TenantID {
		t.Errorf("tenant_id = %s, want %s", claims.TenantID, user.TenantID)
	}
}

func TestValidateToken_InvalidSignature(t *testing.T) {
	ts1, _ := NewTokenService("test-secret-that-is-at-least-32-characters-long")
	ts2, _ := NewTokenService("different-secret-at-least-32-characters-long!!")
	user := model.User{ID: uuid.New(), TenantID: uuid.New()}
	tokenStr, _ := ts1.GenerateAccessToken(user)
	_, err := ts2.ValidateToken(tokenStr)
	if err == nil {
		t.Error("expected error for token signed with different key")
	}
}

func TestValidateToken_Expired(t *testing.T) {
	ts, _ := NewTokenService("test-secret-that-is-at-least-32-characters-long")
	user := model.User{ID: uuid.New(), TenantID: uuid.New()}
	// Create token that expired 1 hour ago
	now := time.Now().Add(-2 * time.Hour)
	claims := model.AuthClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(1 * time.Hour)),
			Issuer:    "openoms",
		},
		TenantID: user.TenantID,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	tokenStr, _ := token.SignedString(ts.privateKey)

	_, err := ts.ValidateToken(tokenStr)
	if err == nil {
		t.Error("expected error for expired token")
	}
}
