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
// CustomerHandler tests
// ===========================================================================

func TestCustomerHandler_Create_InvalidJSON(t *testing.T) {
	h := NewCustomerHandler(nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/customers", strings.NewReader("bad json"))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestCustomerHandler_Create_ValidationError(t *testing.T) {
	svc := service.NewCustomerService(nil, nil, nil, nil, nil)
	h := NewCustomerHandler(svc, nil)

	tenantID := uuid.New()
	userID := uuid.New()

	// Missing name (required field)
	body := `{"email":"test@example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/customers", strings.NewReader(body))
	req = req.WithContext(newContextWithTenantAndUser(req.Context(), tenantID, userID))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "name")
}

func TestCustomerHandler_Get_InvalidID(t *testing.T) {
	h := NewCustomerHandler(nil, nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodGet, "/v1/customers/bad", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Get(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid customer ID", resp["error"])
}

func TestCustomerHandler_Update_InvalidID(t *testing.T) {
	h := NewCustomerHandler(nil, nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodPatch, "/v1/customers/bad", strings.NewReader(`{"name":"Test"}`))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Update(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid customer ID", resp["error"])
}

func TestCustomerHandler_Update_InvalidJSON(t *testing.T) {
	h := NewCustomerHandler(nil, nil)

	id := uuid.New()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id.String())

	req := httptest.NewRequest(http.MethodPatch, "/v1/customers/"+id.String(), strings.NewReader("bad"))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Update(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestCustomerHandler_Delete_InvalidID(t *testing.T) {
	h := NewCustomerHandler(nil, nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodDelete, "/v1/customers/bad", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Delete(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid customer ID", resp["error"])
}

func TestCustomerHandler_ListOrders_InvalidID(t *testing.T) {
	h := NewCustomerHandler(nil, nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "not-a-uuid")

	req := httptest.NewRequest(http.MethodGet, "/v1/customers/not-a-uuid/orders", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.ListOrders(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid customer ID", resp["error"])
}

// ===========================================================================
// WarehouseHandler tests
// ===========================================================================

func TestWarehouseHandler_Create_InvalidJSON(t *testing.T) {
	h := NewWarehouseHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/warehouses", strings.NewReader("bad json"))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestWarehouseHandler_Create_ValidationError(t *testing.T) {
	svc := service.NewWarehouseService(nil, nil, nil, nil, nil)
	h := NewWarehouseHandler(svc)

	tenantID := uuid.New()
	userID := uuid.New()

	// Missing name (required field)
	body := `{"code":"WH1"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/warehouses", strings.NewReader(body))
	req = req.WithContext(newContextWithTenantAndUser(req.Context(), tenantID, userID))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "name")
}

func TestWarehouseHandler_Get_InvalidID(t *testing.T) {
	h := NewWarehouseHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodGet, "/v1/warehouses/bad", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Get(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid warehouse ID", resp["error"])
}

func TestWarehouseHandler_Update_InvalidID(t *testing.T) {
	h := NewWarehouseHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodPatch, "/v1/warehouses/bad", strings.NewReader(`{"name":"Test"}`))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Update(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid warehouse ID", resp["error"])
}

func TestWarehouseHandler_Update_InvalidJSON(t *testing.T) {
	h := NewWarehouseHandler(nil)

	id := uuid.New()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id.String())

	req := httptest.NewRequest(http.MethodPatch, "/v1/warehouses/"+id.String(), strings.NewReader("bad"))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Update(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestWarehouseHandler_Delete_InvalidID(t *testing.T) {
	h := NewWarehouseHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodDelete, "/v1/warehouses/bad", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Delete(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid warehouse ID", resp["error"])
}

func TestWarehouseHandler_ListStock_InvalidID(t *testing.T) {
	h := NewWarehouseHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodGet, "/v1/warehouses/bad/stock", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.ListStock(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid warehouse ID", resp["error"])
}

func TestWarehouseHandler_UpsertStock_InvalidID(t *testing.T) {
	h := NewWarehouseHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodPut, "/v1/warehouses/bad/stock", strings.NewReader(`{}`))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.UpsertStock(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid warehouse ID", resp["error"])
}

func TestWarehouseHandler_UpsertStock_InvalidJSON(t *testing.T) {
	h := NewWarehouseHandler(nil)

	id := uuid.New()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id.String())

	req := httptest.NewRequest(http.MethodPut, "/v1/warehouses/"+id.String()+"/stock", strings.NewReader("bad"))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.UpsertStock(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestWarehouseHandler_ListProductStock_InvalidID(t *testing.T) {
	h := NewWarehouseHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodGet, "/v1/products/bad/stock", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.ListProductStock(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid product ID", resp["error"])
}

// ===========================================================================
// ExchangeRateHandler tests
// ===========================================================================

func TestExchangeRateHandler_Create_InvalidJSON(t *testing.T) {
	h := NewExchangeRateHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/exchange-rates", strings.NewReader("bad json"))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestExchangeRateHandler_Create_ValidationError_MissingBaseCurrency(t *testing.T) {
	svc := service.NewExchangeRateService(nil, nil, nil)
	h := NewExchangeRateHandler(svc)

	tenantID := uuid.New()
	userID := uuid.New()

	// Missing base_currency
	body := `{"target_currency":"USD","rate":4.5}`
	req := httptest.NewRequest(http.MethodPost, "/v1/exchange-rates", strings.NewReader(body))
	req = req.WithContext(newContextWithTenantAndUser(req.Context(), tenantID, userID))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "base_currency")
}

func TestExchangeRateHandler_Create_ValidationError_MissingTargetCurrency(t *testing.T) {
	svc := service.NewExchangeRateService(nil, nil, nil)
	h := NewExchangeRateHandler(svc)

	tenantID := uuid.New()
	userID := uuid.New()

	// Missing target_currency
	body := `{"base_currency":"PLN","rate":4.5}`
	req := httptest.NewRequest(http.MethodPost, "/v1/exchange-rates", strings.NewReader(body))
	req = req.WithContext(newContextWithTenantAndUser(req.Context(), tenantID, userID))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "target_currency")
}

func TestExchangeRateHandler_Create_ValidationError_ZeroRate(t *testing.T) {
	svc := service.NewExchangeRateService(nil, nil, nil)
	h := NewExchangeRateHandler(svc)

	tenantID := uuid.New()
	userID := uuid.New()

	// Zero rate
	body := `{"base_currency":"PLN","target_currency":"USD","rate":0}`
	req := httptest.NewRequest(http.MethodPost, "/v1/exchange-rates", strings.NewReader(body))
	req = req.WithContext(newContextWithTenantAndUser(req.Context(), tenantID, userID))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "rate")
}

func TestExchangeRateHandler_Get_InvalidID(t *testing.T) {
	h := NewExchangeRateHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodGet, "/v1/exchange-rates/bad", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Get(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid exchange rate ID", resp["error"])
}

func TestExchangeRateHandler_Update_InvalidID(t *testing.T) {
	h := NewExchangeRateHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodPatch, "/v1/exchange-rates/bad", strings.NewReader(`{"rate":4.5}`))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Update(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid exchange rate ID", resp["error"])
}

func TestExchangeRateHandler_Update_InvalidJSON(t *testing.T) {
	h := NewExchangeRateHandler(nil)

	id := uuid.New()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id.String())

	req := httptest.NewRequest(http.MethodPatch, "/v1/exchange-rates/"+id.String(), strings.NewReader("bad"))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Update(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestExchangeRateHandler_Delete_InvalidID(t *testing.T) {
	h := NewExchangeRateHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodDelete, "/v1/exchange-rates/bad", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Delete(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid exchange rate ID", resp["error"])
}

func TestExchangeRateHandler_Convert_InvalidJSON(t *testing.T) {
	h := NewExchangeRateHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/exchange-rates/convert", strings.NewReader("bad"))
	rr := httptest.NewRecorder()

	h.Convert(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestExchangeRateHandler_Convert_ValidationError_MissingFrom(t *testing.T) {
	h := NewExchangeRateHandler(nil)

	body := `{"amount":100,"to":"USD"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/exchange-rates/convert", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.Convert(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "from")
}

func TestExchangeRateHandler_Convert_ValidationError_MissingTo(t *testing.T) {
	h := NewExchangeRateHandler(nil)

	body := `{"amount":100,"from":"PLN"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/exchange-rates/convert", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.Convert(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "to")
}

// ===========================================================================
// BarcodeHandler tests
// ===========================================================================

func TestBarcodeHandler_Lookup_MissingCode(t *testing.T) {
	h := NewBarcodeHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("code", "")

	req := httptest.NewRequest(http.MethodGet, "/v1/barcode/", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Lookup(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "barcode code is required", resp["error"])
}

func TestBarcodeHandler_PackOrder_InvalidOrderID(t *testing.T) {
	h := NewBarcodeHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodPost, "/v1/orders/bad/pack", strings.NewReader(`{"scanned_items":[]}`))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.PackOrder(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid order ID", resp["error"])
}

func TestBarcodeHandler_PackOrder_InvalidJSON(t *testing.T) {
	h := NewBarcodeHandler(nil)

	id := uuid.New()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id.String())

	req := httptest.NewRequest(http.MethodPost, "/v1/orders/"+id.String()+"/pack", strings.NewReader("bad"))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.PackOrder(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

// ===========================================================================
// BundleHandler tests
// ===========================================================================

func TestBundleHandler_ListComponents_InvalidProductID(t *testing.T) {
	h := NewBundleHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodGet, "/v1/products/bad/bundle/components", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.ListComponents(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid product ID", resp["error"])
}

func TestBundleHandler_AddComponent_InvalidProductID(t *testing.T) {
	h := NewBundleHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	body := `{"component_product_id":"` + uuid.New().String() + `","quantity":1}`
	req := httptest.NewRequest(http.MethodPost, "/v1/products/bad/bundle/components", strings.NewReader(body))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.AddComponent(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid product ID", resp["error"])
}

func TestBundleHandler_AddComponent_InvalidJSON(t *testing.T) {
	h := NewBundleHandler(nil)

	id := uuid.New()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id.String())

	req := httptest.NewRequest(http.MethodPost, "/v1/products/"+id.String()+"/bundle/components", strings.NewReader("bad"))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.AddComponent(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestBundleHandler_AddComponent_ValidationError(t *testing.T) {
	svc := service.NewBundleService(nil, nil, nil, nil)
	h := NewBundleHandler(svc)

	tenantID := uuid.New()
	userID := uuid.New()
	productID := uuid.New()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", productID.String())

	// Missing component_product_id (zero UUID) and zero quantity
	body := `{"quantity":0}`
	req := httptest.NewRequest(http.MethodPost, "/v1/products/"+productID.String()+"/bundle/components", strings.NewReader(body))
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = newContextWithTenantAndUser(ctx, tenantID, userID)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.AddComponent(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "component_product_id")
}

func TestBundleHandler_UpdateComponent_InvalidComponentID(t *testing.T) {
	h := NewBundleHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("componentId", "bad")

	req := httptest.NewRequest(http.MethodPatch, "/v1/products/bundle/components/bad", strings.NewReader(`{"quantity":2}`))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.UpdateComponent(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid component ID", resp["error"])
}

func TestBundleHandler_UpdateComponent_InvalidJSON(t *testing.T) {
	h := NewBundleHandler(nil)

	id := uuid.New()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("componentId", id.String())

	req := httptest.NewRequest(http.MethodPatch, "/v1/products/bundle/components/"+id.String(), strings.NewReader("bad"))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.UpdateComponent(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestBundleHandler_RemoveComponent_InvalidComponentID(t *testing.T) {
	h := NewBundleHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("componentId", "bad")

	req := httptest.NewRequest(http.MethodDelete, "/v1/products/bundle/components/bad", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.RemoveComponent(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid component ID", resp["error"])
}

func TestBundleHandler_GetBundleStock_InvalidProductID(t *testing.T) {
	h := NewBundleHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodGet, "/v1/products/bad/bundle/stock", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.GetBundleStock(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid product ID", resp["error"])
}

// ===========================================================================
// RoleHandler tests
// ===========================================================================

func TestRoleHandler_Create_InvalidJSON(t *testing.T) {
	h := NewRoleHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/roles", strings.NewReader("bad json"))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestRoleHandler_Create_ValidationError(t *testing.T) {
	svc := service.NewRoleService(nil, nil, nil)
	h := NewRoleHandler(svc)

	tenantID := uuid.New()
	userID := uuid.New()

	// Missing name (required field)
	body := `{"permissions":["orders:read"]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/roles", strings.NewReader(body))
	req = req.WithContext(newContextWithTenantAndUser(req.Context(), tenantID, userID))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "name")
}

func TestRoleHandler_Create_ValidationError_InvalidPermission(t *testing.T) {
	svc := service.NewRoleService(nil, nil, nil)
	h := NewRoleHandler(svc)

	tenantID := uuid.New()
	userID := uuid.New()

	// Invalid permission value
	body := `{"name":"Test Role","permissions":["invalid:perm"]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/roles", strings.NewReader(body))
	req = req.WithContext(newContextWithTenantAndUser(req.Context(), tenantID, userID))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "invalid permission")
}

func TestRoleHandler_Get_InvalidID(t *testing.T) {
	h := NewRoleHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodGet, "/v1/roles/bad", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Get(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid role ID", resp["error"])
}

func TestRoleHandler_Update_InvalidID(t *testing.T) {
	h := NewRoleHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodPatch, "/v1/roles/bad", strings.NewReader(`{"name":"Test"}`))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Update(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid role ID", resp["error"])
}

func TestRoleHandler_Update_InvalidJSON(t *testing.T) {
	h := NewRoleHandler(nil)

	id := uuid.New()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id.String())

	req := httptest.NewRequest(http.MethodPatch, "/v1/roles/"+id.String(), strings.NewReader("bad"))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Update(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestRoleHandler_Delete_InvalidID(t *testing.T) {
	h := NewRoleHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodDelete, "/v1/roles/bad", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Delete(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid role ID", resp["error"])
}
