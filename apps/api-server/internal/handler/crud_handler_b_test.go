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
// PriceListHandler tests
// ===========================================================================

func TestPriceListHandler_Create_InvalidJSON(t *testing.T) {
	h := NewPriceListHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/price-lists", strings.NewReader("bad json"))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestPriceListHandler_Create_ValidationError(t *testing.T) {
	svc := service.NewPriceListService(nil, nil, nil, nil)
	h := NewPriceListHandler(svc)

	tenantID := uuid.New()
	userID := uuid.New()

	// Missing name (required)
	body := `{"currency":"PLN","discount_type":"percentage"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/price-lists", strings.NewReader(body))
	req = req.WithContext(newContextWithTenantAndUser(req.Context(), tenantID, userID))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "name")
}

func TestPriceListHandler_Update_InvalidUUID(t *testing.T) {
	h := NewPriceListHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodPatch, "/v1/price-lists/bad", strings.NewReader(`{}`))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Update(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid price list ID", resp["error"])
}

func TestPriceListHandler_Update_InvalidJSON(t *testing.T) {
	h := NewPriceListHandler(nil)

	id := uuid.New()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id.String())

	req := httptest.NewRequest(http.MethodPatch, "/v1/price-lists/"+id.String(), strings.NewReader("bad"))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Update(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestPriceListHandler_Delete_InvalidUUID(t *testing.T) {
	h := NewPriceListHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodDelete, "/v1/price-lists/bad", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Delete(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid price list ID", resp["error"])
}

func TestPriceListHandler_Get_InvalidUUID(t *testing.T) {
	h := NewPriceListHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "not-a-uuid")

	req := httptest.NewRequest(http.MethodGet, "/v1/price-lists/not-a-uuid", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Get(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid price list ID", resp["error"])
}

// ===========================================================================
// StocktakeHandler tests
// ===========================================================================

func TestStocktakeHandler_Create_InvalidJSON(t *testing.T) {
	h := NewStocktakeHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/stocktakes", strings.NewReader("bad"))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestStocktakeHandler_Create_ValidationError(t *testing.T) {
	svc := service.NewStocktakeService(nil, nil, nil, nil, nil, nil, nil, nil)
	h := NewStocktakeHandler(svc)

	tenantID := uuid.New()
	userID := uuid.New()

	// Missing warehouse_id and name (required)
	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/v1/stocktakes", strings.NewReader(body))
	req = req.WithContext(newContextWithTenantAndUser(req.Context(), tenantID, userID))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "warehouse_id")
}

func TestStocktakeHandler_Get_InvalidUUID(t *testing.T) {
	h := NewStocktakeHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodGet, "/v1/stocktakes/bad", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Get(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid stocktake ID", resp["error"])
}

func TestStocktakeHandler_Delete_InvalidUUID(t *testing.T) {
	h := NewStocktakeHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodDelete, "/v1/stocktakes/bad", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Delete(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid stocktake ID", resp["error"])
}

func TestStocktakeHandler_RecordCount_InvalidStocktakeID(t *testing.T) {
	h := NewStocktakeHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")
	rctx.URLParams.Add("itemId", uuid.New().String())

	req := httptest.NewRequest(http.MethodPost, "/v1/stocktakes/bad/items/"+uuid.New().String()+"/count", strings.NewReader(`{}`))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.RecordCount(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid stocktake ID", resp["error"])
}

func TestStocktakeHandler_RecordCount_InvalidItemID(t *testing.T) {
	h := NewStocktakeHandler(nil)

	stocktakeID := uuid.New()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", stocktakeID.String())
	rctx.URLParams.Add("itemId", "bad")

	req := httptest.NewRequest(http.MethodPost, "/v1/stocktakes/"+stocktakeID.String()+"/items/bad/count", strings.NewReader(`{}`))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.RecordCount(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid item ID", resp["error"])
}

func TestStocktakeHandler_RecordCount_InvalidJSON(t *testing.T) {
	h := NewStocktakeHandler(nil)

	stocktakeID := uuid.New()
	itemID := uuid.New()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", stocktakeID.String())
	rctx.URLParams.Add("itemId", itemID.String())

	req := httptest.NewRequest(http.MethodPost, "/v1/stocktakes/"+stocktakeID.String()+"/items/"+itemID.String()+"/count", strings.NewReader("bad"))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.RecordCount(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestStocktakeHandler_Start_InvalidUUID(t *testing.T) {
	h := NewStocktakeHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodPost, "/v1/stocktakes/bad/start", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Start(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid stocktake ID", resp["error"])
}

// ===========================================================================
// OrderGroupHandler tests
// ===========================================================================

func TestOrderGroupHandler_MergeOrders_InvalidJSON(t *testing.T) {
	h := NewOrderGroupHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/orders/merge", strings.NewReader("bad json"))
	rr := httptest.NewRecorder()

	h.MergeOrders(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestOrderGroupHandler_MergeOrders_ValidationError(t *testing.T) {
	svc := service.NewOrderGroupService(nil, nil, nil, nil)
	h := NewOrderGroupHandler(svc)

	tenantID := uuid.New()
	userID := uuid.New()

	// Only 1 order ID — need at least 2
	body := `{"order_ids":["` + uuid.New().String() + `"]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/orders/merge", strings.NewReader(body))
	req = req.WithContext(newContextWithTenantAndUser(req.Context(), tenantID, userID))
	rr := httptest.NewRecorder()

	h.MergeOrders(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "at least 2")
}

func TestOrderGroupHandler_SplitOrder_InvalidJSON(t *testing.T) {
	h := NewOrderGroupHandler(nil)

	orderID := uuid.New()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", orderID.String())

	req := httptest.NewRequest(http.MethodPost, "/v1/orders/"+orderID.String()+"/split", strings.NewReader("bad"))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.SplitOrder(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestOrderGroupHandler_SplitOrder_InvalidOrderID(t *testing.T) {
	h := NewOrderGroupHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodPost, "/v1/orders/bad/split", strings.NewReader(`{"splits":[]}`))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.SplitOrder(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid order ID", resp["error"])
}

func TestOrderGroupHandler_SplitOrder_ValidationError(t *testing.T) {
	svc := service.NewOrderGroupService(nil, nil, nil, nil)
	h := NewOrderGroupHandler(svc)

	tenantID := uuid.New()
	userID := uuid.New()
	orderID := uuid.New()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", orderID.String())

	// Only 1 split — need at least 2
	body := `{"splits":[{"items":[]}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/orders/"+orderID.String()+"/split", strings.NewReader(body))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(newContextWithTenantAndUser(req.Context(), tenantID, userID))
	rr := httptest.NewRecorder()

	h.SplitOrder(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "at least 2")
}

func TestOrderGroupHandler_ListByOrder_InvalidUUID(t *testing.T) {
	h := NewOrderGroupHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodGet, "/v1/orders/bad/groups", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.ListByOrder(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid order ID", resp["error"])
}

// ===========================================================================
// AutomationHandler tests
// ===========================================================================

func TestAutomationHandler_Create_InvalidJSON(t *testing.T) {
	h := NewAutomationHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/automation/rules", strings.NewReader("bad"))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestAutomationHandler_Create_ValidationError(t *testing.T) {
	svc := service.NewAutomationService(nil, nil, nil, nil, nil)
	h := NewAutomationHandler(svc)

	tenantID := uuid.New()
	userID := uuid.New()

	// Missing name (required)
	body := `{"trigger_event":"order.created","conditions":[],"actions":[]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/automation/rules", strings.NewReader(body))
	req = req.WithContext(newContextWithTenantAndUser(req.Context(), tenantID, userID))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "name")
}

func TestAutomationHandler_Update_InvalidUUID(t *testing.T) {
	h := NewAutomationHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodPatch, "/v1/automation/rules/bad", strings.NewReader(`{}`))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Update(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid rule ID", resp["error"])
}

func TestAutomationHandler_Update_InvalidJSON(t *testing.T) {
	h := NewAutomationHandler(nil)

	id := uuid.New()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id.String())

	req := httptest.NewRequest(http.MethodPatch, "/v1/automation/rules/"+id.String(), strings.NewReader("bad"))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Update(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestAutomationHandler_Delete_InvalidUUID(t *testing.T) {
	h := NewAutomationHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodDelete, "/v1/automation/rules/bad", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Delete(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid rule ID", resp["error"])
}

func TestAutomationHandler_Get_InvalidUUID(t *testing.T) {
	h := NewAutomationHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodGet, "/v1/automation/rules/bad", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Get(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid rule ID", resp["error"])
}

func TestAutomationHandler_TestRule_InvalidUUID(t *testing.T) {
	h := NewAutomationHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodPost, "/v1/automation/rules/bad/test", strings.NewReader(`{}`))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.TestRule(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid rule ID", resp["error"])
}

func TestAutomationHandler_TestRule_InvalidJSON(t *testing.T) {
	h := NewAutomationHandler(nil)

	id := uuid.New()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id.String())

	req := httptest.NewRequest(http.MethodPost, "/v1/automation/rules/"+id.String()+"/test", strings.NewReader("bad"))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.TestRule(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestAutomationHandler_GetLogs_InvalidUUID(t *testing.T) {
	h := NewAutomationHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodGet, "/v1/automation/rules/bad/logs", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.GetLogs(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid rule ID", resp["error"])
}

// ===========================================================================
// PublicReturnHandler tests
// ===========================================================================

func TestPublicReturnHandler_CreatePublicReturn_InvalidJSON(t *testing.T) {
	h := NewPublicReturnHandler(nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/public/returns", strings.NewReader("bad"))
	rr := httptest.NewRecorder()

	h.CreatePublicReturn(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestPublicReturnHandler_CreatePublicReturn_MissingOrderID(t *testing.T) {
	h := NewPublicReturnHandler(nil, nil)

	body := `{"email":"test@example.com","reason":"defective"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/public/returns", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.CreatePublicReturn(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "order_id")
}

func TestPublicReturnHandler_CreatePublicReturn_MissingEmail(t *testing.T) {
	h := NewPublicReturnHandler(nil, nil)

	body := `{"order_id":"` + uuid.New().String() + `","reason":"defective"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/public/returns", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.CreatePublicReturn(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "email")
}

func TestPublicReturnHandler_CreatePublicReturn_MissingReason(t *testing.T) {
	h := NewPublicReturnHandler(nil, nil)

	body := `{"order_id":"` + uuid.New().String() + `","email":"test@example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/public/returns", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.CreatePublicReturn(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "reason")
}

func TestPublicReturnHandler_GetByToken_EmptyToken(t *testing.T) {
	h := NewPublicReturnHandler(nil, nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("token", "")

	req := httptest.NewRequest(http.MethodGet, "/v1/public/returns/", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.GetByToken(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "token is required", resp["error"])
}

func TestPublicReturnHandler_GetStatusByToken_EmptyToken(t *testing.T) {
	h := NewPublicReturnHandler(nil, nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("token", "")

	req := httptest.NewRequest(http.MethodGet, "/v1/public/returns//status", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.GetStatusByToken(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "token is required", resp["error"])
}

// ===========================================================================
// AIHandler tests
// ===========================================================================

func TestAIHandler_Categorize_NotConfigured(t *testing.T) {
	// AI service with empty API key should report as not configured
	aiSvc := service.NewAIService("", "", nil, nil, nil, nil)
	h := NewAIHandler(aiSvc)

	req := httptest.NewRequest(http.MethodPost, "/v1/ai/categorize", strings.NewReader(`{}`))
	rr := httptest.NewRecorder()

	h.Categorize(rr, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "AI")
}

func TestAIHandler_Categorize_InvalidJSON(t *testing.T) {
	aiSvc := service.NewAIService("fake-key", "gpt-4", nil, nil, nil, nil)
	h := NewAIHandler(aiSvc)

	req := httptest.NewRequest(http.MethodPost, "/v1/ai/categorize", strings.NewReader("bad"))
	rr := httptest.NewRecorder()

	h.Categorize(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestAIHandler_Categorize_MissingProductID(t *testing.T) {
	aiSvc := service.NewAIService("fake-key", "gpt-4", nil, nil, nil, nil)
	h := NewAIHandler(aiSvc)

	req := httptest.NewRequest(http.MethodPost, "/v1/ai/categorize", strings.NewReader(`{}`))
	rr := httptest.NewRecorder()

	h.Categorize(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "product_id is required", resp["error"])
}

func TestAIHandler_Describe_NotConfigured(t *testing.T) {
	aiSvc := service.NewAIService("", "", nil, nil, nil, nil)
	h := NewAIHandler(aiSvc)

	req := httptest.NewRequest(http.MethodPost, "/v1/ai/describe", strings.NewReader(`{}`))
	rr := httptest.NewRecorder()

	h.Describe(rr, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rr.Code)
}

func TestAIHandler_Improve_NotConfigured(t *testing.T) {
	aiSvc := service.NewAIService("", "", nil, nil, nil, nil)
	h := NewAIHandler(aiSvc)

	req := httptest.NewRequest(http.MethodPost, "/v1/ai/improve", strings.NewReader(`{}`))
	rr := httptest.NewRecorder()

	h.Improve(rr, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rr.Code)
}

func TestAIHandler_Improve_InvalidJSON(t *testing.T) {
	aiSvc := service.NewAIService("fake-key", "gpt-4", nil, nil, nil, nil)
	h := NewAIHandler(aiSvc)

	req := httptest.NewRequest(http.MethodPost, "/v1/ai/improve", strings.NewReader("bad"))
	rr := httptest.NewRecorder()

	h.Improve(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestAIHandler_Improve_MissingDescription(t *testing.T) {
	aiSvc := service.NewAIService("fake-key", "gpt-4", nil, nil, nil, nil)
	h := NewAIHandler(aiSvc)

	req := httptest.NewRequest(http.MethodPost, "/v1/ai/improve", strings.NewReader(`{}`))
	rr := httptest.NewRecorder()

	h.Improve(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "description is required", resp["error"])
}

func TestAIHandler_Translate_NotConfigured(t *testing.T) {
	aiSvc := service.NewAIService("", "", nil, nil, nil, nil)
	h := NewAIHandler(aiSvc)

	req := httptest.NewRequest(http.MethodPost, "/v1/ai/translate", strings.NewReader(`{}`))
	rr := httptest.NewRecorder()

	h.Translate(rr, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rr.Code)
}

func TestAIHandler_Translate_MissingDescription(t *testing.T) {
	aiSvc := service.NewAIService("fake-key", "gpt-4", nil, nil, nil, nil)
	h := NewAIHandler(aiSvc)

	req := httptest.NewRequest(http.MethodPost, "/v1/ai/translate", strings.NewReader(`{"target_language":"en"}`))
	rr := httptest.NewRecorder()

	h.Translate(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "description is required", resp["error"])
}

func TestAIHandler_Translate_MissingTargetLanguage(t *testing.T) {
	aiSvc := service.NewAIService("fake-key", "gpt-4", nil, nil, nil, nil)
	h := NewAIHandler(aiSvc)

	req := httptest.NewRequest(http.MethodPost, "/v1/ai/translate", strings.NewReader(`{"description":"Hello"}`))
	rr := httptest.NewRecorder()

	h.Translate(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "target_language is required", resp["error"])
}

func TestAIHandler_BulkCategorize_NotConfigured(t *testing.T) {
	aiSvc := service.NewAIService("", "", nil, nil, nil, nil)
	h := NewAIHandler(aiSvc)

	req := httptest.NewRequest(http.MethodPost, "/v1/ai/bulk-categorize", strings.NewReader(`{}`))
	rr := httptest.NewRecorder()

	h.BulkCategorize(rr, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rr.Code)
}

func TestAIHandler_BulkCategorize_InvalidJSON(t *testing.T) {
	aiSvc := service.NewAIService("fake-key", "gpt-4", nil, nil, nil, nil)
	h := NewAIHandler(aiSvc)

	req := httptest.NewRequest(http.MethodPost, "/v1/ai/bulk-categorize", strings.NewReader("bad"))
	rr := httptest.NewRecorder()

	h.BulkCategorize(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestAIHandler_BulkCategorize_EmptyProductIDs(t *testing.T) {
	aiSvc := service.NewAIService("fake-key", "gpt-4", nil, nil, nil, nil)
	h := NewAIHandler(aiSvc)

	req := httptest.NewRequest(http.MethodPost, "/v1/ai/bulk-categorize", strings.NewReader(`{"product_ids":[]}`))
	rr := httptest.NewRecorder()

	h.BulkCategorize(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "product_ids is required", resp["error"])
}

// ===========================================================================
// ReconciliationHandler tests
// ===========================================================================

func TestReconciliationHandler_CreateSettlement_InvalidJSON(t *testing.T) {
	h := NewReconciliationHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/reconciliation/settlements", strings.NewReader("bad"))
	rr := httptest.NewRecorder()

	h.CreateSettlement(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestReconciliationHandler_CreateSettlement_MissingProvider(t *testing.T) {
	h := NewReconciliationHandler(nil)

	body := `{"settlement_date":"2024-01-01","total_amount":100}`
	req := httptest.NewRequest(http.MethodPost, "/v1/reconciliation/settlements", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.CreateSettlement(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "provider is required", resp["error"])
}

func TestReconciliationHandler_CreateSettlement_MissingDate(t *testing.T) {
	h := NewReconciliationHandler(nil)

	body := `{"provider":"allegro","total_amount":100}`
	req := httptest.NewRequest(http.MethodPost, "/v1/reconciliation/settlements", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.CreateSettlement(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "settlement_date is required", resp["error"])
}

func TestReconciliationHandler_GetSettlement_InvalidUUID(t *testing.T) {
	h := NewReconciliationHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodGet, "/v1/reconciliation/settlements/bad", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.GetSettlement(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid settlement ID", resp["error"])
}

func TestReconciliationHandler_AutoMatch_InvalidUUID(t *testing.T) {
	h := NewReconciliationHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodPost, "/v1/reconciliation/settlements/bad/auto-match", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.AutoMatch(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid settlement ID", resp["error"])
}

func TestReconciliationHandler_ManualMatch_InvalidTransactionID(t *testing.T) {
	h := NewReconciliationHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodPost, "/v1/reconciliation/transactions/bad/match", strings.NewReader(`{}`))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.ManualMatch(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid transaction ID", resp["error"])
}

func TestReconciliationHandler_ManualMatch_InvalidJSON(t *testing.T) {
	h := NewReconciliationHandler(nil)

	id := uuid.New()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id.String())

	req := httptest.NewRequest(http.MethodPost, "/v1/reconciliation/transactions/"+id.String()+"/match", strings.NewReader("bad"))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.ManualMatch(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestReconciliationHandler_ManualMatch_MissingOrderID(t *testing.T) {
	h := NewReconciliationHandler(nil)

	id := uuid.New()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id.String())

	req := httptest.NewRequest(http.MethodPost, "/v1/reconciliation/transactions/"+id.String()+"/match", strings.NewReader(`{}`))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.ManualMatch(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "order_id is required", resp["error"])
}

func TestReconciliationHandler_Unmatch_InvalidUUID(t *testing.T) {
	h := NewReconciliationHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodPost, "/v1/reconciliation/transactions/bad/unmatch", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Unmatch(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid transaction ID", resp["error"])
}

func TestReconciliationHandler_ListTransactions_InvalidSettlementIDFilter(t *testing.T) {
	h := NewReconciliationHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/reconciliation/transactions?settlement_id=not-uuid", nil)
	rr := httptest.NewRecorder()

	h.ListTransactions(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid settlement_id filter", resp["error"])
}
