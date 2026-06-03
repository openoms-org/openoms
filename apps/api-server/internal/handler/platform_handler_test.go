package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

type fakePlatformAudit struct {
	entries []model.PlatformAuditEntry
	err     error
}

func (f *fakePlatformAudit) Log(_ context.Context, entry model.PlatformAuditEntry) error {
	f.entries = append(f.entries, entry)
	return f.err
}

func reqWithPlatformAdmin(admin *model.PlatformAdmin) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/v1/platform/me", nil)
	if admin != nil {
		req = req.WithContext(context.WithValue(req.Context(), middleware.PlatformAdminKey, admin))
	}
	return req
}

func TestPlatformHandler_Me_ReturnsIdentityAndAudits(t *testing.T) {
	userID := uuid.New()
	admin := &model.PlatformAdmin{
		ID:          uuid.New(),
		UserID:      userID,
		Permissions: []string{model.PermPlatformProvidersRead, model.PermPlatformProvidersWrite},
	}
	audit := &fakePlatformAudit{}
	h := NewPlatformHandler(audit)
	rr := httptest.NewRecorder()

	h.Me(rr, reqWithPlatformAdmin(admin))

	require.Equal(t, http.StatusOK, rr.Code)
	var body platformMeResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&body))
	assert.Equal(t, userID.String(), body.UserID)
	assert.ElementsMatch(t, []string{model.PermPlatformProvidersRead, model.PermPlatformProvidersWrite}, body.Permissions)

	require.Len(t, audit.entries, 1, "console access must be audited")
	assert.Equal(t, "platform.session.accessed", audit.entries[0].Action)
	assert.Equal(t, userID, audit.entries[0].ActorUserID)
}

func TestPlatformHandler_Me_NoAdminInContext(t *testing.T) {
	audit := &fakePlatformAudit{}
	h := NewPlatformHandler(audit)
	rr := httptest.NewRecorder()

	h.Me(rr, reqWithPlatformAdmin(nil))

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.Empty(t, audit.entries, "no audit when there is no admin")
}

func TestPlatformHandler_Me_AuditFailureDoesNotBlock(t *testing.T) {
	admin := &model.PlatformAdmin{ID: uuid.New(), UserID: uuid.New(), Permissions: []string{}}
	audit := &fakePlatformAudit{err: errors.New("audit db down")}
	h := NewPlatformHandler(audit)
	rr := httptest.NewRecorder()

	h.Me(rr, reqWithPlatformAdmin(admin))

	require.Equal(t, http.StatusOK, rr.Code, "audit failure must not block the read")
	var body platformMeResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&body))
	assert.NotNil(t, body.Permissions)
	assert.Empty(t, body.Permissions, "permissions should serialize as [] not null")
}
