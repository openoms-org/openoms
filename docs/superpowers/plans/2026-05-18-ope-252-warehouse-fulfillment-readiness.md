# OPE-252: Warehouse And Fulfillment Readiness Plan

**Goal:** Validate existing warehouse and fulfillment dashboard modules before client exposure, and keep unverified operational flows hidden from `client-ready`.

**Scope:** Public repository only. This is a readiness/audit pass over existing warehouse, packing and stock-sync surfaces. It does not implement the gated OPE-403 Provider Integration Studio or Fulfillment Orchestration epic, and it does not create new orchestration data models.

**Architecture:** The dashboard readiness registry remains the source of truth for exposure. Existing routes with `controlled`, `verify` or `beta` readiness stay outside `client-ready` until browser smoke evidence and API/data validation exist. This pass records module-level decisions and strengthens regression coverage around logistics/warehouse routes.

**Tech Stack:** Next.js 16 dashboard, TypeScript, Vitest, existing readiness registry in `apps/dashboard/src/lib/readiness.ts`, audit docs under `docs/audit/`.

---

## Current Findings

- `/shipments` is currently `ready`.
- `/carriers` and `/carriers/new` are currently `ready`.
- `/packing` and `/pick-pack` are currently `verify`.
- `/settings/warehouses` is currently `controlled`.
- `/stocktakes` and `/settings/warehouse-documents` are currently `verify`.
- `/stock-sync` is currently `beta`.
- `/settings/inventory` is currently `verify`.
- Existing backend warehouse, warehouse document and stocktake routes are permissioned under warehouse permissions. Pick & Pack API routes still need explicit route-level permissions before client exposure; follow-up OPE-436 tracks that gap.

## Files And Areas

- Modify: `apps/dashboard/src/lib/__tests__/readiness.test.ts`
  - Add explicit regression coverage for warehouse, stocktake, packing, pick-pack, stock-sync and inventory direct route access.
  - Keep shipments/carriers visible in client-ready because they were already made client-ready in the current registry.
- Create: `docs/audit/warehouse-fulfillment-readiness-2026-05-18.md`
  - Record module decisions, missing validation evidence and follow-up gates.
- Modify if the tests reveal a gap: `apps/dashboard/src/lib/readiness.ts`
  - Keep current conservative classifications unless route coverage is missing.
- No intended changes:
  - No OPE-403 child work.
  - No new fulfillment orchestration model.
  - No API endpoint changes.
  - No database migrations.
  - No Helm, workflow, Terraform or production configuration changes.

## Implementation Tasks

### Task 1: Add Warehouse Readiness Regression Tests

- [x] Add direct route assertions in `apps/dashboard/src/lib/__tests__/readiness.test.ts`:
  - `/shipments`, `/shipments/new`, `/carriers`, `/carriers/new` remain accessible in `client-ready`.
  - `/packing`, `/pick-pack`, `/pick-pack/session-1`, `/settings/warehouses`, `/settings/warehouses/warehouse-1`, `/stocktakes`, `/stocktakes/new`, `/stocktakes/stocktake-1`, `/settings/warehouse-documents`, `/settings/warehouse-documents/new`, `/settings/warehouse-documents/doc-1`, `/stock-sync`, `/stock-sync/events`, `/settings/inventory` are not accessible in `client-ready`.
  - The same non-blocked routes are accessible in `full` validation mode.
- [x] Add nav assertions:
  - client-ready logistics/settings nav does not expose packing, pick-pack, warehouses, stocktakes, warehouse documents, stock sync or inventory control.
  - client-ready still exposes shipments and carriers.
- [x] Run:

```bash
cd apps/dashboard
npx vitest run src/lib/__tests__/readiness.test.ts --reporter=dot
```

Expected: readiness tests pass. If a test fails, fix the readiness rule only when the failure exposes an unintended client-ready route.

### Task 2: Write The Warehouse Readiness Matrix

- [x] Create `docs/audit/warehouse-fulfillment-readiness-2026-05-18.md`.
- [x] Include module rows for:
  - shipments,
  - carriers,
  - packing,
  - pick-pack,
  - warehouses,
  - warehouse documents,
  - stocktakes,
  - stock sync,
  - inventory control.
- [x] For each row record:
  - current readiness,
  - client-ready decision,
  - reason,
  - evidence required before exposure,
  - owner/provider input if required.
- [x] Make the boundary explicit: this pass validates existing modules only and does not start OPE-403 orchestration work.

### Task 3: Static Cross-Check

- [x] Confirm nav, settings index and command palette use readiness-filtered routes.
- [x] Confirm direct dashboard routes are protected by `ReadinessRouteGuard`.
- [x] Confirm warehouse pages use auth-aware hooks and do not bypass `apiClient`/`apiFetch`.
- [x] Confirm backend routes use tenant-scoped services and route-level permissions for warehouse/pick-pack/stocktake groups.
- [x] If a concrete broken exposure is found, add a small follow-up issue or fix it if it is inside OPE-252 scope.

### Task 4: Validation

- [x] Run targeted dashboard tests:

```bash
cd apps/dashboard
npx vitest run src/lib/__tests__/readiness.test.ts --reporter=dot
```

- [x] Run dashboard lint:

```bash
cd apps/dashboard
npm run lint:quiet
```

- [x] Run repository diff checks:

```bash
git diff --check
git diff --stat
```

- [x] Before push/PR, run full local CI:

```bash
./scripts/local-ci.sh
```

## Risk And Rollback

- Risk: exposing warehouse or stock-control modules early can let customers run partial stock flows without a validated recovery path.
- Mitigation: keep `verify`, `controlled` and `beta` warehouse surfaces hidden in `client-ready`.
- Risk: this task could be confused with the gated orchestration epic.
- Mitigation: no schema changes, no new orchestration objects, no OPE-403 child work, and no modifications to gated untracked files.
- Rollback: revert the readiness test/doc commit. No data or production configuration rollback should be required.

## Completion Criteria

- OPE-252 has a saved warehouse/fulfillment readiness matrix.
- Readiness tests explicitly cover warehouse, stocktake, packing, pick-pack, stock-sync and inventory routes.
- Hidden modules remain unavailable through navigation, command palette and direct route access in `client-ready`.
- No gated OPE-403 child files or tasks are modified.
- Validation commands are recorded in the PR and Linear comment.
