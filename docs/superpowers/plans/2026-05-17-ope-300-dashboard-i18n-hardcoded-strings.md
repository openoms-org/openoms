# OPE-300 Dashboard I18n Hardcoded Strings Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove the audited hardcoded Polish UI strings from key dashboard/auth/provider surfaces so English locale no longer renders Polish copy.

**Architecture:** Keep the existing `next-intl` model: UI code calls `useTranslations(...)`, copy lives in paired `messages/pl/*.json` and `messages/en/*.json` files, and dynamic provider/business data remains untranslated. Add a source-level regression test for the audited files so new Polish literals in those hot spots fail fast.

**Tech Stack:** Next.js 16, React 19, TypeScript, next-intl, Vitest, JSON message modules.

---

## Files And Areas

- Modify: `apps/dashboard/src/components/dashboard/stat-cards.tsx`
  - Replace remaining hardcoded stat titles with `dashboard.stats.*`.
- Modify: `apps/dashboard/src/app/(auth)/register/page.tsx`
  - Use `useTranslations("auth.pricing")` or a focused auth namespace for pricing/register plan copy.
  - Keep `formatPrice` deterministic for now; locale-aware price formatting can be a follow-up if needed.
- Modify: `apps/dashboard/src/app/(dashboard)/pick-pack/page.tsx`
  - Add `useTranslations("warehouses.pickPack")` or `useTranslations("pickPack")` depending on existing message shape.
  - Translate status labels, table headers, dialog copy, empty states, toasts, and buttons.
- Modify: `apps/dashboard/src/app/(public)/supplier-portal/page.tsx`
  - Add `useTranslations("supplierPortal")`.
  - Translate portal status labels, empty states, table headers, errors, actions, dialogs, and placeholders.
- Modify: selected files under `apps/dashboard/src/app/(dashboard)/marketplaces/allegro/`
  - At minimum: `delivery/page.tsx`, `disputes/page.tsx`, `ratings/page.tsx`, `policies/page.tsx`, plus small audited literals in `catalog/page.tsx`, `finance/page.tsx`, `promotions/page.tsx` if touched by the regression scan.
  - Reuse existing `marketplaces` namespace where keys already exist; add missing scoped keys under `marketplaces.allegro.*`.
- Modify: `apps/dashboard/src/app/(dashboard)/workflows/[id]/page.tsx` and `apps/dashboard/src/components/workflow/*.tsx`
  - Move remaining toolbar/config/error strings into `workflows.*`.
- Modify: `apps/dashboard/messages/pl/*.json` and `apps/dashboard/messages/en/*.json`
  - Add paired keys for every replaced literal.
- Create: `apps/dashboard/src/lib/__tests__/i18n-hardcoded-copy.test.ts`
  - Source-level regression test scanning the audited files for the known Polish hardcoded phrases.

## Task 1: Add Regression Coverage First

- [x] **Step 1: Write the failing source-scan test**

Create `apps/dashboard/src/lib/__tests__/i18n-hardcoded-copy.test.ts`:

```ts
import { readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const dashboardRoot = resolve(dirname(fileURLToPath(import.meta.url)), "../..");

const auditedFiles = [
  "components/dashboard/stat-cards.tsx",
  "app/(auth)/register/page.tsx",
  "app/(dashboard)/pick-pack/page.tsx",
  "app/(public)/supplier-portal/page.tsx",
  "app/(dashboard)/marketplaces/allegro/delivery/page.tsx",
  "app/(dashboard)/marketplaces/allegro/disputes/page.tsx",
  "app/(dashboard)/marketplaces/allegro/ratings/page.tsx",
  "app/(dashboard)/marketplaces/allegro/policies/page.tsx",
  "app/(dashboard)/marketplaces/allegro/catalog/page.tsx",
  "app/(dashboard)/marketplaces/allegro/finance/page.tsx",
  "app/(dashboard)/marketplaces/allegro/promotions/page.tsx",
  "app/(dashboard)/workflows/[id]/page.tsx",
  "components/workflow/workflow-toolbar.tsx",
  "components/workflow/node-config-panel.tsx",
];

const forbiddenPhrases = [
  "Nie udalo sie",
  "Brak zamowien",
  "Brak dostępnych",
  "Brak dostepnych",
  "Wybierz plan",
  "Miesiecznie",
  "Rocznie",
  "Najpopularniejszy",
  "Kompletowanie",
  "Pakowanie",
  "Zakonczone",
  "Anulowane",
  "Utworz",
  "Sprobuj ponownie",
  "Portal Dostawcy",
  "Zamowienia zakupowe",
  "Wiadomosci",
  "Potwierdz zamowienie",
  "Oznacz jako wyslane",
  "Zapisz",
  "Wybierz zdarzenie",
  "Wybierz typ akcji",
];

describe("dashboard audited copy", () => {
  it("does not keep audited Polish UI copy hardcoded in source files", () => {
    const violations = auditedFiles.flatMap((file) => {
      const source = readFileSync(resolve(dashboardRoot, file), "utf8");
      return forbiddenPhrases
        .filter((phrase) => source.includes(phrase))
        .map((phrase) => `${file}: ${phrase}`);
    });

    expect(violations).toEqual([]);
  });
});
```

- [x] **Step 2: Run RED**

Run:

```bash
cd apps/dashboard
npx vitest run src/lib/__tests__/i18n-hardcoded-copy.test.ts --reporter=dot
```

Expected: FAIL with violations from existing hardcoded Polish strings.

## Task 2: Translate Dashboard Stats, Register, And Pick & Pack

- [x] **Step 1: Replace stat-card literals**

In `stat-cards.tsx`, replace:

```tsx
title="Nowe"
title="W transporcie"
title="Dostarczone"
```

with:

```tsx
title={t("stats.new")}
title={t("stats.inTransit")}
title={t("stats.delivered")}
```

- [x] **Step 2: Add register pricing copy keys**

Add under `messages/pl/common.json` and `messages/en/common.json` in the existing top-level `auth` object:

```json
"pricing": {
  "loadPlansError": "...",
  "noPlansTitle": "...",
  "noPlansDescription": "...",
  "login": "...",
  "trialDaysFree": "...",
  "title": "...",
  "trialDescription": "...",
  "monthly": "...",
  "yearly": "...",
  "popular": "...",
  "perMonth": "...",
  "yearlyTotal": "..."
}
```

- [x] **Step 3: Use register pricing translations**

In `register/page.tsx`:

```tsx
const t = useTranslations("auth.pricing");
```

Use `t(...)` for the audited hardcoded pricing strings and preserve dynamic values with interpolation.

- [x] **Step 4: Add Pick & Pack copy keys**

Add a top-level `pickPack` object to `messages/pl/warehouses.json` and `messages/en/warehouses.json` containing status labels, buttons, table headers, empty states, dialogs, and toast copy used by `pick-pack/page.tsx`.

- [x] **Step 5: Use Pick & Pack translations**

In `pick-pack/page.tsx`, add:

```tsx
import { useTranslations } from "next-intl";
```

Then:

```tsx
const t = useTranslations("pickPack");
```

Use `t(...)` for status labels, toasts, PageHeader copy, filters, table headers, empty state, dialog labels, and submit state.

- [x] **Step 6: Run targeted test**

Run:

```bash
cd apps/dashboard
npx vitest run src/lib/__tests__/i18n-hardcoded-copy.test.ts --reporter=dot
```

Expected: still FAIL until Supplier Portal, Allegro, and workflow files are converted.

## Task 3: Translate Supplier Portal

- [x] **Step 1: Add Supplier Portal copy keys**

Expand `messages/pl/suppliers.json` and `messages/en/suppliers.json` from the current string-only `supplierPortal` into an object containing:

```json
"supplierPortal": {
  "title": "...",
  "welcome": "...",
  "loading": "...",
  "errors": {
    "generic": "...",
    "confirm": "...",
    "ship": "...",
    "sendMessage": "...",
    "missingToken": "...",
    "expiredToken": "...",
    "invalidToken": "..."
  },
  "orders": {
    "empty": "...",
    "title": "...",
    "details": "...",
    "columns": { "poNumber": "...", "status": "...", "amount": "...", "expectedDate": "...", "createdAt": "..." }
  },
  "statuses": {
    "sent": "...",
    "confirmed": "...",
    "shipped": "...",
    "partially_received": "...",
    "received": "...",
    "cancelled": "..."
  },
  "detail": {
    "back": "...",
    "amount": "...",
    "expectedDate": "...",
    "createdAt": "...",
    "updatedAt": "...",
    "notes": "...",
    "confirmOrder": "...",
    "markShipped": "...",
    "items": "...",
    "messages": "...",
    "noMessages": "...",
    "supplier": "...",
    "buyer": "...",
    "messagePlaceholder": "...",
    "send": "..."
  },
  "dialogs": {
    "reference": "...",
    "referencePlaceholder": "...",
    "trackingNumber": "...",
    "trackingPlaceholder": "...",
    "carrier": "...",
    "carrierPlaceholder": "...",
    "cancel": "...",
    "confirming": "...",
    "saving": "..."
  }
}
```

- [x] **Step 2: Use Supplier Portal translations**

In `supplier-portal/page.tsx`, pass `t` into helper components that are outside the main page component, or convert helpers to accept translated labels via props. Keep provider data such as supplier name, PO number, SKU, carrier names, and free-text messages as raw business data.

- [x] **Step 3: Run targeted test**

Run:

```bash
cd apps/dashboard
npx vitest run src/lib/__tests__/i18n-hardcoded-copy.test.ts --reporter=dot
```

Expected: still FAIL only for remaining Allegro/workflow audited strings.

## Task 4: Translate Allegro And Workflow Audited Strings

- [x] **Step 1: Add missing Allegro message keys**

Use `messages/pl/marketplaces.json` and `messages/en/marketplaces.json`. Add scoped keys under the existing `marketplaces.allegro` subtree or the closest existing page subtree for delivery, disputes, ratings, policies, catalog, finance, and promotions.

- [x] **Step 2: Replace audited Allegro literals**

Replace only human UI copy and toast fallbacks. Do not translate provider names, API enum values, IDs, policy names returned by Allegro, or user-entered data.

- [x] **Step 3: Add missing workflow message keys**

Use `messages/pl/workflows.json` and `messages/en/workflows.json` for workflow page errors, toolbar save button, and node config placeholders.

- [x] **Step 4: Replace audited workflow literals**

Use the existing `useTranslations("workflows")` in workflow pages/components, adding it where a component currently lacks translations.

- [x] **Step 5: Run GREEN**

Run:

```bash
cd apps/dashboard
npx vitest run src/lib/__tests__/i18n-hardcoded-copy.test.ts --reporter=dot
```

Expected: PASS.

## Task 5: Validation, Docs, PR

- [x] **Step 1: Format and lint touched dashboard files**

Run:

```bash
cd apps/dashboard
npm run lint:quiet -- src/components/dashboard/stat-cards.tsx src/app/'(auth)'/register/page.tsx src/app/'(dashboard)'/pick-pack/page.tsx src/app/'(public)'/supplier-portal/page.tsx src/lib/__tests__/i18n-hardcoded-copy.test.ts
```

Expected: PASS. If Allegro/workflow files are touched, add them to the same lint command.

- [x] **Step 2: Run dashboard tests for touched areas**

Run:

```bash
cd apps/dashboard
npx vitest run src/lib/__tests__/i18n-hardcoded-copy.test.ts src/app/'(auth)'/register/page.test.tsx --reporter=dot
```

Expected: PASS.

- [x] **Step 3: Run repository checks**

Run:

```bash
git diff --check
./scripts/local-ci.sh
```

Expected: PASS before push.

- [ ] **Step 4: Update Linear and PR**

Add a Linear comment with the exact validation commands and results. PR title:

```text
OPE-300: remove audited hardcoded dashboard copy
```

PR body must include:

```markdown
## Docs updated
- [ ] N/A — no API/domain/system documentation changes
```

## Risks And Rollback

- Risk: message-key changes can break runtime rendering if a key is missing. Mitigation: run targeted Vitest, lint, Next build via local CI, and let main CI exercise production build.
- Risk: over-translating provider data could hide raw external values. Mitigation: only translate static UI copy; keep provider/API/user data raw.
- Rollback: revert the PR; no database, API, or infra state changes are involved.
