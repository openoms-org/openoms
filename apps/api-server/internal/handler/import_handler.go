package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/google/uuid"
	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

type csvImportService interface {
	ParseCSV(file io.Reader) (*model.ImportPreviewResponse, error)
	ImportOrders(
		ctx context.Context,
		tenantID uuid.UUID,
		file io.Reader,
		mappings []model.ImportColumnMapping,
		userID uuid.UUID,
		ip string,
		opts service.ImportOrdersOptions,
	) (*model.ImportResult, error)
}

// ImportHandler handles CSV import endpoints for orders.
type ImportHandler struct {
	importService           csvImportService
	baseLinkerImportService *service.BaseLinkerImportService
}

// NewImportHandler creates a new ImportHandler.
func NewImportHandler(importService csvImportService, blImportService *service.BaseLinkerImportService) *ImportHandler {
	return &ImportHandler{
		importService:           importService,
		baseLinkerImportService: blImportService,
	}
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
	defer func() { _ = r.MultipartForm.RemoveAll() }()

	file, _, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing file field")
		return
	}
	defer func() { _ = file.Close() }()

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

	var opts service.ImportOrdersOptions
	if limits := middleware.PlanLimitsFromContext(r.Context()); limits != nil && limits.MaxOrdersMonthly > 0 {
		opts.MaxOrdersMonthly = limits.MaxOrdersMonthly
	}

	// 10MB limit
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "file too large or invalid multipart form")
		return
	}
	defer func() { _ = r.MultipartForm.RemoveAll() }()

	file, _, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing file field")
		return
	}
	defer func() { _ = file.Close() }()

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

	result, err := h.importService.ImportOrders(r.Context(), tenantID, file, mappings, userID, clientIP(r), opts)
	if err != nil {
		if errors.Is(err, service.ErrOrderLimitExceeded) {
			writeError(w, http.StatusForbidden, fmt.Sprintf(
				"Monthly order limit reached for current plan (max: %d). Upgrade to increase.", opts.MaxOrdersMonthly))
		} else {
			writeServerError(w, "failed to import orders", err)
		}
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// BaseLinkerPreview handles POST /v1/import/baselinker/orders/preview
func (h *ImportHandler) BaseLinkerPreview(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "file too large or invalid multipart form")
		return
	}
	defer func() { _ = r.MultipartForm.RemoveAll() }()

	file, _, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing file field")
		return
	}
	defer func() { _ = file.Close() }()

	tenantID := middleware.TenantIDFromContext(r.Context())
	preview, err := h.baseLinkerImportService.PreviewOrders(r.Context(), tenantID, file)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, preview)
}

// BaseLinkerImport handles POST /v1/import/baselinker/orders
func (h *ImportHandler) BaseLinkerImport(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	userID := middleware.UserIDFromContext(r.Context())

	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "file too large or invalid multipart form")
		return
	}
	defer func() { _ = r.MultipartForm.RemoveAll() }()

	file, _, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing file field")
		return
	}
	defer func() { _ = file.Close() }()

	// Check optional "import_customers" form field
	importCustomers := r.FormValue("import_customers") == "true"

	result, err := h.baseLinkerImportService.ImportOrders(r.Context(), tenantID, file, userID, clientIP(r), importCustomers)
	if err != nil {
		writeServerError(w, "failed to import BaseLinker orders", err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
