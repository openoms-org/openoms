package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// AIHandler handles AI-assisted product categorization requests.
type AIHandler struct {
	aiService *service.AIService
}

// NewAIHandler creates a new AIHandler.
func NewAIHandler(aiService *service.AIService) *AIHandler {
	return &AIHandler{aiService: aiService}
}

// Categorize handles POST /v1/ai/categorize
func (h *AIHandler) Categorize(w http.ResponseWriter, r *http.Request) {
	if !h.aiService.IsConfigured() {
		writeError(w, http.StatusUnprocessableEntity, "AI nie jest skonfigurowane")
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
		writeServerError(w, "AI categorization failed", err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// Describe handles POST /v1/ai/describe
func (h *AIHandler) Describe(w http.ResponseWriter, r *http.Request) {
	if !h.aiService.IsConfigured() {
		writeError(w, http.StatusUnprocessableEntity, "AI nie jest skonfigurowane")
		return
	}

	tenantID := middleware.TenantIDFromContext(r.Context())

	var req struct {
		ProductID   uuid.UUID `json:"product_id"`
		Style       string    `json:"style"`
		Language    string    `json:"language"`
		Length      string    `json:"length"`
		Marketplace string    `json:"marketplace"`
		Format      string    `json:"format"`
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
		Format:      req.Format,
	}

	result, err := h.aiService.Describe(r.Context(), tenantID, req.ProductID, opts)
	if err != nil {
		if errors.Is(err, service.ErrProductNotFound) {
			writeError(w, http.StatusNotFound, "product not found")
			return
		}
		writeServerError(w, "AI description generation failed", err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// Improve handles POST /v1/ai/improve
func (h *AIHandler) Improve(w http.ResponseWriter, r *http.Request) {
	if !h.aiService.IsConfigured() {
		writeError(w, http.StatusUnprocessableEntity, "AI nie jest skonfigurowane")
		return
	}

	var req struct {
		Description string `json:"description"`
		Style       string `json:"style"`
		Language    string `json:"language"`
		Format      string `json:"format"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Description == "" {
		writeError(w, http.StatusBadRequest, "description is required")
		return
	}

	result, err := h.aiService.ImproveDescription(r.Context(), req.Description, req.Style, req.Language, req.Format)
	if err != nil {
		writeServerError(w, "AI description improvement failed", err)
		return
	}

	writeJSON(w, http.StatusOK, service.AITextResult{Description: result})
}

// Translate handles POST /v1/ai/translate
func (h *AIHandler) Translate(w http.ResponseWriter, r *http.Request) {
	if !h.aiService.IsConfigured() {
		writeError(w, http.StatusUnprocessableEntity, "AI nie jest skonfigurowane")
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
		writeServerError(w, "AI translation failed", err)
		return
	}

	writeJSON(w, http.StatusOK, service.AITextResult{Description: result})
}

// BulkCategorize handles POST /v1/ai/bulk-categorize
func (h *AIHandler) BulkCategorize(w http.ResponseWriter, r *http.Request) {
	if !h.aiService.IsConfigured() {
		writeError(w, http.StatusUnprocessableEntity, "AI nie jest skonfigurowane")
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

	writeJSON(w, http.StatusOK, map[string]any{
		"results": results,
	})
}
