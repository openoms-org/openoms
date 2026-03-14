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

// ProductCategoryHandler handles HTTP requests for product categories.
type ProductCategoryHandler struct {
	categorySvc *service.ProductCategoryService
}

// NewProductCategoryHandler creates a new ProductCategoryHandler.
func NewProductCategoryHandler(categorySvc *service.ProductCategoryService) *ProductCategoryHandler {
	return &ProductCategoryHandler{categorySvc: categorySvc}
}

// List returns all product categories for the tenant.
func (h *ProductCategoryHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	filter := model.CategoryListFilter{}
	if r.URL.Query().Get("tree") == "true" {
		filter.IncludeTree = true
	}
	if parentIDStr := r.URL.Query().Get("parent_id"); parentIDStr != "" {
		parentID, err := uuid.Parse(parentIDStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid parent_id")
			return
		}
		filter.ParentID = &parentID
	}

	categories, err := h.categorySvc.List(r.Context(), tenantID, filter)
	if err != nil {
		writeServerError(w, "failed to list categories", err)
		return
	}

	if categories == nil {
		categories = []model.ProductCategory{}
	}
	writeJSON(w, http.StatusOK, categories)
}

// Get returns a single product category by ID.
func (h *ProductCategoryHandler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid category ID")
		return
	}

	category, err := h.categorySvc.Get(r.Context(), tenantID, id)
	if err != nil {
		if errors.Is(err, service.ErrCategoryNotFound) {
			writeError(w, http.StatusNotFound, "category not found")
		} else {
			writeServerError(w, "failed to get category", err)
		}
		return
	}
	writeJSON(w, http.StatusOK, category)
}

// Create inserts a new product category.
func (h *ProductCategoryHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	var req model.CreateCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := req.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	category, err := h.categorySvc.Create(r.Context(), tenantID, actorID, req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrCategoryNotFound):
			writeError(w, http.StatusBadRequest, "parent category not found")
		case errors.Is(err, service.ErrCategoryDepthLimit):
			writeError(w, http.StatusBadRequest, "maximum category depth exceeded")
		default:
			writeServerError(w, "failed to create category", err)
		}
		return
	}
	writeJSON(w, http.StatusCreated, category)
}

// Update modifies an existing product category.
func (h *ProductCategoryHandler) Update(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid category ID")
		return
	}

	var req model.UpdateCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := req.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	category, err := h.categorySvc.Update(r.Context(), tenantID, actorID, id, req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrCategoryNotFound):
			writeError(w, http.StatusNotFound, "category not found")
		case errors.Is(err, service.ErrCategoryDepthLimit):
			writeError(w, http.StatusBadRequest, "maximum category depth exceeded")
		case errors.Is(err, service.ErrCircularReference):
			writeError(w, http.StatusBadRequest, "circular parent reference detected")
		default:
			writeServerError(w, "failed to update category", err)
		}
		return
	}
	writeJSON(w, http.StatusOK, category)
}

// Delete removes a product category by ID.
func (h *ProductCategoryHandler) Delete(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid category ID")
		return
	}

	err = h.categorySvc.Delete(r.Context(), tenantID, actorID, id)
	if err != nil {
		if errors.Is(err, service.ErrCategoryNotFound) {
			writeError(w, http.StatusNotFound, "category not found")
		} else {
			writeServerError(w, "failed to delete category", err)
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ListDescendants returns the IDs of all descendant categories.
func (h *ProductCategoryHandler) ListDescendants(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid category ID")
		return
	}

	ids, err := h.categorySvc.GetDescendantIDs(r.Context(), tenantID, id)
	if err != nil {
		writeServerError(w, "failed to get descendants", err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ids": ids})
}
