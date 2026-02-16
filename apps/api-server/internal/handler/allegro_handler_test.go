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
)

// newChiContextWithParam creates a context with a single chi URL parameter set.
// This is useful for handlers that use chi.URLParam to extract route parameters.
func newChiContextWithParam(key, value string) context.Context {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	return context.WithValue(context.Background(), chi.RouteCtxKey, rctx)
}

// newChiContextWithParams creates a context with multiple chi URL parameters set.
func newChiContextWithParams(params map[string]string) context.Context {
	rctx := chi.NewRouteContext()
	for k, v := range params {
		rctx.URLParams.Add(k, v)
	}
	return context.WithValue(context.Background(), chi.RouteCtxKey, rctx)
}

// --- UpdateFulfillment ---

func TestAllegroHandler_UpdateFulfillment_InvalidOrderID(t *testing.T) {
	h := NewAllegroHandler(nil, nil, nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("orderId", "not-a-uuid")

	req := httptest.NewRequest(http.MethodPost, "/v1/integrations/allegro/orders/not-a-uuid/fulfillment", strings.NewReader(`{"status":"SENT"}`))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.UpdateFulfillment(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid order ID", resp["error"])
}

func TestAllegroHandler_UpdateFulfillment_InvalidJSON(t *testing.T) {
	h := NewAllegroHandler(nil, nil, nil)

	orderID := uuid.New()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("orderId", orderID.String())

	req := httptest.NewRequest(http.MethodPost, "/v1/integrations/allegro/orders/"+orderID.String()+"/fulfillment", strings.NewReader("not json"))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.UpdateFulfillment(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestAllegroHandler_UpdateFulfillment_MissingStatus(t *testing.T) {
	h := NewAllegroHandler(nil, nil, nil)

	orderID := uuid.New()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("orderId", orderID.String())

	req := httptest.NewRequest(http.MethodPost, "/v1/integrations/allegro/orders/"+orderID.String()+"/fulfillment", strings.NewReader(`{"status":""}`))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.UpdateFulfillment(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "status is required", resp["error"])
}

func TestAllegroHandler_UpdateFulfillment_EmptyBody(t *testing.T) {
	h := NewAllegroHandler(nil, nil, nil)

	orderID := uuid.New()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("orderId", orderID.String())

	req := httptest.NewRequest(http.MethodPost, "/v1/integrations/allegro/orders/"+orderID.String()+"/fulfillment", strings.NewReader("{}"))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.UpdateFulfillment(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "status is required", resp["error"])
}

func TestAllegroHandler_UpdateFulfillment_MissingOrderIDParam(t *testing.T) {
	h := NewAllegroHandler(nil, nil, nil)

	// No chi route context → orderId will be empty string → uuid.Parse fails
	req := httptest.NewRequest(http.MethodPost, "/v1/integrations/allegro/orders//fulfillment", strings.NewReader(`{"status":"SENT"}`))
	rr := httptest.NewRecorder()

	h.UpdateFulfillment(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid order ID", resp["error"])
}

// --- AddTracking ---

func TestAllegroHandler_AddTracking_InvalidOrderID(t *testing.T) {
	h := NewAllegroHandler(nil, nil, nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("orderId", "bad-id")

	req := httptest.NewRequest(http.MethodPost, "/v1/integrations/allegro/orders/bad-id/tracking", strings.NewReader(`{"carrier_id":"DHL","waybill":"123"}`))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.AddTracking(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid order ID", resp["error"])
}

func TestAllegroHandler_AddTracking_InvalidJSON(t *testing.T) {
	h := NewAllegroHandler(nil, nil, nil)

	orderID := uuid.New()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("orderId", orderID.String())

	req := httptest.NewRequest(http.MethodPost, "/v1/integrations/allegro/orders/"+orderID.String()+"/tracking", strings.NewReader("bad json"))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.AddTracking(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestAllegroHandler_AddTracking_MissingCarrierID(t *testing.T) {
	h := NewAllegroHandler(nil, nil, nil)

	orderID := uuid.New()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("orderId", orderID.String())

	req := httptest.NewRequest(http.MethodPost, "/v1/integrations/allegro/orders/"+orderID.String()+"/tracking", strings.NewReader(`{"carrier_id":"","waybill":"123"}`))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.AddTracking(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "carrier_id and waybill are required", resp["error"])
}

func TestAllegroHandler_AddTracking_MissingWaybill(t *testing.T) {
	h := NewAllegroHandler(nil, nil, nil)

	orderID := uuid.New()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("orderId", orderID.String())

	req := httptest.NewRequest(http.MethodPost, "/v1/integrations/allegro/orders/"+orderID.String()+"/tracking", strings.NewReader(`{"carrier_id":"DHL","waybill":""}`))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.AddTracking(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "carrier_id and waybill are required", resp["error"])
}

func TestAllegroHandler_AddTracking_BothFieldsMissing(t *testing.T) {
	h := NewAllegroHandler(nil, nil, nil)

	orderID := uuid.New()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("orderId", orderID.String())

	req := httptest.NewRequest(http.MethodPost, "/v1/integrations/allegro/orders/"+orderID.String()+"/tracking", strings.NewReader(`{}`))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.AddTracking(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "carrier_id and waybill are required", resp["error"])
}

func TestAllegroHandler_AddTracking_MissingOrderIDParam(t *testing.T) {
	h := NewAllegroHandler(nil, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/integrations/allegro/orders//tracking", strings.NewReader(`{"carrier_id":"DHL","waybill":"12345"}`))
	rr := httptest.NewRecorder()

	h.AddTracking(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid order ID", resp["error"])
}
