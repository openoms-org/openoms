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

// --- DeactivateOffer ---

func TestAllegroAccountHandler_DeactivateOffer_MissingOfferID(t *testing.T) {
	h := NewAllegroAccountHandler(nil, nil)

	// Without chi route context, chi.URLParam returns empty string
	req := httptest.NewRequest(http.MethodPost, "/v1/integrations/allegro/offers/offer-123/deactivate", nil)
	rr := httptest.NewRecorder()

	h.DeactivateOffer(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "offerId is required", resp["error"])
}

func TestAllegroAccountHandler_DeactivateOffer_EmptyOfferIDInRoute(t *testing.T) {
	h := NewAllegroAccountHandler(nil, nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("offerId", "")

	req := httptest.NewRequest(http.MethodPost, "/v1/integrations/allegro/offers//deactivate", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.DeactivateOffer(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "offerId is required", resp["error"])
}

// --- ActivateOffer ---

func TestAllegroAccountHandler_ActivateOffer_MissingOfferID(t *testing.T) {
	h := NewAllegroAccountHandler(nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/integrations/allegro/offers//activate", nil)
	rr := httptest.NewRecorder()

	h.ActivateOffer(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "offerId is required", resp["error"])
}

func TestAllegroAccountHandler_ActivateOffer_EmptyOfferIDInRoute(t *testing.T) {
	h := NewAllegroAccountHandler(nil, nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("offerId", "")

	req := httptest.NewRequest(http.MethodPost, "/v1/integrations/allegro/offers//activate", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.ActivateOffer(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "offerId is required", resp["error"])
}

// --- UpdateOfferStock ---

func TestAllegroAccountHandler_UpdateOfferStock_MissingOfferID(t *testing.T) {
	h := NewAllegroAccountHandler(nil, nil)

	req := httptest.NewRequest(http.MethodPatch, "/v1/integrations/allegro/offers//stock", strings.NewReader(`{"quantity":10}`))
	rr := httptest.NewRecorder()

	h.UpdateOfferStock(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "offerId is required", resp["error"])
}

func TestAllegroAccountHandler_UpdateOfferStock_InvalidJSON(t *testing.T) {
	h := NewAllegroAccountHandler(nil, nil)

	rctx := newChiContextWithParam("offerId", "offer-123")
	req := httptest.NewRequest(http.MethodPatch, "/v1/integrations/allegro/offers/offer-123/stock", strings.NewReader("bad"))
	req = req.WithContext(rctx)
	rr := httptest.NewRecorder()

	h.UpdateOfferStock(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestAllegroAccountHandler_UpdateOfferStock_EmptyBody(t *testing.T) {
	h := NewAllegroAccountHandler(nil, nil)

	rctx := newChiContextWithParam("offerId", "offer-123")
	req := httptest.NewRequest(http.MethodPatch, "/v1/integrations/allegro/offers/offer-123/stock", strings.NewReader(""))
	req = req.WithContext(rctx)
	rr := httptest.NewRecorder()

	h.UpdateOfferStock(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

// --- UpdateOfferPrice ---

func TestAllegroAccountHandler_UpdateOfferPrice_MissingOfferID(t *testing.T) {
	h := NewAllegroAccountHandler(nil, nil)

	req := httptest.NewRequest(http.MethodPatch, "/v1/integrations/allegro/offers//price", strings.NewReader(`{"amount":29.99}`))
	rr := httptest.NewRecorder()

	h.UpdateOfferPrice(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "offerId is required", resp["error"])
}

func TestAllegroAccountHandler_UpdateOfferPrice_InvalidJSON(t *testing.T) {
	h := NewAllegroAccountHandler(nil, nil)

	rctx := newChiContextWithParam("offerId", "offer-456")
	req := httptest.NewRequest(http.MethodPatch, "/v1/integrations/allegro/offers/offer-456/price", strings.NewReader("not json"))
	req = req.WithContext(rctx)
	rr := httptest.NewRecorder()

	h.UpdateOfferPrice(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestAllegroAccountHandler_UpdateOfferPrice_EmptyBody(t *testing.T) {
	h := NewAllegroAccountHandler(nil, nil)

	rctx := newChiContextWithParam("offerId", "offer-456")
	req := httptest.NewRequest(http.MethodPatch, "/v1/integrations/allegro/offers/offer-456/price", strings.NewReader(""))
	req = req.WithContext(rctx)
	rr := httptest.NewRecorder()

	h.UpdateOfferPrice(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}
