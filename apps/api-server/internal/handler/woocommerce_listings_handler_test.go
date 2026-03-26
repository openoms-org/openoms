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

func TestWooCommerceListingsHandler_CreateListing_InvalidProductID(t *testing.T) {
	h := NewWooCommerceListingsHandler(nil, nil, nil, nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("productId", "not-a-uuid")

	req := httptest.NewRequest(http.MethodPost, "/v1/products/not-a-uuid/woocommerce-listings", strings.NewReader(`{"integration_id":"`+uuid.New().String()+`"}`))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.CreateListing(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid product ID", resp["error"])
}

func TestWooCommerceListingsHandler_CreateListing_InvalidJSON(t *testing.T) {
	h := NewWooCommerceListingsHandler(nil, nil, nil, nil)

	productID := uuid.New()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("productId", productID.String())

	req := httptest.NewRequest(http.MethodPost, "/v1/products/"+productID.String()+"/woocommerce-listings", strings.NewReader("not json"))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.CreateListing(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestWooCommerceListingsHandler_CreateListing_MissingIntegrationID(t *testing.T) {
	h := NewWooCommerceListingsHandler(nil, nil, nil, nil)

	productID := uuid.New()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("productId", productID.String())

	req := httptest.NewRequest(http.MethodPost, "/v1/products/"+productID.String()+"/woocommerce-listings", strings.NewReader(`{}`))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.CreateListing(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "integration_id is required", resp["error"])
}

func TestWooCommerceListingsHandler_CreateListing_InvalidIntegrationID(t *testing.T) {
	h := NewWooCommerceListingsHandler(nil, nil, nil, nil)

	productID := uuid.New()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("productId", productID.String())

	req := httptest.NewRequest(http.MethodPost, "/v1/products/"+productID.String()+"/woocommerce-listings", strings.NewReader(`{"integration_id":"not-a-uuid"}`))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.CreateListing(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid integration_id", resp["error"])
}

func TestWooCommerceListingsHandler_CreateListing_MissingProductIDParam(t *testing.T) {
	h := NewWooCommerceListingsHandler(nil, nil, nil, nil)

	rctx := chi.NewRouteContext()
	// No productId URL param set

	req := httptest.NewRequest(http.MethodPost, "/v1/products//woocommerce-listings", strings.NewReader(`{"integration_id":"`+uuid.New().String()+`"}`))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.CreateListing(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}
