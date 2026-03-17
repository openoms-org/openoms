package handler

import (
	"encoding/json"
	"net/http"

	"github.com/redis/go-redis/v9"

	"github.com/openoms-org/openoms/apps/orchestrator/internal/store"
)

type HealthHandler struct {
	rdb       *redis.Client
	taskStore *store.TaskStore
}

func NewHealthHandler(rdb *redis.Client, taskStore *store.TaskStore) *HealthHandler {
	return &HealthHandler{rdb: rdb, taskStore: taskStore}
}

type HealthResponse struct {
	Status     string       `json:"status"`
	Checks     HealthChecks `json:"checks"`
	QueueDepth int          `json:"queue_depth"`
}

type HealthChecks struct {
	Redis string `json:"redis"`
}

func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	resp := HealthResponse{Status: "healthy"}

	if err := h.rdb.Ping(ctx).Err(); err != nil {
		resp.Status = "unhealthy"
		resp.Checks.Redis = "disconnected"
	} else {
		resp.Checks.Redis = "connected"
	}

	depth, err := h.taskStore.QueueDepth(ctx)
	if err != nil {
		resp.Status = "degraded"
	} else {
		resp.QueueDepth = depth
	}

	status := http.StatusOK
	if resp.Status == "unhealthy" {
		status = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(resp) //nolint:errcheck
}
