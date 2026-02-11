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

// WarehouseDocumentHandler handles HTTP requests for warehouse documents.
type WarehouseDocumentHandler struct {
	svc *service.WarehouseDocumentService
}

// NewWarehouseDocumentHandler creates a new WarehouseDocumentHandler.
func NewWarehouseDocumentHandler(svc *service.WarehouseDocumentService) *WarehouseDocumentHandler {
	return &WarehouseDocumentHandler{svc: svc}
}

// List returns warehouse documents for the current tenant.
func (h *WarehouseDocumentHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	pagination := model.ParsePagination(r)

	filter := model.WarehouseDocumentListFilter{
		PaginationParams: pagination,
	}
	if dt := r.URL.Query().Get("document_type"); dt != "" {
		filter.DocumentType = &dt
	}
	if st := r.URL.Query().Get("status"); st != "" {
		filter.Status = &st
	}
	if wid := r.URL.Query().Get("warehouse_id"); wid != "" {
		id, err := uuid.Parse(wid)
		if err == nil {
			filter.WarehouseID = &id
		}
	}

	resp, err := h.svc.List(r.Context(), tenantID, filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list warehouse documents")
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// Get retrieves a single warehouse document by ID.
func (h *WarehouseDocumentHandler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	docID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid document ID")
		return
	}

	doc, err := h.svc.Get(r.Context(), tenantID, docID)
	if err != nil {
		if errors.Is(err, service.ErrWarehouseDocumentNotFound) {
			writeError(w, http.StatusNotFound, "warehouse document not found")
		} else {
			writeError(w, http.StatusInternalServerError, "failed to get warehouse document")
		}
		return
	}
	writeJSON(w, http.StatusOK, doc)
}

// Create creates a new warehouse document.
func (h *WarehouseDocumentHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	var req model.CreateWarehouseDocumentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	doc, err := h.svc.Create(r.Context(), tenantID, req, actorID, clientIP(r))
	if err != nil {
		if isValidationError(err) {
			writeError(w, http.StatusBadRequest, err.Error())
		} else if service.IsForeignKeyError(err) {
			writeError(w, http.StatusBadRequest, "referenced product, variant, or warehouse does not exist")
		} else {
			writeError(w, http.StatusInternalServerError, "failed to create warehouse document")
		}
		return
	}
	writeJSON(w, http.StatusCreated, doc)
}

// Update updates a warehouse document.
func (h *WarehouseDocumentHandler) Update(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	docID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid document ID")
		return
	}

	var req model.UpdateWarehouseDocumentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	doc, err := h.svc.Update(r.Context(), tenantID, docID, req, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrWarehouseDocumentNotFound):
			writeError(w, http.StatusNotFound, "warehouse document not found")
		case errors.Is(err, service.ErrDocumentNotDraft):
			writeError(w, http.StatusConflict, "document is not in draft status")
		default:
			if isValidationError(err) {
				writeError(w, http.StatusBadRequest, err.Error())
			} else {
				writeError(w, http.StatusInternalServerError, "failed to update warehouse document")
			}
		}
		return
	}
	writeJSON(w, http.StatusOK, doc)
}

// Delete deletes a warehouse document.
func (h *WarehouseDocumentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	docID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid document ID")
		return
	}

	err = h.svc.Delete(r.Context(), tenantID, docID, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrWarehouseDocumentNotFound):
			writeError(w, http.StatusNotFound, "warehouse document not found")
		case errors.Is(err, service.ErrDocumentNotDraft):
			writeError(w, http.StatusConflict, "document is not in draft status")
		default:
			writeError(w, http.StatusInternalServerError, "failed to delete warehouse document")
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Confirm confirms a warehouse document and updates stock levels.
func (h *WarehouseDocumentHandler) Confirm(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	docID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid document ID")
		return
	}

	doc, err := h.svc.Confirm(r.Context(), tenantID, docID, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrWarehouseDocumentNotFound):
			writeError(w, http.StatusNotFound, "warehouse document not found")
		case errors.Is(err, service.ErrDocumentNotDraft):
			writeError(w, http.StatusConflict, "document is not in draft status")
		case service.IsForeignKeyError(err):
			writeError(w, http.StatusBadRequest, "referenced product, variant, or warehouse does not exist")
		default:
			writeError(w, http.StatusInternalServerError, "failed to confirm warehouse document")
		}
		return
	}
	writeJSON(w, http.StatusOK, doc)
}

// Cancel cancels a warehouse document.
func (h *WarehouseDocumentHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	docID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid document ID")
		return
	}

	doc, err := h.svc.Cancel(r.Context(), tenantID, docID, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrWarehouseDocumentNotFound):
			writeError(w, http.StatusNotFound, "warehouse document not found")
		case errors.Is(err, service.ErrDocumentNotDraft):
			writeError(w, http.StatusConflict, "document is not in draft status")
		default:
			writeError(w, http.StatusInternalServerError, "failed to cancel warehouse document")
		}
		return
	}
	writeJSON(w, http.StatusOK, doc)
}
