# OPE-165 API Types Split Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Split the monolithic dashboard `src/types/api.ts` file into focused domain type modules while preserving the existing `@/types/api` import path.

**Architecture:** Move related exported types into 12 domain files under `apps/dashboard/src/types/`. Keep `api.ts` as a barrel export so no component or hook imports need to change. Use explicit `import type` edges between domain files where a moved type references another domain type.

**Tech Stack:** TypeScript 6, Next.js 16 dashboard, Vitest, ESLint.

---

## Scope

- Public repo only.
- No API behavior changes.
- No component import rewrites.
- No runtime code changes.

## Target Files

- `apps/dashboard/src/types/api.ts` becomes a barrel export.
- New domain files:
  - `apps/dashboard/src/types/common.ts`
  - `apps/dashboard/src/types/auth.ts`
  - `apps/dashboard/src/types/billing.ts`
  - `apps/dashboard/src/types/orders.ts`
  - `apps/dashboard/src/types/shipments.ts`
  - `apps/dashboard/src/types/products.ts`
  - `apps/dashboard/src/types/integrations.ts`
  - `apps/dashboard/src/types/warehouse.ts`
  - `apps/dashboard/src/types/customers.ts`
  - `apps/dashboard/src/types/invoices.ts`
  - `apps/dashboard/src/types/automation.ts`
  - `apps/dashboard/src/types/operations.ts`
- Update this plan file with validation evidence.

## Domain Mapping

- `common.ts`: shared `Address`, pagination/list response, API error, WebSocket event.
- `auth.ts`: login/register/token/2FA/user/tenant/roles.
- `billing.ts`: public plan, checkout session, subscription status.
- `orders.ts`: orders, order imports, returns, order groups, packing station request/response, bulk status.
- `shipments.ts`: shipments, labels, dispatch orders, InPost points, shipping rate shopping.
- `products.ts`: products, variants, categories, listings, bundles, product import/feed, forecast, repricing, listing sync, stock sync.
- `integrations.ts`: integrations, webhooks, sync jobs.
- `warehouse.ts`: suppliers, supplier portal, BTP wizard, warehouses, inventory, warehouse documents, stocktakes, purchase orders, pick and pack.
- `customers.ts`: customers, customer imports, segments, loyalty.
- `invoices.ts`: invoices, invoicing settings, KSeF, exchange rates, VAT OSS, payment reconciliation.
- `automation.ts`: automation rules, workflow builder, message templates.
- `operations.ts`: dashboard stats, advanced reports, carbon, audit log, company/settings/email/SMS/print templates, AI, marketing, helpdesk, onboarding.

## Implementation Steps

- [x] Create the branch and mark OPE-165 In Progress.
- [x] Extract section ranges from `api.ts` into the domain files above.
- [x] Add `import type` dependencies between domain files.
- [x] Replace `api.ts` with barrel exports for the 12 domain files.
- [x] Run TypeScript/Vitest/ESLint checks and fix compile-only type issues.
- [x] Run full public local CI before push.
- [ ] Open PR with `Docs updated` section and no component import changes.

## Validation Plan

- `cd apps/dashboard && npx tsc --noEmit`
- `cd apps/dashboard && npm run test:quiet`
- `cd apps/dashboard && npm run lint:quiet`
- `./scripts/local-ci.sh`
- `git diff --check`

## Validation Evidence

- `cd apps/dashboard && npx tsc --ignoreConfig --noEmit --strict --module esnext --moduleResolution bundler --target ES2018 --lib DOM,DOM.Iterable,ESNext src/types/*.ts` — passed.
- `cd apps/dashboard && npm run test:quiet` — passed, 55 test files / 283 tests.
- `cd apps/dashboard && npm run lint:quiet` — passed.
- `./scripts/local-ci.sh` — passed, all 10 checks, 84s total.
- `git diff --check` — passed.

Note: full dashboard `npx tsc --noEmit` currently includes existing test-file TypeScript failures unrelated to this refactor, so the type-split validation uses a targeted `src/types/*.ts` compile plus the mandatory full local CI.

## Risk And Rollback

- Risk: cross-domain type references can be missed during the split.
- Mitigation: keep `api.ts` as the public barrel, use `import type`, and rely on `tsc --noEmit` plus full local CI.
- Rollback: revert the PR; no data or runtime migration is involved.
