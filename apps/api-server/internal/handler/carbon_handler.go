package handler

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// CarbonHandler handles carbon footprint API endpoints.
type CarbonHandler struct {
	carbonService *service.CarbonService
}

// NewCarbonHandler creates a new CarbonHandler.
func NewCarbonHandler(carbonService *service.CarbonService) *CarbonHandler {
	return &CarbonHandler{carbonService: carbonService}
}

// GetStats returns aggregate carbon statistics.
// Route: GET /v1/carbon/stats?date_from=2026-01-01&date_to=2026-02-28
func (h *CarbonHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	dateFrom := r.URL.Query().Get("date_from")
	dateTo := r.URL.Query().Get("date_to")

	stats, err := h.carbonService.GetCarbonStats(r.Context(), tenantID, dateFrom, dateTo)
	if err != nil {
		writeServerError(w, "failed to retrieve carbon stats", err)
		return
	}

	writeJSON(w, http.StatusOK, stats)
}

// GetReport returns a CSV report of carbon footprint per shipment.
// Route: GET /v1/carbon/report?date_from=2026-01-01&date_to=2026-02-28
func (h *CarbonHandler) GetReport(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	dateFrom := r.URL.Query().Get("date_from")
	dateTo := r.URL.Query().Get("date_to")

	rows, err := h.carbonService.GetCarbonCSVData(r.Context(), tenantID, dateFrom, dateTo)
	if err != nil {
		writeServerError(w, "failed to generate carbon report", err)
		return
	}

	writeCSVHeaders(w, "carbon-report.csv")

	// Header
	_, _ = fmt.Fprintln(w, "Shipment ID;Order ID;Carrier;Tracking Number;Weight (kg);CO2 (kg);Distance (km);Method;Status;Created At")

	for _, row := range rows {
		trackingNumber := ""
		if row.TrackingNumber != nil {
			trackingNumber = *row.TrackingNumber
		}
		weight := ""
		if row.Weight != nil {
			weight = fmt.Sprintf("%.2f", *row.Weight)
		}
		carbonKg := ""
		if row.CarbonKg != nil {
			carbonKg = fmt.Sprintf("%.3f", *row.CarbonKg)
		}
		distanceKm := ""
		if row.DistanceKm != nil {
			distanceKm = fmt.Sprintf("%.1f", *row.DistanceKm)
		}
		carbonMethod := ""
		if row.CarbonMethod != nil {
			carbonMethod = *row.CarbonMethod
		}

		line := strings.Join([]string{
			row.ShipmentID,
			row.OrderID,
			row.Provider,
			trackingNumber,
			weight,
			carbonKg,
			distanceKm,
			carbonMethod,
			row.Status,
			row.CreatedAt,
		}, ";")
		_, _ = fmt.Fprintln(w, line)
	}
}
