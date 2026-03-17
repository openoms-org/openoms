package handler

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/openoms-org/openoms/apps/api-server/internal/asyncutil"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
	inpostsdk "github.com/openoms-org/openoms/packages/inpost-go-sdk"
)

// InPostWebhookHandler handles incoming InPost webhook events.
type InPostWebhookHandler struct {
	webhookSecret   string
	shipmentService *service.ShipmentService
}

// NewInPostWebhookHandler creates a new InPostWebhookHandler.
func NewInPostWebhookHandler(webhookSecret string, shipmentSvc *service.ShipmentService) *InPostWebhookHandler {
	return &InPostWebhookHandler{
		webhookSecret:   webhookSecret,
		shipmentService: shipmentSvc,
	}
}

// HandleWebhook processes incoming InPost webhook requests.
// POST /v1/webhooks/inpost
// This endpoint is public (no JWT auth) but verifies HMAC-SHA256 signature.
func (h *InPostWebhookHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB limit
	body, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("inpost webhook: failed to read body", "error", err)
		w.WriteHeader(http.StatusOK) // Always return 200 to InPost
		return
	}
	defer func() { _ = r.Body.Close() }()

	// Reject requests if webhook secret is not configured
	if h.webhookSecret == "" {
		slog.Warn("inpost webhook: webhook secret not configured, rejecting request")
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}

	// Verify HMAC-SHA256 signature
	signature := r.Header.Get("X-InPost-Signature")
	if signature == "" {
		slog.Warn("inpost webhook: missing signature header", "source_ip", r.RemoteAddr, "provider", "inpost")
		w.WriteHeader(http.StatusOK)
		return
	}

	if err := inpostsdk.VerifyWebhook(h.webhookSecret, signature, body); err != nil {
		slog.Warn("inpost webhook: invalid signature", "error", err, "source_ip", r.RemoteAddr, "provider", "inpost")
		w.WriteHeader(http.StatusOK)
		return
	}

	// Parse the webhook event
	event, err := inpostsdk.ParseWebhookEvent(body)
	if err != nil {
		slog.Error("inpost webhook: failed to parse event", "error", err)
		w.WriteHeader(http.StatusOK)
		return
	}

	slog.Info("inpost webhook: received event", "event_type", event.Type)

	// Always return 200 OK to InPost immediately — don't block on processing.
	// InPost retries on non-200, so we ACK first and process async.
	w.WriteHeader(http.StatusOK)

	// Dispatch by event type (async — response already sent)
	switch event.Type {
	case "shipment_status_changed":
		asyncutil.SafeGo(func() { h.handleShipmentStatusChanged(event) })
	case "shipment_created":
		slog.Info("inpost webhook: shipment created")
	case "dispatch_order_status_changed":
		slog.Info("inpost webhook: dispatch order status changed")
	default:
		slog.Debug("inpost webhook: unhandled event type", "type", event.Type)
	}
}

// handleShipmentStatusChanged processes a shipment status change event from InPost.
func (h *InPostWebhookHandler) handleShipmentStatusChanged(event *inpostsdk.WebhookEvent) {
	var payload struct {
		ShipmentID     int64  `json:"shipment_id"`
		TrackingNumber string `json:"tracking_number"`
		Status         string `json:"status"`
	}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		slog.Error("inpost webhook: failed to parse shipment status payload", "error", err)
		return
	}

	omsStatus, ok := inpostsdk.MapStatus(payload.Status)
	if !ok {
		slog.Warn("inpost webhook: unknown shipment status",
			"inpost_status", payload.Status,
			"shipment_id", payload.ShipmentID,
		)
		return
	}

	slog.Info("inpost webhook: shipment status changed",
		"shipment_id", payload.ShipmentID,
		"tracking_number", payload.TrackingNumber,
		"inpost_status", payload.Status,
		"oms_status", omsStatus,
	)

	// Update shipment status in DB via shipment service.
	// Uses tracking number to find shipment (cross-tenant via service).
	if h.shipmentService != nil && payload.TrackingNumber != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := h.shipmentService.UpdateStatusByTrackingNumber(ctx, payload.TrackingNumber, "inpost", omsStatus); err != nil {
			slog.Error("inpost webhook: failed to update shipment status",
				"tracking_number", payload.TrackingNumber,
				"oms_status", omsStatus,
				"error", err,
			)
		}
	}
}
