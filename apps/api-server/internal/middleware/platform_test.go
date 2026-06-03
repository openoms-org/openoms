package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

var errNotPlatformAdmin = errors.New("not found")

type fakePlatformLookup struct {
	admins map[uuid.UUID]*model.PlatformAdmin
	err    error // forces a lookup failure regardless of user
}

func (f *fakePlatformLookup) GetByUserID(_ context.Context, userID uuid.UUID) (*model.PlatformAdmin, error) {
	if f.err != nil {
		return nil, f.err
	}
	a, ok := f.admins[userID]
	if !ok {
		return nil, errNotPlatformAdmin
	}
	return a, nil
}

// reqWithUser builds a request whose context carries an authenticated user id,
// as JWTAuth would have set it.
func reqWithUser(userID uuid.UUID) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/v1/platform/me", nil)
	if userID != uuid.Nil {
		req = req.WithContext(context.WithValue(req.Context(), UserIDKey, userID))
	}
	return req
}

func okHandler(called *bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		*called = true
		w.WriteHeader(http.StatusOK)
	})
}

func TestRequirePlatformAdmin_RejectsTenantUser(t *testing.T) {
	// A normal tenant user (owner/admin/member) is authenticated but is NOT in
	// platform_admins. The boundary must reject them, ignoring tenant role.
	lookup := &fakePlatformLookup{admins: map[uuid.UUID]*model.PlatformAdmin{}}
	called := false
	rr := httptest.NewRecorder()

	RequirePlatformAdmin(lookup)(okHandler(&called)).ServeHTTP(rr, reqWithUser(uuid.New()))

	assert.Equal(t, http.StatusForbidden, rr.Code)
	assert.False(t, called, "next handler must not be reached")
}

func TestRequirePlatformAdmin_RejectsMissingUser(t *testing.T) {
	lookup := &fakePlatformLookup{admins: map[uuid.UUID]*model.PlatformAdmin{}}
	called := false
	rr := httptest.NewRecorder()

	RequirePlatformAdmin(lookup)(okHandler(&called)).ServeHTTP(rr, reqWithUser(uuid.Nil))

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.False(t, called)
}

func TestRequirePlatformAdmin_FailsClosedOnLookupError(t *testing.T) {
	lookup := &fakePlatformLookup{err: errors.New("db down")}
	called := false
	rr := httptest.NewRecorder()

	RequirePlatformAdmin(lookup)(okHandler(&called)).ServeHTTP(rr, reqWithUser(uuid.New()))

	assert.Equal(t, http.StatusForbidden, rr.Code, "must deny on lookup error (fail-closed)")
	assert.False(t, called)
}

func TestRequirePlatformAdmin_AllowsPlatformAdminAndSetsContext(t *testing.T) {
	userID := uuid.New()
	admin := &model.PlatformAdmin{ID: uuid.New(), UserID: userID, Permissions: []string{model.PermPlatformProvidersRead}}
	lookup := &fakePlatformLookup{admins: map[uuid.UUID]*model.PlatformAdmin{userID: admin}}

	var seen *model.PlatformAdmin
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = PlatformAdminFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})
	rr := httptest.NewRecorder()

	RequirePlatformAdmin(lookup)(next).ServeHTTP(rr, reqWithUser(userID))

	require.Equal(t, http.StatusOK, rr.Code)
	require.NotNil(t, seen, "platform admin must be placed in context")
	assert.Equal(t, admin.UserID, seen.UserID)
}

// platformAdminCtx returns a request whose context already has a resolved
// platform admin (as RequirePlatformAdmin would set it).
func platformAdminCtx(admin *model.PlatformAdmin) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/v1/platform/secrets", nil)
	return req.WithContext(context.WithValue(req.Context(), PlatformAdminKey, admin))
}

func TestRequirePlatformPermission_RejectsWithoutPermission(t *testing.T) {
	admin := &model.PlatformAdmin{UserID: uuid.New(), Permissions: []string{model.PermPlatformProvidersRead}}
	called := false
	rr := httptest.NewRecorder()

	RequirePlatformPermission(model.PermPlatformProvidersSecrets)(okHandler(&called)).
		ServeHTTP(rr, platformAdminCtx(admin))

	assert.Equal(t, http.StatusForbidden, rr.Code)
	assert.False(t, called, "admin without providers:secrets must not pass")
}

func TestRequirePlatformPermission_AllowsWithPermission(t *testing.T) {
	admin := &model.PlatformAdmin{UserID: uuid.New(), Permissions: []string{model.PermPlatformProvidersSecrets}}
	called := false
	rr := httptest.NewRecorder()

	RequirePlatformPermission(model.PermPlatformProvidersSecrets)(okHandler(&called)).
		ServeHTTP(rr, platformAdminCtx(admin))

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.True(t, called)
}

func TestRequirePlatformPermission_RejectsWhenNoAdminInContext(t *testing.T) {
	called := false
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/platform/secrets", nil)

	RequirePlatformPermission(model.PermPlatformProvidersSecrets)(okHandler(&called)).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
	assert.False(t, called)
}
