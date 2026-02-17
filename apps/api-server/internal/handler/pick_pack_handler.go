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

// PickPackHandler handles HTTP requests for pick-and-pack workflow.
type PickPackHandler struct {
	svc *service.PickPackService
}

// NewPickPackHandler creates a new PickPackHandler.
func NewPickPackHandler(svc *service.PickPackService) *PickPackHandler {
	return &PickPackHandler{svc: svc}
}

// CreateSession creates a new pick-pack session.
func (h *PickPackHandler) CreateSession(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	var req model.CreatePickPackSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	session, err := h.svc.CreateSession(r.Context(), tenantID, req, actorID, clientIP(r))
	if err != nil {
		switch {
		case isValidationError(err):
			writeError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, service.ErrOrderNotProcessing):
			writeError(w, http.StatusConflict, err.Error())
		case service.IsForeignKeyError(err):
			writeError(w, http.StatusBadRequest, "referenced order does not exist")
		default:
			writeError(w, http.StatusInternalServerError, "failed to create pick-pack session")
		}
		return
	}
	writeJSON(w, http.StatusCreated, session)
}

// ListSessions returns pick-pack sessions for the current tenant.
func (h *PickPackHandler) ListSessions(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	pagination := model.ParsePagination(r)

	filter := model.PickPackSessionListFilter{
		PaginationParams: pagination,
	}
	if st := r.URL.Query().Get("status"); st != "" {
		filter.Status = &st
	}
	if uid := r.URL.Query().Get("assigned_to"); uid != "" {
		id, err := uuid.Parse(uid)
		if err == nil {
			filter.AssignedTo = &id
		}
	}

	resp, err := h.svc.ListSessions(r.Context(), tenantID, filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list pick-pack sessions")
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// GetSession retrieves a single session by ID with items.
func (h *PickPackHandler) GetSession(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	sessionID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid session ID")
		return
	}

	session, err := h.svc.GetSession(r.Context(), tenantID, sessionID)
	if err != nil {
		if errors.Is(err, service.ErrPickPackSessionNotFound) {
			writeError(w, http.StatusNotFound, "session not found")
		} else {
			writeError(w, http.StatusInternalServerError, "failed to get session")
		}
		return
	}
	writeJSON(w, http.StatusOK, session)
}

// ScanItem processes a barcode scan during picking.
func (h *PickPackHandler) ScanItem(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	sessionID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid session ID")
		return
	}

	var req model.ScanItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.svc.ScanItem(r.Context(), tenantID, sessionID, req, actorID, clientIP(r))
	if err != nil {
		switch {
		case isValidationError(err):
			writeError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, service.ErrPickPackSessionNotFound):
			writeError(w, http.StatusNotFound, "session not found")
		case errors.Is(err, service.ErrPickPackNotPicking):
			writeError(w, http.StatusConflict, "session is not in picking phase")
		case errors.Is(err, service.ErrPickPackBarcodeNoMatch):
			writeError(w, http.StatusUnprocessableEntity, "barcode does not match any item in this session")
		case errors.Is(err, service.ErrPickPackQuantityExceeded):
			writeError(w, http.StatusConflict, "all items with this SKU have already been picked")
		default:
			writeError(w, http.StatusInternalServerError, "failed to process scan")
		}
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// MoveToPacking transitions the session from picking to packing.
func (h *PickPackHandler) MoveToPacking(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	sessionID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid session ID")
		return
	}

	session, err := h.svc.MoveToPacking(r.Context(), tenantID, sessionID, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrPickPackSessionNotFound):
			writeError(w, http.StatusNotFound, "session not found")
		case errors.Is(err, service.ErrPickPackNotPicking):
			writeError(w, http.StatusConflict, "session is not in picking phase")
		case errors.Is(err, service.ErrPickPackNotAllPicked):
			writeError(w, http.StatusConflict, "not all items have been picked")
		default:
			writeError(w, http.StatusInternalServerError, "failed to move to packing")
		}
		return
	}
	writeJSON(w, http.StatusOK, session)
}

// MarkItemPacked marks an item as packed.
func (h *PickPackHandler) MarkItemPacked(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	sessionID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid session ID")
		return
	}

	itemID, err := uuid.Parse(chi.URLParam(r, "itemId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid item ID")
		return
	}

	var req model.MarkPackedRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	item, err := h.svc.MarkItemPacked(r.Context(), tenantID, sessionID, itemID, req, actorID, clientIP(r))
	if err != nil {
		switch {
		case isValidationError(err):
			writeError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, service.ErrPickPackSessionNotFound):
			writeError(w, http.StatusNotFound, "session not found")
		case errors.Is(err, service.ErrPickPackNotPacking):
			writeError(w, http.StatusConflict, "session is not in packing phase")
		case errors.Is(err, service.ErrPickPackItemNotFound):
			writeError(w, http.StatusNotFound, "item not found")
		case errors.Is(err, service.ErrPickPackQuantityExceeded):
			writeError(w, http.StatusConflict, "quantity would exceed required amount")
		default:
			writeError(w, http.StatusInternalServerError, "failed to mark item packed")
		}
		return
	}
	writeJSON(w, http.StatusOK, item)
}

// CompleteSession completes a session and transitions orders to packed status.
func (h *PickPackHandler) CompleteSession(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	sessionID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid session ID")
		return
	}

	session, err := h.svc.CompleteSession(r.Context(), tenantID, sessionID, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrPickPackSessionNotFound):
			writeError(w, http.StatusNotFound, "session not found")
		case errors.Is(err, service.ErrPickPackNotPacking):
			writeError(w, http.StatusConflict, "session is not in packing phase")
		case errors.Is(err, service.ErrPickPackNotAllPacked):
			writeError(w, http.StatusConflict, "not all items have been packed")
		default:
			writeError(w, http.StatusInternalServerError, "failed to complete session")
		}
		return
	}
	writeJSON(w, http.StatusOK, session)
}

// CancelSession cancels a session.
func (h *PickPackHandler) CancelSession(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	sessionID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid session ID")
		return
	}

	session, err := h.svc.CancelSession(r.Context(), tenantID, sessionID, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrPickPackSessionNotFound):
			writeError(w, http.StatusNotFound, "session not found")
		case errors.Is(err, service.ErrPickPackNotActive):
			writeError(w, http.StatusConflict, "session is already completed or cancelled")
		default:
			writeError(w, http.StatusInternalServerError, "failed to cancel session")
		}
		return
	}
	writeJSON(w, http.StatusOK, session)
}
