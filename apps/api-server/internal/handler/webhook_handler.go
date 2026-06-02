package handler

import (
	"errors"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// WebhookHandler handles inbound webhook requests from external providers.
type WebhookHandler struct {
	webhookService *service.WebhookService
}

// NewWebhookHandler creates a new WebhookHandler.
func NewWebhookHandler(webhookService *service.WebhookService) *WebhookHandler {
	return &WebhookHandler{webhookService: webhookService}
}

// Receive processes an inbound webhook event from a provider.
func (h *WebhookHandler) Receive(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")
	tenantIDStr := chi.URLParam(r, "tenant_id")

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid tenant_id")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB limit
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read request body")
		return
	}

	signature := r.Header.Get("X-Webhook-Signature")

	event, err := h.webhookService.Receive(r.Context(), tenantID, provider, signature, body)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUnknownProvider):
			writeError(w, http.StatusBadRequest, "unknown provider")
		case errors.Is(err, service.ErrInvalidSignature):
			writeError(w, http.StatusUnauthorized, "invalid signature")
		case errors.Is(err, service.ErrWebhookSecretNotConfigured):
			writeError(w, http.StatusUnprocessableEntity, "webhook secret not configured")
		case errors.Is(err, service.ErrUnknownTenant):
			writeError(w, http.StatusNotFound, "unknown tenant")
		default:
			writeServerError(w, "failed to process webhook", err)
		}
		return
	}

	writeJSON(w, http.StatusOK, event)
}
