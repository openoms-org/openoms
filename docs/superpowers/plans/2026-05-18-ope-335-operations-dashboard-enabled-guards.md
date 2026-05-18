# OPE-335 Operations Dashboard Enabled Guards Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Stop non-admin dashboard sessions from fetching or refetching admin-only operational exception queries.

**Architecture:** Keep `useOperationsDashboard` as the policy boundary for admin-only dashboard exception data. Extend the shared CRUD list hook with a small optional React Query options object so existing list hooks can pass `enabled` without changing existing call sites.

**Tech Stack:** Next.js 16, React 19, TypeScript, TanStack Query, Vitest, Testing Library React.

---

## Files

- Modify: `apps/dashboard/src/hooks/create-crud-hooks.ts`
  - Add an optional second argument to generated `useList` hooks with `enabled?: boolean`.
- Modify: `apps/dashboard/src/hooks/use-operations-dashboard.ts`
  - Pass `enabled: isAdmin` to `useOrders` and `useShipments` exception queries.
  - Exclude those disabled queries from non-admin loading/error/refetch aggregation.
- Modify: `apps/dashboard/src/hooks/__tests__/use-operations-dashboard.test.ts`
  - Make `useOrders` and `useShipments` mocks assert the new options.
  - Update the non-admin regression test to expect no exception-query refetches.

## Task 1: Add Regression Test

- [ ] **Step 1: Capture list hook arguments in the test**

Update the mocks in `apps/dashboard/src/hooks/__tests__/use-operations-dashboard.test.ts` so `useOrders` and `useShipments` are spy functions:

```ts
const useOrdersMock = vi.hoisted(() => vi.fn());
const useShipmentsMock = vi.hoisted(() => vi.fn());

vi.mock("@/hooks/use-orders", () => ({
  useOrders: useOrdersMock,
}));

vi.mock("@/hooks/use-shipments", () => ({
  useShipments: useShipmentsMock,
}));
```

In `beforeEach`, return the existing query-shaped objects:

```ts
useOrdersMock.mockReturnValue({
  data: undefined,
  isLoading: false,
  isError: false,
  refetch: onHoldOrdersRefetchMock,
});
useShipmentsMock.mockReturnValue({
  data: undefined,
  isLoading: false,
  isError: false,
  refetch: failedShipmentsRefetchMock,
});
```

- [ ] **Step 2: Write the failing non-admin assertion**

In `does not fetch or refetch admin-only integrations for non-admin users`, assert disabled options and no exception refetches:

```ts
expect(useOrdersMock).toHaveBeenCalledWith(expect.any(Object), { enabled: false });
expect(useShipmentsMock).toHaveBeenCalledWith(expect.any(Object), { enabled: false });

await act(async () => {
  await result.current.refetch();
});

expect(statsRefetchMock).toHaveBeenCalledTimes(1);
expect(onHoldOrdersRefetchMock).not.toHaveBeenCalled();
expect(failedShipmentsRefetchMock).not.toHaveBeenCalled();
expect(apiClientMock).not.toHaveBeenCalled();
```

- [ ] **Step 3: Verify RED**

Run:

```bash
cd apps/dashboard
npx vitest run src/hooks/__tests__/use-operations-dashboard.test.ts --reporter=dot
```

Expected: FAIL because `useOperationsDashboard` still calls `useOrders(params)` / `useShipments(params)` without `{ enabled: false }`, and non-admin `refetch()` still calls both exception query refetches.

## Task 2: Implement Enabled Guards

- [ ] **Step 1: Extend generated list hooks**

In `apps/dashboard/src/hooks/create-crud-hooks.ts`, import the React Query option type and update `useList`:

```ts
import { useQuery, useMutation, useQueryClient, type UseQueryOptions } from "@tanstack/react-query";

type ListQueryOptions<TEntity> = Pick<
  UseQueryOptions<ListResponse<TEntity>>,
  "enabled"
>;

function useList(params: TParams = {} as TParams, queryOptions: ListQueryOptions<TEntity> = {}) {
  const sp = buildSearchParams(params);
  const qs = sp.toString();

  return useQuery({
    queryKey: [resourceKey, params],
    queryFn: () =>
      apiClient<ListResponse<TEntity>>(
        `${basePath}${qs ? `?${qs}` : ""}`
      ),
    ...queryOptions,
  });
}
```

- [ ] **Step 2: Gate operations exception queries**

In `apps/dashboard/src/hooks/use-operations-dashboard.ts`, pass admin-only enabled guards:

```ts
const onHoldOrdersQuery = useOrders(ON_HOLD_ORDER_PARAMS, { enabled: isAdmin });
const failedShipmentsQuery = useShipments(FAILED_SHIPMENT_PARAMS, { enabled: isAdmin });
```

Then make loading/error/refetch aggregation admin-aware:

```ts
isLoading:
  statsQuery.isLoading ||
  (isAdmin && onHoldOrdersQuery.isLoading) ||
  (isAdmin && failedShipmentsQuery.isLoading) ||
  (isAdmin && integrationsQuery.isLoading),
isError:
  statsQuery.isError ||
  (isAdmin && onHoldOrdersQuery.isError) ||
  (isAdmin && failedShipmentsQuery.isError) ||
  (isAdmin && integrationsQuery.isError),
refetch: async () => {
  const refetches: Promise<unknown>[] = [statsQuery.refetch()];

  if (isAdmin) {
    refetches.push(
      onHoldOrdersQuery.refetch(),
      failedShipmentsQuery.refetch(),
      integrationsQuery.refetch()
    );
  }

  await Promise.all(refetches);
},
```

- [ ] **Step 3: Verify GREEN**

Run:

```bash
cd apps/dashboard
npx vitest run src/hooks/__tests__/use-operations-dashboard.test.ts --reporter=dot
```

Expected: PASS.

## Task 3: Validate And Ship

- [ ] **Step 1: Run focused frontend checks**

Run:

```bash
cd apps/dashboard
npm run lint -- --quiet src/hooks/create-crud-hooks.ts src/hooks/use-operations-dashboard.ts src/hooks/__tests__/use-operations-dashboard.test.ts
npx vitest run src/hooks/__tests__/use-operations-dashboard.test.ts --reporter=dot
```

Expected: both PASS.

- [ ] **Step 2: Run repository checks**

Run from repo root:

```bash
git diff --check
./scripts/local-ci.sh
```

Expected: both PASS before push.

- [ ] **Step 3: Commit and push**

Use a Linear-prefixed commit:

```bash
git add apps/dashboard/src/hooks/create-crud-hooks.ts \
  apps/dashboard/src/hooks/use-operations-dashboard.ts \
  apps/dashboard/src/hooks/__tests__/use-operations-dashboard.test.ts \
  docs/superpowers/plans/2026-05-18-ope-335-operations-dashboard-enabled-guards.md
git commit -m "OPE-335: guard operations exception queries"
git push -u origin fix/OPE-335-operations-dashboard-enabled-guards
```

## Risk And Rollback

- Risk: `createCrudHooks.useList` is shared across dashboard hooks. The change is additive and all existing call sites keep the same first-argument API.
- Risk: non-admin dashboard may stop showing operational exception cards. This is intentional because those exception datasets are admin-only; base dashboard stats still load.
- Rollback: revert the PR to restore previous eager query behavior. No database or backend rollback needed.

## Self-Review

- Spec coverage: OPE-335 asks for missing `enabled` guards on `onHoldOrdersQuery` and `failedShipmentsQuery`; Task 2 implements those guards and admin-aware aggregation.
- Placeholder scan: no TBD/TODO placeholders remain.
- Type consistency: `ListQueryOptions<TEntity>` is used only by generated list hooks and carries only `enabled`, matching current need without adding broad React Query API surface.
