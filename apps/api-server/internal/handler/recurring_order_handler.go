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

// RecurringOrderHandler handles HTTP requests for recurring order management.
type RecurringOrderHandler struct {
	recurringOrderService *service.RecurringOrderService
}

// NewRecurringOrderHandler creates a new RecurringOrderHandler.
func NewRecurringOrderHandler(recurringOrderService *service.RecurringOrderService) *RecurringOrderHandler {
	return &RecurringOrderHandler{recurringOrderService: recurringOrderService}
}

// List returns a paginated list of recurring orders.
func (h *RecurringOrderHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	pagination := model.ParsePagination(r)

	filter := model.RecurringOrderListFilter{
		PaginationParams: pagination,
	}
	if status := r.URL.Query().Get("status"); status != "" {
		filter.Status = &status
	}
	if customerID := r.URL.Query().Get("customer_id"); customerID != "" {
		filter.CustomerID = &customerID
	}
	if nextDateBefore := r.URL.Query().Get("next_date_before"); nextDateBefore != "" {
		filter.NextDateBefore = &nextDateBefore
	}

	resp, err := h.recurringOrderService.List(r.Context(), tenantID, filter)
	if err != nil {
		writeServerError(w, "failed to list recurring orders", err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// Get returns a single recurring order by ID.
func (h *RecurringOrderHandler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid recurring order ID")
		return
	}

	ro, err := h.recurringOrderService.Get(r.Context(), tenantID, id)
	if err != nil {
		if errors.Is(err, service.ErrRecurringOrderNotFound) {
			writeError(w, http.StatusNotFound, "recurring order not found")
		} else {
			writeServerError(w, "failed to get recurring order", err)
		}
		return
	}
	writeJSON(w, http.StatusOK, ro)
}

// Create inserts a new recurring order.
func (h *RecurringOrderHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	var req model.CreateRecurringOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	ro, err := h.recurringOrderService.Create(r.Context(), tenantID, req, actorID, clientIP(r))
	if err != nil {
		if isValidationError(err) {
			writeError(w, http.StatusBadRequest, err.Error())
		} else {
			writeServerError(w, "failed to create recurring order", err)
		}
		return
	}
	writeJSON(w, http.StatusCreated, ro)
}

// Update modifies an existing recurring order.
func (h *RecurringOrderHandler) Update(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid recurring order ID")
		return
	}

	var req model.UpdateRecurringOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	ro, err := h.recurringOrderService.Update(r.Context(), tenantID, id, req, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrRecurringOrderNotFound):
			writeError(w, http.StatusNotFound, "recurring order not found")
		default:
			if isValidationError(err) {
				writeError(w, http.StatusBadRequest, err.Error())
			} else {
				writeServerError(w, "failed to update recurring order", err)
			}
		}
		return
	}
	writeJSON(w, http.StatusOK, ro)
}

// Delete removes a recurring order by ID.
func (h *RecurringOrderHandler) Delete(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid recurring order ID")
		return
	}

	err = h.recurringOrderService.Delete(r.Context(), tenantID, id, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrRecurringOrderNotFound):
			writeError(w, http.StatusNotFound, "recurring order not found")
		case errors.Is(err, service.ErrRecurringOrderHasOrders):
			writeError(w, http.StatusConflict, "cannot delete recurring order that has created orders")
		default:
			writeServerError(w, "failed to delete recurring order", err)
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Pause suspends a recurring order from generating new orders.
func (h *RecurringOrderHandler) Pause(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid recurring order ID")
		return
	}

	err = h.recurringOrderService.Pause(r.Context(), tenantID, id, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrRecurringOrderNotFound):
			writeError(w, http.StatusNotFound, "recurring order not found")
		case errors.Is(err, service.ErrRecurringOrderNotActive):
			writeError(w, http.StatusConflict, "recurring order is not active")
		default:
			writeServerError(w, "failed to pause recurring order", err)
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "paused"})
}

// Resume reactivates a paused recurring order.
func (h *RecurringOrderHandler) Resume(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid recurring order ID")
		return
	}

	err = h.recurringOrderService.Resume(r.Context(), tenantID, id, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrRecurringOrderNotFound):
			writeError(w, http.StatusNotFound, "recurring order not found")
		default:
			if isValidationError(err) {
				writeError(w, http.StatusConflict, err.Error())
			} else {
				writeServerError(w, "failed to resume recurring order", err)
			}
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "active"})
}

// Cancel permanently stops a recurring order.
func (h *RecurringOrderHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid recurring order ID")
		return
	}

	err = h.recurringOrderService.Cancel(r.Context(), tenantID, id, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrRecurringOrderNotFound):
			writeError(w, http.StatusNotFound, "recurring order not found")
		default:
			if isValidationError(err) {
				writeError(w, http.StatusConflict, err.Error())
			} else {
				writeServerError(w, "failed to cancel recurring order", err)
			}
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
}
