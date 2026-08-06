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

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
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

func TestBuildWooListingData_UsesCanonicalAvailableStock(t *testing.T) {
	// Legacy stock_quantity is not decremented on shipment, so it can overstate
	// what is really on the shelf. The listing must carry the canonical value.
	product := &model.Product{
		Name:           "Widget",
		Price:          19.99,
		StockQuantity:  42,
		AvailableStock: 7,
	}

	data := buildWooListingData(product, createWooListingRequest{})

	assert.Equal(t, 7, data["stock_quantity"])
	assert.Equal(t, true, data["manage_stock"])
}

func TestBuildWooListingData_StockOverrideWinsOverAvailableStock(t *testing.T) {
	product := &model.Product{
		Name:           "Widget",
		Price:          19.99,
		StockQuantity:  42,
		AvailableStock: 7,
	}
	override := 3

	data := buildWooListingData(product, createWooListingRequest{StockOverride: &override})

	assert.Equal(t, 3, data["stock_quantity"])
}
