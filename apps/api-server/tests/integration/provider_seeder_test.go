//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// TestProviderRegistrySeeder_SeedsAndIsIdempotent runs the OPE-412 seeder
// against a real database and verifies it creates draft definitions with
// schema + capability stubs, and is idempotent on a second run.
func TestProviderRegistrySeeder_SeedsAndIsIdempotent(t *testing.T) {
	ctx := context.Background()
	reg := newRegistryService()
	seeder := service.NewProviderRegistrySeeder(reg)
	catalog := service.ProviderRegistryCatalog()
	require.NotEmpty(t, catalog)

	t.Cleanup(func() {
		for _, e := range catalog {
			if d, err := reg.GetDefinitionByKey(context.Background(), e.ProviderKey); err == nil {
				_, _ = superPool.Exec(context.Background(), "DELETE FROM provider_definitions WHERE id = $1", d.ID)
			}
		}
	})

	res, err := seeder.Seed(ctx, catalog)
	require.NoError(t, err)
	assert.Len(t, res.Created, len(catalog), "first run creates every catalog provider")
	assert.Empty(t, res.Skipped)

	// allegro: definition + version + schema (secret client_id) + unknown caps.
	def, err := reg.GetDefinitionByKey(ctx, "allegro")
	require.NoError(t, err)
	assert.Equal(t, model.ProviderTypeMarketplace, def.ProviderType)

	versions, err := reg.ListVersions(ctx, def.ID)
	require.NoError(t, err)
	require.Len(t, versions, 1)
	assert.Equal(t, model.ProviderStateResearch, versions[0].PublicationState)

	schema, err := reg.GetSchema(ctx, versions[0].ID)
	require.NoError(t, err)
	var sawSecret bool
	for _, g := range schema.Groups {
		for _, f := range g.Fields {
			if f.Key == "client_id" {
				assert.True(t, f.Secret)
				sawSecret = true
			}
		}
	}
	assert.True(t, sawSecret, "allegro schema should carry the client_id secret field")

	caps, err := reg.GetCapabilities(ctx, versions[0].ID)
	require.NoError(t, err)
	require.NotEmpty(t, caps)
	for _, c := range caps {
		assert.Equal(t, model.SupportStatusUnknown, c.SupportStatus)
	}

	// Idempotent: a second run creates nothing.
	res2, err := seeder.Seed(ctx, catalog)
	require.NoError(t, err)
	assert.Empty(t, res2.Created, "second run creates nothing")
	assert.Len(t, res2.Skipped, len(catalog))
}
