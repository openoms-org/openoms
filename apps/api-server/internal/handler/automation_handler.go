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

// AutomationHandler handles HTTP requests for automation rule management.
type AutomationHandler struct {
	automationService *service.AutomationService
}

// NewAutomationHandler creates a new AutomationHandler.
func NewAutomationHandler(automationService *service.AutomationService) *AutomationHandler {
	return &AutomationHandler{automationService: automationService}
}

// List returns a paginated list of automation rules.
func (h *AutomationHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	pagination := model.ParsePagination(r)

	filter := model.AutomationRuleListFilter{
		PaginationParams: pagination,
	}
	if s := r.URL.Query().Get("trigger_event"); s != "" {
		filter.TriggerEvent = &s
	}
	if s := r.URL.Query().Get("enabled"); s != "" {
		b, err := strconv.ParseBool(s)
		if err == nil {
			filter.Enabled = &b
		}
	}

	resp, err := h.automationService.List(r.Context(), tenantID, filter)
	if err != nil {
		writeServerError(w, "failed to list automation rules", err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// Get returns a single automation rule by ID.
func (h *AutomationHandler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	ruleID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid rule ID")
		return
	}

	rule, err := h.automationService.Get(r.Context(), tenantID, ruleID)
	if err != nil {
		if errors.Is(err, service.ErrAutomationRuleNotFound) {
			writeError(w, http.StatusNotFound, "automation rule not found")
			return
		}
		writeServerError(w, "failed to get automation rule", err)
		return
	}
	writeJSON(w, http.StatusOK, rule)
}

// Create inserts a new automation rule.
func (h *AutomationHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	var req model.CreateAutomationRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	rule, err := h.automationService.Create(r.Context(), tenantID, req)
	if err != nil {
		if isValidationError(err) {
			writeError(w, http.StatusBadRequest, err.Error())
		} else {
			writeServerError(w, "failed to create automation rule", err)
		}
		return
	}
	writeJSON(w, http.StatusCreated, rule)
}

// Update modifies an existing automation rule.
func (h *AutomationHandler) Update(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	ruleID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid rule ID")
		return
	}

	var req model.UpdateAutomationRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	rule, err := h.automationService.Update(r.Context(), tenantID, ruleID, req)
	if err != nil {
		if errors.Is(err, service.ErrAutomationRuleNotFound) {
			writeError(w, http.StatusNotFound, "automation rule not found")
			return
		}
		if isValidationError(err) {
			writeError(w, http.StatusBadRequest, err.Error())
		} else {
			writeServerError(w, "failed to update automation rule", err)
		}
		return
	}
	writeJSON(w, http.StatusOK, rule)
}

// Delete removes an automation rule by ID.
func (h *AutomationHandler) Delete(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	ruleID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid rule ID")
		return
	}

	err = h.automationService.Delete(r.Context(), tenantID, ruleID)
	if err != nil {
		if errors.Is(err, service.ErrAutomationRuleNotFound) {
			writeError(w, http.StatusNotFound, "automation rule not found")
			return
		}
		writeServerError(w, "failed to delete automation rule", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GetLogs returns execution logs for an automation rule.
func (h *AutomationHandler) GetLogs(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	ruleID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid rule ID")
		return
	}

	pagination := model.ParsePagination(r)

	resp, err := h.automationService.GetLogs(r.Context(), tenantID, ruleID, pagination.Limit, pagination.Offset)
	if err != nil {
		if errors.Is(err, service.ErrAutomationRuleNotFound) {
			writeError(w, http.StatusNotFound, "automation rule not found")
			return
		}
		writeServerError(w, "failed to list automation rule logs", err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// ListDelayed returns pending delayed automation actions for the tenant.
func (h *AutomationHandler) ListDelayed(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	actions, err := h.automationService.ListDelayed(r.Context(), tenantID)
	if err != nil {
		writeServerError(w, "failed to list delayed actions", err)
		return
	}
	writeJSON(w, http.StatusOK, actions)
}

// TestRule runs an automation rule against a sample payload without persisting changes.
func (h *AutomationHandler) TestRule(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	ruleID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid rule ID")
		return
	}

	var req model.TestAutomationRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Data == nil {
		req.Data = map[string]any{}
	}

	result, err := h.automationService.TestRule(r.Context(), tenantID, ruleID, req.Data)
	if err != nil {
		if errors.Is(err, service.ErrAutomationRuleNotFound) {
			writeError(w, http.StatusNotFound, "automation rule not found")
			return
		}
		if isValidationError(err) {
			writeError(w, http.StatusBadRequest, err.Error())
		} else {
			writeServerError(w, "failed to test automation rule", err)
		}
		return
	}
	writeJSON(w, http.StatusOK, result)
}
