package handler

import (
	"io"
	"log/slog"
	"net/http"

	allegrosdk "github.com/openoms-org/openoms/packages/allegro-go-sdk"
)

// AllegroWebhookHandler handles incoming Allegro webhook events.
type AllegroWebhookHandler struct {
	webhookSecret string
}

// NewAllegroWebhookHandler creates a new AllegroWebhookHandler.
func NewAllegroWebhookHandler(webhookSecret string) *AllegroWebhookHandler {
	return &AllegroWebhookHandler{
		webhookSecret: webhookSecret,
	}
}

// HandleWebhook processes incoming Allegro webhook requests.
// POST /v1/webhooks/allegro
// This endpoint is public (no JWT auth) but verifies HMAC-SHA256 signature.
func (h *AllegroWebhookHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("allegro webhook: failed to read body", "error", err)
		w.WriteHeader(http.StatusOK) // Always return 200 to Allegro
		return
	}
	defer r.Body.Close()

	// Verify signature if webhook secret is configured
	if h.webhookSecret != "" {
		signature := r.Header.Get("X-Allegro-Signature")
		if signature == "" {
			slog.Warn("allegro webhook: missing signature header")
			w.WriteHeader(http.StatusOK)
			return
		}

		if err := allegrosdk.VerifyWebhook(h.webhookSecret, signature, body); err != nil {
			slog.Warn("allegro webhook: invalid signature", "error", err)
			w.WriteHeader(http.StatusOK)
			return
		}
	}

	// Parse the webhook event
	event, err := allegrosdk.ParseWebhookEvent(body)
	if err != nil {
		slog.Error("allegro webhook: failed to parse event", "error", err)
		w.WriteHeader(http.StatusOK)
		return
	}

	// Log the event for now; full processing (order status updates, etc.) can be added later
	slog.Info("allegro webhook: received event",
		"event_type", event.Type,
		"event_id", event.ID,
		"occurred_at", event.OccurredAt,
	)

	// Dispatch by event type
	switch event.Type {
	case "ORDER_STATUS_CHANGED":
		slog.Info("allegro webhook: order status changed", "event_id", event.ID)
	case "ORDER_FILLED_IN":
		slog.Info("allegro webhook: order filled in", "event_id", event.ID)
	default:
		slog.Debug("allegro webhook: unhandled event type", "type", event.Type)
	}

	// Always return 200 OK to Allegro so it doesn't retry
	w.WriteHeader(http.StatusOK)
}
