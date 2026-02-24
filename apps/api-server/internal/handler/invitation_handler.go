package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// InvitationHandler manages admin CRUD for invitations.
type InvitationHandler struct {
	svc *service.InvitationService
}

// NewInvitationHandler creates a new InvitationHandler.
func NewInvitationHandler(svc *service.InvitationService) *InvitationHandler {
	return &InvitationHandler{svc: svc}
}

// Create creates a new invitation. POST /v1/invitations
func (h *InvitationHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	userID := middleware.UserIDFromContext(r.Context())

	var req model.CreateInvitationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.svc.Create(r.Context(), tenantID, userID, req.Email, req.Role)
	if err != nil {
		if isValidationError(err) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeServerError(w, "failed to create invitation", err)
		return
	}

	writeJSON(w, http.StatusCreated, resp)
}

// List lists all invitations. GET /v1/invitations
func (h *InvitationHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	invitations, total, err := h.svc.List(r.Context(), tenantID, limit, offset)
	if err != nil {
		writeServerError(w, "failed to list invitations", err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data":  invitations,
		"total": total,
	})
}

// Delete deletes an invitation. DELETE /v1/invitations/{id}
func (h *InvitationHandler) Delete(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	userID := middleware.UserIDFromContext(r.Context())

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid invitation ID")
		return
	}

	if err := h.svc.Delete(r.Context(), tenantID, id, userID); err != nil {
		writeServerError(w, "failed to delete invitation", err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "invitation deleted"})
}
