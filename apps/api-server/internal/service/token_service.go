package service

import (
	"crypto/ed25519"
	"crypto/sha512"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

const (
	accessTokenDuration  = 1 * time.Hour
	refreshTokenDuration = 30 * 24 * time.Hour
)

// TokenService handles Ed25519 JWT token generation and validation.
type TokenService struct {
	privateKey ed25519.PrivateKey
	publicKey  ed25519.PublicKey
}

// NewTokenService creates a TokenService by deriving an Ed25519 keypair
// from the JWT secret via SHA-512.
func NewTokenService(jwtSecret string) (*TokenService, error) {
	if len(jwtSecret) < 32 {
		return nil, fmt.Errorf("JWT_SECRET must be at least 32 characters, got %d", len(jwtSecret))
	}
	hash := sha512.Sum512([]byte(jwtSecret))
	seed := hash[:ed25519.SeedSize] // 32 bytes
	privateKey := ed25519.NewKeyFromSeed(seed)
	publicKey := privateKey.Public().(ed25519.PublicKey)
	return &TokenService{privateKey: privateKey, publicKey: publicKey}, nil
}

// GenerateAccessToken creates a 1-hour JWT with full user claims.
func (s *TokenService) GenerateAccessToken(user model.User) (string, error) {
	now := time.Now()
	claims := model.AuthClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(accessTokenDuration)),
			Issuer:    "openoms",
		},
		TenantID: user.TenantID,
		Email:    user.Email,
		Role:     user.Role,
		Type:     "access",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	return token.SignedString(s.privateKey)
}

// GenerateRefreshToken creates a 30-day JWT with minimal claims.
func (s *TokenService) GenerateRefreshToken(user model.User) (string, error) {
	now := time.Now()
	claims := model.AuthClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(refreshTokenDuration)),
			Issuer:    "openoms",
		},
		TenantID: user.TenantID,
		Type:     "refresh",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	return token.SignedString(s.privateKey)
}

// ValidateToken parses and validates a JWT, returning the claims.
func (s *TokenService) ValidateToken(tokenStr string) (*model.AuthClaims, error) {
	claims := &model.AuthClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodEd25519); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.publicKey, nil
	})
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}
	if !token.Valid {
		return nil, fmt.Errorf("token is not valid")
	}
	return claims, nil
}
