package handler

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	olxsdk "github.com/openoms-org/openoms/packages/olx-go-sdk"

	"github.com/openoms-org/openoms/apps/api-server/internal/config"
	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// OlxAuthHandler handles the OLX OAuth2 authorization flow.
type OlxAuthHandler struct {
	cfg                *config.Config
	integrationService *service.IntegrationService
	stateStore         OAuthStateStore
}

// NewOlxAuthHandler creates a new OlxAuthHandler with the given dependencies.
func NewOlxAuthHandler(cfg *config.Config, integrationService *service.IntegrationService, stateStore OAuthStateStore) *OlxAuthHandler {
	return &OlxAuthHandler{
		cfg:                cfg,
		integrationService: integrationService,
		stateStore:         stateStore,
	}
}

func (h *OlxAuthHandler) redirectURI() string {
	return h.cfg.FrontendURL + "/marketplaces/olx"
}

// GetAuthURL generates an OLX OAuth2 authorization URL.
// Credentials (client_id, client_secret) are read from the existing integration.
func (h *OlxAuthHandler) GetAuthURL(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	credJSON, _, err := h.integrationService.GetDecryptedCredentialsByProvider(r.Context(), tenantID, "olx")
	if err != nil {
		slog.Error("olx OAuth: failed to get credentials", "error", err)
		writeError(w, http.StatusBadRequest, "Save OLX integration credentials first (Client ID and Client Secret)")
		return
	}

	var creds struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
	}
	if err := json.Unmarshal(credJSON, &creds); err != nil || creds.ClientID == "" || creds.ClientSecret == "" {
		slog.Error("olx OAuth: credential unmarshal failed", "error", err, "json_length", len(credJSON))
		writeError(w, http.StatusBadRequest, "OLX integration missing valid Client ID / Client Secret")
		return
	}

	stateBytes := make([]byte, 16)
	if _, err := rand.Read(stateBytes); err != nil {
		writeServerError(w, "failed to generate state", err)
		return
	}
	state := hex.EncodeToString(stateBytes)

	stateData := &OAuthState{
		ExpiresAt:    time.Now().Add(10 * time.Minute),
		ClientID:     creds.ClientID,
		ClientSecret: creds.ClientSecret,
	}
	if err := h.stateStore.Save(r.Context(), state, stateData, 10*time.Minute); err != nil {
		writeServerError(w, "failed to store OAuth state", err)
		return
	}

	client := olxsdk.NewClient(creds.ClientID, creds.ClientSecret, "",
		olxsdk.WithRedirectURI(h.redirectURI()),
	)
	authURL := client.AuthorizationURL(state, "read", "write", "v2")

	slog.Info("olx OAuth: generated auth URL",
		"redirect_uri", h.redirectURI(),
		"client_id_prefix", creds.ClientID[:min(8, len(creds.ClientID))]+"...",
	)

	writeJSON(w, http.StatusOK, map[string]string{
		"auth_url":     authURL,
		"state":        state,
		"redirect_uri": h.redirectURI(),
	})
}

// HandleCallback exchanges an OLX OAuth2 authorization code for tokens and updates the integration.
func (h *OlxAuthHandler) HandleCallback(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Code  string `json:"code"`
		State string `json:"state"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.Code == "" {
		writeError(w, http.StatusBadRequest, "code is required")
		return
	}
	if body.State == "" {
		writeError(w, http.StatusBadRequest, "state is required")
		return
	}

	oauthState, err := h.stateStore.Load(r.Context(), body.State)
	if err != nil {
		writeServerError(w, "failed to validate OAuth state", err)
		return
	}
	if oauthState == nil {
		writeError(w, http.StatusBadRequest, "invalid or expired state parameter")
		return
	}

	client := olxsdk.NewClient(oauthState.ClientID, oauthState.ClientSecret, "",
		olxsdk.WithRedirectURI(h.redirectURI()),
	)
	tok, err := client.ExchangeCode(r.Context(), body.Code)
	if err != nil {
		slog.Error("olx OAuth: code exchange failed", "error", err)
		writeError(w, http.StatusUnprocessableEntity, "Failed to exchange authorization code for tokens")
		return
	}

	tokenExpiry := time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second)

	credentials := map[string]any{
		"client_id":     oauthState.ClientID,
		"client_secret": oauthState.ClientSecret,
		"access_token":  tok.AccessToken,
		"refresh_token": tok.RefreshToken,
		"token_expiry":  tokenExpiry.Format(time.RFC3339),
	}
	credJSON, err := json.Marshal(credentials)
	if err != nil {
		writeServerError(w, "failed to encode credentials", err)
		return
	}

	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())
	ip := clientIP(r)

	integrations, listErr := h.integrationService.List(r.Context(), tenantID)
	if listErr != nil {
		writeServerError(w, "failed to find integration", listErr)
		return
	}
	for _, integ := range integrations {
		if integ.Provider == "olx" {
			rawCreds := json.RawMessage(credJSON)
			activeStatus := "active"
			updateReq := model.UpdateIntegrationRequest{
				Credentials: &rawCreds,
				Status:      &activeStatus,
			}
			updated, updateErr := h.integrationService.Update(r.Context(), tenantID, integ.ID, updateReq, actorID, ip)
			if updateErr != nil {
				writeServerError(w, "failed to update integration", updateErr)
				return
			}
			writeJSON(w, http.StatusOK, updated)
			return
		}
	}

	// Fallback: create if doesn't exist
	label := "OLX"
	req := model.CreateIntegrationRequest{
		Provider:    "olx",
		Label:       &label,
		Credentials: credJSON,
	}
	result, err := h.integrationService.Create(r.Context(), tenantID, req, actorID, ip)
	if err != nil {
		if errors.Is(err, service.ErrDuplicateProvider) {
			writeError(w, http.StatusConflict, "OLX integration already exists")
			return
		}
		writeServerError(w, "failed to create integration", err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}
