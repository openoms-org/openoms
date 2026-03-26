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

func TestDropshipHandler_Get_MissingIDParam(t *testing.T) {
	h := NewDropshipHandler(nil)

	rctx := chi.NewRouteContext()
	// no "id" param

	req := httptest.NewRequest(http.MethodGet, "/v1/dropship/", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Get(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid dropship order ID", resp["error"])
}

func TestDropshipHandler_Create_EmptyBody(t *testing.T) {
	h := NewDropshipHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/dropship", strings.NewReader(""))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestDropshipHandler_AutoRoute_MissingOrderIDParam(t *testing.T) {
	h := NewDropshipHandler(nil)

	rctx := chi.NewRouteContext()
	// no "order_id" param

	req := httptest.NewRequest(http.MethodPost, "/v1/dropship/auto-route/", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.AutoRoute(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid order ID", resp["error"])
}

func TestDropshipHandler_GetByOrderID_MissingOrderIDParam(t *testing.T) {
	h := NewDropshipHandler(nil)

	rctx := chi.NewRouteContext()

	req := httptest.NewRequest(http.MethodGet, "/v1/dropship/order/", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.GetByOrderID(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid order ID", resp["error"])
}

func TestDropshipHandler_UpdateStatus_EmptyBody(t *testing.T) {
	h := NewDropshipHandler(nil)

	id := uuid.New()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id.String())

	req := httptest.NewRequest(http.MethodPatch, "/v1/dropship/"+id.String()+"/status", strings.NewReader(""))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.UpdateStatus(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestDropshipHandler_Cancel_MissingIDParam(t *testing.T) {
	h := NewDropshipHandler(nil)

	rctx := chi.NewRouteContext()

	req := httptest.NewRequest(http.MethodPost, "/v1/dropship/cancel", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Cancel(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid dropship order ID", resp["error"])
}

func TestDropshipHandler_Get_NumericStringID(t *testing.T) {
	h := NewDropshipHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "12345")

	req := httptest.NewRequest(http.MethodGet, "/v1/dropship/12345", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Get(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid dropship order ID", resp["error"])
}
