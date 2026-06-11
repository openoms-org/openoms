package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

func TestProviderCatalogIncludesApaczka(t *testing.T) {
	catalog := ProviderRegistryCatalog()

	var found *ProviderCatalogEntry
	for i := range catalog {
		if catalog[i].ProviderKey == "apaczka" {
			found = &catalog[i]
			break
		}
	}
	require.NotNilf(t, found, "apaczka entry not found in ProviderRegistryCatalog")
	assert.Equal(t, model.ProviderTypeCarrier, found.ProviderType)
	assert.Greater(t, len(found.Schema), 0, "Schema must not be empty")
	assert.Greater(t, len(found.Capabilities), 0, "Capabilities must not be empty")
}

func TestProviderRegistryCatalog_Valid(t *testing.T) {
	catalog := ProviderRegistryCatalog()
	require.NotEmpty(t, catalog)

	seen := map[string]bool{}
	for _, e := range catalog {
		t.Run(e.ProviderKey, func(t *testing.T) {
			assert.NotEmpty(t, e.ProviderKey)
			assert.NotEmpty(t, e.Version)
			assert.False(t, seen[e.ProviderKey], "duplicate provider key")
			seen[e.ProviderKey] = true

			assert.Truef(t, model.IsValidProviderType(e.ProviderType), "invalid provider_type %q", e.ProviderType)
			assert.NoError(t, model.ValidateFieldSchema(e.Schema), "schema must be structurally valid")
			assert.NoError(t, model.ValidateCapabilities(e.Capabilities), "capabilities must be structurally valid")

			for _, c := range e.Capabilities {
				assert.Equalf(t, model.SupportStatusUnknown, c.SupportStatus,
					"capability %q must be seeded as unknown (verified later via the Studio)", c.CapabilityKey)
			}
		})
	}
}
