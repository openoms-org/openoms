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

func TestReturnHandler_Create_InvalidJSON(t *testing.T) {
	h := NewReturnHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/returns", strings.NewReader("bad"))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestReturnHandler_Get_InvalidID(t *testing.T) {
	h := NewReturnHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodGet, "/v1/returns/bad", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Get(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid return ID", resp["error"])
}

func TestReturnHandler_Update_InvalidID(t *testing.T) {
	h := NewReturnHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodPatch, "/v1/returns/bad", strings.NewReader(`{}`))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Update(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestReturnHandler_Update_InvalidJSON(t *testing.T) {
	h := NewReturnHandler(nil)

	id := uuid.New()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id.String())

	req := httptest.NewRequest(http.MethodPatch, "/v1/returns/"+id.String(), strings.NewReader("bad"))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Update(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestReturnHandler_TransitionStatus_InvalidID(t *testing.T) {
	h := NewReturnHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodPost, "/v1/returns/bad/status", strings.NewReader(`{"status":"approved"}`))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.TransitionStatus(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestReturnHandler_TransitionStatus_InvalidJSON(t *testing.T) {
	h := NewReturnHandler(nil)

	id := uuid.New()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id.String())

	req := httptest.NewRequest(http.MethodPost, "/v1/returns/"+id.String()+"/status", strings.NewReader("bad"))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.TransitionStatus(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestReturnHandler_Delete_InvalidID(t *testing.T) {
	h := NewReturnHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodDelete, "/v1/returns/bad", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Delete(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestReturnHandler_List_InvalidOrderIDFilter(t *testing.T) {
	h := NewReturnHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/returns?order_id=not-uuid", nil)
	rr := httptest.NewRecorder()

	h.List(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid order_id filter", resp["error"])
}

func TestReturnHandler_Create_ValidationError(t *testing.T) {
	svc := service.NewReturnService(nil, nil, nil, nil, nil)
	h := NewReturnHandler(svc)

	tenantID := uuid.New()
	userID := uuid.New()

	// Missing order_id
	body := `{"reason":"defective"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/returns", strings.NewReader(body))
	req = req.WithContext(newContextWithTenantAndUser(req.Context(), tenantID, userID))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "order_id")
}

func TestReturnHandler_TransitionStatus_ValidationError(t *testing.T) {
	svc := service.NewReturnService(nil, nil, nil, nil, nil)
	h := NewReturnHandler(svc)

	tenantID := uuid.New()
	userID := uuid.New()
	returnID := uuid.New()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", returnID.String())

	// Empty status should fail validation
	body := `{"status":""}`
	req := httptest.NewRequest(http.MethodPost, "/v1/returns/"+returnID.String()+"/status", strings.NewReader(body))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(newContextWithTenantAndUser(req.Context(), tenantID, userID))
	rr := httptest.NewRecorder()

	h.TransitionStatus(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestReturnHandler_Update_ValidationError(t *testing.T) {
	svc := service.NewReturnService(nil, nil, nil, nil, nil)
	h := NewReturnHandler(svc)

	tenantID := uuid.New()
	userID := uuid.New()
	returnID := uuid.New()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", returnID.String())

	// Empty body - no fields to update
	body := `{}`
	req := httptest.NewRequest(http.MethodPatch, "/v1/returns/"+returnID.String(), strings.NewReader(body))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(newContextWithTenantAndUser(req.Context(), tenantID, userID))
	rr := httptest.NewRecorder()

	h.Update(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}
