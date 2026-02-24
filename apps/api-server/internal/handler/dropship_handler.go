package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// DropshipHandler handles HTTP requests for dropship orders.
type DropshipHandler struct {
	dropshipService *service.DropshipService
}

// NewDropshipHandler creates a new DropshipHandler.
func NewDropshipHandler(dropshipService *service.DropshipService) *DropshipHandler {
	return &DropshipHandler{dropshipService: dropshipService}
}

// List returns a paginated list of dropship orders.
func (h *DropshipHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	pagination := model.ParsePagination(r)

	filter := model.DropshipOrderListFilter{
		PaginationParams: pagination,
	}
	if status := r.URL.Query().Get("status"); status != "" {
		filter.Status = &status
	}
	if supplierID := r.URL.Query().Get("supplier_id"); supplierID != "" {
		id, err := uuid.Parse(supplierID)
		if err == nil {
			filter.SupplierID = &id
		}
	}
	if orderID := r.URL.Query().Get("order_id"); orderID != "" {
		id, err := uuid.Parse(orderID)
		if err == nil {
			filter.OrderID = &id
		}
	}

	resp, err := h.dropshipService.List(r.Context(), tenantID, filter)
	if err != nil {
		writeServerError(w, "failed to list dropship orders", err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// Get returns a single dropship order with its items.
func (h *DropshipHandler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid dropship order ID")
		return
	}

	d, err := h.dropshipService.Get(r.Context(), tenantID, id)
	if err != nil {
		if errors.Is(err, service.ErrDropshipOrderNotFound) {
			writeError(w, http.StatusNotFound, "dropship order not found")
		} else {
			writeServerError(w, "failed to get dropship order", err)
		}
		return
	}
	writeJSON(w, http.StatusOK, d)
}

// Create creates a new dropship order manually.
func (h *DropshipHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	var req model.CreateDropshipOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	d, err := h.dropshipService.Create(r.Context(), tenantID, req, actorID, clientIP(r))
	if err != nil {
		if isValidationError(err) {
			writeError(w, http.StatusBadRequest, err.Error())
		} else {
			writeServerError(w, "failed to create dropship order", err)
		}
		return
	}
	writeJSON(w, http.StatusCreated, d)
}

// AutoRoute auto-routes an order to suppliers based on dropship product configuration.
func (h *DropshipHandler) AutoRoute(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	orderID, err := uuid.Parse(chi.URLParam(r, "order_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid order ID")
		return
	}

	orders, err := h.dropshipService.AutoRouteOrder(r.Context(), tenantID, orderID, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrNoDropshipItems):
			writeError(w, http.StatusUnprocessableEntity, "no dropship items found in this order")
		default:
			if isValidationError(err) {
				writeError(w, http.StatusBadRequest, err.Error())
			} else {
				writeServerError(w, "failed to auto-route order", err)
			}
		}
		return
	}
	writeJSON(w, http.StatusCreated, orders)
}

// GetByOrderID returns all dropship orders for a customer order.
func (h *DropshipHandler) GetByOrderID(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	orderID, err := uuid.Parse(chi.URLParam(r, "order_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid order ID")
		return
	}

	orders, err := h.dropshipService.GetByOrderID(r.Context(), tenantID, orderID)
	if err != nil {
		writeServerError(w, "failed to get dropship orders for order", err)
		return
	}
	writeJSON(w, http.StatusOK, orders)
}

// UpdateStatus updates the status of a dropship order.
func (h *DropshipHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid dropship order ID")
		return
	}

	var req model.UpdateDropshipStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	d, err := h.dropshipService.UpdateStatus(r.Context(), tenantID, id, req, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrDropshipOrderNotFound):
			writeError(w, http.StatusNotFound, "dropship order not found")
		default:
			if isValidationError(err) {
				writeError(w, http.StatusBadRequest, err.Error())
			} else {
				writeServerError(w, "failed to update dropship order status", err)
			}
		}
		return
	}
	writeJSON(w, http.StatusOK, d)
}

// Cancel cancels a dropship order.
func (h *DropshipHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid dropship order ID")
		return
	}

	d, err := h.dropshipService.Cancel(r.Context(), tenantID, id, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrDropshipOrderNotFound):
			writeError(w, http.StatusNotFound, "dropship order not found")
		case errors.Is(err, service.ErrDropshipAlreadyCancelled):
			writeError(w, http.StatusConflict, "dropship order is already cancelled")
		default:
			if isValidationError(err) {
				writeError(w, http.StatusBadRequest, err.Error())
			} else {
				writeServerError(w, "failed to cancel dropship order", err)
			}
		}
		return
	}
	writeJSON(w, http.StatusOK, d)
}
