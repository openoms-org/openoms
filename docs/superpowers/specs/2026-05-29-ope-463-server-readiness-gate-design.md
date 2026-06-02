# OPE-463 — Server-side enforcement of feature readiness gating — Design

Status: Proposed for review (GATE 1)
Date: 2026-05-29
Lane: L (auth/cross-cutting)

## Problem
The dashboard has a frontend-only readiness registry (`apps/dashboard/src/lib/readiness.ts`) that hides non-ready features in the UI (nav, command palette, route guard, provider pickers) based on a deployment-level surface mode. The backend has **no** readiness concept (except Pick & Pack, which is RBAC-gated by OPE-436). Every non-ready endpoint is fully reachable by a direct authenticated API call. Hiding in the UI is not enforcement.

## Goal & scope
Enforce the readiness registry **server-side** so that, in `client-ready` surface mode, only `ready` features are reachable over the API — closing the gap where staged/incomplete features are callable directly. Single source of truth shared by frontend and backend (no drift). Global (deployment-level), not per-tenant.

In scope: route-group readiness gate + provider-value validation (carrier/marketplace) + shared source of truth + CI drift guard + tests.
Out of scope (follow-ups): runtime exposure of readiness via `/v1/config/public`; per-tenant readiness. (Tracked separately.)

## Decisions (confirmed at GATE-1 pre-brief)
1. **Source of truth:** one canonical JSON keyed by feature-id (`packages/readiness/readiness.json`); `readiness.ts` derives its maps from it, Go embeds it via `//go:embed`. Flip a feature to `ready` in ONE place.
2. **HTTP status for a gated route hit directly:** `404` + JSON `{ "error": "feature_not_available" }` (no capability-inventory leak; consistent for `blocked` and non-ready-in-client-ready).
3. **Provider-value gating is in scope:** `POST /v1/integrations` and `POST /v1/shipments` reject non-ready carrier/marketplace providers per the active surface (`422 { "error": "provider_not_available" }` — the endpoint exists, the value is disallowed → validation semantics).
4. Surface mode is **binary**, mirroring the frontend exactly: `client-ready` → only `ready` passes; `full` → all except `blocked`; `blocked` → always blocked.

## Architecture

### 1. Canonical source of truth — `packages/readiness/readiness.json`
```jsonc
{
  "providers": {            // provider-key -> readiness state (carriers + marketplaces + invoicing)
    "allegro": "ready", "inpost": "ready",
    "olx": "controlled", "dhl": "controlled", "dpd": "controlled", "gls": "controlled",
    "amazon": "beta", "ebay": "beta", "ups": "beta", "fedex": "beta", "...": "..."
  },
  "features": {             // feature-id -> { state, routes (dashboard), endpoints (/v1 prefixes) }
    "invoicing":  { "state": "controlled", "routes": ["/invoices", "/invoicing"], "endpoints": ["/v1/invoices"] },
    "warehouses": { "state": "controlled", "routes": ["/settings/warehouses"], "endpoints": ["/v1/warehouses"] },
    "stocktakes": { "state": "verify", "routes": ["/stocktakes"], "endpoints": ["/v1/stocktakes"] },
    "repricing":  { "state": "beta", "routes": ["/repricing"], "endpoints": ["/v1/repricing"] }
    // ...one entry per gated feature; values taken verbatim from current readiness.ts
  }
}
```
- `state` vocabulary unchanged: `ready | controlled | verify | beta | blocked`.
- Migration of the existing data is mechanical: today's `NAV_ROUTE_READINESS` (route→state) and `PROVIDER_READINESS` (provider→state) are reshaped into this file. No semantic change to which feature is in which state.

### 2. Frontend — derive, don't duplicate
- `readiness.ts` imports `readiness.json` and **builds** `NAV_ROUTE_READINESS` from `features[].routes`→`state` and uses `providers` directly for `PROVIDER_READINESS`. All existing exported functions (`isRouteAccessible`, `getVisibleNavItems`, `getVisibleProviderKeys`, `isShipmentProviderSelectable`, …) keep their signatures and behavior. `getDashboardSurfaceMode()` unchanged (reads `NEXT_PUBLIC_OPENOMS_DASHBOARD_SURFACE`).
- Existing `readiness.test.ts` must still pass (it pins concrete expected provider/route lists) — acts as a regression lock on the JSON migration.

### 3. Backend — config + middleware + embed
- **Config:** add `APISurfaceMode string` to `config.Config`, env `OPENOMS_API_SURFACE` (values `client-ready` | `full`, default `client-ready`). Documented to be set to the SAME value as `NEXT_PUBLIC_OPENOMS_DASHBOARD_SURFACE` in the Helm overlay.
- **Embed:** a new Go package `internal/readiness` does `//go:embed readiness.json` (copy or build-step from `packages/readiness/readiness.json` — see Open item R1), parses into `map[featureID]State` + `map[providerKey]State`, and exposes `IsFeatureEnabled(featureID, mode)` and `IsProviderEnabled(providerKey, mode)` (same binary logic as frontend `isReadinessVisible`).
- **Middleware factory** `middleware.RequireFeature(featureID string)` — mirrors `requirePermission` (the OPE-436 pattern). It is constructed with the surface mode at wire-up (closure over `deps.Config.APISurfaceMode` + the embedded map). On a request, if the feature is not enabled for the mode → write `404 {"error":"feature_not_available"}` and stop; else `next`.
- **Wiring in `router.go`:** add exactly one `r.Use(requireFeature("<feature-id>"))` as the FIRST `r.Use` inside each non-ready route group (before `requirePermission`), so the stack is `JWTAuth → TenantPlanGuard → MaxBodySize → RequireFeature → RequirePermission → handler`. Pick & Pack keeps its `requirePermission(PermWarehousesManage)` and ADDS `requireFeature("pick_pack")` (layered).

### 4. Provider-value validation
- `POST /v1/integrations` (create): validate `provider` against `IsProviderEnabled(provider, mode)`; reject non-ready with `422 {"error":"provider_not_available"}`. (Existing integrations are untouched — only creation is gated, matching the frontend provider-picker behavior.)
- `POST /v1/shipments` (and label/dispatch create paths): validate the carrier `provider`/`carrier` field the same way. `manual` is always allowed (mirrors `isShipmentProviderSelectable`).
- Implemented in the service/handler validation layer (not route middleware) since it is value-level.

### 5. CI drift guard (the anti-stopgap)
A Go test `internal/router/readiness_coverage_test.go` enforces bidirectional coverage:
- **Every** `requireFeature("id")` call in `router.go` references an `id` that exists in `readiness.json`.
- **Every** feature in `readiness.json` whose `state != "ready"` has its `endpoints[]` covered by a `requireFeature` gate on the matching route group (unless listed in an explicit `excludedEndpoints` allowlist — see exclusions).
- Adding a new non-ready endpoint without a gate, or renaming a feature, **fails CI**. This is what makes the gate a durable solution rather than a snapshot.

## Exclusions (must NOT be gated)
- **Public / token-auth routes:** `/v1/webhooks/*`, `/v1/public/*`, `/v1/billing/*` (public), `/v1/config/public`, `/v1/feeds/*` (token), `/v1/supplier-portal/*` (token). These have no JWT and/or are infrastructure.
- **Shared utilities consumed by `ready` surfaces:**
  - `/v1/barcode/{code}` — used by the `ready` orders pack flow AND the `verify` packing page → ungated.
  - `/v1/stats/*` — powers the `ready` home dashboard `/` (the `/reports` dashboard route is frontend-gated, but the API is shared) → ungated.
- These exclusions live in the `excludedEndpoints` allowlist consulted by the drift guard, so they are explicit and reviewed, not silent.

## Edge cases to resolve in the JSON
- `/v1/ai` is absent from the current registry → add feature `ai` with state `beta` (gate it). [confirm]
- `/v1/audit` maps to `/audit` = `ready` in the current registry (flagged as possibly unintended). Treat the registry as source of truth → `ready`, ungated. Flagged for owner confirmation; changing it is a one-line JSON edit.

## HTTP contract
- Gated route hit directly: `404` `{"error":"feature_not_available"}`.
- Non-ready provider on create: `422` `{"error":"provider_not_available"}`.
- `full` mode: behaves exactly as today (no gating except `blocked`).

## Testing plan
- `internal/router/readiness_gate_test.go` (router_test pkg, `httptest` + `chi` + `testify`, `newMinimalRouterDeps` helper): for `client-ready` — each non-ready group returns 404; for `full` — same group passes to handler; `blocked` always 404; an excluded route (`/v1/barcode`, `/v1/stats`) passes in both modes; provider-value create returns 422 for a non-ready provider and passes for a ready one.
- `internal/router/readiness_coverage_test.go` — the drift guard (above).
- Frontend: a `readiness.ts` derivation test (derived `NAV_ROUTE_READINESS`/provider lists equal the pre-migration values) + existing `readiness.test.ts` must pass unchanged.

## Rollout / backward-compat
- Default `client-ready` matches the current frontend default → on deploy, non-ready endpoints start returning 404. This is the intended behavior; the exclusions guarantee no `ready` surface breaks.
- **Deploy dependency (enterprise repo):** set `OPENOMS_API_SURFACE` in `values-production.yaml` / `values-staging.yaml` to match `NEXT_PUBLIC_OPENOMS_DASHBOARD_SURFACE`. Coordinated change; tracked as a checklist item in the OPE-463 PR (and a matching enterprise PR if the var isn't already plumbed).
- **Escape hatch / rollback:** set `OPENOMS_API_SURFACE=full` to disable all readiness gating instantly without a code change; or remove the specific `requireFeature` line.

## Risks
- **Wrongly gating an endpoint the `ready` surface needs** → caught by tests + the explicit exclusions + the `full` escape hatch. Mitigation: enumerate exclusions from the understand phase (barcode, stats) and add a pre-merge check that `ready` dashboard pages don't call gated endpoints.
- **JSON migration changes a state by accident** → `readiness.test.ts` pins the expected lists and locks this.
- **Embed/copy coupling** between `packages/readiness/readiness.json` and the Go embed (Open item R1).

## Open items (for the plan)
- **R1 — embed path:** Go `//go:embed` cannot reach `../../packages/`. Resolve by: (a) a tiny build/codegen step copying `readiness.json` into `internal/readiness/` before build, or (b) placing the canonical file inside the api-server module and having the dashboard import it via a relative path / build copy. Decide in writing-plans; (a) with a `go generate` + a checked-in copy + a CI freshness check is the likely answer.
- **R2 — feature-id list:** the exact, complete feature-id set + endpoint prefixes is enumerated from `readiness.ts` + `router.go` during implementation; the understand-phase mapping is the seed.

## Out of scope (follow-up OPE)
- Runtime readiness via `/v1/config/public` (true single runtime authority; embed already gives build parity).
- Per-tenant readiness / plan-based feature flags.
