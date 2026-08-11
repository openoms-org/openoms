package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
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

func TestWebhookHandler_Receive_RejectsMissingProviderSecret(t *testing.T) {
	tests := []struct {
		name          string
		provider      string
		allegroSecret string
		inpostSecret  string
	}{
		{
			name:          "allegro",
			provider:      "allegro",
			allegroSecret: "",
			inpostSecret:  "inpost-secret",
		},
		{
			name:          "inpost",
			provider:      "inpost",
			allegroSecret: "allegro-secret",
			inpostSecret:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			webhookService := service.NewWebhookService(nil, nil, nil, tt.allegroSecret, tt.inpostSecret)
			h := NewWebhookHandler(webhookService)
			tenantID := uuid.New()

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("provider", tt.provider)
			rctx.URLParams.Add("tenant_id", tenantID.String())

			req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/"+tt.provider+"/"+tenantID.String(), bytes.NewBufferString(`{"type":"ORDER_CREATED"}`))
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
			rr := httptest.NewRecorder()

			h.Receive(rr, req)

			assert.Equal(t, http.StatusUnprocessableEntity, rr.Code)
			var resp map[string]string
			err := json.NewDecoder(rr.Body).Decode(&resp)
			require.NoError(t, err)
			assert.Equal(t, "webhook secret not configured", resp["error"])
		})
	}
}
