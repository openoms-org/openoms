# OPE-431 Supplier Discovery Pack And Access Checklist

- **Date:** 2026-05-21
- **Status:** Template for supplier discovery
- **Scope:** Reusable checklist for wholesalers, dropship suppliers, B2B distributors, feed providers, EDI partners and manual/portal suppliers.
- **Related issues:** `OPE-403`, `OPE-428`, `OPE-430`, `OPE-431`, `OPE-432`, `OPE-433`
- **Related documents:** `../specs/2026-05-17-supplier-integration-research.md`, `../specs/2026-05-21-ope-424-canonical-logistics-state-adr.md`, `provider-integration-builder.md`

## Purpose

Use this pack before OpenOMS implements or certifies any supplier, wholesaler or dropship integration.

The goal is to collect enough real partner material to decide:

- which capability class the supplier belongs to,
- whether OpenOMS can safely automate catalog, price, stock, order, status and tracking,
- what must stay manual,
- what blocks implementation,
- what evidence is required for Provider Integration Studio,
- whether the integration can ever become customer-visible.

Public provider documentation is not enough for production readiness. The concrete account, country, contract, API version, environment and support channel decide what is actually available.

## Discovery Summary

Fill one pack per concrete supplier account/environment.

| Field | Value |
| --- | --- |
| Supplier name |  |
| Legal entity / brand |  |
| Country / region |  |
| Account owner in OpenOMS |  |
| Customer/tenant requesting this supplier |  |
| Supplier contact person |  |
| Supplier support channel |  |
| Contract status | not_started / requested / received / signed / blocked |
| NDA required | yes / no / unknown |
| API access status | none / requested / sandbox / production / blocked |
| Test data status | none / sample files / sandbox data / real account limited |
| Integration class candidate | feed_only / hybrid_feed_api / full_dropship_api / soap_xml_b2b / edi_document / manual_portal / unknown |
| First OpenOMS use case | catalog / stock / dropship / purchase order / B2B document / other |

## Required Documents And Access

Collect links or attachments for:

- contract or partnership terms,
- API documentation,
- feed documentation,
- EDI/XML/SOAP/cXML specification,
- sample catalog file,
- sample price file,
- sample stock/availability file,
- sample order request,
- sample order response/acknowledgement,
- sample order status payload,
- sample shipment/tracking payload,
- sample invoice/credit note where relevant,
- rate limits and fair-use policy,
- sandbox setup guide,
- production approval process,
- support escalation path,
- data processing/security terms.

For each artifact record:

| Artifact | Source | Date received | Environment | Contains PII | Contains secrets | Storage location | Notes |
| --- | --- | --- | --- | --- | --- | --- | --- |
| API docs |  |  |  |  |  |  |  |
| Catalog sample |  |  |  |  |  |  |  |
| Stock sample |  |  |  |  |  |  |  |
| Order sample |  |  |  |  |  |  |  |
| Tracking sample |  |  |  |  |  |  |  |

Do not paste secrets into this document.

## Authentication And Environments

| Question | Answer |
| --- | --- |
| Auth type | API key / Basic / OAuth2 / JWT / SOAP credentials / EDI certificate / SFTP / portal / email |
| Sandbox available | yes / no / unknown |
| Production approval needed | yes / no / unknown |
| Test mode available in production API | yes / no / unknown |
| Token expiration / rotation |  |
| IP allowlist required | yes / no / unknown |
| Callback/webhook URL required | yes / no / unknown |
| Rate limits |  |
| Burst limits |  |
| Support for idempotency key | yes / no / unknown |
| Required account flags/modules |  |

Blocking findings:

- [ ] Credentials cannot be issued.
- [ ] Sandbox is unavailable and destructive production tests are not approved.
- [ ] API docs are incomplete for intended use case.
- [ ] Contract forbids automation or data storage required by OpenOMS.
- [ ] Rate limits cannot support expected tenant volume.

## Catalog Capability

| Question | Answer |
| --- | --- |
| Channel | REST / SOAP / XML / IOF / CSV / XLSX / EDI / portal / email |
| Full catalog available | yes / no / unknown |
| Delta catalog available | yes / no / unknown |
| Product identifiers | SKU / EAN / supplier product ID / variant ID / other |
| Variant model |  |
| Images available | yes / no / unknown |
| Descriptions available | yes / no / unknown |
| Attributes available | yes / no / unknown |
| Categories available | yes / no / unknown |
| Deletion/deactivation semantics |  |
| Minimum sync cadence |  |
| Maximum allowed sync cadence |  |

Evidence required:

- [ ] Full catalog sample.
- [ ] Variant sample.
- [ ] Deleted/deactivated product example.
- [ ] Product with multiple images.
- [ ] Product with missing EAN/SKU case.

## Price Capability

| Question | Answer |
| --- | --- |
| Account-specific prices | yes / no / unknown |
| Net/gross | net / gross / both / unknown |
| Currency |  |
| VAT included | yes / no / unknown |
| Promotions/discounts | yes / no / unknown |
| Price freshness | realtime / near_realtime / scheduled / manual / unknown |
| Minimum price update interval |  |
| Price by quantity tier | yes / no / unknown |
| Price by warehouse/region | yes / no / unknown |

Blocking findings:

- [ ] Price cannot be read for this account.
- [ ] Currency/VAT semantics are unclear.
- [ ] Price freshness cannot satisfy tenant policy.
- [ ] Feed/API can expose prices but contract forbids automated resale use.

## Availability And Stock Capability

This section is critical. Supplier availability is not owned warehouse stock.

| Question | Answer |
| --- | --- |
| Availability channel | REST / webhook / feed / portal / manual / unknown |
| Exact quantity | yes / no / unknown |
| Availability bucket | yes / no / unknown |
| Boolean availability only | yes / no / unknown |
| ETA only | yes / no / unknown |
| Stock by warehouse | yes / no / unknown |
| Lead time min/max | yes / no / unknown |
| Next delivery date | yes / no / unknown |
| Reservation supported | yes / no / unknown |
| Preflight/check before order | yes / no / unknown |
| Availability freshness | realtime / near_realtime / scheduled / manual / unknown |
| Max stale time allowed by supplier |  |
| Supplier removes unavailable products from feed | yes / no / unknown |
| Removed product means | out_of_stock / discontinued / temporarily_hidden / unknown |

Required normalized fields:

- `source_quantity`
- `available_to_sell`
- `availability_type`
- `warehouse_external_id`
- `min_handling_days`
- `max_handling_days`
- `next_delivery_date`
- `freshness_observed_at`
- `max_stale_at`
- `reservation_supported`
- `last_successful_sync_id`

Blocking findings:

- [ ] No reliable availability signal.
- [ ] Availability freshness is undefined.
- [ ] No safety buffer can be derived.
- [ ] Supplier can sell out before OpenOMS submits order and no preflight exists.
- [ ] Availability increase cannot safely propagate to sales channels.
- [ ] Availability decrease cannot be detected fast enough to prevent overselling.

## Order Creation Capability

| Question | Answer |
| --- | --- |
| Automatic order create | yes / no / unknown |
| Manual portal order | yes / no / unknown |
| Email/file order | yes / no / unknown |
| Order preflight | yes / no / unknown |
| Idempotency/external reference | yes / no / unknown |
| Required customer/address fields |  |
| Delivery point support | yes / no / unknown |
| Carrier/service selected by | OpenOMS / supplier / customer / unknown |
| Payment requirement | prepaid / credit / wallet / invoice / unknown |
| Partial acceptance | yes / no / unknown |
| Backorder support | yes / no / unknown |
| Cancel order | yes / no / unknown |
| Edit order | yes / no / unknown |

Evidence required:

- [ ] Successful sandbox/test order.
- [ ] Rejected order example.
- [ ] Partial acceptance example.
- [ ] Duplicate order submission behavior.
- [ ] Missing field error example.
- [ ] Rate-limited order create example if documented.

## Supplier Order Status Capability

| Question | Answer |
| --- | --- |
| Status read channel | REST / webhook / feed / portal / email / EDI / unknown |
| Status level | order / line / shipment / package / unknown |
| Raw status list provided | yes / no / unknown |
| Terminal statuses documented | yes / no / unknown |
| Rejection reasons structured | yes / no / unknown |
| Status freshness | realtime / near_realtime / scheduled / manual / unknown |
| Partial shipment status | yes / no / unknown |
| Backorder/waiting stock status | yes / no / unknown |

Canonical mapping draft:

| Raw supplier status | Raw label | Canonical status/event | Level | Terminal | Confidence | Requires review |
| --- | --- | --- | --- | --- | --- | --- |
|  |  |  |  |  |  |  |

Minimum canonical supplier statuses:

- `draft`
- `submitted`
- `accepted`
- `awaiting_payment`
- `waiting_for_stock`
- `partially_accepted`
- `processing`
- `packed`
- `ready_for_pickup`
- `partially_shipped`
- `shipped`
- `delivered`
- `cancelled`
- `rejected`
- `returned`
- `unknown`

Blocking findings:

- [ ] Supplier has no status visibility and tenant policy requires automatic status.
- [ ] Raw status list is unknown.
- [ ] Rejection is not distinguishable from technical failure.
- [ ] Partial/backorder states cannot be represented.

## Shipment And Tracking Capability

| Question | Answer |
| --- | --- |
| Supplier chooses carrier | yes / no / unknown |
| OpenOMS can choose carrier | yes / no / unknown |
| Tracking number provided | yes / no / unknown |
| Tracking URL provided | yes / no / unknown |
| Carrier code provided | yes / no / unknown |
| Multiple packages | yes / no / unknown |
| Tracking channel | REST / webhook / portal / email / EDI / unknown |
| Tracking appears after | order create / acceptance / packed / shipped / unknown |
| Missing tracking SLA |  |

Blocking findings:

- [ ] No tracking capability but marketplace/customer policy requires tracking.
- [ ] Tracking does not include carrier identifier.
- [ ] Multi-package behavior is unknown.
- [ ] Supplier can ship with unsupported carrier.

## Invoice, Credit Note And Return Capability

| Question | Answer |
| --- | --- |
| Invoice list/detail | yes / no / unknown |
| Invoice PDF | yes / no / unknown |
| Credit note | yes / no / unknown |
| Returns/RMA create | yes / no / unknown |
| Return status | yes / no / unknown |
| Documents channel | REST / SOAP / EDI / portal / email / unknown |
| Retention/contract limits |  |

## Error Model

Collect examples for:

- invalid credentials,
- rate limit,
- missing field,
- invalid SKU/EAN,
- unavailable product,
- price changed,
- stock changed,
- address rejected,
- delivery method rejected,
- payment required,
- duplicate external reference,
- order rejected,
- partial acceptance,
- timeout,
- provider maintenance.

Normalize errors into:

- `business_rejection`,
- `retryable_provider_error`,
- `rate_limited`,
- `credentials_invalid`,
- `mapping_missing`,
- `capability_unsupported`,
- `manual_action_required`,
- `unknown_provider_error`.

## Capability Matrix

Use `supported`, `partially_supported`, `manual_supported`, `not_supported`, `unknown`.

| Capability | State | Channel | Freshness | Evidence | Blocks automation if missing |
| --- | --- | --- | --- | --- | --- |
| `supplier.catalog.read` |  |  |  |  | yes |
| `supplier.catalog.delta.read` |  |  |  |  | no |
| `supplier.price.read` |  |  |  |  | yes |
| `supplier.availability.read` |  |  |  |  | yes |
| `supplier.availability.exact_quantity` |  |  |  |  | policy-dependent |
| `supplier.availability.by_warehouse` |  |  |  |  | policy-dependent |
| `supplier.availability.preflight` |  |  |  |  | policy-dependent |
| `supplier.availability.reserve` |  |  |  |  | policy-dependent |
| `supplier.order.create` |  |  |  |  | yes for auto-submit |
| `supplier.order.cancel` |  |  |  |  | no |
| `supplier.order.status.read` |  |  |  |  | yes for full visibility |
| `supplier.order.line_status.read` |  |  |  |  | yes for partial/backorder |
| `supplier.shipment.notice.read` |  |  |  |  | yes for tracking automation |
| `supplier.tracking.read` |  |  |  |  | yes for marketplace/customer tracking |
| `supplier.invoice.read` |  |  |  |  | no |
| `supplier.return.create` |  |  |  |  | no |
| `supplier.error.structured` |  |  |  |  | no |

## Initial Readiness Verdict

Choose one:

| Verdict | Meaning |
| --- | --- |
| `ready_for_design` | Enough material exists to design provider profile and adapter boundaries. |
| `needs_contract` | Business/legal access is missing. |
| `needs_api_access` | Contract exists but credentials/test account are missing. |
| `needs_sample_payloads` | Docs exist but real examples are missing. |
| `manual_only` | Useful supplier path exists, but automation is not available. |
| `blocked` | Critical unknowns make implementation unsafe. |

Decision:

- Verdict:
- Reason:
- Blocking gaps:
- Next owner action:
- Target issue:

## Handoff To OPE-433

After this pack is complete, OPE-433 should convert real materials into a supplier process/status sample matrix.

Required handoff:

- capability matrix,
- raw status list,
- normalized status mapping draft,
- sample payload index,
- freshness and stock semantics,
- order lifecycle diagram,
- known manual steps,
- blockers and assumptions.

## Security Notes

- Do not store API keys, passwords, tokens or certificates in this document.
- Do not commit supplier confidential PDFs if license/contract forbids it.
- Store sensitive partner documents in approved private storage and link by reference.
- Redact customer PII from samples before attaching to tickets/docs.
- If a sample contains real personal data, mark it as sensitive and do not put it in public repo docs.

## Review Checklist

- [ ] Contract/access status is explicit.
- [ ] Public docs are separated from account-specific evidence.
- [ ] Catalog, price, availability, order, status, tracking and invoice capabilities are each classified.
- [ ] Stock availability has freshness and safety policy notes.
- [ ] Dropship preflight/reservation support is known or marked as blocker.
- [ ] Manual path is modeled if automation is missing.
- [ ] Raw statuses are captured for OPE-433.
- [ ] Blocking gaps are recorded before implementation starts.
