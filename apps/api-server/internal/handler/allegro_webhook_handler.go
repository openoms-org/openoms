package handler

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/openoms-org/openoms/apps/api-server/internal/asyncutil"
	allegrosdk "github.com/openoms-org/openoms/packages/allegro-go-sdk"
)

// AllegroOrderSyncer defines the interface for webhook-triggered order import/update.
// Implemented by worker.AllegroWebhookSyncer.
type AllegroOrderSyncer interface {
	ImportOrder(ctx context.Context, allegroOrderID string)
	UpdateOrderStatus(ctx context.Context, allegroOrderID string)
	ImportOrderForIntegration(ctx context.Context, tenantID, integrationID uuid.UUID, allegroOrderID string)
	UpdateOrderStatusForIntegration(ctx context.Context, tenantID, integrationID uuid.UUID, allegroOrderID string)
}

// AllegroWebhookHandler handles incoming Allegro webhook events.
type AllegroWebhookHandler struct {
	webhookSecret  string
	orderSyncer    AllegroOrderSyncer
	secretResolver providerWebhookSecretResolver
}

// NewAllegroWebhookHandler creates a new AllegroWebhookHandler.
func NewAllegroWebhookHandler(webhookSecret string, orderSyncer AllegroOrderSyncer) *AllegroWebhookHandler {
	return &AllegroWebhookHandler{
		webhookSecret: webhookSecret,
		orderSyncer:   orderSyncer,
	}
}

// SetProviderWebhookSecretResolver enables tenant-scoped webhook secret lookup.
func (h *AllegroWebhookHandler) SetProviderWebhookSecretResolver(resolver providerWebhookSecretResolver) {
	h.secretResolver = resolver
}

// HandleWebhook processes incoming Allegro webhook requests.
// POST /v1/webhooks/allegro
// This endpoint is public (no JWT auth) but verifies HMAC-SHA256 signature.
func (h *AllegroWebhookHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB limit
	body, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("allegro webhook: failed to read body", "error", err)
		w.WriteHeader(http.StatusOK) // Always return 200 to Allegro
		return
	}
	defer func() { _ = r.Body.Close() }()

	webhookSecret, scope, err := resolveProviderWebhookSecret(r.Context(), r, "allegro", h.webhookSecret, h.secretResolver)
	if err != nil {
		if errors.Is(err, errInvalidProviderWebhookIntegrationID) {
			slog.Warn("allegro webhook: invalid integration id", "source_ip", r.RemoteAddr) //nolint:gosec // G706: RemoteAddr is set by net/http, not user input
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		slog.Warn("allegro webhook: scoped secret lookup failed", "error", err, "provider", "allegro")
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}

	// Reject requests if webhook secret is not configured
	if webhookSecret == "" {
		slog.Warn("allegro webhook: webhook secret not configured, rejecting request")
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}

	// Verify HMAC-SHA256 signature
	signature := r.Header.Get("X-Allegro-Signature")
	if signature == "" {
		slog.Warn("allegro webhook: missing signature header", "source_ip", r.RemoteAddr, "provider", "allegro") //nolint:gosec // G706: RemoteAddr is set by net/http, not user input
		w.WriteHeader(http.StatusOK)
		return
	}

	if err := allegrosdk.VerifyWebhook(webhookSecret, signature, body); err != nil {
		slog.Warn("allegro webhook: invalid signature", "error", err, "source_ip", r.RemoteAddr, "provider", "allegro") //nolint:gosec // G706: RemoteAddr is set by net/http, not user input
		w.WriteHeader(http.StatusOK)
		return
	}

	// Parse the webhook event
	event, err := allegrosdk.ParseWebhookEvent(body)
	if err != nil {
		slog.Error("allegro webhook: failed to parse event", "error", err)
		w.WriteHeader(http.StatusOK)
		return
	}

	orderID := event.OrderID()

	slog.Info("allegro webhook: received event",
		"event_type", event.Type,
		"event_id", event.ID,
		"occurred_at", event.OccurredAt,
		"order_id", orderID,
	)

	// Dispatch by event type — process asynchronously after returning 200
	switch event.Type {
	case "ORDER_FILLED_IN":
		slog.Info("allegro webhook: order filled in — triggering import",
			"event_id", event.ID,
			"order_id", orderID,
		)
		if h.orderSyncer != nil && orderID != "" {
			asyncutil.SafeGo(func() {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				if scope != nil {
					h.orderSyncer.ImportOrderForIntegration(ctx, scope.TenantID, scope.IntegrationID, orderID)
					return
				}
				h.orderSyncer.ImportOrder(ctx, orderID)
			})
		}

	case "ORDER_STATUS_CHANGED":
		slog.Info("allegro webhook: order status changed — triggering status update",
			"event_id", event.ID,
			"order_id", orderID,
		)
		if h.orderSyncer != nil && orderID != "" {
			asyncutil.SafeGo(func() {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				if scope != nil {
					h.orderSyncer.UpdateOrderStatusForIntegration(ctx, scope.TenantID, scope.IntegrationID, orderID)
					return
				}
				h.orderSyncer.UpdateOrderStatus(ctx, orderID)
			})
		}

	default:
		slog.Debug("allegro webhook: unhandled event type", "type", event.Type)
	}

	// Always return 200 OK to Allegro so it doesn't retry
	w.WriteHeader(http.StatusOK)
}
