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

	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// newContextWithTenantAndUser creates a context with tenant and user IDs set.
func newContextWithTenantAndUser(ctx context.Context, tenantID, userID uuid.UUID) context.Context {
	ctx = context.WithValue(ctx, middleware.TenantIDKey, tenantID)
	ctx = context.WithValue(ctx, middleware.UserIDKey, userID)
	return ctx
}

func TestOrderHandler_Create_InvalidJSON(t *testing.T) {
	// orderService can be nil because we expect to fail before calling it
	h := NewOrderHandler(nil, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/orders", strings.NewReader("not json"))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestOrderHandler_Get_InvalidOrderID(t *testing.T) {
	h := NewOrderHandler(nil, nil, nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "not-a-uuid")

	req := httptest.NewRequest(http.MethodGet, "/v1/orders/not-a-uuid", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Get(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid order ID", resp["error"])
}

func TestOrderHandler_Update_InvalidOrderID(t *testing.T) {
	h := NewOrderHandler(nil, nil, nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "invalid")

	req := httptest.NewRequest(http.MethodPatch, "/v1/orders/invalid", strings.NewReader(`{"customer_name":"x"}`))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Update(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestOrderHandler_Update_InvalidJSON(t *testing.T) {
	h := NewOrderHandler(nil, nil, nil)

	orderID := uuid.New()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", orderID.String())

	req := httptest.NewRequest(http.MethodPatch, "/v1/orders/"+orderID.String(), strings.NewReader("not json"))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Update(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestOrderHandler_Delete_InvalidOrderID(t *testing.T) {
	h := NewOrderHandler(nil, nil, nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad-id")

	req := httptest.NewRequest(http.MethodDelete, "/v1/orders/bad-id", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Delete(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestOrderHandler_TransitionStatus_InvalidOrderID(t *testing.T) {
	h := NewOrderHandler(nil, nil, nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad-id")

	req := httptest.NewRequest(http.MethodPost, "/v1/orders/bad-id/status", strings.NewReader(`{"status":"confirmed"}`))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.TransitionStatus(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestOrderHandler_TransitionStatus_InvalidJSON(t *testing.T) {
	h := NewOrderHandler(nil, nil, nil)

	orderID := uuid.New()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", orderID.String())

	req := httptest.NewRequest(http.MethodPost, "/v1/orders/"+orderID.String()+"/status", strings.NewReader("bad"))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.TransitionStatus(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestOrderHandler_BulkTransitionStatus_InvalidJSON(t *testing.T) {
	h := NewOrderHandler(nil, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/orders/bulk/status", strings.NewReader("not json"))
	rr := httptest.NewRecorder()

	h.BulkTransitionStatus(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestOrderHandler_GetAudit_InvalidOrderID(t *testing.T) {
	h := NewOrderHandler(nil, nil, nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad-id")

	req := httptest.NewRequest(http.MethodGet, "/v1/orders/bad-id/audit", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.GetAudit(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestOrderHandler_Create_ValidationError(t *testing.T) {
	// OrderService with nil pool will fail at WithTenant, but validation error happens before
	svc := service.NewOrderService(nil, nil, nil, nil, nil, nil)
	h := NewOrderHandler(svc, nil, nil)

	tenantID := uuid.New()
	userID := uuid.New()

	body := `{"customer_name":"","total_amount":10}`
	req := httptest.NewRequest(http.MethodPost, "/v1/orders", strings.NewReader(body))
	req = req.WithContext(newContextWithTenantAndUser(req.Context(), tenantID, userID))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "customer_name")
}

func TestOrderHandler_BulkTransitionStatus_ValidationError(t *testing.T) {
	svc := service.NewOrderService(nil, nil, nil, nil, nil, nil)
	h := NewOrderHandler(svc, nil, nil)

	tenantID := uuid.New()
	userID := uuid.New()

	// Empty order IDs - should fail validation
	body := `{"order_ids":[],"status":"confirmed"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/orders/bulk/status", strings.NewReader(body))
	req = req.WithContext(newContextWithTenantAndUser(req.Context(), tenantID, userID))
	rr := httptest.NewRecorder()

	h.BulkTransitionStatus(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "order_id")
}

func TestOrderHandler_TransitionStatus_ValidationError(t *testing.T) {
	svc := service.NewOrderService(nil, nil, nil, nil, nil, nil)
	h := NewOrderHandler(svc, nil, nil)

	tenantID := uuid.New()
	userID := uuid.New()
	orderID := uuid.New()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", orderID.String())

	// Empty status - should fail validation
	body := `{"status":""}`
	req := httptest.NewRequest(http.MethodPost, "/v1/orders/"+orderID.String()+"/status", strings.NewReader(body))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(newContextWithTenantAndUser(req.Context(), tenantID, userID))
	rr := httptest.NewRecorder()

	h.TransitionStatus(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestOrderHandler_Update_ValidationError(t *testing.T) {
	svc := service.NewOrderService(nil, nil, nil, nil, nil, nil)
	h := NewOrderHandler(svc, nil, nil)

	tenantID := uuid.New()
	userID := uuid.New()
	orderID := uuid.New()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", orderID.String())

	// Empty body - no fields to update - should fail validation
	body := `{}`
	req := httptest.NewRequest(http.MethodPatch, "/v1/orders/"+orderID.String(), strings.NewReader(body))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(newContextWithTenantAndUser(req.Context(), tenantID, userID))
	rr := httptest.NewRecorder()

	h.Update(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}
