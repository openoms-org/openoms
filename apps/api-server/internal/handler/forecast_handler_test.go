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

func TestNewForecastHandler(t *testing.T) {
	h := NewForecastHandler(nil)
	assert.NotNil(t, h)
}

func TestForecastHandler_GetForecast_EmptyProductID(t *testing.T) {
	h := NewForecastHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "")

	req := httptest.NewRequest(http.MethodGet, "/v1/forecast/products/", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.GetForecast(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestForecastHandler_GetForecast_MissingIDParam(t *testing.T) {
	h := NewForecastHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/forecast/products/", nil)
	rr := httptest.NewRecorder()

	h.GetForecast(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestForecastHandler_GetSeasonality_EmptyProductID(t *testing.T) {
	h := NewForecastHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("product_id", "")

	req := httptest.NewRequest(http.MethodGet, "/v1/forecast/seasonality/", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.GetSeasonality(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestForecastHandler_UpdateConfig_EmptyBody(t *testing.T) {
	h := NewForecastHandler(nil)

	req := httptest.NewRequest(http.MethodPut, "/v1/forecast/config", strings.NewReader(""))
	rr := httptest.NewRecorder()

	h.UpdateConfig(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestForecastHandler_UpdateConfig_LeadTimeExactBoundary366(t *testing.T) {
	h := NewForecastHandler(nil)

	body := `{"default_lead_time_days":366,"safety_stock_days":7,"forecast_days_ahead":30}`
	req := httptest.NewRequest(http.MethodPut, "/v1/forecast/config", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.UpdateConfig(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "lead time must be between 1 and 365 days", resp["error"])
}

func TestForecastHandler_UpdateConfig_SafetyStockExactBoundary366(t *testing.T) {
	h := NewForecastHandler(nil)

	body := `{"default_lead_time_days":7,"safety_stock_days":366,"forecast_days_ahead":30}`
	req := httptest.NewRequest(http.MethodPut, "/v1/forecast/config", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.UpdateConfig(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "safety stock must be between 0 and 365 days", resp["error"])
}

func TestForecastHandler_UpdateConfig_ForecastDaysExactBoundary366(t *testing.T) {
	h := NewForecastHandler(nil)

	body := `{"default_lead_time_days":7,"safety_stock_days":7,"forecast_days_ahead":366}`
	req := httptest.NewRequest(http.MethodPut, "/v1/forecast/config", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.UpdateConfig(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "forecast horizon must be between 1 and 365 days", resp["error"])
}

func TestForecastHandler_UpdateConfig_NegativeLeadTime(t *testing.T) {
	h := NewForecastHandler(nil)

	body := `{"default_lead_time_days":-5,"safety_stock_days":7,"forecast_days_ahead":30}`
	req := httptest.NewRequest(http.MethodPut, "/v1/forecast/config", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.UpdateConfig(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "lead time must be between 1 and 365 days", resp["error"])
}

func TestForecastHandler_UpdateConfig_NegativeForecastDays(t *testing.T) {
	h := NewForecastHandler(nil)

	body := `{"default_lead_time_days":7,"safety_stock_days":7,"forecast_days_ahead":-10}`
	req := httptest.NewRequest(http.MethodPut, "/v1/forecast/config", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.UpdateConfig(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "forecast horizon must be between 1 and 365 days", resp["error"])
}
