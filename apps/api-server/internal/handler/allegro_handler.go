package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	allegrosdk "github.com/openoms-org/openoms/packages/allegro-go-sdk"

	allegroIntegration "github.com/openoms-org/openoms/apps/api-server/internal/integration/allegro"
	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// AllegroHandler handles Allegro-specific API endpoints (fulfillment, tracking, etc.).
type AllegroHandler struct {
	integrationService *service.IntegrationService
	orderService       *service.OrderService
	encryptionKey      []byte
}

// NewAllegroHandler creates a new AllegroHandler.
func NewAllegroHandler(integrationService *service.IntegrationService, orderService *service.OrderService, encryptionKey []byte) *AllegroHandler {
	return &AllegroHandler{
		integrationService: integrationService,
		orderService:       orderService,
		encryptionKey:      encryptionKey,
	}
}

// getProvider creates an Allegro provider from the integration credentials for the given tenant.
func (h *AllegroHandler) getProvider(ctx context.Context, tenantID uuid.UUID) (*allegroIntegration.Provider, error) {
	credJSON, _, err := h.integrationService.GetDecryptedCredentialsByProvider(ctx, tenantID, "allegro")
	if err != nil {
		return nil, err
	}
	provider, err := allegroIntegration.NewProvider(credJSON, nil)
	if err != nil {
		return nil, err
	}
	return provider, nil
}

// UpdateFulfillment handles POST /v1/integrations/allegro/orders/{orderId}/fulfillment.
// It updates the fulfillment status of an Allegro order.
func (h *AllegroHandler) UpdateFulfillment(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.TenantIDFromContext(ctx)
	orderIDStr := chi.URLParam(r, "orderId")

	orderID, err := uuid.Parse(orderIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid order ID")
		return
	}

	var body struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.Status == "" {
		writeError(w, http.StatusBadRequest, "status is required")
		return
	}

	// Get order from DB to find external_id
	order, err := h.orderService.Get(ctx, tenantID, orderID)
	if err != nil {
		slog.Error("allegro fulfillment: get order failed", "error", err)
		writeError(w, http.StatusNotFound, "order not found")
		return
	}
	if order.ExternalID == nil || *order.ExternalID == "" {
		writeError(w, http.StatusBadRequest, "order has no external Allegro ID")
		return
	}

	provider, err := h.getProvider(ctx, tenantID)
	if err != nil {
		slog.Error("allegro fulfillment: get provider failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to connect to Allegro")
		return
	}
	defer provider.Close()

	if err := provider.UpdateFulfillment(ctx, *order.ExternalID, body.Status); err != nil {
		slog.Error("allegro fulfillment: update failed", "order_id", orderIDStr, "error", err)
		writeError(w, http.StatusBadGateway, "failed to update fulfillment on Allegro")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// AddTracking handles POST /v1/integrations/allegro/orders/{orderId}/tracking.
// It adds a shipment with tracking info to an Allegro order.
func (h *AllegroHandler) AddTracking(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.TenantIDFromContext(ctx)
	orderIDStr := chi.URLParam(r, "orderId")

	orderID, err := uuid.Parse(orderIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid order ID")
		return
	}

	var body struct {
		CarrierID string `json:"carrier_id"`
		Waybill   string `json:"waybill"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.CarrierID == "" || body.Waybill == "" {
		writeError(w, http.StatusBadRequest, "carrier_id and waybill are required")
		return
	}

	// Get order from DB
	order, err := h.orderService.Get(ctx, tenantID, orderID)
	if err != nil {
		slog.Error("allegro tracking: get order failed", "error", err)
		writeError(w, http.StatusNotFound, "order not found")
		return
	}
	if order.ExternalID == nil || *order.ExternalID == "" {
		writeError(w, http.StatusBadRequest, "order has no external Allegro ID")
		return
	}

	provider, err := h.getProvider(ctx, tenantID)
	if err != nil {
		slog.Error("allegro tracking: get provider failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to connect to Allegro")
		return
	}
	defer provider.Close()

	if err := provider.AddTracking(ctx, *order.ExternalID, body.CarrierID, body.Waybill); err != nil {
		slog.Error("allegro tracking: add failed", "order_id", orderIDStr, "error", err)
		writeError(w, http.StatusBadGateway, "failed to add tracking on Allegro")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ListCarriers handles GET /v1/integrations/allegro/carriers.
// It returns the list of available Allegro shipping carriers.
func (h *AllegroHandler) ListCarriers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.TenantIDFromContext(ctx)

	provider, err := h.getProvider(ctx, tenantID)
	if err != nil {
		slog.Error("allegro carriers: get provider failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to connect to Allegro")
		return
	}
	defer provider.Close()

	carriers, err := provider.ListCarriers(ctx)
	if err != nil {
		slog.Error("allegro carriers: list failed", "error", err)
		writeError(w, http.StatusBadGateway, "failed to list carriers from Allegro")
		return
	}

	writeJSON(w, http.StatusOK, map[string][]allegrosdk.Carrier{"carriers": carriers})
}

// SyncOrders handles POST /v1/integrations/allegro/sync.
// It triggers an immediate order poll from Allegro.
func (h *AllegroHandler) SyncOrders(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.TenantIDFromContext(ctx)

	provider, err := h.getProvider(ctx, tenantID)
	if err != nil {
		slog.Error("allegro sync: get provider failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to connect to Allegro")
		return
	}
	defer provider.Close()

	orders, cursor, err := provider.PollOrders(ctx, "")
	if err != nil {
		slog.Error("allegro sync: poll failed", "error", err)
		writeError(w, http.StatusBadGateway, "failed to poll orders from Allegro")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"synced_count": len(orders),
		"cursor":       cursor,
	})
}
