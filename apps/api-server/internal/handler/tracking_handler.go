package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// TrackingHandler handles public order tracking requests.
type TrackingHandler struct {
	trackingService *service.TrackingService
}

func NewTrackingHandler(trackingService *service.TrackingService) *TrackingHandler {
	return &TrackingHandler{trackingService: trackingService}
}

// TrackOrder handles GET /v1/tracking/{tenant_slug}/{order_id}?email=...
func (h *TrackingHandler) TrackOrder(w http.ResponseWriter, r *http.Request) {
	tenantSlug := chi.URLParam(r, "tenant_slug")
	orderID := chi.URLParam(r, "order_id")
	email := strings.TrimSpace(r.URL.Query().Get("email"))

	if tenantSlug == "" || orderID == "" || email == "" {
		writeError(w, http.StatusBadRequest, "tenant_slug, order_id, and email query parameter are required")
		return
	}

	resp, err := h.trackingService.TrackOrder(r.Context(), tenantSlug, orderID, email)
	if err != nil {
		if errors.Is(err, service.ErrTrackingNotFound) {
			writeError(w, http.StatusNotFound, "order not found")
			return
		}
		if errors.Is(err, service.ErrTrackingEmail) {
			writeError(w, http.StatusForbidden, "email does not match order")
			return
		}
		slog.Error("tracking lookup failed", "error", err, "tenant_slug", tenantSlug, "order_id", orderID)
		writeError(w, http.StatusInternalServerError, "failed to look up order")
		return
	}

	writeJSON(w, http.StatusOK, resp)
}
