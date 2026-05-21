# OPE-424 Canonical Logistics State Model ADR

- **Date:** 2026-05-21
- **Status:** Proposed for review
- **Scope:** Provider-agnostic logistics, fulfillment, inventory availability, external status mapping, blockers, and evidence model for OpenOMS.
- **Related issues:** `OPE-403`, `OPE-420`, `OPE-414`, `OPE-415`, `OPE-416`, `OPE-417`, `OPE-418`, `OPE-419`, `OPE-428`
- **Related documents:** `2026-05-17-fulfillment-orchestration-design.md`, `2026-05-17-fulfillment-orchestration.md`, `2026-05-17-provider-integration-studio-design.md`, `2026-05-17-provider-integration-studio-production-readiness.md`, `2026-05-17-supplier-integration-research.md`

## Decision

OpenOMS will use a provider-agnostic canonical logistics state model.

Core orchestration code must reason through:

- fulfillment process state,
- fulfillment unit state,
- fulfillment step state,
- inventory availability state,
- external observations,
- versioned status mappings,
- capability profiles,
- typed blockers,
- evidence,
- tenant policy.

Core orchestration code must not branch on provider names such as Allegro, InPost, BTP, BigBuy, Matterhorn, Shopify, WooCommerce, MALFINI, TD SYNNEX or any future provider. Provider-specific logic is allowed only in:

- adapter construction,
- provider registry definitions,
- provider validation probes,
- versioned status mappings,
- capability profiles,
- tenant-specific configuration.

## Context

OpenOMS already contains orders, shipments, labels, tracking, warehouses, stock, stock sync, dropship orders, purchase orders, supplier sync, marketplace pollers and automation rules. The problem is not lack of features. The problem is scattered process truth.

Examples:

- marketplace pollers can create orders through paths different from manual/API order creation,
- shipment and label status changes can bypass a durable process timeline,
- stock reservation is currently best-effort and asynchronous,
- supplier feeds can contain stock but not order confirmation,
- provider statuses differ by account, region, API version and contract,
- manual supplier work is a valid business path but is not modeled as first-class orchestration.

The canonical model must make these differences explicit.

## Principles

1. **Commercial order state is not fulfillment process state.**
   The order status visible to customers or marketplaces remains small and business-facing. Physical execution is tracked separately.

2. **Every external fact enters as an observation.**
   Raw statuses, tracking events, supplier feed rows and webhook payloads are stored as sanitized observations/evidence before canonical state changes.

3. **Mappings are explicit and versioned.**
   Unknown external statuses create mapping gaps. OpenOMS does not guess.

4. **Capabilities are data, not assumptions.**
   The system can say `supported`, `requires_manual`, `degraded`, `unsupported`, or `unknown` for every capability.

5. **Manual is a first-class channel.**
   Portal, email, phone and operator entry are modeled as manual capabilities, not as implementation failures.

6. **Stock availability is canonical and bidirectional.**
   Sales decrease available stock and trigger channel propagation. Receipts, returns, corrections and trusted supplier availability can increase available stock and trigger channel propagation.

7. **Redis, n8n and external workflow tools do not own durable business truth.**
   They can help with delivery, automation and notifications, but PostgreSQL remains the source of truth.

## Canonical Process State

`fulfillment_process.aggregate_status` represents progress visible to operators.

Allowed values:

| Status | Meaning |
| --- | --- |
| `new` | Process created but not validated. |
| `validating` | Required data, mappings, capabilities and policy are being checked. |
| `ready` | Process can start execution. |
| `in_progress` | One or more units/steps are actively executing. |
| `waiting_external` | Process waits for provider, supplier, carrier, customer, payment or operator response. |
| `blocked` | Process cannot advance until a blocker is resolved. |
| `completed` | All required units completed or were explicitly skipped/cancelled with audit. |
| `cancelled` | Process was cancelled and no further automatic action should run. |

`fulfillment_process.health_status` is independent from progress.

Allowed values:

| Health | Meaning |
| --- | --- |
| `ok` | No operator action required. |
| `warning` | Process can continue but has reduced transparency or degraded automation. |
| `action_required` | Operator, tenant, supplier or support action is needed. |
| `system_error` | OpenOMS expected an automated capability to work but it failed unexpectedly. |

## Canonical Unit State

`fulfillment_units` represent fulfillable slices of an order.

Unit types:

| Unit type | Meaning |
| --- | --- |
| `warehouse` | Items fulfilled from owned stock. |
| `dropship` | Items fulfilled by supplier directly to customer or customer channel. |
| `backorder` | Items waiting for purchase/receipt before warehouse fulfillment can continue. |
| `mixed_child` | Child unit created after split allocation. |
| `manual` | Execution depends on operator/manual/portal/email path. |

Unit statuses:

| Status | Meaning |
| --- | --- |
| `pending` | Unit exists but cannot execute yet. |
| `ready` | Unit can begin its next step. |
| `running` | A step is currently executing. |
| `waiting_external` | Waiting for external confirmation or event. |
| `blocked` | Unit cannot continue until blocker is resolved. |
| `succeeded` | Unit completed successfully. |
| `failed` | Unit failed and needs recovery or cancellation. |
| `cancelled` | Unit was cancelled with audit. |
| `skipped` | Unit was intentionally skipped with reason. |

## Canonical Step State

Step keys are stable and provider-agnostic.

Core step keys:

- `validate_order`
- `select_fulfillment_policy`
- `allocate_units`
- `reserve_stock`
- `release_stock`
- `recalculate_availability`
- `propagate_stock_to_channels`
- `create_purchase_order`
- `receive_purchase_order`
- `create_dropship_order`
- `preflight_supplier_order`
- `confirm_supplier_order`
- `pick_items`
- `pack_items`
- `select_carrier_service`
- `create_shipment`
- `generate_label`
- `create_dispatch_order`
- `sync_tracking_to_marketplace`
- `notify_customer`
- `await_tracking`
- `close_process`

Step statuses:

- `pending`
- `ready`
- `running`
- `waiting_external`
- `blocked`
- `succeeded`
- `failed`
- `skipped`
- `cancelled`

## Inventory Availability Model

Inventory availability is a canonical logistics concern.

Definitions:

| Term | Meaning |
| --- | --- |
| `owned_quantity` | Quantity physically or legally owned by the tenant. |
| `reserved_quantity` | Quantity allocated to open fulfillment processes. |
| `available_owned` | Owned quantity minus reserved quantity and safety buffers. |
| `source_quantity` | Raw external supplier/provider availability. |
| `available_to_sell` | Policy-adjusted sellable quantity after freshness, safety buffer, lead time, reservation/preflight support and manual overrides. |
| `channel_quantity` | Quantity last pushed or observed on a marketplace/shop listing. |
| `manual_override_quantity` | Operator-controlled quantity that automation must not overwrite unless policy allows it. |

Availability sources:

- warehouse stock,
- stocktake/correction,
- warehouse document confirmation,
- return-to-stock,
- purchase order receipt,
- supplier availability feed/API,
- marketplace/shop observed stock,
- operator manual override.

Default policy:

- automatic stock propagation is enabled by default for normal client operation;
- manual, paused and override modes are supported per tenant, product, listing, channel and supplier source;
- an active manual override wins over automatic propagation unless tenant policy explicitly allows automation to replace it;
- stale supplier availability cannot increase `available_to_sell` unless tenant policy defines allowed staleness and safety buffer;
- stock decreases caused by real orders should propagate aggressively to reduce overselling risk;
- stock increases should propagate only after source trust and policy checks pass.

## Stock Propagation Events

Canonical event types:

| Event | Meaning |
| --- | --- |
| `availability.recalculated` | OpenOMS recalculated canonical availability. |
| `stock.reserved` | Stock was reserved for a process/unit. |
| `stock.released` | Reservation was released. |
| `stock.received` | Warehouse or purchase receipt increased owned stock. |
| `stock.returned` | Return was accepted back into sellable stock. |
| `stock.corrected` | Operator or stocktake changed stock. |
| `supplier.availability.observed` | External supplier availability was observed. |
| `supplier.availability.stale` | Supplier availability exceeded freshness policy. |
| `channel.stock.propagation_requested` | OpenOMS enqueued stock update for a channel. |
| `channel.stock.propagation_succeeded` | Channel accepted or acknowledged stock update. |
| `channel.stock.propagation_failed` | Channel update failed. |
| `channel.stock.stale` | Channel stock is older than accepted SLA. |
| `manual_override.created` | Operator set manual quantity or paused automation. |
| `manual_override.expired` | Override expired or was removed. |

## Blocker Codes

Required blocker additions for this ADR:

| Code | Category | Meaning |
| --- | --- | --- |
| `stock_sync_failed` | integration | Channel stock update failed unexpectedly. |
| `channel_stock_stale` | integration | Channel stock is older than policy permits. |
| `supplier_availability_stale` | supplier | Supplier availability cannot be trusted. |
| `supplier_availability_unknown` | supplier | Supplier does not provide enough availability evidence. |
| `supplier_preflight_required` | supplier | Supplier requires preflight before auto-submit or stock exposure. |
| `manual_stock_review_required` | operator | Automation paused or override requires review. |
| `stock_write_unsupported` | capability | Provider cannot accept stock writes. |
| `stock_ack_missing` | capability | Provider accepts stock update but does not provide acknowledgement. |
| `external_status_unmapped` | mapping | Raw external status is not mapped. |
| `integration_capability_missing` | capability | Required capability is absent. |
| `integration_capability_degraded` | capability | Capability exists but health/freshness is degraded. |

## External Status Mapping

External statuses map into canonical event types, not directly into arbitrary order statuses.

Mapping precedence:

1. system default mapping,
2. provider version mapping,
3. tenant override mapping,
4. integration override mapping.

Status levels:

- commercial order,
- fulfillment process,
- fulfillment unit,
- fulfillment step,
- shipment,
- package,
- supplier order,
- supplier line,
- carrier tracking.

Unknown status handling:

- store sanitized observation,
- create `external_status_unmapped` warning or blocker,
- do not advance process if the next action depends on the status meaning,
- allow operator/platform-admin to add mapping with evidence and confidence.

## Capability Model

Every provider capability uses the following dimensions:

- `entity_type`
- `operation`
- `direction`
- `channel`
- `freshness`
- `authority`
- `support_status`
- `required_fields`
- `provided_fields`
- `latency_sla_seconds`

Inventory capability examples:

| Capability | Direction | Required behavior |
| --- | --- | --- |
| `inventory.availability.read` | inbound | Read external availability with freshness evidence. |
| `inventory.stock.write` | outbound | Push stock to channel/listing. |
| `inventory.stock.increase.write` | outbound | Provider accepts increasing stock. |
| `inventory.stock.decrease.write` | outbound | Provider accepts decreasing stock. |
| `inventory.stock.acknowledge` | inbound | Provider confirms stock update result. |
| `supplier.availability.preflight` | inbound | Supplier can validate availability before order submission. |
| `supplier.availability.reserve` | outbound | Supplier can reserve stock before final order. |

## Acceptance Criteria

- The ADR is accepted before orchestration schema implementation starts.
- Every planned fulfillment table can map back to one of the states/events in this ADR.
- Provider Integration Studio capability profiles can express every state dependency required by fulfillment.
- Stock decrease and stock increase paths are both represented.
- Manual/paused/override stock modes are first-class states.
- Supplier availability is modeled as external availability, not owned stock.
- Unknown provider statuses and capabilities produce explicit gaps/blockers.
- No acceptance criterion requires real supplier agreements; real provider proof remains blocked until supplier material exists.

## Consequences

Positive:

- OpenOMS can explain why an order, shipment, stock sync or dropship route is blocked.
- Providers can be added without changing core orchestration assumptions.
- Manual workflows become visible and auditable.
- Overselling risk is reduced by making stock propagation durable and observable.

Costs:

- More data model surface area.
- More explicit mappings and capabilities to maintain.
- Higher planning burden before provider publication.
- Need for lifecycle and retention rules before evidence volume grows.

## Review Checklist

- [ ] Does this state model cover manual, warehouse, dropship, backorder and mixed fulfillment?
- [ ] Does it cover both stock decrease and stock increase propagation?
- [ ] Does it prevent stale supplier availability from increasing stock by accident?
- [ ] Does it leave room for tenant-specific manual overrides?
- [ ] Does it avoid provider-name branching in core logic?
- [ ] Does it define blockers that operators can understand?
- [ ] Does it avoid making Redis, n8n or external tools durable truth?
