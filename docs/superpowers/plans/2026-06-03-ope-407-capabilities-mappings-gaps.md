# OPE-407 Capability Profiles, Status Mappings & Integration Gaps Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: superpowers:subagent-driven-development / executing-plans.

**Goal:** Per-version capability profiles (what a provider version can do), raw→canonical status mappings, and integration gaps (blocking/informational issues) for the Provider Integration Studio.

**Architecture:** Three platform-managed tables (no RLS), bulk replace-set for capabilities + status mappings (frozen once published, like the OPE-406 schema), and a gaps table written by the platform admin and (later) the OPE-408 validation engine. Endpoints under `/v1/platform/providers/{id}/versions/{version_id}`.

**Tech Stack:** Go 1.25, chi/v5, pgx/v5, golang-migrate, testify. Builds on OPE-405 (`provider_versions`, `IsPublishedState`) + OPE-406 patterns.

---

## Scope

In scope: `provider_capability_profiles`, `provider_status_mappings`, `provider_integration_gaps` — models + structural validation + bulk set/get (capabilities, status mappings; frozen on publish) + gaps create/list/resolve + endpoints + tests.

Out of scope: the evidence model + validation runs/results (OPE-408 "Validation Engine And Evidence" — it produces evidence and auto-creates gaps); the runtime "resolve EFFECTIVE capabilities from provider+version+tenant+credentials+runtime" path (needs tenant integrations, OPE-412+); Studio UI (OPE-410). This task delivers provider-version DEFAULT capabilities/mappings + the gaps store.

## Enums (OPE-420 Task 5/6 + spec §139/§170/§223)

- support_status: `supported|configured|unsupported|requires_manual|degraded|unknown`
- status_domain: `order|shipment|line`
- mapping confidence: `high|medium|low`
- gap_type: `missing_source_docs|missing_credential_field|missing_status_mapping|unsupported_capability|stale_data_risk|missing_tracking|missing_order_preflight|ambiguous_product_identity|auth_failure|provider_business_error|parser_failure|manual_fallback_required`
- gap_severity: `info|warning|action_required|system_error`
- gap_status: `open|acknowledged|resolved`

## Data model (migration 000034) — no RLS, dual-role grants

```sql
CREATE TABLE public.provider_capability_profiles (
    id uuid PRIMARY KEY DEFAULT public.uuid_generate_v4(),
    provider_version_id uuid NOT NULL REFERENCES public.provider_versions(id) ON DELETE CASCADE,
    capability_key text NOT NULL,
    support_status text NOT NULL CHECK (support_status IN ('supported','configured','unsupported','requires_manual','degraded','unknown')),
    channel text NOT NULL DEFAULT '',
    mode text NOT NULL DEFAULT '',
    freshness text NOT NULL DEFAULT '',
    required_inputs text[] NOT NULL DEFAULT '{}',
    provided_outputs text[] NOT NULL DEFAULT '{}',
    latency_sla_seconds integer,
    notes text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (provider_version_id, capability_key)
);
CREATE INDEX idx_provider_caps_version ON public.provider_capability_profiles (provider_version_id); -- migrate:index-lock-ok

CREATE TABLE public.provider_status_mappings (
    id uuid PRIMARY KEY DEFAULT public.uuid_generate_v4(),
    provider_version_id uuid NOT NULL REFERENCES public.provider_versions(id) ON DELETE CASCADE,
    status_domain text NOT NULL CHECK (status_domain IN ('order','shipment','line')),
    raw_status text NOT NULL,
    canonical_status text NOT NULL DEFAULT '',  -- '' = unmapped (creates a gap)
    canonical_event_type text NOT NULL DEFAULT '',
    canonical_step_key text NOT NULL DEFAULT '',
    confidence text NOT NULL DEFAULT 'medium' CHECK (confidence IN ('high','medium','low')),
    is_terminal boolean NOT NULL DEFAULT false,
    notes text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (provider_version_id, status_domain, raw_status)
);
CREATE INDEX idx_provider_status_maps_version ON public.provider_status_mappings (provider_version_id); -- migrate:index-lock-ok

CREATE TABLE public.provider_integration_gaps (
    id uuid PRIMARY KEY DEFAULT public.uuid_generate_v4(),
    provider_version_id uuid NOT NULL REFERENCES public.provider_versions(id) ON DELETE CASCADE,
    gap_type text NOT NULL,
    severity text NOT NULL CHECK (severity IN ('info','warning','action_required','system_error')),
    status text NOT NULL DEFAULT 'open' CHECK (status IN ('open','acknowledged','resolved')),
    description text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    resolved_at timestamptz
);
CREATE INDEX idx_provider_gaps_version ON public.provider_integration_gaps (provider_version_id, status); -- migrate:index-lock-ok
```
Dual-role grants DO-block. `.down.sql`: drop the 3 tables.

## Implementation Tasks (TDD)

### Task 1: Migration + model
- [ ] `migrations/000034_provider_capabilities_mappings_gaps.up.sql`/`.down.sql`.
- [ ] `internal/model/provider_capability.go`: `ProviderCapability`, `ProviderStatusMapping`, `ProviderIntegrationGap` structs; enum constants + `IsValidSupportStatus/StatusDomain/Confidence/GapType/GapSeverity/GapStatus`; `ValidateCapabilities([]ProviderCapability) error` (valid support_status, non-empty unique capability_key, latency>=0); `ValidateStatusMappings([]ProviderStatusMapping) error` (valid domain+confidence, non-empty raw_status, unique (domain,raw_status)).
- [ ] `model/provider_capability_test.go`: validation matrix.

### Task 2: Repositories (pool, no RLS)
- [ ] `repository/provider_capability_repository.go`: `ReplaceCapabilities(ctx, q Querier, versionID, []ProviderCapability) error` (DELETE then INSERT in the caller tx), `ListCapabilities(ctx, versionID)`. `ReplaceStatusMappings(ctx, q, versionID, []ProviderStatusMapping)`, `ListStatusMappings(ctx, versionID)`. `CreateGap(ctx, versionID, gap) (*ProviderIntegrationGap, error)`, `ListGaps(ctx, versionID)`, `UpdateGapStatus(ctx, gapID, status) (*ProviderIntegrationGap, error)`.

### Task 3: Service (on ProviderRegistryService)
- [ ] Add `caps *repository.ProviderCapabilityRepository`; constructor arg.
- [ ] `SetCapabilities(versionID, caps)` / `SetStatusMappings(versionID, mappings)`: GetVersion (404); frozen → `ErrProviderVersionFrozen`; validate → `ErrInvalidCapability`/`ErrInvalidStatusMapping`; replace inside `inTx` (DELETE+INSERT atomic).
- [ ] `GetCapabilities`/`GetStatusMappings` (404 if version absent).
- [ ] `CreateGap(versionID, type, severity, description)` (validate enums → `ErrInvalidGap`), `ListGaps(versionID)`, `ResolveGap(gapID, status)`.

### Task 4: Handler + router
- [ ] Add to `ProviderRegistry` interface + `ProviderHandler`: GetCapabilities/UpdateCapabilities, GetStatusMappings/UpdateStatusMappings, ListGaps/CreateGap/UpdateGap. Map the new `ErrInvalid*` → 422. Audit mutations.
- [ ] Router under `/versions/{version_id}`: `GET/PATCH /capabilities` (read/write), `GET/PATCH /status-mappings` (read/write), `GET /gaps` (read), `POST /gaps` (write), `PATCH /gaps/{gap_id}` (write).
- [ ] Handler tests: invalid → 422, frozen → 422, success audits.

### Task 5: Integration test
- [ ] `tests/integration/provider_capabilities_test.go`: create def+version → SetCapabilities (valid) + GetCapabilities → SetStatusMappings + Get → CreateGap + ListGaps + ResolveGap → invalid each → 422 errs → walk to private_beta → SetCapabilities frozen. appPool.

### Task 6: Validate + docs
- [ ] go test ./..., vet, gofmt, golangci-lint; integration green; migrate down/up.
- [ ] `docs/system-documentation.md` — add the 3 tables + endpoints under the Studio block.

## Risks
- Replace-set atomicity (DELETE+INSERT) — done inside `inTx` (reuse OPE-405 pattern).
- Enum/validation completeness — table-driven tests.
- Migration Safety CI — indexes marked index-lock-ok; keep "CREATE INDEX" out of comments.

## Self-Review
Covers OPE-420 Task 5 (capability dimensions + support states + gaps w/ severity) + Task 6 (status mapping: raw stored, unknown→gap signalled via empty canonical_status, confidence, terminal explicit, domains separate). Evidence + runtime effective-capability resolution are later tasks (OPE-408 / OPE-412).
