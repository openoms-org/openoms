package handler

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

type StatsHandler struct {
	statsService *service.StatsService
}

func NewStatsHandler(statsService *service.StatsService) *StatsHandler {
	return &StatsHandler{statsService: statsService}
}

func (h *StatsHandler) GetDashboard(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	stats, err := h.statsService.GetDashboardStats(r.Context(), tenantID)
	if err != nil {
		slog.Error("dashboard stats failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to retrieve dashboard stats")
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

func (h *StatsHandler) GetTopProducts(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	limit := 10
	if v := r.URL.Query().Get("limit"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	products, err := h.statsService.GetTopProducts(r.Context(), tenantID, limit)
	if err != nil {
		slog.Error("top products stats failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to retrieve top products")
		return
	}
	writeJSON(w, http.StatusOK, products)
}

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
		slog.Error("revenue by source stats failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to retrieve revenue by source")
		return
	}
	writeJSON(w, http.StatusOK, revenue)
}

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
		slog.Error("order trends stats failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to retrieve order trends")
		return
	}
	writeJSON(w, http.StatusOK, trends)
}

func (h *StatsHandler) GetPaymentMethodStats(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	stats, err := h.statsService.GetPaymentMethodStats(r.Context(), tenantID)
	if err != nil {
		slog.Error("payment method stats failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to retrieve payment method stats")
		return
	}
	writeJSON(w, http.StatusOK, stats)
}
