//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// TestProviderCapabilities_SetMapGapFreeze exercises capability profiles, status
// mappings and integration gaps against a real database via the app pool.
func TestProviderCapabilities_SetMapGapFreeze(t *testing.T) {
	ctx := context.Background()
	svc := newRegistryService()

	def, err := svc.CreateDefinition(ctx, service.CreateProviderDefinitionInput{
		ProviderKey: "cap-itest-" + uuid.New().String()[:8], DisplayName: "Cap ITest", ProviderType: model.ProviderTypeSupplier,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = superPool.Exec(context.Background(), "DELETE FROM provider_definitions WHERE id = $1", def.ID)
	})

	ver, err := svc.CreateVersion(ctx, def.ID, "1.0.0", "", "", nil)
	require.NoError(t, err)

	// Capabilities: set + read (replace semantics).
	caps, err := svc.SetCapabilities(ctx, ver.ID, []model.ProviderCapability{
		{CapabilityKey: "supplier.order.create", SupportStatus: model.SupportStatusSupported, Channel: "api"},
		{CapabilityKey: "supplier.tracking.read", SupportStatus: model.SupportStatusRequiresManual},
	})
	require.NoError(t, err)
	require.Len(t, caps, 2)

	got, err := svc.GetCapabilities(ctx, ver.ID)
	require.NoError(t, err)
	require.Len(t, got, 2)
	// Replace with a smaller set.
	caps, err = svc.SetCapabilities(ctx, ver.ID, []model.ProviderCapability{
		{CapabilityKey: "supplier.order.create", SupportStatus: model.SupportStatusDegraded},
	})
	require.NoError(t, err)
	require.Len(t, caps, 1)
	assert.Equal(t, model.SupportStatusDegraded, caps[0].SupportStatus)

	// Invalid capability rejected.
	_, err = svc.SetCapabilities(ctx, ver.ID, []model.ProviderCapability{{CapabilityKey: "x", SupportStatus: "maybe"}})
	require.ErrorIs(t, err, service.ErrInvalidCapability)

	// Status mappings: set + read; an unmapped raw status (empty canonical) is allowed.
	maps, err := svc.SetStatusMappings(ctx, ver.ID, []model.ProviderStatusMapping{
		{StatusDomain: model.StatusDomainOrder, RawStatus: "NEW", CanonicalStatus: "new", Confidence: model.MappingConfidenceHigh},
		{StatusDomain: model.StatusDomainShipment, RawStatus: "SENT", Confidence: model.MappingConfidenceMedium, IsTerminal: true},
	})
	require.NoError(t, err)
	require.Len(t, maps, 2)
	_, err = svc.SetStatusMappings(ctx, ver.ID, []model.ProviderStatusMapping{{StatusDomain: "galaxy", RawStatus: "X", Confidence: model.MappingConfidenceLow}})
	require.ErrorIs(t, err, service.ErrInvalidStatusMapping)

	// Gaps: create, list, resolve.
	gap, err := svc.CreateGap(ctx, ver.ID, model.GapTypeMissingStatusMapping, model.GapSeverityActionRequired, "raw status UNKNOWN not mapped")
	require.NoError(t, err)
	assert.Equal(t, model.GapStatusOpen, gap.Status)
	gaps, err := svc.ListGaps(ctx, ver.ID)
	require.NoError(t, err)
	require.Len(t, gaps, 1)
	resolved, err := svc.UpdateGapStatus(ctx, gap.ID, model.GapStatusResolved)
	require.NoError(t, err)
	assert.Equal(t, model.GapStatusResolved, resolved.Status)
	require.NotNil(t, resolved.ResolvedAt)

	_, err = svc.CreateGap(ctx, ver.ID, "bogus_type", model.GapSeverityInfo, "")
	require.ErrorIs(t, err, service.ErrInvalidGap)
	_, err = svc.UpdateGapStatus(ctx, uuid.New(), model.GapStatusResolved)
	require.ErrorIs(t, err, service.ErrProviderGapNotFound)

	// Frozen once published.
	for _, to := range []string{
		model.ProviderStateDesigned, model.ProviderStateAdapterInProgress,
		model.ProviderStateInternalValidation, model.ProviderStatePrivateBeta,
	} {
		_, err := svc.Transition(ctx, ver.ID, to, nil, "advance")
		require.NoError(t, err)
	}
	_, err = svc.SetCapabilities(ctx, ver.ID, []model.ProviderCapability{{CapabilityKey: "x", SupportStatus: model.SupportStatusSupported}})
	require.ErrorIs(t, err, service.ErrProviderVersionFrozen)
	_, err = svc.SetStatusMappings(ctx, ver.ID, []model.ProviderStatusMapping{{StatusDomain: model.StatusDomainOrder, RawStatus: "Y", Confidence: model.MappingConfidenceHigh}})
	require.ErrorIs(t, err, service.ErrProviderVersionFrozen)
}
