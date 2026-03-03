package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	prestashopsdk "github.com/openoms-org/openoms/packages/prestashop-go-sdk"
	shopersdk "github.com/openoms-org/openoms/packages/shoper-go-sdk"
	shopifysdk "github.com/openoms-org/openoms/packages/shopify-go-sdk"

	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/netutil"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// StoreAuthHandler handles setup/connection endpoints for store platform integrations
// (Shoper, PrestaShop, Shopify).
type StoreAuthHandler struct {
	integrationService *service.IntegrationService
	encryptionKey      []byte
}

// NewStoreAuthHandler creates a new StoreAuthHandler.
func NewStoreAuthHandler(integrationService *service.IntegrationService, encryptionKey []byte) *StoreAuthHandler {
	return &StoreAuthHandler{
		integrationService: integrationService,
		encryptionKey:      encryptionKey,
	}
}

// SetupShoper accepts Shoper credentials and verifies them by making a test API call.
// POST /v1/integrations/shoper/setup
func (h *StoreAuthHandler) SetupShoper(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ShopURL      string `json:"shop_url"`
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.ShopURL == "" || body.ClientID == "" || body.ClientSecret == "" {
		writeError(w, http.StatusBadRequest, "shop_url, client_id and client_secret are required")
		return
	}

	// Verify credentials by making a test API call (SafeHTTPClient blocks private IPs / SSRF)
	client := shopersdk.NewClient(body.ShopURL, body.ClientID, body.ClientSecret,
		shopersdk.WithHTTPClient(netutil.SafeHTTPClient(30*time.Second)),
	)
	_, err := client.Orders.List(r.Context(), shopersdk.OrderListParams{Limit: 1})
	if err != nil {
		slog.Error("failed to verify Shoper credentials", "error", err)
		writeError(w, http.StatusUnprocessableEntity, "Failed to verify Shoper credentials")
		return
	}

	credentials := map[string]any{
		"shop_url":      body.ShopURL,
		"client_id":     body.ClientID,
		"client_secret": body.ClientSecret,
	}

	h.saveIntegration(w, r, "shoper", "Shoper", credentials)
}

// SetupPrestaShop accepts PrestaShop credentials and verifies them.
// POST /v1/integrations/prestashop/setup
func (h *StoreAuthHandler) SetupPrestaShop(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ShopURL string `json:"shop_url"`
		APIKey  string `json:"api_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.ShopURL == "" || body.APIKey == "" {
		writeError(w, http.StatusBadRequest, "shop_url and api_key are required")
		return
	}

	// Verify credentials by making a test API call (SafeHTTPClient blocks private IPs / SSRF)
	client := prestashopsdk.NewClient(body.ShopURL, body.APIKey,
		prestashopsdk.WithHTTPClient(netutil.SafeHTTPClient(30*time.Second)),
	)
	_, err := client.Orders.List(r.Context(), prestashopsdk.OrderListParams{Limit: 1})
	if err != nil {
		slog.Error("failed to verify PrestaShop credentials", "error", err)
		writeError(w, http.StatusUnprocessableEntity, "Failed to verify PrestaShop credentials")
		return
	}

	credentials := map[string]any{
		"shop_url": body.ShopURL,
		"api_key":  body.APIKey,
	}

	h.saveIntegration(w, r, "prestashop", "PrestaShop", credentials)
}

// SetupShopify accepts Shopify credentials and verifies them.
// POST /v1/integrations/shopify/setup
func (h *StoreAuthHandler) SetupShopify(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ShopDomain  string `json:"shop_domain"`
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.ShopDomain == "" || body.AccessToken == "" {
		writeError(w, http.StatusBadRequest, "shop_domain and access_token are required")
		return
	}

	// Verify credentials by making a test API call (SafeHTTPClient blocks private IPs / SSRF)
	client := shopifysdk.NewClient(body.ShopDomain, body.AccessToken,
		shopifysdk.WithHTTPClient(netutil.SafeHTTPClient(30*time.Second)),
	)
	_, err := client.Orders.List(r.Context(), shopifysdk.OrderListParams{Limit: 1, Status: "any"})
	if err != nil {
		slog.Error("failed to verify Shopify credentials", "error", err)
		writeError(w, http.StatusUnprocessableEntity, "Failed to verify Shopify credentials")
		return
	}

	credentials := map[string]any{
		"shop_domain":  body.ShopDomain,
		"access_token": body.AccessToken,
	}

	h.saveIntegration(w, r, "shopify", "Shopify", credentials)
}

// saveIntegration creates or updates an integration with the given credentials.
func (h *StoreAuthHandler) saveIntegration(w http.ResponseWriter, r *http.Request, provider, label string, credentials map[string]any) {
	credJSON, err := json.Marshal(credentials)
	if err != nil {
		writeServerError(w, "failed to encode credentials", err)
		return
	}

	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())
	ip := clientIP(r)

	req := model.CreateIntegrationRequest{
		Provider:    provider,
		Label:       &label,
		Credentials: credJSON,
	}

	result, err := h.integrationService.Create(r.Context(), tenantID, req, actorID, ip)
	if err != nil {
		if errors.Is(err, service.ErrDuplicateProvider) {
			// Update existing integration with new credentials
			integrations, listErr := h.integrationService.List(r.Context(), tenantID)
			if listErr != nil {
				writeServerError(w, "failed to update existing integration", err)
				return
			}
			for _, integ := range integrations {
				if integ.Provider == provider {
					rawCreds := json.RawMessage(credJSON)
					activeStatus := "active"
					updateReq := model.UpdateIntegrationRequest{
						Credentials: &rawCreds,
						Status:      &activeStatus,
					}
					updated, updateErr := h.integrationService.Update(r.Context(), tenantID, integ.ID, updateReq, actorID, ip)
					if updateErr != nil {
						writeServerError(w, "failed to update existing integration", err)
						return
					}
					writeJSON(w, http.StatusOK, updated)
					return
				}
			}
			writeServerError(w, "failed to find existing integration", err)
			return
		}
		writeServerError(w, "failed to create integration", err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}
