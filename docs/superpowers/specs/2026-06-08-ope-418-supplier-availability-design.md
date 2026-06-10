# OPE-418 Supplier-Availability Policy Model — Design

**Status:** Approved design (no implementation yet)
**Date:** 2026-06-08
**Epic:** OPE-403 tor B (Fulfillment Orchestration) — followup deferred from OPE-418 (Phase 7)
**Scope decision:** availability + policy **read-model** + how dropship/backorder routing and channel-stock propagation **consume** it + stale/unknown **blocking**. The supplier-order preflight/submit/reconcile **engine is a separate later spec** and is explicitly out of scope here.

## Goal

Replace the naive single-number supplier stock (`supplier_products.stock_quantity`) with a model that captures the distinctions a real dropship/wholesale feed requires — raw supplier quantity vs the quantity it is *safe to sell*, data freshness, reservation support, and lead time — and feed that into the already-merged fulfillment orchestration so that:

- auto dropship routing only happens on **trusted, sufficient, fresh** availability;
- stale / unknown / insufficient availability creates a **typed blocker** (visible in the OPE-419 ops dashboard) instead of a silent bad auto-decision;
- raising a sales-channel stock level from supplier availability is **gated** (never auto-increase marketplace stock on stale/untrusted data), while *decreasing* it is always allowed.

## Context (already merged)

- Canonical fulfillment model (ADR-424): `fulfillment_processes` / `_units` / `_steps` / `_blockers`, tenant-scoped with FORCE RLS.
- `UnitTypeBackorder` and `UnitTypeDropship` unit types.
- Blocker codes already present: `supplier_availability_stale`, `channel_stock_stale` (category `supplier` / `integration`).
- `ClassifySupplierCapability(supplier, providerRegistered) → api | portal | manual | unsupported` (OPE-418) — decides *how* a dropship order can be placed; this design adds *whether the availability permits it*.
- Existing `supplier_products` (per `supplier × external_id`, flat `stock_quantity integer`, `last_synced_at`) and `warehouse_stock` (own stock with `quantity`/`reserved`) are left unchanged.
- `SupplierSyncWorker` syncs supplier catalogs (XML/IOF/CSV/API).

## Architecture

Two new additive tables + a pure on-read resolver, all behind a feature flag:

```
SupplierSyncWorker ──upsert──▶ supplier_availability (snapshot: raw + observational truth)
                                        │
operator/API ──CRUD──▶ supplier_availability_policy (4 scopes: supplier|product|listing|channel)
                                        │
                          ResolveAvailability(snapshot, policyChain, now)   ← PURE, on-read
                                        │  → { available_to_sell, status, blockers[] }
              ┌─────────────────────────┼─────────────────────────────┐
        dropship routing          backorder routing            channel stock propagation
        (auto-submit gate)        (ETA unit)                   (increase gate; decrease always)
```

`available_to_sell` is **computed on read** from the snapshot + the resolved policy. A policy change therefore takes effect immediately with no recompute sweep; the snapshot stays pure observational data.

**Gating:** new flag `SUPPLIER_AVAILABILITY_ENABLED` (default `false`). When off, behavior is byte-for-byte unchanged — the snapshot is not written and the resolver is not consulted; the legacy `supplier_products.stock_quantity` path stands. When on, the sync worker additionally writes snapshots and routing/propagation consult the resolver.

## Data model

### Table `supplier_availability` (snapshot)

The raw + observational availability of a supplier product at a supplier warehouse. **One row per `(tenant_id, supplier_product_id, warehouse_external_id)`** — the per-warehouse dimension is why this is a separate table and not new columns on `supplier_products` (BigBuy / La Grana split stock by warehouse).

| Column | Type | Notes |
|---|---|---|
| `id` | uuid pk | |
| `tenant_id` | uuid NOT NULL → tenants | RLS key |
| `supplier_id` | uuid NOT NULL → suppliers | denormalized for policy resolution + indexing |
| `supplier_product_id` | uuid NOT NULL → supplier_products | |
| `product_id` | uuid NULL → products | the linked OpenOMS product (nullable until mapped) |
| `warehouse_external_id` | text NOT NULL DEFAULT `''` | supplier's warehouse id; `''` for single-warehouse suppliers |
| `source_quantity` | integer NOT NULL DEFAULT 0 | raw quantity from the supplier |
| `availability_type` | text NOT NULL DEFAULT `'unknown'` | CHECK in (`exact_quantity`,`bucket`,`boolean`,`eta_only`,`unknown`) |
| `min_handling_days` | integer NULL | |
| `max_handling_days` | integer NULL | |
| `next_delivery_date` | date NULL | for `eta_only` / restock |
| `reservation_supported` | boolean NOT NULL DEFAULT false | does the supplier guarantee a reservation? |
| `freshness_observed_at` | timestamptz NOT NULL | when this snapshot was observed from the supplier |
| `source_max_stale_seconds` | integer NULL | the feed's own declared freshness SLA, if any (advisory; policy can override) |
| `last_successful_sync_id` | uuid NULL → sync_jobs | provenance |
| `raw` | jsonb NOT NULL DEFAULT `'{}'` | redacted provider payload (no secrets) |
| `created_at` / `updated_at` | timestamptz | |

- `UNIQUE (tenant_id, supplier_product_id, warehouse_external_id)` — the upsert key.
- Index `(tenant_id, supplier_id)` and `(tenant_id, product_id)` for resolution + propagation lookups.
- FORCE ROW LEVEL SECURITY + `tenant_isolation` policy on `app.current_tenant_id` (OPE-414 pattern).
- Written by `SupplierSyncWorker` (upsert) when `SUPPLIER_AVAILABILITY_ENABLED`.

### Table `supplier_availability_policy` (4-scope policy)

Tenant rules that turn raw availability into *sellable* availability. Resolved by precedence **channel > listing > product > supplier** (most specific wins); each field resolves independently up the chain (a `product`-scope row can set only `safety_buffer` and inherit the rest from the `supplier`-scope row).

| Column | Type | Notes |
|---|---|---|
| `id` | uuid pk | |
| `tenant_id` | uuid NOT NULL → tenants | RLS key |
| `scope` | text NOT NULL | CHECK in (`supplier`,`product`,`listing`,`channel`) |
| `supplier_id` | uuid NULL → suppliers | set for `supplier` and `product` scopes |
| `product_id` | uuid NULL → products | set for `product` scope (per supplier+product) |
| `listing_id` | uuid NULL → product_listings | set for `listing` scope |
| `channel` | text NULL | set for `channel` scope (integration/marketplace key) |
| `mode` | text NOT NULL DEFAULT `'auto'` | CHECK in (`auto`,`manual`,`paused`) |
| `safety_buffer` | integer NOT NULL DEFAULT 0 | subtracted from `source_quantity` |
| `freshness_window_seconds` | integer NULL | max age before stale; null ⇒ inherit, else fall back to `source_max_stale_seconds`, else a system default (e.g. 3600s) |
| `max_lead_time_days` | integer NULL | reject auto-routing when supplier handling days exceed this |
| `override_quantity` | integer NULL | when set, `available_to_sell` = this regardless of source (manual control) |
| `allow_channel_increase` | boolean NOT NULL DEFAULT false | gate: may rising supplier availability **raise** channel stock? |
| `require_reservation` | boolean NOT NULL DEFAULT false | only auto-route when `reservation_supported` |
| `require_preflight` | boolean NOT NULL DEFAULT false | only auto-route when the supplier capability supports preflight |
| `created_at` / `updated_at` | timestamptz | |

- Partial unique indexes per scope so a tenant has at most one row per scope target, e.g.
  `UNIQUE (tenant_id, supplier_id) WHERE scope='supplier'`,
  `UNIQUE (tenant_id, supplier_id, product_id) WHERE scope='product'`,
  `UNIQUE (tenant_id, listing_id) WHERE scope='listing'`,
  `UNIQUE (tenant_id, channel) WHERE scope='channel'`.
- FORCE RLS + `tenant_isolation` (OPE-414 pattern).
- **Override audit:** changes to `override_quantity` / `mode` are written to the existing `audit_log` (`action = supplier_availability.policy_override`), because the research requires that an active override is never silently overwritten by automation.

## Resolver (pure, on-read)

```
ResolveAvailability(snapshot, policyChain, requestedQty, now) → AvailabilityDecision {
    available_to_sell int
    status            // trusted | stale | unknown | paused | manual_override
    auto_routable     bool   // safe to auto-submit a dropship order
    channel_increase_allowed bool
    blocker           *BlockerSignal   // nil when fine
}
```

Resolution order (each field independently, most-specific scope first): `effective = resolve(channel, listing, product, supplier, systemDefault)`.

`status` and `available_to_sell`:
- `override_quantity` set → `status = manual_override`, `available_to_sell = override_quantity`.
- `mode = paused` → `status = paused`, `available_to_sell = 0` (for auto), blocker = none (intentional operator pause).
- `availability_type = unknown` → `status = unknown`, `available_to_sell = 0`, blocker = `supplier_availability_stale` (unknown is treated as untrusted).
- `now − freshness_observed_at > freshness_window` → `status = stale`, `available_to_sell = 0`, blocker = `supplier_availability_stale`.
- otherwise → `status = trusted`, `available_to_sell = max(0, source_quantity − safety_buffer)`.

`auto_routable` (auto dropship submit) = `status = trusted`
  ∧ `available_to_sell ≥ requestedQty`
  ∧ (`¬require_reservation` ∨ `reservation_supported`)
  ∧ (`¬require_preflight` ∨ supplier capability supports preflight)
  ∧ (`max_lead_time_days` is null ∨ `max_handling_days ≤ max_lead_time_days`).

  `require_preflight` defaults to `false` (so by default it never blocks). The preflight
  capability itself is modeled by the deferred supplier-order engine spec; until that
  lands, the gate conservatively treats preflight as **unsupported** — so a tenant that
  sets `require_preflight = true` gets a manual unit rather than an auto-submit, which is
  the safe default.
When `status = trusted` but `available_to_sell < requestedQty` and `next_delivery_date` is set → not auto-routable, signal **backorder** (not a hard blocker). When insufficient and no ETA → blocker `supplier_availability_insufficient`.

`channel_increase_allowed` = `allow_channel_increase` ∧ `status = trusted` ∧ lead-time within policy. A channel-stock **decrease** is always allowed regardless (safety-first).

The resolver is a pure function over its inputs (snapshot + resolved policy + now + requested qty), so all the precedence and arithmetic is unit-testable with no database.

## Consumption by the merged orchestration

- **Dropship routing** (`dropship_service` / `fulfillment_unit_service`, using the existing `ClassifySupplierCapability`): before auto-submitting a dropship unit, call the resolver. `auto_routable = false` → create the typed blocker on the unit (`supplier_availability_stale` / `supplier_availability_insufficient`) and leave the unit waiting/manual instead of silently auto-submitting. `auto_routable = true` → proceed (the actual supplier-order submit remains the separate engine).
- **Backorder routing**: insufficient `available_to_sell` + `next_delivery_date` present → `UnitTypeBackorder` unit carrying the ETA; resumes when a later sync raises availability or a goods receipt arrives.
- **Channel-stock propagation**: the existing `StockSyncWorker` reads `available_to_sell` as the sellable quantity for supplier-backed products and applies `channel_increase_allowed` — an increase to a marketplace level only goes out when the gate is open; a decrease always goes out. *This spec provides the decision + gate; the push execution stays in the existing stock-sync — no new propagation path is introduced here.*

## Blockers

- Reuse `supplier_availability_stale` (covers stale **and** unknown) and `channel_stock_stale`.
- Add one new code `supplier_availability_insufficient` (category `supplier`) — a distinct operator signal for "trusted but not enough stock and no ETA", separate from a freshness problem.

## Rollout / safety

- Flag `SUPPLIER_AVAILABILITY_ENABLED` (env, default `false`). Off ⇒ snapshot not written, resolver not consulted, legacy `stock_quantity` path unchanged.
- Additive only: new tables, one new blocker code (no DB CHECK on `fulfillment_blockers.code`, so no constraint migration). Migration `000041`, FORCE RLS like the other fulfillment/supplier tables. Blue-green safe (no destructive ops).
- Reversible: turning the flag off stops consultation without deleting snapshots/policies.

## Testing

- **Resolver unit tests (no DB):** 4-scope precedence chain (field-by-field inheritance); `available_to_sell` math for buffer / override / paused / stale / unknown / trusted; `auto_routable` gates (reservation, preflight, lead-time, sufficiency); backorder vs insufficient branch; `channel_increase_allowed` gate; decrease-always rule.
- **DB integration tests:** snapshot upsert idempotency on the unique key; per-tenant RLS isolation on both tables; dropship routing creates the correct blocker when stale/unknown/insufficient and proceeds when trusted+sufficient; channel increase suppressed when `allow_channel_increase=false`; flag-off = no snapshot written, no resolver consulted.

## Explicitly deferred (separate later specs)

- The supplier-order **engine**: prepare / preflight / submit / reconcile (4-phase), multi-warehouse order split, partial-error handling.
- The actual channel write-back **push mechanics** beyond the gate decision (stays in the existing stock-sync).
- Per-channel availability *delta* webhooks / push-based supplier updates (polling snapshot only here).
- A dashboard UI for editing availability policies (API + model only here).
