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

func TestVariantHandler_Create_InvalidJSON(t *testing.T) {
	h := NewVariantHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("productId", uuid.New().String())

	req := httptest.NewRequest(http.MethodPost, "/v1/products/"+uuid.New().String()+"/variants", strings.NewReader("bad"))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestVariantHandler_Create_InvalidProductID(t *testing.T) {
	h := NewVariantHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("productId", "bad")

	req := httptest.NewRequest(http.MethodPost, "/v1/products/bad/variants", strings.NewReader(`{"name":"test"}`))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid product ID", resp["error"])
}

func TestVariantHandler_Get_InvalidID(t *testing.T) {
	h := NewVariantHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodGet, "/v1/products/"+uuid.New().String()+"/variants/bad", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Get(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid variant ID", resp["error"])
}

func TestVariantHandler_Update_InvalidID(t *testing.T) {
	h := NewVariantHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodPatch, "/v1/products/"+uuid.New().String()+"/variants/bad", strings.NewReader(`{}`))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Update(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestVariantHandler_Update_InvalidJSON(t *testing.T) {
	h := NewVariantHandler(nil)

	id := uuid.New()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id.String())

	req := httptest.NewRequest(http.MethodPatch, "/v1/products/"+uuid.New().String()+"/variants/"+id.String(), strings.NewReader("bad"))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Update(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestVariantHandler_Delete_InvalidID(t *testing.T) {
	h := NewVariantHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodDelete, "/v1/products/"+uuid.New().String()+"/variants/bad", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Delete(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestVariantHandler_List_InvalidProductID(t *testing.T) {
	h := NewVariantHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("productId", "bad")

	req := httptest.NewRequest(http.MethodGet, "/v1/products/bad/variants", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.List(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid product ID", resp["error"])
}

func TestVariantHandler_Create_ValidationError(t *testing.T) {
	svc := service.NewVariantService(nil, nil, nil, nil)
	h := NewVariantHandler(svc)

	tenantID := uuid.New()
	userID := uuid.New()
	productID := uuid.New()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("productId", productID.String())

	// Missing name (required field)
	body := `{"stock_quantity":10}`
	req := httptest.NewRequest(http.MethodPost, "/v1/products/"+productID.String()+"/variants", strings.NewReader(body))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(newContextWithTenantAndUser(req.Context(), tenantID, userID))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "name")
}

func TestVariantHandler_Update_ValidationError(t *testing.T) {
	svc := service.NewVariantService(nil, nil, nil, nil)
	h := NewVariantHandler(svc)

	tenantID := uuid.New()
	userID := uuid.New()
	variantID := uuid.New()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", variantID.String())

	// Empty body - no fields to update
	body := `{}`
	req := httptest.NewRequest(http.MethodPatch, "/v1/products/"+uuid.New().String()+"/variants/"+variantID.String(), strings.NewReader(body))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(newContextWithTenantAndUser(req.Context(), tenantID, userID))
	rr := httptest.NewRecorder()

	h.Update(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}
