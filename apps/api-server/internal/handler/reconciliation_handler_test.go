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

func TestReconciliation_CreateSettlement_RejectsInvalidJSON(t *testing.T) {
	h := NewReconciliationHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/reconciliation/settlements", strings.NewReader("not json"))
	rr := httptest.NewRecorder()

	h.CreateSettlement(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestReconciliation_CreateSettlement_RejectsMissingProvider(t *testing.T) {
	h := NewReconciliationHandler(nil)

	body := `{"settlement_date":"2024-01-15"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/reconciliation/settlements", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.CreateSettlement(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "provider is required", resp["error"])
}

func TestReconciliation_CreateSettlement_RejectsMissingDate(t *testing.T) {
	h := NewReconciliationHandler(nil)

	body := `{"provider":"stripe"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/reconciliation/settlements", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.CreateSettlement(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "settlement_date is required", resp["error"])
}

func TestReconciliation_GetSettlement_RejectsInvalidUUID(t *testing.T) {
	h := NewReconciliationHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "not-a-uuid")

	req := httptest.NewRequest(http.MethodGet, "/v1/reconciliation/settlements/not-a-uuid", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.GetSettlement(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid settlement ID", resp["error"])
}

func TestReconciliation_AutoMatch_RejectsInvalidUUID(t *testing.T) {
	h := NewReconciliationHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad-id")

	req := httptest.NewRequest(http.MethodPost, "/v1/reconciliation/settlements/bad-id/auto-match", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.AutoMatch(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid settlement ID", resp["error"])
}

func TestReconciliation_ListTransactions_RejectsInvalidSettlementIDFilter(t *testing.T) {
	h := NewReconciliationHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/reconciliation/transactions?settlement_id=not-a-uuid", nil)
	rr := httptest.NewRecorder()

	h.ListTransactions(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid settlement_id filter", resp["error"])
}

func TestReconciliation_ManualMatch_RejectsInvalidTransactionUUID(t *testing.T) {
	h := NewReconciliationHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad-id")

	req := httptest.NewRequest(http.MethodPost, "/v1/reconciliation/transactions/bad-id/match", strings.NewReader(`{"order_id":"`+uuid.New().String()+`"}`))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.ManualMatch(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid transaction ID", resp["error"])
}

func TestReconciliation_ManualMatch_RejectsInvalidJSON(t *testing.T) {
	h := NewReconciliationHandler(nil)

	txID := uuid.New()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", txID.String())

	req := httptest.NewRequest(http.MethodPost, "/v1/reconciliation/transactions/"+txID.String()+"/match", strings.NewReader("not json"))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.ManualMatch(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestReconciliation_ManualMatch_RejectsMissingOrderID(t *testing.T) {
	h := NewReconciliationHandler(nil)

	txID := uuid.New()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", txID.String())

	// order_id is zero UUID when missing from JSON
	req := httptest.NewRequest(http.MethodPost, "/v1/reconciliation/transactions/"+txID.String()+"/match", strings.NewReader(`{}`))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.ManualMatch(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "order_id is required", resp["error"])
}

func TestReconciliation_Unmatch_RejectsInvalidTransactionUUID(t *testing.T) {
	h := NewReconciliationHandler(nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad-id")

	req := httptest.NewRequest(http.MethodPost, "/v1/reconciliation/transactions/bad-id/unmatch", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Unmatch(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid transaction ID", resp["error"])
}

func TestReconciliation_ImportCSV_RejectsMissingProvider(t *testing.T) {
	h := NewReconciliationHandler(nil)

	// Build a minimal multipart form without provider field
	body := &strings.Builder{}
	body.WriteString("------boundary\r\n")
	body.WriteString("Content-Disposition: form-data; name=\"file\"; filename=\"test.csv\"\r\n")
	body.WriteString("Content-Type: text/csv\r\n\r\n")
	body.WriteString("header1,header2\nval1,val2\n")
	body.WriteString("\r\n------boundary--\r\n")

	req := httptest.NewRequest(http.MethodPost, "/v1/reconciliation/import-csv", strings.NewReader(body.String()))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=----boundary")
	rr := httptest.NewRecorder()

	h.ImportCSV(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "provider is required", resp["error"])
}

func TestReconciliation_ImportCSV_RejectsNonMultipartForm(t *testing.T) {
	h := NewReconciliationHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/reconciliation/import-csv", strings.NewReader("plain text"))
	req.Header.Set("Content-Type", "text/plain")
	rr := httptest.NewRecorder()

	h.ImportCSV(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.NotEmpty(t, resp["error"])
}
