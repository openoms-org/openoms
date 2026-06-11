package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

// TestSegmentHandler_RefreshSegment_InvalidID verifies the manual refresh endpoint
// rejects a malformed segment id with 400 before any service call (so a nil service
// is safe — the uuid parse fails first).
func TestSegmentHandler_RefreshSegment_InvalidID(t *testing.T) {
	h := NewSegmentHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/segments/not-a-uuid/refresh", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "not-a-uuid")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rec := httptest.NewRecorder()
	h.RefreshSegment(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
