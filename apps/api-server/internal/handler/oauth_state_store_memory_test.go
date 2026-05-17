package handler

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryOAuthStateStore_LoadPurgesExpiredEntries(t *testing.T) {
	store := NewMemoryOAuthStateStore()
	defer store.Close()
	ctx := context.Background()
	now := time.Now()

	require.NoError(t, store.Save(ctx, "active", &OAuthState{
		ExpiresAt: now.Add(time.Minute),
		TenantID:  uuid.New(),
		UserID:    uuid.New(),
		Provider:  "allegro",
	}, time.Minute))
	require.NoError(t, store.Save(ctx, "expired", &OAuthState{
		ExpiresAt: now.Add(-time.Minute),
		TenantID:  uuid.New(),
		UserID:    uuid.New(),
		Provider:  "allegro",
	}, time.Minute))

	loaded, err := store.Load(ctx, "active")
	require.NoError(t, err)
	require.NotNil(t, loaded)

	store.mu.Lock()
	_, expiredPresent := store.entries["expired"]
	store.mu.Unlock()

	assert.False(t, expiredPresent, "Load should purge unrelated expired OAuth states")
}

func TestMemoryOAuthStateStore_BackgroundSweepPurgesExpiredEntries(t *testing.T) {
	store := newMemoryOAuthStateStore(10 * time.Millisecond)
	defer store.Close()
	ctx := context.Background()

	require.NoError(t, store.Save(ctx, "expired", &OAuthState{
		ExpiresAt: time.Now().Add(-time.Minute),
		TenantID:  uuid.New(),
		UserID:    uuid.New(),
		Provider:  "allegro",
	}, time.Minute))

	require.Eventually(t, func() bool {
		store.mu.Lock()
		defer store.mu.Unlock()
		_, expiredPresent := store.entries["expired"]
		return !expiredPresent
	}, time.Second, 10*time.Millisecond)
}

func TestMemoryOAuthStateStore_CloseIsIdempotent(t *testing.T) {
	store := NewMemoryOAuthStateStore()

	assert.NotPanics(t, func() {
		store.Close()
		store.Close()
	})
}
