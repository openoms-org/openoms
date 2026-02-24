package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// WorkflowHandler handles visual workflow builder endpoints.
type WorkflowHandler struct {
	automationService *service.AutomationService
}

// NewWorkflowHandler creates a new WorkflowHandler.
func NewWorkflowHandler(automationService *service.AutomationService) *WorkflowHandler {
	return &WorkflowHandler{automationService: automationService}
}

// ListTemplates returns pre-built workflow templates.
func (h *WorkflowHandler) ListTemplates(w http.ResponseWriter, r *http.Request) {
	templates := model.GetWorkflowTemplates()
	writeJSON(w, http.StatusOK, templates)
}

// Validate validates a workflow definition without saving.
func (h *WorkflowHandler) Validate(w http.ResponseWriter, r *http.Request) {
	var req model.ValidateWorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	errs := model.ValidateWorkflowDefinition(req.Definition)
	resp := model.ValidateWorkflowResponse{
		Valid:  len(errs) == 0,
		Errors: errs,
	}
	if resp.Errors == nil {
		resp.Errors = []string{}
	}

	writeJSON(w, http.StatusOK, resp)
}

// Convert converts a visual workflow definition to flat automation rule fields.
func (h *WorkflowHandler) Convert(w http.ResponseWriter, r *http.Request) {
	var req model.ConvertWorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate first
	errs := model.ValidateWorkflowDefinition(req.Definition)
	if len(errs) > 0 {
		writeJSON(w, http.StatusBadRequest, model.ValidateWorkflowResponse{
			Valid:  false,
			Errors: errs,
		})
		return
	}

	result, err := model.WorkflowToAutomationRule(req.Definition)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// GetWorkflowForRule converts an existing automation rule to a visual workflow definition.
func (h *WorkflowHandler) GetWorkflowForRule(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	ruleID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid rule ID")
		return
	}

	rule, err := h.automationService.Get(r.Context(), tenantID, ruleID)
	if err != nil {
		writeError(w, http.StatusNotFound, "automation rule not found")
		return
	}

	def, err := model.AutomationRuleToWorkflow(*rule)
	if err != nil {
		writeServerError(w, "failed to convert rule to workflow", err)
		return
	}

	writeJSON(w, http.StatusOK, def)
}
