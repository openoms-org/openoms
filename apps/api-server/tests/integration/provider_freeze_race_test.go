//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// newFreezeRaceVersion creates a definition plus a version advanced to
// internal_validation — the last state before the publish freeze kicks in.
func newFreezeRaceVersion(t *testing.T, ctx context.Context, reg *service.ProviderRegistryService) *model.ProviderVersion {
	t.Helper()
	def, err := reg.CreateDefinition(ctx, service.CreateProviderDefinitionInput{
		ProviderKey: "race-itest-" + uuid.New().String()[:8], DisplayName: "Freeze Race ITest", ProviderType: model.ProviderTypeSupplier,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = superPool.Exec(context.Background(), "DELETE FROM provider_definitions WHERE id = $1", def.ID)
	})
	ver, err := reg.CreateVersion(ctx, def.ID, "1.0.0", "", "", nil)
	require.NoError(t, err)
	for _, to := range []string{
		model.ProviderStateDesigned, model.ProviderStateAdapterInProgress, model.ProviderStateInternalValidation,
	} {
		_, err := reg.Transition(ctx, ver.ID, to, nil, "advance")
		require.NoError(t, err)
	}
	return ver
}

// TestProviderFreeze_MutationRacesWithPublish proves the published-version
// freeze holds under concurrency (OPE-515): a mutation whose fast pre-check
// passed while the version was still mutable must re-check the state under the
// definition-row lock and be rejected once a concurrent transition publishes
// the version. The in-flight Transition(-> private_beta) is emulated by an open
// transaction holding the same definition-row lock the lifecycle code takes.
func TestProviderFreeze_MutationRacesWithPublish(t *testing.T) {
	reg := newRegistryService()
	val := newValidationService()

	cases := []struct {
		name   string
		mutate func(ctx context.Context, versionID uuid.UUID) error
	}{
		{"SetCapabilities", func(ctx context.Context, id uuid.UUID) error {
			_, err := reg.SetCapabilities(ctx, id, []model.ProviderCapability{
				{CapabilityKey: "supplier.order.create", SupportStatus: model.SupportStatusSupported},
			})
			return err
		}},
		{"SetStatusMappings", func(ctx context.Context, id uuid.UUID) error {
			_, err := reg.SetStatusMappings(ctx, id, []model.ProviderStatusMapping{
				{StatusDomain: model.StatusDomainOrder, RawStatus: "NEW", CanonicalStatus: "new", Confidence: model.MappingConfidenceHigh},
			})
			return err
		}},
		{"SetSchema", func(ctx context.Context, id uuid.UUID) error {
			_, err := reg.SetSchema(ctx, id, []model.ProviderFieldGroup{
				{Key: model.FieldGroupSettings, Label: "Settings", Fields: []model.ProviderField{
					{Key: "region", Label: "Region", Type: model.FieldTypeString},
				}},
			})
			return err
		}},
		{"UpdateVersionMetadata", func(ctx context.Context, id uuid.UUID) error {
			_, err := reg.UpdateVersionMetadata(ctx, id, "tampered", "")
			return err
		}},
		{"SetProbes", func(ctx context.Context, id uuid.UUID) error {
			_, err := val.SetProbes(ctx, id, []model.ProviderValidationProbe{
				{Label: "auth", ProbeType: model.ProbeAuthCheck},
			})
			return err
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			ver := newFreezeRaceVersion(t, ctx, reg)

			// Emulate an in-flight Transition(-> private_beta): take the
			// definition-row lock and write the new state without committing yet.
			tx, err := appPool.Begin(ctx)
			require.NoError(t, err)
			defer func() { _ = tx.Rollback(context.Background()) }()
			_, err = tx.Exec(ctx, `SELECT 1 FROM provider_definitions WHERE id = $1 FOR UPDATE`, ver.ProviderDefinitionID)
			require.NoError(t, err)
			_, err = tx.Exec(ctx, `UPDATE provider_versions SET publication_state = $2 WHERE id = $1`, ver.ID, model.ProviderStatePrivateBeta)
			require.NoError(t, err)

			errCh := make(chan error, 1)
			go func() { errCh <- tc.mutate(ctx, ver.ID) }()

			// The mutation must serialize behind the definition lock, not land.
			select {
			case got := <-errCh:
				t.Fatalf("mutation finished while the publish transaction held the definition lock (err=%v) — frozen check is not serialized with Transition", got)
			case <-time.After(250 * time.Millisecond):
			}

			require.NoError(t, tx.Commit(ctx))
			require.ErrorIs(t, <-errCh, service.ErrProviderVersionFrozen,
				"mutation racing a publish must hit the authoritative frozen check")
		})
	}
}
