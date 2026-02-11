package middleware

import (
	"context"
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

// JWTAuth validates the Authorization: Bearer <token> header and sets
// claims, tenant ID, and user ID in the request context.
func JWTAuth(validator TokenValidator) func(http.Handler) http.Handler {
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
			if err != nil {
				writeAuthError(w, "invalid or expired token")
				return
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

			// Set all values in context
			ctx := r.Context()
			ctx = context.WithValue(ctx, ClaimsKey, claims)
			ctx = context.WithValue(ctx, TenantIDKey, claims.TenantID)
			ctx = context.WithValue(ctx, UserIDKey, userID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func writeAuthError(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte(`{"error":"` + message + `"}`))
}
