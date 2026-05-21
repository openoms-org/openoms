# OpenOMS Provider Integration Builder

**Purpose:** reusable document and prompt kit for researching, designing, validating, and preparing a new supplier, marketplace, carrier, shop, invoice, or automation provider integration.

**Audience:** OpenOMS maintainers and platform administrators. This is not customer-facing documentation.

**Core rule:** never implement or publish an integration based only on the provider name. First verify the exact capabilities of the concrete account, region, API version, contract, and environment.

---

## When To Use

Use this builder whenever OpenOMS needs to add or materially change:

- supplier or wholesaler integration,
- marketplace integration,
- carrier integration,
- shop/e-commerce platform integration,
- invoice/accounting integration,
- EDI, cXML, XML, SOAP, REST, webhook, feed, or manual provider workflow,
- provider-specific status mapping,
- provider credential/settings form,
- tenant-facing integration setup flow.

Do not use this for small fixes inside an already verified provider unless the fix changes capabilities, status semantics, credentials, authorization, or published setup fields.

---

## Output Artifacts

Each integration pass must produce these artifacts before implementation starts:

1. **Provider Discovery Brief**
   - factual research,
   - links to source documentation,
   - account/region/API version assumptions,
   - capability matrix,
   - known gaps.

   For suppliers, wholesalers, dropship partners, B2B distributors and feed providers, start from `supplier-discovery-pack.md` before writing this brief.

2. **Provider Capability Draft**
   - `supported`, `partially_supported`, `manual_supported`, `not_supported`, `unknown`,
   - freshness expectations,
   - required input fields,
   - provided output fields,
   - evidence sources.

3. **Credential And Settings Schema**
   - fields shown to tenant or platform admin,
   - secret vs non-secret fields,
   - validation rules,
   - environment selection,
   - test connection behavior.

4. **Status Mapping Draft**
   - raw provider statuses,
   - canonical OpenOMS statuses,
   - confidence,
   - blocking unknown statuses,
   - sample payload references.

5. **Validation Plan**
   - auth check,
   - catalog/feed probe,
   - stock/price probe,
   - order dry run or sandbox create,
   - status/tracking probe,
   - invoice/returns probe where supported,
   - negative/error cases.

6. **Implementation Plan**
   - adapter boundaries,
   - database changes,
   - backend services,
   - dashboard setup fields,
   - tests,
   - rollout and rollback.

7. **Production Operations Brief**
   - owner,
   - publication decision,
   - evidence retention,
   - incident runbooks,
   - deprecation/migration plan,
   - security notes.

---

## Production Decision Matrix

Use this matrix before writing code.

| Decision | Criteria | Required before proceeding |
| --- | --- | --- |
| `core_adapter` | Provider is reusable across many tenants and supports a stable technical contract | Source docs, test credentials, capability matrix, status/error mappings, validation probes, operational owner |
| `certified_custom_adapter` | Provider matters to a tenant or segment but is not broad enough for public catalog | Partner/customer contract, tenant-specific capability profile, private-beta publication path |
| `feed_managed_provider` | Provider supports catalog/price/stock but not reliable order/status automation | Parser spec, freshness policy, deletion semantics, manual order fallback |
| `manual_assisted_provider` | Provider works through portal/email/manual action and still has business value | Manual task workflow, SLA, evidence capture, dashboard blockers |
| `external_automation_connector` | Workflow belongs in controlled external automation but OpenOMS remains state owner | Signed callback, idempotency, event contract, monitoring and rollback |
| `blocked_provider` | Critical capabilities are untestable, legally restricted, unsafe, or unknown | Blocking gaps with responsible owner and required remediation |

Never choose based on implementation convenience. Choose based on long-term operability and truthful customer expectations.

---

## Research Prompt

Use this prompt when asking an AI agent to research a provider.

```text
You are researching a new OpenOMS provider integration.

Provider:
- Name:
- Region/country:
- Provider type: supplier / marketplace / carrier / shop / invoice / EDI / other
- Intended OpenOMS use case:
- Known customer/account context:

Rules:
- Use official provider documentation first.
- If public documentation is incomplete, say exactly what requires partner/account verification.
- Do not infer capabilities from the provider name.
- Distinguish channel, capability, freshness, mapping, and evidence.
- Treat manual/portal/email flows as valid capabilities, not as implementation failures.
- Preserve raw external statuses and fields.
- Mark every critical unknown as a readiness blocker.
- Do not invent endpoint names, status values, credentials, or sample payloads.

Find and summarize:
1. Authentication and environments
   - auth type,
   - sandbox/test availability,
   - production approval process,
   - token expiration,
   - rate limits,
   - required contracts or account flags.

2. Product/catalog data
   - channel: REST, SOAP, XML, IOF, CSV, XLSX, EDI, cXML, portal, email,
   - product identifiers,
   - variant identifiers,
   - EAN/SKU handling,
   - images/descriptions/attributes/categories,
   - deletion/deactivation semantics,
   - full vs delta sync.

3. Price and stock
   - exact quantity, availability bucket, boolean availability, ETA, warehouse split,
   - net/gross/VAT/currency,
   - account-specific prices,
   - freshness/update cadence,
   - reservation support,
   - stale-data behavior.

4. Order creation
   - automatic API/EDI/SOAP create,
   - manual portal/email/file route,
   - preflight/check support,
   - idempotency/external reference,
   - required address/customer fields,
   - delivery point/carrier/service fields,
   - payment requirements,
   - partial/backorder behavior.

5. Order status, shipment, tracking
   - status endpoint or webhook,
   - raw status list,
   - status level: order, line, shipment, package,
   - tracking number/url/carrier,
   - multi-shipment support,
   - SLA when no tracking appears.

6. Invoices, returns, and documents
   - invoice list/detail/PDF,
   - credit notes,
   - returns/RMA,
   - attachments/documents.

7. Errors and operational behavior
   - structured business errors,
   - retryable transport errors,
   - auth failures,
   - throttling,
   - maintenance windows,
   - payload size limits.

Deliver:
- provider summary,
- source links,
- capability matrix,
- missing information,
- suggested OpenOMS adapter shape,
- validation probes,
- risks and rollout notes.
```

---

## Capability Matrix Template

Use one row per capability. Keep unknowns explicit.

| Capability | Support | Channel | Mode | Freshness | Required input | Provided output | Evidence | Readiness impact |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `supplier.catalog.read` | `unknown` | `feed_xml` | `scheduled` | `provider_defined` | feed URL, credentials | products, variants | feed snapshot | blocks catalog import until verified |
| `supplier.price.read` | `unknown` | `rest_api` | `polling` | `provider_defined` | product ID, account | net/gross price, currency | API response | blocks automatic margin rules until verified |
| `supplier.availability.read` | `unknown` | `rest_api` | `polling` | `provider_defined` | product ID or SKU | quantity or availability | API response | blocks automatic order routing until verified |
| `supplier.order.preflight` | `unknown` | `rest_api` | `synchronous` | per request | order draft | accepted/rejected/errors | API response | required for guarded auto-submit when available |
| `supplier.order.create` | `unknown` | `rest_api` | `synchronous` | per request | canonical supplier order | external order ID | API response | blocks automatic supplier order creation until verified |
| `supplier.order.status.read` | `unknown` | `rest_api` | `polling` | provider SLA | external order ID | raw status | API response | blocks transparent post-submit state until verified |
| `supplier.shipment.notice.read` | `unknown` | `rest_api` | `polling` | provider SLA | external order ID | shipment, carrier, tracking | API response | blocks automatic marketplace tracking update until verified |
| `supplier.invoice.read` | `unknown` | `rest_api` | `polling` | provider SLA | external order ID | invoice number, PDF URL | API response | affects accounting reconciliation |
| `supplier.return.create` | `unknown` | `portal` | `manual` | operator SLA | return request | RMA/reference | manual confirmation | creates manual task if unsupported |

Allowed `Support` values:

- `supported`
- `partially_supported`
- `manual_supported`
- `not_supported`
- `unknown`

Allowed `Channel` values:

- `rest_api`
- `soap_api`
- `edi`
- `cxml`
- `webhook`
- `feed_xml`
- `feed_iof`
- `feed_csv`
- `feed_xlsx`
- `portal`
- `email`
- `manual`

Allowed `Mode` values:

- `synchronous`
- `polling`
- `webhook`
- `scheduled`
- `batch`
- `manual`
- `hybrid`

---

## Credential And Settings Schema Template

Use this schema to design fields before creating frontend forms or database configuration.

```json
{
  "provider": "provider_key",
  "version": "2026-05-17",
  "environment_modes": ["sandbox", "production"],
  "credential_fields": [
    {
      "key": "api_key",
      "label": "API key",
      "type": "secret",
      "required": true,
      "validation": {
        "min_length": 16,
        "max_length": 256
      },
      "stored_in": "integrations.credentials"
    }
  ],
  "settings_fields": [
    {
      "key": "base_url",
      "label": "Base URL",
      "type": "url",
      "required": true,
      "allowed_hosts": ["api.provider.example"],
      "stored_in": "integrations.settings"
    },
    {
      "key": "sync_interval_minutes",
      "label": "Sync interval",
      "type": "integer",
      "required": true,
      "min": 5,
      "max": 1440,
      "default": 60,
      "stored_in": "supplier.sync_interval_minutes"
    }
  ],
  "test_connection": {
    "probe": "auth.check",
    "timeout_seconds": 15,
    "success_requires": ["authenticated", "account_id_present"]
  }
}
```

Field rules:

- Secret fields must be stored only in encrypted credentials.
- URLs must use SSRF-safe HTTP clients and host allowlists where possible.
- Environment selection must be explicit.
- Customer-facing forms must never expose platform-only test credentials.
- Optional fields must state what capability they enable.
- Fields that affect order submission must be part of validation readiness.

---

## Status Mapping Template

| Raw status | Raw level | Canonical status | Confidence | Terminal | Blocks automation | Notes |
| --- | --- | --- | --- | --- | --- | --- |
| `new` | order | `submitted` | high | false | false | External order was accepted by provider API but not yet processed |
| `processing` | order | `processing` | high | false | false | Provider is preparing order |
| `waiting_for_payment` | order | `awaiting_payment` | high | false | true | Requires manual or automated payment resolution |
| `partial` | order | `partially_accepted` | medium | false | true | Requires line-level review |
| `sent` | shipment | `shipped` | high | false | false | Requires tracking if provider exposes it |
| `unknown_provider_value` | order | `unknown` | none | false | true | Creates mapping gap and blocks automatic transition |

Mapping rules:

- Raw status is always stored.
- Unknown raw status must not silently fall back to a successful state.
- Shipment status must not automatically overwrite commercial order status.
- Line status must not be collapsed into order status without reconciliation.
- Terminal states require explicit mapping.
- Confidence below `high` should block fully automatic transitions unless policy allows.

---

## Validation Prompt

Use this prompt after research and before implementation or publication.

```text
You are validating readiness for an OpenOMS provider integration.

Inputs:
- Provider Discovery Brief
- Capability Matrix
- Credential And Settings Schema
- Status Mapping Draft
- Available adapter code or planned adapter interface
- Sample payloads, if available

Validate:
1. Critical capabilities are not marked supported without evidence.
2. Every automatic action has required input fields, error mapping, and rollback/retry behavior.
3. Every provider status either maps to canonical status or creates a mapping gap.
4. Freshness expectations exist for stock, price, order status, and tracking.
5. Credentials are separated into encrypted secrets and non-secret settings.
6. Test connection has a deterministic success/failure condition.
7. Unsupported capabilities have manual task fallback or are explicitly disabled.
8. Dashboard-facing problems are typed, not only logged.
9. Adapter implementation plan includes tests for success, provider business errors, transport errors, auth errors, malformed payloads, stale data, and unknown statuses.
10. Publication state is correct: draft, internal, beta, or available.

Return:
- readiness verdict: blocked / ready_for_implementation / ready_for_internal_publish / ready_for_customer_publish
- blocking gaps
- non-blocking risks
- required fields for setup UI
- required validation probes
- recommended first implementation issue.
```

---

## Production Operations Brief Template

Fill this for every provider that will be implemented or published.

```text
Provider:
Provider version:
Owner:
Operational owner:
Support owner:
Security reviewer:

Publication decision:
- core_adapter / certified_custom_adapter / feed_managed_provider / manual_assisted_provider / external_automation_connector / blocked_provider

Customer visibility:
- hidden / internal_validation / private_beta / available / deprecated / retired

Supported workflow:
- What customer workflow does this provider support end to end?
- Which steps are automatic?
- Which steps are manual but controlled by OpenOMS?
- Which steps are intentionally unsupported?

Evidence policy:
- Which raw observations are stored?
- Which payloads are only hashed?
- Which fields are redacted?
- How long are validation runs retained?
- Who can view evidence?

Incident runbooks:
- Auth failure:
- Provider outage:
- Feed stale:
- Unknown status:
- Missing tracking:
- Rate limit:
- Payload/schema change:
- Credential leak suspicion:

Deprecation policy:
- What provider signal starts deprecation?
- How are tenants notified?
- What replacement path exists?
- What data must be retained?

Security notes:
- SSRF risk:
- PII risk:
- Secret exposure risk:
- Partner-confidential document risk:
- Destructive probe risk:
```

---

## Evidence Policy Template

| Evidence type | Store | Redact/hash | Retention | Viewer |
| --- | --- | --- | --- | --- |
| Auth probe | auth result, safe account/scopes | tokens and secrets | 180 days | platform admin |
| API metadata | endpoint key, status code, request ID, duration | request/response body unless redacted | 180 days | platform admin/engineer |
| Feed snapshot | checksum, row count, parser version, duration | signed URL tokens and raw feed body | 180 days | platform admin/engineer |
| Status observation | raw status, canonical status, confidence | PII from raw payload | order retention policy | platform admin/support |
| Manual confirmation | actor, timestamp, reference | attachments unless classified | order retention policy | platform admin/support |

Rules:

- Raw payload storage requires a field-level redaction policy.
- Safe hashes should be stored even when raw payloads are discarded.
- Provider evidence must link to provider version and validation run.
- Runtime tenant evidence must link to tenant integration and provider version.
- AI prompts must use redacted summaries only.

---

## Required Test Scenario Library

Every provider implementation plan should select all scenarios that apply.

| Scenario | Expected result |
| --- | --- |
| Valid credentials | Test connection passes and records account/environment evidence |
| Invalid credentials | Test connection fails with typed auth error and no retry storm |
| Missing required setup field | Schema validation blocks save |
| Provider timeout | Retry/backoff behavior is typed and observable |
| Rate limit | Provider-level backoff and rate-limit finding are recorded |
| Malformed payload | Parser fails closed and stores safe evidence |
| Unknown status | Creates mapping gap and blocks automatic transition |
| Terminal status | Maps only through explicit high-confidence mapping |
| Stale stock/price | Dependent automation is blocked or downgraded |
| Business rejection | Creates actionable blocker, not generic system error |
| Duplicate order submit | Idempotency prevents duplicate side effects or marks behavior unsupported |
| Partial order/shipment | Preserves line/package-level truth |
| Missing tracking after SLA | Creates operational blocker |
| Webhook signature failure | Event rejected and security evidence recorded |
| Credential rotation | Existing evidence remains, new credentials encrypted, probes re-run |

---

## Publication States

Provider definitions should move through explicit states:

| State | Meaning | Visible to customers |
| --- | --- | --- |
| `research` | Facts are being gathered | No |
| `designed` | Capability and field model exists | No |
| `adapter_in_progress` | Code is being built | No |
| `internal_validation` | OpenOMS admin can run probes | No |
| `private_beta` | Enabled for selected tenant/integration | Selected only |
| `available` | Available in customer integration setup | Yes |
| `deprecated` | Existing tenants may continue, new setup hidden | Existing only |
| `retired` | Disabled for new and existing use after migration | No |

Publication gates:

- `research` to `designed`: source links and capability matrix complete.
- `designed` to `adapter_in_progress`: implementation plan approved.
- `adapter_in_progress` to `internal_validation`: tests compile and adapter registers.
- `internal_validation` to `private_beta`: validation probes pass on sandbox or approved test account.
- `private_beta` to `available`: real tenant workflow succeeds and rollback path is documented.
- `available` to `deprecated`: replacement or provider shutdown documented.
- `deprecated` to `retired`: tenant migration completed or integration disabled by policy.

---

## Definition Of Ready For Implementation

A provider is ready for implementation only when:

- provider identity, region, and API/version are explicit,
- credential and settings fields are known,
- at least one source link or partner spec exists,
- critical capabilities are classified,
- unsupported/manual capabilities have a planned workflow,
- raw statuses are collected or the lack of status list is documented,
- freshness rules exist for stock/price/order status/tracking,
- adapter boundaries are defined,
- tests and validation probes are defined,
- security constraints are documented.

## Definition Of Ready For Customer Publish

A provider is ready for customer-visible publication only when:

- platform admin validation passes,
- test connection is deterministic,
- setup form fields match the credential/settings schema,
- all customer-facing secrets are encrypted,
- provider errors are mapped to actionable messages,
- unknown status handling creates mapping gaps,
- dashboard problem signals exist,
- audit/evidence is stored,
- at least one end-to-end flow has been verified for the intended use case,
- rollback or disable path exists.

---

## Implementation Handoff Prompt

Use this prompt when handing a verified provider to an implementation agent.

```text
Implement the OpenOMS provider integration described by the attached Provider Discovery Brief, Capability Matrix, Credential And Settings Schema, Status Mapping Draft, and Validation Plan.

Rules:
- Follow existing OpenOMS provider patterns.
- Keep provider-specific transport in the adapter.
- Store durable state in PostgreSQL/Supabase, not Redis.
- Use encrypted credentials for secrets.
- Do not bypass tenant isolation.
- Do not silently map unknown statuses.
- Preserve raw provider observations.
- Add tests before implementation for supported capabilities.
- Add negative tests for malformed payloads, auth failure, provider business errors, transport errors, stale data, and unknown statuses.
- Add dashboard/setup fields only from the approved schema.
- Do not expose internal admin-only tools to tenant users.

Deliver:
- code changes,
- migrations if needed,
- tests,
- docs/context updates,
- validation commands and results,
- remaining capability gaps.
```
