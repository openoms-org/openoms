# OPE-253: Invoicing, Accounting And Billing Readiness Plan

**Goal:** Validate the invoicing, accounting, KSeF, VAT OSS and billing dashboard surface before client exposure, and keep legal/accounting features hidden until there is provider-level evidence that the flow works end to end.

**Scope:** Public repository only. This pass does not change enterprise deploy, production secrets, provider credentials, Stripe settings, or live accounting systems.

**Architecture:** The existing dashboard readiness registry remains the source of truth for client-ready exposure. Client-ready mode should expose only certified `ready` surfaces; `controlled`, `verify` and `beta` features stay hidden from navigation, provider pickers, command palette and direct route access. The result of this task is a documented readiness decision plus regression coverage proving that legal/accounting routes and providers are not exposed prematurely.

**Tech Stack:** Next.js 16 dashboard, TypeScript, Vitest, existing readiness registry in `apps/dashboard/src/lib/readiness.ts`, audit docs under `docs/audit/`.

---

## Current Findings

- `/invoices`, `/invoicing`, `/settings/accounting` and `/settings/billing` are currently `controlled`.
- `/settings/ksef` is currently `blocked`.
- `/settings/vat-oss` and `/reports/vat-oss` are currently `beta`.
- Fakturownia is `controlled`; wFirma and inFakt are `beta`.
- Client-ready mode only exposes `ready`, so these legal/accounting surfaces should remain hidden for the first customer-safe dashboard.
- Full dashboard mode may show `controlled` and `beta` surfaces for internal/operator validation, except `blocked` routes and providers.

## Files And Areas

- Modify: `apps/dashboard/src/lib/__tests__/readiness.test.ts`
  - Add explicit regression coverage for invoices, invoicing, accounting, billing, KSeF and VAT OSS route exposure.
  - Add explicit provider readiness coverage for Fakturownia, wFirma and inFakt.
- Modify if the tests reveal a gap: `apps/dashboard/src/lib/readiness.ts`
  - Keep current conservative classifications unless coverage proves a missing route/provider rule.
- Create: `docs/audit/invoicing-accounting-billing-readiness-2026-05-18.md`
  - Record the readiness decision matrix and required evidence before any module can move to `ready`.
- No intended changes:
  - No API endpoint behavior changes.
  - No database migrations.
  - No Helm, workflow, Terraform or production configuration changes.
  - No work on OPE-403 or child gated issues.

## Implementation Tasks

### Task 1: Add Readiness Regression Tests

- [ ] Add a test that client-ready navigation does not expose `/invoices`, `/invoicing`, `/settings/accounting`, `/settings/billing`, `/settings/vat-oss` or `/reports/vat-oss`.
- [ ] Add direct route access assertions:
  - `/invoices`, `/invoices/inv-1`, `/invoicing`, `/settings/accounting`, `/settings/billing`, `/settings/vat-oss`, `/reports/vat-oss` are not accessible in `client-ready`.
  - `/invoices`, `/invoicing`, `/settings/accounting`, `/settings/billing`, `/settings/vat-oss`, `/reports/vat-oss` are accessible in `full` when not blocked.
  - `/settings/ksef` remains inaccessible in both `client-ready` and `full`.
- [ ] Add provider assertions:
  - `getVisibleProvidersByCategory("invoicing", { mode: "client-ready" })` returns `[]`.
  - `getVisibleProvidersByCategory("invoicing", { mode: "full" })` returns `["fakturownia", "wfirma", "infakt"]`.
  - `getVisibleProviderKeys(["fakturownia", "wfirma", "infakt"], { mode: "client-ready" })` returns `[]`.
- [ ] Run:

```bash
cd /Users/rafs/praca/openoms-dev/public/apps/dashboard
npx vitest run src/lib/__tests__/readiness.test.ts --reporter=dot
```

Expected: readiness tests pass. If they fail, adjust only the readiness registry or tests needed to preserve the conservative exposure policy.

### Task 2: Write The Audit Decision Matrix

- [ ] Create `docs/audit/invoicing-accounting-billing-readiness-2026-05-18.md`.
- [ ] Include a table for:
  - invoices,
  - invoicing dashboard,
  - Fakturownia,
  - wFirma,
  - inFakt,
  - accounting settings,
  - KSeF,
  - VAT OSS settings/report,
  - billing/subscription settings.
- [ ] For each item record:
  - current readiness state,
  - client-ready decision,
  - evidence required before exposure,
  - missing credential or account requirement,
  - follow-up issue if already known.
- [ ] State explicitly that no route moves to `ready` without browser smoke evidence and provider/account validation.

### Task 3: Static Cross-Check

- [ ] Confirm nav filtering uses `getVisibleNavItems`.
- [ ] Confirm command palette uses readiness-filtered nav/quick actions.
- [ ] Confirm direct dashboard routes are wrapped by `ReadinessRouteGuard`.
- [ ] Confirm provider pickers use `getVisibleProvidersByCategory` or readiness-filtered keys for invoicing providers.
- [ ] If a gap is found, add the smallest test-backed fix within OPE-253 scope.

### Task 4: Linear Follow-Up Notes

- [ ] Comment on OPE-253 with the final readiness decision and validation commands.
- [ ] If live provider validation requires owner action, record it as a Linear comment instead of blocking the whole backlog.
- [ ] Do not create duplicate issues if an existing issue already covers the follow-up:
  - OPE-246 for customer billing/subscription exposure,
  - provider certification issues for Fakturownia, wFirma, inFakt, KSeF or Stripe if they already exist.

### Task 5: Validation

- [ ] Run targeted dashboard tests:

```bash
cd /Users/rafs/praca/openoms-dev/public/apps/dashboard
npx vitest run src/lib/__tests__/readiness.test.ts --reporter=dot
```

- [ ] Run dashboard lint:

```bash
cd /Users/rafs/praca/openoms-dev/public/apps/dashboard
npm run lint:quiet
```

- [ ] Run repository diff checks:

```bash
cd /Users/rafs/praca/openoms-dev/public
git diff --check
git diff --stat
```

- [ ] Before push/PR, run full local CI:

```bash
cd /Users/rafs/praca/openoms-dev/public
./scripts/local-ci.sh
```

## Risk And Rollback

- Risk: making legal/accounting features visible too early could let customers attempt unsupported billing, KSeF or invoice-provider flows.
- Mitigation: keep all non-certified legal/accounting surfaces outside `client-ready` and add explicit regression tests.
- Risk: full/operator mode may still expose unvalidated screens.
- Mitigation: document full mode as internal validation only; `blocked` routes remain blocked in every mode.
- Rollback: revert the readiness test/doc commit. No data, API, migration or production config rollback should be required.

## Completion Criteria

- OPE-253 has a saved audit decision matrix.
- Readiness tests explicitly cover invoicing, accounting, billing, KSeF and VAT OSS.
- No gated OPE-403 child files or tasks are modified.
- No public client-ready route/provider is opened without evidence.
- Validation commands are recorded in the PR and Linear comment.
