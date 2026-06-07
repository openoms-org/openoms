package service

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/obsmetrics"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

// OPE-422: these tests cover the ADDITIVE, best-effort metric wiring on
// FulfillmentService / fulfillment_unit_service. The metric-emitting code paths
// (RecordProviderAttempt, CreateCarrierBlocker, CreateSupplierBlocker,
// RecordUnitTransition, RecordStep) all run inside a DB transaction, so the metric
// emission decision is what we verify here:
//   - WithMetrics wiring is nil-safe and chainable;
//   - the bounded values the service feeds the collector (provider operation/status
//     enums, and the blocker CATEGORY derived from model.BlockerCategory) render as
//     bounded label series — never an unbounded identifier.

func renderFulfillmentMetrics(m *obsmetrics.FulfillmentMetrics) string {
	var b strings.Builder
	m.Render(&b)
	return b.String()
}

func TestFulfillmentService_WithMetrics_Chainable(t *testing.T) {
	m := obsmetrics.NewFulfillmentMetrics()
	svc := NewFulfillmentService(true, repository.NewFulfillmentRepository(), repository.NewOrchestrationRepository()).
		WithMetrics(m)
	require.NotNil(t, svc)
	assert.Same(t, m, svc.metrics)
}

func TestFulfillmentService_WithMetrics_NilReceiverSafe(t *testing.T) {
	var svc *FulfillmentService // nil
	assert.NotPanics(t, func() {
		got := svc.WithMetrics(obsmetrics.NewFulfillmentMetrics())
		assert.Nil(t, got)
	})
}

// Without a wired collector the service field stays nil; every record call on it is
// a no-op (the collector is nil-receiver safe), so recording never breaks the
// primary operation even when metrics are not configured.
func TestFulfillmentService_NilMetrics_RecordIsNoOp(t *testing.T) {
	svc := NewFulfillmentService(true, repository.NewFulfillmentRepository(), repository.NewOrchestrationRepository())
	require.Nil(t, svc.metrics)
	assert.NotPanics(t, func() {
		svc.metrics.RecordProviderAttempt(model.ProviderOpCreateShipment, model.ProviderAttemptSucceeded)
		svc.metrics.RecordBlocker(model.BlockerCategory(model.BlockerIntegrationCapabilityMissing))
		svc.metrics.RecordUnitTransition(model.FulfillmentStatusRunning)
		svc.metrics.RecordStepTransition(model.FulfillmentStatusSucceeded)
	})
}

// The provider-attempt operation+status the service records (from the model enums)
// must render as a bounded two-label series.
func TestFulfillmentService_ProviderAttemptMetric_Bounded(t *testing.T) {
	m := obsmetrics.NewFulfillmentMetrics()
	// Exactly the values RecordProviderAttempt forwards (in.Operation, in.Status).
	m.RecordProviderAttempt(model.ProviderOpGenerateLabel, model.ProviderAttemptFailed)

	out := renderFulfillmentMetrics(m)
	assert.Contains(t, out, `openoms_fulfillment_provider_attempts_total{operation="generate_label",status="failed"} 1`)
}

// CreateCarrierBlocker / CreateSupplierBlocker derive the metric category via
// model.BlockerCategory(code). Each canonical carrier failure class must map to a
// bounded category label — not the raw code or any id.
func TestFulfillmentService_CarrierBlockerCategory_Bounded(t *testing.T) {
	failureClasses := []string{
		model.CarrierFailMissingData,
		model.CarrierFailProviderRejection,
		model.CarrierFailProviderOutage,
		model.CarrierFailRateLimit,
		model.CarrierFailAuth,
		"unknown-class",
	}
	allowedCategories := map[string]struct{}{
		"integration": {}, "supplier": {}, "operator": {}, "capability": {}, "mapping": {},
	}
	m := obsmetrics.NewFulfillmentMetrics()
	for _, fc := range failureClasses {
		code := model.CarrierFailureBlockerCode(fc)
		category := model.BlockerCategory(code)
		require.Containsf(t, allowedCategories, category,
			"carrier failure %q -> code %q -> category %q must be a bounded category", fc, code, category)
		m.RecordBlocker(category)
	}

	out := renderFulfillmentMetrics(m)
	assert.Contains(t, out, "# TYPE openoms_fulfillment_blockers_total counter")
	// Every carrier failure class maps to capability/operator categories — assert the
	// rendered series use only the bounded category label.
	assert.Contains(t, out, `openoms_fulfillment_blockers_total{category="capability"}`)
}

// The unit/step transition statuses the service records (FulfillmentStatus enum)
// must render as bounded status-label series.
func TestFulfillmentService_UnitStepTransitionMetrics_Bounded(t *testing.T) {
	m := obsmetrics.NewFulfillmentMetrics()
	m.RecordUnitTransition(model.FulfillmentStatusRunning)
	m.RecordStepTransition(model.FulfillmentStatusSucceeded)

	out := renderFulfillmentMetrics(m)
	assert.Contains(t, out, `openoms_fulfillment_unit_transitions_total{status="running"} 1`)
	assert.Contains(t, out, `openoms_fulfillment_step_transitions_total{status="succeeded"} 1`)
}

// Cardinality guard at the service boundary: even when the service is handed an
// unbounded identifier in place of a bounded value (a calling bug), the rendered
// exposition must NOT gain a new id-keyed series — it is coerced to "other".
func TestFulfillmentService_RecordingNeverLeaksIdentifiers(t *testing.T) {
	m := obsmetrics.NewFulfillmentMetrics()
	m.RecordProviderAttempt("11111111-2222-3333-4444-555555555555", "weird")
	m.RecordBlocker("c0ffeec0-ffee-c0ff-eec0-ffeec0ffeec0")
	m.RecordUnitTransition("99999999-aaaa-bbbb-cccc-dddddddddddd")

	out := renderFulfillmentMetrics(m)
	assert.NotContains(t, out, "11111111-2222-3333-4444-555555555555")
	assert.NotContains(t, out, "c0ffeec0-ffee-c0ff-eec0-ffeec0ffeec0")
	assert.NotContains(t, out, "99999999-aaaa-bbbb-cccc-dddddddddddd")
	assert.Contains(t, out, `operation="other"`)
	assert.Contains(t, out, `category="other"`)
}
