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
		ProductID   uuid.UUID `json:"product_id"`
		Style       string    `json:"style"`
		Language    string    `json:"language"`
		Length      string    `json:"length"`
		Marketplace string    `json:"marketplace"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.ProductID == uuid.Nil {
		writeError(w, http.StatusBadRequest, "product_id is required")
		return
	}

	opts := &service.DescribeOptions{
		Style:       req.Style,
		Language:    req.Language,
		Length:      req.Length,
		Marketplace: req.Marketplace,
	}

	result, err := h.aiService.Describe(r.Context(), tenantID, req.ProductID, opts)
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

// Improve handles POST /v1/ai/improve
func (h *AIHandler) Improve(w http.ResponseWriter, r *http.Request) {
	if !h.aiService.IsConfigured() {
		writeError(w, http.StatusServiceUnavailable, "AI nie jest skonfigurowane")
		return
	}

	var req struct {
		Description string `json:"description"`
		Style       string `json:"style"`
		Language    string `json:"language"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Description == "" {
		writeError(w, http.StatusBadRequest, "description is required")
		return
	}

	result, err := h.aiService.ImproveDescription(r.Context(), req.Description, req.Style, req.Language)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "AI description improvement failed")
		return
	}

	writeJSON(w, http.StatusOK, service.AITextResult{Description: result})
}

// Translate handles POST /v1/ai/translate
func (h *AIHandler) Translate(w http.ResponseWriter, r *http.Request) {
	if !h.aiService.IsConfigured() {
		writeError(w, http.StatusServiceUnavailable, "AI nie jest skonfigurowane")
		return
	}

	var req struct {
		Description    string `json:"description"`
		TargetLanguage string `json:"target_language"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Description == "" {
		writeError(w, http.StatusBadRequest, "description is required")
		return
	}
	if req.TargetLanguage == "" {
		writeError(w, http.StatusBadRequest, "target_language is required")
		return
	}

	result, err := h.aiService.TranslateDescription(r.Context(), req.Description, req.TargetLanguage)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "AI translation failed")
		return
	}

	writeJSON(w, http.StatusOK, service.AITextResult{Description: result})
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
