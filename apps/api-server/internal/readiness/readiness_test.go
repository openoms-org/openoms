package readiness

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsFeatureEnabled(t *testing.T) {
	cases := []struct {
		feature, mode string
		want          bool
	}{
		{"repricing", "client-ready", false},       // beta blocked in client-ready
		{"repricing", "full", true},                // beta allowed in full
		{"marketing", "full", false},               // blocked never allowed
		{"unknown_feature", "client-ready", false}, // unknown -> treat as not-ready
		{"unknown_feature", "full", true},          // unknown -> non-blocked, allowed in full
	}
	for _, c := range cases {
		assert.Equal(t, c.want, IsFeatureEnabled(c.feature, c.mode),
			"IsFeatureEnabled(%q,%q)", c.feature, c.mode)
	}
}

func TestIsProviderEnabled(t *testing.T) {
	assert.True(t, IsProviderEnabled("allegro", "client-ready"), "allegro should be enabled in client-ready")
	assert.False(t, IsProviderEnabled("amazon", "client-ready"), "amazon (beta) must be blocked in client-ready")
	assert.False(t, IsProviderEnabled("shopify", "full"), "shopify (blocked) must never be enabled")
}
