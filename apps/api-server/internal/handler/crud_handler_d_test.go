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

// ===========================================================================
// PurchaseOrderHandler tests
// ===========================================================================

func TestPurchaseOrderHandler_Create_InvalidJSON(t *testing.T) {
	h := NewPurchaseOrderHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/purchase-orders", strings.NewReader("bad json"))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestPurchaseOrderHandler_Get_InvalidID(t *testing.T) {
	h := NewPurchaseOrderHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodGet, "/v1/purchase-orders/bad", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Get(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid purchase order ID", resp["error"])
}

func TestPurchaseOrderHandler_Update_InvalidID(t *testing.T) {
	h := NewPurchaseOrderHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodPatch, "/v1/purchase-orders/bad", strings.NewReader(`{"notes":"test"}`))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Update(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid purchase order ID", resp["error"])
}

func TestPurchaseOrderHandler_Update_InvalidJSON(t *testing.T) {
	h := NewPurchaseOrderHandler(nil)

	id := uuid.New()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id.String())

	req := httptest.NewRequest(http.MethodPatch, "/v1/purchase-orders/"+id.String(), strings.NewReader("bad"))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Update(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestPurchaseOrderHandler_Delete_InvalidID(t *testing.T) {
	h := NewPurchaseOrderHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodDelete, "/v1/purchase-orders/bad", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Delete(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid purchase order ID", resp["error"])
}

func TestPurchaseOrderHandler_Send_InvalidID(t *testing.T) {
	h := NewPurchaseOrderHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodPost, "/v1/purchase-orders/bad/send", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Send(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid purchase order ID", resp["error"])
}

func TestPurchaseOrderHandler_Receive_InvalidID(t *testing.T) {
	h := NewPurchaseOrderHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodPost, "/v1/purchase-orders/bad/receive", strings.NewReader(`{}`))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Receive(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid purchase order ID", resp["error"])
}

func TestPurchaseOrderHandler_Receive_InvalidJSON(t *testing.T) {
	h := NewPurchaseOrderHandler(nil)

	id := uuid.New()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id.String())

	req := httptest.NewRequest(http.MethodPost, "/v1/purchase-orders/"+id.String()+"/receive", strings.NewReader("bad"))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Receive(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestPurchaseOrderHandler_Cancel_InvalidID(t *testing.T) {
	h := NewPurchaseOrderHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodPost, "/v1/purchase-orders/bad/cancel", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Cancel(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid purchase order ID", resp["error"])
}

// ===========================================================================
// RecurringOrderHandler tests
// ===========================================================================

func TestRecurringOrderHandler_Create_InvalidJSON(t *testing.T) {
	h := NewRecurringOrderHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/recurring-orders", strings.NewReader("bad json"))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestRecurringOrderHandler_Get_InvalidID(t *testing.T) {
	h := NewRecurringOrderHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodGet, "/v1/recurring-orders/bad", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Get(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid recurring order ID", resp["error"])
}

func TestRecurringOrderHandler_Update_InvalidID(t *testing.T) {
	h := NewRecurringOrderHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodPatch, "/v1/recurring-orders/bad", strings.NewReader(`{}`))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Update(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid recurring order ID", resp["error"])
}

func TestRecurringOrderHandler_Update_InvalidJSON(t *testing.T) {
	h := NewRecurringOrderHandler(nil)

	id := uuid.New()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id.String())

	req := httptest.NewRequest(http.MethodPatch, "/v1/recurring-orders/"+id.String(), strings.NewReader("bad"))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Update(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestRecurringOrderHandler_Delete_InvalidID(t *testing.T) {
	h := NewRecurringOrderHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodDelete, "/v1/recurring-orders/bad", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Delete(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid recurring order ID", resp["error"])
}

func TestRecurringOrderHandler_Pause_InvalidID(t *testing.T) {
	h := NewRecurringOrderHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodPost, "/v1/recurring-orders/bad/pause", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Pause(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid recurring order ID", resp["error"])
}

func TestRecurringOrderHandler_Resume_InvalidID(t *testing.T) {
	h := NewRecurringOrderHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodPost, "/v1/recurring-orders/bad/resume", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Resume(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid recurring order ID", resp["error"])
}

func TestRecurringOrderHandler_Cancel_InvalidID(t *testing.T) {
	h := NewRecurringOrderHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodPost, "/v1/recurring-orders/bad/cancel", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Cancel(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid recurring order ID", resp["error"])
}

// ===========================================================================
// RepricingHandler tests
// ===========================================================================

func TestRepricingHandler_CreateRule_InvalidJSON(t *testing.T) {
	h := NewRepricingHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/repricing/rules", strings.NewReader("bad json"))
	rr := httptest.NewRecorder()

	h.CreateRule(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestRepricingHandler_GetRule_InvalidID(t *testing.T) {
	h := NewRepricingHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodGet, "/v1/repricing/rules/bad", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.GetRule(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid rule ID", resp["error"])
}

func TestRepricingHandler_UpdateRule_InvalidID(t *testing.T) {
	h := NewRepricingHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodPatch, "/v1/repricing/rules/bad", strings.NewReader(`{}`))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.UpdateRule(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid rule ID", resp["error"])
}

func TestRepricingHandler_UpdateRule_InvalidJSON(t *testing.T) {
	h := NewRepricingHandler(nil)

	id := uuid.New()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id.String())

	req := httptest.NewRequest(http.MethodPatch, "/v1/repricing/rules/"+id.String(), strings.NewReader("bad"))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.UpdateRule(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestRepricingHandler_DeleteRule_InvalidID(t *testing.T) {
	h := NewRepricingHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodDelete, "/v1/repricing/rules/bad", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.DeleteRule(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid rule ID", resp["error"])
}

func TestRepricingHandler_SimulateRule_InvalidID(t *testing.T) {
	h := NewRepricingHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodPost, "/v1/repricing/rules/bad/simulate", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.SimulateRule(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid rule ID", resp["error"])
}

func TestRepricingHandler_ListLog_InvalidRuleID(t *testing.T) {
	h := NewRepricingHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/repricing/log?rule_id=not-uuid", nil)
	rr := httptest.NewRecorder()

	h.ListLog(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid rule_id", resp["error"])
}

func TestRepricingHandler_ListLog_InvalidProductID(t *testing.T) {
	h := NewRepricingHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/repricing/log?product_id=not-uuid", nil)
	rr := httptest.NewRecorder()

	h.ListLog(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid product_id", resp["error"])
}

// ===========================================================================
// SegmentHandler tests
// ===========================================================================

func TestSegmentHandler_Create_InvalidJSON(t *testing.T) {
	h := NewSegmentHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/segments", strings.NewReader("bad"))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestSegmentHandler_Get_InvalidID(t *testing.T) {
	h := NewSegmentHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodGet, "/v1/segments/bad", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Get(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid segment ID", resp["error"])
}

func TestSegmentHandler_Update_InvalidID(t *testing.T) {
	h := NewSegmentHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodPatch, "/v1/segments/bad", strings.NewReader(`{}`))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Update(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid segment ID", resp["error"])
}

func TestSegmentHandler_Update_InvalidJSON(t *testing.T) {
	h := NewSegmentHandler(nil)

	id := uuid.New()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id.String())

	req := httptest.NewRequest(http.MethodPatch, "/v1/segments/"+id.String(), strings.NewReader("bad"))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Update(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestSegmentHandler_Delete_InvalidID(t *testing.T) {
	h := NewSegmentHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodDelete, "/v1/segments/bad", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Delete(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid segment ID", resp["error"])
}

func TestSegmentHandler_ListMembers_InvalidID(t *testing.T) {
	h := NewSegmentHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodGet, "/v1/segments/bad/members", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.ListMembers(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid segment ID", resp["error"])
}

func TestSegmentHandler_AddMember_InvalidSegmentID(t *testing.T) {
	h := NewSegmentHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodPost, "/v1/segments/bad/members", strings.NewReader(`{}`))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.AddMember(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid segment ID", resp["error"])
}

func TestSegmentHandler_AddMember_InvalidJSON(t *testing.T) {
	h := NewSegmentHandler(nil)

	id := uuid.New()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id.String())

	req := httptest.NewRequest(http.MethodPost, "/v1/segments/"+id.String()+"/members", strings.NewReader("bad"))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.AddMember(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestSegmentHandler_RemoveMember_InvalidSegmentID(t *testing.T) {
	h := NewSegmentHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")
	rctx.URLParams.Add("customer_id", uuid.New().String())

	req := httptest.NewRequest(http.MethodDelete, "/v1/segments/bad/members/"+uuid.New().String(), nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.RemoveMember(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid segment ID", resp["error"])
}

func TestSegmentHandler_RemoveMember_InvalidCustomerID(t *testing.T) {
	h := NewSegmentHandler(nil)

	segmentID := uuid.New()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", segmentID.String())
	rctx.URLParams.Add("customer_id", "bad")

	req := httptest.NewRequest(http.MethodDelete, "/v1/segments/"+segmentID.String()+"/members/bad", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.RemoveMember(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid customer ID", resp["error"])
}

func TestSegmentHandler_GetCustomerSegments_InvalidCustomerID(t *testing.T) {
	h := NewSegmentHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("customer_id", "bad")

	req := httptest.NewRequest(http.MethodGet, "/v1/segments/customers/bad", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.GetCustomerSegments(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid customer ID", resp["error"])
}

// ===========================================================================
// StockSyncHandler tests
// ===========================================================================

func TestStockSyncHandler_CreateChannel_InvalidJSON(t *testing.T) {
	h := NewStockSyncHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/stock-sync/channels", strings.NewReader("bad"))
	rr := httptest.NewRecorder()

	h.CreateChannel(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestStockSyncHandler_GetChannel_InvalidID(t *testing.T) {
	h := NewStockSyncHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodGet, "/v1/stock-sync/channels/bad", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.GetChannel(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid channel ID", resp["error"])
}

func TestStockSyncHandler_UpdateChannel_InvalidID(t *testing.T) {
	h := NewStockSyncHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodPut, "/v1/stock-sync/channels/bad", strings.NewReader(`{}`))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.UpdateChannel(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid channel ID", resp["error"])
}

func TestStockSyncHandler_UpdateChannel_InvalidJSON(t *testing.T) {
	h := NewStockSyncHandler(nil)

	id := uuid.New()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id.String())

	req := httptest.NewRequest(http.MethodPut, "/v1/stock-sync/channels/"+id.String(), strings.NewReader("bad"))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.UpdateChannel(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestStockSyncHandler_DeleteChannel_InvalidID(t *testing.T) {
	h := NewStockSyncHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodDelete, "/v1/stock-sync/channels/bad", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.DeleteChannel(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid channel ID", resp["error"])
}

func TestStockSyncHandler_PushProduct_InvalidID(t *testing.T) {
	h := NewStockSyncHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("product_id", "bad")

	req := httptest.NewRequest(http.MethodPost, "/v1/stock-sync/push/bad", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.PushProduct(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid product ID", resp["error"])
}

func TestStockSyncHandler_ReconcileProduct_InvalidID(t *testing.T) {
	h := NewStockSyncHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("product_id", "bad")

	req := httptest.NewRequest(http.MethodPost, "/v1/stock-sync/reconcile/bad", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.ReconcileProduct(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid product ID", resp["error"])
}

func TestStockSyncHandler_ListEvents_InvalidProductID(t *testing.T) {
	h := NewStockSyncHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/stock-sync/events?product_id=not-uuid", nil)
	rr := httptest.NewRecorder()

	h.ListEvents(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid product_id", resp["error"])
}

func TestStockSyncHandler_PushListing_InvalidID(t *testing.T) {
	h := NewStockSyncHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("listing_id", "bad")

	req := httptest.NewRequest(http.MethodPost, "/v1/stock-sync/push/listing/bad", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.PushListing(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid listing ID", resp["error"])
}

func TestStockSyncHandler_GetAllocations_InvalidID(t *testing.T) {
	h := NewStockSyncHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("product_id", "bad")

	req := httptest.NewRequest(http.MethodGet, "/v1/stock-sync/allocations/bad", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.GetAllocations(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid product ID", resp["error"])
}

// ===========================================================================
// ListingSyncHandler tests
// ===========================================================================

func TestListingSyncHandler_CreateConfig_InvalidJSON(t *testing.T) {
	h := NewListingSyncHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/listing-sync/configs", strings.NewReader("bad"))
	rr := httptest.NewRecorder()

	h.CreateConfig(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid input data", resp["error"])
}

func TestListingSyncHandler_CreateConfig_ValidationError_MissingIntegrationID(t *testing.T) {
	h := NewListingSyncHandler(nil)

	body := `{"sync_direction":"push"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/listing-sync/configs", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.CreateConfig(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "integration_id")
}

func TestListingSyncHandler_GetConfig_InvalidID(t *testing.T) {
	h := NewListingSyncHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodGet, "/v1/listing-sync/configs/bad", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.GetConfig(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid ID", resp["error"])
}

func TestListingSyncHandler_UpdateConfig_InvalidID(t *testing.T) {
	h := NewListingSyncHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodPut, "/v1/listing-sync/configs/bad", strings.NewReader(`{}`))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.UpdateConfig(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid ID", resp["error"])
}

func TestListingSyncHandler_UpdateConfig_InvalidJSON(t *testing.T) {
	h := NewListingSyncHandler(nil)

	id := uuid.New()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id.String())

	req := httptest.NewRequest(http.MethodPut, "/v1/listing-sync/configs/"+id.String(), strings.NewReader("bad"))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.UpdateConfig(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid input data", resp["error"])
}

func TestListingSyncHandler_UpdateConfig_ValidationError_EmptyBody(t *testing.T) {
	h := NewListingSyncHandler(nil)

	id := uuid.New()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id.String())

	req := httptest.NewRequest(http.MethodPut, "/v1/listing-sync/configs/"+id.String(), strings.NewReader(`{}`))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.UpdateConfig(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "at least one field")
}

func TestListingSyncHandler_DeleteConfig_InvalidID(t *testing.T) {
	h := NewListingSyncHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodDelete, "/v1/listing-sync/configs/bad", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.DeleteConfig(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid ID", resp["error"])
}

func TestListingSyncHandler_TriggerSync_InvalidID(t *testing.T) {
	h := NewListingSyncHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodPost, "/v1/listing-sync/configs/bad/sync", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.TriggerSync(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid ID", resp["error"])
}

func TestListingSyncHandler_TriggerSyncPrices_InvalidID(t *testing.T) {
	h := NewListingSyncHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodPost, "/v1/listing-sync/configs/bad/sync-prices", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.TriggerSyncPrices(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid ID", resp["error"])
}

func TestListingSyncHandler_TriggerSyncStock_InvalidID(t *testing.T) {
	h := NewListingSyncHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodPost, "/v1/listing-sync/configs/bad/sync-stock", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.TriggerSyncStock(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid ID", resp["error"])
}

func TestListingSyncHandler_ListLogs_InvalidID(t *testing.T) {
	h := NewListingSyncHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodGet, "/v1/listing-sync/configs/bad/logs", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.ListLogs(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid ID", resp["error"])
}

// ===========================================================================
// ProductCategoryHandler tests
// ===========================================================================

func TestProductCategoryHandler_Create_InvalidJSON(t *testing.T) {
	h := NewProductCategoryHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/product-categories", strings.NewReader("bad"))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestProductCategoryHandler_Create_ValidationError_MissingName(t *testing.T) {
	h := NewProductCategoryHandler(nil)

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/v1/product-categories", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "name")
}

func TestProductCategoryHandler_Get_InvalidID(t *testing.T) {
	h := NewProductCategoryHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodGet, "/v1/product-categories/bad", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Get(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid category ID", resp["error"])
}

func TestProductCategoryHandler_Update_InvalidID(t *testing.T) {
	h := NewProductCategoryHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodPatch, "/v1/product-categories/bad", strings.NewReader(`{}`))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Update(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid category ID", resp["error"])
}

func TestProductCategoryHandler_Update_InvalidJSON(t *testing.T) {
	h := NewProductCategoryHandler(nil)

	id := uuid.New()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id.String())

	req := httptest.NewRequest(http.MethodPatch, "/v1/product-categories/"+id.String(), strings.NewReader("bad"))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Update(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestProductCategoryHandler_Update_ValidationError_EmptyBody(t *testing.T) {
	h := NewProductCategoryHandler(nil)

	id := uuid.New()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id.String())

	req := httptest.NewRequest(http.MethodPatch, "/v1/product-categories/"+id.String(), strings.NewReader(`{}`))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Update(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "at least one field")
}

func TestProductCategoryHandler_Delete_InvalidID(t *testing.T) {
	h := NewProductCategoryHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodDelete, "/v1/product-categories/bad", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Delete(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid category ID", resp["error"])
}

func TestProductCategoryHandler_ListDescendants_InvalidID(t *testing.T) {
	h := NewProductCategoryHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodGet, "/v1/product-categories/bad/descendants", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.ListDescendants(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid category ID", resp["error"])
}

func TestProductCategoryHandler_List_InvalidParentID(t *testing.T) {
	h := NewProductCategoryHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/product-categories?parent_id=not-uuid", nil)
	rr := httptest.NewRecorder()

	h.List(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid parent_id", resp["error"])
}

// ===========================================================================
// PickPackHandler tests
// ===========================================================================

func TestPickPackHandler_CreateSession_InvalidJSON(t *testing.T) {
	h := NewPickPackHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/pick-pack/sessions", strings.NewReader("bad"))
	rr := httptest.NewRecorder()

	h.CreateSession(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestPickPackHandler_GetSession_InvalidID(t *testing.T) {
	h := NewPickPackHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodGet, "/v1/pick-pack/sessions/bad", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.GetSession(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid session ID", resp["error"])
}

func TestPickPackHandler_ScanItem_InvalidSessionID(t *testing.T) {
	h := NewPickPackHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodPost, "/v1/pick-pack/sessions/bad/scan", strings.NewReader(`{}`))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.ScanItem(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid session ID", resp["error"])
}

func TestPickPackHandler_ScanItem_InvalidJSON(t *testing.T) {
	h := NewPickPackHandler(nil)

	id := uuid.New()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id.String())

	req := httptest.NewRequest(http.MethodPost, "/v1/pick-pack/sessions/"+id.String()+"/scan", strings.NewReader("bad"))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.ScanItem(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestPickPackHandler_MoveToPacking_InvalidID(t *testing.T) {
	h := NewPickPackHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodPost, "/v1/pick-pack/sessions/bad/pack", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.MoveToPacking(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid session ID", resp["error"])
}

func TestPickPackHandler_MarkItemPacked_InvalidSessionID(t *testing.T) {
	h := NewPickPackHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")
	rctx.URLParams.Add("itemId", uuid.New().String())

	req := httptest.NewRequest(http.MethodPost, "/v1/pick-pack/sessions/bad/items/"+uuid.New().String()+"/pack", strings.NewReader(`{}`))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.MarkItemPacked(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid session ID", resp["error"])
}

func TestPickPackHandler_MarkItemPacked_InvalidItemID(t *testing.T) {
	h := NewPickPackHandler(nil)

	sessionID := uuid.New()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", sessionID.String())
	rctx.URLParams.Add("itemId", "bad")

	req := httptest.NewRequest(http.MethodPost, "/v1/pick-pack/sessions/"+sessionID.String()+"/items/bad/pack", strings.NewReader(`{}`))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.MarkItemPacked(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid item ID", resp["error"])
}

func TestPickPackHandler_MarkItemPacked_InvalidJSON(t *testing.T) {
	h := NewPickPackHandler(nil)

	sessionID := uuid.New()
	itemID := uuid.New()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", sessionID.String())
	rctx.URLParams.Add("itemId", itemID.String())

	req := httptest.NewRequest(http.MethodPost, "/v1/pick-pack/sessions/"+sessionID.String()+"/items/"+itemID.String()+"/pack", strings.NewReader("bad"))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.MarkItemPacked(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestPickPackHandler_CompleteSession_InvalidID(t *testing.T) {
	h := NewPickPackHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodPost, "/v1/pick-pack/sessions/bad/complete", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.CompleteSession(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid session ID", resp["error"])
}

func TestPickPackHandler_CancelSession_InvalidID(t *testing.T) {
	h := NewPickPackHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodPost, "/v1/pick-pack/sessions/bad/cancel", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.CancelSession(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid session ID", resp["error"])
}

// ===========================================================================
// LoyaltyHandler tests
// ===========================================================================

func TestLoyaltyHandler_CreateProgram_InvalidJSON(t *testing.T) {
	h := NewLoyaltyHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/loyalty/programs", strings.NewReader("bad"))
	rr := httptest.NewRecorder()

	h.CreateProgram(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestLoyaltyHandler_GetProgram_InvalidID(t *testing.T) {
	h := NewLoyaltyHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodGet, "/v1/loyalty/programs/bad", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.GetProgram(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid program ID", resp["error"])
}

func TestLoyaltyHandler_UpdateProgram_InvalidID(t *testing.T) {
	h := NewLoyaltyHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodPatch, "/v1/loyalty/programs/bad", strings.NewReader(`{}`))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.UpdateProgram(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid program ID", resp["error"])
}

func TestLoyaltyHandler_UpdateProgram_InvalidJSON(t *testing.T) {
	h := NewLoyaltyHandler(nil)

	id := uuid.New()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id.String())

	req := httptest.NewRequest(http.MethodPatch, "/v1/loyalty/programs/"+id.String(), strings.NewReader("bad"))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.UpdateProgram(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestLoyaltyHandler_DeleteProgram_InvalidID(t *testing.T) {
	h := NewLoyaltyHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodDelete, "/v1/loyalty/programs/bad", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.DeleteProgram(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid program ID", resp["error"])
}

func TestLoyaltyHandler_GetCustomerLoyaltyStatus_InvalidCustomerID(t *testing.T) {
	h := NewLoyaltyHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("customer_id", "bad")

	req := httptest.NewRequest(http.MethodGet, "/v1/loyalty/customers/bad", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.GetCustomerLoyaltyStatus(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid customer ID", resp["error"])
}

func TestLoyaltyHandler_AwardPoints_InvalidProgramID(t *testing.T) {
	h := NewLoyaltyHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodPost, "/v1/loyalty/programs/bad/award", strings.NewReader(`{}`))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.AwardPoints(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid program ID", resp["error"])
}

func TestLoyaltyHandler_AwardPoints_InvalidJSON(t *testing.T) {
	h := NewLoyaltyHandler(nil)

	id := uuid.New()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id.String())

	req := httptest.NewRequest(http.MethodPost, "/v1/loyalty/programs/"+id.String()+"/award", strings.NewReader("bad"))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.AwardPoints(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestLoyaltyHandler_RedeemPoints_InvalidProgramID(t *testing.T) {
	h := NewLoyaltyHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodPost, "/v1/loyalty/programs/bad/redeem", strings.NewReader(`{}`))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.RedeemPoints(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid program ID", resp["error"])
}

func TestLoyaltyHandler_RedeemPoints_InvalidJSON(t *testing.T) {
	h := NewLoyaltyHandler(nil)

	id := uuid.New()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id.String())

	req := httptest.NewRequest(http.MethodPost, "/v1/loyalty/programs/"+id.String()+"/redeem", strings.NewReader("bad"))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.RedeemPoints(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestLoyaltyHandler_GetLeaderboard_InvalidProgramID(t *testing.T) {
	h := NewLoyaltyHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodGet, "/v1/loyalty/programs/bad/leaderboard", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.GetLeaderboard(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid program ID", resp["error"])
}

// ===========================================================================
// WorkflowHandler tests
// ===========================================================================

func TestWorkflowHandler_Validate_InvalidJSON(t *testing.T) {
	h := NewWorkflowHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/workflows/validate", strings.NewReader("bad"))
	rr := httptest.NewRecorder()

	h.Validate(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestWorkflowHandler_Convert_InvalidJSON(t *testing.T) {
	h := NewWorkflowHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/workflows/convert", strings.NewReader("bad"))
	rr := httptest.NewRecorder()

	h.Convert(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestWorkflowHandler_GetWorkflowForRule_InvalidID(t *testing.T) {
	h := NewWorkflowHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodGet, "/v1/workflows/rules/bad", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.GetWorkflowForRule(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid rule ID", resp["error"])
}

// ===========================================================================
// VatOssHandler tests
// ===========================================================================

func TestVatOssHandler_Calculate_InvalidJSON(t *testing.T) {
	vatSvc := service.NewVATOSSService(nil, nil)
	h := NewVATOSSHandler(vatSvc)

	req := httptest.NewRequest(http.MethodPost, "/v1/vat-oss/calculate", strings.NewReader("bad"))
	rr := httptest.NewRecorder()

	h.Calculate(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestVatOssHandler_Calculate_ZeroAmount(t *testing.T) {
	vatSvc := service.NewVATOSSService(nil, nil)
	h := NewVATOSSHandler(vatSvc)

	body := `{"amount":0,"country":"DE"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/vat-oss/calculate", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.Calculate(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "amount must be positive", resp["error"])
}

func TestVatOssHandler_Calculate_MissingCountry(t *testing.T) {
	vatSvc := service.NewVATOSSService(nil, nil)
	h := NewVATOSSHandler(vatSvc)

	body := `{"amount":100}`
	req := httptest.NewRequest(http.MethodPost, "/v1/vat-oss/calculate", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.Calculate(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "country is required", resp["error"])
}

func TestVatOssHandler_Calculate_NonEUCountry(t *testing.T) {
	vatSvc := service.NewVATOSSService(nil, nil)
	h := NewVATOSSHandler(vatSvc)

	body := `{"amount":100,"country":"US"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/vat-oss/calculate", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.Calculate(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "not an EU member state")
}

func TestVatOssHandler_UpdateConfig_InvalidJSON(t *testing.T) {
	vatSvc := service.NewVATOSSService(nil, nil)
	h := NewVATOSSHandler(vatSvc)

	req := httptest.NewRequest(http.MethodPut, "/v1/vat-oss/config", strings.NewReader("bad"))
	rr := httptest.NewRecorder()

	h.UpdateConfig(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestVatOssHandler_UpdateConfig_NonEUHomeCountry(t *testing.T) {
	vatSvc := service.NewVATOSSService(nil, nil)
	h := NewVATOSSHandler(vatSvc)

	body := `{"home_country":"US","enabled":true}`
	req := httptest.NewRequest(http.MethodPut, "/v1/vat-oss/config", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.UpdateConfig(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "not an EU member state")
}

func TestVatOssHandler_UpdateConfig_InvalidVATRateType(t *testing.T) {
	vatSvc := service.NewVATOSSService(nil, nil)
	h := NewVATOSSHandler(vatSvc)

	body := `{"home_country":"PL","default_vat_rate":"invalid_type"}`
	req := httptest.NewRequest(http.MethodPut, "/v1/vat-oss/config", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.UpdateConfig(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid default VAT rate type", resp["error"])
}

func TestVatOssHandler_GetReport_InvalidQuarter(t *testing.T) {
	vatSvc := service.NewVATOSSService(nil, nil)
	h := NewVATOSSHandler(vatSvc)

	req := httptest.NewRequest(http.MethodGet, "/v1/vat-oss/report?quarter=5&year=2026", nil)
	rr := httptest.NewRecorder()

	h.GetReport(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "invalid quarter")
}

func TestVatOssHandler_GetReport_InvalidYear(t *testing.T) {
	vatSvc := service.NewVATOSSService(nil, nil)
	h := NewVATOSSHandler(vatSvc)

	req := httptest.NewRequest(http.MethodGet, "/v1/vat-oss/report?quarter=1&year=1990", nil)
	rr := httptest.NewRecorder()

	h.GetReport(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "invalid year")
}

func TestVatOssHandler_GetThreshold_InvalidYear(t *testing.T) {
	vatSvc := service.NewVATOSSService(nil, nil)
	h := NewVATOSSHandler(vatSvc)

	req := httptest.NewRequest(http.MethodGet, "/v1/vat-oss/threshold?year=abc", nil)
	rr := httptest.NewRecorder()

	h.GetThreshold(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid year parameter", resp["error"])
}

func TestVatOssHandler_GetCountryRates_NotFound(t *testing.T) {
	vatSvc := service.NewVATOSSService(nil, nil)
	h := NewVATOSSHandler(vatSvc)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("country", "XX")

	req := httptest.NewRequest(http.MethodGet, "/v1/vat-oss/rates/XX", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.GetCountryRates(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "not found")
}

// ===========================================================================
// MarketingHandler tests
// ===========================================================================

func TestMarketingHandler_CreateCampaign_InvalidJSON(t *testing.T) {
	h := NewMarketingHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/marketing/campaigns", strings.NewReader("bad"))
	rr := httptest.NewRecorder()

	h.CreateCampaign(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestMarketingHandler_CreateCampaign_MissingFields(t *testing.T) {
	h := NewMarketingHandler(nil)

	body := `{"name":"Test"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/marketing/campaigns", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.CreateCampaign(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "name, subject and content are required")
}

// ===========================================================================
// HelpdeskHandler tests
// ===========================================================================

func TestHelpdeskHandler_ListOrderTickets_InvalidOrderID(t *testing.T) {
	h := NewHelpdeskHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodGet, "/v1/orders/bad/tickets", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.ListOrderTickets(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid order ID", resp["error"])
}

func TestHelpdeskHandler_CreateOrderTicket_InvalidOrderID(t *testing.T) {
	h := NewHelpdeskHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodPost, "/v1/orders/bad/tickets", strings.NewReader(`{}`))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.CreateOrderTicket(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid order ID", resp["error"])
}

func TestHelpdeskHandler_CreateOrderTicket_InvalidJSON(t *testing.T) {
	h := NewHelpdeskHandler(nil)

	id := uuid.New()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id.String())

	req := httptest.NewRequest(http.MethodPost, "/v1/orders/"+id.String()+"/tickets", strings.NewReader("bad"))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.CreateOrderTicket(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestHelpdeskHandler_CreateOrderTicket_MissingFields(t *testing.T) {
	h := NewHelpdeskHandler(nil)

	id := uuid.New()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id.String())

	body := `{"subject":"Test"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/orders/"+id.String()+"/tickets", strings.NewReader(body))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.CreateOrderTicket(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "subject, description and email are required")
}

// ===========================================================================
// KSeFHandler tests
// ===========================================================================

func TestKSeFHandler_UpdateSettings_InvalidJSON(t *testing.T) {
	h := NewKSeFHandler(nil)

	req := httptest.NewRequest(http.MethodPut, "/v1/ksef/settings", strings.NewReader("bad"))
	rr := httptest.NewRecorder()

	h.UpdateSettings(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestKSeFHandler_SendToKSeF_InvalidInvoiceID(t *testing.T) {
	h := NewKSeFHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodPost, "/v1/ksef/invoices/bad/send", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.SendToKSeF(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid invoice ID", resp["error"])
}

func TestKSeFHandler_CheckKSeFStatus_InvalidInvoiceID(t *testing.T) {
	h := NewKSeFHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodGet, "/v1/ksef/invoices/bad/status", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.CheckKSeFStatus(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid invoice ID", resp["error"])
}

func TestKSeFHandler_GetUPO_InvalidInvoiceID(t *testing.T) {
	h := NewKSeFHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodGet, "/v1/ksef/invoices/bad/upo", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.GetUPO(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid invoice ID", resp["error"])
}

func TestKSeFHandler_BulkSendToKSeF_InvalidJSON(t *testing.T) {
	h := NewKSeFHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/ksef/invoices/bulk-send", strings.NewReader("bad"))
	rr := httptest.NewRecorder()

	h.BulkSendToKSeF(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestKSeFHandler_BulkSendToKSeF_EmptyInvoiceIDs(t *testing.T) {
	h := NewKSeFHandler(nil)

	body := `{"invoice_ids":[]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/ksef/invoices/bulk-send", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.BulkSendToKSeF(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invoice_ids is required", resp["error"])
}

func TestKSeFHandler_BulkSendToKSeF_InvalidInvoiceIDInList(t *testing.T) {
	h := NewKSeFHandler(nil)

	body := `{"invoice_ids":["not-a-uuid"]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/ksef/invoices/bulk-send", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.BulkSendToKSeF(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "invalid invoice ID")
}

// ===========================================================================
// SupplierPortalHandler tests
// ===========================================================================

func TestSupplierPortalHandler_GenerateLink_InvalidSupplierID(t *testing.T) {
	h := NewSupplierPortalHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodPost, "/v1/suppliers/bad/portal/link", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.GenerateLink(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid supplier ID", resp["error"])
}

func TestSupplierPortalHandler_RevokeAccess_InvalidSupplierID(t *testing.T) {
	h := NewSupplierPortalHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodPost, "/v1/suppliers/bad/portal/revoke", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.RevokeAccess(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid supplier ID", resp["error"])
}

func TestSupplierPortalHandler_GetPortalStatus_InvalidSupplierID(t *testing.T) {
	h := NewSupplierPortalHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodGet, "/v1/suppliers/bad/portal/status", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.GetPortalStatus(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid supplier ID", resp["error"])
}

func TestSupplierPortalHandler_ListOrders_MissingToken(t *testing.T) {
	h := NewSupplierPortalHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/supplier-portal/orders", nil)
	rr := httptest.NewRecorder()

	h.ListOrders(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "portal token required", resp["error"])
}

func TestSupplierPortalHandler_ListOrders_IgnoresQueryToken(t *testing.T) {
	h := NewSupplierPortalHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/supplier-portal/orders?token=leaked", nil)
	rr := httptest.NewRecorder()

	h.ListOrders(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "portal token required", resp["error"])
}

func TestSupplierPortalHandler_GetOrder_MissingToken(t *testing.T) {
	h := NewSupplierPortalHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", uuid.New().String())

	req := httptest.NewRequest(http.MethodGet, "/v1/supplier-portal/orders/"+uuid.New().String(), nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.GetOrder(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "portal token required", resp["error"])
}

func TestSupplierPortalHandler_ConfirmOrder_MissingToken(t *testing.T) {
	h := NewSupplierPortalHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", uuid.New().String())

	req := httptest.NewRequest(http.MethodPost, "/v1/supplier-portal/orders/"+uuid.New().String()+"/confirm", strings.NewReader(`{}`))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.ConfirmOrder(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "portal token required", resp["error"])
}

func TestSupplierPortalHandler_ShipOrder_MissingToken(t *testing.T) {
	h := NewSupplierPortalHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", uuid.New().String())

	req := httptest.NewRequest(http.MethodPost, "/v1/supplier-portal/orders/"+uuid.New().String()+"/ship", strings.NewReader(`{}`))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.ShipOrder(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "portal token required", resp["error"])
}

func TestSupplierPortalHandler_AddMessage_MissingToken(t *testing.T) {
	h := NewSupplierPortalHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", uuid.New().String())

	req := httptest.NewRequest(http.MethodPost, "/v1/supplier-portal/orders/"+uuid.New().String()+"/messages", strings.NewReader(`{}`))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.AddMessage(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "portal token required", resp["error"])
}

func TestSupplierPortalHandler_ListMessages_MissingToken(t *testing.T) {
	h := NewSupplierPortalHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", uuid.New().String())

	req := httptest.NewRequest(http.MethodGet, "/v1/supplier-portal/orders/"+uuid.New().String()+"/messages", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.ListMessages(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "portal token required", resp["error"])
}

// ===========================================================================
// SyncJobHandler tests
// ===========================================================================

func TestSyncJobHandler_Get_InvalidID(t *testing.T) {
	h := NewSyncJobHandler(nil, nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodGet, "/v1/sync-jobs/bad", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Get(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid sync job ID", resp["error"])
}
