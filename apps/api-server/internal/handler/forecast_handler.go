package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// ForecastHandler handles demand forecasting and reorder recommendation endpoints.
type ForecastHandler struct {
	forecastService *service.ForecastService
}

// NewForecastHandler creates a new ForecastHandler.
func NewForecastHandler(forecastService *service.ForecastService) *ForecastHandler {
	return &ForecastHandler{forecastService: forecastService}
}

// ListForecasts handles GET /v1/forecast/products — all product forecasts.
func (h *ForecastHandler) ListForecasts(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	daysAhead := 0 // 0 means use config default
	if v := r.URL.Query().Get("days"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 && parsed <= 365 {
			daysAhead = parsed
		}
	}

	forecasts, err := h.forecastService.ForecastAll(r.Context(), tenantID, daysAhead)
	if err != nil {
		writeServerError(w, "failed to generate forecasts", err)
		return
	}
	writeJSON(w, http.StatusOK, forecasts)
}

// GetForecast handles GET /v1/forecast/products/{id} — single product forecast.
func (h *ForecastHandler) GetForecast(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	productID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid product ID")
		return
	}

	daysAhead := 0
	if v := r.URL.Query().Get("days"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 && parsed <= 365 {
			daysAhead = parsed
		}
	}

	forecast, err := h.forecastService.ForecastDemand(r.Context(), tenantID, productID, daysAhead)
	if err != nil {
		if errors.Is(err, service.ErrProductNotFound) {
			writeError(w, http.StatusNotFound, "product not found")
			return
		}
		writeServerError(w, "failed to generate forecast", err)
		return
	}
	writeJSON(w, http.StatusOK, forecast)
}

// GetReorderRecommendations handles GET /v1/forecast/reorder — reorder suggestions.
func (h *ForecastHandler) GetReorderRecommendations(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	recs, err := h.forecastService.GetReorderRecommendations(r.Context(), tenantID)
	if err != nil {
		writeServerError(w, "failed to generate reorder recommendations", err)
		return
	}
	writeJSON(w, http.StatusOK, recs)
}

// GetSeasonality handles GET /v1/forecast/seasonality/{product_id} — seasonality analysis.
func (h *ForecastHandler) GetSeasonality(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	productID, err := uuid.Parse(chi.URLParam(r, "product_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid product ID")
		return
	}

	data, err := h.forecastService.GetSeasonalityAnalysis(r.Context(), tenantID, productID)
	if err != nil {
		if errors.Is(err, service.ErrProductNotFound) {
			writeError(w, http.StatusNotFound, "product not found")
			return
		}
		writeServerError(w, "failed to generate seasonality analysis", err)
		return
	}
	writeJSON(w, http.StatusOK, data)
}

// GetVelocity handles GET /v1/forecast/velocity — ABC analysis / product velocity.
func (h *ForecastHandler) GetVelocity(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	velocity, err := h.forecastService.GetProductVelocity(r.Context(), tenantID)
	if err != nil {
		writeServerError(w, "failed to generate velocity analysis", err)
		return
	}
	writeJSON(w, http.StatusOK, velocity)
}

// GetConfig handles GET /v1/forecast/config — get forecast configuration.
func (h *ForecastHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	cfg, err := h.forecastService.GetForecastConfig(r.Context(), tenantID)
	if err != nil {
		writeServerError(w, "failed to retrieve forecast configuration", err)
		return
	}
	writeJSON(w, http.StatusOK, cfg)
}

// UpdateConfig handles PUT /v1/forecast/config — update forecast configuration.
func (h *ForecastHandler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	var cfg model.ForecastConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		writeError(w, http.StatusBadRequest, "invalid input data")
		return
	}

	// Validate ranges
	if cfg.DefaultLeadTimeDays < 1 || cfg.DefaultLeadTimeDays > 365 {
		writeError(w, http.StatusBadRequest, "lead time must be between 1 and 365 days")
		return
	}
	if cfg.SafetyStockDays < 0 || cfg.SafetyStockDays > 365 {
		writeError(w, http.StatusBadRequest, "safety stock must be between 0 and 365 days")
		return
	}
	if cfg.ForecastDaysAhead < 1 || cfg.ForecastDaysAhead > 365 {
		writeError(w, http.StatusBadRequest, "forecast horizon must be between 1 and 365 days")
		return
	}

	if err := h.forecastService.UpdateForecastConfig(r.Context(), tenantID, cfg); err != nil {
		writeServerError(w, "failed to update forecast configuration", err)
		return
	}
	writeJSON(w, http.StatusOK, cfg)
}
