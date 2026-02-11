package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	amazonsdk "github.com/openoms-org/openoms/packages/amazon-sp-sdk"

	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

type AmazonAuthHandler struct {
	integrationService *service.IntegrationService
	encryptionKey      []byte
}

func NewAmazonAuthHandler(integrationService *service.IntegrationService, encryptionKey []byte) *AmazonAuthHandler {
	return &AmazonAuthHandler{
		integrationService: integrationService,
		encryptionKey:      encryptionKey,
	}
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
		writeError(w, http.StatusBadGateway, "failed to verify Amazon credentials")
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
		writeError(w, http.StatusInternalServerError, "failed to encode credentials")
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
				writeError(w, http.StatusInternalServerError, "failed to update existing integration")
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
