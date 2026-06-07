package service

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// computeParity is a PURE function (no DB), so the coverage math, threshold
// normalisation, verdict and edge cases are exercised here without a database. The
// DB-bound counting + RLS isolation is covered by the integration test.

func TestComputeParity_FullCoverage_MeetsDefaultThreshold(t *testing.T) {
	in := parityInputs{
		nonTerminalOrders:    100,
		fulfillmentProcesses: 100,
		ordersMissingProcess: 0,
		legacyProblemOrders:  3,
		processedExceptions:  2,
	}
	r := computeParity(in, 0) // 0 -> default threshold

	assert.Equal(t, 100, r.NonTerminalOrders)
	assert.Equal(t, 100, r.FulfillmentProcesses)
	assert.Equal(t, 0, r.OrdersMissingProcess)
	assert.InDelta(t, 1.0, r.ProcessCoverage, 1e-9)
	assert.Equal(t, 3, r.LegacyProblemOrders)
	assert.Equal(t, 2, r.ProcessBackedExceptions)
	assert.Equal(t, DefaultParityCoverageThreshold, r.CoverageThreshold)
	assert.True(t, r.ProcessCoverageMet, "full coverage meets the default threshold")
}

func TestComputeParity_PartialCoverage_BelowThreshold(t *testing.T) {
	// 90 of 100 covered -> 0.90 < 0.99 default -> gate NOT met.
	in := parityInputs{nonTerminalOrders: 100, ordersMissingProcess: 10}
	r := computeParity(in, 0)

	assert.InDelta(t, 0.90, r.ProcessCoverage, 1e-9)
	assert.False(t, r.ProcessCoverageMet, "0.90 coverage is below the 0.99 default gate")
}

func TestComputeParity_JustAtThreshold_Meets(t *testing.T) {
	// 99 of 100 covered -> exactly 0.99, which is >= 0.99 (inclusive) -> met.
	in := parityInputs{nonTerminalOrders: 100, ordersMissingProcess: 1}
	r := computeParity(in, 0)

	assert.InDelta(t, 0.99, r.ProcessCoverage, 1e-9)
	assert.True(t, r.ProcessCoverageMet, "coverage == threshold is met (inclusive)")
}

func TestComputeParity_ZeroOrders_VacuouslyCovered(t *testing.T) {
	// No non-terminal orders: coverage is 1.0 (nothing to cover), gate met — an
	// empty/quiet tenant must not be blocked from cutover by a 0/0.
	r := computeParity(parityInputs{}, 0)

	assert.Equal(t, 0, r.NonTerminalOrders)
	assert.InDelta(t, 1.0, r.ProcessCoverage, 1e-9)
	assert.True(t, r.ProcessCoverageMet)
}

func TestComputeParity_CustomThreshold_FullCoverageRequired(t *testing.T) {
	// Operator demands full 1.0 coverage: 999/1000 (0.999) is NOT enough.
	in := parityInputs{nonTerminalOrders: 1000, ordersMissingProcess: 1}
	r := computeParity(in, 1.0)

	assert.Equal(t, 1.0, r.CoverageThreshold)
	assert.InDelta(t, 0.999, r.ProcessCoverage, 1e-9)
	assert.False(t, r.ProcessCoverageMet, "0.999 < 1.0 required -> not met")
}

func TestComputeParity_ThresholdClamping(t *testing.T) {
	// Above-1 thresholds clamp to 1; non-positive falls back to the default.
	assert.Equal(t, 1.0, computeParity(parityInputs{}, 5.0).CoverageThreshold)
	assert.Equal(t, DefaultParityCoverageThreshold, computeParity(parityInputs{}, -0.5).CoverageThreshold)
	assert.Equal(t, DefaultParityCoverageThreshold, computeParity(parityInputs{}, 0).CoverageThreshold)
	assert.InDelta(t, 0.5, computeParity(parityInputs{}, 0.5).CoverageThreshold, 1e-9)
}

func TestComputeParity_MissingNeverProducesNegativeCoverage(t *testing.T) {
	// Defensive: missing should never exceed the population, but if a caller fed an
	// inconsistent pair, coverage must never go below 0.
	in := parityInputs{nonTerminalOrders: 10, ordersMissingProcess: 50}
	r := computeParity(in, 0)

	assert.GreaterOrEqual(t, r.ProcessCoverage, 0.0, "coverage is floored at 0")
	assert.False(t, r.ProcessCoverageMet)
}

func TestNormaliseThreshold(t *testing.T) {
	assert.Equal(t, DefaultParityCoverageThreshold, normaliseThreshold(0))
	assert.Equal(t, DefaultParityCoverageThreshold, normaliseThreshold(-1))
	assert.Equal(t, 1.0, normaliseThreshold(2))
	assert.InDelta(t, 0.95, normaliseThreshold(0.95), 1e-9)
}

func TestIsProcessBackedException(t *testing.T) {
	cases := []struct {
		name      string
		aggregate string
		health    string
		want      bool
	}{
		{"blocked is an exception", model.ProcessStatusBlocked, model.ProcessHealthActionRequired, true},
		{"waiting_external (provider_issue) is an exception", model.ProcessStatusWaitingExternal, model.ProcessHealthWarning, true},
		{"in_progress + system_error (stuck) is an exception", model.ProcessStatusInProgress, model.ProcessHealthSystemError, true},
		{"healthy in_progress is NOT an exception", model.ProcessStatusInProgress, model.ProcessHealthOK, false},
		{"ready is NOT an exception", model.ProcessStatusReady, model.ProcessHealthOK, false},
		{"new is NOT an exception", model.ProcessStatusNew, model.ProcessHealthOK, false},
		{"completed (terminal) is NOT an exception", model.ProcessStatusCompleted, model.ProcessHealthOK, false},
		{"cancelled (terminal) is NOT an exception", model.ProcessStatusCancelled, model.ProcessHealthOK, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, isProcessBackedException(tc.aggregate, tc.health))
		})
	}
}
