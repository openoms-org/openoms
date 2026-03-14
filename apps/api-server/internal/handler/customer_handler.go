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

// CustomerHandler handles HTTP requests for customer management.
type CustomerHandler struct {
	customerService       *service.CustomerService
	customerImportService *service.CustomerImportService
}

// NewCustomerHandler creates a new CustomerHandler.
func NewCustomerHandler(customerService *service.CustomerService, customerImportService *service.CustomerImportService) *CustomerHandler {
	return &CustomerHandler{
		customerService:       customerService,
		customerImportService: customerImportService,
	}
}

// List returns a paginated list of customers.
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
		writeServerError(w, "failed to list customers", err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// Get returns a single customer by ID.
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
			writeServerError(w, "failed to get customer", err)
		}
		return
	}
	writeJSON(w, http.StatusOK, customer)
}

// Create inserts a new customer.
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
			writeServerError(w, "failed to create customer", err)
		}
		return
	}
	writeJSON(w, http.StatusCreated, customer)
}

// Update modifies an existing customer.
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
				writeServerError(w, "failed to update customer", err)
			}
		}
		return
	}
	writeJSON(w, http.StatusOK, customer)
}

// Delete removes a customer by ID.
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
			writeServerError(w, "failed to delete customer", err)
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ListOrders returns orders placed by a specific customer.
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
			writeServerError(w, "failed to list customer orders", err)
		}
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// ImportPreview handles POST /v1/customers/import/preview
func (h *CustomerHandler) ImportPreview(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "file too large or invalid multipart form")
		return
	}
	defer func() { _ = r.MultipartForm.RemoveAll() }()

	file, _, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing file field")
		return
	}
	defer func() { _ = file.Close() }()

	tenantID := middleware.TenantIDFromContext(r.Context())
	preview, err := h.customerImportService.PreviewCSV(r.Context(), tenantID, file)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, preview)
}

// ImportCSV handles POST /v1/customers/import
func (h *CustomerHandler) ImportCSV(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	userID := middleware.UserIDFromContext(r.Context())

	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "file too large or invalid multipart form")
		return
	}
	defer func() { _ = r.MultipartForm.RemoveAll() }()

	file, _, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing file field")
		return
	}
	defer func() { _ = file.Close() }()

	result, err := h.customerImportService.ImportCSV(r.Context(), tenantID, file, userID, clientIP(r))
	if err != nil {
		writeServerError(w, "failed to import customers", err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}
