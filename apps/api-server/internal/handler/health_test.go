package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteJSON_HealthLikeResponse(t *testing.T) {
	// We can't easily test HealthHandler without a real pgxpool.Pool,
	// but we verify the response format the handler uses.
	rr := httptest.NewRecorder()
	writeJSON(rr, http.StatusOK, healthResponse{
		Status:   "ok",
		Database: "connected",
		Version:  "0.1.0",
	})

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	var resp healthResponse
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "ok", resp.Status)
	assert.Equal(t, "connected", resp.Database)
	assert.Equal(t, "0.1.0", resp.Version)
}
