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

// MessageTemplateHandler handles HTTP requests for message template management.
type MessageTemplateHandler struct {
	svc *service.MessageTemplateService
}

// NewMessageTemplateHandler creates a new MessageTemplateHandler.
func NewMessageTemplateHandler(svc *service.MessageTemplateService) *MessageTemplateHandler {
	return &MessageTemplateHandler{svc: svc}
}

// List returns all message templates for the tenant.
func (h *MessageTemplateHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	pagination := model.ParsePagination(r)

	filter := model.MessageTemplateListFilter{
		PaginationParams: pagination,
	}
	if channel := r.URL.Query().Get("channel"); channel != "" {
		filter.Channel = &channel
	}
	if enabled := r.URL.Query().Get("enabled"); enabled == "true" {
		v := true
		filter.Enabled = &v
	} else if enabled == "false" {
		v := false
		filter.Enabled = &v
	}

	templates, total, err := h.svc.List(r.Context(), tenantID, filter)
	if err != nil {
		writeServerError(w, "failed to list message templates", err)
		return
	}
	if templates == nil {
		templates = []model.MessageTemplate{}
	}
	writeJSON(w, http.StatusOK, model.ListResponse[model.MessageTemplate]{
		Items:  templates,
		Total:  total,
		Limit:  pagination.Limit,
		Offset: pagination.Offset,
	})
}

// Get returns a single message template by ID.
func (h *MessageTemplateHandler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid message template ID")
		return
	}

	t, err := h.svc.Get(r.Context(), tenantID, id)
	if err != nil {
		if errors.Is(err, service.ErrMessageTemplateNotFound) {
			writeError(w, http.StatusNotFound, "message template not found")
		} else {
			writeServerError(w, "failed to get message template", err)
		}
		return
	}
	writeJSON(w, http.StatusOK, t)
}

// Create inserts a new message template.
func (h *MessageTemplateHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	var req model.CreateMessageTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	t, err := h.svc.Create(r.Context(), tenantID, req)
	if err != nil {
		if isValidationError(err) {
			writeError(w, http.StatusBadRequest, err.Error())
		} else {
			writeServerError(w, "failed to create message template", err)
		}
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

// Update modifies an existing message template.
func (h *MessageTemplateHandler) Update(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid message template ID")
		return
	}

	var req model.UpdateMessageTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	t, err := h.svc.Update(r.Context(), tenantID, id, req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrMessageTemplateNotFound):
			writeError(w, http.StatusNotFound, "message template not found")
		default:
			if isValidationError(err) {
				writeError(w, http.StatusBadRequest, err.Error())
			} else {
				writeServerError(w, "failed to update message template", err)
			}
		}
		return
	}
	writeJSON(w, http.StatusOK, t)
}

// Delete removes a message template by ID.
func (h *MessageTemplateHandler) Delete(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid message template ID")
		return
	}

	err = h.svc.Delete(r.Context(), tenantID, id)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrMessageTemplateNotFound):
			writeError(w, http.StatusNotFound, "message template not found")
		default:
			writeServerError(w, "failed to delete message template", err)
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
