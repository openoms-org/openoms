package handler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	allegrosdk "github.com/openoms-org/openoms/packages/allegro-go-sdk"

	"github.com/openoms-org/openoms/apps/api-server/internal/config"
	allegroint "github.com/openoms-org/openoms/apps/api-server/internal/integration/allegro"
	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

type allegroOAuthStore interface {
	GetDecryptedCredentialsByProvider(ctx context.Context, tenantID uuid.UUID, provider string) ([]byte, *model.Integration, error)
	List(ctx context.Context, tenantID uuid.UUID) ([]model.Integration, error)
	Update(ctx context.Context, tenantID, integrationID uuid.UUID, req model.UpdateIntegrationRequest, actorID uuid.UUID, ip string) (*model.Integration, error)
	Create(ctx context.Context, tenantID uuid.UUID, req model.CreateIntegrationRequest, actorID uuid.UUID, ip string) (*model.Integration, error)
}

type allegroCodeExchanger func(ctx context.Context, clientID, clientSecret string, sandbox bool, redirectURI, code string) (*allegrosdk.TokenResponse, error)

// AllegroAuthHandler handles the Allegro OAuth2 authorization flow.
type AllegroAuthHandler struct {
	cfg                *config.Config
	integrationService allegroOAuthStore
	encryptionKey      []byte
	stateStore         OAuthStateStore
	oauthStore         allegroOAuthStore
	exchangeCode       allegroCodeExchanger
}

// NewAllegroAuthHandler creates a new AllegroAuthHandler with the given dependencies.
func NewAllegroAuthHandler(cfg *config.Config, integrationService *service.IntegrationService, encryptionKey []byte, stateStore OAuthStateStore) *AllegroAuthHandler {
	h := &AllegroAuthHandler{
		cfg:                cfg,
		integrationService: integrationService,
		encryptionKey:      encryptionKey,
		stateStore:         stateStore,
		exchangeCode:       exchangeAllegroAuthorizationCode,
	}
	if integrationService != nil {
		h.oauthStore = integrationService
	}
	return h
}

func (h *AllegroAuthHandler) store() allegroOAuthStore {
	if h.oauthStore != nil {
		return h.oauthStore
	}
	return h.integrationService
}

func exchangeAllegroAuthorizationCode(ctx context.Context, clientID, clientSecret string, sandbox bool, redirectURI, code string) (*allegrosdk.TokenResponse, error) {
	opts := []allegrosdk.Option{allegrosdk.WithRedirectURI(redirectURI)}
	if sandbox {
		opts = append(opts, allegrosdk.WithSandbox())
	}
	client := allegrosdk.NewClient(clientID, clientSecret, opts...)
	defer client.Close()
	return client.ExchangeCode(ctx, code)
}

// redirectURI computes the OAuth redirect URI from the frontend URL.
func (h *AllegroAuthHandler) redirectURI() string {
	return h.cfg.FrontendURL + "/marketplaces/allegro"
}

// GetAuthURL generates an Allegro OAuth2 authorization URL.
// Credentials (client_id, client_secret, sandbox) are read from the existing integration.
func (h *AllegroAuthHandler) GetAuthURL(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	// Read credentials from existing integration
	credJSON, _, err := h.store().GetDecryptedCredentialsByProvider(r.Context(), tenantID, "allegro")
	if err != nil {
		slog.Error("allegro OAuth: failed to get credentials", "error", err)
		writeError(w, http.StatusBadRequest, "Najpierw zapisz dane integracji Allegro (Client ID i Client Secret)")
		return
	}

	var creds struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
		Sandbox      bool   `json:"sandbox"`
	}
	if err := json.Unmarshal(credJSON, &creds); err != nil || creds.ClientID == "" || creds.ClientSecret == "" {
		slog.Error("allegro OAuth: credential unmarshal failed", "error", err, "json_length", len(credJSON))
		writeError(w, http.StatusBadRequest, "Integracja Allegro nie ma poprawnych danych Client ID / Client Secret")
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
		TenantID:     tenantID,
		UserID:       actorID,
		Provider:     "allegro",
		ClientID:     creds.ClientID,
		ClientSecret: creds.ClientSecret,
		Sandbox:      creds.Sandbox,
	}
	if err := h.stateStore.Save(r.Context(), state, stateData, 10*time.Minute); err != nil {
		writeServerError(w, "failed to store OAuth state", err)
		return
	}

	opts := []allegrosdk.Option{allegrosdk.WithRedirectURI(h.redirectURI())}
	if creds.Sandbox {
		opts = append(opts, allegrosdk.WithSandbox())
	}
	client := allegrosdk.NewClient(creds.ClientID, creds.ClientSecret, opts...)
	defer client.Close()

	authURL := client.AuthorizationURL(state)

	slog.Info("allegro OAuth: generated auth URL",
		"auth_url", authURL,
		"redirect_uri", h.redirectURI(),
		"sandbox", creds.Sandbox,
		"client_id_prefix", creds.ClientID[:min(8, len(creds.ClientID))]+"...",
	)

	writeJSON(w, http.StatusOK, map[string]string{
		"auth_url":     authURL,
		"state":        state,
		"redirect_uri": h.redirectURI(),
	})
}

// HandleCallback exchanges an OAuth2 authorization code for tokens and updates the integration.
func (h *AllegroAuthHandler) HandleCallback(w http.ResponseWriter, r *http.Request) {
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
	if !validateOAuthStateBinding(w, r, oauthState, "allegro") {
		return
	}

	exchange := h.exchangeCode
	if exchange == nil {
		exchange = exchangeAllegroAuthorizationCode
	}
	tok, err := exchange(r.Context(), oauthState.ClientID, oauthState.ClientSecret, oauthState.Sandbox, h.redirectURI(), body.Code)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "failed to exchange authorization code")
		return
	}

	tokenExpiry := time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second)
	credJSON, err := allegroint.ReconnectCredentialJSON(
		oauthState.ClientID, oauthState.ClientSecret, tok.AccessToken, tok.RefreshToken, tokenExpiry, oauthState.Sandbox,
	)
	if err != nil {
		writeServerError(w, "failed to encode credentials", err)
		return
	}

	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())
	ip := clientIP(r)

	// Persist new tokens before returning success. Do not report reconnect done
	// until this write commits.
	integrations, listErr := h.store().List(r.Context(), tenantID)
	if listErr != nil {
		writeServerError(w, "failed to find integration", listErr)
		return
	}
	for _, integ := range integrations {
		if integ.Provider == "allegro" {
			rawCreds := json.RawMessage(credJSON)
			activeStatus := "active"
			updateReq := model.UpdateIntegrationRequest{
				Credentials: &rawCreds,
				Status:      &activeStatus,
			}
			updated, updateErr := h.store().Update(r.Context(), tenantID, integ.ID, updateReq, actorID, ip)
			if updateErr != nil {
				writeServerError(w, "failed to update integration", updateErr)
				return
			}
			writeJSON(w, http.StatusOK, updated)
			return
		}
	}

	// Fallback: create if somehow doesn't exist (shouldn't happen in normal flow)
	label := "Allegro"
	req := model.CreateIntegrationRequest{
		Provider:    "allegro",
		Label:       &label,
		Credentials: credJSON,
	}
	result, err := h.store().Create(r.Context(), tenantID, req, actorID, ip)
	if err != nil {
		if errors.Is(err, service.ErrDuplicateProvider) {
			writeError(w, http.StatusConflict, "allegro integration already exists")
			return
		}
		writeServerError(w, "failed to create integration", err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}
