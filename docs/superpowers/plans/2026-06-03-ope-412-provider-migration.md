# OPE-412 Existing Provider Migration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: superpowers:subagent-driven-development / executing-plans.

**Goal:** Seed the provider registry (OPE-405/406/407) with draft definitions for existing providers, class-first (§526), without changing tenant behavior. Capabilities are seeded conservatively as `unknown` (verified later via the Studio), not fabricated as `supported`.

**Architecture:** A static Go catalog of class-first representatives + an idempotent `ProviderRegistrySeeder` that creates definition + research version + credential schema + capability stubs per entry (skipping providers already present). Invoked as an env-gated startup backfill (`SEED_PROVIDERS=true`), mirroring the existing tenant-encryption backfill. Additive only — tenant `integrations.provider` is untouched.

**Tech Stack:** Go 1.25, pgx/v5, testify. Builds on OPE-405/406/407.

---

## Scope

In scope: static catalog (Allegro/marketplace, InPost/carrier, BTP/supplier, Fakturownia/invoicing, Shopify/shop — the §526 representatives), idempotent seeder, credential schemas migrated from known integration fields, capability stubs at `unknown`, env-gated startup invocation, tests.

Out of scope: status-mapping migration + capability verification (done via the OPE-407 workbench + OPE-408 validation by platform admins; the static maps lack rich raw→canonical data); publication of the seeded versions (OPE-413, gated); migrating ALL ~25 providers (class-first first — the rest follow the same pattern later); replacing the frontend static maps (kept as fallback during rollout per §526.10).

## Catalog (class-first, §526) — derived from apps/dashboard/src/lib/provider-info.ts

| key | type | credential schema (secret*) | capability stubs (all `unknown`) |
|-----|------|------------------------------|----------------------------------|
| allegro | marketplace | client_id*, client_secret*, environment(enum sandbox/production) | marketplace.order.pull, marketplace.order.status.push |
| inpost | carrier | api_token*, organization_id | carrier.shipment.create, carrier.tracking.read |
| btp | supplier | username*, password*, feed_url(url) | supplier.catalog.read, supplier.availability.read, supplier.price.read |
| fakturownia | invoicing | api_token*, domain | invoice.issue, invoice.pdf.read |
| shopify | shop | shop_url(url), access_token* | shop.order.pull, shop.product.push |

All capabilities seeded with `support_status: unknown` (conservative — the Studio verifies/upgrades them). Each definition gets one version `1.0.0` in `research` (a draft).

## Implementation Tasks (TDD)

### Task 1: Catalog + seeder
- [ ] `internal/service/provider_catalog.go`: `type ProviderCatalogEntry struct { ProviderKey, DisplayName, ProviderType, Version, Notes string; Regions, BusinessDomains []string; Schema []model.ProviderFieldGroup; Capabilities []model.ProviderCapability }` + `func ProviderRegistryCatalog() []ProviderCatalogEntry` returning the 5 entries above.
- [ ] Add `GetDefinitionByKey(ctx, key) (*model.ProviderDefinition, error)` to `ProviderRegistryService` (wraps `defs.GetByKey`; pgx.ErrNoRows → `ErrProviderDefinitionNotFound`).
- [ ] `internal/service/provider_registry_seeder.go`: `ProviderRegistrySeeder` holding `*ProviderRegistryService`; `Seed(ctx, []ProviderCatalogEntry) (SeedResult, error)`. Per entry: `GetDefinitionByKey` → if found, skip; else `CreateDefinition` + `CreateVersion` (research) + `SetSchema` + `SetCapabilities`. `SeedResult{Created, Skipped []string}`. Idempotent + resilient (a single bad entry returns an error, not a partial silent skip).

### Task 2: Catalog validity unit test
- [ ] `internal/service/provider_catalog_test.go`: for every entry assert `model.IsValidProviderType`, `model.ValidateFieldSchema(Schema)` ok, `model.ValidateCapabilities(Capabilities)` ok, all caps `support_status == unknown`, unique provider keys, non-empty version.

### Task 3: Config + startup wiring
- [ ] `internal/config/config.go`: add `SeedProviders bool \`env:"SEED_PROVIDERS" envDefault:"false"\``.
- [ ] `cmd/server/main.go`: after the registry service is built, `if cfg.SeedProviders { res, err := seeder.Seed(ctx, service.ProviderRegistryCatalog()); log/return-on-error }` (mirror the encryption-backfill block; non-fatal in dev, fatal in prod).

### Task 4: Integration test
- [ ] `tests/integration/provider_seeder_test.go`: build the registry service + seeder on appPool; `Seed(ctx, catalog)` → assert 5 created; for `allegro` assert GetDefinitionByKey returns marketplace + a version exists + GetSchema has the client_id secret field + GetCapabilities all `unknown`. Re-run `Seed` → 0 created, 5 skipped (idempotent). Cleanup: delete the 5 definitions by key.

### Task 5: Validate + docs
- [ ] go test ./..., vet, gofmt, golangci-lint; integration green.
- [ ] `docs/system-documentation.md` — note the seeder + `SEED_PROVIDERS` under the Studio block.

## Risks
- Idempotency: keyed on `provider_key` UNIQUE; `GetDefinitionByKey` gate prevents duplicates. Re-runnable.
- Honesty: no fabricated `supported` capabilities — all `unknown`, upgraded only by real validation (OPE-408).
- Additive: tenant integrations untouched; frontend static maps kept as fallback (§526.10). No SQL migration (registry tables already exist).

## Self-Review
Covers OPE-420 Task 9: inventory (catalog), draft registry definitions without changing tenant behavior, migrate static setup fields into schemas, unverified capabilities as `unknown`, compatibility path (additive). Status-map migration + verification deferred to the Studio workbench/validation by design.
