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

type ReturnHandler struct {
	returnService *service.ReturnService
}

func NewReturnHandler(returnService *service.ReturnService) *ReturnHandler {
	return &ReturnHandler{returnService: returnService}
}

func (h *ReturnHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	pagination := model.ParsePagination(r)

	filter := model.ReturnListFilter{
		PaginationParams: pagination,
	}
	if s := r.URL.Query().Get("status"); s != "" {
		filter.Status = &s
	}
	if s := r.URL.Query().Get("order_id"); s != "" {
		id, err := uuid.Parse(s)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid order_id filter")
			return
		}
		filter.OrderID = &id
	}

	resp, err := h.returnService.List(r.Context(), tenantID, filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list returns")
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *ReturnHandler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	returnID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid return ID")
		return
	}

	ret, err := h.returnService.Get(r.Context(), tenantID, returnID)
	if err != nil {
		if errors.Is(err, service.ErrReturnNotFound) {
			writeError(w, http.StatusNotFound, "return not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get return")
		return
	}
	writeJSON(w, http.StatusOK, ret)
}

func (h *ReturnHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	var req model.CreateReturnRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	ret, err := h.returnService.Create(r.Context(), tenantID, req, actorID, clientIP(r))
	if err != nil {
		if isValidationError(err) {
			writeError(w, http.StatusBadRequest, err.Error())
		} else {
			writeError(w, http.StatusInternalServerError, "failed to create return")
		}
		return
	}
	writeJSON(w, http.StatusCreated, ret)
}

func (h *ReturnHandler) Update(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	returnID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid return ID")
		return
	}

	var req model.UpdateReturnRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	ret, err := h.returnService.Update(r.Context(), tenantID, returnID, req, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrReturnNotFound):
			writeError(w, http.StatusNotFound, "return not found")
		default:
			if isValidationError(err) {
				writeError(w, http.StatusBadRequest, err.Error())
			} else {
				writeError(w, http.StatusInternalServerError, "failed to update return")
			}
		}
		return
	}
	writeJSON(w, http.StatusOK, ret)
}

func (h *ReturnHandler) TransitionStatus(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	returnID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid return ID")
		return
	}

	var req model.ReturnStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	ret, err := h.returnService.TransitionStatus(r.Context(), tenantID, returnID, req, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrReturnNotFound):
			writeError(w, http.StatusNotFound, "return not found")
		case errors.Is(err, service.ErrInvalidReturnTransition):
			writeError(w, http.StatusUnprocessableEntity, err.Error())
		default:
			if isValidationError(err) {
				writeError(w, http.StatusBadRequest, err.Error())
			} else {
				writeError(w, http.StatusInternalServerError, "failed to transition return status")
			}
		}
		return
	}
	writeJSON(w, http.StatusOK, ret)
}

func (h *ReturnHandler) Delete(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	returnID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid return ID")
		return
	}

	err = h.returnService.Delete(r.Context(), tenantID, returnID, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrReturnNotFound):
			writeError(w, http.StatusNotFound, "return not found")
		default:
			writeError(w, http.StatusInternalServerError, "failed to delete return")
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
