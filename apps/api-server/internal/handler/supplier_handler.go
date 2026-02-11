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

type SupplierHandler struct {
	supplierService *service.SupplierService
}

func NewSupplierHandler(supplierService *service.SupplierService) *SupplierHandler {
	return &SupplierHandler{supplierService: supplierService}
}

func (h *SupplierHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	pagination := model.ParsePagination(r)

	filter := model.SupplierListFilter{
		PaginationParams: pagination,
	}
	if status := r.URL.Query().Get("status"); status != "" {
		filter.Status = &status
	}

	resp, err := h.supplierService.List(r.Context(), tenantID, filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list suppliers")
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *SupplierHandler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	supplierID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid supplier ID")
		return
	}

	supplier, err := h.supplierService.Get(r.Context(), tenantID, supplierID)
	if err != nil {
		if errors.Is(err, service.ErrSupplierNotFound) {
			writeError(w, http.StatusNotFound, "supplier not found")
		} else {
			writeError(w, http.StatusInternalServerError, "failed to get supplier")
		}
		return
	}
	writeJSON(w, http.StatusOK, supplier)
}

func (h *SupplierHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	var req model.CreateSupplierRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	supplier, err := h.supplierService.Create(r.Context(), tenantID, req, actorID, clientIP(r))
	if err != nil {
		if isValidationError(err) {
			writeError(w, http.StatusBadRequest, err.Error())
		} else {
			writeError(w, http.StatusInternalServerError, "failed to create supplier")
		}
		return
	}
	writeJSON(w, http.StatusCreated, supplier)
}

func (h *SupplierHandler) Update(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	supplierID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid supplier ID")
		return
	}

	var req model.UpdateSupplierRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	supplier, err := h.supplierService.Update(r.Context(), tenantID, supplierID, req, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrSupplierNotFound):
			writeError(w, http.StatusNotFound, "supplier not found")
		default:
			if isValidationError(err) {
				writeError(w, http.StatusBadRequest, err.Error())
			} else {
				writeError(w, http.StatusInternalServerError, "failed to update supplier")
			}
		}
		return
	}
	writeJSON(w, http.StatusOK, supplier)
}

func (h *SupplierHandler) Delete(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	supplierID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid supplier ID")
		return
	}

	err = h.supplierService.Delete(r.Context(), tenantID, supplierID, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrSupplierNotFound):
			writeError(w, http.StatusNotFound, "supplier not found")
		default:
			writeError(w, http.StatusInternalServerError, "failed to delete supplier")
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *SupplierHandler) Sync(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	supplierID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid supplier ID")
		return
	}

	err = h.supplierService.SyncFeed(r.Context(), tenantID, supplierID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrSupplierNotFound):
			writeError(w, http.StatusNotFound, "supplier not found")
		case errors.Is(err, service.ErrNoFeedURL):
			writeError(w, http.StatusBadRequest, "supplier has no feed URL configured")
		default:
			writeError(w, http.StatusInternalServerError, "failed to sync feed")
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "feed synced successfully"})
}

func (h *SupplierHandler) ListProducts(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	pagination := model.ParsePagination(r)

	supplierID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid supplier ID")
		return
	}

	filter := model.SupplierProductListFilter{
		SupplierID:       &supplierID,
		PaginationParams: pagination,
	}
	if ean := r.URL.Query().Get("ean"); ean != "" {
		filter.EAN = &ean
	}
	if linked := r.URL.Query().Get("linked"); linked != "" {
		val := linked == "true"
		filter.Linked = &val
	}

	resp, err := h.supplierService.ListProducts(r.Context(), tenantID, filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list supplier products")
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *SupplierHandler) LinkProduct(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	supplierProductID, err := uuid.Parse(chi.URLParam(r, "spid"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid supplier product ID")
		return
	}

	var req model.LinkProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := req.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	err = h.supplierService.LinkProduct(r.Context(), tenantID, supplierProductID, req.ProductID, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrSupplierProductNotFound):
			writeError(w, http.StatusNotFound, "supplier product not found")
		default:
			writeError(w, http.StatusInternalServerError, "failed to link product")
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "product linked"})
}
