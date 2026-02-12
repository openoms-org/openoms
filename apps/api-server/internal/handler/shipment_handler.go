package handler

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	engine "github.com/openoms-org/openoms/packages/order-engine"

	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

type ShipmentHandler struct {
	shipmentService *service.ShipmentService
	labelService    *service.LabelService
}

func NewShipmentHandler(shipmentService *service.ShipmentService, labelService *service.LabelService) *ShipmentHandler {
	return &ShipmentHandler{shipmentService: shipmentService, labelService: labelService}
}

func (h *ShipmentHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	pagination := model.ParsePagination(r)

	filter := model.ShipmentListFilter{
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

	resp, err := h.shipmentService.List(r.Context(), tenantID, filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list shipments")
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *ShipmentHandler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	shipmentID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid shipment ID")
		return
	}

	shipment, err := h.shipmentService.Get(r.Context(), tenantID, shipmentID)
	if err != nil {
		if errors.Is(err, service.ErrShipmentNotFound) {
			writeError(w, http.StatusNotFound, "shipment not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get shipment")
		return
	}
	writeJSON(w, http.StatusOK, shipment)
}

func (h *ShipmentHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	var req model.CreateShipmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	shipment, err := h.shipmentService.Create(r.Context(), tenantID, req, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrOrderNotFoundForShipment):
			writeError(w, http.StatusUnprocessableEntity, "order not found for shipment")
		default:
			if isValidationError(err) {
				writeError(w, http.StatusBadRequest, err.Error())
			} else {
				writeError(w, http.StatusInternalServerError, "failed to create shipment")
			}
		}
		return
	}
	writeJSON(w, http.StatusCreated, shipment)
}

func (h *ShipmentHandler) Update(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	shipmentID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid shipment ID")
		return
	}

	var req model.UpdateShipmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	shipment, err := h.shipmentService.Update(r.Context(), tenantID, shipmentID, req, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrShipmentNotFound):
			writeError(w, http.StatusNotFound, "shipment not found")
		default:
			if isValidationError(err) {
				writeError(w, http.StatusBadRequest, err.Error())
			} else {
				writeError(w, http.StatusInternalServerError, "failed to update shipment")
			}
		}
		return
	}
	writeJSON(w, http.StatusOK, shipment)
}

func (h *ShipmentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	shipmentID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid shipment ID")
		return
	}

	err = h.shipmentService.Delete(r.Context(), tenantID, shipmentID, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrShipmentNotFound):
			writeError(w, http.StatusNotFound, "shipment not found")
		default:
			writeError(w, http.StatusInternalServerError, "failed to delete shipment")
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ShipmentHandler) TransitionStatus(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	shipmentID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid shipment ID")
		return
	}

	var req model.ShipmentStatusTransitionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	shipment, err := h.shipmentService.TransitionStatus(r.Context(), tenantID, shipmentID, req, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrShipmentNotFound):
			writeError(w, http.StatusNotFound, "shipment not found")
		case errors.Is(err, engine.ErrInvalidTransition), errors.Is(err, engine.ErrUnknownStatus):
			writeError(w, http.StatusUnprocessableEntity, err.Error())
		case errors.Is(err, service.ErrOrderNotFoundForShipment):
			writeError(w, http.StatusUnprocessableEntity, "order not found for shipment")
		default:
			if isValidationError(err) {
				writeError(w, http.StatusBadRequest, err.Error())
			} else {
				writeError(w, http.StatusInternalServerError, "failed to transition shipment status")
			}
		}
		return
	}
	writeJSON(w, http.StatusOK, shipment)
}

func (h *ShipmentHandler) GenerateLabel(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	shipmentID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid shipment ID")
		return
	}

	var req model.GenerateLabelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	shipment, err := h.labelService.GenerateLabel(r.Context(), tenantID, shipmentID, req, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrShipmentNotFound):
			writeError(w, http.StatusNotFound, "shipment not found")
		case errors.Is(err, service.ErrShipmentNotCreated):
			writeError(w, http.StatusUnprocessableEntity, err.Error())
		case errors.Is(err, service.ErrNoCarrierIntegration):
			writeError(w, http.StatusUnprocessableEntity, err.Error())
		case errors.Is(err, service.ErrNoCustomerContact):
			writeError(w, http.StatusUnprocessableEntity, err.Error())
		default:
			if isValidationError(err) {
				writeError(w, http.StatusBadRequest, err.Error())
			} else {
				slog.Error("label generation failed", "error", err)
				// Extract carrier API error details for better user feedback
				errMsg := err.Error()
				if strings.Contains(errMsg, "opłacenie przesyłki") || strings.Contains(errMsg, "debt_collection") {
					writeError(w, http.StatusBadGateway, errMsg)
				} else if strings.Contains(errMsg, "401") || strings.Contains(errMsg, "Token is missing or invalid") {
					writeError(w, http.StatusBadGateway, "Błąd autoryzacji kuriera — sprawdź dane logowania w ustawieniach integracji")
				} else if strings.Contains(errMsg, "api error 4") {
					// 4xx from carrier — pass through the carrier message
					writeError(w, http.StatusBadGateway, "Błąd API kuriera: "+errMsg)
				} else {
					writeError(w, http.StatusInternalServerError, "Nie udało się wygenerować etykiety — spróbuj ponownie lub sprawdź ustawienia integracji")
				}
			}
		}
		return
	}
	writeJSON(w, http.StatusOK, shipment)
}

func (h *ShipmentHandler) GetTracking(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	shipmentID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid shipment ID")
		return
	}

	events, err := h.labelService.GetTracking(r.Context(), tenantID, shipmentID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrShipmentNotFound):
			writeError(w, http.StatusNotFound, "shipment not found")
		case errors.Is(err, service.ErrNoCarrierIntegration):
			writeError(w, http.StatusUnprocessableEntity, err.Error())
		default:
			slog.Error("get tracking failed", "error", err)
			writeError(w, http.StatusInternalServerError, "failed to get tracking info")
		}
		return
	}
	writeJSON(w, http.StatusOK, events)
}

// CreateDispatchOrder creates a dispatch order (courier pickup) for given shipments.
func (h *ShipmentHandler) CreateDispatchOrder(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	var req model.CreateDispatchOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.ShipmentIDs) == 0 {
		writeError(w, http.StatusBadRequest, "shipment_ids is required")
		return
	}

	resp, err := h.labelService.CreateDispatchOrder(r.Context(), tenantID, req, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrShipmentNotFound):
			writeError(w, http.StatusNotFound, "shipment not found")
		case errors.Is(err, service.ErrNoCarrierIntegration):
			writeError(w, http.StatusUnprocessableEntity, err.Error())
		default:
			if isValidationError(err) {
				writeError(w, http.StatusBadRequest, err.Error())
			} else {
				slog.Error("dispatch order creation failed", "error", err)
				writeError(w, http.StatusInternalServerError, "Nie udało się utworzyć zlecenia odbioru")
			}
		}
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}

// BatchLabels collects label files for multiple shipments and returns them as a ZIP archive.
func (h *ShipmentHandler) BatchLabels(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	var req model.BatchLabelsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.ShipmentIDs) == 0 {
		writeError(w, http.StatusBadRequest, "shipment_ids is required")
		return
	}
	if len(req.ShipmentIDs) > 100 {
		writeError(w, http.StatusBadRequest, "maximum 100 shipments per batch")
		return
	}

	labels, err := h.shipmentService.GetBatchLabelURLs(r.Context(), tenantID, req.ShipmentIDs)
	if err != nil {
		slog.Error("batch labels: failed to get label URLs", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to get label URLs")
		return
	}

	if len(labels) == 0 {
		writeError(w, http.StatusUnprocessableEntity, "no shipments with labels found")
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename=labels.zip")

	zipWriter := zip.NewWriter(w)
	defer zipWriter.Close()

	for i, label := range labels {
		filename := fmt.Sprintf("label_%s.pdf", label.ShipmentID[:8])
		if label.Data == nil {
			continue
		}

		f, err := zipWriter.Create(filename)
		if err != nil {
			slog.Error("batch labels: zip create entry", "index", i, "error", err)
			continue
		}
		if _, err := f.Write(label.Data); err != nil {
			slog.Error("batch labels: zip write entry", "index", i, "error", err)
			continue
		}
	}
}
