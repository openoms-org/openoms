# OPE-245 Customer Help Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the current community-first Help page with a customer-facing support experience that keeps OSS support links secondary.

**Architecture:** This is a dashboard-only UI/copy change. The Help page remains a client component using `next-intl`; static link metadata lives near the page, and no backend or runtime configuration is introduced.

**Tech Stack:** Next.js 16 App Router, React 19, TypeScript, Tailwind v4, shadcn/ui, lucide-react, Vitest, Testing Library.

---

## Files

- Modify: `apps/dashboard/src/app/(dashboard)/help/page.tsx`
- Create: `apps/dashboard/src/app/(dashboard)/help/page.test.tsx`
- Modify: `apps/dashboard/messages/pl/misc.json`
- Modify: `apps/dashboard/messages/en/misc.json`
- Modify: `docs/audit/feature-readiness-matrix-2026-05-11.md`

## Task 1: Add Help Page Regression Test

- [ ] **Step 1: Create the failing test**

Create `apps/dashboard/src/app/(dashboard)/help/page.test.tsx`:

```tsx
import { render, screen, within } from "@testing-library/react";
import HelpPage from "./page";

const translations: Record<string, string> = {
  title: "Pomoc",
  subtitle: "Skontaktuj sie z OpenOMS albo sprawdz sprawdzone materialy.",
  officialSupportTitle: "Pomoc OpenOMS",
  officialSupportDescription: "Napisz do nas, jesli potrzebujesz pomocy z kontem, integracja albo procesem obslugi zamowien.",
  officialSupportCta: "Napisz do supportu",
  officialSupportMeta: "Odpowiadamy w kanale obslugi OpenOMS.",
  prepareTitle: "Co dolaczyc do zgloszenia",
  prepareTenant: "Nazwa firmy lub organizacji",
  prepareContext: "Modul, integracja albo numer zamowienia",
  prepareEvidence: "Krotki opis, screenshot albo komunikat bledu",
  userGuide: "Poradnik uzytkownika",
  userGuideDescription: "Najwazniejsze kroki konfiguracji i pracy operacyjnej.",
  faq: "FAQ",
  faqDescription: "Odpowiedzi na najczestsze pytania sprzedawcow.",
  ossTitle: "Open-source i self-hosting",
  ossDescription: "Dla problemow technicznych z publicznym repo uzyj GitHub Issues.",
  reportBug: "GitHub Issues",
};

vi.mock("next-intl", () => ({
  useTranslations: () => (key: string) => translations[key] ?? key,
}));

describe("HelpPage", () => {
  it("prioritizes official customer support over community issue reporting", () => {
    render(<HelpPage />);

    const supportLink = screen.getByRole("link", { name: /napisz do supportu/i });
    expect(supportLink).toHaveAttribute("href", "mailto:support@openoms.org");

    expect(screen.getByText("Pomoc OpenOMS")).toBeInTheDocument();
    expect(screen.getByText("Co dolaczyc do zgloszenia")).toBeInTheDocument();
    expect(screen.queryByText(/discord/i)).not.toBeInTheDocument();

    const ossSection = screen.getByText("Open-source i self-hosting").closest("section");
    expect(ossSection).not.toBeNull();
    expect(
      within(ossSection as HTMLElement).getByRole("link", { name: /github issues/i })
    ).toHaveAttribute("href", "https://github.com/openoms-org/openoms/issues");
  });
});
```

- [ ] **Step 2: Run the test and confirm it fails**

Run:

```bash
cd apps/dashboard
npx vitest run 'src/app/(dashboard)/help/page.test.tsx' --reporter=dot
```

Expected: FAIL because the current page still renders Discord/community-first content and does not include the new support copy.

## Task 2: Implement the Customer-Facing Help Page

- [ ] **Step 1: Replace the page structure**

Modify `apps/dashboard/src/app/(dashboard)/help/page.tsx` to:

- keep `"use client"`,
- use `Button` for the primary mail CTA,
- render official support first,
- render documentation cards second,
- render OSS support in a secondary section,
- remove Discord from the page.

Key constants:

```tsx
const SUPPORT_EMAIL = "support@openoms.org";
const USER_GUIDE_URL = "https://github.com/openoms-org/openoms/blob/main/docs/poradnik-uzytkownika.md";
const FAQ_URL = "https://github.com/openoms-org/openoms/blob/main/docs/faq-sprzedawcy.md";
const GITHUB_ISSUES_URL = "https://github.com/openoms-org/openoms/issues";
```

- [ ] **Step 2: Add PL/EN translations**

Update `apps/dashboard/messages/pl/misc.json` and `apps/dashboard/messages/en/misc.json` under the `help` namespace with these keys:

```json
{
  "officialSupportTitle": "...",
  "officialSupportDescription": "...",
  "officialSupportCta": "...",
  "officialSupportMeta": "...",
  "prepareTitle": "...",
  "prepareTenant": "...",
  "prepareContext": "...",
  "prepareEvidence": "...",
  "ossTitle": "...",
  "ossDescription": "..."
}
```

Remove unused Discord-specific help keys only if they are no longer referenced anywhere.

- [ ] **Step 3: Run the Help page test**

Run:

```bash
cd apps/dashboard
npx vitest run 'src/app/(dashboard)/help/page.test.tsx' --reporter=dot
```

Expected: PASS.

## Task 3: Update Readiness Documentation

- [ ] **Step 1: Update the Help row**

Modify `docs/audit/feature-readiness-matrix-2026-05-11.md` so the `/help` row states that it is customer-facing, support-first, and does not expose unfinished helpdesk flows.

- [ ] **Step 2: Verify docs formatting**

Run:

```bash
git diff --check
```

Expected: no whitespace errors.

## Task 4: Validate and Commit

- [ ] **Step 1: Run targeted validation**

Run:

```bash
cd apps/dashboard
npx vitest run 'src/app/(dashboard)/help/page.test.tsx' --reporter=dot
npx eslint --quiet 'src/app/(dashboard)/help/page.tsx' 'src/app/(dashboard)/help/page.test.tsx'
```

Expected: both commands pass.

- [ ] **Step 2: Run quick local CI**

Run:

```bash
./scripts/local-ci.sh --quick
```

Expected: all quick checks pass.

- [ ] **Step 3: Self-review diff**

Run:

```bash
git diff --check
git diff --stat
git diff
```

Expected: diff contains only OPE-245 Help page, translations, test, and docs changes.

- [ ] **Step 4: Commit**

Run:

```bash
git add apps/dashboard/src/app/'(dashboard)'/help/page.tsx \
  apps/dashboard/src/app/'(dashboard)'/help/page.test.tsx \
  apps/dashboard/messages/pl/misc.json \
  apps/dashboard/messages/en/misc.json \
  docs/audit/feature-readiness-matrix-2026-05-11.md \
  docs/superpowers/specs/2026-05-22-ope-245-customer-help-design.md \
  docs/superpowers/plans/2026-05-22-ope-245-customer-help.md
git commit -m "OPE-245: redesign customer help experience"
```

Expected: commit succeeds with no generated attribution.
