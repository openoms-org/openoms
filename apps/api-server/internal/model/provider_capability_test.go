package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateCapabilities(t *testing.T) {
	require.NoError(t, ValidateCapabilities([]ProviderCapability{
		{CapabilityKey: "supplier.order.create", SupportStatus: SupportStatusSupported},
		{CapabilityKey: "supplier.tracking.read", SupportStatus: SupportStatusRequiresManual},
	}))

	neg := -1
	cases := map[string][]ProviderCapability{
		"empty key":   {{CapabilityKey: "", SupportStatus: SupportStatusSupported}},
		"dup key":     {{CapabilityKey: "a", SupportStatus: SupportStatusSupported}, {CapabilityKey: "a", SupportStatus: SupportStatusUnknown}},
		"bad support": {{CapabilityKey: "a", SupportStatus: "maybe"}},
		"neg latency": {{CapabilityKey: "a", SupportStatus: SupportStatusSupported, LatencySLASeconds: &neg}},
	}
	for name, caps := range cases {
		t.Run(name, func(t *testing.T) { assert.Error(t, ValidateCapabilities(caps)) })
	}
}

func TestValidateStatusMappings(t *testing.T) {
	require.NoError(t, ValidateStatusMappings([]ProviderStatusMapping{
		{StatusDomain: StatusDomainOrder, RawStatus: "NEW", CanonicalStatus: "new", Confidence: MappingConfidenceHigh},
		{StatusDomain: StatusDomainShipment, RawStatus: "NEW", Confidence: MappingConfidenceLow}, // same raw, different domain OK
	}))

	cases := map[string][]ProviderStatusMapping{
		"empty raw":  {{StatusDomain: StatusDomainOrder, RawStatus: "", Confidence: MappingConfidenceHigh}},
		"bad domain": {{StatusDomain: "galaxy", RawStatus: "X", Confidence: MappingConfidenceHigh}},
		"bad conf":   {{StatusDomain: StatusDomainOrder, RawStatus: "X", Confidence: "sure"}},
		"dup pair":   {{StatusDomain: StatusDomainOrder, RawStatus: "X", Confidence: MappingConfidenceHigh}, {StatusDomain: StatusDomainOrder, RawStatus: "X", Confidence: MappingConfidenceLow}},
	}
	for name, m := range cases {
		t.Run(name, func(t *testing.T) { assert.Error(t, ValidateStatusMappings(m)) })
	}
}

func TestGapEnumValidators(t *testing.T) {
	assert.True(t, IsValidGapType(GapTypeMissingStatusMapping))
	assert.False(t, IsValidGapType("meteor_strike"))
	assert.True(t, IsValidGapSeverity(GapSeverityActionRequired))
	assert.False(t, IsValidGapSeverity("meh"))
	assert.True(t, IsValidGapStatus(GapStatusResolved))
	assert.False(t, IsValidGapStatus("pending"))
}
