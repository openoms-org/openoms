package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDocsHandler_ServeSpecSetsStrictCSP(t *testing.T) {
	handler := NewDocsHandler([]byte("openapi: 3.1.0"))
	req := httptest.NewRequest(http.MethodGet, "/v1/openapi.yaml", nil)
	rr := httptest.NewRecorder()

	handler.ServeSpec(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	csp := rr.Header().Get("Content-Security-Policy")
	assert.Contains(t, csp, "default-src 'none'")
	assert.Contains(t, csp, "frame-ancestors 'none'")
	assert.Contains(t, csp, "object-src 'none'")
	assert.NotContains(t, csp, "script-src")
}

func TestDocsHandler_ServeSwaggerUIAllowsOnlyRequiredDocsAssets(t *testing.T) {
	handler := NewDocsHandler([]byte("openapi: 3.1.0"))
	req := httptest.NewRequest(http.MethodGet, "/v1/docs", nil)
	rr := httptest.NewRecorder()

	handler.ServeSwaggerUI(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	csp := rr.Header().Get("Content-Security-Policy")
	assert.Contains(t, csp, "default-src 'none'")
	assert.Contains(t, csp, "frame-ancestors 'none'")
	assert.Contains(t, csp, "object-src 'none'")
	assert.Contains(t, csp, "script-src https://unpkg.com 'unsafe-inline'")
	assert.Contains(t, csp, "style-src https://unpkg.com 'unsafe-inline'")
	assert.Contains(t, csp, "connect-src 'self'")
	assert.NotContains(t, csp, "default-src https:")
	assert.NotContains(t, csp, "*")
}
