# OPE-344 DataTable Typed Nested Value Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove the `any` escape hatch from the shared dashboard `DataTable` nested accessor helper while preserving current nested key rendering behavior.

**Architecture:** Keep the existing `DataTable<T>` public API unchanged. Replace the unsafe helper with a small `Record<string, unknown>`/`unknown` based traversal guard so non-object values fail closed to `undefined`.

**Tech Stack:** Next.js 16, React 19, TypeScript, Vitest, Testing Library, ESLint.

---

### Task 1: Add Regression Coverage

**Files:**
- Modify: `apps/dashboard/src/components/shared/__tests__/data-table.test.tsx`

- [x] **Step 1: Add a source-level regression test**

Add a test that reads `data-table.tsx` and fails while `getNestedValue` still uses `any` or an eslint suppression:

```ts
import { readFileSync } from "node:fs";
```

```ts
it("keeps nested accessor lookup typed without any escapes", () => {
  const source = readFileSync("src/components/shared/data-table.tsx", "utf8");

  expect(source).not.toContain("eslint-disable-next-line @typescript-eslint/no-explicit-any");
  expect(source).toMatch(
    /function getNestedValue\(\s*obj: Record<string, unknown>,\s*path: string\s*\): unknown/,
  );
  expect(source).not.toContain("function getNestedValue(obj: any");
});
```

- [x] **Step 2: Run the targeted test and verify RED**

Run:

```bash
cd apps/dashboard
npx vitest run src/components/shared/__tests__/data-table.test.tsx --reporter=dot
```

Expected: FAIL because `data-table.tsx` currently contains the `no-explicit-any` suppression and `obj: any`.

### Task 2: Type the Nested Accessor Helper

**Files:**
- Modify: `apps/dashboard/src/components/shared/data-table.tsx`

- [x] **Step 1: Replace `any` traversal with an object guard**

Use this implementation shape:

```ts
function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

function getNestedValue(obj: Record<string, unknown>, path: string): unknown {
  return path.split(".").reduce<unknown>((acc, part) => {
    if (isRecord(acc)) {
      return acc[part];
    }
    return undefined;
  }, obj);
}
```

- [x] **Step 2: Keep generic call sites compatible**

Where `DataTable<T>` passes generic rows, cast only at the helper boundary:

```ts
getNestedValue(row as Record<string, unknown>, key)
```

This avoids forcing all `T` row interfaces to declare an index signature.

### Task 3: Verify and Publish

**Files:**
- No docs update required beyond this plan because no API, DB, architecture, deploy, or user-facing behavior changes.

- [x] **Step 1: Run targeted tests**

```bash
cd apps/dashboard
npx vitest run src/components/shared/__tests__/data-table.test.tsx --reporter=dot
npm run lint:quiet -- src/components/shared/data-table.tsx src/components/shared/__tests__/data-table.test.tsx
```

- [x] **Step 2: Run repository whitespace validation**

```bash
cd /Users/rafs/praca/openoms-dev/public
git diff --check
```

- [x] **Step 3: Commit**

```bash
git add apps/dashboard/src/components/shared/data-table.tsx apps/dashboard/src/components/shared/__tests__/data-table.test.tsx docs/superpowers/plans/2026-05-17-ope-344-datatable-typed-nested-value.md
git commit -m "OPE-344: type DataTable nested accessor"
```

- [ ] **Step 4: Run full local CI on clean HEAD, then push and PR**

```bash
./scripts/local-ci.sh
git push --set-upstream origin fix/OPE-344-datatable-typed-nested-value
```

Open a PR titled `OPE-344: type DataTable nested accessor` with `Docs updated: N/A — no product docs needed`.

### Risk and Rollback

- Risk: `DataTable<T>` row interfaces without an index signature could fail if the generic type is constrained directly. The helper boundary cast avoids that.
- Runtime behavior should remain unchanged: nested key misses return `undefined`, rendered as an empty string.
- Rollback: revert the single commit/PR if dashboard tests or CI reveal unexpected type fallout.
