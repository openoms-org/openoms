package handler

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

type HealthHandler struct {
	DB *pgxpool.Pool
}

type healthResponse struct {
	Status   string `json:"status"`
	Database string `json:"database"`
}

func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	dbStatus := "connected"
	status := "ok"
	httpStatus := http.StatusOK

	if err := h.DB.Ping(r.Context()); err != nil {
		dbStatus = "disconnected"
		status = "degraded"
		httpStatus = http.StatusServiceUnavailable
	}

	writeJSON(w, httpStatus, healthResponse{
		Status:   status,
		Database: dbStatus,
	})
}
