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

type InvoiceHandler struct {
	invoiceService *service.InvoiceService
}

func NewInvoiceHandler(invoiceService *service.InvoiceService) *InvoiceHandler {
	return &InvoiceHandler{invoiceService: invoiceService}
}

func (h *InvoiceHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	pagination := model.ParsePagination(r)

	filter := model.InvoiceListFilter{
		PaginationParams: pagination,
	}
	if s := r.URL.Query().Get("status"); s != "" {
		filter.Status = &s
	}
	if s := r.URL.Query().Get("provider"); s != "" {
		filter.Provider = &s
	}
	if s := r.URL.Query().Get("order_id"); s != "" {
		id, err := uuid.Parse(s)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid order_id filter")
			return
		}
		filter.OrderID = &id
	}

	resp, err := h.invoiceService.List(r.Context(), tenantID, filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list invoices")
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *InvoiceHandler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	invoiceID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid invoice ID")
		return
	}

	inv, err := h.invoiceService.Get(r.Context(), tenantID, invoiceID)
	if err != nil {
		if errors.Is(err, service.ErrInvoiceNotFound) {
			writeError(w, http.StatusNotFound, "invoice not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get invoice")
		return
	}
	writeJSON(w, http.StatusOK, inv)
}

func (h *InvoiceHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	var req model.CreateInvoiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	inv, err := h.invoiceService.Create(r.Context(), tenantID, req, actorID, clientIP(r))
	if err != nil {
		if isValidationError(err) {
			writeError(w, http.StatusBadRequest, err.Error())
		} else {
			writeError(w, http.StatusInternalServerError, "failed to create invoice")
		}
		return
	}
	writeJSON(w, http.StatusCreated, inv)
}

func (h *InvoiceHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	invoiceID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid invoice ID")
		return
	}

	err = h.invoiceService.Cancel(r.Context(), tenantID, invoiceID, actorID, clientIP(r))
	if err != nil {
		if errors.Is(err, service.ErrInvoiceNotFound) {
			writeError(w, http.StatusNotFound, "invoice not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to cancel invoice")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *InvoiceHandler) GetPDF(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	invoiceID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid invoice ID")
		return
	}

	pdfData, err := h.invoiceService.GetPDF(r.Context(), tenantID, invoiceID)
	if err != nil {
		if errors.Is(err, service.ErrInvoiceNotFound) {
			writeError(w, http.StatusNotFound, "invoice not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get invoice PDF")
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "attachment; filename=invoice.pdf")
	w.WriteHeader(http.StatusOK)
	w.Write(pdfData)
}

// ListByOrder returns all invoices for a specific order.
func (h *InvoiceHandler) ListByOrder(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	orderID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid order ID")
		return
	}

	invoices, err := h.invoiceService.ListByOrderID(r.Context(), tenantID, orderID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list invoices")
		return
	}
	writeJSON(w, http.StatusOK, invoices)
}
