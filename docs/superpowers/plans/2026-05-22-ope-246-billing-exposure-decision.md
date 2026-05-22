# OPE-246 Billing Exposure Decision Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Keep customer-facing billing hidden from the default client-ready dashboard until the Stripe/self-service flow is certified, while preventing any visible banner from linking users to the hidden billing settings route.

**Architecture:** The existing readiness registry remains the source of truth: `/settings/billing` stays `controlled`, so `client-ready` cannot navigate to it. Subscription banners must check the same readiness rule before rendering a billing link; if billing is hidden, banners render support-oriented copy without an actionable link. The decision is documented in audit docs so future work knows what evidence is required before billing can move to `ready`.

**Tech Stack:** Next.js 16 dashboard, React 19, TypeScript, Vitest/Testing Library, existing dashboard readiness registry in `apps/dashboard/src/lib/readiness.ts`, public audit docs under `docs/audit/`.

---

## Scope

- Public repo only.
- Do not change backend billing enforcement, Stripe webhooks, checkout registration, production deployment, or enterprise config.
- Do not make `/settings/billing` client-ready.
- Do not add a customer-facing payment update flow in this PR.

## Files

- Modify: `apps/dashboard/src/components/subscription-banner.tsx`
  - Import `isRouteAccessible`.
  - Render `/settings/billing` links only when that route is accessible in the current dashboard surface mode.
  - Render plain support copy when billing is hidden.
  - Treat Stripe inactive/payment statuses consistently with backend plan guard statuses.
- Create: `apps/dashboard/src/components/__tests__/subscription-banner.test.tsx`
  - Cover hidden billing route behavior in default `client-ready` mode.
  - Cover linked billing route behavior in `NEXT_PUBLIC_OPENOMS_DASHBOARD_SURFACE=full`.
  - Cover no-tenant/no-banner behavior.
- Modify: `apps/dashboard/src/types/billing.ts`
  - Expand `SubscriptionStatus.status` to include Stripe statuses already returned/handled by the backend: `incomplete`, `incomplete_expired`, `unpaid`, `paused`.
- Modify: `apps/dashboard/messages/pl/layout.json`
  - Add support-copy strings used by the banner when billing is not exposed.
- Modify: `apps/dashboard/messages/en/layout.json`
  - Add English support-copy strings for fallback locale parity.
- Modify: `apps/dashboard/messages/pl/dashboard.json`
  - Add labels for `unpaid`, `incomplete`, `incomplete_expired`, and `paused` if the internal billing page is opened in `full` mode.
- Modify: `apps/dashboard/messages/en/dashboard.json`
  - Add English status labels for the same internal billing states.
- Modify: `apps/dashboard/src/app/(dashboard)/settings/billing/page.tsx`
  - Map the expanded statuses to readable labels and badge variants.
- Modify: `docs/audit/invoicing-accounting-billing-readiness-2026-05-18.md`
  - Replace the OPE-246 pending wording with the final conservative decision.
- Modify: `docs/audit/feature-readiness-matrix-2026-05-11.md`
  - Update the billing row with the final decision and evidence required before future exposure.

## Tasks

### Task 1: Add regression coverage for subscription banner visibility

**Files:**
- Create: `apps/dashboard/src/components/__tests__/subscription-banner.test.tsx`

- [ ] **Step 1: Create a failing test file**

```tsx
import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { SubscriptionBanner } from "@/components/subscription-banner";
import { useAuthStore } from "@/lib/auth";
import type { SubscriptionStatus } from "@/types/api";

let subscription: SubscriptionStatus | undefined;

vi.mock("@/hooks/use-billing", () => ({
  useSubscription: () => ({ data: subscription }),
}));

vi.mock("next-intl", () => ({
  useTranslations: () => (key: string, values?: Record<string, unknown>) =>
    values?.date ? `${key}:${values.date}` : key,
}));

const tenant = {
  id: "tenant-1",
  name: "OpenOMS",
  slug: "openoms",
  plan: "plus",
};

describe("SubscriptionBanner", () => {
  beforeEach(() => {
    vi.unstubAllEnvs();
    subscription = undefined;
    useAuthStore.setState({
      token: "token",
      tenant: tenant as never,
      user: null,
      isAuthenticated: true,
      isLoading: false,
      locale: "pl",
    });
  });

  it("does not render without tenant context", () => {
    useAuthStore.setState({ tenant: null });
    subscription = {
      plan: "plus",
      status: "trialing",
      trial_end: "2099-01-10T00:00:00Z",
    };

    const { container } = render(<SubscriptionBanner />);

    expect(container).toBeEmptyDOMElement();
  });

  it("does not link to hidden billing settings in client-ready mode", () => {
    subscription = {
      plan: "plus",
      status: "trialing",
      trial_end: "2099-01-10T00:00:00Z",
    };

    render(<SubscriptionBanner />);

    expect(screen.getByText("subscriptionManagedBySupport")).toBeInTheDocument();
    expect(
      screen.queryByRole("link", { name: "zarzadzajSubskrypcja" }),
    ).not.toBeInTheDocument();
  });

  it("links to billing settings only in full dashboard mode", () => {
    vi.stubEnv("NEXT_PUBLIC_OPENOMS_DASHBOARD_SURFACE", "full");
    subscription = {
      plan: "plus",
      status: "trialing",
      trial_end: "2099-01-10T00:00:00Z",
    };

    render(<SubscriptionBanner />);

    expect(
      screen.getByRole("link", { name: "zarzadzajSubskrypcja" }),
    ).toHaveAttribute("href", "/settings/billing");
  });

  it("renders inactive subscription support copy without hidden billing links", () => {
    subscription = {
      plan: "plus",
      status: "canceled",
      current_period_end: "2099-01-10T00:00:00Z",
    };

    render(<SubscriptionBanner />);

    expect(screen.getByText("renewViaSupport")).toBeInTheDocument();
    expect(
      screen.queryByRole("link", { name: "odnowSubskrypcje" }),
    ).not.toBeInTheDocument();
  });
});
```

- [ ] **Step 2: Run the test and confirm the current behavior fails**

Run:

```bash
cd apps/dashboard
npx vitest run src/components/__tests__/subscription-banner.test.tsx --reporter=dot
```

Expected before implementation: at least one assertion fails because the current banner renders links to `/settings/billing` even when that route is hidden in `client-ready`.

### Task 2: Make subscription banner actions readiness-aware

**Files:**
- Modify: `apps/dashboard/src/components/subscription-banner.tsx`
- Modify: `apps/dashboard/messages/pl/layout.json`
- Modify: `apps/dashboard/messages/en/layout.json`

- [ ] **Step 1: Update banner logic**

In `apps/dashboard/src/components/subscription-banner.tsx`, add:

```tsx
import { isRouteAccessible } from "@/lib/readiness";
```

Add helper functions before `SubscriptionBanner`:

```tsx
function isPaymentAttentionStatus(status: string | undefined): boolean {
  return status === "past_due" || status === "unpaid" || status === "incomplete";
}

function isInactiveStatus(status: string | undefined): boolean {
  return status === "canceled" || status === "paused" || status === "incomplete_expired";
}
```

Inside the component, after `const status = subscription?.status;`, add:

```tsx
  const canManageBilling = isRouteAccessible("/settings/billing");
```

Replace the trial link block with:

```tsx
        {canManageBilling ? (
          <Link href="/settings/billing" className="underline hover:no-underline">
            {t("zarzadzajSubskrypcja")}
          </Link>
        ) : (
          <span className="font-medium">{t("subscriptionManagedBySupport")}</span>
        )}
```

Replace the canceled-only block with an inactive-status block:

```tsx
  if (isInactiveStatus(status) && subscription?.current_period_end) {
    const expiresAt = new Date(subscription.current_period_end).toLocaleDateString("pl-PL");
    return (
      <div className="bg-orange-500 px-4 py-3 text-center text-sm font-medium text-white">
        <AlertTriangle className="mr-2 inline h-4 w-4" />
        {t("subscriptionCancelled", { date: expiresAt })}{" "}
        {canManageBilling ? (
          <Link href="/settings/billing" className="underline hover:no-underline">
            {t("odnowSubskrypcje")}
          </Link>
        ) : (
          <span className="font-medium">{t("renewViaSupport")}</span>
        )}
      </div>
    );
  }
```

Use `isPaymentAttentionStatus(status)` for the past-due banner so the frontend matches backend guard statuses:

```tsx
  if (isPaymentAttentionStatus(status)) {
```

- [ ] **Step 2: Add translations**

In `apps/dashboard/messages/pl/layout.json`, under `shared`, add:

```json
"renewViaSupport": "Skontaktuj się z obsługą OpenOMS, aby odnowić dostęp.",
"subscriptionManagedBySupport": "Skontaktuj się z obsługą OpenOMS, aby zmienić plan lub płatność."
```

In `apps/dashboard/messages/en/layout.json`, under `shared`, add:

```json
"renewViaSupport": "Contact OpenOMS support to renew access.",
"subscriptionManagedBySupport": "Contact OpenOMS support to change your plan or payment."
```

- [ ] **Step 3: Run the banner test**

Run:

```bash
cd apps/dashboard
npx vitest run src/components/__tests__/subscription-banner.test.tsx --reporter=dot
```

Expected: all `SubscriptionBanner` tests pass.

### Task 3: Align dashboard subscription status types and labels with backend statuses

**Files:**
- Modify: `apps/dashboard/src/types/billing.ts`
- Modify: `apps/dashboard/src/app/(dashboard)/settings/billing/page.tsx`
- Modify: `apps/dashboard/messages/pl/dashboard.json`
- Modify: `apps/dashboard/messages/en/dashboard.json`

- [ ] **Step 1: Expand the frontend status union**

Change `SubscriptionStatus.status` to:

```ts
status:
  | "trialing"
  | "active"
  | "past_due"
  | "canceled"
  | "suspended"
  | "incomplete"
  | "incomplete_expired"
  | "unpaid"
  | "paused";
```

- [ ] **Step 2: Add billing page labels and variants**

In `apps/dashboard/src/app/(dashboard)/settings/billing/page.tsx`, extend `STATUS_VARIANTS`:

```ts
incomplete: "destructive",
incomplete_expired: "destructive",
unpaid: "destructive",
paused: "destructive",
```

Extend `STATUS_LABELS`:

```ts
incomplete: t("statusIncomplete"),
incomplete_expired: t("statusIncompleteExpired"),
unpaid: t("statusUnpaid"),
paused: t("statusPaused"),
```

- [ ] **Step 3: Add translation labels**

In `apps/dashboard/messages/pl/dashboard.json`, under `subscription`, add:

```json
"statusIncomplete": "Płatność nieukończona",
"statusIncompleteExpired": "Płatność wygasła",
"statusPaused": "Wstrzymana",
"statusUnpaid": "Nieopłacona"
```

In `apps/dashboard/messages/en/dashboard.json`, under `subscription`, add:

```json
"statusIncomplete": "Incomplete payment",
"statusIncompleteExpired": "Payment expired",
"statusPaused": "Paused",
"statusUnpaid": "Unpaid"
```

- [ ] **Step 4: Type/lint the touched files**

Run:

```bash
cd apps/dashboard
npx eslint --quiet src/components/subscription-banner.tsx src/components/__tests__/subscription-banner.test.tsx 'src/app/(dashboard)/settings/billing/page.tsx' src/types/billing.ts
```

Expected: no ESLint errors.

### Task 4: Document the billing exposure decision

**Files:**
- Modify: `docs/audit/invoicing-accounting-billing-readiness-2026-05-18.md`
- Modify: `docs/audit/feature-readiness-matrix-2026-05-11.md`

- [ ] **Step 1: Update OPE-253 billing row**

Replace the `Billing SaaS` row decision with:

```markdown
| Billing SaaS | `/settings/billing` | `controlled` | Nie | Pozostaje ukryte dla pierwszego klienta; self-service billing wróci dopiero po Stripe E2E i decyzji support/recovery | Stripe checkout/subscription/webhook E2E, statusy active/trial/past_due/canceled/unpaid/paused/manual contract, bezpieczne CTA bez linków do ukrytych ekranów, puste/error states, recovery przez operatora. |
```

Replace the follow-up line:

```markdown
- OPE-246: decyzja, jak klient ma widzieć billing/subskrypcję w SaaS.
```

with:

```markdown
- OPE-246: domyślna decyzja SaaS v1 to brak customer-facing self-service billing; banner płatności nie może linkować do `/settings/billing`, jeśli route jest ukryty w `client-ready`.
```

- [ ] **Step 2: Update feature readiness matrix billing row**

Replace the `Subskrypcja/billing` row with:

```markdown
| Subskrypcja/billing | `/settings/billing` | `controlled` | Ukryte dla pierwszego klienta; dostęp tylko w `full`/operator validation | Stripe checkout/subscription/webhook E2E, manual/enterprise contract copy, statusy inactive/payment, brak linków do ukrytego route z bannerów. |
```

- [ ] **Step 3: Validate markdown formatting**

Run:

```bash
git diff --check
```

Expected: no whitespace errors.

### Task 5: Final validation and PR prep

**Files:**
- All touched files.

- [ ] **Step 1: Run targeted dashboard tests**

Run:

```bash
cd apps/dashboard
npx vitest run src/components/__tests__/subscription-banner.test.tsx src/lib/__tests__/readiness.test.ts --reporter=dot
```

Expected: both test files pass.

- [ ] **Step 2: Run targeted ESLint**

Run:

```bash
cd apps/dashboard
npx eslint --quiet src/components/subscription-banner.tsx src/components/__tests__/subscription-banner.test.tsx 'src/app/(dashboard)/settings/billing/page.tsx' src/types/billing.ts
```

Expected: no ESLint errors.

- [ ] **Step 3: Run public repo validation before push**

Run:

```bash
./scripts/local-ci.sh
```

Expected: all local CI checks pass.

- [ ] **Step 4: Commit**

Run:

```bash
git add apps/dashboard/src/components/subscription-banner.tsx apps/dashboard/src/components/__tests__/subscription-banner.test.tsx apps/dashboard/src/types/billing.ts 'apps/dashboard/src/app/(dashboard)/settings/billing/page.tsx' apps/dashboard/messages/pl/layout.json apps/dashboard/messages/en/layout.json apps/dashboard/messages/pl/dashboard.json apps/dashboard/messages/en/dashboard.json docs/audit/invoicing-accounting-billing-readiness-2026-05-18.md docs/audit/feature-readiness-matrix-2026-05-11.md docs/superpowers/plans/2026-05-22-ope-246-billing-exposure-decision.md
git commit -m "OPE-246: document billing exposure decision"
```

Expected: commit succeeds with only OPE-246 scoped changes.

## Risk and Rollback

- Risk: Hiding the billing link removes an immediate self-service path if a future deployment deliberately exposes billing. Mitigation: `full` mode still renders the link because the route is accessible there.
- Risk: Support copy may be too operational for future self-service SaaS. Mitigation: the strings are only used while billing is hidden; future billing certification can replace them with payment portal links.
- Rollback: revert this PR. No migrations, API changes, production config, or data rollback is needed.

## Self-Review

- Spec coverage: billing visibility decision, safe dashboard exposure, manual/enterprise non-broken behavior, and tested empty/hidden route behavior are covered.
- Placeholder scan: no TBD/TODO placeholders.
- Type consistency: frontend billing status values match backend plan guard/Stripe status names.
