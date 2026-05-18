# OPE-336 Registration API Client Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace raw `fetch()` calls in registration pages with the shared API client so public registration flows use the same response/error normalization as the rest of the dashboard.

**Architecture:** Keep registration endpoints public and unauthenticated. Use `apiClient<T>()` for `/v1/billing/plans` and `/v1/billing/checkout/:session_id`, and map generic `errors.*` keys back to existing translated fallback text where these pages render user-facing copy directly.

**Tech Stack:** Next.js 16 client components, TypeScript, Vitest static regression test, OpenOMS dashboard `apiClient` / `ApiClientError` / `getErrorMessage`.

---

## Scope

This PR is limited to the two auth registration pages named in OPE-336. It does not change backend billing endpoints, invite registration, checkout creation, auth store behavior, or public config fetching in `usePublicConfig`.

## Files

- Create: `apps/dashboard/src/app/(auth)/register/registration-api-usage.test.ts`
- Modify: `apps/dashboard/src/app/(auth)/register/page.tsx`
- Modify: `apps/dashboard/src/app/(auth)/register/page.test.tsx`
- Modify: `apps/dashboard/src/app/(auth)/register/complete/page.tsx`

## Risk And Rollback

- Risk: `apiClient` adds `credentials: "include"` and JSON `Content-Type`; this matches existing checkout POST usage and is acceptable for public GET endpoints.
- Risk: 404 checkout-session handling must remain a specific “payment session not found” UI state rather than a generic error. The implementation keeps an `ApiClientError.status === 404` branch.
- Rollback: revert the PR; no API or data changes are involved.

## Task 1: Add Regression Test For Raw Fetch In Registration Pages

**Files:**
- Create: `apps/dashboard/src/app/(auth)/register/registration-api-usage.test.ts`

- [x] **Step 1: Write failing test**

Create a static test that reads:

```ts
[
  "apps/dashboard/src/app/(auth)/register/page.tsx",
  "apps/dashboard/src/app/(auth)/register/complete/page.tsx",
]
```

Assert each source file does not match:

```ts
/\bfetch\s*\(/
```

- [x] **Step 2: Verify red**

Run:

```bash
cd /Users/rafs/praca/openoms-dev/public/apps/dashboard
npx vitest run 'src/app/(auth)/register/registration-api-usage.test.ts' --reporter=dot
```

Observed: FAIL listed both registration files, because both contained raw `fetch()`.

## Task 2: Replace Raw Fetch In Plan Selection Page

**Files:**
- Modify: `apps/dashboard/src/app/(auth)/register/page.tsx`
- Modify: `apps/dashboard/src/app/(auth)/register/page.test.tsx`

- [x] **Step 1: Use apiClient for billing plans**

Change the plan-loading effect from raw `fetch(`${API_URL}/v1/billing/plans`, ...)` to:

```ts
apiClient<PublicPlanInfo[]>("/v1/billing/plans")
  .then((data) => {
    setPlans(data);
  })
  .catch((error) => {
    const message = getErrorMessage(error);
    toast.error(message.startsWith("errors.") ? loadPlansError : message);
  })
  .finally(() => setIsLoading(false));
```

Remove the now-unused `API_URL` import.

- [x] **Step 2: Update existing register page test**

The existing test should still assert that plan loading starts only after public config resolves, but account for the shared client request shape:

```ts
await waitFor(() => expect(fetch).toHaveBeenCalledWith(
  "/v1/billing/plans",
  expect.objectContaining({ credentials: "include" }),
));
```

## Task 3: Replace Raw Fetch In Complete Registration Polling

**Files:**
- Modify: `apps/dashboard/src/app/(auth)/register/complete/page.tsx`

- [x] **Step 1: Use apiClient for checkout status polling**

Change polling from raw response handling to:

```ts
const data = await apiClient<CheckoutSessionStatus>(
  `/v1/billing/checkout/${sessionId}`,
  { signal: controller.signal },
);
```

In `catch`, keep:

```ts
if (err instanceof ApiClientError && err.status === 404) {
  setError(t("paymentSessionNotFound"));
  return;
}
```

For final non-404 errors, use `getErrorMessage(err)` and fall back to `t("paymentVerificationFailedRetry")` when the normalized message is an internal `errors.*` key.

## Task 4: Validate Dashboard Checks

**Files:**
- No additional code changes expected.

- [x] **Step 1: Verify regression test green**

Run:

```bash
cd /Users/rafs/praca/openoms-dev/public/apps/dashboard
npx vitest run 'src/app/(auth)/register/registration-api-usage.test.ts' --reporter=dot
```

Observed: PASS.

- [x] **Step 2: Run affected tests**

Run:

```bash
cd /Users/rafs/praca/openoms-dev/public/apps/dashboard
npx vitest run 'src/app/(auth)/register/page.test.tsx' 'src/app/(auth)/register/registration-api-usage.test.ts' --reporter=dot
```

Observed: PASS.

- [x] **Step 3: Run targeted lint**

Run:

```bash
cd /Users/rafs/praca/openoms-dev/public/apps/dashboard
npx eslint --quiet 'src/app/(auth)/register/page.tsx' 'src/app/(auth)/register/complete/page.tsx' 'src/app/(auth)/register/page.test.tsx' 'src/app/(auth)/register/registration-api-usage.test.ts'
```

Observed: PASS.

- [x] **Step 4: Run full pre-push validation**

Run:

```bash
cd /Users/rafs/praca/openoms-dev/public
git diff --check
./scripts/local-ci.sh
```

Observed: `git diff --check` produced no output and `./scripts/local-ci.sh` passed all 10 checks in 56s.

## Task 5: Publish PR

**Files:**
- Git metadata only.

- [ ] **Step 1: Commit**

```bash
cd /Users/rafs/praca/openoms-dev/public
git add 'apps/dashboard/src/app/(auth)/register/page.tsx' 'apps/dashboard/src/app/(auth)/register/page.test.tsx' 'apps/dashboard/src/app/(auth)/register/complete/page.tsx' 'apps/dashboard/src/app/(auth)/register/registration-api-usage.test.ts' docs/superpowers/plans/2026-05-18-ope-336-registration-api-client.md
git commit -m "OPE-336: use api client in registration pages"
```

- [ ] **Step 2: Push and create PR**

```bash
git push -u origin fix/OPE-336-registration-api-client
gh pr create --title "OPE-336: use api client in registration pages" --body-file /tmp/ope-336-pr.md
```

PR body must include:

```md
## Summary
- replace raw registration-page fetches with apiClient
- keep 404 checkout-session handling explicit
- add a static regression test preventing raw fetch from returning to these pages

## Test plan
- npx vitest run 'src/app/(auth)/register/page.test.tsx' 'src/app/(auth)/register/registration-api-usage.test.ts' --reporter=dot
- npx eslint --quiet ...
- git diff --check
- ./scripts/local-ci.sh

## Docs updated
- [x] docs/superpowers/plans/2026-05-18-ope-336-registration-api-client.md
- [ ] N/A — no API, DB, workflow, or system documentation changes needed
```

- [ ] **Step 3: Review gate**

Read GitHub checks and CodeRabbit comments. Fix blockers before merge. If CodeRabbit is rate-limited, comment in Linear and continue with the next independent Todo.

## Self-Review

- Spec coverage: OPE-336 names both registration pages; Task 1 prevents raw fetch in both, Tasks 2 and 3 replace the live call-sites.
- Placeholder scan: no TBD/TODO placeholders.
- Type consistency: uses existing `PublicPlanInfo`, `CheckoutSessionStatus`, `apiClient`, `ApiClientError`, and `getErrorMessage` APIs.
