# OPE-334 Public Config Query Cache Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove the module-level mutable cache from `usePublicConfig` and make public config caching explicit through TanStack Query.

**Architecture:** `usePublicConfig` should rely on the app-wide `QueryProvider` instead of a singleton `cachedConfig` variable. The hook returns safe defaults while loading or after fetch failure, while successful responses live in the QueryClient cache and can be invalidated in tests/HMR.

**Tech Stack:** Next.js 16, React 19, TypeScript, TanStack Query, Vitest, Testing Library React.

---

## Files

- Modify: `apps/dashboard/src/hooks/use-public-config.ts`
  - Replace `useEffect`/`useState` and `cachedConfig` with `useQuery`.
  - Export `publicConfigQueryKey` for invalidation and tests.
- Create: `apps/dashboard/src/hooks/__tests__/use-public-config.test.ts`
  - Add a regression test proving a fresh QueryClient fetches fresh config instead of reusing module state.

## Task 1: Add Regression Test

- [ ] **Step 1: Create the hook test**

Create `apps/dashboard/src/hooks/__tests__/use-public-config.test.ts`:

```ts
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createElement, type ReactNode } from "react";
import { usePublicConfig } from "@/hooks/use-public-config";

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

function jsonResponse(data: unknown) {
  return Promise.resolve(
    new Response(JSON.stringify(data), {
      status: 200,
      headers: { "Content-Type": "application/json" },
    })
  );
}

const fetchMock = vi.fn();

beforeEach(() => {
  vi.stubGlobal("fetch", fetchMock);
  fetchMock.mockReset();
});

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("usePublicConfig", () => {
  it("uses the QueryClient cache instead of module-level stale config", async () => {
    fetchMock
      .mockReturnValueOnce(
        jsonResponse({
          registration_mode: "invite",
          license_enabled: false,
          billing_enabled: true,
          stripe_public_key: "pk_first",
        })
      )
      .mockReturnValueOnce(
        jsonResponse({
          registration_mode: "closed",
          license_enabled: true,
          billing_enabled: false,
        })
      );

    const first = renderHook(() => usePublicConfig(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(first.result.current.registration_mode).toBe("invite");
    });
    first.unmount();

    const second = renderHook(() => usePublicConfig(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(second.result.current.registration_mode).toBe("closed");
    });
    expect(second.result.current.license_enabled).toBe(true);
    expect(second.result.current.billing_enabled).toBe(false);
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });
});
```

- [ ] **Step 2: Verify RED**

Run:

```bash
cd apps/dashboard
npx vitest run src/hooks/__tests__/use-public-config.test.ts --reporter=dot
```

Expected: FAIL because the current `cachedConfig` module variable prevents the second fresh QueryClient from fetching the second response.

## Task 2: Replace Module Cache With Query Cache

- [ ] **Step 1: Implement the hook through TanStack Query**

Update `apps/dashboard/src/hooks/use-public-config.ts`:

```ts
"use client";

import { useQuery } from "@tanstack/react-query";
import { API_URL } from "@/lib/api-client";

interface PublicConfigData {
  registration_mode: "open" | "invite" | "closed" | "disabled";
  license_enabled: boolean;
  billing_enabled: boolean;
  stripe_public_key?: string;
}

interface PublicConfig extends PublicConfigData {
  isLoading: boolean;
}

const defaultConfig: PublicConfigData = {
  registration_mode: "open",
  license_enabled: false,
  billing_enabled: false,
};

export const publicConfigQueryKey = ["public-config"] as const;

async function fetchPublicConfig(): Promise<PublicConfigData> {
  const response = await fetch(`${API_URL}/v1/config/public`, {
    credentials: "include",
  });

  if (!response.ok) {
    throw new Error("Failed to load public config");
  }

  return response.json() as Promise<PublicConfigData>;
}

export function usePublicConfig() {
  const query = useQuery({
    queryKey: publicConfigQueryKey,
    queryFn: fetchPublicConfig,
    staleTime: 60_000,
    retry: false,
  });

  return {
    ...(query.data ?? defaultConfig),
    isLoading: query.isLoading,
  } satisfies PublicConfig;
}
```

- [ ] **Step 2: Verify GREEN**

Run:

```bash
cd apps/dashboard
npx vitest run src/hooks/__tests__/use-public-config.test.ts --reporter=dot
```

Expected: PASS.

## Task 3: Validate And Ship

- [ ] **Step 1: Run focused frontend checks**

Run:

```bash
cd apps/dashboard
npx eslint --quiet src/hooks/use-public-config.ts src/hooks/__tests__/use-public-config.test.ts
npx vitest run src/hooks/__tests__/use-public-config.test.ts --reporter=dot
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
git add apps/dashboard/src/hooks/use-public-config.ts \
  apps/dashboard/src/hooks/__tests__/use-public-config.test.ts \
  docs/superpowers/plans/2026-05-18-ope-334-public-config-query-cache.md
git commit -m "OPE-334: use query cache for public config"
git push -u origin fix/OPE-334-public-config-query-cache
```

## Risk And Rollback

- Risk: auth pages depend on `QueryProvider`; root layout already wraps auth routes in `QueryProvider`, so this remains supported.
- Risk: failed config requests now throw inside React Query. The hook still returns safe default config and `isLoading: false` after the query settles.
- Rollback: revert the PR to restore the previous module-level cache. No backend or database rollback needed.

## Self-Review

- Spec coverage: OPE-334 asks to remove module-level mutable cache; Task 2 removes `cachedConfig` and uses QueryClient cache.
- Placeholder scan: no TBD/TODO placeholders remain.
- Type consistency: `PublicConfigData`, `PublicConfig`, and `publicConfigQueryKey` are defined once in the hook and consumed directly by the test.
