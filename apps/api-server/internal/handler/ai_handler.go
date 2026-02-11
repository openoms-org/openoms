package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

type AIHandler struct {
	aiService *service.AIService
}

func NewAIHandler(aiService *service.AIService) *AIHandler {
	return &AIHandler{aiService: aiService}
}

// Categorize handles POST /v1/ai/categorize
func (h *AIHandler) Categorize(w http.ResponseWriter, r *http.Request) {
	if !h.aiService.IsConfigured() {
		writeError(w, http.StatusServiceUnavailable, "AI nie jest skonfigurowane")
		return
	}

	tenantID := middleware.TenantIDFromContext(r.Context())

	var req struct {
		ProductID uuid.UUID `json:"product_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.ProductID == uuid.Nil {
		writeError(w, http.StatusBadRequest, "product_id is required")
		return
	}

	result, err := h.aiService.Categorize(r.Context(), tenantID, req.ProductID)
	if err != nil {
		if errors.Is(err, service.ErrProductNotFound) {
			writeError(w, http.StatusNotFound, "product not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "AI categorization failed")
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// Describe handles POST /v1/ai/describe
func (h *AIHandler) Describe(w http.ResponseWriter, r *http.Request) {
	if !h.aiService.IsConfigured() {
		writeError(w, http.StatusServiceUnavailable, "AI nie jest skonfigurowane")
		return
	}

	tenantID := middleware.TenantIDFromContext(r.Context())

	var req struct {
		ProductID uuid.UUID `json:"product_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.ProductID == uuid.Nil {
		writeError(w, http.StatusBadRequest, "product_id is required")
		return
	}

	result, err := h.aiService.Describe(r.Context(), tenantID, req.ProductID)
	if err != nil {
		if errors.Is(err, service.ErrProductNotFound) {
			writeError(w, http.StatusNotFound, "product not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "AI description generation failed")
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// BulkCategorize handles POST /v1/ai/bulk-categorize
func (h *AIHandler) BulkCategorize(w http.ResponseWriter, r *http.Request) {
	if !h.aiService.IsConfigured() {
		writeError(w, http.StatusServiceUnavailable, "AI nie jest skonfigurowane")
		return
	}

	tenantID := middleware.TenantIDFromContext(r.Context())

	var req struct {
		ProductIDs []uuid.UUID `json:"product_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if len(req.ProductIDs) == 0 {
		writeError(w, http.StatusBadRequest, "product_ids is required")
		return
	}
	if len(req.ProductIDs) > 50 {
		writeError(w, http.StatusBadRequest, "maximum 50 products at once")
		return
	}

	type bulkResult struct {
		ProductID  uuid.UUID `json:"product_id"`
		Categories []string  `json:"categories"`
		Tags       []string  `json:"tags"`
		Error      string    `json:"error,omitempty"`
	}

	results := make([]bulkResult, 0, len(req.ProductIDs))
	for _, pid := range req.ProductIDs {
		suggestion, err := h.aiService.Categorize(r.Context(), tenantID, pid)
		if err != nil {
			results = append(results, bulkResult{
				ProductID: pid,
				Error:     err.Error(),
			})
			continue
		}
		results = append(results, bulkResult{
			ProductID:  suggestion.ProductID,
			Categories: suggestion.Categories,
			Tags:       suggestion.Tags,
		})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"results": results,
	})
}
