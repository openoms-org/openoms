# OpenOMS Fulfillment Orchestration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a durable, tenant-safe fulfillment orchestration layer for OpenOMS that unifies orders, shipments, dropship, warehouse execution, backorders, provider side effects, automation, operational blockers, dashboard visibility, and observability.

**Architecture:** OpenOMS remains the orchestrator. Supabase PostgreSQL is the production source of truth for process state, outbox, blockers, attempts, and timelines. Go services execute typed domain commands through transactional state changes and idempotent workers; Redis supports scheduling/locks but does not own durable business state.

**Tech Stack:** Go 1.25, chi/v5, pgx/v5, PostgreSQL 16 on Supabase, RLS, Redis/asynq-compatible worker patterns, Next.js 16, React 19, TypeScript, Tailwind v4, shadcn/ui, Playwright, OpenTelemetry-compatible observability.

**Companion research:** `docs/superpowers/specs/2026-05-17-supplier-integration-research.md` documents real supplier/wholesale communication patterns and should be used as input for Epic 4.

**Companion tooling design:** `docs/superpowers/specs/2026-05-17-provider-integration-studio-design.md`, `docs/superpowers/specs/2026-05-17-provider-integration-studio-gap-analysis.md`, `docs/superpowers/specs/2026-05-17-provider-integration-studio-production-readiness.md`, `docs/superpowers/specs/2026-05-17-provider-integration-studio-ui-ux-design.md`, and `docs/superpowers/templates/provider-integration-builder.md` define the reusable provider integration workflow, internal admin validator, required changes to the current environment, production readiness gates, and platform-admin UI/UX.

---

## Scope

In scope:

- First-class fulfillment process model.
- Supabase/PostgreSQL schema and RLS.
- Transactional outbox for all fulfillment side effects.
- Idempotent orchestration workers.
- Unified order, shipment, marketplace import, label, tracking, dropship, warehouse, purchase/backorder, and pick-pack flow.
- Integration capability profiles for suppliers, marketplaces, carriers, and shops with explicit mapping of supported, missing, manual, degraded, and unknown capabilities.
- Typed blockers and attempts.
- Operations dashboard backed by process/blocker read models.
- Automation v2 that routes state-changing actions through orchestrator commands.
- Optional n8n connector for controlled custom actions after the core state model is stable.
- Monitoring, metrics, logs, audit, and restore/backfill strategy.

Out of scope:

- Replacing OpenOMS with n8n, Camunda, or Temporal as the source of truth.
- Moving production data out of Supabase.
- Rewriting unrelated billing, catalog, marketing, or AI modules.
- Removing legacy order/shipment statuses before migration is complete.

## Implementation Strategy

This plan should be executed as a sequence of Linear issues. Each phase should produce software that can run in production while coexisting with the previous behavior. The first implementation pass must favor additive schema, dual-read/dual-write where needed, feature flags, and measurable parity before any old behavior is retired.

Required architecture decisions before runtime implementation:

- Canonical logistics state model for fulfillment, shipment, supplier order, marketplace order and external provider observations.
- Data lifecycle and retention strategy for process, outbox, attempt, evidence, blocker and validation tables.
- Capability-class proof gates for marketplace, carrier, supplier, dropship, shop, B2B document and enterprise distributor integrations.
- Rule that named providers in tests are representatives of capability classes; acceptance depends on reusable class contracts, not a single provider happy path.

Recommended Linear decomposition:

- Epic 0: Canonical logistics state model, data lifecycle and capability-class proof gates.
- Epic 1: Fulfillment data model and RLS.
- Epic 2: Outbox and orchestration worker.
- Epic 3: Domain command unification.
- Epic 4: Integration capability, status mapping, and evidence model.
- Epic 5: Shipment, label, tracking, and provider attempts.
- Epic 6: Warehouse, pick-pack, dropship, and backorder units.
- Epic 7: Dashboard and operations APIs.
- Epic 8: Automation v2 and n8n connector.
- Epic 9: Observability, backfill, rollout, and hardening.

## Files And Areas

New backend areas:

- `apps/api-server/internal/model/fulfillment.go`
- `apps/api-server/internal/repository/fulfillment_process_repository.go`
- `apps/api-server/internal/repository/fulfillment_unit_repository.go`
- `apps/api-server/internal/repository/fulfillment_step_repository.go`
- `apps/api-server/internal/repository/fulfillment_event_repository.go`
- `apps/api-server/internal/repository/fulfillment_blocker_repository.go`
- `apps/api-server/internal/repository/orchestration_outbox_repository.go`
- `apps/api-server/internal/repository/orchestration_attempt_repository.go`
- `apps/api-server/internal/repository/integration_capability_repository.go`
- `apps/api-server/internal/repository/external_status_mapping_repository.go`
- `apps/api-server/internal/repository/integration_observation_repository.go`
- `apps/api-server/internal/service/fulfillment_orchestrator.go`
- `apps/api-server/internal/service/fulfillment_commands.go`
- `apps/api-server/internal/service/fulfillment_policy_service.go`
- `apps/api-server/internal/service/fulfillment_read_service.go`
- `apps/api-server/internal/service/integration_capability_service.go`
- `apps/api-server/internal/service/external_status_mapping_service.go`
- `apps/api-server/internal/worker/orchestration_worker.go`
- `apps/api-server/internal/handler/fulfillment_handler.go`
- `apps/api-server/internal/handler/operations_handler.go`

Backend areas to modify:

- `apps/api-server/internal/service/order_service.go`
- `apps/api-server/internal/service/shipment_service.go`
- `apps/api-server/internal/service/label_service.go`
- `apps/api-server/internal/service/dropship_service.go`
- `apps/api-server/internal/service/purchase_order_service.go`
- `apps/api-server/internal/service/pick_pack_service.go`
- `apps/api-server/internal/worker/marketplace_order_poller.go`
- `apps/api-server/internal/worker/tracking_poller.go`
- `apps/api-server/internal/automation/engine.go`
- `apps/api-server/internal/automation/actions.go`
- `apps/api-server/internal/router/router.go`
- `apps/api-server/internal/middleware/metrics.go`
- `apps/api-server/migrations/`

Dashboard areas to modify:

- `apps/dashboard/src/hooks/use-operations-dashboard.ts`
- `apps/dashboard/src/lib/operations-dashboard.ts`
- `apps/dashboard/src/components/dashboard/orchestration-map.tsx`
- `apps/dashboard/src/components/dashboard/operational-exceptions.tsx`
- `apps/dashboard/src/app/(dashboard)/page.tsx`
- `apps/dashboard/src/app/(dashboard)/orders/[id]/`
- `apps/dashboard/src/types/api.ts`

Docs to update:

- `.Codex/context/API_CONTRACTS.md`
- `.Codex/context/DOMAIN_MODEL.md`
- `.Codex/context/DECISIONS.md`
- `.Codex/context/SECURITY_POSTURE.md`
- `.Codex/context/PROJECT_STATE.md`
- `docs/system-documentation.md`

## Phase 1: Schema Foundation

**Goal:** Add durable process, unit, step, event, blocker, outbox, attempt, and policy tables without changing runtime behavior.

**Files:**

- Create migration: `apps/api-server/migrations/<next>_fulfillment_orchestration.up.sql`
- Create migration rollback: `apps/api-server/migrations/<next>_fulfillment_orchestration.down.sql`
- Modify docs: `.Codex/context/DOMAIN_MODEL.md`
- Modify docs: `.Codex/context/SECURITY_POSTURE.md`

Tasks:

- [ ] Add `fulfillment_processes`.
- [ ] Add `fulfillment_units`.
- [ ] Add `fulfillment_steps`.
- [ ] Add `fulfillment_events`.
- [ ] Add `fulfillment_blockers`.
- [ ] Add `orchestration_outbox`.
- [ ] Add `orchestration_attempts`.
- [ ] Add `fulfillment_policies`.
- [ ] Add `provider_capability_definitions`.
- [ ] Add `integration_capability_profiles`.
- [ ] Add `external_status_mappings`.
- [ ] Add `integration_observations`.
- [ ] Add `integration_capability_gaps`.
- [ ] Add indexes for tenant, process, order, status, health, blockers, outbox status, next attempt, and idempotency keys.
- [ ] Add indexes for provider type, integration, supplier, capability operation, external status, mapping scope, and unresolved capability gaps.
- [ ] Enable and force RLS on every new tenant-scoped table.
- [ ] Add tenant isolation policies using `current_setting('app.current_tenant_id', true)` with no tautological fallback.
- [ ] Add check constraints for enum-like status columns.
- [ ] Add comments for all operationally important columns.
- [ ] Verify migrations on a fresh local PostgreSQL database.
- [ ] Verify RLS denies reads when tenant context is unset.
- [ ] Verify RLS allows reads when tenant context matches.

Validation:

- `cd public/apps/api-server && go test ./internal/database/...`
- Run fresh migration against a throwaway PostgreSQL 16 container.
- Run SQL smoke checks for table existence, indexes, constraints, and RLS.

Acceptance criteria:

- Fresh database migrates successfully.
- Existing tests pass.
- New tables cannot be read without tenant context.
- No application behavior changes yet.

## Phase 2: Model And Repository Layer

**Goal:** Add typed Go models and repositories for the new orchestration tables.

**Files:**

- Create: `apps/api-server/internal/model/fulfillment.go`
- Create: `apps/api-server/internal/repository/fulfillment_process_repository.go`
- Create: `apps/api-server/internal/repository/fulfillment_unit_repository.go`
- Create: `apps/api-server/internal/repository/fulfillment_step_repository.go`
- Create: `apps/api-server/internal/repository/fulfillment_event_repository.go`
- Create: `apps/api-server/internal/repository/fulfillment_blocker_repository.go`
- Create: `apps/api-server/internal/repository/orchestration_outbox_repository.go`
- Create: `apps/api-server/internal/repository/orchestration_attempt_repository.go`
- Add tests beside each repository or in existing repository test suites.

Tasks:

- [ ] Define typed statuses, blocker codes, unit types, step keys, actor types, and payload structs.
- [ ] Add validation methods for process status transitions and step status transitions.
- [ ] Add repository create/get/list/update methods under tenant context.
- [ ] Add outbox claim method using `FOR UPDATE SKIP LOCKED`.
- [ ] Add idempotency-key lookup and conflict handling.
- [ ] Add blocker resolve method with actor and resolution note.
- [ ] Add append-only event method.
- [ ] Add attempt start/finish methods.
- [ ] Add tests for tenant isolation, idempotency conflicts, outbox claiming, blocker resolution, and event ordering.

Validation:

- `cd public/apps/api-server && go test ./internal/model ./internal/repository/...`

Acceptance criteria:

- Repository layer is usable without touching order/shipment behavior.
- Concurrent outbox claims do not claim the same row twice.
- Event order is stable by creation time and ID.

## Phase 2A: Integration Capability, Mapping, And Evidence Layer

**Goal:** Make supplier, marketplace, carrier, and shop differences explicit so OpenOMS can distinguish automated, manual, unsupported, degraded, and missing integration behavior.

**Files:**

- Create: `apps/api-server/internal/model/integration_capability.go`
- Create: `apps/api-server/internal/repository/integration_capability_repository.go`
- Create: `apps/api-server/internal/repository/external_status_mapping_repository.go`
- Create: `apps/api-server/internal/repository/integration_observation_repository.go`
- Create: `apps/api-server/internal/service/integration_capability_service.go`
- Create: `apps/api-server/internal/service/external_status_mapping_service.go`
- Modify: `apps/api-server/internal/model/integration.go`
- Modify: `apps/api-server/internal/model/supplier.go`
- Modify: `apps/api-server/internal/integration/marketplace.go`
- Modify: `apps/api-server/internal/integration/supplier.go`
- Modify: `apps/api-server/internal/integration/carrier.go`
- Add tests for capability resolution, mapping, and gap creation.

Tasks:

- [ ] Define capability enums: `entity_type`, `operation`, `direction`, `channel`, `freshness`, `authority`, and `support_status`.
- [ ] Define gap severities: `info`, `warning`, `action_required`, and `system_error`.
- [ ] Add provider adapter default capability declarations for marketplaces, suppliers, and carriers.
- [ ] Add stock capability declarations for inbound availability reads and outbound stock writes: exact quantity, availability bucket, stock increase support, stock decrease support, bulk update, async acknowledgement, freshness SLA, and unsupported/manual-only modes.
- [ ] Add integration-level effective capability profiles derived from adapter defaults, tenant settings, credentials health, and runtime checks.
- [ ] Add mapping service that converts external statuses/events into canonical fulfillment events and step updates.
- [ ] Add mapping precedence: system default, provider version, tenant override, integration override.
- [ ] Add unmapped-status handling that creates `external_status_unmapped` observations and warning/blocker records depending on whether fulfillment depends on the status.
- [ ] Add evidence records for API responses, webhooks, pollers, feeds, CSV imports, supplier portal updates, operator entries, email parsing, and system inference.
- [ ] Add capability gap creation for missing required fields, unsupported operations, degraded credentials, stale feeds, and manual-only flows.
- [ ] Add capability gaps when stock propagation depends on unavailable or stale provider behavior: missing supplier freshness, unsupported marketplace stock write, no acknowledgement, stale channel stock, or manual override active.
- [ ] Add tests for supplier feed with no order API, supplier portal confirmation, marketplace custom status mapping, carrier without tracking API, and unmapped status blocker.
- [ ] Add tests for automatic stock decrease after marketplace order import, automatic stock increase after warehouse receipt, supplier availability stale blocking stock exposure, and manual override preventing automatic overwrite.

Validation:

- `cd public/apps/api-server && go test ./internal/model ./internal/repository ./internal/service -run 'Capability|Mapping|Observation|Gap' -count=1`

Acceptance criteria:

- OpenOMS can explain what each integration can and cannot do.
- Unsupported provider behavior is not treated as a system error.
- Missing mappings and missing data become explicit warnings or blockers.
- Orchestrator decisions can consume capability profiles without provider-specific branching.
- Stock availability decisions can consume capability profiles without assuming every provider supports bidirectional stock automation.

## Phase 3: Orchestrator Service Core

**Goal:** Introduce the service boundary that owns fulfillment process creation, updates, blockers, and events.

**Files:**

- Create: `apps/api-server/internal/service/fulfillment_orchestrator.go`
- Create: `apps/api-server/internal/service/fulfillment_commands.go`
- Create: `apps/api-server/internal/service/fulfillment_policy_service.go`
- Create: `apps/api-server/internal/service/fulfillment_read_service.go`
- Add tests in: `apps/api-server/internal/service/fulfillment_orchestrator_test.go`

Tasks:

- [ ] Add `CreateProcessForOrder`.
- [ ] Add `ValidateOrder`.
- [ ] Add `AllocateUnits`.
- [ ] Add capability-aware routing so each unit knows whether the next step is automatic, manual, unsupported, degraded, or blocked.
- [ ] Add `CreateBlocker`.
- [ ] Add `ResolveBlocker`.
- [ ] Add `AdvanceStep`.
- [ ] Add `RetryStep`.
- [ ] Add `AppendEvent`.
- [ ] Add `EnqueueOutboxEvent`.
- [ ] Add aggregate status recalculation from units, steps, and blockers.
- [ ] Add canonical availability recalculation for reservation, release, warehouse receipt, return-to-stock, supplier availability refresh, dropship preflight, and manual inventory correction.
- [ ] Add default automatic stock propagation policy with per tenant, product, listing, channel, and supplier-source overrides for `automatic`, `manual`, `paused`, and `override_quantity`.
- [ ] Add read service for process detail, order fulfillment summary, and operations summary.
- [ ] Add tests for warehouse-only, dropship-only, mixed, missing-data blocked, manual-only supplier, unsupported carrier tracking, stock propagation blockers, manual stock override, and resolved-blocker resumed flows.

Validation:

- `cd public/apps/api-server && go test ./internal/service -run Fulfillment -count=1`

Acceptance criteria:

- The service can model the full process without calling external providers.
- The service can model a provider that has feed/manual data only and no API.
- Blockers always affect health status.
- Resolving a blocker resumes the correct step.

## Phase 4: Transactional Outbox Worker

**Goal:** Replace invisible fire-and-forget side effects with durable, claimable, retryable outbox work.

**Files:**

- Create: `apps/api-server/internal/worker/orchestration_worker.go`
- Create: `apps/api-server/internal/service/orchestration_dispatcher.go`
- Modify: `apps/api-server/internal/worker/manager.go`
- Modify: `apps/api-server/internal/config/config.go`
- Add tests for worker claim/retry/failure behavior.

Tasks:

- [ ] Add outbox worker loop with bounded batch size.
- [ ] Add claim timeout and retry policy.
- [ ] Add permanent failure transition that creates a fulfillment blocker.
- [ ] Add per-action dispatcher interface.
- [ ] Add idempotency key propagation to provider calls.
- [ ] Add graceful shutdown behavior.
- [ ] Add metrics for outbox lag, attempts, success, failure, and retry.
- [ ] Add Sentry capture for unexpected panics only, with redacted metadata.
- [ ] Add tests for retryable failure, non-retryable failure, duplicate idempotency key, and worker restart.

Validation:

- `cd public/apps/api-server && go test ./internal/worker ./internal/service -run Orchestration -count=1`

Acceptance criteria:

- Side effects are not lost after process restart.
- Repeated worker execution does not duplicate provider actions.
- Failed side effects are visible as process events and blockers.

## Phase 5: Unify Order Creation And Marketplace Import

**Goal:** Ensure every order, regardless of source, creates the same fulfillment process and events.

**Files:**

- Modify: `apps/api-server/internal/service/order_service.go`
- Modify: `apps/api-server/internal/worker/marketplace_order_poller.go`
- Modify: `apps/api-server/internal/service/import_service.go`
- Modify: `apps/api-server/internal/service/allegro_import_service.go`
- Add tests for manual/API/marketplace/import paths.

Tasks:

- [ ] Route manual/API order creation through process creation.
- [ ] Replace direct marketplace order repository insertion side effects with a domain command path.
- [ ] Ensure `order.created` process event is emitted for every source.
- [ ] Record marketplace payloads and status values as redacted `integration_observations` before canonical mapping.
- [ ] Route marketplace external statuses through `external_status_mappings` before changing process or order state.
- [ ] Create a mapping blocker when a marketplace/shop custom status is required for automation but not mapped.
- [ ] Ensure stock reservation is modeled as a fulfillment step.
- [ ] Ensure order-created stock reservation decreases canonical `available_to_sell` and enqueues outbound stock propagation for every eligible marketplace/shop listing.
- [ ] Ensure stock release, return-to-stock, warehouse document confirmation, purchase order receipt, and inventory correction increase canonical `available_to_sell` and enqueue outbound stock propagation for every eligible marketplace/shop listing.
- [ ] Ensure stock propagation writes durable attempts, evidence, retry state, and typed blockers instead of relying on best-effort worker logs.
- [ ] Preserve manual stock mode and active override quantities so automatic propagation does not overwrite operator decisions without explicit tenant policy.
- [ ] Ensure auto-create shipment is represented as a step/outbox command.
- [ ] Add feature flag for process creation if rollout needs staged enablement.
- [ ] Backfill active orders into fulfillment processes after schema is deployed.

Validation:

- `cd public/apps/api-server && go test ./internal/service ./internal/worker -run 'Order|Marketplace|Fulfillment' -count=1`

Acceptance criteria:

- Manual, API, import, and marketplace-created orders produce equivalent process records.
- Marketplace-specific statuses do not bypass the mapping layer.
- Existing order APIs keep response compatibility.
- Marketplace duplicates remain protected by external ID uniqueness.

## Phase 6: Shipment, Label, And Tracking Integration

**Goal:** Make shipments and labels first-class orchestration steps with consistent events and retry behavior.

**Files:**

- Modify: `apps/api-server/internal/service/shipment_service.go`
- Modify: `apps/api-server/internal/service/label_service.go`
- Modify: `apps/api-server/internal/worker/tracking_poller.go`
- Modify: `public/packages/order-engine/status.go`
- Modify: `public/packages/order-engine/transitions.go`
- Add tests for label and tracking transitions.

Tasks:

- [ ] Model `create_shipment` as a fulfillment step.
- [ ] Model `generate_label` as a fulfillment step.
- [ ] Stop setting `label_ready` without process event emission.
- [ ] Decide whether `generating_label` remains an internal transient shipment state or moves entirely into step state.
- [ ] Ensure label failures create typed blockers.
- [ ] Ensure tracking webhook and tracking poller use the same transition path.
- [ ] Resolve carrier tracking and dispatch-order capabilities before creating tracking/dispatch steps.
- [ ] Treat unsupported carrier tracking as an explicit unsupported capability event, not as a failure.
- [ ] Treat configured but failing carrier tracking as degraded/system-error according to capability profile and last failure.
- [ ] Ensure order delivery completion is derived from all relevant fulfillment units.
- [ ] Add provider attempt rows for label creation, label download, tracking sync, and marketplace tracking sync.

Validation:

- `cd public/apps/api-server && go test ./internal/service ./internal/worker -run 'Shipment|Label|Tracking|Fulfillment' -count=1`

Acceptance criteria:

- Label generation success and failure are visible in process timeline.
- Tracking changes are consistent whether they come from pollers or webhooks.
- Carriers without tracking APIs produce clear dashboard messaging instead of noisy worker failures.
- Dashboard can show label/tracking issues without parsing raw shipment state.

## Phase 7: Warehouse, Pick-Pack, Dropship, And Backorder Units

**Goal:** Represent physical fulfillment as units and steps instead of ad-hoc order status changes.

**Files:**

- Modify: `apps/api-server/internal/service/pick_pack_service.go`
- Modify: `apps/api-server/internal/service/barcode_service.go`
- Modify: `apps/api-server/internal/service/dropship_service.go`
- Modify: `apps/api-server/internal/service/purchase_order_service.go`
- Modify: `apps/api-server/internal/service/warehouse_document_service.go`
- Add tests for unit aggregation.

Tasks:

- [ ] Convert pick-pack session progress into fulfillment step updates.
- [ ] Remove dependency on `packed` as an order status for process truth.
- [ ] Model dropship orders as `dropship` fulfillment units.
- [ ] Model supplier confirmation, rejection, and tracking as unit events.
- [ ] Resolve supplier capabilities before selecting dropship automation: API order, feed-only, supplier portal, email/manual, or unsupported.
- [ ] Create manual/supplier-portal steps when a supplier cannot accept orders through API.
- [ ] Create capability gaps when supplier feed lacks stock, price, lead time, confirmation, or tracking fields required by tenant policy.
- [ ] Model supplier availability as external availability, not owned warehouse stock: raw `source_quantity`, policy-adjusted `available_to_sell`, freshness window, safety buffer, lead time, reservation support, and preflight requirement.
- [ ] Block automatic dropship routing or marketplace stock increase when supplier availability is stale, unknown, below safety buffer, or not reservable under tenant policy.
- [ ] Model purchase order waiting as `backorder` fulfillment unit state.
- [ ] Resume warehouse fulfillment when purchase order receipt creates available stock.
- [ ] Add mixed-order aggregation logic.

Validation:

- `cd public/apps/api-server && go test ./internal/service -run 'PickPack|Barcode|Dropship|Purchase|Warehouse|Fulfillment' -count=1`

Acceptance criteria:

- Mixed warehouse/dropship/backorder orders show correct aggregate state.
- Supplier portal updates and supplier API updates feed the same process timeline.
- Feed-only suppliers remain usable with explicit manual steps and visible missing capabilities.
- Operators can see exactly which line/unit is blocked or waiting.

## Phase 8: Automation V2

**Goal:** Make automation reliable, versioned, observable, and safe for critical state changes.

**Files:**

- Modify: `apps/api-server/internal/automation/engine.go`
- Modify: `apps/api-server/internal/automation/actions.go`
- Modify: `apps/api-server/internal/service/automation_service.go`
- Modify: `apps/api-server/internal/model/automation.go`
- Modify: `apps/api-server/internal/model/workflow.go`
- Modify: `apps/dashboard/src/app/(dashboard)/workflows/`
- Add tests for versioning, dry-run, conflict detection, and command routing.

Tasks:

- [ ] Add rule versioning.
- [ ] Add execution attempts linked to fulfillment process/step where applicable.
- [ ] Add dry-run against captured fulfillment event snapshots.
- [ ] Add conflict detection for multiple rules trying to advance the same critical state.
- [ ] Route `set_status` and other state-changing actions through orchestrator commands.
- [ ] Keep notification/webhook actions as outbox-backed side effects.
- [ ] Add typed action registry with allowed inputs.
- [ ] Add policy distinction between system policies and tenant custom rules.

Validation:

- `cd public/apps/api-server && go test ./internal/automation ./internal/service -run Automation -count=1`
- `cd public/apps/dashboard && npm test -- workflows`

Acceptance criteria:

- Automation failures are visible in process timeline.
- Critical state changes are auditable and idempotent.
- Existing workflow builder remains usable while gaining safer backend semantics.

## Phase 9: Operations API And Dashboard Read Model

**Goal:** Replace heuristic operations dashboard data with process-backed summaries and exceptions.

**Files:**

- Create: `apps/api-server/internal/handler/fulfillment_handler.go`
- Create: `apps/api-server/internal/handler/operations_handler.go`
- Modify: `apps/api-server/internal/router/router.go`
- Modify: `apps/dashboard/src/types/api.ts`
- Modify: `apps/dashboard/src/hooks/use-operations-dashboard.ts`
- Modify: `apps/dashboard/src/lib/operations-dashboard.ts`
- Modify: `apps/dashboard/src/components/dashboard/orchestration-map.tsx`
- Modify: `apps/dashboard/src/components/dashboard/operational-exceptions.tsx`
- Modify: `apps/dashboard/src/app/(dashboard)/page.tsx`

Tasks:

- [ ] Add process list endpoint.
- [ ] Add process detail endpoint.
- [ ] Add order fulfillment endpoint.
- [ ] Add blocker resolve endpoint.
- [ ] Add step retry endpoint.
- [ ] Add operations summary endpoint.
- [ ] Add operations exceptions endpoint.
- [ ] Add integration capability summary endpoint for automation coverage, manual steps, unsupported capabilities, stale data, and missing mappings.
- [ ] Add integration detail view section showing “co umiemy zrobić automatycznie”, “co wymaga ręcznie”, and “czego brakuje”.
- [ ] Update dashboard hook to consume process-backed summaries.
- [ ] Update dashboard components to show high-level status and action-required items.
- [ ] Add drilldown from dashboard exception to order fulfillment detail.
- [ ] Add Playwright coverage for blocked, waiting external, and completed flows.

Validation:

- `cd public/apps/api-server && go test ./internal/handler -run 'Fulfillment|Operations' -count=1`
- `cd public/apps/dashboard && npm test -- operations-dashboard`
- `cd public/apps/dashboard && npx playwright test operations-dashboard.spec.ts`

Acceptance criteria:

- Dashboard no longer guesses operational problems from scattered statuses.
- Operators can see and act on blockers.
- Operators can distinguish system errors from known integration limitations.
- Existing dashboard remains clear and not overloaded with internal details.

## Phase 10: Observability And Operational Hardening

**Goal:** Make orchestration measurable, debuggable, and alertable.

**Files:**

- Modify: `apps/api-server/internal/middleware/metrics.go`
- Create: `apps/api-server/internal/observability/fulfillment_metrics.go`
- Modify: `apps/api-server/internal/asyncutil/safego.go` if remaining async paths require contextual logging.
- Modify: enterprise monitoring dashboards/alerts where applicable.
- Update: `docs/system-documentation.md`

Tasks:

- [ ] Add fulfillment process metrics.
- [ ] Add fulfillment step metrics.
- [ ] Add blocker metrics.
- [ ] Add capability gap metrics.
- [ ] Add unmapped external status metrics.
- [ ] Add stale integration observation metrics.
- [ ] Add outbox lag metrics.
- [ ] Add provider error metrics.
- [ ] Add structured log fields for process, step, attempt, provider, and correlation IDs.
- [ ] Add redaction tests for event payloads and logs.
- [ ] Add alerts for sustained outbox lag, blocker spikes, provider failure spikes, unmapped status spikes, stale integration observations, and label failure spikes.
- [ ] Add runbook entries for retrying blocked processes and diagnosing provider incidents.

Validation:

- `cd public/apps/api-server && go test ./internal/middleware ./internal/observability ./internal/service -count=1`
- Query `/metrics` in a local dev run and confirm new metric names appear after test traffic.

Acceptance criteria:

- Operators can diagnose stuck fulfillment without database spelunking.
- Alerts point to process IDs, not only pod logs.
- Sensitive data is not exposed in logs, metrics, or Sentry events.

## Phase 11: Supabase Operations And Data Lifecycle

**Goal:** Align orchestration persistence with Supabase production operations.

**Files:**

- Modify: `enterprise/docs/runbooks/backup-restore.md`
- Modify: `public/docs/system-documentation.md`
- Modify: `.Codex/context/DECISIONS.md`
- Modify: `.Codex/context/SECURITY_POSTURE.md`

Tasks:

- [ ] Document production connection modes for app, worker, migration, backup, and restore.
- [ ] Document Supabase PITR expectations and OpenOMS off-site backup validation.
- [ ] Add restore validation queries for fulfillment tables.
- [ ] Add connection pool sizing notes for orchestrator workers.
- [ ] Add migration safety notes for large fulfillment tables.
- [ ] Add retention policy for process events and provider attempts.
- [ ] Decide whether old process events are archived to cheaper storage after the operational window.

Validation:

- Review runbook against current Supabase docs.
- Run backup restore test in isolated environment before declaring the flow operational.

Acceptance criteria:

- Production data remains managed by Supabase.
- Restore runbooks include fulfillment data.
- Connection strategy is explicit and avoids pool exhaustion.

## Phase 12: Backfill And Rollout

**Goal:** Move production safely from legacy scattered state to process-backed orchestration.

**Files:**

- Create: `apps/api-server/internal/service/fulfillment_backfill_service.go`
- Create: `apps/api-server/internal/worker/fulfillment_backfill_worker.go`
- Create migration or command for backfill tracking if needed.
- Modify release and runbook docs.

Tasks:

- [ ] Add idempotent backfill for active orders.
- [ ] Create fulfillment processes for orders in non-terminal states.
- [ ] Create units from existing shipments, dropship orders, purchase orders, and pick-pack sessions.
- [ ] Create integration capability profiles for existing integrations and suppliers from adapter defaults, current settings, feed format, portal flag, and credentials status.
- [ ] Create initial external status mappings for currently supported providers.
- [ ] Create blockers for known failed shipments, invalid integration state, and on-hold orders.
- [ ] Create capability gaps for known unsupported or manual-only flows without marking them as system errors.
- [ ] Add dry-run mode that reports counts without writing.
- [ ] Add batch mode with tenant and order range filters.
- [ ] Add verification report comparing legacy dashboard counts to process-backed counts.
- [ ] Enable process-backed dashboard after parity threshold is met.
- [ ] Keep rollback path by preserving legacy status-driven dashboard until cutover is validated.

Validation:

- Backfill dry-run on staging data.
- Backfill write-run on staging data.
- Compare process-backed summaries to known active orders and shipments.

Acceptance criteria:

- Backfill can be resumed safely.
- No duplicate processes are created.
- Dashboard cutover is based on measured parity.

## Phase 13: n8n Custom Action Connector

**Goal:** Add n8n as a safe extension point without giving it ownership of core fulfillment state.

**Files:**

- Create: `apps/api-server/internal/service/external_workflow_service.go`
- Create: `apps/api-server/internal/handler/external_workflow_callback_handler.go`
- Modify: `apps/api-server/internal/automation/actions.go`
- Modify: `enterprise/docs/runbooks/n8n-automation-hub.md`
- Modify network policies if n8n is deployed in-cluster.

Tasks:

- [ ] Add signed outbound event action.
- [ ] Add callback endpoint with HMAC verification.
- [ ] Add timeout behavior that creates warning or blocker according to policy.
- [ ] Add RBAC-scoped token model for n8n callbacks.
- [ ] Store n8n execution ID in `orchestration_attempts`.
- [ ] Add redacted payload templates.
- [ ] Add audit entries for callback-driven actions.
- [ ] Add network policy allowing n8n to call only approved OpenOMS endpoints.

Validation:

- Local integration test with a fake n8n webhook.
- Callback signature tests.
- Timeout and retry tests.

Acceptance criteria:

- n8n can enrich customer workflows.
- n8n cannot mutate fulfillment truth directly.
- n8n failures are visible and recoverable.

## Phase 14: End-To-End Validation

**Goal:** Prove the target architecture works across realistic fulfillment scenarios.

Scenarios:

- [ ] Warehouse order with complete data, successful label and delivery.
- [ ] Carrier-label class requires phone, selected carrier representative is missing phone, operator resolves blocker, label succeeds.
- [ ] Marketplace order imported and fulfilled without losing automation events.
- [ ] Marketplace custom status is unmapped and blocks automation until mapping is added.
- [ ] Bulk status change does not bypass process truth.
- [ ] Dropship order accepted by supplier and tracking synced to marketplace.
- [ ] Dropship order rejected by supplier and surfaced as blocker.
- [ ] Supplier feed-only integration creates manual supplier confirmation step instead of system error.
- [ ] Carrier without tracking API produces unsupported tracking capability event and clear dashboard message.
- [ ] Mixed order with warehouse and dropship units.
- [ ] Backorder waits for purchase receipt and resumes warehouse fulfillment.
- [ ] Carrier label generation fails, retries, then creates blocker.
- [ ] Tracking webhook and poller produce the same final process state.
- [ ] Automation rule changes status through orchestrator command.
- [ ] n8n custom action fails and is surfaced without corrupting process state.

Validation:

- Go integration tests for services and workers.
- Playwright tests for operations dashboard and order detail.
- Load test for outbox and dashboard queries.
- Restore test for fulfillment tables.
- Security review for RLS and privileged worker paths.

Acceptance criteria:

- The process timeline explains every state transition.
- The dashboard clearly shows action-required work.
- Provider failures are retryable, auditable, and visible.
- No core state is owned by Redis, n8n, or local databases.

## Rollback Strategy

- New tables are additive.
- Existing order and shipment APIs remain compatible during rollout.
- Dashboard can remain on legacy data until process-backed parity is proven.
- Outbox worker can be disabled without deleting process state.
- Backfill is idempotent and resumable.
- Legacy automation rules remain readable while state-changing actions are migrated to orchestrator commands.

## Documentation Updates Required

- `.Codex/context/API_CONTRACTS.md` — new fulfillment and operations endpoints.
- `.Codex/context/DOMAIN_MODEL.md` — new process, unit, step, event, blocker, outbox, attempt, policy, capability, mapping, observation, and gap tables.
- `.Codex/context/DECISIONS.md` — decision to keep OpenOMS as orchestrator, Supabase Postgres as source of truth, and capability profiles as the integration abstraction.
- `.Codex/context/SECURITY_POSTURE.md` — RLS, worker role, event/observation redaction, n8n callback boundaries.
- `.Codex/context/PROJECT_STATE.md` — implementation progress and rollout state.
- `docs/system-documentation.md` — orchestration architecture, workers, API surface, operational flows.

## Self-Review

- **Spec coverage:** The plan maps to the design sections: process model, Supabase source of truth, integration capability/mapping/evidence, outbox, reliability, warehouse, dropship, backorder, mixed fulfillment, automation, dashboard, observability, security, migration, and n8n connector.
- **No local production database dependency:** Production state stays on Supabase PostgreSQL. Local PostgreSQL appears only for development and isolated validation.
- **No tactical shortcuts:** The plan is additive, durable, and migration-safe; it does not use n8n, Redis, or ad-hoc statuses as fulfillment truth.
- **Type consistency:** Process, unit, step, event, blocker, outbox, attempt, and policy terminology is consistent with the design document.
- **Risk controls:** RLS, idempotency, outbox retries, blocker creation, capability gaps, status mapping, evidence redaction, audit, pool sizing, backfill, and rollback are all explicitly covered.
- **Scope check:** This is larger than one implementation issue. It is intentionally structured as epics and phases, with each phase suitable for its own detailed TDD task plan before code changes.
