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

func newRegistryService() *service.ProviderRegistryService {
	return service.NewProviderRegistryService(
		appPool,
		repository.NewProviderDefinitionRepository(appPool),
		repository.NewProviderVersionRepository(appPool),
		repository.NewProviderPublicationRepository(appPool),
		repository.NewProviderSchemaRepository(appPool),
	)
}

// TestProviderSchema_SetGetFreeze exercises the field-schema contract against a
// real database: set a valid schema, read it back, then confirm it is frozen
// once the version is published and that invalid schemas are rejected.
func TestProviderSchema_SetGetFreeze(t *testing.T) {
	ctx := context.Background()
	svc := newRegistryService()

	def, err := svc.CreateDefinition(ctx, service.CreateProviderDefinitionInput{
		ProviderKey: "schema-itest-" + uuid.New().String()[:8], DisplayName: "Schema ITest", ProviderType: model.ProviderTypeSupplier,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = superPool.Exec(context.Background(), "DELETE FROM provider_definitions WHERE id = $1", def.ID)
	})

	ver, err := svc.CreateVersion(ctx, def.ID, "1.0.0", "", "", nil)
	require.NoError(t, err)

	// Empty schema before any set.
	empty, err := svc.GetSchema(ctx, ver.ID)
	require.NoError(t, err)
	assert.Empty(t, empty.Groups)

	groups := []model.ProviderFieldGroup{
		{Key: model.FieldGroupSecretCredentials, Label: "Credentials", Fields: []model.ProviderField{
			{Key: "api_key", Label: "API Key", Type: model.FieldTypePassword, Required: true, Secret: true, TestConnectionDependency: true},
		}},
		{Key: model.FieldGroupSettings, Label: "Settings", Fields: []model.ProviderField{
			{Key: "region", Label: "Region", Type: model.FieldTypeEnum, Validation: model.ProviderFieldValidation{Enum: []string{"PL", "DE"}}},
		}},
	}
	saved, err := svc.SetSchema(ctx, ver.ID, groups)
	require.NoError(t, err)
	require.Len(t, saved.Groups, 2)

	got, err := svc.GetSchema(ctx, ver.ID)
	require.NoError(t, err)
	require.Len(t, got.Groups, 2)
	assert.Equal(t, "api_key", got.Groups[0].Fields[0].Key)
	assert.True(t, got.Groups[0].Fields[0].Secret)
	assert.Equal(t, []string{"PL", "DE"}, got.Groups[1].Fields[0].Validation.Enum)

	// Invalid schema (duplicate field key) is rejected.
	_, err = svc.SetSchema(ctx, ver.ID, []model.ProviderFieldGroup{
		{Key: model.FieldGroupSettings, Fields: []model.ProviderField{
			{Key: "dup", Label: "A", Type: model.FieldTypeString},
			{Key: "dup", Label: "B", Type: model.FieldTypeString},
		}},
	})
	require.ErrorIs(t, err, service.ErrInvalidFieldSchema)

	// Walk to private_beta, then the schema is frozen.
	for _, to := range []string{
		model.ProviderStateDesigned, model.ProviderStateAdapterInProgress,
		model.ProviderStateInternalValidation, model.ProviderStatePrivateBeta,
	} {
		_, err := svc.Transition(ctx, ver.ID, to, nil, "advance")
		require.NoError(t, err)
	}
	_, err = svc.SetSchema(ctx, ver.ID, groups)
	require.ErrorIs(t, err, service.ErrProviderVersionFrozen)
}
