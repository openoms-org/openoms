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

type BundleHandler struct {
	bundleService *service.BundleService
}

func NewBundleHandler(bundleService *service.BundleService) *BundleHandler {
	return &BundleHandler{bundleService: bundleService}
}

func (h *BundleHandler) ListComponents(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	productID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid product ID")
		return
	}

	components, err := h.bundleService.ListComponents(r.Context(), tenantID, productID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list bundle components")
		return
	}
	writeJSON(w, http.StatusOK, components)
}

func (h *BundleHandler) AddComponent(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	productID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid product ID")
		return
	}

	var req model.CreateBundleComponentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	component, err := h.bundleService.AddComponent(r.Context(), tenantID, productID, req, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrProductNotBundle):
			writeError(w, http.StatusBadRequest, "product is not a bundle")
		default:
			if isValidationError(err) {
				writeError(w, http.StatusBadRequest, err.Error())
			} else {
				writeError(w, http.StatusInternalServerError, "failed to add bundle component")
			}
		}
		return
	}
	writeJSON(w, http.StatusCreated, component)
}

func (h *BundleHandler) UpdateComponent(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	componentID, err := uuid.Parse(chi.URLParam(r, "componentId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid component ID")
		return
	}

	var req model.UpdateBundleComponentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	component, err := h.bundleService.UpdateComponent(r.Context(), tenantID, componentID, req, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrBundleComponentNotFound):
			writeError(w, http.StatusNotFound, "bundle component not found")
		default:
			if isValidationError(err) {
				writeError(w, http.StatusBadRequest, err.Error())
			} else {
				writeError(w, http.StatusInternalServerError, "failed to update bundle component")
			}
		}
		return
	}
	writeJSON(w, http.StatusOK, component)
}

func (h *BundleHandler) RemoveComponent(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	componentID, err := uuid.Parse(chi.URLParam(r, "componentId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid component ID")
		return
	}

	err = h.bundleService.RemoveComponent(r.Context(), tenantID, componentID, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrBundleComponentNotFound):
			writeError(w, http.StatusNotFound, "bundle component not found")
		default:
			writeError(w, http.StatusInternalServerError, "failed to remove bundle component")
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *BundleHandler) GetBundleStock(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	productID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid product ID")
		return
	}

	stock, err := h.bundleService.CalculateBundleStock(r.Context(), tenantID, productID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to calculate bundle stock")
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"stock": stock})
}
