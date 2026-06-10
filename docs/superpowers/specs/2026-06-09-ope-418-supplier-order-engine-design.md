# OPE-418/Phase-7 Supplier-Order Engine — Design

**Status:** Approved design (no implementation yet)
**Date:** 2026-06-09
**Epic:** OPE-403 tor B (Fulfillment Orchestration) — Phase 7, the OPE-418 supplier-order followup
**Scope decision:** the **app-side** engine that turns a routable dropship unit into a supplier order through **prepare → preflight → submit → reconcile**, for API-capable suppliers, and routes portal/manual/email/unsupported suppliers to an explicit operator step + typed blocker (resolved via the EXISTING supplier portal / ops dashboard — NO new task-queue subsystem). Generic supplier-provider interface; per-vendor adapters (beyond the verified btp) and any external credentials are out of this spec.

## Goal

Place dropship orders with suppliers safely and observably: only auto-submit when availability is trusted and the supplier supports it; surface every failure class as a typed, actionable blocker instead of a silent adapter-log error; never double-submit; and reconcile the supplier's order id / status / tracking back into the canonical order. Suppliers without API support get an explicit operator step, not a silent drop.

## Context (already merged)

- **Supplier-availability read-model** (OPE-418 #556): `ResolveAvailability` → `available_to_sell`, `auto_routable`, `require_preflight`/`require_reservation`, stale/unknown/insufficient blockers; `ClassifySupplierCapability(supplier, providerRegistered) → api | portal | manual | unsupported`.
- **Fulfillment orchestration** (ADR-424): `orchestration_outbox` (idempotent enqueue, retry/backoff, permanent→`fulfillment_blocker`), `orchestration_attempts` (+ `external_execution_id` column from OPE-421), `OrchestrationWorker` + `OrchestrationDispatcher` (event_type→handler), gated `ORCHESTRATION_WORKER_ENABLED`.
- **Units/steps/blockers**: `UnitTypeDropship`/`UnitTypeBackorder`; step keys `StepCreateDropshipOrder`, `StepPreflightSupplierOrder`, `StepConfirmSupplierOrder`, `StepCreatePurchaseOrder`, `StepReceivePurchaseOrder`; the OPE-418 `gateDropshipAvailability` already creates the dropship unit and (when not routable) the availability blocker.
- **`dropship_orders`** table (status, `supplier_reference`, `tracking_number`, `carrier`, `sent_at`/`confirmed_at`/`shipped_at`/`delivered_at`, `total_cost`, `currency`) — the canonical supplier-order record; `purchase_orders` is the symmetric replenishment record.
- **`integration.SupplierProvider`** interface — currently only `CreateOrder(req) (*SupplierOrderResult, error)`; the verified `btp` adapter implements it.
- **OPE-417 marketplace-tracking** path — reused to push reconciled tracking to the originating marketplace.

## Architecture

```
dropship unit auto_routable (OPE-418 availability gate)
   │
   ├─ capability=api ─────────▶ enqueue orchestration_outbox  supplier.order.submit  (gated)
   │                                  │
   │                          OrchestrationWorker → SupplierOrderHandler (one idempotent dispatch)
   │                                  prepare → preflight → submit   (each = a fulfillment step)
   │                                  └─ stores supplier_reference + status on dropship_orders
   │
   ├─ capability=portal|manual|email ▶ manual fulfillment step + supplier_manual_submission_required blocker
   │                                   (operator submits out-of-band, records the reference via the
   │                                    existing supplier portal / ops dashboard → feeds the poller)
   │
   └─ capability=unsupported ────────▶ integration_capability_missing blocker

SupplierOrderStatusPoller (recurring worker, like TrackingPoller, gated)
   reads supplier order status for non-terminal dropship_orders → reconcile (status/tracking) → step + unit transition
```

All behind `SUPPLIER_ORDER_ENABLED` (default `false`): off ⇒ the gate's auto-submit enqueue is skipped, the handler + poller are unregistered, behavior unchanged (the OPE-418 availability gate itself is unaffected).

**Relationship to the OPE-418 gate:** today `gateDropshipAvailability` returns "held" (with a blocker / backorder unit) when a dropship unit is not routable, and otherwise simply falls through — there is no pre-existing automatic supplier submit. This engine adds the submit at exactly that fall-through point, gated: when `SUPPLIER_ORDER_ENABLED` is on and the supplier capability is `api`, it enqueues `supplier.order.submit`; when off, the fall-through stays the no-op it is today. So this is purely additive on top of OPE-418.

## Adapter contract (optional capability sub-interfaces)

The core `SupplierProvider` (CreateOrder) is unchanged. Two **optional** sub-interfaces express per-operation capability; the engine type-asserts and falls back when absent — so the `btp` adapter is untouched and a new adapter opts into preflight/status by implementing them:

```go
type SupplierPreflighter interface {
    Preflight(ctx context.Context, req SupplierOrderRequest) (*SupplierPreflightResult, error)
}
type SupplierStatusReader interface {
    GetOrderStatus(ctx context.Context, externalID string) (*SupplierOrderStatus, error)
}
```

`SupplierPreflightResult` carries: accepted total, split lines, missing-field list, business errors. `SupplierOrderStatus` carries: a raw provider status + tracking number/carrier (mapped to canonical downstream). "Supports preflight/status" = "implements the sub-interface" — this is the engine's per-operation capability, finer than the coarse api/portal/manual/unsupported class.

## The four phases (SupplierOrderHandler, one idempotent dispatch)

Registered on the dispatcher for `supplier.order.submit` (only when `ORCHESTRATION_WORKER_ENABLED` AND `SUPPLIER_ORDER_ENABLED`). Each phase is recorded as a fulfillment step; a phase that cannot proceed creates a typed blocker and stops (the unit waits); transport errors are normal retryable attempt failures (existing backoff).

1. **Prepare** — assemble the `SupplierOrderRequest` from the order + the unit's dropship line(s) + ship-to. Validate required inputs (a resolvable SKU/EAN identity, a complete address). Missing field → `supplier_order_missing_data`; an SKU/EAN that doesn't resolve to a single supplier product → `supplier_order_ambiguous_sku`. Stop on either.
2. **Preflight** (`StepPreflightSupplierOrder`) — if the adapter implements `SupplierPreflighter`, call it (stock/price/address/carrier/payment/split). A provider business rejection → `supplier_order_rejected`; a transport error → retryable. If the adapter does NOT implement it: skip — unless the availability policy's `require_preflight` is set (OPE-418), in which case route to the manual branch (`supplier_manual_submission_required`).
3. **Submit** (`StepCreateDropshipOrder`) — **idempotency**: if `dropship_orders.supplier_reference` is already set for this unit's order+supplier, skip re-submit (already placed). Otherwise call `CreateOrder(req)` with a deterministic idempotency key (`supplier.order:<dropship_unit_id>`) which the adapter forwards to the supplier API where supported. On success → store `supplier_reference` (external order id) + status + `sent_at` on `dropship_orders`, record the submit attempt's external id on `orchestration_attempts.external_execution_id`, mark the step succeeded, transition the unit to waiting_external (awaiting supplier confirmation/status). Failure classes: `awaiting_payment` → `supplier_payment_awaiting` + waiting; an accepted partial/split → `supplier_partial_fulfillment` (v1: an operator-review blocker — multi-document split execution is deferred); business rejection → `supplier_order_rejected`; transport error → retryable.

## Reconcile (SupplierOrderStatusPoller, recurring worker)

A recurring worker (mirroring `TrackingPoller`), gated. For `dropship_orders` in a non-terminal supplier state whose adapter implements `SupplierStatusReader`: poll `GetOrderStatus(supplier_reference)` → map the raw provider status to a canonical status (raw preserved alongside) → update `dropship_orders.status`/`tracking_number`/`carrier`/`confirmed_at`/`shipped_at`, record `StepConfirmSupplierOrder`, transition the unit; an unmapped external status → `external_status_unmapped`. Reconciled tracking is pushed to the originating marketplace via the OPE-417 path. Suppliers without `SupplierStatusReader` (manual/portal): the operator advances status via the existing supplier portal / ops dashboard (no automatic poll).

## Failure taxonomy → blockers

The research spec's 8 missing-data classes map to ADR-424 blocker codes — reuse where one exists, add the supplier-order-specific ones (all category `supplier`):

| Research class | Blocker code | New? |
|---|---|---|
| missing_required_input | `supplier_order_missing_data` | **new** |
| ambiguous_identity | `supplier_order_ambiguous_sku` | **new** |
| provider_business_error | `supplier_order_rejected` | **new** |
| payment-awaiting | `supplier_payment_awaiting` | **new** |
| partial/split (v1 review) | `supplier_partial_fulfillment` | **new** |
| manual_action_required | `supplier_manual_submission_required` | **new** |
| missing_mapping | `external_status_unmapped` | exists |
| unsupported_capability | `integration_capability_missing` | exists |
| stale_data | `supplier_availability_stale` | exists |
| provider_transport_error | (retryable attempt; existing permanent→blocker on exhaustion) | exists |

## Idempotency & storage

- **Idempotency:** the outbox idempotency key dedups the submit event; the local `supplier_reference`-present guard prevents a re-dispatch from double-creating; the deterministic key passed to `CreateOrder` lets the supplier API dedup where it supports idempotency keys.
- **Storage:** reuse `dropship_orders` (no new table) — `supplier_reference` is the external order id, plus status/tracking/carrier/*_at. The fulfillment unit + steps + blockers are the timeline; the submit attempt's external id lives on `orchestration_attempts.external_execution_id`.
- **No migration** expected: the new blocker codes are app-validated (no DB CHECK on `fulfillment_blockers.code`), `dropship_orders` already has the needed columns, and idempotency uses existing fields + the outbox key. (If implementation finds a genuinely required column — e.g. a canonical `external_status` distinct from the free-text `status` — add a single additive column then.)

## Gating / safety

- `SUPPLIER_ORDER_ENABLED` (env, default `false`). Off ⇒ the availability gate's auto-submit enqueue is skipped, the `SupplierOrderHandler` + `SupplierOrderStatusPoller` are unregistered → byte-for-byte unchanged.
- Additive only. No durable supplier mutation happens outside the gated, idempotent handler; manual/portal/unsupported suppliers never auto-submit.

## Testing

- **Unit:** prepare validation (missing/ambiguous → correct blocker class); capability branch (api→enqueue, portal/manual→manual step+blocker, unsupported→blocker); the submit idempotency guard (existing supplier_reference → skip); preflight present/absent (+ require_preflight→manual); status mapping (raw preserved); each failure class → its blocker code.
- **DB-bound integration:** api supplier → submit → `supplier_reference` + step + unit transition; missing data → blocker, no submit; portal supplier → manual step + `supplier_manual_submission_required`, no submit; the status poller reconciles status/tracking + records the reconcile step; **idempotent re-submit = no duplicate supplier order**; cross-tenant RLS on the dropship/units path; flag-off = no enqueue, no handler/poller.

## Explicitly deferred (separate work)

- Per-vendor supplier adapters beyond btp (+ their credentials/registration).
- Multi-document order **split** execution (v1 raises `supplier_partial_fulfillment` for operator review).
- Purchase-order / backorder **submit** to suppliers (the symmetric `StepCreatePurchaseOrder`/`StepReceivePurchaseOrder` flow) — analogous; a separate slice if needed.
- An operator order-submission UI beyond the existing supplier portal / ops dashboard.
- A canonical supplier-status mapping table per provider (v1 maps in the adapter / a small in-code map; the OPE-407 status-mapping workbench is the platform-side home for a richer version).
