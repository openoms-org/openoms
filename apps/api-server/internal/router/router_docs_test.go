package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/config"
	"github.com/openoms-org/openoms/apps/api-server/internal/handler"
)

func docsRouter(t *testing.T, cfg *config.Config) *chi.Mux {
	t.Helper()
	r := chi.NewRouter()
	registerDocsRoutes(r, handler.NewDocsHandler([]byte("openapi: 3.1.0")), cfg)
	return r
}

func TestRegisterDocsRoutes_DisabledInProductionReturns404(t *testing.T) {
	r := docsRouter(t, &config.Config{Env: "production", EnableAPIDocs: false})

	for _, path := range []string{"/v1/openapi.yaml", "/v1/docs"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusNotFound, rr.Code, "%s must be hidden when docs are disabled", path)
	}
}

func TestRegisterDocsRoutes_EnabledViaFlagServes(t *testing.T) {
	r := docsRouter(t, &config.Config{Env: "production", EnableAPIDocs: true})

	for _, path := range []string{"/v1/openapi.yaml", "/v1/docs"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code, "%s should be served when ENABLE_API_DOCS is set", path)
	}
}

func TestRegisterDocsRoutes_DevelopmentServesWithoutFlag(t *testing.T) {
	r := docsRouter(t, &config.Config{Env: "development", EnableAPIDocs: false})

	req := httptest.NewRequest(http.MethodGet, "/v1/docs", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)
}

func TestApiDocsEnabled(t *testing.T) {
	assert.False(t, apiDocsEnabled(nil), "nil config must fail closed")
	assert.False(t, apiDocsEnabled(&config.Config{Env: "production"}), "production without flag must be disabled")
	assert.True(t, apiDocsEnabled(&config.Config{Env: "production", EnableAPIDocs: true}))
	assert.True(t, apiDocsEnabled(&config.Config{Env: "development"}), "development always serves docs")
}
