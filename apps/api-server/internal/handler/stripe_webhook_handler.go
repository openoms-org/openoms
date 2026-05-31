package handler

import (
	"errors"
	"io"
	"net/http"

	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// StripeWebhookHandler handles Stripe webhook events.
type StripeWebhookHandler struct {
	webhookSvc *service.StripeWebhookService
}

// NewStripeWebhookHandler creates a new StripeWebhookHandler.
func NewStripeWebhookHandler(webhookSvc *service.StripeWebhookService) *StripeWebhookHandler {
	return &StripeWebhookHandler{webhookSvc: webhookSvc}
}

// HandleWebhook processes incoming Stripe webhook events.
// POST /v1/webhooks/stripe
func (h *StripeWebhookHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	// Read raw body for signature verification
	payload, err := io.ReadAll(io.LimitReader(r.Body, 1<<16)) // 64KB max
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read request body")
		return
	}

	sigHeader := r.Header.Get("Stripe-Signature")
	if sigHeader == "" {
		writeError(w, http.StatusBadRequest, "missing Stripe-Signature header")
		return
	}

	if err := h.webhookSvc.HandleEvent(r.Context(), payload, sigHeader); err != nil {
		// Signature verification failures are client errors (400) — don't let Stripe retry them
		if errors.Is(err, service.ErrWebhookSignature) {
			writeError(w, http.StatusBadRequest, "invalid webhook signature")
			return
		}
		// Permanent processing failures (e.g. malformed payload) are also client
		// errors (400) so Stripe stops retrying them. Transient failures fall
		// through to 500, which Stripe retries.
		if errors.Is(err, service.ErrWebhookUnprocessable) {
			writeError(w, http.StatusBadRequest, "unprocessable webhook payload")
			return
		}
		writeServerError(w, "webhook processing failed", err)
		return
	}

	w.WriteHeader(http.StatusOK)
}
