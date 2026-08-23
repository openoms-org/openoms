package handler

import (
	"context"
	_ "embed"
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

//go:embed public_return_handler.go
var publicReturnHandlerSource string

func newPublicReturnHandler() *PublicReturnHandler {
	return NewPublicReturnHandler(nil, nil)
}

// TestPublicReturnHandler_CreatesThroughReturnService pins the CORR-05 contract: the
// public endpoint must not write the return row itself. It used to call
// returnRepo.Create directly, which produced no audit entry, no return.created webhook
// and no automation event for customer-submitted returns.
func TestPublicReturnHandler_CreatesThroughReturnService(t *testing.T) {
	assert.NotContains(t, publicReturnHandlerSource, "func (h *PublicReturnHandler) withTenant")
	assert.NotContains(t, publicReturnHandlerSource, "returnRepo.Create",
		"the public endpoint must not insert returns behind the service")
	assert.Contains(t, publicReturnHandlerSource, "h.returnService.CreatePublic(")
}

// ---------------------------------------------------------------------------
// CreatePublicReturn — input validation (before DB)
// ---------------------------------------------------------------------------

func TestPublicReturnHandler_Submit_InvalidJSON(t *testing.T) {
	h := newPublicReturnHandler()

	req := httptest.NewRequest(http.MethodPost, "/v1/public/returns", strings.NewReader("not json"))
	rr := httptest.NewRecorder()

	h.CreatePublicReturn(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)

	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestPublicReturnHandler_Submit_EmptyBody(t *testing.T) {
	h := newPublicReturnHandler()

	req := httptest.NewRequest(http.MethodPost, "/v1/public/returns", strings.NewReader("{}"))
	rr := httptest.NewRecorder()

	h.CreatePublicReturn(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)

	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "order_id is required", resp["error"])
}

func TestPublicReturnHandler_Submit_MissingOrderID(t *testing.T) {
	h := newPublicReturnHandler()

	body := `{"email":"test@example.com","reason":"defective","items":[]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/public/returns", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.CreatePublicReturn(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)

	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "order_id is required", resp["error"])
}

func TestPublicReturnHandler_Submit_MissingEmail(t *testing.T) {
	h := newPublicReturnHandler()

	body := `{"order_id":"` + uuid.New().String() + `","reason":"defective","items":[]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/public/returns", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.CreatePublicReturn(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)

	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "email is required", resp["error"])
}

func TestPublicReturnHandler_Submit_MissingReason(t *testing.T) {
	h := newPublicReturnHandler()

	body := `{"order_id":"` + uuid.New().String() + `","email":"test@example.com","items":[]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/public/returns", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.CreatePublicReturn(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)

	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "reason is required", resp["error"])
}

func TestPublicReturnHandler_Submit_MissingFields(t *testing.T) {
	h := newPublicReturnHandler()

	tests := []struct {
		name     string
		body     string
		expected string
	}{
		{
			name:     "empty body",
			body:     `{}`,
			expected: "order_id is required",
		},
		{
			name:     "only order_id",
			body:     `{"order_id":"` + uuid.New().String() + `"}`,
			expected: "email is required",
		},
		{
			name:     "only order_id and email",
			body:     `{"order_id":"` + uuid.New().String() + `","email":"a@b.com"}`,
			expected: "reason is required",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/v1/public/returns", strings.NewReader(tc.body))
			rr := httptest.NewRecorder()

			h.CreatePublicReturn(rr, req)

			assert.Equal(t, http.StatusBadRequest, rr.Code)

			var resp map[string]string
			err := json.NewDecoder(rr.Body).Decode(&resp)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, resp["error"])
		})
	}
}

func TestPublicReturnHandler_Submit_InvalidOrderIDFormat(t *testing.T) {
	h := newPublicReturnHandler()

	body := `{"order_id":"not-a-uuid","email":"test@example.com","reason":"defective"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/public/returns", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.CreatePublicReturn(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)

	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid order_id format", resp["error"])
}

func TestPublicReturnHandler_Submit_ReasonTooLong(t *testing.T) {
	h := newPublicReturnHandler()

	longReason := strings.Repeat("a", 2001)
	body := `{"order_id":"` + uuid.New().String() + `","email":"test@example.com","reason":"` + longReason + `"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/public/returns", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.CreatePublicReturn(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)

	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "reason")
}

func TestPublicReturnHandler_Submit_NotesTooLong(t *testing.T) {
	h := newPublicReturnHandler()

	longNotes := strings.Repeat("b", 5001)
	body := `{"order_id":"` + uuid.New().String() + `","email":"test@example.com","reason":"broken","notes":"` + longNotes + `"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/public/returns", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.CreatePublicReturn(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)

	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "notes")
}

// ---------------------------------------------------------------------------
// GetStatusByToken — input validation (before DB)
// ---------------------------------------------------------------------------

func TestPublicReturnHandler_GetStatusByToken_MissingToken(t *testing.T) {
	h := newPublicReturnHandler()

	// No token URL param set — chi returns ""
	rctx := chi.NewRouteContext()
	// Intentionally not adding "token" param

	req := httptest.NewRequest(http.MethodGet, "/v1/public/returns//status", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.GetStatusByToken(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)

	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "token is required", resp["error"])
}

func TestPublicReturnHandler_GetStatusByToken_NoTokenParam(t *testing.T) {
	h := newPublicReturnHandler()

	// RouteContext exists but has no "token" key at all
	rctx := chi.NewRouteContext()

	req := httptest.NewRequest(http.MethodGet, "/v1/public/returns//status", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.GetStatusByToken(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)

	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "token is required", resp["error"])
}

// ---------------------------------------------------------------------------
// GetByToken — input validation (before DB)
// ---------------------------------------------------------------------------

func TestPublicReturnHandler_GetByToken_MissingToken(t *testing.T) {
	h := newPublicReturnHandler()

	rctx := chi.NewRouteContext()
	// No "token" param

	req := httptest.NewRequest(http.MethodGet, "/v1/public/returns/", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.GetByToken(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)

	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "token is required", resp["error"])
}

func TestPublicReturnHandler_GetByToken_NoTokenParam(t *testing.T) {
	h := newPublicReturnHandler()

	// RouteContext exists but has no "token" key at all
	rctx := chi.NewRouteContext()

	req := httptest.NewRequest(http.MethodGet, "/v1/public/returns/", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.GetByToken(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)

	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "token is required", resp["error"])
}
