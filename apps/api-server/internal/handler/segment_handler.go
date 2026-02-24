package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

type SegmentHandler struct {
	segmentService *service.SegmentService
}

func NewSegmentHandler(segmentService *service.SegmentService) *SegmentHandler {
	return &SegmentHandler{segmentService: segmentService}
}

func (h *SegmentHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	pagination := model.ParsePagination(r)

	filter := model.SegmentListFilter{
		PaginationParams: pagination,
	}

	resp, err := h.segmentService.List(r.Context(), tenantID, filter)
	if err != nil {
		writeServerError(w, "failed to list segments", err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *SegmentHandler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	segmentID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid segment ID")
		return
	}

	segment, err := h.segmentService.Get(r.Context(), tenantID, segmentID)
	if err != nil {
		if errors.Is(err, service.ErrSegmentNotFound) {
			writeError(w, http.StatusNotFound, "segment not found")
		} else {
			writeServerError(w, "failed to get segment", err)
		}
		return
	}
	writeJSON(w, http.StatusOK, segment)
}

func (h *SegmentHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	var req model.CreateSegmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	segment, err := h.segmentService.Create(r.Context(), tenantID, req, actorID, clientIP(r))
	if err != nil {
		if isValidationError(err) {
			writeError(w, http.StatusBadRequest, err.Error())
		} else {
			writeServerError(w, "failed to create segment", err)
		}
		return
	}
	writeJSON(w, http.StatusCreated, segment)
}

func (h *SegmentHandler) Update(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	segmentID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid segment ID")
		return
	}

	var req model.UpdateSegmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	segment, err := h.segmentService.Update(r.Context(), tenantID, segmentID, req, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrSegmentNotFound):
			writeError(w, http.StatusNotFound, "segment not found")
		default:
			if isValidationError(err) {
				writeError(w, http.StatusBadRequest, err.Error())
			} else {
				writeServerError(w, "failed to update segment", err)
			}
		}
		return
	}
	writeJSON(w, http.StatusOK, segment)
}

func (h *SegmentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	segmentID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid segment ID")
		return
	}

	err = h.segmentService.Delete(r.Context(), tenantID, segmentID, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrSegmentNotFound):
			writeError(w, http.StatusNotFound, "segment not found")
		default:
			writeServerError(w, "failed to delete segment", err)
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *SegmentHandler) ListMembers(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	segmentID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid segment ID")
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	resp, err := h.segmentService.ListMembers(r.Context(), tenantID, segmentID, limit, offset)
	if err != nil {
		if errors.Is(err, service.ErrSegmentNotFound) {
			writeError(w, http.StatusNotFound, "segment not found")
		} else {
			writeServerError(w, "failed to list members", err)
		}
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *SegmentHandler) AddMember(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	segmentID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid segment ID")
		return
	}

	var req model.AddSegmentMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	err = h.segmentService.AddMember(r.Context(), tenantID, segmentID, req, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrSegmentNotFound):
			writeError(w, http.StatusNotFound, "segment not found")
		default:
			if isValidationError(err) {
				writeError(w, http.StatusBadRequest, err.Error())
			} else {
				writeServerError(w, "failed to add member", err)
			}
		}
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (h *SegmentHandler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	segmentID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid segment ID")
		return
	}

	customerID, err := uuid.Parse(chi.URLParam(r, "customer_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid customer ID")
		return
	}

	err = h.segmentService.RemoveMember(r.Context(), tenantID, segmentID, customerID, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrSegmentNotFound):
			writeError(w, http.StatusNotFound, "segment not found")
		default:
			writeServerError(w, "failed to remove member", err)
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *SegmentHandler) RunRFMAnalysis(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	results, err := h.segmentService.RunRFMAnalysis(r.Context(), tenantID)
	if err != nil {
		writeServerError(w, "failed to run RFM analysis", err)
		return
	}
	writeJSON(w, http.StatusOK, results)
}

func (h *SegmentHandler) GetCustomerSegments(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	customerID, err := uuid.Parse(chi.URLParam(r, "customer_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid customer ID")
		return
	}

	segments, err := h.segmentService.GetCustomerSegments(r.Context(), tenantID, customerID)
	if err != nil {
		writeServerError(w, "failed to get customer segments", err)
		return
	}
	writeJSON(w, http.StatusOK, segments)
}
