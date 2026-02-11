package handler

import (
	"encoding/json"
	"net/http"

	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// ImportHandler handles CSV import endpoints for orders.
type ImportHandler struct {
	importService *service.ImportService
}

// NewImportHandler creates a new ImportHandler.
func NewImportHandler(importService *service.ImportService) *ImportHandler {
	return &ImportHandler{importService: importService}
}

// Preview handles POST /v1/orders/import/preview
// Accepts multipart form with a "file" field (CSV).
func (h *ImportHandler) Preview(w http.ResponseWriter, r *http.Request) {
	// 10MB limit
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "file too large or invalid multipart form")
		return
	}
	defer r.MultipartForm.RemoveAll()

	file, _, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing file field")
		return
	}
	defer file.Close()

	preview, err := h.importService.ParseCSV(file)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, preview)
}

// Import handles POST /v1/orders/import
// Accepts multipart form with "file" (CSV) and "mappings" (JSON string) fields.
func (h *ImportHandler) Import(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	userID := middleware.UserIDFromContext(r.Context())

	// 10MB limit
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "file too large or invalid multipart form")
		return
	}
	defer r.MultipartForm.RemoveAll()

	file, _, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing file field")
		return
	}
	defer file.Close()

	// Parse mappings from form field
	mappingsStr := r.FormValue("mappings")
	if mappingsStr == "" {
		writeError(w, http.StatusBadRequest, "missing mappings field")
		return
	}

	var mappings []model.ImportColumnMapping
	if err := json.Unmarshal([]byte(mappingsStr), &mappings); err != nil {
		writeError(w, http.StatusBadRequest, "invalid mappings JSON")
		return
	}

	result, err := h.importService.ImportOrders(r.Context(), tenantID, file, mappings, userID, clientIP(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}
