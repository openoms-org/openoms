package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	allegrosdk "github.com/openoms-org/openoms/packages/allegro-go-sdk"

	allegroIntegration "github.com/openoms-org/openoms/apps/api-server/internal/integration/allegro"
	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// AllegroHandler handles Allegro-specific API endpoints (fulfillment, tracking, import, etc.).
type AllegroHandler struct {
	integrationService   *service.IntegrationService
	orderService         *service.OrderService
	allegroImportService *service.AllegroImportService
	orderInbound         *service.AllegroOrderInboundService
	encryptionKey        []byte
}

// NewAllegroHandler creates a new AllegroHandler.
func NewAllegroHandler(integrationService *service.IntegrationService, orderService *service.OrderService, allegroImportService *service.AllegroImportService, encryptionKey []byte) *AllegroHandler {
	return &AllegroHandler{
		integrationService:   integrationService,
		orderService:         orderService,
		allegroImportService: allegroImportService,
		encryptionKey:        encryptionKey,
	}
}

// SetOrderInbound wires the checkout-form → OMS upsert used by POST /sync.
func (h *AllegroHandler) SetOrderInbound(inbound *service.AllegroOrderInboundService) {
	if h == nil {
		return
	}
	h.orderInbound = inbound
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
		if errors.Is(err, service.ErrOrderNotFound) {
			writeError(w, http.StatusNotFound, "order not found")
		} else {
			writeServerError(w, "failed to get order", err)
		}
		return
	}
	if order.ExternalID == nil || *order.ExternalID == "" {
		writeError(w, http.StatusBadRequest, "order has no external Allegro ID")
		return
	}

	provider, err := h.getProvider(ctx, tenantID)
	if err != nil {
		writeServerError(w, "failed to connect to Allegro", err)
		return
	}
	defer provider.Close()

	if err := provider.UpdateFulfillment(ctx, *order.ExternalID, body.Status); err != nil {
		slog.Error("allegro fulfillment: update failed", "order_id", orderIDStr, "error", err) //nolint:gosec
		writeAllegroError(w, "failed to update fulfillment on Allegro", err)
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
		if errors.Is(err, service.ErrOrderNotFound) {
			writeError(w, http.StatusNotFound, "order not found")
		} else {
			writeServerError(w, "failed to get order", err)
		}
		return
	}
	if order.ExternalID == nil || *order.ExternalID == "" {
		writeError(w, http.StatusBadRequest, "order has no external Allegro ID")
		return
	}

	provider, err := h.getProvider(ctx, tenantID)
	if err != nil {
		writeServerError(w, "failed to connect to Allegro", err)
		return
	}
	defer provider.Close()

	if err := provider.AddTracking(ctx, *order.ExternalID, body.CarrierID, body.Waybill); err != nil {
		slog.Error("allegro tracking: add failed", "order_id", orderIDStr, "error", err) //nolint:gosec
		writeAllegroError(w, "failed to add tracking on Allegro", err)
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
		writeServerError(w, "failed to connect to Allegro", err)
		return
	}
	defer provider.Close()

	carriers, err := provider.ListCarriers(ctx)
	if err != nil {
		slog.Error("allegro carriers: list failed", "error", err)
		writeAllegroError(w, "failed to list carriers from Allegro", err)
		return
	}

	writeJSON(w, http.StatusOK, map[string][]allegrosdk.Carrier{"carriers": carriers})
}

// SyncOrders handles POST /v1/integrations/allegro/sync.
// It lists checkout-forms and upserts them as source=allegro OMS orders.
func (h *AllegroHandler) SyncOrders(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.TenantIDFromContext(ctx)
	if tenantID == uuid.Nil {
		writeError(w, http.StatusUnauthorized, "missing tenant context")
		return
	}
	if h.orderInbound == nil {
		writeServerError(w, "order sync is not configured", errors.New("allegro order inbound service is nil"))
		return
	}

	result, err := h.orderInbound.SyncOrders(ctx, tenantID)
	if err != nil {
		if errors.Is(err, service.ErrIntegrationNotFound) {
			writeError(w, http.StatusNotFound, "allegro integration not found")
			return
		}
		slog.Error("allegro sync: inbound import failed", "error", err)
		writeAllegroError(w, "failed to sync orders from Allegro", err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// ImportOffers handles POST /v1/integrations/allegro/import-offers.
// It fetches all seller offers from Allegro and imports them as Product + ProductListing records.
func (h *AllegroHandler) ImportOffers(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	if tenantID == uuid.Nil {
		writeError(w, http.StatusUnauthorized, "missing tenant context")
		return
	}

	result, err := h.allegroImportService.ImportOffers(r.Context(), tenantID)
	if err != nil {
		writeServerError(w, "failed to import offers from Allegro", err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}
