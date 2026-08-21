package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

type memoryAPITokenStore struct {
	byHash map[string]*model.APIToken
	byID   map[uuid.UUID]*model.APIToken
}

func newMemoryAPITokenStore() *memoryAPITokenStore {
	return &memoryAPITokenStore{
		byHash: map[string]*model.APIToken{},
		byID:   map[uuid.UUID]*model.APIToken{},
	}
}

func (s *memoryAPITokenStore) Insert(_ context.Context, _ uuid.UUID, token *model.APIToken) error {
	cp := *token
	s.byHash[token.TokenHash] = &cp
	s.byID[token.ID] = &cp
	return nil
}

func (s *memoryAPITokenStore) ListByUser(_ context.Context, tenantID, userID uuid.UUID) ([]model.APIToken, error) {
	out := make([]model.APIToken, 0)
	for _, tok := range s.byID {
		if tok.TenantID == tenantID && tok.UserID == userID && tok.RevokedAt == nil {
			out = append(out, *tok)
		}
	}
	return out, nil
}

func (s *memoryAPITokenStore) Revoke(_ context.Context, tenantID, userID, id uuid.UUID) (bool, error) {
	tok, ok := s.byID[id]
	if !ok || tok.TenantID != tenantID || tok.UserID != userID || tok.RevokedAt != nil {
		return false, nil
	}
	now := time.Now()
	tok.RevokedAt = &now
	return true, nil
}

func (s *memoryAPITokenStore) FindActiveByHash(_ context.Context, tokenHash string) (*model.APIToken, error) {
	tok, ok := s.byHash[tokenHash]
	if !ok || tok.RevokedAt != nil {
		return nil, nil
	}
	cp := *tok
	return &cp, nil
}

func (s *memoryAPITokenStore) TouchLastUsed(_ context.Context, _ uuid.UUID, id uuid.UUID) error {
	if tok, ok := s.byID[id]; ok {
		now := time.Now()
		tok.LastUsedAt = &now
	}
	return nil
}

type stubAPITokenUsers struct {
	users map[uuid.UUID]*model.User
}

func (s stubAPITokenUsers) FindByID(_ context.Context, tenantID, id uuid.UUID) (*model.User, error) {
	u, ok := s.users[id]
	if !ok || u.TenantID != tenantID {
		return nil, nil
	}
	cp := *u
	return &cp, nil
}

func testAPITokenService(store *memoryAPITokenStore, users stubAPITokenUsers) *APITokenService {
	return NewAPITokenService(store, users, nil)
}

func TestAPITokenService_StoresHashNotPlaintext(t *testing.T) {
	store := newMemoryAPITokenStore()
	tenantID := uuid.New()
	ownerID := uuid.New()
	svc := testAPITokenService(store, stubAPITokenUsers{})

	created, err := svc.Create(context.Background(), tenantID, ownerID, "owner", model.CreateAPITokenRequest{Name: "allegro-sync"}, ownerID, "127.0.0.1")
	require.NoError(t, err)
	require.NotNil(t, created)
	require.NotEmpty(t, created.Token)
	assert.True(t, strings.HasPrefix(created.Token, "oms_"))

	require.Len(t, store.byID, 1)
	var stored *model.APIToken
	for _, tok := range store.byID {
		stored = tok
	}
	assert.NotEqual(t, created.Token, stored.TokenHash)
	assert.Equal(t, model.HashAPIToken(created.Token), stored.TokenHash)
	assert.NotContains(t, stored.TokenHash, created.Token)
}

func TestAPITokenService_SecretShownOnceOnCreateNotOnList(t *testing.T) {
	store := newMemoryAPITokenStore()
	tenantID := uuid.New()
	ownerID := uuid.New()
	svc := testAPITokenService(store, stubAPITokenUsers{})

	created, err := svc.Create(context.Background(), tenantID, ownerID, "owner", model.CreateAPITokenRequest{Name: "cli"}, ownerID, "127.0.0.1")
	require.NoError(t, err)
	assert.NotEmpty(t, created.Token)

	listed, err := svc.List(context.Background(), tenantID, ownerID, "owner")
	require.NoError(t, err)
	require.Len(t, listed, 1)
	assert.Equal(t, created.ID, listed[0].ID)
	assert.Equal(t, "cli", listed[0].Name)
	assert.Empty(t, listed[0].TokenHash)
}

func TestAPITokenService_RevokeRejectsBearer(t *testing.T) {
	store := newMemoryAPITokenStore()
	tenantID := uuid.New()
	ownerID := uuid.New()
	users := stubAPITokenUsers{users: map[uuid.UUID]*model.User{
		ownerID: {ID: ownerID, TenantID: tenantID, Email: "rafal@example.com", Role: "owner"},
	}}
	svc := testAPITokenService(store, users)

	created, err := svc.Create(context.Background(), tenantID, ownerID, "owner", model.CreateAPITokenRequest{Name: "cli"}, ownerID, "127.0.0.1")
	require.NoError(t, err)

	claims, err := svc.AuthenticateAPIToken(context.Background(), created.Token)
	require.NoError(t, err)
	require.NotNil(t, claims)
	assert.Equal(t, ownerID.String(), claims.Subject)
	assert.Equal(t, tenantID, claims.TenantID)
	assert.Equal(t, "owner", claims.Role)
	assert.Equal(t, "access", claims.Type)
	assert.Contains(t, claims.Permissions, model.PermOrdersView)

	require.NoError(t, svc.Revoke(context.Background(), tenantID, ownerID, "owner", created.ID, ownerID, "127.0.0.1"))

	_, err = svc.AuthenticateAPIToken(context.Background(), created.Token)
	assert.ErrorIs(t, err, ErrAPITokenInvalid)
}

func TestAPITokenService_CreateRequiresOwner(t *testing.T) {
	store := newMemoryAPITokenStore()
	svc := testAPITokenService(store, stubAPITokenUsers{})
	_, err := svc.Create(context.Background(), uuid.New(), uuid.New(), "admin", model.CreateAPITokenRequest{Name: "nope"}, uuid.New(), "127.0.0.1")
	assert.ErrorIs(t, err, ErrAPITokenOwnerRequired)
}

func TestAPITokenService_UnknownTokenIsInvalid(t *testing.T) {
	svc := testAPITokenService(newMemoryAPITokenStore(), stubAPITokenUsers{})
	_, err := svc.AuthenticateAPIToken(context.Background(), "oms_not-a-real-token")
	assert.ErrorIs(t, err, ErrAPITokenInvalid)
}
