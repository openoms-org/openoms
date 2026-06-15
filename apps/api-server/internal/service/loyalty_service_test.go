package service

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestHighestQualifyingTier locks the tier-selection logic used by AwardPointsForOrder:
// given a tier config and total spent, it iterates config tiers in order and keeps the last whose
// min_spent is met.
func TestHighestQualifyingTier(t *testing.T) {
	cfg := json.RawMessage(`{"tiers":[
		{"name":"bronze","min_spent":0},
		{"name":"silver","min_spent":500},
		{"name":"gold","min_spent":1000}
	]}`)

	cases := []struct {
		name       string
		config     json.RawMessage
		totalSpent float64
		wantTier   string
		wantOK     bool
	}{
		{"below first qualifies bronze", cfg, 0, "bronze", true},
		{"mid qualifies silver", cfg, 750, "silver", true},
		{"high qualifies gold", cfg, 5000, "gold", true},
		{"exact boundary silver", cfg, 500, "silver", true},
		{"no tier qualifies", json.RawMessage(`{"tiers":[{"name":"vip","min_spent":100}]}`), 50, "", false},
		{"invalid config", json.RawMessage(`not json`), 1000, "", false},
		{"empty tiers", json.RawMessage(`{"tiers":[]}`), 1000, "", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tier, ok := highestQualifyingTier(tc.config, tc.totalSpent)
			assert.Equal(t, tc.wantOK, ok)
			assert.Equal(t, tc.wantTier, tier)
		})
	}
}
