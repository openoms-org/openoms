# OPE-320 Dashboard Reference Dropdowns Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Stop supplier, warehouse, product, and price-list dropdowns from silently showing only the first 100 records.

**Architecture:** Backend pagination has `MaxLimit = 100`, so the dashboard cannot fix this by requesting a larger page. Add a small shared frontend helper that fetches all pages for reference-option lists in 100-record batches, then expose domain-specific hooks (`useAllProducts`, `useAllSuppliers`, `useAllWarehouses`, `useAllPriceLists`) for dropdown/filter usage.

**Tech Stack:** Next.js 16 dashboard, React Query, TypeScript, Vitest static and unit tests, OpenOMS `apiClient` and `ListResponse<T>`.

---

## Scope

This PR is limited to frontend reference-option list loading. It does not change backend pagination, table pagination, exports, API contracts, or non-dropdown list views.

## Files

- Create: `apps/dashboard/src/hooks/use-all-list-items.ts`
- Create: `apps/dashboard/src/hooks/use-all-list-items.test.ts`
- Create: `apps/dashboard/src/app/(dashboard)/__tests__/reference-dropdown-limits.test.ts`
- Modify: `apps/dashboard/src/hooks/use-products.ts`
- Modify: `apps/dashboard/src/hooks/use-suppliers.ts`
- Modify: `apps/dashboard/src/hooks/use-warehouses.ts`
- Modify: `apps/dashboard/src/hooks/use-price-lists.ts`
- Modify dropdown/filter call-sites that currently call `useProducts`, `useSuppliers`, `useWarehouses`, or `usePriceLists` with `{ limit: 100 }`.

## Risk And Rollback

- Risk: fetching all options can make pages with thousands of reference records heavier. This is the correct short-term behavior because silent missing options are worse than visible loading; future work can replace huge selects with server-search comboboxes.
- Risk: table/list queries must keep normal pagination. The PR only replaces dropdown/reference option hooks, not table data hooks such as stock rows, price-list items, roles, exchange rates, exports, or operational lists.
- Rollback: revert the PR; no API or database changes are involved.

## Task 1: Add Paged Reference List Helper

**Files:**
- Create: `apps/dashboard/src/hooks/use-all-list-items.ts`
- Create: `apps/dashboard/src/hooks/use-all-list-items.test.ts`

- [x] **Step 1: Write failing helper tests**

Test pure pagination behavior with an injected `fetchPage` function:

```ts
const calls: Array<{ limit?: number; offset?: number; search?: string }> = [];
const result = await fetchAllListItems(
  { search: "abc" },
  async (params) => {
    calls.push(params);
    if (params.offset === 0) return { items: [{ id: "1" }], total: 2, limit: 100, offset: 0 };
    return { items: [{ id: "2" }], total: 2, limit: 100, offset: 100 };
  },
);
expect(calls).toEqual([
  { search: "abc", limit: 100, offset: 0 },
  { search: "abc", limit: 100, offset: 100 },
]);
expect(result.items).toEqual([{ id: "1" }, { id: "2" }]);
```

- [x] **Step 2: Verify red**

Run:

```bash
cd apps/dashboard
npx vitest run src/hooks/use-all-list-items.test.ts --reporter=dot
```

Observed: FAIL because the helper did not exist yet.

- [x] **Step 3: Implement helper and hook**

Create:

```ts
export const REFERENCE_LIST_PAGE_LIMIT = 100;

export async function fetchAllListItems<TEntity, TParams extends Record<string, unknown>>(
  params: TParams,
  fetchPage: (params: TParams & { limit: number; offset: number }) => Promise<ListResponse<TEntity>>,
): Promise<ListResponse<TEntity>> { ... }

export function useAllListItems<TEntity, TParams extends Record<string, unknown>>(
  queryKey: readonly unknown[],
  basePath: string,
  params: TParams = {} as TParams,
  queryOptions: Pick<UseQueryOptions<ListResponse<TEntity>>, "enabled"> = {},
) { ... }
```

`useAllListItems` should build query strings with `buildSearchParams` and call `apiClient<ListResponse<TEntity>>()` per page.

## Task 2: Export Domain-Specific All-Option Hooks

**Files:**
- Modify: `apps/dashboard/src/hooks/use-products.ts`
- Modify: `apps/dashboard/src/hooks/use-suppliers.ts`
- Modify: `apps/dashboard/src/hooks/use-warehouses.ts`
- Modify: `apps/dashboard/src/hooks/use-price-lists.ts`

- [x] **Step 1: Add exports**

Add:

```ts
export function useAllProducts(params: ProductListParams = {}) {
  return useAllListItems<Product, ProductListParams>(["products", "all", params], "/v1/products", params);
}
```

Repeat for suppliers, warehouses, and price lists using their existing types and base paths.

## Task 3: Add Static Guard Against Returning To `{ limit: 100 }`

**Files:**
- Create: `apps/dashboard/src/app/(dashboard)/__tests__/reference-dropdown-limits.test.ts`

- [x] **Step 1: Write failing static test**

Scan `src/app/(dashboard)` and `src/components` production `.tsx` files for:

```ts
/use(?:Products|Suppliers|Warehouses|PriceLists)\s*\(\s*\{[^}]*limit:\s*100/
```

Observed before replacements: FAIL with 10 current call-sites.

- [x] **Step 2: Verify red**

Run:

```bash
cd apps/dashboard
npx vitest run 'src/app/(dashboard)/__tests__/reference-dropdown-limits.test.ts' --reporter=dot
```

Observed: FAIL listing the current dropdown/reference files.

## Task 4: Replace Dropdown Call-Sites

**Files:**
- Modify target dashboard pages/components discovered by the static test.

- [x] **Step 1: Replace supplier dropdown queries**

Use `useAllSuppliers()` for supplier selectors/filters that currently do `useSuppliers({ limit: 100 })`.

- [x] **Step 2: Replace warehouse dropdown queries**

Use `useAllWarehouses()` for warehouse selectors/filters that currently do `useWarehouses({ limit: 100 })`.

- [x] **Step 3: Replace product dropdown queries**

Use `useAllProducts()` for product selectors that currently do `useProducts({ limit: 100 })`. Keep table/list and explicit search queries paginated, e.g. purchase-order product search with `limit: 20`.

- [x] **Step 4: Replace price-list dropdown queries**

Use `useAllPriceLists({ active: true })` for customer price-list assignment.

## Task 5: Validate Dashboard Checks

**Files:**
- No additional code changes expected.

- [x] **Step 1: Verify tests green**

Run:

```bash
cd apps/dashboard
npx vitest run src/hooks/use-all-list-items.test.ts 'src/app/(dashboard)/__tests__/reference-dropdown-limits.test.ts' --reporter=dot
```

Observed: PASS.

- [x] **Step 2: Run targeted lint**

Run:

```bash
cd apps/dashboard
npx eslint --quiet src/hooks/use-all-list-items.ts src/hooks/use-all-list-items.test.ts 'src/app/(dashboard)/__tests__/reference-dropdown-limits.test.ts'
```

Observed: PASS.

- [x] **Step 3: Run full pre-push validation**

Run:

```bash
cd <repository-root>
git diff --check
./scripts/local-ci.sh
```

Observed: `git diff --check` produced no output and `./scripts/local-ci.sh` passed all 10 checks in 119s.

## Task 6: Publish PR

**Files:**
- Git metadata only.

- [ ] **Step 1: Commit**

```bash
cd <repository-root>
git add apps/dashboard/src/hooks/use-all-list-items.ts apps/dashboard/src/hooks/use-all-list-items.test.ts 'apps/dashboard/src/app/(dashboard)/__tests__/reference-dropdown-limits.test.ts' apps/dashboard/src/hooks/use-products.ts apps/dashboard/src/hooks/use-suppliers.ts apps/dashboard/src/hooks/use-warehouses.ts apps/dashboard/src/hooks/use-price-lists.ts docs/superpowers/plans/2026-05-18-ope-320-dashboard-reference-dropdowns.md
git add 'apps/dashboard/src/app/(dashboard)' apps/dashboard/src/components
git commit -m "OPE-320: load complete reference dropdown options"
```

- [ ] **Step 2: Push and create PR**

```bash
git push -u origin fix/OPE-320-dashboard-dropdown-list-truncation
gh pr create --title "OPE-320: load complete reference dropdown options" --body-file /tmp/ope-320-pr.md
```

PR body must include:

```md
## Summary
- add a shared all-pages reference list hook
- switch supplier/warehouse/product/price-list dropdowns away from one-page limit:100 calls
- add regression tests for paged fetching and raw limit:100 hook usage

## Test plan
- npx vitest run src/hooks/use-all-list-items.test.ts 'src/app/(dashboard)/__tests__/reference-dropdown-limits.test.ts' --reporter=dot
- npx eslint --quiet ...
- git diff --check
- ./scripts/local-ci.sh

## Docs updated
- [x] docs/superpowers/plans/2026-05-18-ope-320-dashboard-reference-dropdowns.md
- [ ] N/A — no API, DB, workflow, or system documentation changes needed
```

- [ ] **Step 3: Review gate**

Read GitHub checks and CodeRabbit comments. Fix blockers before merge. If CodeRabbit is rate-limited, comment in Linear and continue with the next independent Todo.

## Self-Review

- Spec coverage: OPE-320 calls out supplier, warehouse, product, and price-list dropdown truncation; Tasks 1-4 cover all four with a guard.
- Placeholder scan: no TBD/TODO placeholders.
- Type consistency: uses existing `ListResponse<T>`, domain list param types, React Query, `apiClient`, and `buildSearchParams`.
