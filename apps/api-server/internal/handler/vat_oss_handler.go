package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// VATOSSHandler handles VAT OSS API endpoints.
type VATOSSHandler struct {
	vatOSSSvc *service.VATOSSService
}

// NewVATOSSHandler creates a new VATOSSHandler.
func NewVATOSSHandler(vatOSSSvc *service.VATOSSService) *VATOSSHandler {
	return &VATOSSHandler{vatOSSSvc: vatOSSSvc}
}

// GetAllRates returns VAT rates for all EU countries.
// Route: GET /v1/vat-oss/rates
func (h *VATOSSHandler) GetAllRates(w http.ResponseWriter, _ *http.Request) {
	rates := h.vatOSSSvc.GetAllRates()
	writeJSON(w, http.StatusOK, rates)
}

// GetCountryRates returns VAT rates for a specific country.
// Route: GET /v1/vat-oss/rates/{country}
func (h *VATOSSHandler) GetCountryRates(w http.ResponseWriter, r *http.Request) {
	country := strings.ToUpper(chi.URLParam(r, "country"))
	rates, ok := h.vatOSSSvc.GetCountryRates(country)
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Sprintf("country %q not found in EU VAT rates", country))
		return
	}
	writeJSON(w, http.StatusOK, rates)
}

// Calculate calculates VAT for a given amount, country, and rate type.
// Route: POST /v1/vat-oss/calculate
func (h *VATOSSHandler) Calculate(w http.ResponseWriter, r *http.Request) {
	var req model.VATCalculateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Amount <= 0 {
		writeError(w, http.StatusBadRequest, "amount must be positive")
		return
	}

	country := strings.ToUpper(strings.TrimSpace(req.Country))
	if country == "" {
		writeError(w, http.StatusBadRequest, "country is required")
		return
	}

	if !h.vatOSSSvc.IsEUCountry(country) {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("country %q is not an EU member state", country))
		return
	}

	rateType := req.RateType
	if rateType == "" {
		rateType = "standard"
	}

	calc := h.vatOSSSvc.CalculateVAT(req.Amount, country, rateType)
	writeJSON(w, http.StatusOK, calc)
}

// GetConfig returns the tenant's OSS configuration.
// Route: GET /v1/vat-oss/config
func (h *VATOSSHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	cfg, err := h.vatOSSSvc.GetOSSConfig(r.Context(), tenantID)
	if err != nil {
		writeServerError(w, "failed to load VAT OSS config", err)
		return
	}

	writeJSON(w, http.StatusOK, cfg)
}

// UpdateConfig updates the tenant's OSS configuration.
// Route: PUT /v1/vat-oss/config
func (h *VATOSSHandler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	var cfg model.OSSConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate home country
	homeCountry := strings.ToUpper(strings.TrimSpace(cfg.HomeCountry))
	if homeCountry == "" {
		homeCountry = "PL"
	}
	if !h.vatOSSSvc.IsEUCountry(homeCountry) {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("home country %q is not an EU member state", homeCountry))
		return
	}
	cfg.HomeCountry = homeCountry

	// Validate rate type
	validRateTypes := map[string]bool{"standard": true, "reduced_1": true, "reduced_2": true, "super_reduced": true}
	if cfg.DefaultVATRate == "" {
		cfg.DefaultVATRate = "standard"
	}
	if !validRateTypes[cfg.DefaultVATRate] {
		writeError(w, http.StatusBadRequest, "invalid default VAT rate type")
		return
	}

	if err := h.vatOSSSvc.UpdateOSSConfig(r.Context(), tenantID, cfg); err != nil {
		writeServerError(w, "failed to save VAT OSS config", err)
		return
	}

	writeJSON(w, http.StatusOK, cfg)
}

// GetReport generates a quarterly OSS report.
// Route: GET /v1/vat-oss/report?quarter=1&year=2026
func (h *VATOSSHandler) GetReport(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	quarter, year, err := h.parseQuarterYear(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	report, err := h.vatOSSSvc.GenerateOSSReport(r.Context(), tenantID, quarter, year)
	if err != nil {
		writeServerError(w, "failed to generate OSS report", err)
		return
	}

	writeJSON(w, http.StatusOK, report)
}

// GetReportCSV downloads the OSS report as CSV.
// Route: GET /v1/vat-oss/report/csv?quarter=1&year=2026
func (h *VATOSSHandler) GetReportCSV(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	quarter, year, err := h.parseQuarterYear(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	report, err := h.vatOSSSvc.GenerateOSSReport(r.Context(), tenantID, quarter, year)
	if err != nil {
		writeServerError(w, "failed to generate OSS report", err)
		return
	}

	filename := fmt.Sprintf("vat-oss-report-Q%d-%d.csv", quarter, year)
	writeCSVHeaders(w, filename)

	// Header row
	_, _ = fmt.Fprintln(w, "Kraj;Nazwa kraju;Liczba zamowien;Sprzedaz (EUR);Stawka VAT (%);Kwota VAT (EUR)")

	for _, cr := range report.ByCountry {
		line := strings.Join([]string{
			cr.Country,
			cr.CountryName,
			strconv.Itoa(cr.OrderCount),
			fmt.Sprintf("%.2f", cr.TotalSales),
			fmt.Sprintf("%.1f", cr.VATRate),
			fmt.Sprintf("%.2f", cr.VATAmount),
		}, ";")
		_, _ = fmt.Fprintln(w, line)
	}

	// Summary row
	_, _ = fmt.Fprintf(w, "RAZEM;;%d;%.2f;;%.2f\n", // #nosec G705
		h.sumOrderCount(report.ByCountry),
		report.TotalSales,
		report.TotalVAT,
	)
}

// GetThreshold returns the annual cross-border threshold status.
// Route: GET /v1/vat-oss/threshold?year=2026
func (h *VATOSSHandler) GetThreshold(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	yearStr := r.URL.Query().Get("year")
	year := time.Now().Year()
	if yearStr != "" {
		var err error
		year, err = strconv.Atoi(yearStr)
		if err != nil || year < 2021 || year > 2100 {
			writeError(w, http.StatusBadRequest, "invalid year parameter")
			return
		}
	}

	status, err := h.vatOSSSvc.GetOSSThresholdStatus(r.Context(), tenantID, year)
	if err != nil {
		writeServerError(w, "failed to check threshold status", err)
		return
	}

	writeJSON(w, http.StatusOK, status)
}

// parseQuarterYear extracts and validates quarter and year query params.
func (h *VATOSSHandler) parseQuarterYear(r *http.Request) (int, int, error) {
	quarterStr := r.URL.Query().Get("quarter")
	yearStr := r.URL.Query().Get("year")

	now := time.Now()

	// Default to current quarter
	quarter := (int(now.Month())-1)/3 + 1
	year := now.Year()

	if quarterStr != "" {
		q, err := strconv.Atoi(quarterStr)
		if err != nil || q < 1 || q > 4 {
			return 0, 0, fmt.Errorf("invalid quarter: must be 1-4")
		}
		quarter = q
	}

	if yearStr != "" {
		y, err := strconv.Atoi(yearStr)
		if err != nil || y < 2021 || y > 2100 {
			return 0, 0, fmt.Errorf("invalid year: must be between 2021 and 2100")
		}
		year = y
	}

	return quarter, year, nil
}

// sumOrderCount sums order_count across all country reports.
func (h *VATOSSHandler) sumOrderCount(reports []model.OSSCountryReport) int {
	total := 0
	for _, r := range reports {
		total += r.OrderCount
	}
	return total
}
