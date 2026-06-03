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

// TestProviderRegistry_LifecycleEndToEnd exercises the registry + lifecycle
// state machine against a real database via the RLS-scoped app pool (proving
// the migration grants + that the platform tables are reachable with no RLS).
func TestProviderRegistry_LifecycleEndToEnd(t *testing.T) {
	ctx := context.Background()
	svc := service.NewProviderRegistryService(
		appPool,
		repository.NewProviderDefinitionRepository(appPool),
		repository.NewProviderVersionRepository(appPool),
		repository.NewProviderPublicationRepository(appPool),
	)

	def, err := svc.CreateDefinition(ctx, service.CreateProviderDefinitionInput{
		ProviderKey:     "itest-" + uuid.New().String()[:8],
		DisplayName:     "Integration Test Provider",
		ProviderType:    model.ProviderTypeSupplier,
		Regions:         []string{"PL", "DE"},
		BusinessDomains: []string{"supplier"},
		Owner:           "ops@example.test",
		SourceLinks:     []string{"https://example.test/docs"},
	})
	require.NoError(t, err)
	assert.Equal(t, model.ProviderTypeSupplier, def.ProviderType)
	assert.Equal(t, []string{"PL", "DE"}, def.Regions)
	assert.Equal(t, []string{"https://example.test/docs"}, def.SourceLinks)
	t.Cleanup(func() {
		// Cascades to versions -> publication_events + tenant_enables.
		_, _ = superPool.Exec(context.Background(), "DELETE FROM provider_definitions WHERE id = $1", def.ID)
	})

	invalidType, err := svc.CreateDefinition(ctx, service.CreateProviderDefinitionInput{
		ProviderKey: "bad", DisplayName: "Bad", ProviderType: "banana",
	})
	require.Nil(t, invalidType)
	require.ErrorIs(t, err, service.ErrInvalidProviderType)

	// New version starts in research with a 'create' event.
	ver, err := svc.CreateVersion(ctx, def.ID, "1.0.0", "initial", "", nil)
	require.NoError(t, err)
	assert.Equal(t, model.ProviderStateResearch, ver.PublicationState)
	events, err := svc.ListPublicationEvents(ctx, ver.ID)
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, model.ProviderEventCreate, events[0].Action)

	// Walk the happy path to 'available'.
	for _, to := range []string{
		model.ProviderStateDesigned, model.ProviderStateAdapterInProgress,
		model.ProviderStateInternalValidation, model.ProviderStatePrivateBeta, model.ProviderStateAvailable,
	} {
		v, err := svc.Transition(ctx, ver.ID, to, nil, "advance")
		require.NoErrorf(t, err, "transition to %s", to)
		assert.Equal(t, to, v.PublicationState)
	}

	// Becoming available sets published_at and the definition's latest pointer.
	avail, err := svc.GetVersion(ctx, ver.ID)
	require.NoError(t, err)
	require.NotNil(t, avail.PublishedAt)
	d, err := svc.GetDefinition(ctx, def.ID)
	require.NoError(t, err)
	require.NotNil(t, d.LatestPublishedVersionID)
	assert.Equal(t, ver.ID, *d.LatestPublishedVersionID)

	// create + 5 transitions = 6 events.
	events, err = svc.ListPublicationEvents(ctx, ver.ID)
	require.NoError(t, err)
	assert.Len(t, events, 6)

	// Frozen: an available version's metadata cannot be edited.
	_, err = svc.UpdateVersionMetadata(ctx, ver.ID, "tampered", "")
	require.ErrorIs(t, err, service.ErrProviderVersionFrozen)

	// Illegal transition: available -> research is not an edge.
	_, err = svc.Transition(ctx, ver.ID, model.ProviderStateResearch, nil, "nope")
	require.ErrorIs(t, err, service.ErrIllegalProviderTransition)

	// Private-beta tenant enablement (valid while available).
	tenantID := seedTenant(t, ctx)
	enable, err := svc.EnableTenant(ctx, ver.ID, tenantID, nil)
	require.NoError(t, err)
	assert.Equal(t, tenantID, enable.TenantID)
	// Idempotent.
	_, err = svc.EnableTenant(ctx, ver.ID, tenantID, nil)
	require.NoError(t, err)

	// Emergency disable pulls it back to internal_validation and clears the
	// definition's latest pointer (no other available version).
	disabled, err := svc.EmergencyDisable(ctx, ver.ID, nil, "regression")
	require.NoError(t, err)
	assert.Equal(t, model.ProviderStateInternalValidation, disabled.PublicationState)
	d, err = svc.GetDefinition(ctx, def.ID)
	require.NoError(t, err)
	assert.Nil(t, d.LatestPublishedVersionID, "latest pointer must be cleared after emergency disable")

	events, err = svc.ListPublicationEvents(ctx, ver.ID)
	require.NoError(t, err)
	assert.Equal(t, model.ProviderEventEmergencyDisable, events[0].Action, "newest event is the emergency disable")
}
