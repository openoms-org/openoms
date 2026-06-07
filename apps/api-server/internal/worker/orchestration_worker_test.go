package worker

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/openoms-org/openoms/apps/api-server/internal/metrics"
)

func renderMetrics(m *metrics.FulfillmentMetrics) string {
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
	m := metrics.NewFulfillmentMetrics()
	w := NewOrchestrationWorker(nil, nil, nil, nil, 0, nil).WithMetrics(m)

	w.recordOutcome("processed")
	w.recordOutcome("processed")

	out := renderMetrics(m)
	assert.Contains(t, out, `openoms_orchestration_outbox_events_total{result="processed"} 2`)
}

// TestOrchestrationWorker_RecordOutcome_Failed asserts the failure path increments the
// bounded "failed" outbox result counter.
func TestOrchestrationWorker_RecordOutcome_Failed(t *testing.T) {
	m := metrics.NewFulfillmentMetrics()
	w := NewOrchestrationWorker(nil, nil, nil, nil, 0, nil).WithMetrics(m)

	w.recordOutcome("failed")

	out := renderMetrics(m)
	assert.Contains(t, out, `openoms_orchestration_outbox_events_total{result="failed"} 1`)
}

// TestOrchestrationWorker_RecordClaimed asserts each claimed event is counted and the
// queue-depth gauge is published.
func TestOrchestrationWorker_RecordClaimedAndQueueDepth(t *testing.T) {
	m := metrics.NewFulfillmentMetrics()
	w := NewOrchestrationWorker(nil, nil, nil, nil, 0, nil).WithMetrics(m)

	w.recordClaimed(4)
	w.setQueueDepth(9)

	out := renderMetrics(m)
	assert.Contains(t, out, `openoms_orchestration_outbox_events_total{result="claimed"} 4`)
	assert.Contains(t, out, "openoms_orchestration_outbox_queue_depth 9")
}
