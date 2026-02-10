package handler

import (
	"errors"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

type WebhookHandler struct {
	webhookService *service.WebhookService
}

func NewWebhookHandler(webhookService *service.WebhookService) *WebhookHandler {
	return &WebhookHandler{webhookService: webhookService}
}

func (h *WebhookHandler) Receive(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")
	tenantIDStr := chi.URLParam(r, "tenant_id")

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid tenant_id")
		return
	}

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
		default:
			writeError(w, http.StatusInternalServerError, "failed to process webhook")
		}
		return
	}

	writeJSON(w, http.StatusOK, event)
}
