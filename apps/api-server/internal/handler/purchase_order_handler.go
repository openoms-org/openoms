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

// PurchaseOrderHandler handles HTTP requests for purchase orders.
type PurchaseOrderHandler struct {
	poService *service.PurchaseOrderService
}

// NewPurchaseOrderHandler creates a new PurchaseOrderHandler.
func NewPurchaseOrderHandler(poService *service.PurchaseOrderService) *PurchaseOrderHandler {
	return &PurchaseOrderHandler{poService: poService}
}

// List returns a paginated list of purchase orders.
func (h *PurchaseOrderHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	pagination := model.ParsePagination(r)

	filter := model.PurchaseOrderListFilter{
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

	resp, err := h.poService.List(r.Context(), tenantID, filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list purchase orders")
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// Get returns a single purchase order with its items.
func (h *PurchaseOrderHandler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	poID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid purchase order ID")
		return
	}

	po, err := h.poService.Get(r.Context(), tenantID, poID)
	if err != nil {
		if errors.Is(err, service.ErrPurchaseOrderNotFound) {
			writeError(w, http.StatusNotFound, "purchase order not found")
		} else {
			writeError(w, http.StatusInternalServerError, "failed to get purchase order")
		}
		return
	}
	writeJSON(w, http.StatusOK, po)
}

// Create creates a new purchase order.
func (h *PurchaseOrderHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	var req model.CreatePurchaseOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	po, err := h.poService.Create(r.Context(), tenantID, req, actorID, clientIP(r))
	if err != nil {
		if isValidationError(err) {
			writeError(w, http.StatusBadRequest, err.Error())
		} else {
			writeError(w, http.StatusInternalServerError, "failed to create purchase order")
		}
		return
	}
	writeJSON(w, http.StatusCreated, po)
}

// Update updates an existing purchase order.
func (h *PurchaseOrderHandler) Update(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	poID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid purchase order ID")
		return
	}

	var req model.UpdatePurchaseOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	po, err := h.poService.Update(r.Context(), tenantID, poID, req, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrPurchaseOrderNotFound):
			writeError(w, http.StatusNotFound, "purchase order not found")
		case errors.Is(err, service.ErrPONotDraft):
			writeError(w, http.StatusConflict, "can only edit purchase orders in draft status")
		default:
			if isValidationError(err) {
				writeError(w, http.StatusBadRequest, err.Error())
			} else {
				writeError(w, http.StatusInternalServerError, "failed to update purchase order")
			}
		}
		return
	}
	writeJSON(w, http.StatusOK, po)
}

// Delete deletes a purchase order.
func (h *PurchaseOrderHandler) Delete(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	poID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid purchase order ID")
		return
	}

	err = h.poService.Delete(r.Context(), tenantID, poID, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrPurchaseOrderNotFound):
			writeError(w, http.StatusNotFound, "purchase order not found")
		case errors.Is(err, service.ErrPONotDraft):
			writeError(w, http.StatusConflict, "can only delete purchase orders in draft status")
		default:
			writeError(w, http.StatusInternalServerError, "failed to delete purchase order")
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Send marks a purchase order as sent.
func (h *PurchaseOrderHandler) Send(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	poID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid purchase order ID")
		return
	}

	po, err := h.poService.Send(r.Context(), tenantID, poID, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrPurchaseOrderNotFound):
			writeError(w, http.StatusNotFound, "purchase order not found")
		default:
			if isValidationError(err) {
				writeError(w, http.StatusBadRequest, err.Error())
			} else {
				writeError(w, http.StatusInternalServerError, "failed to send purchase order")
			}
		}
		return
	}
	writeJSON(w, http.StatusOK, po)
}

// Receive receives items against a purchase order.
func (h *PurchaseOrderHandler) Receive(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	poID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid purchase order ID")
		return
	}

	var req model.ReceiveItemsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	po, err := h.poService.ReceiveItems(r.Context(), tenantID, poID, req, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrPurchaseOrderNotFound):
			writeError(w, http.StatusNotFound, "purchase order not found")
		case errors.Is(err, service.ErrPOItemNotFound):
			writeError(w, http.StatusNotFound, "purchase order item not found")
		default:
			if isValidationError(err) {
				writeError(w, http.StatusBadRequest, err.Error())
			} else {
				writeError(w, http.StatusInternalServerError, "failed to receive items")
			}
		}
		return
	}
	writeJSON(w, http.StatusOK, po)
}

// Cancel cancels a purchase order.
func (h *PurchaseOrderHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	poID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid purchase order ID")
		return
	}

	po, err := h.poService.Cancel(r.Context(), tenantID, poID, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrPurchaseOrderNotFound):
			writeError(w, http.StatusNotFound, "purchase order not found")
		case errors.Is(err, service.ErrPOAlreadyCancelled):
			writeError(w, http.StatusConflict, "purchase order is already cancelled")
		default:
			if isValidationError(err) {
				writeError(w, http.StatusBadRequest, err.Error())
			} else {
				writeError(w, http.StatusInternalServerError, "failed to cancel purchase order")
			}
		}
		return
	}
	writeJSON(w, http.StatusOK, po)
}
