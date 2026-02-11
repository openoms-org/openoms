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

type BarcodeHandler struct {
	barcodeService *service.BarcodeService
}

func NewBarcodeHandler(barcodeService *service.BarcodeService) *BarcodeHandler {
	return &BarcodeHandler{barcodeService: barcodeService}
}

// Lookup handles GET /v1/barcode/{code}
func (h *BarcodeHandler) Lookup(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	code := chi.URLParam(r, "code")
	if code == "" {
		writeError(w, http.StatusBadRequest, "barcode code is required")
		return
	}

	resp, err := h.barcodeService.Lookup(r.Context(), tenantID, code)
	if err != nil {
		if errors.Is(err, service.ErrBarcodeNotFound) {
			writeError(w, http.StatusNotFound, "no product found for this barcode")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to lookup barcode")
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// PackOrder handles POST /v1/orders/{id}/pack
func (h *BarcodeHandler) PackOrder(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	orderID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid order ID")
		return
	}

	var req model.PackOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.barcodeService.PackOrder(r.Context(), tenantID, orderID, actorID, req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrOrderNotFound):
			writeError(w, http.StatusNotFound, "order not found")
		case errors.Is(err, service.ErrPackingItemMismatch):
			writeError(w, http.StatusUnprocessableEntity, err.Error())
		default:
			if isValidationError(err) {
				writeError(w, http.StatusBadRequest, err.Error())
			} else {
				writeError(w, http.StatusInternalServerError, "failed to pack order")
			}
		}
		return
	}
	writeJSON(w, http.StatusOK, resp)
}
