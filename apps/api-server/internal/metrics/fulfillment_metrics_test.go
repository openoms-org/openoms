package metrics_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/metrics"
)

func render(m *metrics.FulfillmentMetrics) string {
	var b strings.Builder
	m.Render(&b)
	return b.String()
}

func TestNewFulfillmentMetrics_NotNil(t *testing.T) {
	assert.NotNil(t, metrics.NewFulfillmentMetrics())
}

func TestFulfillmentMetrics_NilReceiver_NoPanic(t *testing.T) {
	var m *metrics.FulfillmentMetrics // nil
	// None of the record methods may panic on a nil collector — this is the
	// guarantee that wiring it in is always safe and best-effort.
	require.NotPanics(t, func() {
		m.RecordProviderAttempt("create_shipment", "succeeded")
		m.RecordBlocker("capability")
		m.RecordOutboxEvent("processed")
		m.SetOutboxQueueDepth(5)
		m.SetStuckProcesses(1)
		m.SetBlockedProcesses(2)
		m.RecordValidationRun("passed")
		m.RecordValidationFailure()
		m.RecordPublicationTransition("available")
		m.RecordUnitTransition("running")
		m.RecordStepTransition("succeeded")
		var b strings.Builder
		m.Render(&b)
	})
}

func TestFulfillmentMetrics_ProviderAttemptCounter(t *testing.T) {
	m := metrics.NewFulfillmentMetrics()
	m.RecordProviderAttempt("generate_label", "failed")
	m.RecordProviderAttempt("generate_label", "failed")
	out := render(m)
	assert.Contains(t, out, "# TYPE openoms_fulfillment_provider_attempts_total counter")
	assert.Contains(t, out, `openoms_fulfillment_provider_attempts_total{operation="generate_label",status="failed"} 2`)
}

func TestFulfillmentMetrics_UnknownLabelCoercedToOther(t *testing.T) {
	m := metrics.NewFulfillmentMetrics()
	// An unbounded / unexpected value (e.g. an id leaked by a bug) must NOT create
	// a new high-cardinality series — it is coerced to the bounded "other" bucket.
	m.RecordProviderAttempt("11111111-2222-3333-4444-555555555555", "weird-status")
	out := render(m)
	assert.Contains(t, out, `operation="other"`)
	assert.Contains(t, out, `status="other"`)
	assert.NotContains(t, out, "11111111-2222-3333-4444-555555555555")
	assert.NotContains(t, out, "weird-status")
}

func TestFulfillmentMetrics_BlockerCategoryCounter(t *testing.T) {
	m := metrics.NewFulfillmentMetrics()
	m.RecordBlocker("supplier")
	out := render(m)
	assert.Contains(t, out, "# TYPE openoms_fulfillment_blockers_total counter")
	assert.Contains(t, out, `openoms_fulfillment_blockers_total{category="supplier"} 1`)
}

func TestFulfillmentMetrics_OutboxCountersAndGauge(t *testing.T) {
	m := metrics.NewFulfillmentMetrics()
	m.RecordOutboxEvent("claimed")
	m.RecordOutboxEvent("processed")
	m.RecordOutboxEvent("failed")
	m.SetOutboxQueueDepth(7)
	out := render(m)
	assert.Contains(t, out, "# TYPE openoms_orchestration_outbox_events_total counter")
	assert.Contains(t, out, `openoms_orchestration_outbox_events_total{result="claimed"} 1`)
	assert.Contains(t, out, `openoms_orchestration_outbox_events_total{result="processed"} 1`)
	assert.Contains(t, out, `openoms_orchestration_outbox_events_total{result="failed"} 1`)
	assert.Contains(t, out, "# TYPE openoms_orchestration_outbox_queue_depth gauge")
	assert.Contains(t, out, "openoms_orchestration_outbox_queue_depth 7")
}

func TestFulfillmentMetrics_StuckBlockedGauges(t *testing.T) {
	m := metrics.NewFulfillmentMetrics()
	m.SetStuckProcesses(3)
	m.SetBlockedProcesses(4)
	out := render(m)
	assert.Contains(t, out, "# TYPE openoms_fulfillment_stuck_processes gauge")
	assert.Contains(t, out, "openoms_fulfillment_stuck_processes 3")
	assert.Contains(t, out, "# TYPE openoms_fulfillment_blocked_processes gauge")
	assert.Contains(t, out, "openoms_fulfillment_blocked_processes 4")
}

func TestFulfillmentMetrics_ValidationCounters(t *testing.T) {
	m := metrics.NewFulfillmentMetrics()
	m.RecordValidationRun("failed")
	m.RecordValidationFailure()
	m.RecordValidationFailure()
	out := render(m)
	assert.Contains(t, out, "# TYPE openoms_provider_validation_runs_total counter")
	assert.Contains(t, out, `openoms_provider_validation_runs_total{result="failed"} 1`)
	assert.Contains(t, out, "# TYPE openoms_provider_validation_failures_total counter")
	assert.Contains(t, out, "openoms_provider_validation_failures_total 2")
}

func TestFulfillmentMetrics_PublicationTransitionCounter(t *testing.T) {
	m := metrics.NewFulfillmentMetrics()
	m.RecordPublicationTransition("available")
	out := render(m)
	assert.Contains(t, out, "# TYPE openoms_provider_publication_transitions_total counter")
	assert.Contains(t, out, `openoms_provider_publication_transitions_total{state="available"} 1`)
}

func TestFulfillmentMetrics_UnitAndStepTransitions(t *testing.T) {
	m := metrics.NewFulfillmentMetrics()
	m.RecordUnitTransition("running")
	m.RecordStepTransition("succeeded")
	out := render(m)
	assert.Contains(t, out, "# TYPE openoms_fulfillment_unit_transitions_total counter")
	assert.Contains(t, out, `openoms_fulfillment_unit_transitions_total{status="running"} 1`)
	assert.Contains(t, out, "# TYPE openoms_fulfillment_step_transitions_total counter")
	assert.Contains(t, out, `openoms_fulfillment_step_transitions_total{status="succeeded"} 1`)
}

// TestFulfillmentMetrics_NoIDLikeLabels is the cardinality-discipline guard: the
// rendered exposition MUST NOT contain any unbounded id-like label key. If a future
// change introduces tenant_id/order_id/etc. as a metric label, this test fails.
func TestFulfillmentMetrics_NoIDLikeLabels(t *testing.T) {
	m := metrics.NewFulfillmentMetrics()
	// Exercise every metric so all label keys appear in the output.
	m.RecordProviderAttempt("create_shipment", "succeeded")
	m.RecordBlocker("capability")
	m.RecordOutboxEvent("processed")
	m.SetOutboxQueueDepth(1)
	m.SetStuckProcesses(1)
	m.SetBlockedProcesses(1)
	m.RecordValidationRun("passed")
	m.RecordValidationFailure()
	m.RecordPublicationTransition("available")
	m.RecordUnitTransition("running")
	m.RecordStepTransition("succeeded")

	out := render(m)
	forbidden := []string{"tenant_id", "order_id", "process_id", "unit_id", "user_id", "step_id", "blocker_id", "version_id", "run_id"}
	for _, key := range forbidden {
		assert.NotContainsf(t, out, key+"=", "metric exposition must not use the unbounded label %q", key)
	}
}
