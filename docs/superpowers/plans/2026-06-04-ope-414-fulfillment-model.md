# OPE-414 Fulfillment Process Data Model, RLS & Repository Foundation

> **For agentic workers:** REQUIRED SUB-SKILL: superpowers:subagent-driven-development / executing-plans.

**Goal:** The canonical, tenant-scoped fulfillment data model (processes / units / steps / blockers) + RLS + a repository foundation, per ADR-424. This is the base the orchestration outbox/worker (OPE-415+) builds on. No service/handlers/orchestration yet.

**Architecture:** Four tenant-scoped tables with `FORCE ROW LEVEL SECURITY` + `tenant_isolation` policy (like orders/shipments), a Go model with the canonical enums/step-keys/blocker-codes + validators, and a `FulfillmentRepository` whose methods take a `pgx.Tx` and run inside `database.WithTenant` (the established tenant-scoped repo pattern). Tenant isolation is proven by a DB-bound integration test (the OPE-507 RLS pattern).

**Tech Stack:** Go 1.25, pgx/v5, golang-migrate, PostgreSQL RLS, testify.

---

## Scope

In scope: migration (4 tables + RLS + dual-role grants), `model/fulfillment.go` (structs + enums + validators), `repository/fulfillment_repository.go` (tx-based CRUD: processes/units/steps/blockers), DB-bound integration test (RLS cross-tenant isolation + CRUD round-trip).

Out of scope (OPE-415+): orchestration outbox/attempts/idempotent worker (OPE-415), routing order creation through fulfillment commands (OPE-416), shipment/tracking integration (OPE-417), warehouse/pick-pack/dropship units execution (OPE-418), operations API + dashboard read model (OPE-419), inventory availability model + stock propagation events (later — ADR-424 §170/§206), external_status_mappings runtime (uses OPE-407 mappings). The repo is the foundation; it is wired into bootstrap by OPE-415.

## Canonical model (ADR-424)

- process aggregate_status: `new|validating|ready|in_progress|waiting_external|blocked|completed|cancelled`
- process health_status (independent): `ok|warning|action_required|system_error`
- unit_type: `warehouse|dropship|backorder|mixed_child|manual`
- unit/step status: `pending|ready|running|waiting_external|blocked|succeeded|failed|cancelled|skipped`
- 22 step keys (validate_order … close_process) — validated in Go (extensible; no DB CHECK)
- blocker codes (12) with categories `integration|supplier|operator|capability|mapping` — validated in Go
- blocker status: `open|acknowledged|resolved`

## Data model (migration 000036) — TENANT-SCOPED, RLS

```sql
CREATE TABLE public.fulfillment_processes (
    id uuid PRIMARY KEY DEFAULT public.uuid_generate_v4(),
    tenant_id uuid NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    order_id uuid NOT NULL REFERENCES public.orders(id) ON DELETE CASCADE,
    aggregate_status text NOT NULL DEFAULT 'new'
        CHECK (aggregate_status IN ('new','validating','ready','in_progress','waiting_external','blocked','completed','cancelled')),
    health_status text NOT NULL DEFAULT 'ok'
        CHECK (health_status IN ('ok','warning','action_required','system_error')),
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_fulfillment_processes_order ON public.fulfillment_processes (tenant_id, order_id); -- migrate:index-lock-ok

CREATE TABLE public.fulfillment_units (
    id uuid PRIMARY KEY DEFAULT public.uuid_generate_v4(),
    tenant_id uuid NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    process_id uuid NOT NULL REFERENCES public.fulfillment_processes(id) ON DELETE CASCADE,
    parent_unit_id uuid REFERENCES public.fulfillment_units(id) ON DELETE SET NULL,
    unit_type text NOT NULL CHECK (unit_type IN ('warehouse','dropship','backorder','mixed_child','manual')),
    status text NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending','ready','running','waiting_external','blocked','succeeded','failed','cancelled','skipped')),
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_fulfillment_units_process ON public.fulfillment_units (tenant_id, process_id); -- migrate:index-lock-ok

CREATE TABLE public.fulfillment_steps (
    id uuid PRIMARY KEY DEFAULT public.uuid_generate_v4(),
    tenant_id uuid NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    unit_id uuid NOT NULL REFERENCES public.fulfillment_units(id) ON DELETE CASCADE,
    step_key text NOT NULL,
    status text NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending','ready','running','waiting_external','blocked','succeeded','failed','cancelled','skipped')),
    attempts integer NOT NULL DEFAULT 0,
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_fulfillment_steps_unit ON public.fulfillment_steps (tenant_id, unit_id); -- migrate:index-lock-ok

CREATE TABLE public.fulfillment_blockers (
    id uuid PRIMARY KEY DEFAULT public.uuid_generate_v4(),
    tenant_id uuid NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    process_id uuid NOT NULL REFERENCES public.fulfillment_processes(id) ON DELETE CASCADE,
    unit_id uuid REFERENCES public.fulfillment_units(id) ON DELETE CASCADE,
    code text NOT NULL,
    category text NOT NULL,
    status text NOT NULL DEFAULT 'open' CHECK (status IN ('open','acknowledged','resolved')),
    description text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    resolved_at timestamptz
);
CREATE INDEX idx_fulfillment_blockers_process ON public.fulfillment_blockers (tenant_id, process_id, status); -- migrate:index-lock-ok
```
Each table: `ENABLE` + `FORCE ROW LEVEL SECURITY` + `CREATE POLICY tenant_isolation ... USING (tenant_id = (current_setting('app.current_tenant_id'::text, true))::uuid)` (verbatim match of the existing orders/shipments policy). Dual-role grants DO-block (openoms_app/openoms). `.down.sql`: drop blockers, steps, units, processes.

## Implementation Tasks (TDD)

### Task 1: Migration + model
- [ ] `migrations/000036_fulfillment_model.up.sql`/`.down.sql`.
- [ ] `internal/model/fulfillment.go`: `FulfillmentProcess`, `FulfillmentUnit`, `FulfillmentStep`, `FulfillmentBlocker` structs; enum constants; `validFulfillmentStepKeys` + `validBlockerCodes` (with `BlockerCategory(code) string`); `IsValid{AggregateStatus,HealthStatus,UnitType,UnitStatus,StepKey,StepStatus,BlockerCode,BlockerStatus}`.
- [ ] `model/fulfillment_test.go`: validator + step-key + blocker-category matrices.

### Task 2: Repository (tenant-scoped, tx-based — AuditRepository pattern)
- [ ] `repository/fulfillment_repository.go`: `FulfillmentRepository` (no pool field; methods take `tx pgx.Tx`, run inside `database.WithTenant`):
  - `CreateProcess(ctx, tx, p) (*FulfillmentProcess, error)`, `GetProcess`, `GetProcessByOrder`, `ListProcesses`, `UpdateProcessStatus(id, aggregate, health)`.
  - `CreateUnit`, `ListUnits(processID)`, `UpdateUnitStatus`.
  - `CreateStep`, `ListSteps(unitID)`, `UpdateStepStatus(id, status, attempts)`.
  - `CreateBlocker`, `ListBlockers(processID)`, `ResolveBlocker(id, status)`.
  - Inserts set tenant_id from the row (RLS enforces it matches the WithTenant context).

### Task 3: Integration test (RLS + CRUD)
- [ ] `tests/integration/fulfillment_test.go` (build tag integration): seed tenantA + tenantB + an order for A; via `database.WithTenant(appPool, tenantA)` create a process → unit → step → blocker; read them back; update statuses. Then `WithTenant(appPool, tenantB)` GetProcessByOrder → pgx.ErrNoRows / empty (RLS isolation); A sees it. Cross-tenant create rejected by RLS (insert with mismatched context fails). Mirrors the OPE-507 RLS test.

### Task 4: Validate + docs
- [ ] go test ./..., vet, gofmt, golangci-lint (full); integration green; migrate down/up.
- [ ] `docs/system-documentation.md` — fulfillment model section.

## Risks
- RLS policy must match the existing pattern exactly (`current_setting(...,true)::uuid`); cross-tenant writes/reads must be blocked — proven by the integration test.
- Step keys / blocker codes are extensible (Go-validated, no DB CHECK) so OPE-415+ can add without a migration; core state enums are DB-CHECK'd.
- Migration Safety CI: indexes marked index-lock-ok (new empty tables); no "CREATE INDEX" phrase in comments.

## Self-Review
Covers OPE-414: canonical process/unit/step/blocker data model (ADR-424), RLS (FORCE + tenant_isolation), repository foundation (tx-based, WithTenant). Orchestration/inventory/stock-propagation are later tasks. The repo is foundation-only (wired into bootstrap by OPE-415).
