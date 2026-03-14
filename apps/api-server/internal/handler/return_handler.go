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

// ReturnHandler handles HTTP requests for return management.
type ReturnHandler struct {
	returnService *service.ReturnService
}

// NewReturnHandler creates a new ReturnHandler.
func NewReturnHandler(returnService *service.ReturnService) *ReturnHandler {
	return &ReturnHandler{returnService: returnService}
}

// List returns a paginated list of returns.
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
		writeServerError(w, "failed to list returns", err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// Get returns a single return by ID.
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
		writeServerError(w, "failed to get return", err)
		return
	}
	writeJSON(w, http.StatusOK, ret)
}

// Create inserts a new return.
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
			writeServerError(w, "failed to create return", err)
		}
		return
	}
	writeJSON(w, http.StatusCreated, ret)
}

// Update modifies an existing return.
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
				writeServerError(w, "failed to update return", err)
			}
		}
		return
	}
	writeJSON(w, http.StatusOK, ret)
}

// TransitionStatus moves a return to a new status.
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
				writeServerError(w, "failed to transition return status", err)
			}
		}
		return
	}
	writeJSON(w, http.StatusOK, ret)
}

// Delete removes a return by ID.
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
			writeServerError(w, "failed to delete return", err)
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
