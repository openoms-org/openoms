//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// newPendingRun creates a definition, version and pending validation run with a
// single recorded "auth" result in the given status.
func newPendingRun(t *testing.T, ctx context.Context, val *service.ProviderValidationService, seedStatus string) *model.ProviderValidationRun {
	t.Helper()
	reg := newRegistryService()
	def, err := reg.CreateDefinition(ctx, service.CreateProviderDefinitionInput{
		ProviderKey: "valrace-itest-" + uuid.New().String()[:8], DisplayName: "Validation Race ITest", ProviderType: model.ProviderTypeSupplier,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = superPool.Exec(context.Background(), "DELETE FROM provider_definitions WHERE id = $1", def.ID)
	})
	ver, err := reg.CreateVersion(ctx, def.ID, "1.0.0", "", "", nil)
	require.NoError(t, err)
	run, err := val.StartRun(ctx, ver.ID, model.ValidationEnvSandbox, false, nil)
	require.NoError(t, err)
	_, err = val.RecordResult(ctx, run.ID, model.ProbeAuthCheck, "auth", seedStatus, "seed", "", "")
	require.NoError(t, err)
	return run
}

// TestProviderValidation_FinalizedRunWriteRace proves run finalization is
// atomic (OPE-524): a write whose unlocked pre-check passed while the run was
// still pending must serialize behind the run-row lock and be rejected once a
// concurrent CompleteRun finalizes the run. The in-flight CompleteRun is
// emulated by an open transaction holding the run-row lock the service takes.
func TestProviderValidation_FinalizedRunWriteRace(t *testing.T) {
	val := newValidationService()

	cases := []struct {
		name       string
		seedStatus string
		write      func(ctx context.Context, runID uuid.UUID) error
	}{
		// Seed a FAILED result so a late CompleteRun would compute a different
		// verdict ("failed") than the concurrent finalize ("passed") — an
		// overwrite is therefore observable.
		{"CompleteRun", model.ResultStatusFailed, func(ctx context.Context, runID uuid.UUID) error {
			_, err := val.CompleteRun(ctx, runID)
			return err
		}},
		// Seed a PASSED result and try to flip it after the run is finalized.
		{"RecordResult", model.ResultStatusPassed, func(ctx context.Context, runID uuid.UUID) error {
			_, err := val.RecordResult(ctx, runID, model.ProbeAuthCheck, "auth", model.ResultStatusFailed, "late", "", "")
			return err
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			run := newPendingRun(t, ctx, val, tc.seedStatus)

			// Emulate an in-flight CompleteRun: lock the run row and finalize
			// it without committing yet.
			tx, err := appPool.Begin(ctx)
			require.NoError(t, err)
			defer func() { _ = tx.Rollback(context.Background()) }()
			_, err = tx.Exec(ctx, `SELECT 1 FROM provider_validation_runs WHERE id = $1 FOR UPDATE`, run.ID)
			require.NoError(t, err)
			_, err = tx.Exec(ctx, `UPDATE provider_validation_runs SET verdict = $2, finished_at = now() WHERE id = $1`, run.ID, model.RunVerdictPassed)
			require.NoError(t, err)

			errCh := make(chan error, 1)
			go func() { errCh <- tc.write(ctx, run.ID) }()

			// The write must serialize behind the run lock, not land.
			select {
			case got := <-errCh:
				t.Fatalf("write finished while the finalize transaction held the run lock (err=%v) — finalized check is not serialized", got)
			case <-time.After(250 * time.Millisecond):
			}

			require.NoError(t, tx.Commit(ctx))
			require.ErrorIs(t, <-errCh, service.ErrValidationRunFinalized,
				"write racing a finalize must hit the authoritative finalized check")

			// The concurrent finalize stands untouched: verdict, results, gaps.
			final, err := val.GetRunWithResults(ctx, run.ID)
			require.NoError(t, err)
			assert.Equal(t, model.RunVerdictPassed, final.Verdict)
			require.NotNil(t, final.FinishedAt)
			require.Len(t, final.Results, 1)
			assert.Equal(t, tc.seedStatus, final.Results[0].Status)
			assert.Equal(t, "seed", final.Results[0].Observation)
			gaps, err := newRegistryService().ListGaps(ctx, run.ProviderVersionID)
			require.NoError(t, err)
			assert.Empty(t, gaps, "no gaps may be created by a rejected late CompleteRun")
		})
	}
}

// TestProviderValidationRepo_FinalizedRunSQLGuards proves the SQL-level guards
// (defense in depth below the service lock): writes against an already
// finalized run match zero rows and surface pgx.ErrNoRows.
func TestProviderValidationRepo_FinalizedRunSQLGuards(t *testing.T) {
	ctx := context.Background()
	val := newValidationService()
	run := newPendingRun(t, ctx, val, model.ResultStatusPassed)

	_, err := superPool.Exec(ctx,
		`UPDATE provider_validation_runs SET verdict = $2, finished_at = now() WHERE id = $1`,
		run.ID, model.RunVerdictPassed)
	require.NoError(t, err)

	repo := repository.NewProviderValidationRepository(appPool)
	err = repo.FinalizeRun(ctx, appPool, run.ID, model.RunVerdictFailed, time.Now().UTC())
	require.ErrorIs(t, err, pgx.ErrNoRows, "finalizing a finalized run must match zero rows")

	_, err = repo.UpsertResult(ctx, appPool, model.ProviderValidationResult{
		RunID: run.ID, ProbeType: model.ProbeAuthCheck, Label: "auth", Status: model.ResultStatusFailed,
	})
	require.ErrorIs(t, err, pgx.ErrNoRows, "recording a result on a finalized run must match zero rows")

	final, err := val.GetRunWithResults(ctx, run.ID)
	require.NoError(t, err)
	assert.Equal(t, model.RunVerdictPassed, final.Verdict)
	require.Len(t, final.Results, 1)
	assert.Equal(t, model.ResultStatusPassed, final.Results[0].Status)
}
