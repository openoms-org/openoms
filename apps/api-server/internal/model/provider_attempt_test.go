package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProviderAttemptValidators(t *testing.T) {
	assert.True(t, IsValidProviderAttemptOperation(ProviderOpCreateShipment))
	assert.True(t, IsValidProviderAttemptOperation(ProviderOpGenerateLabel))
	assert.True(t, IsValidProviderAttemptOperation(ProviderOpDownloadLabel))
	assert.True(t, IsValidProviderAttemptOperation(ProviderOpSyncTracking))
	assert.False(t, IsValidProviderAttemptOperation("teleport_parcel"))

	assert.True(t, IsValidProviderAttemptStatus(ProviderAttemptPending))
	assert.True(t, IsValidProviderAttemptStatus(ProviderAttemptSucceeded))
	assert.True(t, IsValidProviderAttemptStatus(ProviderAttemptFailed))
	assert.False(t, IsValidProviderAttemptStatus("maybe"))
}

func TestCarrierFailureBlockerCode_ReusesExistingCodes(t *testing.T) {
	cases := map[string]string{
		CarrierFailMissingData:       BlockerManualStockReviewRequired,
		CarrierFailProviderRejection: BlockerIntegrationCapabilityMissing,
		CarrierFailProviderOutage:    BlockerIntegrationCapabilityDegraded,
		CarrierFailRateLimit:         BlockerIntegrationCapabilityDegraded,
		CarrierFailAuth:              BlockerIntegrationCapabilityMissing,
		"unknown_class":              BlockerIntegrationCapabilityDegraded, // default fallback
	}
	for failClass, wantCode := range cases {
		got := CarrierFailureBlockerCode(failClass)
		assert.Equal(t, wantCode, got, "failure class %q", failClass)
		// CRITICAL: every mapped code MUST be an existing, valid blocker code —
		// OPE-417 introduces no new blocker codes.
		assert.True(t, IsValidBlockerCode(got), "mapped code %q must be a known blocker code", got)
		assert.NotEmpty(t, BlockerCategory(got), "mapped code %q must resolve to a category", got)
	}
}

func TestNewTrackingStatusMapping_PreservesRaw(t *testing.T) {
	// Mapped: canonical retained, raw preserved.
	m := NewTrackingStatusMapping("DELIVERED_TO_LOCKER", "delivered", true)
	assert.Equal(t, "DELIVERED_TO_LOCKER", m.Raw)
	assert.Equal(t, "delivered", m.Canonical)
	assert.True(t, m.Mapped)

	// Unmapped: raw STILL preserved, canonical forced empty, mapped=false.
	u := NewTrackingStatusMapping("SOME_EXOTIC_CARRIER_STATE", "should_be_dropped", false)
	assert.Equal(t, "SOME_EXOTIC_CARRIER_STATE", u.Raw, "raw provider status must never be lost")
	assert.Empty(t, u.Canonical, "canonical must be empty when unmapped")
	assert.False(t, u.Mapped)
}
