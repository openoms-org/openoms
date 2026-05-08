package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateOAuthStateBinding(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()

	tests := []struct {
		name       string
		state      *OAuthState
		provider   string
		wantStatus int
		wantOK     bool
	}{
		{
			name: "valid binding",
			state: &OAuthState{
				TenantID: tenantID,
				UserID:   userID,
				Provider: "allegro",
			},
			provider:   "allegro",
			wantStatus: http.StatusOK,
			wantOK:     true,
		},
		{
			name: "tenant mismatch",
			state: &OAuthState{
				TenantID: uuid.New(),
				UserID:   userID,
				Provider: "allegro",
			},
			provider:   "allegro",
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "user mismatch",
			state: &OAuthState{
				TenantID: tenantID,
				UserID:   uuid.New(),
				Provider: "allegro",
			},
			provider:   "allegro",
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "provider mismatch",
			state: &OAuthState{
				TenantID: tenantID,
				UserID:   userID,
				Provider: "olx",
			},
			provider:   "allegro",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/callback", nil)
			req = req.WithContext(newContextWithTenantAndUser(req.Context(), tenantID, userID))
			rr := httptest.NewRecorder()

			ok := validateOAuthStateBinding(rr, req, tt.state, tt.provider)

			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.wantStatus, rr.Code)
		})
	}
}

func TestAllegroAuthHandler_HandleCallback_RejectsStateFromDifferentTenant(t *testing.T) {
	store := NewMemoryOAuthStateStore()
	state := "state-from-tenant-a"
	tenantA := uuid.New()
	tenantB := uuid.New()
	userID := uuid.New()

	err := store.Save(context.Background(), state, &OAuthState{
		ExpiresAt:    time.Now().Add(10 * time.Minute),
		TenantID:     tenantA,
		UserID:       userID,
		Provider:     "allegro",
		ClientID:     "client-id",
		ClientSecret: "client-secret",
	}, 10*time.Minute)
	require.NoError(t, err)

	h := NewAllegroAuthHandler(nil, nil, nil, store)
	req := httptest.NewRequest(http.MethodPost, "/v1/integrations/allegro/callback", strings.NewReader(`{"code":"code","state":"state-from-tenant-a"}`))
	req = req.WithContext(newContextWithTenantAndUser(req.Context(), tenantB, userID))
	rr := httptest.NewRecorder()

	h.HandleCallback(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err = json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid or expired state parameter", resp["error"])

	remaining, err := store.Load(context.Background(), state)
	require.NoError(t, err)
	assert.Nil(t, remaining, "mismatched OAuth state should be consumed to prevent replay")
}
