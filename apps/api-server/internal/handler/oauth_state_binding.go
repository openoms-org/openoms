package handler

import (
	"log/slog"
	"net/http"

	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
)

func validateOAuthStateBinding(w http.ResponseWriter, r *http.Request, state *OAuthState, provider string) bool {
	tenantID := middleware.TenantIDFromContext(r.Context())
	userID := middleware.UserIDFromContext(r.Context())

	providerMatches := state.Provider == provider
	tenantMatches := state.TenantID == tenantID
	userMatches := state.UserID == userID
	if providerMatches && tenantMatches && userMatches {
		return true
	}

	slog.Warn("OAuth state binding mismatch",
		"provider", provider,
		"state_provider", state.Provider,
		"tenant_match", tenantMatches,
		"user_match", userMatches,
	)
	writeError(w, http.StatusBadRequest, "invalid or expired state parameter")
	return false
}
