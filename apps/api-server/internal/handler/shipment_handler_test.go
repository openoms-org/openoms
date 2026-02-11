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

func TestShipmentHandler_Create_InvalidJSON(t *testing.T) {
	h := NewShipmentHandler(nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/shipments", strings.NewReader("bad json"))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestShipmentHandler_Get_InvalidID(t *testing.T) {
	h := NewShipmentHandler(nil, nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodGet, "/v1/shipments/bad", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Get(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid shipment ID", resp["error"])
}

func TestShipmentHandler_Update_InvalidID(t *testing.T) {
	h := NewShipmentHandler(nil, nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodPatch, "/v1/shipments/bad", strings.NewReader(`{}`))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Update(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestShipmentHandler_Update_InvalidJSON(t *testing.T) {
	h := NewShipmentHandler(nil, nil)

	id := uuid.New()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id.String())

	req := httptest.NewRequest(http.MethodPatch, "/v1/shipments/"+id.String(), strings.NewReader("bad"))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Update(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestShipmentHandler_Delete_InvalidID(t *testing.T) {
	h := NewShipmentHandler(nil, nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodDelete, "/v1/shipments/bad", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Delete(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestShipmentHandler_TransitionStatus_InvalidID(t *testing.T) {
	h := NewShipmentHandler(nil, nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodPost, "/v1/shipments/bad/status", strings.NewReader(`{"status":"delivered"}`))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.TransitionStatus(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestShipmentHandler_TransitionStatus_InvalidJSON(t *testing.T) {
	h := NewShipmentHandler(nil, nil)

	id := uuid.New()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id.String())

	req := httptest.NewRequest(http.MethodPost, "/v1/shipments/"+id.String()+"/status", strings.NewReader("bad"))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.TransitionStatus(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestShipmentHandler_GenerateLabel_InvalidID(t *testing.T) {
	h := NewShipmentHandler(nil, nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodPost, "/v1/shipments/bad/label", strings.NewReader(`{}`))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.GenerateLabel(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestShipmentHandler_GenerateLabel_InvalidJSON(t *testing.T) {
	h := NewShipmentHandler(nil, nil)

	id := uuid.New()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id.String())

	req := httptest.NewRequest(http.MethodPost, "/v1/shipments/"+id.String()+"/label", strings.NewReader("bad"))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.GenerateLabel(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestShipmentHandler_List_InvalidOrderIDFilter(t *testing.T) {
	h := NewShipmentHandler(nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/shipments?order_id=not-uuid", nil)
	rr := httptest.NewRecorder()

	h.List(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid order_id filter", resp["error"])
}

func TestShipmentHandler_Create_ValidationError(t *testing.T) {
	svc := service.NewShipmentService(nil, nil, nil, nil, nil)
	h := NewShipmentHandler(svc, nil)

	tenantID := uuid.New()
	userID := uuid.New()

	// Missing provider
	body := `{"order_id":"` + uuid.New().String() + `"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/shipments", strings.NewReader(body))
	req = req.WithContext(newContextWithTenantAndUser(req.Context(), tenantID, userID))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "provider")
}

func TestShipmentHandler_TransitionStatus_ValidationError(t *testing.T) {
	svc := service.NewShipmentService(nil, nil, nil, nil, nil)
	h := NewShipmentHandler(svc, nil)

	tenantID := uuid.New()
	userID := uuid.New()
	shipmentID := uuid.New()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", shipmentID.String())

	// Empty status should fail validation
	body := `{"status":""}`
	req := httptest.NewRequest(http.MethodPost, "/v1/shipments/"+shipmentID.String()+"/status", strings.NewReader(body))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(newContextWithTenantAndUser(req.Context(), tenantID, userID))
	rr := httptest.NewRecorder()

	h.TransitionStatus(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestShipmentHandler_Update_ValidationError(t *testing.T) {
	svc := service.NewShipmentService(nil, nil, nil, nil, nil)
	h := NewShipmentHandler(svc, nil)

	tenantID := uuid.New()
	userID := uuid.New()
	shipmentID := uuid.New()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", shipmentID.String())

	// Empty body - no fields to update
	body := `{}`
	req := httptest.NewRequest(http.MethodPatch, "/v1/shipments/"+shipmentID.String(), strings.NewReader(body))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(newContextWithTenantAndUser(req.Context(), tenantID, userID))
	rr := httptest.NewRecorder()

	h.Update(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}
