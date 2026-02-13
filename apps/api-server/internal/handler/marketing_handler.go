package handler

import (
	"encoding/json"
	"net/http"

	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

type MarketingHandler struct {
	mailchimpService *service.MailchimpService
}

func NewMarketingHandler(mailchimpService *service.MailchimpService) *MarketingHandler {
	return &MarketingHandler{mailchimpService: mailchimpService}
}

// Sync handles POST /v1/marketing/sync — triggers customer sync to Mailchimp.
func (h *MarketingHandler) Sync(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	synced, failed, err := h.mailchimpService.SyncAllCustomers(r.Context(), tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"synced": synced,
		"failed": failed,
	})
}

// Status handles GET /v1/marketing/status — returns Mailchimp integration status.
func (h *MarketingHandler) Status(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	settings, err := h.mailchimpService.GetSettings(r.Context(), tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load mailchimp settings")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"enabled":    settings.Enabled,
		"configured": settings.APIKey != "" && settings.ListID != "",
	})
}

// CreateCampaign handles POST /v1/marketing/campaigns — creates a Mailchimp campaign.
func (h *MarketingHandler) CreateCampaign(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	var req struct {
		Name    string `json:"name"`
		Subject string `json:"subject"`
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" || req.Subject == "" || req.Content == "" {
		writeError(w, http.StatusBadRequest, "name, subject and content are required")
		return
	}

	campaignID, err := h.mailchimpService.CreateCampaign(r.Context(), tenantID, req.Name, req.Subject, req.Content)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"campaign_id": campaignID,
	})
}
