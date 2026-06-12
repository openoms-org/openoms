package handler

import (
	"net/http"
	"strconv"

	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// StatsHandler handles HTTP requests for dashboard statistics.
type StatsHandler struct {
	statsService *service.StatsService
}

// NewStatsHandler creates a new StatsHandler.
func NewStatsHandler(statsService *service.StatsService) *StatsHandler {
	return &StatsHandler{statsService: statsService}
}

// GetDashboard returns aggregated dashboard statistics.
func (h *StatsHandler) GetDashboard(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	stats, err := h.statsService.GetDashboardStats(r.Context(), tenantID)
	if err != nil {
		writeServerError(w, "failed to retrieve dashboard stats", err)
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

// GetTopProducts returns the best-selling products for the tenant.
func (h *StatsHandler) GetTopProducts(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	limit := 10
	if v := r.URL.Query().Get("limit"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	// Trailing window for the ranking; bounds the per-order items JSONB expansion.
	// Defaults to 90 days and is capped at 5 years to keep the scan bounded.
	days := 90
	if v := r.URL.Query().Get("days"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 && parsed <= 1825 {
			days = parsed
		}
	}

	products, err := h.statsService.GetTopProducts(r.Context(), tenantID, days, limit)
	if err != nil {
		writeServerError(w, "failed to retrieve top products", err)
		return
	}
	writeJSON(w, http.StatusOK, products)
}

// GetRevenueBySource returns revenue broken down by sales channel.
func (h *StatsHandler) GetRevenueBySource(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	days := 30
	if v := r.URL.Query().Get("days"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 && parsed <= 365 {
			days = parsed
		}
	}

	revenue, err := h.statsService.GetRevenueBySource(r.Context(), tenantID, days)
	if err != nil {
		writeServerError(w, "failed to retrieve revenue by source", err)
		return
	}
	writeJSON(w, http.StatusOK, revenue)
}

// GetOrderTrends returns order volume trends over time.
func (h *StatsHandler) GetOrderTrends(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	days := 30
	if v := r.URL.Query().Get("days"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 && parsed <= 365 {
			days = parsed
		}
	}

	trends, err := h.statsService.GetOrderTrends(r.Context(), tenantID, days)
	if err != nil {
		writeServerError(w, "failed to retrieve order trends", err)
		return
	}
	writeJSON(w, http.StatusOK, trends)
}

// GetPaymentMethodStats returns order counts grouped by payment method.
func (h *StatsHandler) GetPaymentMethodStats(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	stats, err := h.statsService.GetPaymentMethodStats(r.Context(), tenantID)
	if err != nil {
		writeServerError(w, "failed to retrieve payment method stats", err)
		return
	}
	writeJSON(w, http.StatusOK, stats)
}
