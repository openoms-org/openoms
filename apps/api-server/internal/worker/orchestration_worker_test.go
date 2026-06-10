package worker

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/obsmetrics"
)

// TestNextRetryAt_BackoffInputConsistency guards the OPE-522 off-by-one: every
// requeue path (panic, start-attempt failure, dispatch failure) computes its
// backoff via nextRetryAt from the event's PRE-attempt counter, and the backoff
// input must be the attempt number that just failed (attempts+1) — never the raw
// pre-attempt counter.
func TestNextRetryAt_BackoffInputConsistency(t *testing.T) {
	now := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)

	// First failure (Attempts=0 before the attempt) backs off by
	// NextOutboxBackoff(1) = 60s, NOT NextOutboxBackoff(0) = 30s.
	assert.Equal(t, now.Add(60*time.Second), nextRetryAt(now, 0))

	for _, attempts := range []int{0, 1, 2, 5, 20} {
		assert.Equal(t, now.Add(model.NextOutboxBackoff(attempts+1)), nextRetryAt(now, attempts),
			"attempts=%d", attempts)
	}
}

func renderMetrics(m *obsmetrics.FulfillmentMetrics) string {
	var b strings.Builder
	m.Render(&b)
	return b.String()
}

// TestOrchestrationWorker_WithMetrics_NilSafe verifies the metrics handle is optional:
// a worker constructed without WithMetrics records nothing and never panics.
func TestOrchestrationWorker_WithMetrics_NilSafe(t *testing.T) {
	w := NewOrchestrationWorker(nil, nil, nil, nil, 0, nil)
	assert.NotPanics(t, func() {
		w.recordOutcome("processed")
		w.recordOutcome("failed")
		w.recordClaimed(3)
		w.setQueueDepth(5)
	})
}

// TestOrchestrationWorker_RecordOutcome_Processed asserts the success path increments
// the bounded "processed" outbox result counter.
func TestOrchestrationWorker_RecordOutcome_Processed(t *testing.T) {
	m := obsmetrics.NewFulfillmentMetrics()
	w := NewOrchestrationWorker(nil, nil, nil, nil, 0, nil).WithMetrics(m)

	w.recordOutcome("processed")
	w.recordOutcome("processed")

	out := renderMetrics(m)
	assert.Contains(t, out, `openoms_orchestration_outbox_events_total{result="processed"} 2`)
}

// TestOrchestrationWorker_RecordOutcome_Failed asserts the failure path increments the
// bounded "failed" outbox result counter.
func TestOrchestrationWorker_RecordOutcome_Failed(t *testing.T) {
	m := obsmetrics.NewFulfillmentMetrics()
	w := NewOrchestrationWorker(nil, nil, nil, nil, 0, nil).WithMetrics(m)

	w.recordOutcome("failed")

	out := renderMetrics(m)
	assert.Contains(t, out, `openoms_orchestration_outbox_events_total{result="failed"} 1`)
}

// TestOrchestrationWorker_RecordClaimed asserts each claimed event is counted and the
// queue-depth gauge is published.
func TestOrchestrationWorker_RecordClaimedAndQueueDepth(t *testing.T) {
	m := obsmetrics.NewFulfillmentMetrics()
	w := NewOrchestrationWorker(nil, nil, nil, nil, 0, nil).WithMetrics(m)

	w.recordClaimed(4)
	w.setQueueDepth(9)

	out := renderMetrics(m)
	assert.Contains(t, out, `openoms_orchestration_outbox_events_total{result="claimed"} 4`)
	assert.Contains(t, out, "openoms_orchestration_outbox_queue_depth 9")
}
