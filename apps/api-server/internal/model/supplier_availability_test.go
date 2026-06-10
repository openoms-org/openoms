package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSupplierAvailabilityValidators(t *testing.T) {
	assert.True(t, IsValidAvailabilityType("exact_quantity"))
	assert.True(t, IsValidAvailabilityType("bucket"))
	assert.True(t, IsValidAvailabilityType("boolean"))
	assert.True(t, IsValidAvailabilityType("eta_only"))
	assert.True(t, IsValidAvailabilityType("unknown"))
	assert.False(t, IsValidAvailabilityType("magic"))

	assert.True(t, IsValidPolicyScope("supplier"))
	assert.True(t, IsValidPolicyScope("product"))
	assert.True(t, IsValidPolicyScope("listing"))
	assert.True(t, IsValidPolicyScope("channel"))
	assert.False(t, IsValidPolicyScope("global"))

	assert.True(t, IsValidPolicyMode("auto"))
	assert.True(t, IsValidPolicyMode("manual"))
	assert.True(t, IsValidPolicyMode("paused"))
	assert.False(t, IsValidPolicyMode("frozen"))
}
