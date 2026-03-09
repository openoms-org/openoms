package handler

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	ebaysdk "github.com/openoms-org/openoms/packages/ebay-go-sdk"

	"github.com/openoms-org/openoms/apps/api-server/internal/config"
	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// EbayAuthHandler handles the eBay OAuth2 authorization flow.
type EbayAuthHandler struct {
	cfg                *config.Config
	integrationService *service.IntegrationService
	stateStore         OAuthStateStore
}

// NewEbayAuthHandler creates a new EbayAuthHandler with the given dependencies.
func NewEbayAuthHandler(cfg *config.Config, integrationService *service.IntegrationService, stateStore OAuthStateStore) *EbayAuthHandler {
	return &EbayAuthHandler{
		cfg:                cfg,
		integrationService: integrationService,
		stateStore:         stateStore,
	}
}

func (h *EbayAuthHandler) redirectURI() string {
	return h.cfg.FrontendURL + "/marketplaces/ebay"
}

// GetAuthURL generates an eBay OAuth2 authorization URL.
// Credentials (app_id, cert_id, dev_id, sandbox) are read from the existing integration.
func (h *EbayAuthHandler) GetAuthURL(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	credJSON, _, err := h.integrationService.GetDecryptedCredentialsByProvider(r.Context(), tenantID, "ebay")
	if err != nil {
		slog.Error("ebay OAuth: failed to get credentials", "error", err)
		writeError(w, http.StatusBadRequest, "Najpierw zapisz dane integracji eBay (App ID i Cert ID)")
		return
	}

	var creds struct {
		AppID   string `json:"app_id"`
		CertID  string `json:"cert_id"`
		DevID   string `json:"dev_id"`
		Sandbox bool   `json:"sandbox"`
	}
	if err := json.Unmarshal(credJSON, &creds); err != nil || creds.AppID == "" || creds.CertID == "" {
		slog.Error("ebay OAuth: credential unmarshal failed", "error", err, "json_length", len(credJSON))
		writeError(w, http.StatusBadRequest, "Integracja eBay nie ma poprawnych danych App ID / Cert ID")
		return
	}

	stateBytes := make([]byte, 16)
	if _, err := rand.Read(stateBytes); err != nil {
		writeServerError(w, "failed to generate state", err)
		return
	}
	state := hex.EncodeToString(stateBytes)

	// Store state + credentials for the callback
	stateData := &OAuthState{
		ExpiresAt:    time.Now().Add(10 * time.Minute),
		ClientID:     creds.AppID,
		ClientSecret: creds.CertID,
		DevID:        creds.DevID,
		Sandbox:      creds.Sandbox,
	}
	if err := h.stateStore.Save(r.Context(), state, stateData, 10*time.Minute); err != nil {
		writeServerError(w, "failed to store OAuth state", err)
		return
	}

	opts := []ebaysdk.Option{ebaysdk.WithRedirectURI(h.redirectURI())}
	if creds.Sandbox {
		opts = append(opts, ebaysdk.WithSandbox())
	}
	client := ebaysdk.NewClient(creds.AppID, creds.CertID, creds.DevID, "", opts...)

	authURL := client.AuthorizationURL(state)

	slog.Info("ebay OAuth: generated auth URL",
		"auth_url", authURL,
		"redirect_uri", h.redirectURI(),
		"sandbox", creds.Sandbox,
		"app_id_prefix", creds.AppID[:min(8, len(creds.AppID))]+"...",
	)

	writeJSON(w, http.StatusOK, map[string]string{
		"auth_url":     authURL,
		"state":        state,
		"redirect_uri": h.redirectURI(),
	})
}

// HandleCallback exchanges an eBay OAuth2 authorization code for tokens and updates the integration.
func (h *EbayAuthHandler) HandleCallback(w http.ResponseWriter, r *http.Request) {
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

	// Validate state and retrieve stored credentials (atomic load + delete)
	oauthState, err := h.stateStore.Load(r.Context(), body.State)
	if err != nil {
		writeServerError(w, "failed to validate OAuth state", err)
		return
	}
	if oauthState == nil {
		writeError(w, http.StatusBadRequest, "invalid or expired state parameter")
		return
	}

	opts := []ebaysdk.Option{ebaysdk.WithRedirectURI(h.redirectURI())}
	if oauthState.Sandbox {
		opts = append(opts, ebaysdk.WithSandbox())
	}
	client := ebaysdk.NewClient(oauthState.ClientID, oauthState.ClientSecret, oauthState.DevID, "", opts...)

	tok, err := client.ExchangeCode(r.Context(), body.Code)
	if err != nil {
		slog.Error("ebay OAuth: code exchange failed", "error", err)
		writeError(w, http.StatusUnprocessableEntity, "Nie udało się wymienić kodu autoryzacji na tokeny")
		return
	}

	tokenExpiry := time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second)

	credentials := map[string]any{
		"app_id":        oauthState.ClientID,
		"cert_id":       oauthState.ClientSecret,
		"dev_id":        oauthState.DevID,
		"access_token":  tok.AccessToken,
		"refresh_token": tok.RefreshToken,
		"token_expiry":  tokenExpiry.Format(time.RFC3339),
		"sandbox":       oauthState.Sandbox,
	}
	credJSON, err := json.Marshal(credentials)
	if err != nil {
		writeServerError(w, "failed to encode credentials", err)
		return
	}

	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())
	ip := clientIP(r)

	// Update existing ebay integration with OAuth tokens
	integrations, listErr := h.integrationService.List(r.Context(), tenantID)
	if listErr != nil {
		writeServerError(w, "failed to find integration", listErr)
		return
	}
	for _, integ := range integrations {
		if integ.Provider == "ebay" {
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

	// Fallback: create if somehow doesn't exist (shouldn't happen in normal flow)
	label := "eBay"
	req := model.CreateIntegrationRequest{
		Provider:    "ebay",
		Label:       &label,
		Credentials: credJSON,
	}
	result, err := h.integrationService.Create(r.Context(), tenantID, req, actorID, ip)
	if err != nil {
		if errors.Is(err, service.ErrDuplicateProvider) {
			writeError(w, http.StatusConflict, "Integracja eBay już istnieje")
			return
		}
		writeServerError(w, "failed to create integration", err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}
