package router

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/config"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// readinessRequest builds the router with the given APISurfaceMode and an owner
// token so that role/permission gates always pass and the readiness feature gate
// is the only thing that can short-circuit the request with a 404. Handler deps
// are left nil; the test paths target GET list endpoints whose groups are
// registered unconditionally so the feature-gate middleware actually runs.
func readinessRequest(t *testing.T, mode, method, path string) (http.Handler, *http.Request) {
	t.Helper()

	user := model.User{ID: uuid.New(), TenantID: uuid.New(), Email: "owner@example.com", Role: "owner"}
	tokenSvc, err := service.NewTokenService("test-jwt-secret-for-readiness-gate")
	require.NoError(t, err)
	token, err := tokenSvc.GenerateAccessToken(user)
	require.NoError(t, err)

	r := New(RouterDeps{
		Config:   &config.Config{Env: "development", FrontendURL: "http://localhost:3000", UploadDir: t.TempDir(), APISurfaceMode: mode},
		TokenSvc: tokenSvc,
	})

	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	return r, req
}

// isFeatureGate404 reports whether the response is the readiness gate's 404
// (body {"error":"feature_not_available"}) rather than chi's generic NotFound.
func isFeatureGate404(t *testing.T, rr *httptest.ResponseRecorder) bool {
	t.Helper()
	if rr.Code != http.StatusNotFound {
		return false
	}
	var body map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		return false
	}
	return body["error"] == "feature_not_available"
}

func TestReadinessGate_ClientReadyBlocksNonReady(t *testing.T) {
	// GET list endpoints on gated groups that are registered unconditionally, so the
	// feature gate runs even with nil handler deps (the gate short-circuits first).
	for _, p := range []string{"/v1/repricing/rules", "/v1/reconciliation/settlements"} {
		t.Run(p, func(t *testing.T) {
			r, req := readinessRequest(t, "client-ready", http.MethodGet, p)
			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)
			require.Equal(t, http.StatusNotFound, rr.Code)
			require.True(t, isFeatureGate404(t, rr), "expected feature_not_available gate 404, got body %q", rr.Body.String())
		})
	}
}

func TestReadinessGate_ExcludedRoutesPassInClientReady(t *testing.T) {
	// /v1/stats is shared by the ready home dashboard -> must NOT be gated. With nil
	// deps the handler panics and the Recoverer returns 500, but it must not be the
	// feature gate's 404.
	r, req := readinessRequest(t, "client-ready", http.MethodGet, "/v1/stats/dashboard")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	require.False(t, isFeatureGate404(t, rr), "stats must not be gated, got %d body %q", rr.Code, rr.Body.String())
}

func TestReadinessGate_FullModeAllowsNonReady(t *testing.T) {
	// In full mode the gate passes; the request reaches the (nil-dep) handler, which
	// panics and the Recoverer turns it into a 500. The key assertion is that it is
	// NOT the feature gate's 404.
	r, req := readinessRequest(t, "full", http.MethodGet, "/v1/repricing/rules")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	require.False(t, isFeatureGate404(t, rr), "full mode must not gate repricing, got %d body %q", rr.Code, rr.Body.String())
}
