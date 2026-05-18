# OPE-321 Locale Provider Login Reload Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Stop `LocaleProvider` from hard-reloading the dashboard during auth hydration when `user.language` differs from the current `NEXT_LOCALE` cookie.

**Architecture:** Keep explicit language changes owned by `LanguageSelector`, which persists the new language and intentionally reloads to pick up next-intl messages. Make `LocaleProvider` only synchronize the locale cookie from hydrated auth state without calling `window.location.reload()`.

**Tech Stack:** Next.js 16 client components, React 19, Zustand auth store, Vitest, Testing Library, happy-dom.

---

## Scope

This PR is limited to dashboard locale hydration behavior. It does not change API routes, translations, backend auth, middleware, or i18n routing.

## Files

- Create: `apps/dashboard/src/components/providers/locale-provider.test.tsx`
- Modify: `apps/dashboard/src/components/providers/locale-provider.tsx`

## Risk And Rollback

- Risk: after login, the already-rendered page may stay in the previously loaded next-intl message bundle until normal navigation or a user-triggered language change. This is preferable to a hard reload loop on every fresh login.
- Explicit user language changes still reload through `LanguageSelector`, preserving the current UI behavior for intentional locale switches.
- Rollback: revert the PR; no persistent schema or API changes are involved.

## Task 1: Add Regression Test

**Files:**
- Create: `apps/dashboard/src/components/providers/locale-provider.test.tsx`

- [x] **Step 1: Write failing test**

Create a test that:

```tsx
document.cookie = "NEXT_LOCALE=en; path=/";
useAuthStore.getState().setAuth("token", { ...mockUser, language: "pl" }, mockTenant);

render(
  <LocaleProvider>
    <div>child</div>
  </LocaleProvider>,
);
```

Assert:

```ts
await waitFor(() => expect(document.cookie).toContain("NEXT_LOCALE=pl"));
expect(reloadSpy).not.toHaveBeenCalled();
```

- [x] **Step 2: Verify red**

Run:

```bash
cd /Users/rafs/praca/openoms-dev/public/apps/dashboard
npx vitest run src/components/providers/locale-provider.test.tsx --reporter=dot
```

Observed: FAIL because `window.location.reload()` was called once when auth hydration changed the cookie.

## Task 2: Make Hydration Cookie Sync Non-Reloading

**Files:**
- Modify: `apps/dashboard/src/components/providers/locale-provider.tsx`

- [x] **Step 1: Implement minimal fix**

Change the mismatched-cookie branch from:

```ts
document.cookie = `NEXT_LOCALE=${user.language}; path=/; max-age=31536000; SameSite=Lax`;
window.location.reload();
```

to:

```ts
document.cookie = `NEXT_LOCALE=${user.language}; path=/; max-age=31536000; SameSite=Lax`;
```

Keep the `locale-changing` guard because the language selector still sets the marker before its own explicit reload.

- [x] **Step 2: Verify green**

Run:

```bash
cd /Users/rafs/praca/openoms-dev/public/apps/dashboard
npx vitest run src/components/providers/locale-provider.test.tsx --reporter=dot
```

Observed: PASS.

## Task 3: Validate Dashboard Checks

**Files:**
- No additional code changes expected.

- [x] **Step 1: Run targeted lint**

Run:

```bash
cd /Users/rafs/praca/openoms-dev/public/apps/dashboard
npx eslint --quiet src/components/providers/locale-provider.tsx src/components/providers/locale-provider.test.tsx
```

Observed: PASS.

- [x] **Step 2: Run diff hygiene check**

Run:

```bash
cd /Users/rafs/praca/openoms-dev/public
git diff --check
```

Observed: no output.

- [x] **Step 3: Run full pre-push validation before pushing**

Run:

```bash
cd /Users/rafs/praca/openoms-dev/public
./scripts/local-ci.sh
```

Observed: PASS, `All 10 checks passed`, 58s total.

## Task 4: Publish PR

**Files:**
- Git metadata only.

- [ ] **Step 1: Commit**

```bash
cd /Users/rafs/praca/openoms-dev/public
git add apps/dashboard/src/components/providers/locale-provider.tsx apps/dashboard/src/components/providers/locale-provider.test.tsx docs/superpowers/plans/2026-05-18-ope-321-locale-provider-login-reload.md
git commit -m "OPE-321: avoid login locale reload"
```

- [ ] **Step 2: Push and create PR**

```bash
git push -u origin fix/OPE-321-locale-provider-login-reload
gh pr create --title "OPE-321: avoid login locale reload" --body-file /tmp/ope-321-pr.md
```

PR body must include:

```md
## Summary
- add a regression test for auth-hydration locale sync
- stop LocaleProvider from hard-reloading on login-driven cookie sync

## Test plan
- npx vitest run src/components/providers/locale-provider.test.tsx --reporter=dot
- npx eslint --quiet src/components/providers/locale-provider.tsx src/components/providers/locale-provider.test.tsx
- git diff --check
- ./scripts/local-ci.sh

## Docs updated
- [x] docs/superpowers/plans/2026-05-18-ope-321-locale-provider-login-reload.md
- [ ] N/A — no API, DB, workflow, or system documentation changes needed
```

- [ ] **Step 3: Review gate**

Read GitHub checks and CodeRabbit comments. Fix blockers before merge. If CodeRabbit is rate-limited, comment in Linear and move to the next independent Todo until review can be triggered.

## Self-Review

- Spec coverage: OPE-321 asks to stop reload on auth hydration; Task 1 reproduces it and Task 2 removes the reload from that provider.
- Placeholder scan: no TBD/TODO placeholders.
- Type consistency: uses existing `LocaleProvider`, `useAuthStore`, `User`, `Tenant`, Vitest, and Testing Library APIs.
