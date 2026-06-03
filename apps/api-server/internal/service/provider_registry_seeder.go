package service

import (
	"context"
	"errors"
	"fmt"
)

// SeedResult reports which providers a seed run created vs skipped (already present).
type SeedResult struct {
	Created []string
	Skipped []string
}

// ProviderRegistrySeeder seeds the provider registry with draft definitions for
// existing providers (OPE-412). It is idempotent: providers already present (by
// provider_key) are skipped, so it is safe to run repeatedly.
type ProviderRegistrySeeder struct {
	registry *ProviderRegistryService
}

// NewProviderRegistrySeeder creates a seeder backed by the registry service.
func NewProviderRegistrySeeder(registry *ProviderRegistryService) *ProviderRegistrySeeder {
	return &ProviderRegistrySeeder{registry: registry}
}

// Seed creates a draft definition + research version + credential schema +
// capability stubs for each catalog entry that does not already exist. It does
// not modify existing definitions (forward-only, additive — tenant integrations
// are untouched).
func (s *ProviderRegistrySeeder) Seed(ctx context.Context, catalog []ProviderCatalogEntry) (SeedResult, error) {
	var res SeedResult
	for _, entry := range catalog {
		existing, err := s.registry.GetDefinitionByKey(ctx, entry.ProviderKey)
		if err != nil && !errors.Is(err, ErrProviderDefinitionNotFound) {
			return res, fmt.Errorf("look up provider %q: %w", entry.ProviderKey, err)
		}
		if existing != nil {
			res.Skipped = append(res.Skipped, entry.ProviderKey)
			continue
		}
		if err := s.seedEntry(ctx, entry); err != nil {
			return res, fmt.Errorf("seed provider %q: %w", entry.ProviderKey, err)
		}
		res.Created = append(res.Created, entry.ProviderKey)
	}
	return res, nil
}

func (s *ProviderRegistrySeeder) seedEntry(ctx context.Context, entry ProviderCatalogEntry) error {
	def, err := s.registry.CreateDefinition(ctx, CreateProviderDefinitionInput{
		ProviderKey:     entry.ProviderKey,
		DisplayName:     entry.DisplayName,
		ProviderType:    entry.ProviderType,
		Regions:         entry.Regions,
		BusinessDomains: entry.BusinessDomains,
		Owner:           "platform",
		Notes:           entry.Notes,
	})
	if err != nil {
		return err
	}
	version, err := s.registry.CreateVersion(ctx, def.ID, entry.Version, "Seeded from existing integration", "", nil)
	if err != nil {
		return err
	}
	if len(entry.Schema) > 0 {
		if _, err := s.registry.SetSchema(ctx, version.ID, entry.Schema); err != nil {
			return err
		}
	}
	if len(entry.Capabilities) > 0 {
		if _, err := s.registry.SetCapabilities(ctx, version.ID, entry.Capabilities); err != nil {
			return err
		}
	}
	return nil
}
