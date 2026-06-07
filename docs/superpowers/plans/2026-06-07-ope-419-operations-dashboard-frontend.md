# OPE-419 Operations Control Tower & Fulfillment Detail (Frontend) Implementation Plan

> **For agentic workers:** Steps use checkbox (`- [ ]`) syntax. This is an ADDITIVE slice — do not remove or rewrite the existing heuristic operations dashboard.

**Goal:** Surface the process-backed operations control tower (6 buckets), an actionable exceptions feed with drilldown, and an order/process fulfillment detail timeline (units → steps, blockers, provider attempts) with resolve/retry actions — all reading the OPE-419 backend (#540), rendering clean empty-states while `FULFILLMENT_PROCESS_ENABLED` is off.

**Architecture:** New TS types mirroring the Go DTOs (no invented fields) → React Query hooks (read + 2 mutations) → presentational components (summary buckets, exceptions feed, detail panel rendered in a Sheet) → a new `/operations/fulfillment` route + an additive panel on the existing dashboard home. Existing heuristic dashboard (`useOperationsDashboard`, `OrchestrationMap`, `OperationalExceptions`) is untouched.

**Tech Stack:** Next.js 16 App Router, React Query v5, next-intl (keys in `dashboard.json` under `fulfillment.*` namespace), shadcn/ui (Sheet, AlertDialog/ConfirmDialog, Badge, Tooltip+TooltipProvider), sonner toast, Vitest, Playwright.

---

## Backend contract (verified, do not invent)

- `GET /v1/fulfillment/processes?aggregate_status&health_status&limit&offset` → `ListResponse<FulfillmentProcess>` `{items,total,limit,offset}`. Permission `orders:view`.
- `GET /v1/fulfillment/processes/{id}` → `ProcessDetail` `{process, units:[{unit,steps[]}], blockers[], provider_attempts[]}`. 404 -> not found.
- `GET /v1/fulfillment/orders/{orderID}` → `ProcessDetail` (same shape). 404 when order has no process.
- `GET /v1/operations/summary` → `{buckets:{ready,processing,stuck,blocked,provider_issue,missing_data:int}, total:int}`.
- `GET /v1/operations/exceptions?limit` → `{items:ExceptionItem[], total:int}` where `ExceptionItem={process, bucket, top_blocker?}`.
- `GET /v1/operations/integration-capability-summary?scan_limit` → `IntegrationCapabilitySummaryResult` (snake_case fields).
- `POST /v1/fulfillment/blockers/{id}/resolve` → `FulfillmentBlocker`. Permission `orders:edit`.
- `POST /v1/fulfillment/steps/{id}/retry` → `FulfillmentStep`. 400 when not retryable. Permission `orders:edit`.

Model field names (snake_case JSON) — `FulfillmentProcess{id,tenant_id,order_id,aggregate_status,health_status,metadata?,created_at,updated_at}`; `FulfillmentUnit{...,process_id,parent_unit_id?,unit_type,status,...}`; `FulfillmentStep{...,unit_id,step_key,status,attempts,...}`; `FulfillmentBlocker{...,process_id,unit_id?,code,category,status,description,...,resolved_at?}`; `ProviderAttempt{id,tenant_id,process_id?,provider,operation,status,request_id?,correlation_id?,raw_provider_status?,result_redacted?,payload_hash?,error_code?,created_at}`.

---

## File structure

- Create `src/types/fulfillment.ts` — TS types mirroring DTOs + bucket/status string unions + i18n key helpers.
- Create `src/hooks/use-fulfillment.ts` — query keys + 6 read hooks + 2 mutation hooks.
- Create `src/components/dashboard/operations-bucket-summary.tsx` — 6-bucket control-tower grid (loading/empty/error).
- Create `src/components/dashboard/fulfillment-exceptions-feed.tsx` — actionable exceptions with top blocker + drilldown trigger.
- Create `src/components/fulfillment/fulfillment-detail-panel.tsx` — Sheet timeline (units→steps, blockers, attempts) + resolve/retry.
- Create `src/components/fulfillment/fulfillment-status-badge.tsx` — shared status/health/bucket badges (TooltipProvider-safe).
- Create `src/app/(dashboard)/operations/fulfillment/page.tsx` — dedicated process-backed control tower route.
- Modify `src/app/(dashboard)/page.tsx` — add additive `<FulfillmentControlTowerPanel/>` section below existing dashboard (no removals).
- Create `src/components/dashboard/fulfillment-control-tower-panel.tsx` — the additive home-page panel (summary strip + small exceptions preview + link to full page).
- Modify `src/lib/nav-items.ts` — add a nav entry to `/operations/fulfillment`.
- Modify `messages/en/dashboard.json` + `messages/pl/dashboard.json` — add `fulfillment.*` keys (en==pl parity).
- Modify `messages/en/common.json` + `messages/pl/common.json` — add `navigation.fulfillmentOps` label.
- Create `src/components/fulfillment/__tests__/fulfillment.test.tsx` — Vitest: bucket rendering, empty-state, exception drilldown target, resolve/retry wiring.
- Create `e2e/fulfillment-operations.spec.ts` — Playwright (mocked routes): blocked, waiting_external, completed flows.

---

## Tasks (executed inline)

### Task 1: Types
Create `src/types/fulfillment.ts` with exact DTO mirrors, union types for aggregate/health/unit/step/blocker statuses + the 6 operator buckets + blocker codes/categories. Re-export from `src/types/api.ts` barrel if present.

### Task 2: Hooks
Create `src/hooks/use-fulfillment.ts`: `fulfillmentKeys` factory; `useFulfillmentProcesses(params)`, `useFulfillmentProcess(id)`, `useOrderFulfillment(orderId)`, `useOperationsSummary()`, `useFulfillmentExceptions(limit)`, `useIntegrationCapabilitySummary()`, `useResolveBlocker()`, `useRetryStep()`. Mutations invalidate process detail + summary + exceptions. `enabled` guards on ids.

### Task 3: Status/badge primitives
Create `fulfillment-status-badge.tsx` mapping statuses/health/buckets to Badge variants + i18n label keys. No bare `<Tooltip>` without provider.

### Task 4: Bucket summary + exceptions feed components
Create `operations-bucket-summary.tsx` (6 buckets, skeleton, all-zero empty-state) and `fulfillment-exceptions-feed.tsx` (items with top-blocker reason, onSelect drilldown callback, empty-state, skeleton, error).

### Task 5: Detail panel (Sheet)
Create `fulfillment-detail-panel.tsx`: takes `{processId?|orderId?, open, onOpenChange}`, fetches detail, renders process header (aggregate+health badges), blockers (with Resolve action), units→steps timeline (with Retry on failed/blocked steps), provider attempts. Confirm dialog + toast + invalidation. Loading/empty/error/404 states.

### Task 6: Home panel + dedicated route
Create `fulfillment-control-tower-panel.tsx` (summary strip + top-3 exceptions preview + "view all" link, opens detail panel on exception click). Create `/operations/fulfillment/page.tsx` (full control tower: buckets + capability summary + full exceptions list + detail panel). Wire panel into dashboard home additively. Add nav item.

### Task 7: i18n
Add `dashboard.fulfillment.*` keys to en+pl `dashboard.json`; add `navigation.fulfillmentOps` to en+pl `common.json`. Maintain en==pl key parity.

### Task 8: Vitest
Bucket rendering (counts + all-zero empty), exceptions feed drilldown target (onSelect called with process id), resolve/retry action wiring (mutation called), empty-state for processes.

### Task 9: Playwright
`e2e/fulfillment-operations.spec.ts` using `page.route` to mock summary/exceptions/detail for blocked, waiting_external, completed processes + resolve/retry POSTs. Uses `gotoWithAuth` + Polish UI labels. Note: requires live backend session to actually run; written + documented as not-executed-here.

### Task 10: Validate + commit
`npx next build`, `npx eslint --quiet src/`, `npx tsc --noEmit` (report pre-existing), `npx vitest run --reporter=dot`, i18n parity check. Commit (no push).
