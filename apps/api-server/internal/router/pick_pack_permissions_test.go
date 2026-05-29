package router

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/config"
	"github.com/openoms-org/openoms/apps/api-server/internal/handler"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

func TestPickPackRoutesRequireWarehousePermission(t *testing.T) {
	routes := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{name: "create session", method: http.MethodPost, path: "/v1/pick-pack/sessions", body: "bad"},
		{name: "list sessions", method: http.MethodGet, path: "/v1/pick-pack/sessions"},
		{name: "get session", method: http.MethodGet, path: "/v1/pick-pack/sessions/bad"},
		{name: "scan item", method: http.MethodPost, path: "/v1/pick-pack/sessions/bad/scan", body: `{}`},
		{name: "move to packing", method: http.MethodPost, path: "/v1/pick-pack/sessions/bad/move-to-packing"},
		{name: "mark item packed", method: http.MethodPost, path: "/v1/pick-pack/sessions/bad/items/bad/pack", body: `{}`},
		{name: "complete session", method: http.MethodPost, path: "/v1/pick-pack/sessions/bad/complete"},
		{name: "cancel session", method: http.MethodPost, path: "/v1/pick-pack/sessions/bad/cancel"},
	}

	for _, route := range routes {
		t.Run(route.name, func(t *testing.T) {
			router, req := newPickPackPermissionRequest(t, route.method, route.path, route.body, model.User{
				Role:        "member",
				Permissions: []string{model.PermOrdersView, model.PermOrdersEdit},
			})
			rr := httptest.NewRecorder()

			router.ServeHTTP(rr, req)

			require.Equal(t, http.StatusForbidden, rr.Code)
		})
	}
}

func TestPickPackRoutesAllowWarehousePermissionToReachHandlerValidation(t *testing.T) {
	routes := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{name: "create session", method: http.MethodPost, path: "/v1/pick-pack/sessions", body: "bad"},
		{name: "get session", method: http.MethodGet, path: "/v1/pick-pack/sessions/bad"},
		{name: "scan item", method: http.MethodPost, path: "/v1/pick-pack/sessions/bad/scan", body: `{}`},
		{name: "move to packing", method: http.MethodPost, path: "/v1/pick-pack/sessions/bad/move-to-packing"},
		{name: "mark item packed", method: http.MethodPost, path: "/v1/pick-pack/sessions/bad/items/bad/pack", body: `{}`},
		{name: "complete session", method: http.MethodPost, path: "/v1/pick-pack/sessions/bad/complete"},
		{name: "cancel session", method: http.MethodPost, path: "/v1/pick-pack/sessions/bad/cancel"},
	}

	for _, route := range routes {
		t.Run(route.name, func(t *testing.T) {
			router, req := newPickPackPermissionRequest(t, route.method, route.path, route.body, model.User{
				Role:        "member",
				Permissions: []string{model.PermWarehousesManage},
			})
			rr := httptest.NewRecorder()

			router.ServeHTTP(rr, req)

			require.Equal(t, http.StatusBadRequest, rr.Code)
		})
	}
}

func TestPickPackRoutesAllowOwnerAndAdminRoleDefaults(t *testing.T) {
	for _, role := range []string{"owner", "admin"} {
		t.Run(role, func(t *testing.T) {
			router, req := newPickPackPermissionRequest(t, http.MethodPost, "/v1/pick-pack/sessions", "bad", model.User{
				Role: role,
			})
			rr := httptest.NewRecorder()

			router.ServeHTTP(rr, req)

			require.Equal(t, http.StatusBadRequest, rr.Code)
		})
	}
}

func newPickPackPermissionRequest(t *testing.T, method, path, body string, user model.User) (http.Handler, *http.Request) {
	t.Helper()

	user.ID = uuid.New()
	user.TenantID = uuid.New()
	user.Email = "operator@example.com"

	tokenSvc, err := service.NewTokenService("test-jwt-secret-for-pick-pack-rbac")
	require.NoError(t, err)
	token, err := tokenSvc.GenerateAccessToken(user)
	require.NoError(t, err)

	router := New(RouterDeps{
		Config: &config.Config{
			Env:         "development",
			FrontendURL: "http://localhost:3000",
			UploadDir:   t.TempDir(),
			// "full" surface disables the readiness feature gate so these tests
			// exercise only the pick-pack permission gate.
			APISurfaceMode: "full",
		},
		TokenSvc: tokenSvc,
		PickPack: handler.NewPickPackHandler(nil),
	})

	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	if method != http.MethodGet && method != http.MethodHead && method != http.MethodOptions {
		req.AddCookie(&http.Cookie{Name: "csrf_token", Value: "test-csrf-token"})
		req.Header.Set("X-CSRF-Token", "test-csrf-token")
	}
	return router, req
}
