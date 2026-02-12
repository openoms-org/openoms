package handler

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"time"

	allegrosdk "github.com/openoms-org/openoms/packages/allegro-go-sdk"

	"github.com/openoms-org/openoms/apps/api-server/internal/config"
	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

type AllegroAuthHandler struct {
	cfg                *config.Config
	integrationService *service.IntegrationService
	encryptionKey      []byte
	stateMu            sync.Mutex
	stateStore         map[string]time.Time
}

func NewAllegroAuthHandler(cfg *config.Config, integrationService *service.IntegrationService, encryptionKey []byte) *AllegroAuthHandler {
	return &AllegroAuthHandler{
		cfg:                cfg,
		integrationService: integrationService,
		encryptionKey:      encryptionKey,
		stateStore:         make(map[string]time.Time),
	}
}

// GetAuthURL generates an Allegro OAuth2 authorization URL.
func (h *AllegroAuthHandler) GetAuthURL(w http.ResponseWriter, r *http.Request) {
	stateBytes := make([]byte, 16)
	if _, err := rand.Read(stateBytes); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate state")
		return
	}
	state := hex.EncodeToString(stateBytes)

	// Store the state with a 10-minute TTL for CSRF validation
	h.stateMu.Lock()
	// Cleanup expired states while we hold the lock
	now := time.Now()
	for k, exp := range h.stateStore {
		if now.After(exp) {
			delete(h.stateStore, k)
		}
	}
	h.stateStore[state] = now.Add(10 * time.Minute)
	h.stateMu.Unlock()

	opts := []allegrosdk.Option{allegrosdk.WithRedirectURI(h.cfg.AllegroRedirectURI)}
	if h.cfg.AllegroSandbox {
		opts = append(opts, allegrosdk.WithSandbox())
	}
	client := allegrosdk.NewClient(h.cfg.AllegroClientID, h.cfg.AllegroClientSecret, opts...)
	defer client.Close()

	authURL := client.AuthorizationURL(state)

	writeJSON(w, http.StatusOK, map[string]string{
		"auth_url": authURL,
		"state":    state,
	})
}

// HandleCallback exchanges an OAuth2 authorization code for tokens and creates/updates the integration.
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

	// Validate the state parameter against the server-side store (CSRF protection)
	h.stateMu.Lock()
	expiry, exists := h.stateStore[body.State]
	if exists {
		delete(h.stateStore, body.State)
	}
	h.stateMu.Unlock()

	if !exists || time.Now().After(expiry) {
		writeError(w, http.StatusBadRequest, "invalid or expired state parameter")
		return
	}

	cbOpts := []allegrosdk.Option{allegrosdk.WithRedirectURI(h.cfg.AllegroRedirectURI)}
	if h.cfg.AllegroSandbox {
		cbOpts = append(cbOpts, allegrosdk.WithSandbox())
	}
	client := allegrosdk.NewClient(h.cfg.AllegroClientID, h.cfg.AllegroClientSecret, cbOpts...)
	defer client.Close()

	tok, err := client.ExchangeCode(r.Context(), body.Code)
	if err != nil {
		writeError(w, http.StatusBadGateway, "failed to exchange authorization code")
		return
	}

	tokenExpiry := time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second)

	credentials := map[string]any{
		"client_id":     h.cfg.AllegroClientID,
		"client_secret": h.cfg.AllegroClientSecret,
		"access_token":  tok.AccessToken,
		"refresh_token": tok.RefreshToken,
		"token_expiry":  tokenExpiry.Format(time.RFC3339),
		"sandbox":       h.cfg.AllegroSandbox,
	}
	credJSON, err := json.Marshal(credentials)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to encode credentials")
		return
	}

	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())
	ip := clientIP(r)

	label := "Allegro"
	req := model.CreateIntegrationRequest{
		Provider:    "allegro",
		Label:       &label,
		Credentials: credJSON,
	}

	result, err := h.integrationService.Create(r.Context(), tenantID, req, actorID, ip)
	if err != nil {
		if errors.Is(err, service.ErrDuplicateProvider) {
			// Update existing integration with new credentials
			integrations, listErr := h.integrationService.List(r.Context(), tenantID)
			if listErr != nil {
				writeError(w, http.StatusInternalServerError, "failed to update existing integration")
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
					updated, updateErr := h.integrationService.Update(r.Context(), tenantID, integ.ID, updateReq, actorID, ip)
					if updateErr != nil {
						writeError(w, http.StatusInternalServerError, "failed to update existing integration")
						return
					}
					writeJSON(w, http.StatusOK, updated)
					return
				}
			}
			writeError(w, http.StatusInternalServerError, "failed to find existing integration")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to create integration")
		return
	}
	writeJSON(w, http.StatusCreated, result)
}
