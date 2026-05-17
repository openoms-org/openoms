# OPE-301 React Hooks State Effects Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove the broad `react-hooks/set-state-in-effect` suppressions from dashboard code and keep only small, documented hook-level exceptions where browser-only hydration or editable server-seeded forms require them.

**Architecture:** Treat each suppression by behavior, not by search-and-replace. Derived values become render-time calculations, dialog resets move into close handlers, query-param defaults become lazy `useState` initializers, and repeated "seed editable state from external data" patterns go through one reviewed hook. A Vitest inventory test prevents this class of mass suppression from returning.

**Tech Stack:** Next.js 16 App Router, React 19, TypeScript, ESLint 9 with `eslint-plugin-react-hooks`, Vitest + Testing Library.

---

## File Structure

- Create `apps/dashboard/src/hooks/use-effect-synced-state.ts`
  - Central hook for the two legitimate exception classes: browser-only hydration and editable state seeded from external data.
- Create `apps/dashboard/src/hooks/use-effect-synced-state.test.tsx`
  - Behavioral tests for hydration and external reset behavior.
- Create `apps/dashboard/src/lib/__tests__/react-hooks-suppressions.test.ts`
  - Repository inventory test that fails if component/page files add `react-hooks/set-state-in-effect` suppressions again.
- Modify component/page files currently containing `react-hooks/set-state-in-effect`
  - Replace per-file suppressions with render-time derivation, event-based reset, lazy state initializers, or `useEffectSyncedState`.
- No API, DB, Helm, Terraform, or production configuration changes.

## Root Cause Notes

- The rule is real and active in `apps/dashboard/eslint.config.mjs`.
- Current issue is not "unknown lint rule"; it is many local suppressions hiding different patterns under the same comment.
- Safe categories:
  - Browser storage hydration after mount cannot run during SSR.
  - Editable forms sometimes need a local draft seeded by API data.
- Unsafe categories:
  - Derived values such as trial days left do not need state.
  - Modal reset can run in `onOpenChange(false)` instead of an effect.
  - Query-param defaults can be lazy initial state.

## Tasks

### Task 1: Add Suppression Inventory Test

**Files:**
- Create: `apps/dashboard/src/lib/__tests__/react-hooks-suppressions.test.ts`

- [ ] **Step 1: Write the failing test**

```ts
import { describe, expect, it } from "vitest";
import { readdirSync, readFileSync, statSync } from "node:fs";
import { join, relative } from "node:path";

const dashboardSrc = join(process.cwd(), "src");
const allowedFiles = new Set(["src/hooks/use-effect-synced-state.ts"]);

function walk(dir: string): string[] {
  return readdirSync(dir).flatMap((entry) => {
    const fullPath = join(dir, entry);
    if (entry === "node_modules" || entry === ".next") return [];
    if (statSync(fullPath).isDirectory()) return walk(fullPath);
    return /\.(ts|tsx)$/.test(entry) ? [fullPath] : [];
  });
}

describe("React hooks lint suppressions", () => {
  it("keeps set-state-in-effect suppressions centralized and documented", () => {
    const offenders = walk(dashboardSrc).flatMap((file) => {
      const relativePath = relative(process.cwd(), file);
      const lines = readFileSync(file, "utf8").split("\n");

      return lines.flatMap((line, index) => {
        if (!line.includes("react-hooks/set-state-in-effect")) return [];
        if (allowedFiles.has(relativePath)) return [];
        return [`${relativePath}:${index + 1}`];
      });
    });

    expect(offenders).toEqual([]);
  });
});
```

- [ ] **Step 2: Run RED**

Run:

```bash
cd apps/dashboard
npx vitest run src/lib/__tests__/react-hooks-suppressions.test.ts --reporter=dot
```

Expected: FAIL listing current component/page suppressions.

### Task 2: Add Central Hook For Legitimate Effects

**Files:**
- Create: `apps/dashboard/src/hooks/use-effect-synced-state.ts`
- Create: `apps/dashboard/src/hooks/use-effect-synced-state.test.tsx`

- [ ] **Step 1: Write hook tests**

Test cases:
- `useHydratedValue` renders the fallback first and then uses the browser-only reader after mount.
- `useEffectSyncedState` seeds local editable state and resets when the external reset key changes.
- `useEffectSyncedState` preserves user edits while the reset key is unchanged.

- [ ] **Step 2: Run RED**

Run:

```bash
cd apps/dashboard
npx vitest run src/hooks/use-effect-synced-state.test.tsx --reporter=dot
```

Expected: FAIL because the hook file does not exist yet.

- [ ] **Step 3: Implement minimal hook**

Implementation shape:

```ts
"use client";

import { useEffect, useRef, useState, type Dispatch, type SetStateAction } from "react";

export function useHydratedValue<T>(fallback: T, readValue: () => T): T {
  const readRef = useRef(readValue);
  readRef.current = readValue;
  const [value, setValue] = useState(fallback);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- Browser-only hydration boundary; callers cannot read storage during SSR.
    setValue(readRef.current());
  }, []);

  return value;
}

export function useEffectSyncedState<T>(
  sourceValue: T,
  resetKey: string | number | boolean | null | undefined,
): [T, Dispatch<SetStateAction<T>>] {
  const [value, setValue] = useState(sourceValue);
  const previousResetKey = useRef(resetKey);

  useEffect(() => {
    if (Object.is(previousResetKey.current, resetKey)) return;
    previousResetKey.current = resetKey;
    // eslint-disable-next-line react-hooks/set-state-in-effect -- Editable local draft reset when a new external record/config is loaded.
    setValue(sourceValue);
  }, [resetKey, sourceValue]);

  return [value, setValue];
}
```

- [ ] **Step 4: Run GREEN**

Run:

```bash
cd apps/dashboard
npx vitest run src/hooks/use-effect-synced-state.test.tsx --reporter=dot
```

Expected: PASS.

### Task 3: Replace Simple Derived And Event Cases

**Files:**
- Modify: `apps/dashboard/src/components/subscription-banner.tsx`
- Modify: `apps/dashboard/src/app/(dashboard)/settings/billing/page.tsx`
- Modify: `apps/dashboard/src/app/(dashboard)/products/[id]/listings/page.tsx`
- Modify: modal sections in `apps/dashboard/src/app/(dashboard)/orders/[id]/page.tsx`

- [ ] **Step 1: Replace trial days state with derived values**

Use:

```ts
const daysLeft =
  subscription?.status === "trialing" && subscription.trial_end
    ? computeDaysLeft(subscription.trial_end)
    : null;
```

- [ ] **Step 2: Replace query-param auto-open effect**

Use:

```ts
const [showCreate, setShowCreate] = useState(
  () => searchParams.get("listing") === "new",
);
```

- [ ] **Step 3: Move modal resets into close handlers**

For each affected modal component, add a local reset function and call it from `onOpenChange(false)` instead of `useEffect`.

- [ ] **Step 4: Run targeted lint**

Run:

```bash
cd apps/dashboard
npx eslint --quiet src/components/subscription-banner.tsx 'src/app/(dashboard)/settings/billing/page.tsx' 'src/app/(dashboard)/products/[id]/listings/page.tsx' 'src/app/(dashboard)/orders/[id]/page.tsx'
```

Expected: PASS.

### Task 4: Replace Browser Storage Hydration Cases

**Files:**
- Modify: `apps/dashboard/src/hooks/use-group-expansion.ts`
- Modify: `apps/dashboard/src/app/(dashboard)/page.tsx`
- Modify: `apps/dashboard/src/app/(dashboard)/workflows/new-editor/page.tsx`
- Modify or preserve tests under `apps/dashboard/src/app/(dashboard)/__tests__/layout.test.tsx`

- [ ] **Step 1: Use `useHydratedValue` for storage reads**

Replace mount effects that only read `localStorage` or `sessionStorage` with `useHydratedValue`.

- [ ] **Step 2: Keep writes in normal effects or event handlers**

Persisting already-derived state to storage stays in effects if it does not synchronously derive render state.

- [ ] **Step 3: Run focused tests**

Run:

```bash
cd apps/dashboard
npx vitest run src/hooks/use-effect-synced-state.test.tsx 'src/app/(dashboard)/__tests__/layout.test.tsx' --reporter=dot
```

Expected: PASS.

### Task 5: Replace Editable External Data Sync Cases

**Files:**
- Modify settings pages with config draft state:
  - `apps/dashboard/src/app/(dashboard)/settings/print-templates/page.tsx`
  - `apps/dashboard/src/app/(dashboard)/invoicing/page.tsx`
  - `apps/dashboard/src/app/(dashboard)/settings/vat-oss/page.tsx`
  - `apps/dashboard/src/app/(dashboard)/settings/webhooks/page.tsx`
  - `apps/dashboard/src/app/(dashboard)/settings/custom-fields/page.tsx`
  - `apps/dashboard/src/app/(dashboard)/settings/order-statuses/page.tsx`
  - `apps/dashboard/src/app/(dashboard)/settings/feeds/page.tsx`
  - `apps/dashboard/src/app/(dashboard)/settings/accounting/page.tsx`
- Modify entity edit pages:
  - `apps/dashboard/src/app/(dashboard)/settings/warehouses/[id]/page.tsx`
  - `apps/dashboard/src/app/(dashboard)/settings/roles/[id]/page.tsx`
  - `apps/dashboard/src/app/(dashboard)/settings/automation/[id]/page.tsx`
  - `apps/dashboard/src/app/(dashboard)/workflows/[id]/page.tsx`
  - `apps/dashboard/src/app/(dashboard)/suppliers/[id]/page.tsx`
  - `apps/dashboard/src/components/integrations/marketplace-shipment-settings.tsx`
  - `apps/dashboard/src/components/orders/order-filters.tsx`
  - `apps/dashboard/src/components/orders/kanban-board.tsx`
  - `apps/dashboard/src/components/shared/data-table.tsx`
  - `apps/dashboard/src/app/(onboarding)/onboarding/page.tsx`

- [ ] **Step 1: Convert each config/entity draft to `useEffectSyncedState`**

Use a stable reset key such as:
- config `updated_at` when present,
- entity `id`,
- serialized provider/default settings for small settings objects,
- `filters.search` or `filters.tag` for filter input drafts.

- [ ] **Step 2: Preserve dirty guards**

If existing code avoids overwriting user edits, keep that guard by choosing a reset key that changes only when the external record changes, or by keeping existing dirty checks around the setter.

- [ ] **Step 3: Run lint after each cluster**

Run:

```bash
cd apps/dashboard
npx eslint --quiet src/hooks/use-effect-synced-state.ts src/components src/app
```

Expected: PASS or only unrelated pre-existing errors that are not from `react-hooks/set-state-in-effect`; fix any touched-file errors.

### Task 6: Final Validation

**Files:**
- Verify all modified dashboard files.

- [ ] **Step 1: Inventory check**

Run:

```bash
cd apps/dashboard
rg -n "react-hooks/set-state-in-effect" src
```

Expected: only `src/hooks/use-effect-synced-state.ts`.

- [ ] **Step 2: Targeted tests**

Run:

```bash
cd apps/dashboard
npx vitest run src/hooks/use-effect-synced-state.test.tsx src/lib/__tests__/react-hooks-suppressions.test.ts --reporter=dot
```

Expected: PASS.

- [ ] **Step 3: Dashboard lint**

Run:

```bash
cd apps/dashboard
npm run lint:quiet
```

Expected: PASS.

- [ ] **Step 4: Full public local CI before push**

Run:

```bash
cd .
./scripts/local-ci.sh
```

Expected: PASS.

## Risk And Rollback

- Main risk is silently resetting user draft edits when API data refetches. Mitigation: keep reset keys explicit and preserve existing dirty guards.
- Browser storage hydration still needs one documented hook-level lint exception. Mitigation: inventory test allows only the hook file.
- Rollback is safe with `git revert` of this PR because there are no DB or API contract changes.

## Docs Updated

- No user-facing docs expected.
- PR description should mark docs as N/A, while the implementation plan itself documents the maintenance rule.
