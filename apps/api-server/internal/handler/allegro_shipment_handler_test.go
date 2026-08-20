package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- CreateShipment ---

func TestAllegroShipmentHandler_CreateShipment_InvalidJSON(t *testing.T) {
	h := NewAllegroShipmentHandler(nil, nil, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/integrations/allegro/shipments", strings.NewReader("not json"))
	rr := httptest.NewRecorder()

	h.CreateShipment(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "Invalid request body", resp["error"])
}

// --- GetLabel ---

func TestAllegroShipmentHandler_GetLabel_MissingShipmentID(t *testing.T) {
	h := NewAllegroShipmentHandler(nil, nil, nil, nil, nil, nil)

	// Without chi route context, shipmentId will be empty
	req := httptest.NewRequest(http.MethodGet, "/v1/integrations/allegro/shipments//label", nil)
	rr := httptest.NewRecorder()

	h.GetLabel(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "Missing shipment ID", resp["error"])
}

func TestAllegroShipmentHandler_GetLabel_EmptyShipmentIDInRoute(t *testing.T) {
	h := NewAllegroShipmentHandler(nil, nil, nil, nil, nil, nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("shipmentId", "")

	req := httptest.NewRequest(http.MethodGet, "/v1/integrations/allegro/shipments//label", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.GetLabel(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "Missing shipment ID", resp["error"])
}

// --- CancelShipment ---

func TestAllegroShipmentHandler_CancelShipment_MissingShipmentID(t *testing.T) {
	h := NewAllegroShipmentHandler(nil, nil, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodDelete, "/v1/integrations/allegro/shipments/", nil)
	rr := httptest.NewRecorder()

	h.CancelShipment(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "Missing shipment ID", resp["error"])
}

func TestAllegroShipmentHandler_CancelShipment_EmptyShipmentIDInRoute(t *testing.T) {
	h := NewAllegroShipmentHandler(nil, nil, nil, nil, nil, nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("shipmentId", "")

	req := httptest.NewRequest(http.MethodDelete, "/v1/integrations/allegro/shipments/", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.CancelShipment(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "Missing shipment ID", resp["error"])
}

func TestAllegroShipmentHandler_GetDeliveryProposals_MissingOrderID(t *testing.T) {
	h := NewAllegroShipmentHandler(nil, nil, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/integrations/allegro/delivery-proposals/", nil)
	rr := httptest.NewRecorder()

	h.GetDeliveryProposals(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "Missing Allegro order ID", resp["error"])
}

// --- GetPickupProposals ---

func TestAllegroShipmentHandler_GetPickupProposals_InvalidJSON(t *testing.T) {
	h := NewAllegroShipmentHandler(nil, nil, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/integrations/allegro/pickup-proposals", strings.NewReader("bad"))
	rr := httptest.NewRecorder()

	h.GetPickupProposals(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "Invalid request body", resp["error"])
}

// --- SchedulePickup ---

func TestAllegroShipmentHandler_SchedulePickup_InvalidJSON(t *testing.T) {
	h := NewAllegroShipmentHandler(nil, nil, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/integrations/allegro/pickups", strings.NewReader("bad"))
	rr := httptest.NewRecorder()

	h.SchedulePickup(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "Invalid request body", resp["error"])
}

// --- GenerateProtocol ---

func TestAllegroShipmentHandler_GenerateProtocol_InvalidJSON(t *testing.T) {
	h := NewAllegroShipmentHandler(nil, nil, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/integrations/allegro/protocol", strings.NewReader("bad"))
	rr := httptest.NewRecorder()

	h.GenerateProtocol(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "Invalid request body", resp["error"])
}

func TestAllegroShipmentHandler_GenerateProtocol_EmptyShipmentIDs(t *testing.T) {
	h := NewAllegroShipmentHandler(nil, nil, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/integrations/allegro/protocol", strings.NewReader(`{"shipment_ids":[]}`))
	rr := httptest.NewRecorder()

	h.GenerateProtocol(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "At least one shipment ID is required", resp["error"])
}

func TestAllegroShipmentHandler_GenerateProtocol_MissingShipmentIDsKey(t *testing.T) {
	h := NewAllegroShipmentHandler(nil, nil, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/integrations/allegro/protocol", strings.NewReader(`{}`))
	rr := httptest.NewRecorder()

	h.GenerateProtocol(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "At least one shipment ID is required", resp["error"])
}
