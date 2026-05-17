# OPE-302 Order Update Request Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove the unsafe `as unknown as UpdateOrderRequest` cast from the order edit page and make the update payload explicit.

**Architecture:** Keep `OrderForm` as the shared create/edit form, but add a small mapper that converts its create-shaped submit payload into the narrower `UpdateOrderRequest`. The mapper is pure and unit-tested so future type drift is caught without rendering the whole order detail page.

**Tech Stack:** Next.js 16 dashboard, TypeScript, Vitest.

---

## Files

- Create: `apps/dashboard/src/components/orders/order-request-mappers.ts`
- Create: `apps/dashboard/src/components/orders/__tests__/order-request-mappers.test.ts`
- Modify: `apps/dashboard/src/app/(dashboard)/orders/[id]/page.tsx`

## Tasks

### Task 1: Add failing mapper regression test

- [x] Create `apps/dashboard/src/components/orders/__tests__/order-request-mappers.test.ts`.
- [x] Test that `mapCreateOrderRequestToUpdateOrderRequest` preserves update-supported fields.
- [x] Test that create-only fields are omitted: `source`, `integration_id`, `ordered_at`, `shipment_provider`, `auto_create_shipment`.
- [x] Run:

```bash
(cd apps/dashboard && npx vitest run src/components/orders/__tests__/order-request-mappers.test.ts --reporter=dot)
```

Expected before implementation: test fails because the mapper module does not exist.

### Task 2: Implement mapper

- [x] Create `apps/dashboard/src/components/orders/order-request-mappers.ts`.
- [x] Import `CreateOrderRequest` and `UpdateOrderRequest`.
- [x] Return an object containing only `UpdateOrderRequest` keys.
- [x] Keep values unchanged; do not normalize or drop empty optional values in this mapper.
- [x] Re-run the targeted Vitest test until green.

### Task 3: Use mapper on edit page

- [x] Import `mapCreateOrderRequestToUpdateOrderRequest` in `apps/dashboard/src/app/(dashboard)/orders/[id]/page.tsx`.
- [x] Replace `data as unknown as UpdateOrderRequest` with `mapCreateOrderRequestToUpdateOrderRequest(data)`.
- [x] Remove the now-unused `UpdateOrderRequest` import from the page.
- [x] Verify no `as unknown as UpdateOrderRequest` remains:

```bash
rg -n "as unknown as UpdateOrderRequest|\bUpdateOrderRequest\b" "apps/dashboard/src/app/(dashboard)/orders/[id]/page.tsx"
```

Expected: no matches for the unsafe cast and no unused type import.

### Task 4: Validation

- [x] Run targeted Vitest:

```bash
(cd apps/dashboard && npx vitest run src/components/orders/__tests__/order-request-mappers.test.ts --reporter=dot)
```

- [x] Run dashboard lint:

```bash
(cd apps/dashboard && npm run lint:quiet)
```

- [x] Run repository whitespace check:

```bash
git diff --check
```

- [x] Before push, run full local CI from repo root:

```bash
./scripts/local-ci.sh
```

## Risk And Rollback

- Risk: the edit page stops sending create-only fields that the backend might currently ignore. This is intended, because those fields are not in `UpdateOrderRequest`.
- Risk: if a future update endpoint starts accepting a new editable field, the mapper must be extended with a test.
- Rollback: revert this PR; no API, DB, or migration changes are involved.
