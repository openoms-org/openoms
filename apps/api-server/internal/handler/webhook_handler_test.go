package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebhookHandler_Receive_InvalidTenantID(t *testing.T) {
	h := NewWebhookHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("provider", "allegro")
	rctx.URLParams.Add("tenant_id", "bad")

	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/allegro/bad", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Receive(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid tenant_id", resp["error"])
}
