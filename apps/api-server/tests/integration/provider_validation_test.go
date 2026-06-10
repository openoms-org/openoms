//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

func newValidationService() *service.ProviderValidationService {
	return service.NewProviderValidationService(
		appPool,
		repository.NewProviderVersionRepository(appPool),
		repository.NewProviderCapabilityRepository(appPool),
		repository.NewProviderValidationRepository(appPool),
	)
}

// TestProviderValidation_RunLifecycleAndGap exercises the validation engine end
// to end against a real database: probe set, destructive gate, immutable run,
// verdict, and auto-creation of an integration gap from a failed probe.
func TestProviderValidation_RunLifecycleAndGap(t *testing.T) {
	ctx := context.Background()
	reg := newRegistryService()
	val := newValidationService()

	def, err := reg.CreateDefinition(ctx, service.CreateProviderDefinitionInput{
		ProviderKey: "val-itest-" + uuid.New().String()[:8], DisplayName: "Validation ITest", ProviderType: model.ProviderTypeSupplier,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = superPool.Exec(context.Background(), "DELETE FROM provider_definitions WHERE id = $1", def.ID)
	})

	ver, err := reg.CreateVersion(ctx, def.ID, "1.0.0", "", "", nil)
	require.NoError(t, err)

	// Declare probes including a destructive one.
	_, err = val.SetProbes(ctx, ver.ID, []model.ProviderValidationProbe{
		{Label: "auth", ProbeType: model.ProbeAuthCheck},
		{Label: "create", ProbeType: model.ProbeSandboxOrderCreate, Destructive: true},
	})
	require.NoError(t, err)

	// Destructive gate: must confirm.
	_, err = val.StartRun(ctx, ver.ID, model.ValidationEnvSandbox, false, nil)
	require.ErrorIs(t, err, service.ErrDestructiveProbeNotConfirmed)
	// Bad environment.
	_, err = val.StartRun(ctx, ver.ID, "staging", true, nil)
	require.ErrorIs(t, err, service.ErrInvalidValidationEnv)

	run, err := val.StartRun(ctx, ver.ID, model.ValidationEnvSandbox, true, nil)
	require.NoError(t, err)
	assert.Equal(t, model.RunVerdictPending, run.Verdict)

	// Record one passing and one failing (auth) result.
	_, err = val.RecordResult(ctx, run.ID, model.ProbeSandboxOrderCreate, "create", model.ResultStatusPassed, "ok", model.HashPayload([]byte("{}")), "")
	require.NoError(t, err)
	_, err = val.RecordResult(ctx, run.ID, model.ProbeAuthCheck, "auth", model.ResultStatusFailed, "401", "", "invalid key")
	require.NoError(t, err)
	// Invalid status rejected.
	_, err = val.RecordResult(ctx, run.ID, model.ProbeAuthCheck, "auth", "exploded", "", "", "")
	require.ErrorIs(t, err, service.ErrInvalidResultStatus)

	// Complete: verdict failed, finished, and an auth_failure gap auto-created.
	done, err := val.CompleteRun(ctx, run.ID)
	require.NoError(t, err)
	assert.Equal(t, model.RunVerdictFailed, done.Verdict)
	require.NotNil(t, done.FinishedAt)
	require.Len(t, done.Results, 2)

	gaps, err := reg.ListGaps(ctx, ver.ID)
	require.NoError(t, err)
	require.Len(t, gaps, 1, "a gap should be auto-created from the failed auth probe")
	assert.Equal(t, model.GapTypeAuthFailure, gaps[0].GapType)
	assert.Equal(t, model.GapSeverityActionRequired, gaps[0].Severity)

	// Run is now immutable.
	_, err = val.RecordResult(ctx, run.ID, model.ProbeAuthCheck, "auth", model.ResultStatusPassed, "", "", "")
	require.ErrorIs(t, err, service.ErrValidationRunFinalized)
	_, err = val.CompleteRun(ctx, run.ID)
	require.ErrorIs(t, err, service.ErrValidationRunFinalized)

	// The rejected writes left the finalized run untouched: verdict, results
	// and the auto-created gap are unchanged.
	unchanged, err := val.GetRunWithResults(ctx, run.ID)
	require.NoError(t, err)
	assert.Equal(t, model.RunVerdictFailed, unchanged.Verdict)
	assert.Equal(t, done.FinishedAt, unchanged.FinishedAt)
	require.Len(t, unchanged.Results, 2)
	for _, res := range unchanged.Results {
		if res.Label == "auth" {
			assert.Equal(t, model.ResultStatusFailed, res.Status, "rejected RecordResult must not change a finalized result")
		}
	}
	gaps, err = reg.ListGaps(ctx, ver.ID)
	require.NoError(t, err)
	assert.Len(t, gaps, 1, "rejected CompleteRun must not create duplicate gaps")

	// Probes frozen once the version is published.
	for _, to := range []string{
		model.ProviderStateDesigned, model.ProviderStateAdapterInProgress,
		model.ProviderStateInternalValidation, model.ProviderStatePrivateBeta,
	} {
		_, err := reg.Transition(ctx, ver.ID, to, nil, "advance")
		require.NoError(t, err)
	}
	_, err = val.SetProbes(ctx, ver.ID, []model.ProviderValidationProbe{{Label: "x", ProbeType: model.ProbeAuthCheck}})
	require.ErrorIs(t, err, service.ErrProviderVersionFrozen)
}
