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

// StocktakeHandler handles HTTP requests for stocktaking operations.
type StocktakeHandler struct {
	svc *service.StocktakeService
}

// NewStocktakeHandler creates a new StocktakeHandler.
func NewStocktakeHandler(svc *service.StocktakeService) *StocktakeHandler {
	return &StocktakeHandler{svc: svc}
}

// Create creates a new stocktake.
func (h *StocktakeHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	var req model.CreateStocktakeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	stocktake, err := h.svc.CreateStocktake(r.Context(), tenantID, req, actorID, clientIP(r))
	if err != nil {
		if isValidationError(err) {
			writeError(w, http.StatusBadRequest, err.Error())
		} else if service.IsForeignKeyError(err) {
			writeError(w, http.StatusBadRequest, "referenced warehouse or product does not exist")
		} else {
			writeError(w, http.StatusInternalServerError, "failed to create stocktake")
		}
		return
	}
	writeJSON(w, http.StatusCreated, stocktake)
}

// List returns stocktakes for the current tenant.
func (h *StocktakeHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	pagination := model.ParsePagination(r)

	filter := model.StocktakeListFilter{
		PaginationParams: pagination,
	}
	if wid := r.URL.Query().Get("warehouse_id"); wid != "" {
		id, err := uuid.Parse(wid)
		if err == nil {
			filter.WarehouseID = &id
		}
	}
	if st := r.URL.Query().Get("status"); st != "" {
		filter.Status = &st
	}

	resp, err := h.svc.ListStocktakes(r.Context(), tenantID, filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list stocktakes")
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// Get retrieves a single stocktake by ID with stats.
func (h *StocktakeHandler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	stocktakeID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid stocktake ID")
		return
	}

	stocktake, err := h.svc.GetStocktake(r.Context(), tenantID, stocktakeID)
	if err != nil {
		if errors.Is(err, service.ErrStocktakeNotFound) {
			writeError(w, http.StatusNotFound, "stocktake not found")
		} else {
			writeError(w, http.StatusInternalServerError, "failed to get stocktake")
		}
		return
	}
	writeJSON(w, http.StatusOK, stocktake)
}

// Delete deletes a stocktake (only if draft).
func (h *StocktakeHandler) Delete(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	stocktakeID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid stocktake ID")
		return
	}

	err = h.svc.DeleteStocktake(r.Context(), tenantID, stocktakeID, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrStocktakeNotFound):
			writeError(w, http.StatusNotFound, "stocktake not found")
		case errors.Is(err, service.ErrStocktakeNotDraft):
			writeError(w, http.StatusConflict, "stocktake is not in draft status")
		default:
			writeError(w, http.StatusInternalServerError, "failed to delete stocktake")
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Start starts a stocktake (sets status to in_progress).
func (h *StocktakeHandler) Start(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	stocktakeID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid stocktake ID")
		return
	}

	stocktake, err := h.svc.StartStocktake(r.Context(), tenantID, stocktakeID, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrStocktakeNotFound):
			writeError(w, http.StatusNotFound, "stocktake not found")
		case errors.Is(err, service.ErrStocktakeNotDraft):
			writeError(w, http.StatusConflict, "stocktake is not in draft status")
		default:
			writeError(w, http.StatusInternalServerError, "failed to start stocktake")
		}
		return
	}
	writeJSON(w, http.StatusOK, stocktake)
}

// RecordCount records the counted quantity for a stocktake item.
func (h *StocktakeHandler) RecordCount(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	stocktakeID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid stocktake ID")
		return
	}

	itemID, err := uuid.Parse(chi.URLParam(r, "itemId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid item ID")
		return
	}

	var req model.UpdateStocktakeItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	item, err := h.svc.RecordCount(r.Context(), tenantID, stocktakeID, itemID, req, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrStocktakeNotFound):
			writeError(w, http.StatusNotFound, "stocktake not found")
		case errors.Is(err, service.ErrStocktakeNotActive):
			writeError(w, http.StatusConflict, "stocktake is not in progress")
		case errors.Is(err, service.ErrStocktakeItemNotFound):
			writeError(w, http.StatusNotFound, "stocktake item not found")
		case isValidationError(err):
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "failed to record count")
		}
		return
	}
	writeJSON(w, http.StatusOK, item)
}

// Complete finalizes a stocktake and creates adjustment documents.
func (h *StocktakeHandler) Complete(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	stocktakeID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid stocktake ID")
		return
	}

	stocktake, err := h.svc.CompleteStocktake(r.Context(), tenantID, stocktakeID, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrStocktakeNotFound):
			writeError(w, http.StatusNotFound, "stocktake not found")
		case errors.Is(err, service.ErrStocktakeNotActive):
			writeError(w, http.StatusConflict, "stocktake is not in progress")
		case errors.Is(err, service.ErrNotAllItemsCounted):
			writeError(w, http.StatusConflict, "not all items have been counted")
		default:
			writeError(w, http.StatusInternalServerError, "failed to complete stocktake")
		}
		return
	}
	writeJSON(w, http.StatusOK, stocktake)
}

// Cancel cancels a stocktake.
func (h *StocktakeHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	stocktakeID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid stocktake ID")
		return
	}

	stocktake, err := h.svc.CancelStocktake(r.Context(), tenantID, stocktakeID, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrStocktakeNotFound):
			writeError(w, http.StatusNotFound, "stocktake not found")
		default:
			writeError(w, http.StatusInternalServerError, "failed to cancel stocktake")
		}
		return
	}
	writeJSON(w, http.StatusOK, stocktake)
}

// ListItems returns items for a stocktake with pagination and filtering.
func (h *StocktakeHandler) ListItems(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	stocktakeID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid stocktake ID")
		return
	}

	pagination := model.ParsePagination(r)
	filterStr := r.URL.Query().Get("filter")
	if filterStr == "" {
		filterStr = "all"
	}

	filter := model.StocktakeItemListFilter{
		Filter:           filterStr,
		PaginationParams: pagination,
	}

	resp, err := h.svc.GetStocktakeItems(r.Context(), tenantID, stocktakeID, filter)
	if err != nil {
		if errors.Is(err, service.ErrStocktakeNotFound) {
			writeError(w, http.StatusNotFound, "stocktake not found")
		} else {
			writeError(w, http.StatusInternalServerError, "failed to list stocktake items")
		}
		return
	}
	writeJSON(w, http.StatusOK, resp)
}
