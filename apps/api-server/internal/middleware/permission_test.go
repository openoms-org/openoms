package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

func TestRequirePermission_AllowsExplicitPermission(t *testing.T) {
	claims := &model.AuthClaims{Permissions: []string{model.PermOrdersView}}
	req := httptest.NewRequest(http.MethodGet, "/orders", nil)
	req = req.WithContext(context.WithValue(req.Context(), ClaimsKey, claims))
	rr := httptest.NewRecorder()
	called := false

	RequirePermission(model.PermOrdersView)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	})).ServeHTTP(rr, req)

	assert.True(t, called)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestRequirePermission_DeniesMissingExplicitPermission(t *testing.T) {
	roleID := uuid.New()
	claims := &model.AuthClaims{Role: "member", RoleID: roleID, Permissions: []string{model.PermOrdersView}}
	req := httptest.NewRequest(http.MethodDelete, "/orders/1", nil)
	req = req.WithContext(context.WithValue(req.Context(), ClaimsKey, claims))
	rr := httptest.NewRecorder()

	RequirePermission(model.PermOrdersDelete)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("handler should not be called")
	})).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestRequirePermission_FallsBackToLegacySystemRoleForOldTokens(t *testing.T) {
	claims := &model.AuthClaims{Role: "owner"}
	req := httptest.NewRequest(http.MethodDelete, "/users/1", nil)
	req = req.WithContext(context.WithValue(req.Context(), ClaimsKey, claims))
	rr := httptest.NewRecorder()
	called := false

	RequirePermission(model.PermUsersManage)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	})).ServeHTTP(rr, req)

	assert.True(t, called)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestSystemRoleAdminPermissionsExcludeUsersManage(t *testing.T) {
	assert.False(t, claimsHasPermission(&model.AuthClaims{Role: "admin"}, model.PermUsersManage))
	assert.True(t, claimsHasPermission(&model.AuthClaims{Role: "owner"}, model.PermUsersManage))
}
