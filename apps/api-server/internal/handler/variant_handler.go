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

// VariantHandler handles HTTP requests for product variant CRUD.
type VariantHandler struct {
	variantService *service.VariantService
}

// NewVariantHandler creates a new VariantHandler.
func NewVariantHandler(variantService *service.VariantService) *VariantHandler {
	return &VariantHandler{variantService: variantService}
}

// List returns all variants for a product.
func (h *VariantHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	pagination := model.ParsePagination(r)

	productID, err := uuid.Parse(chi.URLParam(r, "productId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid product ID")
		return
	}

	filter := model.VariantListFilter{
		ProductID:        productID,
		PaginationParams: pagination,
	}
	if a := r.URL.Query().Get("active"); a == "true" {
		active := true
		filter.Active = &active
	} else if a == "false" {
		active := false
		filter.Active = &active
	}

	variants, total, err := h.variantService.List(r.Context(), tenantID, filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list variants")
		return
	}
	if variants == nil {
		variants = []model.ProductVariant{}
	}
	writeJSON(w, http.StatusOK, model.ListResponse[model.ProductVariant]{
		Items:  variants,
		Total:  total,
		Limit:  pagination.Limit,
		Offset: pagination.Offset,
	})
}

// Get retrieves a single variant by ID.
func (h *VariantHandler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	variantID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid variant ID")
		return
	}

	variant, err := h.variantService.Get(r.Context(), tenantID, variantID)
	if err != nil {
		if errors.Is(err, service.ErrVariantNotFound) {
			writeError(w, http.StatusNotFound, "variant not found")
		} else {
			writeError(w, http.StatusInternalServerError, "failed to get variant")
		}
		return
	}
	writeJSON(w, http.StatusOK, variant)
}

// Create creates a new variant for a product.
func (h *VariantHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	productID, err := uuid.Parse(chi.URLParam(r, "productId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid product ID")
		return
	}

	var req model.CreateVariantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	variant, err := h.variantService.Create(r.Context(), tenantID, productID, req, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrProductNotFound):
			writeError(w, http.StatusNotFound, "product not found")
		default:
			if isValidationError(err) {
				writeError(w, http.StatusBadRequest, err.Error())
			} else {
				writeError(w, http.StatusInternalServerError, "failed to create variant")
			}
		}
		return
	}
	writeJSON(w, http.StatusCreated, variant)
}

// Update updates an existing variant.
func (h *VariantHandler) Update(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	variantID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid variant ID")
		return
	}

	var req model.UpdateVariantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	variant, err := h.variantService.Update(r.Context(), tenantID, variantID, req, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrVariantNotFound):
			writeError(w, http.StatusNotFound, "variant not found")
		default:
			if isValidationError(err) {
				writeError(w, http.StatusBadRequest, err.Error())
			} else {
				writeError(w, http.StatusInternalServerError, "failed to update variant")
			}
		}
		return
	}
	writeJSON(w, http.StatusOK, variant)
}

// Delete removes a variant.
func (h *VariantHandler) Delete(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	variantID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid variant ID")
		return
	}

	err = h.variantService.Delete(r.Context(), tenantID, variantID, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrVariantNotFound):
			writeError(w, http.StatusNotFound, "variant not found")
		default:
			writeError(w, http.StatusInternalServerError, "failed to delete variant")
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
