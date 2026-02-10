package handler

import (
	"encoding/json"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

type HealthHandler struct {
	DB *pgxpool.Pool
}

type healthResponse struct {
	Status   string `json:"status"`
	Database string `json:"database"`
	Version  string `json:"version"`
}

func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	dbStatus := "connected"
	status := "ok"
	httpStatus := http.StatusOK

	if err := h.DB.Ping(r.Context()); err != nil {
		dbStatus = "disconnected"
		status = "degraded"
		httpStatus = http.StatusServiceUnavailable
	}

	w.WriteHeader(httpStatus)
	json.NewEncoder(w).Encode(healthResponse{
		Status:   status,
		Database: dbStatus,
		Version:  "0.1.0",
	})
}
