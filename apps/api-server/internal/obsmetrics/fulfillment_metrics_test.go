package obsmetrics_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/obsmetrics"
)

func render(m *obsmetrics.FulfillmentMetrics) string {
	var b strings.Builder
	m.Render(&b)
	return b.String()
}

func TestNewFulfillmentMetrics_NotNil(t *testing.T) {
	assert.NotNil(t, obsmetrics.NewFulfillmentMetrics())
}

func TestFulfillmentMetrics_NilReceiver_NoPanic(t *testing.T) {
	var m *obsmetrics.FulfillmentMetrics // nil
	// None of the record methods may panic on a nil collector — this is the
	// guarantee that wiring it in is always safe and best-effort.
	require.NotPanics(t, func() {
		m.RecordProviderAttempt("create_shipment", "succeeded")
		m.RecordBlocker("capability")
		m.RecordOutboxEvent("processed")
		m.SetOutboxQueueDepth(5)
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
	m := obsmetrics.NewFulfillmentMetrics()
	m.RecordProviderAttempt("generate_label", "failed")
	m.RecordProviderAttempt("generate_label", "failed")
	out := render(m)
	assert.Contains(t, out, "# TYPE openoms_fulfillment_provider_attempts_total counter")
	assert.Contains(t, out, `openoms_fulfillment_provider_attempts_total{operation="generate_label",status="failed"} 2`)
}

func TestFulfillmentMetrics_UnknownLabelCoercedToOther(t *testing.T) {
	m := obsmetrics.NewFulfillmentMetrics()
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
	m := obsmetrics.NewFulfillmentMetrics()
	m.RecordBlocker("supplier")
	out := render(m)
	assert.Contains(t, out, "# TYPE openoms_fulfillment_blockers_total counter")
	assert.Contains(t, out, `openoms_fulfillment_blockers_total{category="supplier"} 1`)
}

func TestFulfillmentMetrics_OutboxCountersAndGauge(t *testing.T) {
	m := obsmetrics.NewFulfillmentMetrics()
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

func TestFulfillmentMetrics_GlobalProcessGauges(t *testing.T) {
	m := obsmetrics.NewFulfillmentMetrics()
	m.SetStuckProcesses(3)
	m.SetBlockedProcesses(5)
	out := render(m)
	assert.Contains(t, out, "# TYPE openoms_fulfillment_stuck_processes gauge")
	assert.Contains(t, out, "openoms_fulfillment_stuck_processes 3")
	assert.Contains(t, out, "# TYPE openoms_fulfillment_blocked_processes gauge")
	assert.Contains(t, out, "openoms_fulfillment_blocked_processes 5")
	// Cardinality discipline: these are GLOBAL aggregates and must carry NO label set.
	assert.NotContains(t, out, "openoms_fulfillment_stuck_processes{")
	assert.NotContains(t, out, "openoms_fulfillment_blocked_processes{")
	assert.NotContains(t, out, "tenant_id")
}

func TestFulfillmentMetrics_ValidationCounters(t *testing.T) {
	m := obsmetrics.NewFulfillmentMetrics()
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
	m := obsmetrics.NewFulfillmentMetrics()
	m.RecordPublicationTransition("available")
	out := render(m)
	assert.Contains(t, out, "# TYPE openoms_provider_publication_transitions_total counter")
	assert.Contains(t, out, `openoms_provider_publication_transitions_total{state="available"} 1`)
}

func TestFulfillmentMetrics_UnitAndStepTransitions(t *testing.T) {
	m := obsmetrics.NewFulfillmentMetrics()
	m.RecordUnitTransition("running")
	m.RecordStepTransition("succeeded")
	out := render(m)
	assert.Contains(t, out, "# TYPE openoms_fulfillment_unit_transitions_total counter")
	assert.Contains(t, out, `openoms_fulfillment_unit_transitions_total{status="running"} 1`)
	assert.Contains(t, out, "# TYPE openoms_fulfillment_step_transitions_total counter")
	assert.Contains(t, out, `openoms_fulfillment_step_transitions_total{status="succeeded"} 1`)
}

// TestFulfillmentMetrics_AllowListsCoverModelEnums is the allow-list drift guard
// (OPE-527): every model.ProviderOp* operation and every blocker category produced
// by model.BlockerCategory must be accepted as-is by the metric allow-lists, never
// coerced to the bounded "other" bucket. If a domain enum gains a value without the
// matching allow-list entry, this test fails.
func TestFulfillmentMetrics_AllowListsCoverModelEnums(t *testing.T) {
	operations := []string{
		model.ProviderOpCreateShipment,
		model.ProviderOpGenerateLabel,
		model.ProviderOpDownloadLabel,
		model.ProviderOpSyncTracking,
		model.ProviderOpSyncTrackingToMarketplace,
		model.ProviderOpSyncFulfillmentStatus,
	}
	for _, op := range operations {
		m := obsmetrics.NewFulfillmentMetrics()
		m.RecordProviderAttempt(op, "succeeded")
		out := render(m)
		assert.Containsf(t, out, fmt.Sprintf("operation=%q", op), "operation %q must be allow-listed", op)
		assert.NotContainsf(t, out, `operation="other"`, "operation %q must not be coerced to other", op)
	}

	// One representative blocker code per category, resolved through the same
	// model.BlockerCategory pathway callers use (e.g. the orchestration worker).
	representativeBlockerCodes := []string{
		model.BlockerStockSyncFailed,           // integration
		model.BlockerSupplierAvailabilityStale, // supplier
		model.BlockerManualStockReviewRequired, // operator
		model.BlockerStockWriteUnsupported,     // capability
		model.BlockerExternalStatusUnmapped,    // mapping
		model.BlockerAutomationActionFailed,    // automation (OPE-527)
	}
	for _, code := range representativeBlockerCodes {
		category := model.BlockerCategory(code)
		require.NotEmptyf(t, category, "blocker code %q has no category", code)
		m := obsmetrics.NewFulfillmentMetrics()
		m.RecordBlocker(category)
		out := render(m)
		assert.Containsf(t, out, fmt.Sprintf("category=%q", category), "category %q must be allow-listed", category)
		assert.NotContainsf(t, out, `category="other"`, "category %q must not be coerced to other", category)
	}
}

// TestFulfillmentMetrics_NoIDLikeLabels is the cardinality-discipline guard: the
// rendered exposition MUST NOT contain any unbounded id-like label key. If a future
// change introduces tenant_id/order_id/etc. as a metric label, this test fails.
func TestFulfillmentMetrics_NoIDLikeLabels(t *testing.T) {
	m := obsmetrics.NewFulfillmentMetrics()
	// Exercise every metric so all label keys appear in the output.
	m.RecordProviderAttempt("create_shipment", "succeeded")
	m.RecordProviderAttempt("sync_tracking_to_marketplace", "succeeded")
	m.RecordProviderAttempt("sync_fulfillment_status", "failed")
	m.RecordBlocker("capability")
	m.RecordBlocker("automation")
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
