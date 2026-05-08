package worker

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

func TestPlanDelayedActionFailureRetriesWithExponentialBackoff(t *testing.T) {
	now := time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC)

	cases := []struct {
		name         string
		attemptCount int
		wantDelay    time.Duration
	}{
		{name: "first failure", attemptCount: 0, wantDelay: time.Minute},
		{name: "second failure", attemptCount: 1, wantDelay: 2 * time.Minute},
		{name: "fourth failure", attemptCount: 3, wantDelay: 8 * time.Minute},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			plan := planDelayedActionFailure(model.DelayedAction{AttemptCount: tc.attemptCount}, now)

			assert.True(t, plan.Retry)
			assert.Equal(t, tc.attemptCount+1, plan.AttemptCount)
			assert.Equal(t, now.Add(tc.wantDelay), plan.NextExecuteAt)
		})
	}
}

func TestPlanDelayedActionFailurePermanentAfterMaxAttempts(t *testing.T) {
	now := time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC)

	plan := planDelayedActionFailure(model.DelayedAction{AttemptCount: delayedActionMaxAttempts - 1}, now)

	assert.False(t, plan.Retry)
	assert.Equal(t, delayedActionMaxAttempts, plan.AttemptCount)
	assert.True(t, plan.NextExecuteAt.IsZero())
}
