package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFulfillmentValidators(t *testing.T) {
	assert.True(t, IsValidAggregateStatus(ProcessStatusInProgress))
	assert.False(t, IsValidAggregateStatus("teleporting"))
	assert.True(t, IsValidHealthStatus(ProcessHealthActionRequired))
	assert.False(t, IsValidHealthStatus("meh"))
	assert.True(t, IsValidUnitType(UnitTypeDropship))
	assert.False(t, IsValidUnitType("teleporter"))
	assert.True(t, IsValidFulfillmentStatus(FulfillmentStatusWaitingExternal))
	assert.False(t, IsValidFulfillmentStatus("done"))
	assert.True(t, IsValidStepKey(StepGenerateLabel))
	assert.False(t, IsValidStepKey("summon_owl"))
	assert.True(t, IsValidBlockerStatus(BlockerStatusResolved))
	assert.False(t, IsValidBlockerStatus("ignored"))
}

func TestFulfillmentStepKeysComplete(t *testing.T) {
	assert.Len(t, validFulfillmentStepKeys, 22, "ADR-424 defines 22 canonical step keys")
}

func TestBlockerCategory(t *testing.T) {
	cases := map[string]string{
		BlockerStockSyncFailed:               "integration",
		BlockerSupplierAvailabilityStale:     "supplier",
		BlockerManualStockReviewRequired:     "operator",
		BlockerStockWriteUnsupported:         "capability",
		BlockerExternalStatusUnmapped:        "mapping",
		BlockerIntegrationCapabilityDegraded: "capability",
	}
	for code, cat := range cases {
		assert.True(t, IsValidBlockerCode(code))
		assert.Equal(t, cat, BlockerCategory(code))
	}
	assert.False(t, IsValidBlockerCode("alien_invasion"))
	assert.Equal(t, "", BlockerCategory("alien_invasion"))
}

func TestBlockerSupplierAvailabilityInsufficient_IsValidWithCategory(t *testing.T) {
	assert.True(t, IsValidBlockerCode(BlockerSupplierAvailabilityInsufficient))
	assert.Equal(t, "supplier_availability_insufficient", BlockerSupplierAvailabilityInsufficient)
	assert.Equal(t, "supplier", blockerCategories[BlockerSupplierAvailabilityInsufficient])
}
