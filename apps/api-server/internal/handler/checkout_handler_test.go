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

	"github.com/openoms-org/openoms/apps/api-server/internal/config"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

func testCheckoutService(plans []config.PlanConfig) *service.CheckoutService {
	return service.NewCheckoutService(nil, nil, plans)
}

func TestCheckoutHandler_ListPlans(t *testing.T) {
	plans := []config.PlanConfig{
		{
			ID:            "standard",
			Name:          "Standard",
			MonthlyAmount: 9900,
			YearlyAmount:  99000,
			Currency:      "pln",
			TrialDays:     14,
			Features:      []string{"5 users", "1000 orders/mo"},
			Limits:        config.PlanLimits{MaxUsers: 5, MaxOrdersMonthly: 1000, MaxIntegrations: 2},
		},
	}
	svc := testCheckoutService(plans)
	h := NewCheckoutHandler(svc, nil, nil, "https://app.example.com")

	req := httptest.NewRequest(http.MethodGet, "/v1/billing/plans", nil)
	rr := httptest.NewRecorder()

	h.ListPlans(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var result []map[string]any
	err := json.NewDecoder(rr.Body).Decode(&result)
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "standard", result[0]["id"])
	assert.Equal(t, "Standard", result[0]["name"])
	// Verify no Stripe Price IDs are exposed
	_, hasMonthlyPriceID := result[0]["monthly_price_id"]
	assert.False(t, hasMonthlyPriceID, "monthly_price_id should not be exposed")
}

func TestCheckoutHandler_ListPlans_Empty(t *testing.T) {
	svc := testCheckoutService(nil)
	h := NewCheckoutHandler(svc, nil, nil, "https://app.example.com")

	req := httptest.NewRequest(http.MethodGet, "/v1/billing/plans", nil)
	rr := httptest.NewRecorder()

	h.ListPlans(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var result []map[string]any
	err := json.NewDecoder(rr.Body).Decode(&result)
	require.NoError(t, err)
	assert.Len(t, result, 0)
}

func TestCheckoutHandler_CreateCheckoutSession_InvalidJSON(t *testing.T) {
	svc := testCheckoutService(nil)
	h := NewCheckoutHandler(svc, nil, nil, "https://app.example.com")

	req := httptest.NewRequest(http.MethodPost, "/v1/billing/checkout", strings.NewReader("not json"))
	rr := httptest.NewRecorder()

	h.CreateCheckoutSession(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestCheckoutHandler_CreateCheckoutSession_MissingPlanID(t *testing.T) {
	svc := testCheckoutService(nil)
	h := NewCheckoutHandler(svc, nil, nil, "https://app.example.com")

	body := `{"interval":"month"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/billing/checkout", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.CreateCheckoutSession(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "plan_id is required", resp["error"])
}

func TestCheckoutHandler_CreateCheckoutSession_InvalidInterval(t *testing.T) {
	svc := testCheckoutService(nil)
	h := NewCheckoutHandler(svc, nil, nil, "https://app.example.com")

	body := `{"plan_id":"standard","interval":"weekly"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/billing/checkout", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.CreateCheckoutSession(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "interval must be 'month' or 'year'", resp["error"])
}

func TestCheckoutHandler_CreateCheckoutSession_PlanNotFound(t *testing.T) {
	svc := testCheckoutService(nil)
	h := NewCheckoutHandler(svc, nil, nil, "https://app.example.com")

	body := `{"plan_id":"nonexistent","interval":"month"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/billing/checkout", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.CreateCheckoutSession(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid plan_id", resp["error"])
}

func TestCheckoutHandler_GetCheckoutSessionStatus_EmptySessionID(t *testing.T) {
	svc := testCheckoutService(nil)
	h := NewCheckoutHandler(svc, nil, nil, "https://app.example.com")

	req := httptest.NewRequest(http.MethodGet, "/v1/billing/checkout/", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("session_id", "")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.GetCheckoutSessionStatus(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "session_id is required", resp["error"])
}
