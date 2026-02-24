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

type RepricingHandler struct {
	repricingService *service.RepricingService
}

func NewRepricingHandler(repricingService *service.RepricingService) *RepricingHandler {
	return &RepricingHandler{repricingService: repricingService}
}

func (h *RepricingHandler) ListRules(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	pagination := model.ParsePagination(r)

	filter := model.RepricingRuleListFilter{
		PaginationParams: pagination,
	}
	if s := r.URL.Query().Get("status"); s != "" {
		filter.Status = &s
	}
	if s := r.URL.Query().Get("strategy"); s != "" {
		filter.Strategy = &s
	}

	resp, err := h.repricingService.List(r.Context(), tenantID, filter)
	if err != nil {
		writeServerError(w, "failed to list repricing rules", err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *RepricingHandler) GetRule(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	ruleID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid rule ID")
		return
	}

	rule, err := h.repricingService.Get(r.Context(), tenantID, ruleID)
	if err != nil {
		if errors.Is(err, service.ErrRepricingRuleNotFound) {
			writeError(w, http.StatusNotFound, "repricing rule not found")
			return
		}
		writeServerError(w, "failed to get repricing rule", err)
		return
	}
	writeJSON(w, http.StatusOK, rule)
}

func (h *RepricingHandler) CreateRule(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	var req model.CreateRepricingRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	rule, err := h.repricingService.Create(r.Context(), tenantID, req, actorID, clientIP(r))
	if err != nil {
		if isValidationError(err) {
			writeError(w, http.StatusBadRequest, err.Error())
		} else {
			writeServerError(w, "failed to create repricing rule", err)
		}
		return
	}
	writeJSON(w, http.StatusCreated, rule)
}

func (h *RepricingHandler) UpdateRule(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	ruleID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid rule ID")
		return
	}

	var req model.UpdateRepricingRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	rule, err := h.repricingService.Update(r.Context(), tenantID, ruleID, req, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrRepricingRuleNotFound):
			writeError(w, http.StatusNotFound, "repricing rule not found")
		default:
			if isValidationError(err) {
				writeError(w, http.StatusBadRequest, err.Error())
			} else {
				writeServerError(w, "failed to update repricing rule", err)
			}
		}
		return
	}
	writeJSON(w, http.StatusOK, rule)
}

func (h *RepricingHandler) DeleteRule(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	ruleID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid rule ID")
		return
	}

	err = h.repricingService.Delete(r.Context(), tenantID, ruleID, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrRepricingRuleNotFound):
			writeError(w, http.StatusNotFound, "repricing rule not found")
		default:
			writeServerError(w, "failed to delete repricing rule", err)
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *RepricingHandler) SimulateRule(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	ruleID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid rule ID")
		return
	}

	results, err := h.repricingService.SimulateRule(r.Context(), tenantID, ruleID)
	if err != nil {
		if errors.Is(err, service.ErrRepricingRuleNotFound) {
			writeError(w, http.StatusNotFound, "repricing rule not found")
			return
		}
		writeServerError(w, "failed to simulate repricing rule", err)
		return
	}
	writeJSON(w, http.StatusOK, results)
}

func (h *RepricingHandler) ApplyRules(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	result, err := h.repricingService.ApplyRules(r.Context(), tenantID)
	if err != nil {
		writeServerError(w, "failed to apply repricing rules", err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *RepricingHandler) ListLog(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	// Optional filter by rule_id or product_id
	if ruleIDStr := r.URL.Query().Get("rule_id"); ruleIDStr != "" {
		ruleID, err := uuid.Parse(ruleIDStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid rule_id")
			return
		}
		logs, total, err := h.repricingService.GetLogForRule(r.Context(), tenantID, ruleID, limit, offset)
		if err != nil {
			writeServerError(w, "failed to list repricing log", err)
			return
		}
		writeJSON(w, http.StatusOK, model.ListResponse[model.RepricingLog]{
			Items: logs, Total: total, Limit: limit, Offset: offset,
		})
		return
	}

	if productIDStr := r.URL.Query().Get("product_id"); productIDStr != "" {
		productID, err := uuid.Parse(productIDStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid product_id")
			return
		}
		logs, total, err := h.repricingService.GetLogForProduct(r.Context(), tenantID, productID, limit, offset)
		if err != nil {
			writeServerError(w, "failed to list repricing log", err)
			return
		}
		writeJSON(w, http.StatusOK, model.ListResponse[model.RepricingLog]{
			Items: logs, Total: total, Limit: limit, Offset: offset,
		})
		return
	}

	resp, err := h.repricingService.ListLog(r.Context(), tenantID, limit, offset)
	if err != nil {
		writeServerError(w, "failed to list repricing log", err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *RepricingHandler) GetSummary(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	summary, err := h.repricingService.GetSummary(r.Context(), tenantID)
	if err != nil {
		writeServerError(w, "failed to get repricing summary", err)
		return
	}
	writeJSON(w, http.StatusOK, summary)
}
