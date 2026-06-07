package service

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/obsmetrics"
)

// OPE-422: ADDITIVE, best-effort metric wiring on the provider validation +
// registry services. CompleteRun / Transition run inside a DB transaction, so the
// pure assertions here cover the bounded values the services feed the collector
// (validation verdict enum, publication-state enum) and the nil-safe setters.

func TestProviderValidationService_WithMetrics_Chainable(t *testing.T) {
	m := obsmetrics.NewFulfillmentMetrics()
	svc := NewProviderValidationService(nil, nil, nil, nil).WithMetrics(m)
	require.NotNil(t, svc)
	assert.Same(t, m, svc.metrics)
}

func TestProviderRegistryService_WithMetrics_Chainable(t *testing.T) {
	m := obsmetrics.NewFulfillmentMetrics()
	svc := NewProviderRegistryService(nil, nil, nil, nil, nil, nil).WithMetrics(m)
	require.NotNil(t, svc)
	assert.Same(t, m, svc.metrics)
}

func TestProviderServices_WithMetrics_NilReceiverSafe(t *testing.T) {
	var val *ProviderValidationService
	var reg *ProviderRegistryService
	assert.NotPanics(t, func() {
		assert.Nil(t, val.WithMetrics(obsmetrics.NewFulfillmentMetrics()))
		assert.Nil(t, reg.WithMetrics(obsmetrics.NewFulfillmentMetrics()))
	})
}

// Every validation run verdict the service can compute is a bounded enum that
// renders as a bounded result-label series.
func TestProviderValidation_VerdictMetric_Bounded(t *testing.T) {
	verdicts := []string{model.RunVerdictPassed, model.RunVerdictFailed, model.RunVerdictError, model.RunVerdictPending}
	m := obsmetrics.NewFulfillmentMetrics()
	for _, v := range verdicts {
		m.RecordValidationRun(v)
	}
	m.RecordValidationFailure()
	m.RecordValidationFailure()

	var b strings.Builder
	m.Render(&b)
	out := b.String()
	assert.Contains(t, out, "# TYPE openoms_provider_validation_runs_total counter")
	for _, v := range verdicts {
		assert.Containsf(t, out, `openoms_provider_validation_runs_total{result="`+v+`"} 1`,
			"verdict %q must render as a bounded result label", v)
	}
	assert.Contains(t, out, "openoms_provider_validation_failures_total 2")
}

// Every publication state a Transition can move to is a bounded enum that renders as
// a bounded state-label series — never a version id.
func TestProviderRegistry_PublicationTransitionMetric_Bounded(t *testing.T) {
	states := []string{
		model.ProviderStateResearch, model.ProviderStateDesigned, model.ProviderStateAdapterInProgress,
		model.ProviderStateInternalValidation, model.ProviderStatePrivateBeta, model.ProviderStateAvailable,
		model.ProviderStateDeprecated, model.ProviderStateRetired,
	}
	m := obsmetrics.NewFulfillmentMetrics()
	for _, st := range states {
		m.RecordPublicationTransition(st)
	}

	var b strings.Builder
	m.Render(&b)
	out := b.String()
	assert.Contains(t, out, "# TYPE openoms_provider_publication_transitions_total counter")
	for _, st := range states {
		assert.Containsf(t, out, `openoms_provider_publication_transitions_total{state="`+st+`"} 1`,
			"publication state %q must render as a bounded state label", st)
	}
	// Cardinality guard: no id-like label keys appear.
	assert.NotContains(t, out, "version_id")
	assert.NotContains(t, out, "run_id")
}
