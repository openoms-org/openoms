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

func TestIntegrationHandler_Create_InvalidJSON(t *testing.T) {
	h := NewIntegrationHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/integrations", strings.NewReader("not json"))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestIntegrationHandler_Get_InvalidID(t *testing.T) {
	h := NewIntegrationHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "not-a-uuid")

	req := httptest.NewRequest(http.MethodGet, "/v1/integrations/not-a-uuid", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Get(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid integration ID", resp["error"])
}

func TestIntegrationHandler_Update_InvalidID(t *testing.T) {
	h := NewIntegrationHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodPatch, "/v1/integrations/bad", strings.NewReader(`{"status":"active"}`))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Update(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestIntegrationHandler_Update_InvalidJSON(t *testing.T) {
	h := NewIntegrationHandler(nil)

	id := uuid.New()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id.String())

	req := httptest.NewRequest(http.MethodPatch, "/v1/integrations/"+id.String(), strings.NewReader("bad"))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Update(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestIntegrationHandler_Delete_InvalidID(t *testing.T) {
	h := NewIntegrationHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodDelete, "/v1/integrations/bad", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Delete(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestIntegrationHandler_Create_ValidationError(t *testing.T) {
	key := make([]byte, 32)
	svc := service.NewIntegrationService(nil, nil, nil, key)
	h := NewIntegrationHandler(svc)

	tenantID := uuid.New()
	userID := uuid.New()

	// Missing provider
	body := `{"credentials":{"key":"val"}}`
	req := httptest.NewRequest(http.MethodPost, "/v1/integrations", strings.NewReader(body))
	req = req.WithContext(newContextWithTenantAndUser(req.Context(), tenantID, userID))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "provider")
}

func TestIntegrationHandler_Update_ValidationError(t *testing.T) {
	key := make([]byte, 32)
	svc := service.NewIntegrationService(nil, nil, nil, key)
	h := NewIntegrationHandler(svc)

	tenantID := uuid.New()
	userID := uuid.New()
	id := uuid.New()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id.String())

	// Empty body - no fields to update
	body := `{}`
	req := httptest.NewRequest(http.MethodPatch, "/v1/integrations/"+id.String(), strings.NewReader(body))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(newContextWithTenantAndUser(req.Context(), tenantID, userID))
	rr := httptest.NewRecorder()

	h.Update(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}
