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

type PriceListHandler struct {
	priceListService *service.PriceListService
}

func NewPriceListHandler(priceListService *service.PriceListService) *PriceListHandler {
	return &PriceListHandler{priceListService: priceListService}
}

func (h *PriceListHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	pagination := model.ParsePagination(r)

	filter := model.PriceListListFilter{
		PaginationParams: pagination,
	}
	if active := r.URL.Query().Get("active"); active == "true" {
		v := true
		filter.Active = &v
	} else if active == "false" {
		v := false
		filter.Active = &v
	}

	priceLists, total, err := h.priceListService.List(r.Context(), tenantID, filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list price lists")
		return
	}
	if priceLists == nil {
		priceLists = []model.PriceList{}
	}
	writeJSON(w, http.StatusOK, model.ListResponse[model.PriceList]{
		Items:  priceLists,
		Total:  total,
		Limit:  pagination.Limit,
		Offset: pagination.Offset,
	})
}

func (h *PriceListHandler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid price list ID")
		return
	}

	pl, err := h.priceListService.Get(r.Context(), tenantID, id)
	if err != nil {
		if errors.Is(err, service.ErrPriceListNotFound) {
			writeError(w, http.StatusNotFound, "price list not found")
		} else {
			writeError(w, http.StatusInternalServerError, "failed to get price list")
		}
		return
	}
	writeJSON(w, http.StatusOK, pl)
}

func (h *PriceListHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	var req model.CreatePriceListRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	pl, err := h.priceListService.Create(r.Context(), tenantID, req, actorID, clientIP(r))
	if err != nil {
		if isValidationError(err) {
			writeError(w, http.StatusBadRequest, err.Error())
		} else {
			writeError(w, http.StatusInternalServerError, "failed to create price list")
		}
		return
	}
	writeJSON(w, http.StatusCreated, pl)
}

func (h *PriceListHandler) Update(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid price list ID")
		return
	}

	var req model.UpdatePriceListRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	pl, err := h.priceListService.Update(r.Context(), tenantID, id, req, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrPriceListNotFound):
			writeError(w, http.StatusNotFound, "price list not found")
		default:
			if isValidationError(err) {
				writeError(w, http.StatusBadRequest, err.Error())
			} else {
				writeError(w, http.StatusInternalServerError, "failed to update price list")
			}
		}
		return
	}
	writeJSON(w, http.StatusOK, pl)
}

func (h *PriceListHandler) Delete(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid price list ID")
		return
	}

	err = h.priceListService.Delete(r.Context(), tenantID, id, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrPriceListNotFound):
			writeError(w, http.StatusNotFound, "price list not found")
		default:
			writeError(w, http.StatusInternalServerError, "failed to delete price list")
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Price List Items ---

func (h *PriceListHandler) ListItems(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	pagination := model.ParsePagination(r)

	priceListID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid price list ID")
		return
	}

	items, total, err := h.priceListService.ListItems(r.Context(), tenantID, priceListID, pagination.Limit, pagination.Offset)
	if err != nil {
		if errors.Is(err, service.ErrPriceListNotFound) {
			writeError(w, http.StatusNotFound, "price list not found")
		} else {
			writeError(w, http.StatusInternalServerError, "failed to list price list items")
		}
		return
	}
	if items == nil {
		items = []model.PriceListItem{}
	}
	writeJSON(w, http.StatusOK, model.ListResponse[model.PriceListItem]{
		Items:  items,
		Total:  total,
		Limit:  pagination.Limit,
		Offset: pagination.Offset,
	})
}

func (h *PriceListHandler) CreateItem(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	priceListID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid price list ID")
		return
	}

	var req model.CreatePriceListItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	item, err := h.priceListService.CreateItem(r.Context(), tenantID, priceListID, req, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrPriceListNotFound):
			writeError(w, http.StatusNotFound, "price list not found")
		default:
			if isValidationError(err) {
				writeError(w, http.StatusBadRequest, err.Error())
			} else {
				writeError(w, http.StatusInternalServerError, "failed to create price list item")
			}
		}
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (h *PriceListHandler) DeleteItem(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	itemID, err := uuid.Parse(chi.URLParam(r, "itemId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid item ID")
		return
	}

	err = h.priceListService.DeleteItem(r.Context(), tenantID, itemID, actorID, clientIP(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete price list item")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
