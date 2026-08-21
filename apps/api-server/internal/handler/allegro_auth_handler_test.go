package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	allegrosdk "github.com/openoms-org/openoms/packages/allegro-go-sdk"

	"github.com/openoms-org/openoms/apps/api-server/internal/config"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

type recordingAllegroOAuthStore struct {
	mu           sync.Mutex
	integrations []model.Integration
	updatedJSON  []byte
	updateCalled bool
	updateDone   chan struct{}
	updateErr    error
	respondAfter func()
	createCalled bool
}

func (s *recordingAllegroOAuthStore) GetDecryptedCredentialsByProvider(context.Context, uuid.UUID, string) ([]byte, *model.Integration, error) {
	return nil, nil, service.ErrIntegrationNotFound
}

func (s *recordingAllegroOAuthStore) List(context.Context, uuid.UUID) ([]model.Integration, error) {
	return s.integrations, nil
}

func (s *recordingAllegroOAuthStore) Update(_ context.Context, _, _ uuid.UUID, req model.UpdateIntegrationRequest, _ uuid.UUID, _ string) (*model.Integration, error) {
	s.mu.Lock()
	s.updateCalled = true
	if req.Credentials != nil {
		s.updatedJSON = append([]byte(nil), *req.Credentials...)
	}
	if s.updateDone != nil {
		close(s.updateDone)
	}
	err := s.updateErr
	integ := s.integrations[0]
	now := time.Now()
	integ.UpdatedAt = now
	s.mu.Unlock()
	if s.respondAfter != nil {
		s.respondAfter()
	}
	if err != nil {
		return nil, err
	}
	return &integ, nil
}

func (s *recordingAllegroOAuthStore) Create(context.Context, uuid.UUID, model.CreateIntegrationRequest, uuid.UUID, string) (*model.Integration, error) {
	s.createCalled = true
	return nil, errors.New("create must not run when an allegro integration exists")
}

func TestHandleCallback_PersistsNewTokensBeforeReturningSuccess(t *testing.T) {
	store := NewMemoryOAuthStateStore()
	defer store.Close()

	tenantID := uuid.New()
	userID := uuid.New()
	integID := uuid.MustParse("66ed6658-0000-0000-0000-000000000001")
	oauthStore := &recordingAllegroOAuthStore{
		integrations: []model.Integration{{
			ID:        integID,
			TenantID:  tenantID,
			Provider:  "allegro",
			Status:    "error",
			UpdatedAt: time.Date(2026, 8, 20, 8, 0, 0, 0, time.UTC),
		}},
		updateDone: make(chan struct{}),
	}

	require.NoError(t, store.Save(context.Background(), "oauth-state", &OAuthState{
		ExpiresAt:    time.Now().Add(10 * time.Minute),
		TenantID:     tenantID,
		UserID:       userID,
		Provider:     "allegro",
		ClientID:     "cid",
		ClientSecret: "csecret",
		Sandbox:      true,
	}, 10*time.Minute))

	h := NewAllegroAuthHandler(&config.Config{FrontendURL: "http://localhost:3000"}, nil, nil, store)
	h.oauthStore = oauthStore
	h.exchangeCode = func(context.Context, string, string, bool, string, string) (*allegrosdk.TokenResponse, error) {
		return &allegrosdk.TokenResponse{
			AccessToken:  "oauth-at",
			RefreshToken: "oauth-rt",
			ExpiresIn:    3600,
		}, nil
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/integrations/allegro/callback", strings.NewReader(`{"code":"auth-code","state":"oauth-state"}`))
	req = req.WithContext(newContextWithTenantAndUser(req.Context(), tenantID, userID))
	rr := httptest.NewRecorder()

	h.HandleCallback(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	select {
	case <-oauthStore.updateDone:
	default:
		t.Fatal("reconnect success returned before credentials were written")
	}
	require.True(t, oauthStore.updateCalled)
	assert.False(t, oauthStore.createCalled)

	var creds map[string]any
	require.NoError(t, json.Unmarshal(oauthStore.updatedJSON, &creds))
	assert.Equal(t, "oauth-at", creds["access_token"])
	assert.Equal(t, "oauth-rt", creds["refresh_token"])
	assert.Equal(t, "cid", creds["client_id"])
	assert.Equal(t, true, creds["sandbox"])
	assert.NotEmpty(t, creds["token_expiry"])

	var resp model.Integration
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.False(t, resp.UpdatedAt.Equal(time.Date(2026, 8, 20, 8, 0, 0, 0, time.UTC)), "updated_at must change after persist")
}

func TestHandleCallback_WriteFailureIsNotReportedAsSuccess(t *testing.T) {
	store := NewMemoryOAuthStateStore()
	defer store.Close()

	tenantID := uuid.New()
	userID := uuid.New()
	oauthStore := &recordingAllegroOAuthStore{
		integrations: []model.Integration{{
			ID:       uuid.New(),
			TenantID: tenantID,
			Provider: "allegro",
			Status:   "active",
		}},
		updateErr: errors.New("commit failed"),
	}

	require.NoError(t, store.Save(context.Background(), "oauth-state", &OAuthState{
		ExpiresAt:    time.Now().Add(10 * time.Minute),
		TenantID:     tenantID,
		UserID:       userID,
		Provider:     "allegro",
		ClientID:     "cid",
		ClientSecret: "csecret",
	}, 10*time.Minute))

	h := NewAllegroAuthHandler(&config.Config{FrontendURL: "http://localhost:3000"}, nil, nil, store)
	h.oauthStore = oauthStore
	h.exchangeCode = func(context.Context, string, string, bool, string, string) (*allegrosdk.TokenResponse, error) {
		return &allegrosdk.TokenResponse{AccessToken: "oauth-at", RefreshToken: "oauth-rt", ExpiresIn: 3600}, nil
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/integrations/allegro/callback", strings.NewReader(`{"code":"auth-code","state":"oauth-state"}`))
	req = req.WithContext(newContextWithTenantAndUser(req.Context(), tenantID, userID))
	rr := httptest.NewRecorder()

	h.HandleCallback(rr, req)

	assert.GreaterOrEqual(t, rr.Code, 500)
	assert.NotEqual(t, http.StatusOK, rr.Code)
}

func TestAllegroAuthHandler_HandleCallback_InvalidJSON(t *testing.T) {
	store := NewMemoryOAuthStateStore()
	defer store.Close()
	h := NewAllegroAuthHandler(nil, nil, nil, store)

	req := httptest.NewRequest(http.MethodPost, "/v1/integrations/allegro/callback", strings.NewReader("bad"))
	rr := httptest.NewRecorder()

	h.HandleCallback(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestAllegroAuthHandler_HandleCallback_MissingCode(t *testing.T) {
	store := NewMemoryOAuthStateStore()
	defer store.Close()
	h := NewAllegroAuthHandler(nil, nil, nil, store)

	body := `{"code":"","state":"abc"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/integrations/allegro/callback", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.HandleCallback(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "code is required", resp["error"])
}
