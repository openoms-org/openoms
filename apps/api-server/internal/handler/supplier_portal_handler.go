package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// SupplierPortalHandler handles HTTP requests for the supplier portal.
type SupplierPortalHandler struct {
	portalService *service.SupplierPortalService
}

// NewSupplierPortalHandler creates a new SupplierPortalHandler.
func NewSupplierPortalHandler(portalService *service.SupplierPortalService) *SupplierPortalHandler {
	return &SupplierPortalHandler{portalService: portalService}
}

// ---------- Admin endpoints (JWT auth) ----------

// GenerateLink generates a portal link for a supplier.
func (h *SupplierPortalHandler) GenerateLink(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	supplierID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid supplier ID")
		return
	}

	resp, err := h.portalService.GeneratePortalLink(r.Context(), tenantID, supplierID, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrSupplierNotFound):
			writeError(w, http.StatusNotFound, "supplier not found")
		default:
			writeServerError(w, "failed to generate portal link", err)
		}
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// RevokeAccess revokes all portal tokens for a supplier.
func (h *SupplierPortalHandler) RevokeAccess(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	supplierID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid supplier ID")
		return
	}

	if err := h.portalService.RevokeTokens(r.Context(), tenantID, supplierID, actorID, clientIP(r)); err != nil {
		writeServerError(w, "failed to revoke portal access", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "portal access revoked"})
}

// GetPortalStatus returns portal status for a supplier (admin UI).
func (h *SupplierPortalHandler) GetPortalStatus(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	supplierID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid supplier ID")
		return
	}

	enabled, lastUsedAt, err := h.portalService.GetPortalStatus(r.Context(), tenantID, supplierID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrSupplierNotFound):
			writeError(w, http.StatusNotFound, "supplier not found")
		default:
			writeServerError(w, "failed to get portal status", err)
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"portal_enabled": enabled,
		"last_used_at":   lastUsedAt,
	})
}

// ---------- Public portal endpoints (token auth) ----------

// extractPortalToken gets the raw token from the Authorization header only.
// Query-param fallback was removed because tokens leak via nginx/Cloudflare
// access logs, browser history, and Referer headers (OPE-169).
func extractPortalToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if after, ok := strings.CutPrefix(authHeader, "Bearer "); ok {
		return strings.TrimSpace(after)
	}
	return ""
}

// ListOrders lists purchase orders for the authenticated supplier.
func (h *SupplierPortalHandler) ListOrders(w http.ResponseWriter, r *http.Request) {
	rawToken := extractPortalToken(r)
	if rawToken == "" {
		writeError(w, http.StatusUnauthorized, "portal token required")
		return
	}

	supplier, tenantID, err := h.portalService.ValidateToken(r.Context(), rawToken)
	if err != nil {
		h.handlePortalAuthError(w, err)
		return
	}

	orders, err := h.portalService.ListPurchaseOrders(r.Context(), tenantID, supplier.ID)
	if err != nil {
		writeServerError(w, "failed to list orders", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"supplier": map[string]any{
			"id":   supplier.ID,
			"name": supplier.Name,
		},
		"orders": orders,
	})
}

// GetOrder returns a single PO with items for the authenticated supplier.
func (h *SupplierPortalHandler) GetOrder(w http.ResponseWriter, r *http.Request) {
	rawToken := extractPortalToken(r)
	if rawToken == "" {
		writeError(w, http.StatusUnauthorized, "portal token required")
		return
	}

	supplier, tenantID, err := h.portalService.ValidateToken(r.Context(), rawToken)
	if err != nil {
		h.handlePortalAuthError(w, err)
		return
	}

	poID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid order ID")
		return
	}

	po, err := h.portalService.GetPurchaseOrder(r.Context(), tenantID, supplier.ID, poID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrPortalPONotFound), errors.Is(err, service.ErrPortalPONotOwned):
			writeError(w, http.StatusNotFound, "purchase order not found")
		default:
			writeServerError(w, "failed to get order", err)
		}
		return
	}
	writeJSON(w, http.StatusOK, po)
}

// ConfirmOrder allows the supplier to confirm a PO.
func (h *SupplierPortalHandler) ConfirmOrder(w http.ResponseWriter, r *http.Request) {
	rawToken := extractPortalToken(r)
	if rawToken == "" {
		writeError(w, http.StatusUnauthorized, "portal token required")
		return
	}

	supplier, tenantID, err := h.portalService.ValidateToken(r.Context(), rawToken)
	if err != nil {
		h.handlePortalAuthError(w, err)
		return
	}

	poID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid order ID")
		return
	}

	var req model.SupplierConfirmRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Body is optional for confirm
		req = model.SupplierConfirmRequest{}
	}

	if err := h.portalService.ConfirmOrder(r.Context(), tenantID, supplier.ID, poID, req.Reference); err != nil {
		switch {
		case errors.Is(err, service.ErrPortalPONotFound), errors.Is(err, service.ErrPortalPONotOwned):
			writeError(w, http.StatusNotFound, "purchase order not found")
		case errors.Is(err, service.ErrPortalInvalidAction):
			writeError(w, http.StatusConflict, "cannot confirm this order in its current status")
		default:
			writeServerError(w, "failed to confirm order", err)
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "order confirmed"})
}

// ShipOrder allows the supplier to mark a PO as shipped.
func (h *SupplierPortalHandler) ShipOrder(w http.ResponseWriter, r *http.Request) {
	rawToken := extractPortalToken(r)
	if rawToken == "" {
		writeError(w, http.StatusUnauthorized, "portal token required")
		return
	}

	supplier, tenantID, err := h.portalService.ValidateToken(r.Context(), rawToken)
	if err != nil {
		h.handlePortalAuthError(w, err)
		return
	}

	poID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid order ID")
		return
	}

	var req model.SupplierShipRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := req.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.portalService.MarkShipped(r.Context(), tenantID, supplier.ID, poID, req.TrackingNumber, req.Carrier); err != nil {
		switch {
		case errors.Is(err, service.ErrPortalPONotFound), errors.Is(err, service.ErrPortalPONotOwned):
			writeError(w, http.StatusNotFound, "purchase order not found")
		case errors.Is(err, service.ErrPortalInvalidAction):
			writeError(w, http.StatusConflict, "cannot ship this order in its current status")
		default:
			writeServerError(w, "failed to mark order as shipped", err)
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "order marked as shipped"})
}

// AddMessage adds a message from the supplier on a PO.
func (h *SupplierPortalHandler) AddMessage(w http.ResponseWriter, r *http.Request) {
	rawToken := extractPortalToken(r)
	if rawToken == "" {
		writeError(w, http.StatusUnauthorized, "portal token required")
		return
	}

	supplier, tenantID, err := h.portalService.ValidateToken(r.Context(), rawToken)
	if err != nil {
		h.handlePortalAuthError(w, err)
		return
	}

	poID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid order ID")
		return
	}

	var req model.CreateSupplierMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := req.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	msg, err := h.portalService.AddMessage(r.Context(), tenantID, supplier.ID, poID, req.Message)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrPortalPONotFound), errors.Is(err, service.ErrPortalPONotOwned):
			writeError(w, http.StatusNotFound, "purchase order not found")
		default:
			writeServerError(w, "failed to add message", err)
		}
		return
	}
	writeJSON(w, http.StatusCreated, msg)
}

// ListMessages lists messages for a PO.
func (h *SupplierPortalHandler) ListMessages(w http.ResponseWriter, r *http.Request) {
	rawToken := extractPortalToken(r)
	if rawToken == "" {
		writeError(w, http.StatusUnauthorized, "portal token required")
		return
	}

	supplier, tenantID, err := h.portalService.ValidateToken(r.Context(), rawToken)
	if err != nil {
		h.handlePortalAuthError(w, err)
		return
	}

	poID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid order ID")
		return
	}

	messages, err := h.portalService.ListMessages(r.Context(), tenantID, supplier.ID, poID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrPortalPONotFound), errors.Is(err, service.ErrPortalPONotOwned):
			writeError(w, http.StatusNotFound, "purchase order not found")
		default:
			writeServerError(w, "failed to list messages", err)
		}
		return
	}
	writeJSON(w, http.StatusOK, messages)
}

// handlePortalAuthError writes the appropriate error response for portal auth errors.
func (h *SupplierPortalHandler) handlePortalAuthError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrPortalTokenInvalid):
		writeError(w, http.StatusUnauthorized, "invalid portal token")
	case errors.Is(err, service.ErrPortalTokenExpired):
		writeError(w, http.StatusUnauthorized, "portal token has expired")
	case errors.Is(err, service.ErrPortalNotEnabled):
		writeError(w, http.StatusForbidden, "portal access has been disabled")
	default:
		writeServerError(w, "authentication failed", err)
	}
}
