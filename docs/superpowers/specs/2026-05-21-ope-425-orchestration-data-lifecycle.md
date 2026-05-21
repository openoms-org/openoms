# OPE-425 Orchestration Data Lifecycle, Retention And Partitioning Strategy

- **Date:** 2026-05-21
- **Status:** Proposed for review
- **Scope:** Data lifecycle for Provider Integration Studio and Fulfillment Orchestration records before high-volume tables are implemented.
- **Related issues:** `OPE-403`, `OPE-420`, `OPE-424`, `OPE-414`, `OPE-415`, `OPE-422`, `OPE-423`
- **Related documents:** `2026-05-21-ope-424-canonical-logistics-state-adr.md`, `2026-05-17-fulfillment-orchestration-design.md`, `2026-05-17-provider-integration-studio-production-readiness.md`

## Goal

Define how OpenOMS stores, retains, partitions, redacts, archives and deletes orchestration and provider evidence data before implementation creates high-volume production tables.

This document is planning-only. It does not create migrations or production configuration.

## Why This Matters

Provider Integration Studio and Fulfillment Orchestration will create durable records for:

- fulfillment processes,
- fulfillment units,
- fulfillment steps,
- fulfillment events,
- blockers,
- outbox commands,
- side-effect attempts,
- provider validation runs,
- validation results,
- integration observations,
- evidence records,
- publication events,
- status mappings,
- supplier availability snapshots,
- stock propagation attempts.

Without lifecycle rules, these tables can become slow, expensive and difficult to operate. Evidence can also become a privacy and security risk if raw payloads, PII or secrets are retained without strict rules.

## Data Classes

| Class | Examples | Durability requirement | Default retention |
| --- | --- | --- | --- |
| Business truth | fulfillment processes, units, current blockers, canonical state | Durable source of truth | Keep while tenant/account exists, then legal policy |
| Timeline/audit | fulfillment events, publication events, blocker history | Durable audit trail | 24 months, configurable |
| Side-effect execution | outbox rows, provider attempts, retries, idempotency keys | Required for reliability and debugging | 90 days after terminal state |
| Validation history | validation runs, validation results, probe verdicts | Required for publication evidence | 12 months after provider version retirement |
| Evidence metadata | hashes, redacted summaries, source, correlation IDs | Required for traceability | 24 months, configurable |
| Raw/redacted payloads | API response snippets, feed row excerpts, webhook samples | Sensitive; store only when needed | 30-90 days by sensitivity |
| Supplier availability snapshots | raw quantities, freshness, source references | Operationally useful, high volume | 30-180 days depending granularity |
| Aggregated metrics | daily counts, SLO, failure rates | Low sensitivity, useful long-term | 24 months or more |

## Retention Tiers

### Tier 0: Current State

Purpose: fast operational reads.

Examples:

- active `fulfillment_processes`,
- open blockers,
- latest effective capability profile,
- latest supplier availability per product/source,
- latest channel stock propagation state.

Policy:

- indexed for dashboard and worker access;
- no raw payload bloat;
- must fit common tenant-scoped queries.

### Tier 1: Recent Timeline

Purpose: debugging active and recent work.

Examples:

- events from the last 90 days,
- attempts from the last 90 days,
- recent validation runs.

Policy:

- queryable from dashboard/API;
- redacted payload summaries only;
- partitioned by time if table volume requires it.

### Tier 2: Audit Archive

Purpose: compliance, support, root-cause and customer dispute handling.

Examples:

- publication history,
- blocker lifecycle,
- operator override history,
- provider certification results.

Policy:

- slower access is acceptable;
- immutable where relevant;
- no secrets;
- PII minimized.

### Tier 3: Purged/External Archive

Purpose: remove or export data no longer needed in hot Postgres tables.

Examples:

- old attempt payloads,
- old validation detail payloads,
- high-frequency availability snapshots.

Policy:

- purge by scheduled job after retention expires;
- if archived, archive must be encrypted and have access audit;
- delete raw payloads before metadata when possible.

## Table Categories And Lifecycle

### Fulfillment Core Tables

Tables:

- `fulfillment_processes`
- `fulfillment_units`
- `fulfillment_steps`
- `fulfillment_blockers`

Lifecycle:

- active rows stay in primary tables;
- completed/cancelled processes remain queryable for order history;
- soft-delete is not used for business truth;
- tenant deletion must cascade or anonymize according to account policy.

Indexing expectations:

- `(tenant_id, aggregate_status, updated_at DESC)`
- `(tenant_id, health_status, updated_at DESC)`
- `(tenant_id, order_id)`
- `(tenant_id, process_id)`
- `(tenant_id, owner_type, resolved_at)` for blockers

Partitioning:

- not required at day one unless volume tests show pressure;
- revisit if `fulfillment_events` or attempts exceed expected dashboard/query budgets.

### Timeline And Event Tables

Tables:

- `fulfillment_events`
- `provider_publication_events`
- `provider_audit_events`

Lifecycle:

- append-only;
- events older than the recent window can move to archive partitions or remain in cold partitions;
- payload must be redacted by default.

Partitioning:

- monthly range partitioning by `created_at` is the preferred future strategy for high-volume event tables;
- every partition must include tenant-aware indexes.

Minimum indexes:

- `(tenant_id, process_id, created_at DESC)`
- `(tenant_id, event_type, created_at DESC)`
- `(tenant_id, created_at DESC)`

### Outbox And Attempt Tables

Tables:

- `orchestration_outbox`
- `provider_attempts`
- `stock_propagation_attempts`
- `webhook_delivery_attempts` if unified later

Lifecycle:

- pending/running/failed-retry rows stay hot;
- terminal success rows can be compacted after 30-90 days;
- terminal failure rows stay at least 90 days or until blocker is resolved plus retention window;
- idempotency keys must remain long enough to prevent duplicate side effects after retries, redeploys and provider lag.

Partitioning:

- not required for first version if attempts are bounded and cleaned;
- monthly partitions recommended if attempts become the largest operational table.

Indexes:

- `(status, next_attempt_at)`
- `(tenant_id, aggregate_type, aggregate_id)`
- `(tenant_id, idempotency_key)`
- `(tenant_id, provider, operation, created_at DESC)`

### Provider Registry Tables

Tables:

- `provider_definitions`
- `provider_versions`
- `provider_field_schemas`
- `provider_capability_profiles`
- `provider_status_mappings`
- `provider_validation_probes`
- `provider_tenant_enables`

Lifecycle:

- provider definitions and versions are durable business configuration;
- published versions are immutable;
- deprecated and retired versions remain for historical tenant/process references;
- deletion should be administrative and rare.

Partitioning:

- no partitioning expected.

### Validation And Evidence Tables

Tables:

- `provider_validation_runs`
- `provider_validation_results`
- `provider_integration_gaps`
- `integration_observations`
- `provider_evidence_records`

Lifecycle:

- validation verdicts remain at least as long as the provider version is published or referenced;
- raw payload detail expires faster than verdict metadata;
- redacted evidence summaries and hashes can live longer;
- blocking gaps remain until resolved, accepted as risk, or the provider version is retired.

Indexes:

- `(provider_version_id, created_at DESC)`
- `(provider_version_id, verdict, created_at DESC)`
- `(tenant_id, integration_id, created_at DESC)` for tenant-specific validation
- `(severity, resolved_at, created_at DESC)` for gaps

Partitioning:

- validation results and evidence can be monthly partitioned once volume justifies it;
- provider definitions and mappings stay unpartitioned.

### Supplier Availability And Stock Propagation

Tables or table families:

- latest supplier availability state,
- supplier availability snapshots,
- latest channel stock state,
- stock propagation attempts.

Lifecycle:

- latest availability state remains hot;
- snapshots can be retained for 30-180 days depending on operational need;
- high-frequency supplier feeds should store deltas or compressed summaries where possible;
- raw feed rows should not be retained indefinitely unless required for dispute/audit.

Retention recommendations:

| Data | Default |
| --- | --- |
| Latest supplier availability | Keep current |
| Supplier availability snapshots | 90 days |
| Raw supplier feed excerpts | 30 days |
| Stock propagation attempts | 90 days after terminal |
| Manual override history | 24 months |

## Redaction And Privacy Rules

Never store:

- provider credentials,
- API tokens,
- refresh tokens,
- webhook secrets,
- full authorization headers,
- private keys,
- password material.

Store only when necessary and redacted:

- customer names,
- customer emails,
- phone numbers,
- street addresses,
- invoice numbers,
- tracking numbers if customer-identifying in context,
- raw order payloads.

Evidence payload policy:

- default to hash plus selected redacted fields;
- store full raw payload only for explicit debug mode with short TTL and platform-admin access;
- separate evidence payload body from evidence metadata so payload can expire earlier;
- include redaction version so older records can be reprocessed if redaction improves.

## Partitioning Decision

Initial implementation should not over-partition every table.

Use normal tables first for:

- provider definitions,
- provider versions,
- schemas,
- mappings,
- capability profiles,
- current process/unit/step/blocker state.

Design for future monthly partitioning for:

- fulfillment events,
- provider attempts,
- validation results,
- evidence payloads,
- supplier availability snapshots,
- stock propagation attempts.

The first migration should avoid painting us into a corner:

- include `created_at` on every high-volume table;
- include `tenant_id` on every tenant-scoped high-volume table;
- avoid foreign keys that make future partitioning impossible or very expensive without review;
- prefer IDs and indexes that support pruning by time and tenant.

## Cleanup Jobs

Required cleanup jobs before production scale:

| Job | Purpose |
| --- | --- |
| `orchestration_attempt_cleanup` | Remove/compact terminal attempts after retention. |
| `evidence_payload_cleanup` | Delete raw payloads after TTL while preserving metadata. |
| `validation_history_cleanup` | Archive or compact old validation detail. |
| `supplier_availability_snapshot_cleanup` | Purge old feed snapshots. |
| `stock_propagation_attempt_cleanup` | Retain failures longer than successes; purge terminal successes. |

Cleanup requirements:

- tenant-aware where applicable;
- safe batch size;
- context cancellation;
- observable metrics;
- dry-run mode for first rollout;
- no large table locks;
- logs must not print payloads.

## Observability

Metrics to add with implementation:

- number of active fulfillment processes by status/health,
- open blockers by code/severity,
- outbox pending/running/failed counts,
- attempt retry age p95/p99,
- validation run failures by probe class,
- evidence payload cleanup lag,
- supplier availability staleness by provider/source,
- channel stock staleness by provider/channel,
- stock propagation failure count.

Alerts:

- outbox oldest pending age exceeds SLO,
- high `system_error` blocker rate,
- cleanup lag exceeds retention by more than one day,
- raw payload table grows unexpectedly,
- supplier availability stale for critical supplier,
- marketplace stock propagation stale for active channel.

## Security And Access

- Platform evidence access requires platform-admin permission.
- Tenant users see only tenant-safe summaries and their own integration health.
- Raw payload access, if enabled, requires elevated platform permission and audit.
- Evidence export must be explicit and audited.
- Retention jobs must not bypass tenant boundaries except as controlled platform jobs.

## Rollout Plan

1. Accept this strategy before DB schema implementation.
2. Implement schemas with lifecycle fields: `created_at`, `updated_at`, `resolved_at`, `terminal_at`, `retention_class`, `payload_expires_at` where relevant.
3. Add indexes for active state and recent history.
4. Implement cleanup jobs in dry-run mode.
5. Enable cleanup metrics.
6. Run staging volume test with synthetic events/attempts/evidence.
7. Enable cleanup execution.
8. Review table growth after first production week.

## Acceptance Criteria

- Every planned high-volume table has an owner, retention class and indexing strategy.
- Raw payloads have shorter TTL than metadata.
- Stock propagation and supplier availability data have explicit retention.
- Cleanup jobs are required before production-scale enablement.
- ADR `OPE-424` can reference this strategy before schema work starts.
- No table requires real supplier agreements to design its lifecycle.

## Open Questions

- Final default retention for audit history after first legal/GDPR review.
- Whether raw payload storage should be disabled entirely in production until a dedicated encrypted archive exists.
- Whether Supabase plan and table size require partitioning in version one or only partition-ready schemas.
- Whether tenant export/delete workflows need to include provider evidence from the first release.
