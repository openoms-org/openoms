# OPE-282 Settings Overview Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `/settings` a usable settings overview instead of redirecting directly to password/security settings.

**Architecture:** The settings index page will be a client component because it reads the current user role from the auth store. It will reuse `navItems` and `getVisibleNavItems` so the overview follows the same admin/readiness filtering as the sidebar and does not expose hidden or unready functionality.

**Tech Stack:** Next.js App Router, React 19, TypeScript, next-intl, Tailwind v4, Vitest, Testing Library.

---

### Task 1: Lock The Regression With A Test

**Files:**
- Modify: `apps/dashboard/src/app/(dashboard)/settings/page.test.tsx`

- [ ] **Step 1: Replace the redirect assertion with overview expectations**

The test must verify that `SettingsIndexPage` does not call `redirect`, renders an `Ustawienia` heading, links to `/settings/company` and `/settings/security` for admins, and hides `/settings/company` for regular users.

- [ ] **Step 2: Run the test to verify it fails**

Run:

```bash
cd apps/dashboard
npx vitest run 'src/app/(dashboard)/settings/page.test.tsx' --reporter=dot
```

Expected: FAIL because the current page calls `redirect("/settings/security")`.

### Task 2: Render A Role-Aware Settings Overview

**Files:**
- Modify: `apps/dashboard/src/app/(dashboard)/settings/page.tsx`

- [ ] **Step 1: Convert the page into a client component**

Use `"use client"` because the page reads `useAuthStore`.

- [ ] **Step 2: Build visible settings sections from existing navigation data**

Use `getVisibleNavItems(navItems, { isAdmin })`, then keep entries whose `href` starts with `/settings/` or whose group is `settings`. Deduplicate by `href`.

- [ ] **Step 3: Render cards linking to each visible section**

Each card should show the existing navigation label, icon, route, and a small open action. Do not add buttons for blocked or beta routes in client-ready mode.

- [ ] **Step 4: Run the settings page test to verify it passes**

Run:

```bash
cd apps/dashboard
npx vitest run 'src/app/(dashboard)/settings/page.test.tsx' --reporter=dot
```

Expected: PASS.

### Task 3: Add Settings Index Copy

**Files:**
- Modify: `apps/dashboard/messages/pl/settings.json`
- Modify: `apps/dashboard/messages/en/settings.json`
- Modify: `apps/dashboard/messages/pl/statuses.json`
- Modify: `apps/dashboard/messages/en/statuses.json`
- Modify: `apps/dashboard/src/lib/__tests__/constants.test.ts`

- [ ] **Step 1: Add `settings.index` messages**

Add title, description, section count, empty state, and open action labels in Polish and English.

- [ ] **Step 2: Validate JSON**

Run:

```bash
cd apps/dashboard
node -e "JSON.parse(require('fs').readFileSync('messages/pl/settings.json','utf8')); JSON.parse(require('fs').readFileSync('messages/en/settings.json','utf8'))"
```

Expected: no output and exit code 0.

- [ ] **Step 3: Keep system role translations complete**

Add a constants regression test that verifies every role in `ROLES` exists in both `messages/en/statuses.json` and `messages/pl/statuses.json`. Add missing `roles.admin` labels so the user menu does not emit missing-message errors during browser smoke tests.

### Task 4: Validate And Ship

**Files:**
- Test: `apps/dashboard/src/app/(dashboard)/settings/page.test.tsx`
- Test: `apps/dashboard/src/lib/__tests__/readiness.test.ts`

- [ ] **Step 1: Run targeted tests**

```bash
cd apps/dashboard
npx vitest run 'src/app/(dashboard)/settings/page.test.tsx' 'src/lib/__tests__/readiness.test.ts' --reporter=dot
```

Expected: PASS.

- [ ] **Step 2: Run targeted lint**

```bash
cd apps/dashboard
npx eslint --quiet 'src/app/(dashboard)/settings/page.tsx' 'src/app/(dashboard)/settings/page.test.tsx'
```

Expected: PASS.

- [ ] **Step 3: Run full local CI before push**

```bash
cd ../..
./scripts/local-ci.sh
```

Expected: all checks pass.

- [ ] **Step 4: Browser verification**

Open `/settings` and verify it stays on `/settings`, shows a settings overview, links to Security, and does not show admin-only settings for a non-admin user.
