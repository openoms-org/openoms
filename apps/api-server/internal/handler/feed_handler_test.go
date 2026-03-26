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

func TestNewFeedHandler(t *testing.T) {
	h := NewFeedHandler(nil)
	assert.NotNil(t, h)
}

func TestFeedHandler_UpdateConfig_EmptyBody(t *testing.T) {
	h := NewFeedHandler(nil)

	req := httptest.NewRequest(http.MethodPut, "/v1/feeds/config", strings.NewReader(""))
	rr := httptest.NewRecorder()

	h.UpdateConfig(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestFeedHandler_RegenerateToken_EmptyFeedType(t *testing.T) {
	h := NewFeedHandler(nil)

	body := `{"feed_type":""}`
	req := httptest.NewRequest(http.MethodPost, "/v1/feeds/regenerate-token", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.RegenerateToken(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "feed_type must be 'ceneo' or 'google'", resp["error"])
}

func TestFeedHandler_RegenerateToken_MissingFeedType(t *testing.T) {
	h := NewFeedHandler(nil)

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/v1/feeds/regenerate-token", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.RegenerateToken(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestFeedHandler_RegenerateToken_EmptyBody(t *testing.T) {
	h := NewFeedHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/feeds/regenerate-token", strings.NewReader(""))
	rr := httptest.NewRecorder()

	h.RegenerateToken(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestFeedHandler_ServeCeneoFeed_EmptyTenantID(t *testing.T) {
	h := NewFeedHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("tenant_id", "")
	rctx.URLParams.Add("token", "some-token")

	req := httptest.NewRequest(http.MethodGet, "/v1/feeds/ceneo//some-token", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.ServeCeneoFeed(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestFeedHandler_ServeCeneoFeed_MissingParams(t *testing.T) {
	h := NewFeedHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/feeds/ceneo/", nil)
	rr := httptest.NewRecorder()

	h.ServeCeneoFeed(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestFeedHandler_ServeGoogleFeed_EmptyTenantID(t *testing.T) {
	h := NewFeedHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("tenant_id", "")
	rctx.URLParams.Add("token", "some-token")

	req := httptest.NewRequest(http.MethodGet, "/v1/feeds/google//some-token", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.ServeGoogleFeed(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestFeedHandler_RegenerateToken_UnsupportedFeedTypeAmazon(t *testing.T) {
	h := NewFeedHandler(nil)

	body := `{"feed_type":"amazon"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/feeds/regenerate-token", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.RegenerateToken(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "feed_type must be 'ceneo' or 'google'", resp["error"])
}

func TestFeedHandler_ServeCeneoFeed_EmptyTokenWithValidTenantID(t *testing.T) {
	h := NewFeedHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("tenant_id", "11111111-1111-1111-1111-111111111111")
	rctx.URLParams.Add("token", "")

	req := httptest.NewRequest(http.MethodGet, "/v1/feeds/ceneo/11111111-1111-1111-1111-111111111111/", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.ServeCeneoFeed(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "token is required", resp["error"])
}

func TestFeedHandler_ServeGoogleFeed_EmptyTokenWithValidTenantID(t *testing.T) {
	h := NewFeedHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("tenant_id", "11111111-1111-1111-1111-111111111111")
	rctx.URLParams.Add("token", "")

	req := httptest.NewRequest(http.MethodGet, "/v1/feeds/google/11111111-1111-1111-1111-111111111111/", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.ServeGoogleFeed(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "token is required", resp["error"])
}
