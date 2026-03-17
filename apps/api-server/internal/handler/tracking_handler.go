package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// TrackingHandler handles public order tracking requests.
type TrackingHandler struct {
	trackingService *service.TrackingService
}

// NewTrackingHandler creates a new TrackingHandler.
func NewTrackingHandler(trackingService *service.TrackingService) *TrackingHandler {
	return &TrackingHandler{trackingService: trackingService}
}

// TrackOrder handles GET and POST /v1/tracking/{tenant_slug}/{order_id}
// Email from POST JSON body (preferred) or GET query param (legacy).
// POST body prevents PII (email) from appearing in access logs.
func (h *TrackingHandler) TrackOrder(w http.ResponseWriter, r *http.Request) {
	tenantSlug := chi.URLParam(r, "tenant_slug")
	orderID := chi.URLParam(r, "order_id")

	// Prefer email from POST body, fall back to query param for backward compat
	var email string
	if r.Method == http.MethodPost && r.Body != nil {
		var body struct {
			Email string `json:"email"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err == nil {
			email = strings.TrimSpace(body.Email)
		}
	}
	if email == "" {
		email = strings.TrimSpace(r.URL.Query().Get("email"))
	}

	if tenantSlug == "" || orderID == "" || email == "" {
		writeError(w, http.StatusBadRequest, "tenant_slug, order_id, and email are required")
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
		writeServerError(w, "failed to look up order", err)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}
