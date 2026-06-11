package handler

import (
	"archive/zip"
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
	"github.com/openoms-org/openoms/apps/api-server/internal/readiness"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// ShipmentHandler handles HTTP requests for shipment management.
type ShipmentHandler struct {
	shipmentService *service.ShipmentService
	labelService    *service.LabelService
	surfaceMode     string
}

// NewShipmentHandler creates a new ShipmentHandler. surfaceMode is the API
// surface readiness mode (config.APISurfaceMode) used to gate carrier provider
// selection on create.
func NewShipmentHandler(shipmentService *service.ShipmentService, labelService *service.LabelService, surfaceMode string) *ShipmentHandler {
	return &ShipmentHandler{shipmentService: shipmentService, labelService: labelService, surfaceMode: surfaceMode}
}

// ListByOrder returns all shipments for a given order, sorted by package_number.
// Route: GET /v1/orders/{id}/shipments
func (h *ShipmentHandler) ListByOrder(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	orderID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid order ID")
		return
	}

	shipments, err := h.shipmentService.ListByOrder(r.Context(), tenantID, orderID)
	if err != nil {
		writeServerError(w, "failed to list order shipments", err)
		return
	}
	writeJSON(w, http.StatusOK, shipments)
}

// CreateForOrder creates a shipment for a specific order, taking order_id from the URL path.
// Route: POST /v1/orders/{id}/shipments
func (h *ShipmentHandler) CreateForOrder(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	orderID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid order ID")
		return
	}

	var req model.CreateShipmentRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	req.OrderID = orderID

	// Gate carrier provider by API surface readiness mode. "manual" shipments
	// (no carrier integration) are always allowed. An empty provider falls
	// through to the service's required-field validation (400).
	if req.Provider != "" && req.Provider != "manual" && !readiness.IsProviderEnabled(req.Provider, h.surfaceMode) {
		writeError(w, http.StatusUnprocessableEntity, "provider_not_available")
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
				writeServerError(w, "failed to create shipment", err)
			}
		}
		return
	}
	writeJSON(w, http.StatusCreated, shipment)
}

// List returns a paginated list of shipments.
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
	if ids := r.URL.Query().Get("order_ids"); ids != "" {
		for raw := range strings.SplitSeq(ids, ",") {
			if len(filter.OrderIDs) >= 100 {
				writeError(w, http.StatusBadRequest, "maximum 100 order_ids per request")
				return
			}
			id, err := uuid.Parse(strings.TrimSpace(raw))
			if err != nil {
				writeError(w, http.StatusBadRequest, "invalid order_ids filter")
				return
			}
			filter.OrderIDs = append(filter.OrderIDs, id)
		}
	}

	resp, err := h.shipmentService.List(r.Context(), tenantID, filter)
	if err != nil {
		writeServerError(w, "failed to list shipments", err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// Get returns a single shipment by ID.
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
		writeServerError(w, "failed to get shipment", err)
		return
	}
	writeJSON(w, http.StatusOK, shipment)
}

// Create inserts a new shipment.
func (h *ShipmentHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	var req model.CreateShipmentRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	// Gate carrier provider by API surface readiness mode. "manual" shipments
	// (no carrier integration) are always allowed. An empty provider falls
	// through to the service's required-field validation (400).
	if req.Provider != "" && req.Provider != "manual" && !readiness.IsProviderEnabled(req.Provider, h.surfaceMode) {
		writeError(w, http.StatusUnprocessableEntity, "provider_not_available")
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
				writeServerError(w, "failed to create shipment", err)
			}
		}
		return
	}
	writeJSON(w, http.StatusCreated, shipment)
}

// Update modifies an existing shipment.
func (h *ShipmentHandler) Update(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	shipmentID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid shipment ID")
		return
	}

	var req model.UpdateShipmentRequest
	if !decodeJSON(w, r, &req) {
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
				writeServerError(w, "failed to update shipment", err)
			}
		}
		return
	}
	writeJSON(w, http.StatusOK, shipment)
}

// Delete removes a shipment by ID.
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
			writeServerError(w, "failed to delete shipment", err)
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// TransitionStatus moves a shipment to a new status.
func (h *ShipmentHandler) TransitionStatus(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	shipmentID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid shipment ID")
		return
	}

	var req model.ShipmentStatusTransitionRequest
	if !decodeJSON(w, r, &req) {
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
				writeServerError(w, "failed to transition shipment status", err)
			}
		}
		return
	}
	writeJSON(w, http.StatusOK, shipment)
}

// GenerateLabel creates a carrier label for a shipment.
func (h *ShipmentHandler) GenerateLabel(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	shipmentID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid shipment ID")
		return
	}

	var req model.GenerateLabelRequest
	if !decodeJSON(w, r, &req) {
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
				switch {
				case strings.Contains(errMsg, "shipment payment failed") || strings.Contains(errMsg, "debt_collection"):
					writeError(w, http.StatusUnprocessableEntity, errMsg)
				case strings.Contains(errMsg, "401") || strings.Contains(errMsg, "Token is missing or invalid"):
					writeError(w, http.StatusUnprocessableEntity, "Carrier authorization error — check login credentials in integration settings")
				case strings.Contains(errMsg, "api error 4"):
					// 4xx from carrier — pass through the carrier message
					writeError(w, http.StatusUnprocessableEntity, "Carrier API error: "+errMsg)
				default:
					writeServerError(w, "Failed to generate label — try again or check integration settings", err)
				}
			}
		}
		return
	}
	writeJSON(w, http.StatusOK, shipment)
}

// GetTracking returns the tracking status from the carrier for a shipment.
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
			writeServerError(w, "failed to get tracking info", err)
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
	if !decodeJSON(w, r, &req) {
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
				writeServerError(w, "failed to create pickup order", err)
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
	if !decodeJSON(w, r, &req) {
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
		writeServerError(w, "failed to get label URLs", err)
		return
	}

	if len(labels) == 0 {
		writeError(w, http.StatusUnprocessableEntity, "no shipments with labels found")
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename=labels.zip")

	zipWriter := zip.NewWriter(w)
	defer func() {
		if err := zipWriter.Close(); err != nil {
			slog.Error("shipment: failed to close zip writer", "error", err)
		}
	}()

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
