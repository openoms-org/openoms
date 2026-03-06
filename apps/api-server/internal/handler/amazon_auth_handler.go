package handler

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	amazonsdk "github.com/openoms-org/openoms/packages/amazon-sp-sdk"

	"github.com/openoms-org/openoms/apps/api-server/internal/config"
	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

type AmazonAuthHandler struct {
	cfg                *config.Config
	integrationService *service.IntegrationService
	encryptionKey      []byte
	stateStore         OAuthStateStore
}

func NewAmazonAuthHandler(cfg *config.Config, integrationService *service.IntegrationService, encryptionKey []byte, stateStore OAuthStateStore) *AmazonAuthHandler {
	return &AmazonAuthHandler{
		cfg:                cfg,
		integrationService: integrationService,
		encryptionKey:      encryptionKey,
		stateStore:         stateStore,
	}
}

func (h *AmazonAuthHandler) redirectURI() string {
	return h.cfg.FrontendURL + "/marketplaces/amazon"
}

// Setup accepts Amazon SP-API credentials and verifies them by making a test API call.
// POST /v1/integrations/amazon/setup
func (h *AmazonAuthHandler) Setup(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ClientID      string `json:"client_id"`
		ClientSecret  string `json:"client_secret"`
		RefreshToken  string `json:"refresh_token"`
		MarketplaceID string `json:"marketplace_id"`
		Sandbox       bool   `json:"sandbox,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.ClientID == "" || body.ClientSecret == "" || body.RefreshToken == "" || body.MarketplaceID == "" {
		writeError(w, http.StatusBadRequest, "client_id, client_secret, refresh_token, and marketplace_id are required")
		return
	}

	// Verify credentials by making a test API call
	var opts []amazonsdk.Option
	opts = append(opts, amazonsdk.WithRefreshToken(body.RefreshToken))
	if body.Sandbox {
		opts = append(opts, amazonsdk.WithSandbox())
	}

	client := amazonsdk.NewClient(body.ClientID, body.ClientSecret, opts...)

	// Test the credentials by listing recent orders
	_, err := client.Orders.List(r.Context(), "", []string{body.MarketplaceID}, "")
	if err != nil {
		slog.Error("failed to verify Amazon credentials", "error", err)
		writeError(w, http.StatusUnprocessableEntity, "failed to verify Amazon credentials")
		return
	}

	// Store credentials
	credentials := map[string]any{
		"client_id":      body.ClientID,
		"client_secret":  body.ClientSecret,
		"refresh_token":  body.RefreshToken,
		"marketplace_id": body.MarketplaceID,
		"sandbox":        body.Sandbox,
	}
	credJSON, err := json.Marshal(credentials)
	if err != nil {
		writeServerError(w, "failed to encode credentials", err)
		return
	}

	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())
	ip := clientIP(r)

	label := "Amazon"
	req := model.CreateIntegrationRequest{
		Provider:    "amazon",
		Label:       &label,
		Credentials: credJSON,
	}

	result, err := h.integrationService.Create(r.Context(), tenantID, req, actorID, ip)
	if err != nil {
		if errors.Is(err, service.ErrDuplicateProvider) {
			// Update existing integration with new credentials
			integrations, listErr := h.integrationService.List(r.Context(), tenantID)
			if listErr != nil {
				writeServerError(w, "failed to update existing integration", listErr)
				return
			}
			for _, integ := range integrations {
				if integ.Provider == "amazon" {
					rawCreds := json.RawMessage(credJSON)
					activeStatus := "active"
					updateReq := model.UpdateIntegrationRequest{
						Credentials: &rawCreds,
						Status:      &activeStatus,
					}
					updated, updateErr := h.integrationService.Update(r.Context(), tenantID, integ.ID, updateReq, actorID, ip)
					if updateErr != nil {
						writeServerError(w, "failed to update existing integration", updateErr)
						return
					}
					writeJSON(w, http.StatusOK, updated)
					return
				}
			}
			writeServerError(w, "failed to find existing integration", errors.New("amazon integration not found"))
			return
		}
		writeServerError(w, "failed to create integration", err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

// GetAuthURL generates an Amazon Seller Central OAuth2 authorization URL.
// Credentials (client_id, client_secret, application_id, marketplace_id) are read from the existing integration.
func (h *AmazonAuthHandler) GetAuthURL(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	credJSON, _, err := h.integrationService.GetDecryptedCredentialsByProvider(r.Context(), tenantID, "amazon")
	if err != nil {
		slog.Error("amazon OAuth: failed to get credentials", "error", err)
		writeError(w, http.StatusBadRequest, "Najpierw zapisz dane integracji Amazon (Application ID, Client ID i Client Secret)")
		return
	}

	var creds struct {
		ClientID      string `json:"client_id"`
		ClientSecret  string `json:"client_secret"`
		ApplicationID string `json:"application_id"`
		MarketplaceID string `json:"marketplace_id"`
		Sandbox       bool   `json:"sandbox"`
	}
	if err := json.Unmarshal(credJSON, &creds); err != nil || creds.ClientID == "" || creds.ClientSecret == "" {
		slog.Error("amazon OAuth: credential unmarshal failed", "error", err, "json_length", len(credJSON))
		writeError(w, http.StatusBadRequest, "Integracja Amazon nie ma poprawnych danych Client ID / Client Secret")
		return
	}
	if creds.ApplicationID == "" {
		writeError(w, http.StatusBadRequest, "Application ID jest wymagane do autoryzacji OAuth")
		return
	}
	if creds.MarketplaceID == "" {
		writeError(w, http.StatusBadRequest, "Marketplace ID jest wymagane do autoryzacji OAuth")
		return
	}

	stateBytes := make([]byte, 16)
	if _, err := rand.Read(stateBytes); err != nil {
		writeServerError(w, "failed to generate state", err)
		return
	}
	state := hex.EncodeToString(stateBytes)

	stateData := &OAuthState{
		ExpiresAt:     time.Now().Add(10 * time.Minute),
		ClientID:      creds.ClientID,
		ClientSecret:  creds.ClientSecret,
		Sandbox:       creds.Sandbox,
		ApplicationID: creds.ApplicationID,
		MarketplaceID: creds.MarketplaceID,
	}
	if err := h.stateStore.Save(r.Context(), state, stateData, 10*time.Minute); err != nil {
		writeServerError(w, "failed to store OAuth state", err)
		return
	}

	client := amazonsdk.NewClient(creds.ClientID, creds.ClientSecret,
		amazonsdk.WithRedirectURI(h.redirectURI()),
	)
	authURL, err := client.AuthorizationURL(creds.ApplicationID, state, creds.MarketplaceID)
	if err != nil {
		slog.Error("amazon OAuth: failed to build auth URL", "error", err)
		writeError(w, http.StatusBadRequest, "Nie udalo sie wygenerowac adresu autoryzacji dla wybranego marketplace")
		return
	}

	slog.Info("amazon OAuth: generated auth URL",
		"redirect_uri", h.redirectURI(),
		"marketplace_id", creds.MarketplaceID,
		"client_id_prefix", creds.ClientID[:min(8, len(creds.ClientID))]+"...",
	)

	writeJSON(w, http.StatusOK, map[string]string{
		"auth_url":     authURL,
		"state":        state,
		"redirect_uri": h.redirectURI(),
	})
}

// UpdateCredentials updates only the app-level credential fields (client_id, client_secret, application_id, marketplace_id)
// while preserving existing OAuth tokens. After updating app credentials, OAuth re-authorization is required.
func (h *AmazonAuthHandler) UpdateCredentials(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ApplicationID string `json:"application_id"`
		ClientID      string `json:"client_id"`
		ClientSecret  string `json:"client_secret"`
		MarketplaceID string `json:"marketplace_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.ClientID == "" || body.ClientSecret == "" {
		writeError(w, http.StatusBadRequest, "client_id and client_secret are required")
		return
	}

	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())
	ip := clientIP(r)

	// Read existing credentials to preserve OAuth tokens
	existingJSON, integ, err := h.integrationService.GetDecryptedCredentialsByProvider(r.Context(), tenantID, "amazon")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Nie znaleziono integracji Amazon")
		return
	}

	var existing map[string]any
	if err := json.Unmarshal(existingJSON, &existing); err != nil {
		writeServerError(w, "failed to parse existing credentials", err)
		return
	}

	// Update only app-level fields
	existing["client_id"] = body.ClientID
	existing["client_secret"] = body.ClientSecret
	if body.ApplicationID != "" {
		existing["application_id"] = body.ApplicationID
	}
	if body.MarketplaceID != "" {
		existing["marketplace_id"] = body.MarketplaceID
	}

	// Clear OAuth tokens since app credentials changed — re-auth required
	delete(existing, "access_token")
	delete(existing, "token_expiry")

	mergedJSON, err := json.Marshal(existing)
	if err != nil {
		writeServerError(w, "failed to encode credentials", err)
		return
	}

	rawCreds := json.RawMessage(mergedJSON)
	pendingStatus := "pending"
	updateReq := model.UpdateIntegrationRequest{
		Credentials: &rawCreds,
		Status:      &pendingStatus,
	}
	updated, err := h.integrationService.Update(r.Context(), tenantID, integ.ID, updateReq, actorID, ip)
	if err != nil {
		writeServerError(w, "failed to update credentials", err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

// HandleCallback exchanges an Amazon SP-API OAuth2 authorization code for tokens and updates the integration.
func (h *AmazonAuthHandler) HandleCallback(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Code             string `json:"spapi_oauth_code"`
		State            string `json:"state"`
		SellingPartnerID string `json:"selling_partner_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.Code == "" {
		writeError(w, http.StatusBadRequest, "spapi_oauth_code is required")
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

	client := amazonsdk.NewClient(oauthState.ClientID, oauthState.ClientSecret,
		amazonsdk.WithRedirectURI(h.redirectURI()),
	)
	tok, err := client.ExchangeCode(r.Context(), body.Code)
	if err != nil {
		slog.Error("amazon OAuth: code exchange failed", "error", err)
		writeError(w, http.StatusUnprocessableEntity, "Nie udalo sie wymienic kodu autoryzacji na tokeny")
		return
	}

	tokenExpiry := time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second)

	credentials := map[string]any{
		"client_id":          oauthState.ClientID,
		"client_secret":      oauthState.ClientSecret,
		"application_id":     oauthState.ApplicationID,
		"access_token":       tok.AccessToken,
		"refresh_token":      tok.RefreshToken,
		"token_expiry":       tokenExpiry.Format(time.RFC3339),
		"marketplace_id":     oauthState.MarketplaceID,
		"selling_partner_id": body.SellingPartnerID,
		"sandbox":            oauthState.Sandbox,
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
		if integ.Provider == "amazon" {
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
	label := "Amazon"
	req := model.CreateIntegrationRequest{
		Provider:    "amazon",
		Label:       &label,
		Credentials: credJSON,
	}
	result, err := h.integrationService.Create(r.Context(), tenantID, req, actorID, ip)
	if err != nil {
		if errors.Is(err, service.ErrDuplicateProvider) {
			writeError(w, http.StatusConflict, "Integracja Amazon juz istnieje")
			return
		}
		writeServerError(w, "failed to create integration", err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}
