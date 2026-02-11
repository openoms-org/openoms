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

// WarehouseHandler handles HTTP requests for warehouse CRUD and stock management.
type WarehouseHandler struct {
	warehouseService *service.WarehouseService
}

// NewWarehouseHandler creates a new WarehouseHandler.
func NewWarehouseHandler(warehouseService *service.WarehouseService) *WarehouseHandler {
	return &WarehouseHandler{warehouseService: warehouseService}
}

// List returns all warehouses for the current tenant.
func (h *WarehouseHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	pagination := model.ParsePagination(r)

	filter := model.WarehouseListFilter{
		PaginationParams: pagination,
	}
	if a := r.URL.Query().Get("active"); a == "true" {
		active := true
		filter.Active = &active
	} else if a == "false" {
		active := false
		filter.Active = &active
	}

	resp, err := h.warehouseService.List(r.Context(), tenantID, filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list warehouses")
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// Get retrieves a single warehouse by ID.
func (h *WarehouseHandler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	warehouseID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid warehouse ID")
		return
	}

	warehouse, err := h.warehouseService.Get(r.Context(), tenantID, warehouseID)
	if err != nil {
		if errors.Is(err, service.ErrWarehouseNotFound) {
			writeError(w, http.StatusNotFound, "warehouse not found")
		} else {
			writeError(w, http.StatusInternalServerError, "failed to get warehouse")
		}
		return
	}
	writeJSON(w, http.StatusOK, warehouse)
}

// Create creates a new warehouse.
func (h *WarehouseHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	var req model.CreateWarehouseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	warehouse, err := h.warehouseService.Create(r.Context(), tenantID, req, actorID, clientIP(r))
	if err != nil {
		if isValidationError(err) {
			writeError(w, http.StatusBadRequest, err.Error())
		} else {
			writeError(w, http.StatusInternalServerError, "failed to create warehouse")
		}
		return
	}
	writeJSON(w, http.StatusCreated, warehouse)
}

// Update updates an existing warehouse.
func (h *WarehouseHandler) Update(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	warehouseID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid warehouse ID")
		return
	}

	var req model.UpdateWarehouseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	warehouse, err := h.warehouseService.Update(r.Context(), tenantID, warehouseID, req, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrWarehouseNotFound):
			writeError(w, http.StatusNotFound, "warehouse not found")
		default:
			if isValidationError(err) {
				writeError(w, http.StatusBadRequest, err.Error())
			} else {
				writeError(w, http.StatusInternalServerError, "failed to update warehouse")
			}
		}
		return
	}
	writeJSON(w, http.StatusOK, warehouse)
}

// Delete removes a warehouse.
func (h *WarehouseHandler) Delete(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	warehouseID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid warehouse ID")
		return
	}

	err = h.warehouseService.Delete(r.Context(), tenantID, warehouseID, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrWarehouseNotFound):
			writeError(w, http.StatusNotFound, "warehouse not found")
		default:
			writeError(w, http.StatusInternalServerError, "failed to delete warehouse")
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ListStock returns stock entries for a warehouse.
func (h *WarehouseHandler) ListStock(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	warehouseID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid warehouse ID")
		return
	}

	pagination := model.ParsePagination(r)
	filter := model.WarehouseStockListFilter{
		PaginationParams: pagination,
	}

	resp, err := h.warehouseService.ListStock(r.Context(), tenantID, warehouseID, filter)
	if err != nil {
		if errors.Is(err, service.ErrWarehouseNotFound) {
			writeError(w, http.StatusNotFound, "warehouse not found")
		} else {
			writeError(w, http.StatusInternalServerError, "failed to list warehouse stock")
		}
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// UpsertStock creates or updates a stock entry for a warehouse.
func (h *WarehouseHandler) UpsertStock(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	warehouseID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid warehouse ID")
		return
	}

	var req model.UpsertWarehouseStockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	stock, err := h.warehouseService.UpsertStock(r.Context(), tenantID, warehouseID, req, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrWarehouseNotFound):
			writeError(w, http.StatusNotFound, "warehouse not found")
		default:
			if isValidationError(err) {
				writeError(w, http.StatusBadRequest, err.Error())
			} else {
				writeError(w, http.StatusInternalServerError, "failed to upsert stock")
			}
		}
		return
	}
	writeJSON(w, http.StatusOK, stock)
}

// ListProductStock returns stock entries across all warehouses for a product.
func (h *WarehouseHandler) ListProductStock(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	productID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid product ID")
		return
	}

	stocks, err := h.warehouseService.ListProductStock(r.Context(), tenantID, productID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list product stock")
		return
	}
	writeJSON(w, http.StatusOK, stocks)
}
