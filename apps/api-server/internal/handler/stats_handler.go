package handler

import (
	"log/slog"
	"net/http"

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
