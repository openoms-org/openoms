# OPE-299 Integrations Query Key Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove the React Query cache split for `/v1/integrations` so integration mutations refresh the operations dashboard integration health immediately.

**Architecture:** Introduce one canonical query-key helper for integration list/detail data and use it wherever the dashboard fetches or invalidates the integration list. The operations dashboard should fetch `/v1/integrations` under the same list key as `useIntegrations`, not under a dashboard-specific key.

**Tech Stack:** Next.js 16, React 19, TypeScript, TanStack React Query v5, Vitest, Testing Library React hooks.

---

## Scope

- Repo: `public`.
- Linear: `OPE-299`.
- Branch: `fix/OPE-299-integrations-query-key`.
- No API, database, Helm, production config, or UI layout changes.

## Files

- Create: `apps/dashboard/src/hooks/integration-query-keys.ts`
  - Owns canonical query keys for integration list/detail data.
- Modify: `apps/dashboard/src/hooks/use-integrations.ts`
  - Reuse canonical query keys for list/detail queries and mutation invalidation.
- Modify: `apps/dashboard/src/hooks/use-operations-dashboard.ts`
  - Use the canonical integration list query key instead of `["integrations", "operations-dashboard"]`.
- Modify: `apps/dashboard/src/hooks/use-onboarding.ts`
  - Reuse the canonical integration list key for the same `/v1/integrations` endpoint.
- Modify: `apps/dashboard/src/hooks/use-store-integrations.ts`
  - Reuse the canonical integration list key for setup mutation invalidation.
- Modify: `apps/dashboard/src/hooks/__tests__/use-operations-dashboard.test.ts`
  - Add a regression test that fails while the dashboard-specific key exists.

## Task 1: Write The Regression Test

- [ ] Add `waitFor` to the Testing Library import in `apps/dashboard/src/hooks/__tests__/use-operations-dashboard.test.ts`:

```ts
import { act, renderHook, waitFor } from "@testing-library/react";
```

- [ ] Extract query-client creation so the test can inspect the cache:

```ts
function createTestQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: 0 },
    },
  });
}

function createWrapper(queryClient = createTestQueryClient()) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return createElement(QueryClientProvider, { client: queryClient }, children);
  };
}
```

- [ ] Add this test inside `describe("useOperationsDashboard", () => { ... })`:

```ts
it("uses the canonical integrations query key for admin integration health", async () => {
  useAuthStore.setState({
    user: {
      id: "admin",
      email: "admin@test.com",
      role: "admin",
      role_id: "admin",
      name: "Admin User",
    },
  });
  apiClientMock.mockResolvedValue([]);
  const queryClient = createTestQueryClient();

  renderHook(() => useOperationsDashboard(), {
    wrapper: createWrapper(queryClient),
  });

  await waitFor(() => {
    expect(apiClientMock).toHaveBeenCalledWith("/v1/integrations");
  });

  expect(
    queryClient.getQueryCache().find({
      queryKey: ["integrations"],
      exact: true,
    })
  ).toBeDefined();
  expect(
    queryClient.getQueryCache().find({
      queryKey: ["integrations", "operations-dashboard"],
      exact: true,
    })
  ).toBeUndefined();
});
```

- [ ] Run the targeted test and verify RED:

```bash
cd /Users/rafs/praca/openoms-dev/public/apps/dashboard
npx vitest run src/hooks/__tests__/use-operations-dashboard.test.ts --reporter=dot
```

Expected: FAIL because the current implementation still creates `["integrations", "operations-dashboard"]`.

## Task 2: Add Canonical Integration Query Keys

- [ ] Create `apps/dashboard/src/hooks/integration-query-keys.ts`:

```ts
export const integrationQueryKeys = {
  all: ["integrations"] as const,
  detail: (id: string) => [...integrationQueryKeys.all, id] as const,
};
```

- [ ] Update `apps/dashboard/src/hooks/use-integrations.ts` imports:

```ts
import { integrationQueryKeys } from "./integration-query-keys";
```

- [ ] Replace list/detail query keys and mutation invalidation keys in `use-integrations.ts`:

```ts
queryKey: integrationQueryKeys.all,
queryKey: integrationQueryKeys.detail(id),
queryClient.invalidateQueries({ queryKey: integrationQueryKeys.all });
queryClient.invalidateQueries({ queryKey: integrationQueryKeys.detail(id) });
```

- [ ] Update `apps/dashboard/src/hooks/use-operations-dashboard.ts` imports:

```ts
import { integrationQueryKeys } from "./integration-query-keys";
```

- [ ] Replace the dashboard-specific key:

```ts
queryKey: integrationQueryKeys.all,
```

- [ ] Update `apps/dashboard/src/hooks/use-onboarding.ts` to import `integrationQueryKeys` and replace `queryKey: ["integrations"]` with `queryKey: integrationQueryKeys.all`.

- [ ] Update `apps/dashboard/src/hooks/use-store-integrations.ts` to import `integrationQueryKeys` and replace each `queryClient.invalidateQueries({ queryKey: ["integrations"] })` with:

```ts
queryClient.invalidateQueries({ queryKey: integrationQueryKeys.all });
```

## Task 3: Verify Targeted Dashboard Behavior

- [ ] Run the previously failing targeted test and verify GREEN:

```bash
cd /Users/rafs/praca/openoms-dev/public/apps/dashboard
npx vitest run src/hooks/__tests__/use-operations-dashboard.test.ts --reporter=dot
```

Expected: PASS.

- [ ] Run related hook tests:

```bash
cd /Users/rafs/praca/openoms-dev/public/apps/dashboard
npx vitest run src/hooks/__tests__/use-integrations.test.ts src/hooks/__tests__/use-onboarding.test.ts src/hooks/__tests__/use-operations-dashboard.test.ts --reporter=dot
```

Expected: PASS.

- [ ] Run targeted lint:

```bash
cd /Users/rafs/praca/openoms-dev/public/apps/dashboard
npx eslint --quiet src/hooks/use-integrations.ts src/hooks/use-operations-dashboard.ts src/hooks/use-onboarding.ts src/hooks/use-store-integrations.ts src/hooks/__tests__/use-operations-dashboard.test.ts
```

Expected: no output and exit code 0.

## Task 4: Self-Review And Final Validation

- [ ] Review the diff:

```bash
cd /Users/rafs/praca/openoms-dev/public
git diff --check
git diff --stat
git diff
```

- [ ] Confirm no production documentation updates are required:
  - No API contract change.
  - No DB/domain/security/deploy change.
  - No user-facing UI copy or workflow change.
  - Plan file is the only docs artifact for this bugfix.

- [ ] Commit the change:

```bash
cd /Users/rafs/praca/openoms-dev/public
git add apps/dashboard/src/hooks/integration-query-keys.ts apps/dashboard/src/hooks/use-integrations.ts apps/dashboard/src/hooks/use-operations-dashboard.ts apps/dashboard/src/hooks/use-onboarding.ts apps/dashboard/src/hooks/use-store-integrations.ts apps/dashboard/src/hooks/__tests__/use-operations-dashboard.test.ts docs/superpowers/plans/2026-05-17-ope-299-integrations-query-key.md
git commit -m "OPE-299: unify integrations query key"
```

- [ ] Run full public local CI on clean `HEAD` before push:

```bash
cd /Users/rafs/praca/openoms-dev/public
./scripts/local-ci.sh
```

Expected: `STATUS=pass` in `/tmp/openoms-local-ci-full-results.txt` for the current clean `HEAD`.

## Risk And Rollback

- Risk: low. The change only unifies React Query cache identity for the same endpoint and data shape.
- Runtime behavior: admin operations dashboard may now reuse list cache from integration pages/onboarding and refresh when integration mutations invalidate the canonical key.
- Rollback: revert the PR commit. No database, API, deployment, or migration rollback is needed.

## Self-Review

- Spec coverage: OPE-299 requires eliminating incompatible cache keys for `/v1/integrations`; Tasks 1 and 2 cover the regression and implementation.
- Placeholder scan: no TODO/TBD/fill-in placeholders.
- Type consistency: `integrationQueryKeys.all` is a readonly tuple accepted by TanStack Query; `detail(id)` preserves the existing `["integrations", id]` shape.
