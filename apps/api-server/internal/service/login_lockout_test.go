package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ===========================================================================
// MemoryLoginLockoutStore — direct store tests
// ===========================================================================

func TestMemoryLoginLockoutStore_IncrFailures(t *testing.T) {
	store := NewMemoryLoginLockoutStore()
	ctx := context.Background()

	t.Run("increment from 0 returns 1", func(t *testing.T) {
		count, err := store.IncrFailures(ctx, "key1", 5*time.Minute)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})

	t.Run("increment twice returns 2", func(t *testing.T) {
		count, err := store.IncrFailures(ctx, "key1", 5*time.Minute)
		require.NoError(t, err)
		assert.Equal(t, 2, count)
	})

	t.Run("different keys are independent", func(t *testing.T) {
		count, err := store.IncrFailures(ctx, "keyA", 5*time.Minute)
		require.NoError(t, err)
		assert.Equal(t, 1, count)

		count, err = store.IncrFailures(ctx, "keyB", 5*time.Minute)
		require.NoError(t, err)
		assert.Equal(t, 1, count)

		// keyA should still be at 1 (now 2 after another increment)
		count, err = store.IncrFailures(ctx, "keyA", 5*time.Minute)
		require.NoError(t, err)
		assert.Equal(t, 2, count)
	})
}

func TestMemoryLoginLockoutStore_SetLockout_GetRemaining(t *testing.T) {
	store := NewMemoryLoginLockoutStore()
	ctx := context.Background()

	t.Run("set lockout then remaining is positive", func(t *testing.T) {
		err := store.SetLockout(ctx, "lock1", 1*time.Second)
		require.NoError(t, err)

		remaining, err := store.GetLockoutRemaining(ctx, "lock1")
		require.NoError(t, err)
		assert.True(t, remaining > 0, "expected positive remaining duration, got %v", remaining)
		assert.True(t, remaining <= 1*time.Second, "remaining should not exceed set duration")
	})

	t.Run("after lockout expires remaining is 0", func(t *testing.T) {
		err := store.SetLockout(ctx, "lock2", 1*time.Millisecond)
		require.NoError(t, err)

		time.Sleep(5 * time.Millisecond)

		remaining, err := store.GetLockoutRemaining(ctx, "lock2")
		require.NoError(t, err)
		assert.Equal(t, time.Duration(0), remaining)
	})

	t.Run("no lockout set returns 0", func(t *testing.T) {
		remaining, err := store.GetLockoutRemaining(ctx, "never-locked")
		require.NoError(t, err)
		assert.Equal(t, time.Duration(0), remaining)
	})
}

func TestMemoryLoginLockoutStore_FailureWindowExpiry(t *testing.T) {
	store := NewMemoryLoginLockoutStore()
	ctx := context.Background()

	// Record a failure with a very short window
	count, err := store.IncrFailures(ctx, "expiring-key", 1*time.Millisecond)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	// Wait for the window to expire
	time.Sleep(5 * time.Millisecond)

	// After expiry the counter should reset, so next increment returns 1
	count, err = store.IncrFailures(ctx, "expiring-key", 5*time.Minute)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "failure count should reset after window expiry")
}

func TestMemoryLoginLockoutStore_ResetFailures(t *testing.T) {
	store := NewMemoryLoginLockoutStore()
	ctx := context.Background()

	// Accumulate failures
	for range 3 {
		_, err := store.IncrFailures(ctx, "reset-key", 5*time.Minute)
		require.NoError(t, err)
	}

	// Reset
	err := store.ResetFailures(ctx, "reset-key")
	require.NoError(t, err)

	// After reset, count restarts from 1
	count, err := store.IncrFailures(ctx, "reset-key", 5*time.Minute)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

// ===========================================================================
// LoginLockout orchestrator tests
// ===========================================================================

func TestLoginLockout_CheckLocked_NoLockout(t *testing.T) {
	store := NewMemoryLoginLockoutStore()
	lockout := NewLoginLockout(store)
	ctx := context.Background()

	remaining, err := lockout.CheckLocked(ctx, "tenant1", "user@example.com")
	require.NoError(t, err)
	assert.Equal(t, time.Duration(0), remaining, "new user should have no lockout")
}

func TestLoginLockout_RecordFailure_TriggersLockout(t *testing.T) {
	store := NewMemoryLoginLockoutStore()
	lockout := NewLoginLockout(store)
	ctx := context.Background()

	// Record 4 failures — should NOT trigger lockout (threshold is 5)
	for range 4 {
		err := lockout.RecordFailure(ctx, "tenant1", "user@example.com")
		require.NoError(t, err)
	}

	remaining, err := lockout.CheckLocked(ctx, "tenant1", "user@example.com")
	require.NoError(t, err)
	assert.Equal(t, time.Duration(0), remaining, "4 failures should not trigger lockout")

	// 5th failure triggers lockout
	err = lockout.RecordFailure(ctx, "tenant1", "user@example.com")
	require.NoError(t, err)

	remaining, err = lockout.CheckLocked(ctx, "tenant1", "user@example.com")
	require.NoError(t, err)
	assert.True(t, remaining > 0, "5th failure should trigger lockout, got remaining=%v", remaining)
	assert.True(t, remaining <= 1*time.Minute, "first lockout should be at most 1m, got %v", remaining)
}

func TestLoginLockout_RecordFailure_EscalatingDurations(t *testing.T) {
	store := NewMemoryLoginLockoutStore()
	lockout := NewLoginLockout(store)
	ctx := context.Background()

	// The lockoutDuration function uses cumulative failure count:
	//   failures <= 5  -> 30s
	//   failures <= 7  -> 1m
	//   failures <= 10 -> 5m
	//   failures <= 15 -> 15m
	//   failures > 15  -> 30m

	// Record 5 failures -> lockout at 30s
	for range 5 {
		err := lockout.RecordFailure(ctx, "tenant1", "escalate@example.com")
		require.NoError(t, err)
	}
	remaining1, err := lockout.CheckLocked(ctx, "tenant1", "escalate@example.com")
	require.NoError(t, err)
	assert.True(t, remaining1 > 0, "should be locked after 5 failures")
	assert.True(t, remaining1 <= 1*time.Minute, "5 failures -> 1m lockout, got %v", remaining1)

	// Record failures 6 and 7 -> lockout escalates to 5m
	for range 2 {
		err := lockout.RecordFailure(ctx, "tenant1", "escalate@example.com")
		require.NoError(t, err)
	}
	remaining2, err := lockout.CheckLocked(ctx, "tenant1", "escalate@example.com")
	require.NoError(t, err)
	assert.True(t, remaining2 > 1*time.Minute, "7 failures -> 5m lockout, got %v", remaining2)
	assert.True(t, remaining2 <= 5*time.Minute, "7 failures -> 5m lockout, got %v", remaining2)

	// Record failures 8, 9, 10 -> lockout escalates to 15m
	for range 3 {
		err := lockout.RecordFailure(ctx, "tenant1", "escalate@example.com")
		require.NoError(t, err)
	}
	remaining3, err := lockout.CheckLocked(ctx, "tenant1", "escalate@example.com")
	require.NoError(t, err)
	assert.True(t, remaining3 > 5*time.Minute, "10 failures -> 15m lockout, got %v", remaining3)
	assert.True(t, remaining3 <= 15*time.Minute, "10 failures -> 15m lockout, got %v", remaining3)

	// Record failure 11 -> lockout escalates to 30m
	err = lockout.RecordFailure(ctx, "tenant1", "escalate@example.com")
	require.NoError(t, err)
	remaining4, err := lockout.CheckLocked(ctx, "tenant1", "escalate@example.com")
	require.NoError(t, err)
	assert.True(t, remaining4 > 15*time.Minute, "11 failures -> 30m lockout, got %v", remaining4)
	assert.True(t, remaining4 <= 30*time.Minute, "11 failures -> 30m lockout, got %v", remaining4)

	// Record failure 16 -> lockout escalates to 30m
	err = lockout.RecordFailure(ctx, "tenant1", "escalate@example.com")
	require.NoError(t, err)
	remaining5, err := lockout.CheckLocked(ctx, "tenant1", "escalate@example.com")
	require.NoError(t, err)
	assert.True(t, remaining5 > 15*time.Minute, "16 failures -> 30m lockout, got %v", remaining5)
	assert.True(t, remaining5 <= 30*time.Minute, "16 failures -> 30m lockout, got %v", remaining5)
}

func TestLoginLockout_NilStore_FailsOpen(t *testing.T) {
	lockout := NewLoginLockout(nil)
	ctx := context.Background()

	t.Run("CheckLocked returns 0 with nil store", func(t *testing.T) {
		remaining, err := lockout.CheckLocked(ctx, "tenant1", "user@example.com")
		require.NoError(t, err)
		assert.Equal(t, time.Duration(0), remaining)
	})

	t.Run("RecordFailure returns nil with nil store", func(t *testing.T) {
		err := lockout.RecordFailure(ctx, "tenant1", "user@example.com")
		assert.NoError(t, err)
	})

	t.Run("ResetOnSuccess does not panic with nil store", func(t *testing.T) {
		assert.NotPanics(t, func() {
			lockout.ResetOnSuccess(ctx, "tenant1", "user@example.com")
		})
	})
}

func TestLoginLockout_NilReceiver_FailsOpen(t *testing.T) {
	var lockout *LoginLockout
	ctx := context.Background()

	remaining, err := lockout.CheckLocked(ctx, "tenant1", "user@example.com")
	require.NoError(t, err)
	assert.Equal(t, time.Duration(0), remaining)

	err = lockout.RecordFailure(ctx, "tenant1", "user@example.com")
	assert.NoError(t, err)

	assert.NotPanics(t, func() {
		lockout.ResetOnSuccess(ctx, "tenant1", "user@example.com")
	})
}

func TestLoginLockout_RecordSuccess_ClearsState(t *testing.T) {
	store := NewMemoryLoginLockoutStore()
	lockout := NewLoginLockout(store)
	ctx := context.Background()

	// Record 3 failures (below threshold, no lockout yet)
	for range 3 {
		err := lockout.RecordFailure(ctx, "tenant1", "user@example.com")
		require.NoError(t, err)
	}

	// Successful login clears state
	lockout.ResetOnSuccess(ctx, "tenant1", "user@example.com")

	// After reset, should not be locked
	remaining, err := lockout.CheckLocked(ctx, "tenant1", "user@example.com")
	require.NoError(t, err)
	assert.Equal(t, time.Duration(0), remaining)

	// Failure counter should have been reset — need 5 more failures to trigger lockout
	for range 4 {
		err := lockout.RecordFailure(ctx, "tenant1", "user@example.com")
		require.NoError(t, err)
	}

	// 4 new failures (not 7 total) — should NOT be locked
	remaining, err = lockout.CheckLocked(ctx, "tenant1", "user@example.com")
	require.NoError(t, err)
	assert.Equal(t, time.Duration(0), remaining, "after success reset, 4 new failures should not trigger lockout")
}

func TestLoginLockout_RecordSuccess_ClearsLockout(t *testing.T) {
	store := NewMemoryLoginLockoutStore()
	lockout := NewLoginLockout(store)
	ctx := context.Background()

	// Trigger a lockout with 5 failures
	for range 5 {
		err := lockout.RecordFailure(ctx, "tenant1", "locked@example.com")
		require.NoError(t, err)
	}

	// Verify locked
	remaining, err := lockout.CheckLocked(ctx, "tenant1", "locked@example.com")
	require.NoError(t, err)
	assert.True(t, remaining > 0, "should be locked")

	// Successful login clears both failure counter and lockout
	lockout.ResetOnSuccess(ctx, "tenant1", "locked@example.com")

	remaining, err = lockout.CheckLocked(ctx, "tenant1", "locked@example.com")
	require.NoError(t, err)
	assert.Equal(t, time.Duration(0), remaining, "lockout should be cleared after successful login")
}

func TestLoginLockout_TenantIsolation(t *testing.T) {
	store := NewMemoryLoginLockoutStore()
	lockout := NewLoginLockout(store)
	ctx := context.Background()

	// Record 5 failures for tenant1 — should lock tenant1
	for range 5 {
		err := lockout.RecordFailure(ctx, "tenant1", "user@example.com")
		require.NoError(t, err)
	}

	remaining, err := lockout.CheckLocked(ctx, "tenant1", "user@example.com")
	require.NoError(t, err)
	assert.True(t, remaining > 0, "tenant1 should be locked")

	// Same email on tenant2 should NOT be locked
	remaining, err = lockout.CheckLocked(ctx, "tenant2", "user@example.com")
	require.NoError(t, err)
	assert.Equal(t, time.Duration(0), remaining, "tenant2 should not be affected by tenant1 lockout")
}

// ===========================================================================
// 2FA per-account lockout — keyed on tenantID:userID, separate namespace
// ===========================================================================

func TestLoginLockout_2FA_RecordFailure_TriggersLockout(t *testing.T) {
	store := NewMemoryLoginLockoutStore()
	lockout := NewLoginLockout(store)
	ctx := context.Background()
	tenantID := "11111111-1111-1111-1111-111111111111"
	userID := "22222222-2222-2222-2222-222222222222"

	// 4 wrong codes — below threshold, not locked yet.
	for range 4 {
		require.NoError(t, lockout.Record2FAFailure(ctx, tenantID, userID))
	}
	remaining, err := lockout.Check2FALocked(ctx, tenantID, userID)
	require.NoError(t, err)
	assert.Equal(t, time.Duration(0), remaining, "4 wrong codes should not lock the account")

	// 5th wrong code locks the account regardless of source IP.
	require.NoError(t, lockout.Record2FAFailure(ctx, tenantID, userID))
	remaining, err = lockout.Check2FALocked(ctx, tenantID, userID)
	require.NoError(t, err)
	assert.True(t, remaining > 0, "5th wrong code must lock the account, got remaining=%v", remaining)
}

func TestLoginLockout_2FA_Reset_ClearsLockout(t *testing.T) {
	store := NewMemoryLoginLockoutStore()
	lockout := NewLoginLockout(store)
	ctx := context.Background()
	tenantID := "11111111-1111-1111-1111-111111111111"
	userID := "22222222-2222-2222-2222-222222222222"

	for range 5 {
		require.NoError(t, lockout.Record2FAFailure(ctx, tenantID, userID))
	}
	remaining, err := lockout.Check2FALocked(ctx, tenantID, userID)
	require.NoError(t, err)
	require.True(t, remaining > 0, "should be locked after 5 wrong codes")

	// A successful TOTP verification clears the counter and the lockout.
	lockout.Reset2FAOnSuccess(ctx, tenantID, userID)
	remaining, err = lockout.Check2FALocked(ctx, tenantID, userID)
	require.NoError(t, err)
	assert.Equal(t, time.Duration(0), remaining, "successful 2FA must clear the lockout")
}

// TestLoginLockout_2FA_SeparateNamespaceFromPassword proves the 2FA counter does not
// share state with the password-login counter. This is the security property that stops
// an attacker who knows the password from resetting the 2FA brute-force counter by
// re-authenticating (password success calls ResetOnSuccess on the password namespace only).
func TestLoginLockout_2FA_SeparateNamespaceFromPassword(t *testing.T) {
	store := NewMemoryLoginLockoutStore()
	lockout := NewLoginLockout(store)
	ctx := context.Background()
	tenantSlug := "acme"
	email := "user@example.com"
	tenantID := "11111111-1111-1111-1111-111111111111"
	userID := "22222222-2222-2222-2222-222222222222"

	// Lock the 2FA namespace.
	for range 5 {
		require.NoError(t, lockout.Record2FAFailure(ctx, tenantID, userID))
	}
	twofa, err := lockout.Check2FALocked(ctx, tenantID, userID)
	require.NoError(t, err)
	require.True(t, twofa > 0, "2FA should be locked")

	// A successful password login resets only the password namespace; the 2FA lockout stands.
	lockout.ResetOnSuccess(ctx, tenantSlug, email)
	twofa, err = lockout.Check2FALocked(ctx, tenantID, userID)
	require.NoError(t, err)
	assert.True(t, twofa > 0, "password-login success must NOT clear the 2FA lockout")

	// Conversely, password-login failures must not lock the 2FA flow.
	store2 := NewMemoryLoginLockoutStore()
	lockout2 := NewLoginLockout(store2)
	for range 5 {
		require.NoError(t, lockout2.RecordFailure(ctx, tenantSlug, email))
	}
	pwLock, err := lockout2.CheckLocked(ctx, tenantSlug, email)
	require.NoError(t, err)
	require.True(t, pwLock > 0, "password login should be locked")
	twofa2, err := lockout2.Check2FALocked(ctx, tenantID, userID)
	require.NoError(t, err)
	assert.Equal(t, time.Duration(0), twofa2, "password-login failures must NOT lock the 2FA flow")
}

func TestLoginLockout_2FA_UserIsolation(t *testing.T) {
	store := NewMemoryLoginLockoutStore()
	lockout := NewLoginLockout(store)
	ctx := context.Background()
	tenantID := "11111111-1111-1111-1111-111111111111"
	userA := "22222222-2222-2222-2222-222222222222"
	userB := "33333333-3333-3333-3333-333333333333"

	for range 5 {
		require.NoError(t, lockout.Record2FAFailure(ctx, tenantID, userA))
	}
	remaining, err := lockout.Check2FALocked(ctx, tenantID, userB)
	require.NoError(t, err)
	assert.Equal(t, time.Duration(0), remaining, "userA lockout must not affect userB")
}

func TestLoginLockout_2FA_NilStoreAndReceiver_FailOpen(t *testing.T) {
	ctx := context.Background()

	nilStore := NewLoginLockout(nil)
	remaining, err := nilStore.Check2FALocked(ctx, "t", "u")
	require.NoError(t, err)
	assert.Equal(t, time.Duration(0), remaining)
	assert.NoError(t, nilStore.Record2FAFailure(ctx, "t", "u"))
	assert.NotPanics(t, func() { nilStore.Reset2FAOnSuccess(ctx, "t", "u") })

	var nilReceiver *LoginLockout
	remaining, err = nilReceiver.Check2FALocked(ctx, "t", "u")
	require.NoError(t, err)
	assert.Equal(t, time.Duration(0), remaining)
	assert.NoError(t, nilReceiver.Record2FAFailure(ctx, "t", "u"))
	assert.NotPanics(t, func() { nilReceiver.Reset2FAOnSuccess(ctx, "t", "u") })
}

func TestLoginLockout_LockoutDuration(t *testing.T) {
	// Unit test for the lockoutDuration function directly
	tests := []struct {
		failures int
		expected time.Duration
	}{
		{1, 1 * time.Minute},
		{5, 1 * time.Minute},
		{6, 5 * time.Minute},
		{7, 5 * time.Minute},
		{8, 15 * time.Minute},
		{10, 15 * time.Minute},
		{11, 30 * time.Minute},
		{100, 30 * time.Minute},
	}

	for _, tc := range tests {
		got := lockoutDuration(tc.failures)
		assert.Equal(t, tc.expected, got, "lockoutDuration(%d) = %v, want %v", tc.failures, got, tc.expected)
	}
}
