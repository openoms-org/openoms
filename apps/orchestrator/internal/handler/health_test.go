package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/orchestrator/internal/handler"
	"github.com/openoms-org/openoms/apps/orchestrator/internal/store"
)

func setupHealthTest(t *testing.T) (*handler.HealthHandler, *redis.Client) {
	t.Helper()
	url := os.Getenv("REDIS_URL")
	if url == "" {
		url = "redis://localhost:6379/1"
	}
	opts, err := redis.ParseURL(url)
	require.NoError(t, err)
	rdb := redis.NewClient(opts)
	require.NoError(t, rdb.Ping(context.Background()).Err())
	rdb.FlushDB(context.Background())
	t.Cleanup(func() {
		rdb.FlushDB(context.Background())
		rdb.Close()
	})

	taskStore := store.New(rdb)
	return handler.NewHealthHandler(rdb, taskStore), rdb
}

func TestHealth_Healthy(t *testing.T) {
	h, _ := setupHealthTest(t)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	h.Health(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "healthy", resp["status"])
}
