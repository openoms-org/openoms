package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func iptr(v int) *int { return &v }

// baseSnapshot: trusted, exact, 100 units, fresh (observed now), reservation supported.
func baseSnapshot(now time.Time) SupplierAvailability {
	return SupplierAvailability{
		SourceQuantity:       100,
		AvailabilityType:     AvailabilityExactQuantity,
		ReservationSupported: true,
		FreshnessObservedAt:  now,
		MaxHandlingDays:      iptr(3),
	}
}

func TestResolve_Trusted_BufferApplied(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	snap := baseSnapshot(now)
	pol := EffectivePolicy{Mode: PolicyModeAuto, SafetyBuffer: 10, FreshnessWindowSeconds: 3600}
	d := ResolveAvailability(snap, pol, 5, false, now)
	assert.Equal(t, AvailabilityStatusTrusted, d.Status)
	assert.Equal(t, 90, d.AvailableToSell) // 100 - 10
	assert.True(t, d.AutoRoutable)
	assert.Nil(t, d.BlockerCode)
}

func TestResolve_Override_Wins(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	snap := baseSnapshot(now)
	pol := EffectivePolicy{Mode: PolicyModeManual, SafetyBuffer: 10, OverrideQuantity: iptr(7), FreshnessWindowSeconds: 3600}
	d := ResolveAvailability(snap, pol, 5, false, now)
	assert.Equal(t, AvailabilityStatusManualOverride, d.Status)
	assert.Equal(t, 7, d.AvailableToSell)
}

func TestResolve_Paused_ZeroNoBlocker(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	pol := EffectivePolicy{Mode: PolicyModePaused, FreshnessWindowSeconds: 3600}
	d := ResolveAvailability(baseSnapshot(now), pol, 5, false, now)
	assert.Equal(t, AvailabilityStatusPaused, d.Status)
	assert.Equal(t, 0, d.AvailableToSell)
	assert.False(t, d.AutoRoutable)
	assert.Nil(t, d.BlockerCode) // intentional operator pause, not a blocker
}

func TestResolve_Stale_Blocks(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	snap := baseSnapshot(now.Add(-2 * time.Hour)) // observed 2h ago
	pol := EffectivePolicy{Mode: PolicyModeAuto, FreshnessWindowSeconds: 3600}
	d := ResolveAvailability(snap, pol, 5, false, now)
	assert.Equal(t, AvailabilityStatusStale, d.Status)
	assert.Equal(t, 0, d.AvailableToSell)
	assert.False(t, d.AutoRoutable)
	assert.Equal(t, BlockerSupplierAvailabilityStale, *d.BlockerCode)
}

func TestResolve_Unknown_Blocks(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	snap := baseSnapshot(now)
	snap.AvailabilityType = AvailabilityUnknown
	pol := EffectivePolicy{Mode: PolicyModeAuto, FreshnessWindowSeconds: 3600}
	d := ResolveAvailability(snap, pol, 5, false, now)
	assert.Equal(t, AvailabilityStatusUnknown, d.Status)
	assert.Equal(t, BlockerSupplierAvailabilityUnknown, *d.BlockerCode)
}

func TestResolve_Insufficient_NoETA_Blocks(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	snap := baseSnapshot(now)
	snap.SourceQuantity = 3
	pol := EffectivePolicy{Mode: PolicyModeAuto, FreshnessWindowSeconds: 3600}
	d := ResolveAvailability(snap, pol, 5, false, now) // need 5, have 3, no ETA
	assert.Equal(t, AvailabilityStatusTrusted, d.Status)
	assert.False(t, d.AutoRoutable)
	assert.False(t, d.Backorder)
	assert.Equal(t, BlockerSupplierAvailabilityInsufficient, *d.BlockerCode)
}

func TestResolve_Insufficient_WithETA_Backorder(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	snap := baseSnapshot(now)
	snap.SourceQuantity = 3
	eta := now.Add(72 * time.Hour)
	snap.NextDeliveryDate = &eta
	pol := EffectivePolicy{Mode: PolicyModeAuto, FreshnessWindowSeconds: 3600}
	d := ResolveAvailability(snap, pol, 5, false, now)
	assert.True(t, d.Backorder)
	assert.False(t, d.AutoRoutable)
	assert.Nil(t, d.BlockerCode) // backorder is a unit state, not a blocker
}

func TestResolve_RequireReservation_NotSupported_NotRoutable(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	snap := baseSnapshot(now)
	snap.ReservationSupported = false
	pol := EffectivePolicy{Mode: PolicyModeAuto, FreshnessWindowSeconds: 3600, RequireReservation: true}
	d := ResolveAvailability(snap, pol, 5, false, now)
	assert.False(t, d.AutoRoutable)
}

func TestResolve_RequirePreflight_DefaultsUnsupported_NotRoutable(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	pol := EffectivePolicy{Mode: PolicyModeAuto, FreshnessWindowSeconds: 3600, RequirePreflight: true}
	// preflightSupported = false (the engine that models it is a later spec).
	d := ResolveAvailability(baseSnapshot(now), pol, 5, false, now)
	assert.False(t, d.AutoRoutable)
}

func TestResolve_LeadTimeExceeded_NotRoutable(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	snap := baseSnapshot(now)
	snap.MaxHandlingDays = iptr(10)
	pol := EffectivePolicy{Mode: PolicyModeAuto, FreshnessWindowSeconds: 3600, MaxLeadTimeDays: iptr(5)}
	d := ResolveAvailability(snap, pol, 5, false, now)
	assert.False(t, d.AutoRoutable)
}

func TestResolve_ChannelIncrease_Gate(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	snap := baseSnapshot(now)
	// allow_channel_increase=false -> never allowed even when trusted.
	off := EffectivePolicy{Mode: PolicyModeAuto, FreshnessWindowSeconds: 3600, AllowChannelIncrease: false}
	assert.False(t, ResolveAvailability(snap, off, 5, false, now).ChannelIncreaseAllowed)
	// allow_channel_increase=true + trusted -> allowed.
	on := EffectivePolicy{Mode: PolicyModeAuto, FreshnessWindowSeconds: 3600, AllowChannelIncrease: true}
	assert.True(t, ResolveAvailability(snap, on, 5, false, now).ChannelIncreaseAllowed)
	// allow_channel_increase=true but stale -> not allowed.
	staleSnap := baseSnapshot(now.Add(-2 * time.Hour))
	assert.False(t, ResolveAvailability(staleSnap, on, 5, false, now).ChannelIncreaseAllowed)
}

func TestResolvePolicyChain_Precedence(t *testing.T) {
	// channel > listing > product > supplier; each field resolves independently.
	supplier := SupplierAvailabilityPolicy{Scope: PolicyScopeSupplier, Mode: PolicyModeAuto, SafetyBuffer: 5, FreshnessWindowSecs: iptr(7200)}
	product := SupplierAvailabilityPolicy{Scope: PolicyScopeProduct, SafetyBuffer: 9} // overrides buffer only
	eff := ResolvePolicyChain([]SupplierAvailabilityPolicy{supplier, product})
	assert.Equal(t, 9, eff.SafetyBuffer)              // from product (more specific)
	assert.Equal(t, 7200, eff.FreshnessWindowSeconds) // inherited from supplier
	assert.Equal(t, PolicyModeAuto, eff.Mode)
}
