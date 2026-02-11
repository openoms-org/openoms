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

func TestInvoiceHandler_Create_InvalidJSON(t *testing.T) {
	h := NewInvoiceHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/invoices", strings.NewReader("bad"))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestInvoiceHandler_Get_InvalidID(t *testing.T) {
	h := NewInvoiceHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodGet, "/v1/invoices/bad", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Get(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid invoice ID", resp["error"])
}

func TestInvoiceHandler_Cancel_InvalidID(t *testing.T) {
	h := NewInvoiceHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodPost, "/v1/invoices/bad/cancel", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Cancel(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestInvoiceHandler_GetPDF_InvalidID(t *testing.T) {
	h := NewInvoiceHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodGet, "/v1/invoices/bad/pdf", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.GetPDF(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestInvoiceHandler_ListByOrder_InvalidID(t *testing.T) {
	h := NewInvoiceHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")

	req := httptest.NewRequest(http.MethodGet, "/v1/orders/bad/invoices", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.ListByOrder(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestInvoiceHandler_List_InvalidOrderIDFilter(t *testing.T) {
	h := NewInvoiceHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/invoices?order_id=not-uuid", nil)
	rr := httptest.NewRecorder()

	h.List(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid order_id filter", resp["error"])
}

func TestInvoiceHandler_Create_ValidationError(t *testing.T) {
	svc := service.NewInvoiceService(nil, nil, nil, nil, nil, nil)
	h := NewInvoiceHandler(svc)

	tenantID := uuid.New()
	userID := uuid.New()

	// Missing order_id
	body := `{"provider":"fakturownia"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/invoices", strings.NewReader(body))
	req = req.WithContext(newContextWithTenantAndUser(req.Context(), tenantID, userID))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	decErr := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, decErr)
	assert.Contains(t, resp["error"], "order_id")
}
