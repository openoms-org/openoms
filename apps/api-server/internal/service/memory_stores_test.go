package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ===========================================================================
// MemoryLoginLockoutStore tests
// ===========================================================================

func TestMemoryLoginLockout_IncrFailures(t *testing.T) {
	store := NewMemoryLoginLockoutStore()
	ctx := context.Background()

	count, err := store.IncrFailures(ctx, "user@example.com", 5*time.Minute)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	count, err = store.IncrFailures(ctx, "user@example.com", 5*time.Minute)
	require.NoError(t, err)
	assert.Equal(t, 2, count)

	count, err = store.IncrFailures(ctx, "user@example.com", 5*time.Minute)
	require.NoError(t, err)
	assert.Equal(t, 3, count)
}

func TestMemoryLoginLockout_SetLockoutAndCheck(t *testing.T) {
	store := NewMemoryLoginLockoutStore()
	ctx := context.Background()

	// No lockout initially
	remaining, err := store.GetLockoutRemaining(ctx, "user@example.com")
	require.NoError(t, err)
	assert.Equal(t, time.Duration(0), remaining)

	// Set lockout
	err = store.SetLockout(ctx, "user@example.com", 10*time.Minute)
	require.NoError(t, err)

	remaining, err = store.GetLockoutRemaining(ctx, "user@example.com")
	require.NoError(t, err)
	assert.True(t, remaining > 0, "expected positive remaining lockout duration")
	assert.True(t, remaining <= 10*time.Minute, "remaining should not exceed the set duration")
}

func TestMemoryLoginLockout_ResetFailures(t *testing.T) {
	store := NewMemoryLoginLockoutStore()
	ctx := context.Background()

	// Record some failures
	_, err := store.IncrFailures(ctx, "user@example.com", 5*time.Minute)
	require.NoError(t, err)
	_, err = store.IncrFailures(ctx, "user@example.com", 5*time.Minute)
	require.NoError(t, err)

	// Reset
	err = store.ResetFailures(ctx, "user@example.com")
	require.NoError(t, err)

	// After reset, count should start from 1 again
	count, err := store.IncrFailures(ctx, "user@example.com", 5*time.Minute)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestMemoryLoginLockout_ExpiredEntry(t *testing.T) {
	store := NewMemoryLoginLockoutStore()
	ctx := context.Background()

	// Record a failure with a very short window
	count, err := store.IncrFailures(ctx, "user@example.com", 1*time.Millisecond)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	// Wait for the window to expire
	time.Sleep(5 * time.Millisecond)

	// After expiry, count should reset to 1
	count, err = store.IncrFailures(ctx, "user@example.com", 5*time.Minute)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestMemoryLoginLockout_ExpiredLockout(t *testing.T) {
	store := NewMemoryLoginLockoutStore()
	ctx := context.Background()

	// Set a very short lockout
	err := store.SetLockout(ctx, "user@example.com", 1*time.Millisecond)
	require.NoError(t, err)

	// Wait for it to expire
	time.Sleep(5 * time.Millisecond)

	remaining, err := store.GetLockoutRemaining(ctx, "user@example.com")
	require.NoError(t, err)
	assert.Equal(t, time.Duration(0), remaining)
}

func TestMemoryLoginLockout_IndependentKeys(t *testing.T) {
	store := NewMemoryLoginLockoutStore()
	ctx := context.Background()

	count, err := store.IncrFailures(ctx, "user1@example.com", 5*time.Minute)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	count, err = store.IncrFailures(ctx, "user1@example.com", 5*time.Minute)
	require.NoError(t, err)
	assert.Equal(t, 2, count)

	// Different key starts at 1
	count, err = store.IncrFailures(ctx, "user2@example.com", 5*time.Minute)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

// ===========================================================================
// MemoryRefreshTokenStore tests
// ===========================================================================

func TestMemoryRefreshTokenStore_StoreAndGetFamily(t *testing.T) {
	store := NewMemoryRefreshTokenStore()
	defer store.Close()
	ctx := context.Background()

	family := &RefreshTokenFamily{
		FamilyID:         "fam-1",
		UserID:           "user-1",
		TenantID:         "tenant-1",
		CurrentTokenHash: "hash-1",
		CreatedAt:        time.Now(),
	}

	err := store.StoreFamily(ctx, family, 10*time.Minute)
	require.NoError(t, err)

	got, err := store.GetFamily(ctx, "fam-1")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "fam-1", got.FamilyID)
	assert.Equal(t, "user-1", got.UserID)
	assert.Equal(t, "tenant-1", got.TenantID)
	assert.Equal(t, "hash-1", got.CurrentTokenHash)
}

func TestMemoryRefreshTokenStore_GetFamily_NotFound(t *testing.T) {
	store := NewMemoryRefreshTokenStore()
	defer store.Close()
	ctx := context.Background()

	got, err := store.GetFamily(ctx, "nonexistent")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestMemoryRefreshTokenStore_DeleteFamily(t *testing.T) {
	store := NewMemoryRefreshTokenStore()
	defer store.Close()
	ctx := context.Background()

	family := &RefreshTokenFamily{
		FamilyID:         "fam-1",
		UserID:           "user-1",
		TenantID:         "tenant-1",
		CurrentTokenHash: "hash-1",
		CreatedAt:        time.Now(),
	}
	err := store.StoreFamily(ctx, family, 10*time.Minute)
	require.NoError(t, err)

	err = store.DeleteFamily(ctx, "fam-1")
	require.NoError(t, err)

	got, err := store.GetFamily(ctx, "fam-1")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestMemoryRefreshTokenStore_FamilyExpiry(t *testing.T) {
	store := NewMemoryRefreshTokenStore()
	defer store.Close()
	ctx := context.Background()

	family := &RefreshTokenFamily{
		FamilyID:         "fam-1",
		UserID:           "user-1",
		TenantID:         "tenant-1",
		CurrentTokenHash: "hash-1",
		CreatedAt:        time.Now(),
	}
	err := store.StoreFamily(ctx, family, 1*time.Millisecond)
	require.NoError(t, err)

	// Wait for expiry
	time.Sleep(5 * time.Millisecond)

	got, err := store.GetFamily(ctx, "fam-1")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestMemoryRefreshTokenStore_StoreAndGetToken(t *testing.T) {
	store := NewMemoryRefreshTokenStore()
	defer store.Close()
	ctx := context.Background()

	entry := &RefreshTokenEntry{
		FamilyID: "fam-1",
		Used:     false,
	}
	err := store.StoreToken(ctx, "token-hash-1", entry, 10*time.Minute)
	require.NoError(t, err)

	got, err := store.GetToken(ctx, "token-hash-1")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "fam-1", got.FamilyID)
	assert.False(t, got.Used)
}

func TestMemoryRefreshTokenStore_DeleteToken(t *testing.T) {
	store := NewMemoryRefreshTokenStore()
	defer store.Close()
	ctx := context.Background()

	entry := &RefreshTokenEntry{FamilyID: "fam-1", Used: false}
	err := store.StoreToken(ctx, "token-hash-1", entry, 10*time.Minute)
	require.NoError(t, err)

	err = store.DeleteToken(ctx, "token-hash-1")
	require.NoError(t, err)

	got, err := store.GetToken(ctx, "token-hash-1")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestMemoryRefreshTokenStore_TokenExpiry(t *testing.T) {
	store := NewMemoryRefreshTokenStore()
	defer store.Close()
	ctx := context.Background()

	entry := &RefreshTokenEntry{FamilyID: "fam-1", Used: false}
	err := store.StoreToken(ctx, "token-hash-1", entry, 1*time.Millisecond)
	require.NoError(t, err)

	time.Sleep(5 * time.Millisecond)

	got, err := store.GetToken(ctx, "token-hash-1")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestMemoryRefreshTokenStore_MarkTokenUsed(t *testing.T) {
	store := NewMemoryRefreshTokenStore()
	defer store.Close()
	ctx := context.Background()

	entry := &RefreshTokenEntry{FamilyID: "fam-1", Used: false}
	err := store.StoreToken(ctx, "token-hash-1", entry, 10*time.Minute)
	require.NoError(t, err)

	err = store.MarkTokenUsed(ctx, "token-hash-1")
	require.NoError(t, err)

	got, err := store.GetToken(ctx, "token-hash-1")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.True(t, got.Used)
}

func TestMemoryRefreshTokenStore_MarkTokenUsed_NonExistent(t *testing.T) {
	store := NewMemoryRefreshTokenStore()
	defer store.Close()
	ctx := context.Background()

	// Should not error even if the token doesn't exist
	err := store.MarkTokenUsed(ctx, "nonexistent")
	require.NoError(t, err)
}

func TestMemoryRefreshTokenStore_UpdateFamily(t *testing.T) {
	store := NewMemoryRefreshTokenStore()
	defer store.Close()
	ctx := context.Background()

	family := &RefreshTokenFamily{
		FamilyID:         "fam-1",
		UserID:           "user-1",
		TenantID:         "tenant-1",
		CurrentTokenHash: "hash-1",
		CreatedAt:        time.Now(),
	}
	err := store.StoreFamily(ctx, family, 10*time.Minute)
	require.NoError(t, err)

	// Update the family with a new token hash
	updated := &RefreshTokenFamily{
		FamilyID:         "fam-1",
		UserID:           "user-1",
		TenantID:         "tenant-1",
		CurrentTokenHash: "hash-2",
		CreatedAt:        family.CreatedAt,
	}
	err = store.UpdateFamily(ctx, updated, 10*time.Minute)
	require.NoError(t, err)

	got, err := store.GetFamily(ctx, "fam-1")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "hash-2", got.CurrentTokenHash)
}

func TestMemoryRefreshTokenStore_ReturnsCopy(t *testing.T) {
	store := NewMemoryRefreshTokenStore()
	defer store.Close()
	ctx := context.Background()

	family := &RefreshTokenFamily{
		FamilyID:         "fam-1",
		UserID:           "user-1",
		TenantID:         "tenant-1",
		CurrentTokenHash: "hash-1",
		CreatedAt:        time.Now(),
	}
	err := store.StoreFamily(ctx, family, 10*time.Minute)
	require.NoError(t, err)

	// Modify the returned copy — should not affect the store
	got, err := store.GetFamily(ctx, "fam-1")
	require.NoError(t, err)
	got.CurrentTokenHash = "modified"

	got2, err := store.GetFamily(ctx, "fam-1")
	require.NoError(t, err)
	assert.Equal(t, "hash-1", got2.CurrentTokenHash, "store should return independent copies")
}

// ===========================================================================
// MemoryWSTicketStore tests
// ===========================================================================

func TestMemoryWSTicketStore_StoreAndConsume(t *testing.T) {
	store := NewMemoryWSTicketStore()
	ctx := context.Background()

	data := WSTicketData{Token: "jwt-access-token"}
	err := store.Store(ctx, "ticket-123", data, 5*time.Minute)
	require.NoError(t, err)

	got, err := store.Consume(ctx, "ticket-123")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "jwt-access-token", got.Token)
}

func TestMemoryWSTicketStore_ConsumeOnce(t *testing.T) {
	store := NewMemoryWSTicketStore()
	ctx := context.Background()

	data := WSTicketData{Token: "jwt-access-token"}
	err := store.Store(ctx, "ticket-123", data, 5*time.Minute)
	require.NoError(t, err)

	// First consume succeeds
	got, err := store.Consume(ctx, "ticket-123")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "jwt-access-token", got.Token)

	// Second consume returns nil (ticket already consumed)
	got, err = store.Consume(ctx, "ticket-123")
	require.NoError(t, err)
	assert.Nil(t, got, "ticket should be consumed on first use and gone after that")
}

func TestMemoryWSTicketStore_ConsumeNonExistent(t *testing.T) {
	store := NewMemoryWSTicketStore()
	ctx := context.Background()

	got, err := store.Consume(ctx, "nonexistent")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestMemoryWSTicketStore_TicketExpiry(t *testing.T) {
	store := NewMemoryWSTicketStore()
	ctx := context.Background()

	data := WSTicketData{Token: "jwt-access-token"}
	err := store.Store(ctx, "ticket-123", data, 1*time.Millisecond)
	require.NoError(t, err)

	// Wait for the ticket to expire
	time.Sleep(5 * time.Millisecond)

	got, err := store.Consume(ctx, "ticket-123")
	require.NoError(t, err)
	assert.Nil(t, got, "expired ticket should not be consumable")
}

func TestMemoryWSTicketStore_IndependentTickets(t *testing.T) {
	store := NewMemoryWSTicketStore()
	ctx := context.Background()

	err := store.Store(ctx, "ticket-a", WSTicketData{Token: "token-a"}, 5*time.Minute)
	require.NoError(t, err)
	err = store.Store(ctx, "ticket-b", WSTicketData{Token: "token-b"}, 5*time.Minute)
	require.NoError(t, err)

	// Consuming one should not affect the other
	gotA, err := store.Consume(ctx, "ticket-a")
	require.NoError(t, err)
	require.NotNil(t, gotA)
	assert.Equal(t, "token-a", gotA.Token)

	gotB, err := store.Consume(ctx, "ticket-b")
	require.NoError(t, err)
	require.NotNil(t, gotB)
	assert.Equal(t, "token-b", gotB.Token)
}

func TestMemoryWSTicketStore_OverwriteTicket(t *testing.T) {
	store := NewMemoryWSTicketStore()
	ctx := context.Background()

	err := store.Store(ctx, "ticket-123", WSTicketData{Token: "token-v1"}, 5*time.Minute)
	require.NoError(t, err)

	// Overwrite with new data
	err = store.Store(ctx, "ticket-123", WSTicketData{Token: "token-v2"}, 5*time.Minute)
	require.NoError(t, err)

	got, err := store.Consume(ctx, "ticket-123")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "token-v2", got.Token, "should return the most recently stored value")
}
