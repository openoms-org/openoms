// Package middleware provides HTTP middleware for authentication, logging, and other cross-cutting concerns.
package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// TokenValidator validates JWT tokens. Implemented by service.TokenService.
// Defined as interface here to avoid importing the service package.
type TokenValidator interface {
	ValidateToken(tokenStr string) (*model.AuthClaims, error)
}

// APITokenAuthenticator resolves a hashed long-lived API token to the same
// AuthClaims a session JWT would carry. Optional: JWTAuth works without it.
type APITokenAuthenticator interface {
	AuthenticateAPIToken(ctx context.Context, rawToken string) (*model.AuthClaims, error)
}

// JWTAuth validates the Authorization: Bearer <token> header and sets
// claims, tenant ID, and user ID in the request context.
// An optional TokenBlacklist can be provided to reject revoked tokens.
func JWTAuth(validator TokenValidator, blacklists ...*TokenBlacklist) func(http.Handler) http.Handler {
	return jwtAuth(validator, nil, firstBlacklist(blacklists))
}

// JWTAuthWithAPITokens is JWTAuth plus a hashed API-token fallback when the
// Bearer value is not a valid access JWT. Same context keys and RLS tenant.
func JWTAuthWithAPITokens(validator TokenValidator, apiTokens APITokenAuthenticator, blacklists ...*TokenBlacklist) func(http.Handler) http.Handler {
	return jwtAuth(validator, apiTokens, firstBlacklist(blacklists))
}

func firstBlacklist(blacklists []*TokenBlacklist) *TokenBlacklist {
	if len(blacklists) > 0 {
		return blacklists[0]
	}
	return nil
}

func jwtAuth(validator TokenValidator, apiTokens APITokenAuthenticator, blacklist *TokenBlacklist) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				writeAuthError(w, "missing authorization header")
				return
			}

			if !strings.HasPrefix(authHeader, "Bearer ") {
				writeAuthError(w, "authorization header must start with Bearer")
				return
			}

			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
			if tokenStr == "" {
				writeAuthError(w, "empty bearer token")
				return
			}

			claims, err := validator.ValidateToken(tokenStr)
			usedAPIToken := false
			if err != nil {
				if apiTokens == nil {
					writeAuthError(w, "invalid or expired token")
					return
				}
				claims, err = apiTokens.AuthenticateAPIToken(r.Context(), tokenStr)
				if err != nil || claims == nil {
					writeAuthError(w, "invalid or expired token")
					return
				}
				usedAPIToken = true
			}

			// Reject non-access tokens (e.g. refresh tokens)
			if claims.Type != "" && claims.Type != "access" {
				writeAuthError(w, "invalid or expired token")
				return
			}

			// Parse user ID from JWT subject
			userID, err := uuid.Parse(claims.Subject)
			if err != nil {
				writeAuthError(w, "invalid user ID in token")
				return
			}

			// Check JWT revocation only after JWT validation. API tokens are
			// revoked in the database, not the JWT blacklist.
			if blacklist != nil && !usedAPIToken {
				tokenHash := hashToken(tokenStr)
				if blacklist.IsRevoked(tokenHash) {
					writeAuthError(w, "token has been revoked")
					return
				}
			}

			// Set all values in context
			ctx := r.Context()
			ctx = context.WithValue(ctx, ClaimsKey, claims)
			ctx = context.WithValue(ctx, TenantIDKey, claims.TenantID)
			ctx = context.WithValue(ctx, UserIDKey, userID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// hashToken returns a SHA-256 hex hash of the token string.
// We store hashes instead of raw tokens to avoid keeping sensitive
// material in memory.
func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// HashToken is the exported version of hashToken for use by handlers
// that need to add tokens to the blacklist.
func HashToken(token string) string {
	return hashToken(token)
}

func writeAuthError(w http.ResponseWriter, message string) {
	writeJSONError(w, http.StatusUnauthorized, message)
}
