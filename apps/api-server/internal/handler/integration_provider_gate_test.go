package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// decodeError extracts the "error" field from a JSON error response body.
func decodeError(t *testing.T, rr *httptest.ResponseRecorder) string {
	t.Helper()
	var resp map[string]string
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	return resp["error"]
}

func TestIntegrationHandler_Create_ProviderGate(t *testing.T) {
	key := make([]byte, 32)
	svc := service.NewIntegrationService(nil, nil, nil, key)

	cases := []struct {
		name        string
		provider    string
		mode        string
		wantBlocked bool // true -> 422 provider_not_available short-circuit
	}{
		{name: "beta provider blocked in client-ready", provider: "amazon", mode: "client-ready", wantBlocked: true},
		{name: "ready provider passes gate in client-ready", provider: "allegro", mode: "client-ready", wantBlocked: false},
		{name: "beta provider passes gate in full", provider: "amazon", mode: "full", wantBlocked: false},
		{name: "blocked provider blocked even in full", provider: "shopify", mode: "full", wantBlocked: true},
		{name: "unknown provider blocked in client-ready", provider: "nonexistent", mode: "client-ready", wantBlocked: true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h := NewIntegrationHandler(svc, nil, nil, c.mode)

			// Empty credentials => service-level validation error for the
			// passing cases, so the request never reaches the nil repo/pool.
			body := `{"provider":"` + c.provider + `"}`
			req := httptest.NewRequest(http.MethodPost, "/v1/integrations", strings.NewReader(body))
			req = req.WithContext(newContextWithTenantAndUser(req.Context(), uuid.New(), uuid.New()))
			rr := httptest.NewRecorder()

			h.Create(rr, req)

			if c.wantBlocked {
				assert.Equal(t, http.StatusUnprocessableEntity, rr.Code)
				assert.Equal(t, "provider_not_available", decodeError(t, rr))
				return
			}
			// Provider passed the gate: it must NOT be the 422 short-circuit.
			// The empty credentials make the service return a validation error.
			assert.NotEqual(t, "provider_not_available", decodeError(t, rr))
		})
	}
}

func TestShipmentHandler_Create_ProviderGate(t *testing.T) {
	svc := service.NewShipmentService(nil, nil, nil, nil, nil, nil, nil, "")
	// overLong tracking number forces a service validation error after the
	// provider gate but before any DB access, keeping passing cases panic-free.
	overLong := strings.Repeat("x", 201)

	cases := []struct {
		name        string
		provider    string
		mode        string
		wantBlocked bool
	}{
		{name: "manual always allowed in client-ready", provider: "manual", mode: "client-ready", wantBlocked: false},
		{name: "ready provider passes gate in client-ready", provider: "inpost", mode: "client-ready", wantBlocked: false},
		{name: "beta provider blocked in client-ready", provider: "ups", mode: "client-ready", wantBlocked: true},
		{name: "blocked provider blocked even in full", provider: "shopify", mode: "full", wantBlocked: true},
		{name: "unknown provider blocked in client-ready", provider: "nonexistent", mode: "client-ready", wantBlocked: true},
	}

	for _, c := range cases {
		t.Run("Create_"+c.name, func(t *testing.T) {
			h := NewShipmentHandler(svc, nil, c.mode)

			body := `{"provider":"` + c.provider + `","order_id":"` + uuid.New().String() + `","tracking_number":"` + overLong + `"}`
			req := httptest.NewRequest(http.MethodPost, "/v1/shipments", strings.NewReader(body))
			req = req.WithContext(newContextWithTenantAndUser(req.Context(), uuid.New(), uuid.New()))
			rr := httptest.NewRecorder()

			h.Create(rr, req)

			if c.wantBlocked {
				assert.Equal(t, http.StatusUnprocessableEntity, rr.Code)
				assert.Equal(t, "provider_not_available", decodeError(t, rr))
				return
			}
			assert.NotEqual(t, "provider_not_available", decodeError(t, rr))
		})

		t.Run("CreateForOrder_"+c.name, func(t *testing.T) {
			h := NewShipmentHandler(svc, nil, c.mode)

			orderID := uuid.New()
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", orderID.String())

			body := `{"provider":"` + c.provider + `","tracking_number":"` + overLong + `"}`
			req := httptest.NewRequest(http.MethodPost, "/v1/orders/"+orderID.String()+"/shipments", strings.NewReader(body))
			ctx := newContextWithTenantAndUser(req.Context(), uuid.New(), uuid.New())
			ctx = context.WithValue(ctx, chi.RouteCtxKey, rctx)
			req = req.WithContext(ctx)
			rr := httptest.NewRecorder()

			h.CreateForOrder(rr, req)

			if c.wantBlocked {
				assert.Equal(t, http.StatusUnprocessableEntity, rr.Code)
				assert.Equal(t, "provider_not_available", decodeError(t, rr))
				return
			}
			assert.NotEqual(t, "provider_not_available", decodeError(t, rr))
		})
	}
}
