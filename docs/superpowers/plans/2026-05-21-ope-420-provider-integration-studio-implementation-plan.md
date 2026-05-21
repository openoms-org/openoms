# OPE-420 Provider Integration Studio Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build Provider Integration Studio as an internal platform-admin system for designing, validating, versioning, publishing, and operating OpenOMS provider integrations.

**Architecture:** The Studio is capability-class-first. Provider names are test vectors and registry entries; core OpenOMS works through provider definitions, versions, schemas, capability profiles, validation probes, status mappings, evidence, gaps, and publication state. Customer-visible integrations read only published or explicitly tenant-enabled provider versions.

**Tech Stack:** Go 1.25 API server, PostgreSQL 16 with RLS for tenant tables, Next.js 16 dashboard, React 19, TypeScript, Tailwind v4, shadcn/ui, Linear issue tracking.

---

## Scope

This plan covers `OPE-404` through `OPE-412`, the Provider Integration Studio side of `OPE-403`.

Included:

- platform-admin access boundary,
- provider registry and version lifecycle,
- credential/settings schemas,
- capability profiles,
- status mappings,
- validation probes and immutable validation runs,
- evidence and gap records,
- internal platform UI shell,
- publication and tenant visibility controls,
- migration of existing static provider assumptions into registry records.

Excluded from this plan:

- fulfillment orchestration runtime implementation (`OPE-414` through `OPE-423`),
- supplier class proof and certification (`OPE-428`, `OPE-413`),
- real supplier access execution (`OPE-432`),
- customer-visible provider enablement before publication gates are met.

## Preconditions

- `OPE-434` must record explicit owner approval for implementation scope.
- `OPE-424` canonical logistics state ADR must be accepted before schemas or mappings are implemented.
- `OPE-425` lifecycle/retention strategy must be accepted before high-volume evidence, validation, and attempt tables are created.
- Existing OPE-403 source docs must be committed or intentionally included in the same docs PR.

## Files And Areas

Backend areas:

- `public/apps/api-server/internal/router/router.go` — platform route group.
- `public/apps/api-server/internal/middleware/` — platform-admin auth middleware.
- `public/apps/api-server/internal/model/` — provider registry, schemas, capabilities, mappings, validation, evidence.
- `public/apps/api-server/internal/repository/` — Postgres repositories.
- `public/apps/api-server/internal/service/` — lifecycle, capability, validation, publication services.
- `public/apps/api-server/internal/worker/` — validation worker if probes become async.
- `public/apps/api-server/migrations/` — future DB migrations after ADR approval.
- `public/apps/api-server/docs/openapi.yaml` — platform API contract after endpoints exist.

Frontend areas:

- `public/apps/dashboard/src/app/(dashboard)/platform/` or a platform route group selected during implementation.
- `public/apps/dashboard/src/components/platform/` — Studio components.
- `public/apps/dashboard/src/hooks/` — provider registry and validation hooks.
- `public/apps/dashboard/src/lib/` — platform API client helpers only if existing client cannot cover the route cleanly.
- `public/apps/dashboard/src/types/` — provider registry types.

Documentation areas:

- `public/docs/superpowers/specs/2026-05-17-provider-integration-studio-design.md`
- `public/docs/superpowers/specs/2026-05-17-provider-integration-studio-production-readiness.md`
- `public/docs/superpowers/specs/2026-05-17-provider-integration-studio-ui-ux-design.md`
- `public/docs/superpowers/specs/2026-05-21-ope-424-canonical-logistics-state-adr.md`
- `public/docs/superpowers/specs/2026-05-21-ope-425-orchestration-data-lifecycle.md`
- `public/.claude/context/API_CONTRACTS.md`
- `public/.claude/context/DOMAIN_MODEL.md`
- `public/.claude/context/DECISIONS.md`
- `public/.claude/context/SECURITY_POSTURE.md`
- `public/docs/system-documentation.md`

## Implementation Tasks

### Task 1: Persist Planning Baseline

**Files:**

- Add or update: `public/docs/superpowers/specs/2026-05-17-provider-integration-studio-design.md`
- Add or update: `public/docs/superpowers/specs/2026-05-17-provider-integration-studio-production-readiness.md`
- Add or update: `public/docs/superpowers/specs/2026-05-17-provider-integration-studio-ui-ux-design.md`
- Add or update: `public/docs/superpowers/specs/2026-05-17-provider-integration-studio-gap-analysis.md`
- Add or update: `public/docs/superpowers/templates/provider-integration-builder.md`
- Add: `public/docs/superpowers/plans/2026-05-21-ope-420-provider-integration-studio-implementation-plan.md`

- [ ] Confirm all source docs are tracked by git and not only local untracked files.
- [ ] Add links from `OPE-403`, `OPE-420`, and related Linear issues to the committed docs.
- [ ] Verify the docs state that Studio is internal platform-admin tooling, not a customer-facing setup screen.
- [ ] Verify the docs state provider names are test vectors and not architecture axes.
- [ ] Run `cd public && git diff --check`.

### Task 2: Platform-Admin Boundary

**Files:**

- Future modify: `public/apps/api-server/internal/middleware/`
- Future modify: `public/apps/api-server/internal/router/router.go`
- Future modify: `public/apps/api-server/internal/model/`
- Future modify: `public/apps/api-server/internal/repository/`
- Future modify: `public/apps/api-server/internal/service/`
- Future modify: `public/apps/dashboard/src/app/`

- [ ] Define `platform_admins` and platform permission model after `OPE-424` and `OPE-425` approval.
- [ ] Add `RequirePlatformAdmin` middleware that does not reuse tenant `RequireRole` or `RequirePermission`.
- [ ] Ensure platform routes do not trust frontend visibility as authorization.
- [ ] Add audit records for platform read/write/validate/publish/secrets actions.
- [ ] Add tests proving tenant owners/admins/members cannot access platform APIs.
- [ ] Add tests proving platform admins without specific permission cannot publish or access secrets.

Validation:

- `cd public/apps/api-server && go test ./internal/middleware ./internal/handler -run 'Platform|Provider' -count=1`

### Task 3: Provider Registry And Lifecycle Data Model

**Files:**

- Future create migration under `public/apps/api-server/migrations/`
- Future create/modify models, repositories, services under `public/apps/api-server/internal/`

- [ ] Create provider definition and provider version models.
- [ ] Create lifecycle states: `research`, `designed`, `adapter_in_progress`, `internal_validation`, `private_beta`, `available`, `deprecated`, `retired`.
- [ ] Enforce allowed lifecycle transitions in service code.
- [ ] Make published versions immutable except controlled publication state transitions.
- [ ] Add publication events as append-only audit records.
- [ ] Add tenant private-beta enablement records.
- [ ] Add emergency disable path with audit and rollback to previous version.

Validation:

- `cd public/apps/api-server && go test ./internal/model ./internal/repository ./internal/service -run 'Provider|Lifecycle|Publication' -count=1`

### Task 4: Credential And Settings Schema Contract

**Files:**

- Future create/modify backend model/repository/service files.
- Future create dashboard schema-builder types and components.

- [ ] Define schema field groups: secret credentials, non-secret settings, environment settings, sync settings, feature toggles, provider-specific options.
- [ ] Define field validation: required, type, enum, regex, min/max, redaction policy, environment scope.
- [ ] Generate platform-admin validation forms from the schema.
- [ ] Generate tenant setup forms only for published or tenant-enabled provider versions.
- [ ] Ensure secret values are never returned to the dashboard after save.
- [ ] Add credential rotation rules and audit.

Validation:

- `cd public/apps/api-server && go test ./internal/service -run 'ProviderSchema|Credentials|Settings' -count=1`
- `cd public/apps/dashboard && npx vitest run --reporter=dot src --runInBand`

### Task 5: Capability Profiles And Gaps

**Files:**

- Future create/modify capability model, repository, service and UI files.

- [ ] Define capability dimensions: `entity_type`, `operation`, `direction`, `channel`, `freshness`, `authority`, `support_status`, `required_fields`, `provided_fields`, `latency_sla_seconds`.
- [ ] Include inventory capabilities for inbound availability reads and outbound stock writes.
- [ ] Support `supported`, `configured`, `unsupported`, `requires_manual`, `degraded`, and `unknown`.
- [ ] Resolve effective capabilities from provider defaults, provider version, tenant settings, credentials health, and runtime checks.
- [ ] Store gaps with severity: `info`, `warning`, `action_required`, `system_error`.
- [ ] Create gaps for stale supplier availability, unsupported stock write, no stock acknowledgement, manual override active, unknown external status, and missing tracking.

Validation:

- `cd public/apps/api-server && go test ./internal/service -run 'Capability|Gap|Availability|Stock' -count=1`

### Task 6: Status Mapping Workbench Contract

**Files:**

- Future create/modify mapping model, repository, service and UI files.

- [ ] Store raw statuses as sanitized observations before canonical mapping.
- [ ] Define mapping precedence: system default, provider version, tenant override, integration override.
- [ ] Separate order, line, shipment, package, supplier order, and carrier tracking status mappings.
- [ ] Block automation when a required external status is unmapped.
- [ ] Allow low-confidence mapping only with explicit review state.
- [ ] Add tests for unknown status, terminal status, tenant override, and integration override.

Validation:

- `cd public/apps/api-server && go test ./internal/service -run 'StatusMapping|ExternalStatus|Observation' -count=1`

### Task 7: Validation Engine And Evidence

**Files:**

- Future create/modify validation model, repository, service, worker, and UI files.

- [ ] Define probe classes: auth, schema, catalog, stock read, stock write, stock increase, stock decrease, stock acknowledgement, price, order preflight, order create test, order status, tracking, invoice, webhook signature, malformed payload, rate limit, evidence redaction.
- [ ] Mark probes as safe, read-only, destructive-test, or production-forbidden.
- [ ] Persist immutable validation runs and validation results.
- [ ] Store evidence with redaction, payload hashes, TTL class, actor, environment, provider version, and correlation ID.
- [ ] Create blocking gaps from failed required probes.
- [ ] Prevent publication while blocking gaps remain unresolved.

Validation:

- `cd public/apps/api-server && go test ./internal/service ./internal/worker -run 'Validation|Probe|Evidence' -count=1`

### Task 8: Platform UI Shell And Workspaces

**Files:**

- Future create dashboard platform route group and components.

- [ ] Add platform entry visible only to platform-admin users.
- [ ] Add Provider Registry table with lifecycle, readiness, owner, blockers, region, provider class, and latest validation.
- [ ] Add Provider Detail tabs: overview, versions, credential schema, capabilities, mappings, validation lab, evidence, tenant visibility, runbooks, audit.
- [ ] Add autosave-safe editing for draft provider versions.
- [ ] Ensure publish/private beta/deprecate actions require review flow and permission checks.
- [ ] Use existing OpenOMS UI system; do not add decorative landing pages.

Validation:

- `cd public/apps/dashboard && npx eslint --quiet src/`
- `cd public/apps/dashboard && npx vitest run --reporter=dot src`

### Task 9: Existing Provider Migration

**Files:**

- Future modify existing provider adapter metadata in `public/apps/api-server/internal/integration/`.
- Future add registry seed/backfill logic after schema approval.

- [ ] Inventory current marketplace, carrier, supplier, invoice, shop, and automation providers.
- [ ] Create draft registry definitions for existing providers without changing tenant behavior.
- [ ] Migrate static setup fields into provider schemas.
- [ ] Migrate static status maps into versioned mappings.
- [ ] Mark unsupported and unverified capabilities as `unknown` or `requires_manual`, not `supported`.
- [ ] Add compatibility path where `integrations.provider` continues to work during rollout.

Validation:

- `cd public/apps/api-server && go test ./internal/integration ./internal/service -run 'Provider|Registry|Migration' -count=1`

### Task 10: Publication Gates And Rollout

**Files:**

- Future modify services, UI, docs, and tests.

- [ ] Implement gates for `internal_validation`, `private_beta`, `available`, `deprecated`, and `retired`.
- [ ] Require evidence for every critical capability before `available`.
- [ ] Require private beta tenant enablement before customer-visible testing.
- [ ] Add emergency disable and rollback.
- [ ] Add runbook links and ownership metadata.
- [ ] Add dashboard warnings when tenant integrations depend on deprecated or disabled provider versions.

Validation:

- `cd public/apps/api-server && go test ./internal/service -run 'Publication|TenantEnable|ProviderDisable' -count=1`
- `cd public && ./scripts/local-ci.sh --quick`

## Risks

- Platform-admin auth can accidentally become tenant-admin auth if the boundary is reused. Mitigation: separate middleware, tests, and route group.
- Provider registry can become a second source of truth while old `integrations.provider` remains. Mitigation: compatibility path and staged migration.
- Evidence can leak credentials or PII. Mitigation: redaction-by-default, payload hashes, TTL classes, and security review before storage.
- Validation probes can perform destructive actions against production accounts. Mitigation: probe safety classification and production-forbidden controls.
- UI can expose draft providers to customers too early. Mitigation: backend publication gates and tenant visibility checks.

## Rollback Notes

- Docs-only planning rollback: revert the docs PR.
- Early implementation rollback: disable platform routes behind feature flag and keep existing integration setup path unchanged.
- Data model rollback must be planned per migration. Do not drop platform tables until no code references them and evidence retention obligations are reviewed.
- Provider publication rollback: emergency disable provider version or revert tenant enablement to previous published version.

## Open Questions

- Final platform-admin identity source: local table, existing users with separate claim, or Cloudflare Access identity mapped to local platform permissions.
- Exact evidence TTL by class after `OPE-425`.
- Whether validation probes run synchronously for read-only checks and asynchronously for destructive or long-running checks.
- How much of Provider Integration Studio should ship before fulfillment orchestration data model exists.

## Self-Review

- Spec coverage: maps to Provider Integration Studio design, production readiness, UI/UX, supplier research, and fulfillment capability requirements.
- Placeholder scan: this plan intentionally leaves future file names at area level because implementation is blocked until ADR acceptance; no code is implemented here.
- Type consistency: capability dimensions, lifecycle states, and gap severities match the source specs.
