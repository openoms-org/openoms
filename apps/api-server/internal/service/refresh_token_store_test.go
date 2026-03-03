package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryRefreshTokenStore_FamilyCRUD(t *testing.T) {
	store := NewMemoryRefreshTokenStore()
	defer close(store.done)
	ctx := context.Background()

	userID := uuid.New().String()
	tenantID := uuid.New().String()
	familyID := "fam-crud-" + uuid.New().String()
	createdAt := time.Now().Truncate(time.Millisecond)

	family := &RefreshTokenFamily{
		FamilyID:         familyID,
		UserID:           userID,
		TenantID:         tenantID,
		CurrentTokenHash: "initial-hash",
		CreatedAt:        createdAt,
	}

	// Store and retrieve
	err := store.StoreFamily(ctx, family, 10*time.Minute)
	require.NoError(t, err)

	got, err := store.GetFamily(ctx, familyID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, familyID, got.FamilyID)
	assert.Equal(t, userID, got.UserID)
	assert.Equal(t, tenantID, got.TenantID)
	assert.Equal(t, "initial-hash", got.CurrentTokenHash)
	assert.Equal(t, createdAt, got.CreatedAt)

	// Update CurrentTokenHash via UpdateFamily
	updated := &RefreshTokenFamily{
		FamilyID:         familyID,
		UserID:           userID,
		TenantID:         tenantID,
		CurrentTokenHash: "rotated-hash",
		CreatedAt:        createdAt,
	}
	err = store.UpdateFamily(ctx, updated, 10*time.Minute)
	require.NoError(t, err)

	got, err = store.GetFamily(ctx, familyID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "rotated-hash", got.CurrentTokenHash)

	// Delete and verify gone
	err = store.DeleteFamily(ctx, familyID)
	require.NoError(t, err)

	got, err = store.GetFamily(ctx, familyID)
	require.NoError(t, err)
	assert.Nil(t, got, "family should be nil after deletion")
}

func TestMemoryRefreshTokenStore_FamilyExpiryShortTTL(t *testing.T) {
	store := NewMemoryRefreshTokenStore()
	defer close(store.done)
	ctx := context.Background()

	family := &RefreshTokenFamily{
		FamilyID:         "fam-expiry",
		UserID:           uuid.New().String(),
		TenantID:         uuid.New().String(),
		CurrentTokenHash: "hash-exp",
		CreatedAt:        time.Now(),
	}

	err := store.StoreFamily(ctx, family, 1*time.Millisecond)
	require.NoError(t, err)

	// Wait for the TTL to elapse
	time.Sleep(5 * time.Millisecond)

	got, err := store.GetFamily(ctx, "fam-expiry")
	require.NoError(t, err)
	assert.Nil(t, got, "expired family should return nil")
}

func TestMemoryRefreshTokenStore_TokenCRUD(t *testing.T) {
	store := NewMemoryRefreshTokenStore()
	defer close(store.done)
	ctx := context.Background()

	tokenHash := "tok-crud-" + uuid.New().String()
	entry := &RefreshTokenEntry{
		FamilyID: "fam-tok-crud",
		Used:     false,
	}

	// Store and retrieve
	err := store.StoreToken(ctx, tokenHash, entry, 10*time.Minute)
	require.NoError(t, err)

	got, err := store.GetToken(ctx, tokenHash)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "fam-tok-crud", got.FamilyID)
	assert.False(t, got.Used, "newly stored token should not be marked as used")

	// Delete and verify gone
	err = store.DeleteToken(ctx, tokenHash)
	require.NoError(t, err)

	got, err = store.GetToken(ctx, tokenHash)
	require.NoError(t, err)
	assert.Nil(t, got, "token should be nil after deletion")
}

func TestMemoryRefreshTokenStore_MarkTokenUsedAndVerify(t *testing.T) {
	store := NewMemoryRefreshTokenStore()
	defer close(store.done)
	ctx := context.Background()

	tokenHash := "tok-mark-" + uuid.New().String()
	entry := &RefreshTokenEntry{
		FamilyID: "fam-mark",
		Used:     false,
	}

	err := store.StoreToken(ctx, tokenHash, entry, 10*time.Minute)
	require.NoError(t, err)

	// Verify initially not used
	got, err := store.GetToken(ctx, tokenHash)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.False(t, got.Used)

	// Mark as used
	err = store.MarkTokenUsed(ctx, tokenHash)
	require.NoError(t, err)

	// Verify now used
	got, err = store.GetToken(ctx, tokenHash)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.True(t, got.Used, "token should be marked as used after MarkTokenUsed")
	assert.Equal(t, "fam-mark", got.FamilyID, "FamilyID should remain unchanged after marking used")

	// MarkTokenUsed on non-existent token should not error
	err = store.MarkTokenUsed(ctx, "nonexistent-token")
	require.NoError(t, err)
}

func TestMemoryRefreshTokenStore_TokenExpiryShortTTL(t *testing.T) {
	store := NewMemoryRefreshTokenStore()
	defer close(store.done)
	ctx := context.Background()

	tokenHash := "tok-expiry-" + uuid.New().String()
	entry := &RefreshTokenEntry{
		FamilyID: "fam-tok-expiry",
		Used:     false,
	}

	err := store.StoreToken(ctx, tokenHash, entry, 1*time.Millisecond)
	require.NoError(t, err)

	// Wait for the TTL to elapse
	time.Sleep(5 * time.Millisecond)

	got, err := store.GetToken(ctx, tokenHash)
	require.NoError(t, err)
	assert.Nil(t, got, "expired token should return nil")
}

func TestMemoryRefreshTokenStore_IsolatedKeys(t *testing.T) {
	store := NewMemoryRefreshTokenStore()
	defer close(store.done)
	ctx := context.Background()

	// Store two distinct families
	familyA := &RefreshTokenFamily{
		FamilyID:         "fam-iso-a",
		UserID:           uuid.New().String(),
		TenantID:         uuid.New().String(),
		CurrentTokenHash: "hash-a",
		CreatedAt:        time.Now(),
	}
	familyB := &RefreshTokenFamily{
		FamilyID:         "fam-iso-b",
		UserID:           uuid.New().String(),
		TenantID:         uuid.New().String(),
		CurrentTokenHash: "hash-b",
		CreatedAt:        time.Now(),
	}
	err := store.StoreFamily(ctx, familyA, 10*time.Minute)
	require.NoError(t, err)
	err = store.StoreFamily(ctx, familyB, 10*time.Minute)
	require.NoError(t, err)

	// Store two distinct tokens
	entryA := &RefreshTokenEntry{FamilyID: "fam-iso-a", Used: false}
	entryB := &RefreshTokenEntry{FamilyID: "fam-iso-b", Used: false}
	err = store.StoreToken(ctx, "tok-iso-a", entryA, 10*time.Minute)
	require.NoError(t, err)
	err = store.StoreToken(ctx, "tok-iso-b", entryB, 10*time.Minute)
	require.NoError(t, err)

	// Deleting family A should not affect family B
	err = store.DeleteFamily(ctx, "fam-iso-a")
	require.NoError(t, err)

	gotA, err := store.GetFamily(ctx, "fam-iso-a")
	require.NoError(t, err)
	assert.Nil(t, gotA, "deleted family A should be nil")

	gotB, err := store.GetFamily(ctx, "fam-iso-b")
	require.NoError(t, err)
	require.NotNil(t, gotB)
	assert.Equal(t, "hash-b", gotB.CurrentTokenHash)

	// Marking token A as used should not affect token B
	err = store.MarkTokenUsed(ctx, "tok-iso-a")
	require.NoError(t, err)

	gotTokA, err := store.GetToken(ctx, "tok-iso-a")
	require.NoError(t, err)
	require.NotNil(t, gotTokA)
	assert.True(t, gotTokA.Used)

	gotTokB, err := store.GetToken(ctx, "tok-iso-b")
	require.NoError(t, err)
	require.NotNil(t, gotTokB)
	assert.False(t, gotTokB.Used, "token B should remain unused when token A is marked used")
}

func TestMemoryRefreshTokenStore_GetReturnsDeepCopy(t *testing.T) {
	store := NewMemoryRefreshTokenStore()
	defer close(store.done)
	ctx := context.Background()

	family := &RefreshTokenFamily{
		FamilyID:         "fam-copy",
		UserID:           uuid.New().String(),
		TenantID:         uuid.New().String(),
		CurrentTokenHash: "original-hash",
		CreatedAt:        time.Now(),
	}
	err := store.StoreFamily(ctx, family, 10*time.Minute)
	require.NoError(t, err)

	// Retrieve and mutate the returned value
	got1, err := store.GetFamily(ctx, "fam-copy")
	require.NoError(t, err)
	require.NotNil(t, got1)
	got1.CurrentTokenHash = "tampered-hash"
	got1.UserID = "tampered-user"

	// Retrieve again and verify the store's data is unaffected
	got2, err := store.GetFamily(ctx, "fam-copy")
	require.NoError(t, err)
	require.NotNil(t, got2)
	assert.Equal(t, "original-hash", got2.CurrentTokenHash, "store should return independent copies — mutation must not propagate")
	assert.Equal(t, family.UserID, got2.UserID, "UserID should be unchanged after mutating a previous copy")

	// Also verify that mutating the input to StoreFamily does not affect stored data
	family.CurrentTokenHash = "post-store-mutation"
	got3, err := store.GetFamily(ctx, "fam-copy")
	require.NoError(t, err)
	require.NotNil(t, got3)
	assert.Equal(t, "original-hash", got3.CurrentTokenHash, "mutating the original input after StoreFamily should not affect stored data")

	// Verify the same deep-copy behavior for tokens
	entry := &RefreshTokenEntry{FamilyID: "fam-copy", Used: false}
	err = store.StoreToken(ctx, "tok-copy", entry, 10*time.Minute)
	require.NoError(t, err)

	gotTok1, err := store.GetToken(ctx, "tok-copy")
	require.NoError(t, err)
	require.NotNil(t, gotTok1)
	gotTok1.FamilyID = "tampered-family"
	gotTok1.Used = true

	gotTok2, err := store.GetToken(ctx, "tok-copy")
	require.NoError(t, err)
	require.NotNil(t, gotTok2)
	assert.Equal(t, "fam-copy", gotTok2.FamilyID, "token FamilyID should be unchanged after mutating a previous copy")
	assert.False(t, gotTok2.Used, "token Used should be unchanged after mutating a previous copy")
}
