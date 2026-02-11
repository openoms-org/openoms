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

type CustomerHandler struct {
	customerService *service.CustomerService
}

func NewCustomerHandler(customerService *service.CustomerService) *CustomerHandler {
	return &CustomerHandler{customerService: customerService}
}

func (h *CustomerHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	pagination := model.ParsePagination(r)

	filter := model.CustomerListFilter{
		PaginationParams: pagination,
	}
	if search := r.URL.Query().Get("search"); search != "" {
		filter.Search = &search
	}
	if tags := r.URL.Query().Get("tags"); tags != "" {
		filter.Tags = &tags
	}

	resp, err := h.customerService.List(r.Context(), tenantID, filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list customers")
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *CustomerHandler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	customerID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid customer ID")
		return
	}

	customer, err := h.customerService.Get(r.Context(), tenantID, customerID)
	if err != nil {
		if errors.Is(err, service.ErrCustomerNotFound) {
			writeError(w, http.StatusNotFound, "customer not found")
		} else {
			writeError(w, http.StatusInternalServerError, "failed to get customer")
		}
		return
	}
	writeJSON(w, http.StatusOK, customer)
}

func (h *CustomerHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	var req model.CreateCustomerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	customer, err := h.customerService.Create(r.Context(), tenantID, req, actorID, clientIP(r))
	if err != nil {
		if isValidationError(err) {
			writeError(w, http.StatusBadRequest, err.Error())
		} else {
			writeError(w, http.StatusInternalServerError, "failed to create customer")
		}
		return
	}
	writeJSON(w, http.StatusCreated, customer)
}

func (h *CustomerHandler) Update(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	customerID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid customer ID")
		return
	}

	var req model.UpdateCustomerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	customer, err := h.customerService.Update(r.Context(), tenantID, customerID, req, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrCustomerNotFound):
			writeError(w, http.StatusNotFound, "customer not found")
		default:
			if isValidationError(err) {
				writeError(w, http.StatusBadRequest, err.Error())
			} else {
				writeError(w, http.StatusInternalServerError, "failed to update customer")
			}
		}
		return
	}
	writeJSON(w, http.StatusOK, customer)
}

func (h *CustomerHandler) Delete(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	customerID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid customer ID")
		return
	}

	err = h.customerService.Delete(r.Context(), tenantID, customerID, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrCustomerNotFound):
			writeError(w, http.StatusNotFound, "customer not found")
		default:
			writeError(w, http.StatusInternalServerError, "failed to delete customer")
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *CustomerHandler) ListOrders(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	pagination := model.ParsePagination(r)

	customerID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid customer ID")
		return
	}

	filter := model.OrderListFilter{
		PaginationParams: pagination,
	}
	if status := r.URL.Query().Get("status"); status != "" {
		filter.Status = &status
	}

	resp, err := h.customerService.ListOrders(r.Context(), tenantID, customerID, filter)
	if err != nil {
		if errors.Is(err, service.ErrCustomerNotFound) {
			writeError(w, http.StatusNotFound, "customer not found")
		} else {
			writeError(w, http.StatusInternalServerError, "failed to list customer orders")
		}
		return
	}
	writeJSON(w, http.StatusOK, resp)
}
