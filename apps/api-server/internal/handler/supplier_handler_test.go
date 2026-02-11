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

func TestSupplierHandler_Create_InvalidJSON(t *testing.T) {
	h := NewSupplierHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/suppliers", strings.NewReader("bad"))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestSupplierHandler_Get_InvalidID(t *testing.T) {
	h := NewSupplierHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodGet, "/v1/suppliers/bad", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Get(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid supplier ID", resp["error"])
}

func TestSupplierHandler_Update_InvalidID(t *testing.T) {
	h := NewSupplierHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodPatch, "/v1/suppliers/bad", strings.NewReader(`{}`))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Update(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestSupplierHandler_Update_InvalidJSON(t *testing.T) {
	h := NewSupplierHandler(nil)

	id := uuid.New()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id.String())

	req := httptest.NewRequest(http.MethodPatch, "/v1/suppliers/"+id.String(), strings.NewReader("bad"))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Update(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestSupplierHandler_Delete_InvalidID(t *testing.T) {
	h := NewSupplierHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodDelete, "/v1/suppliers/bad", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Delete(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestSupplierHandler_Sync_InvalidID(t *testing.T) {
	h := NewSupplierHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodPost, "/v1/suppliers/bad/sync", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Sync(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestSupplierHandler_ListProducts_InvalidID(t *testing.T) {
	h := NewSupplierHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodGet, "/v1/suppliers/bad/products", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.ListProducts(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestSupplierHandler_LinkProduct_InvalidID(t *testing.T) {
	h := NewSupplierHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("spid", "bad")

	req := httptest.NewRequest(http.MethodPost, "/v1/suppliers/products/bad/link", strings.NewReader(`{"product_id":"` + uuid.New().String() + `"}`))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.LinkProduct(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestSupplierHandler_LinkProduct_InvalidJSON(t *testing.T) {
	h := NewSupplierHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("spid", uuid.New().String())

	req := httptest.NewRequest(http.MethodPost, "/v1/suppliers/products/"+uuid.New().String()+"/link", strings.NewReader("bad"))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.LinkProduct(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestSupplierHandler_LinkProduct_ValidationError(t *testing.T) {
	h := NewSupplierHandler(nil)

	spid := uuid.New()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("spid", spid.String())

	// Missing product_id (will be zero UUID)
	req := httptest.NewRequest(http.MethodPost, "/v1/suppliers/products/"+spid.String()+"/link", strings.NewReader(`{}`))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.LinkProduct(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestSupplierHandler_Create_ValidationError(t *testing.T) {
	svc := service.NewSupplierService(nil, nil, nil, nil, nil, nil)
	h := NewSupplierHandler(svc)

	tenantID := uuid.New()
	userID := uuid.New()

	// Missing name (required field)
	body := `{"feed_format":"iof"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/suppliers", strings.NewReader(body))
	req = req.WithContext(newContextWithTenantAndUser(req.Context(), tenantID, userID))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "name")
}

func TestSupplierHandler_Update_ValidationError(t *testing.T) {
	svc := service.NewSupplierService(nil, nil, nil, nil, nil, nil)
	h := NewSupplierHandler(svc)

	tenantID := uuid.New()
	userID := uuid.New()
	supplierID := uuid.New()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", supplierID.String())

	// Empty body - no fields to update
	body := `{}`
	req := httptest.NewRequest(http.MethodPatch, "/v1/suppliers/"+supplierID.String(), strings.NewReader(body))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(newContextWithTenantAndUser(req.Context(), tenantID, userID))
	rr := httptest.NewRecorder()

	h.Update(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}
