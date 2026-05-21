# Provider Integration Studio — Production Readiness Specification

- **Data:** 2026-05-17
- **Status:** docelowa specyfikacja produkcyjna do review
- **Zakres:** Provider Integration Studio, provider registry, validation engine, platform-admin access, publication gates, evidence, operations, migration, certification suite.
- **Powiązane dokumenty:** `../plans/2026-05-21-ope-420-provider-integration-studio-implementation-plan.md`, `2026-05-21-ope-424-canonical-logistics-state-adr.md`, `2026-05-21-ope-425-orchestration-data-lifecycle.md`, `2026-05-17-provider-integration-studio-design.md`, `2026-05-17-provider-integration-studio-gap-analysis.md`, `2026-05-17-provider-integration-studio-ui-ux-design.md`, `2026-05-17-supplier-integration-research.md`, `../templates/provider-integration-builder.md`, `../templates/supplier-discovery-pack.md`, `2026-05-17-fulfillment-orchestration-design.md`

## Cel

Ten dokument definiuje, co musi zawierać produkcyjnie gotowy opis i późniejsza implementacja Provider Integration Studio. To nie jest uproszczona lista startowa. To docelowy kontrakt jakości, bezpieczeństwa, operacyjności i utrzymania dla systemu, który decyduje, które integracje OpenOMS może pokazać klientom i na jakich warunkach.

Provider Integration Studio ma stać się wewnętrznym systemem OpenOMS do:

- definiowania providerów,
- wersjonowania providerów,
- projektowania credential/settings schema,
- klasyfikacji capabilities,
- mapowania statusów,
- uruchamiania walidacji,
- przechowywania bezpiecznego evidence,
- publikowania integracji,
- prywatnego włączania integracji dla tenantów,
- migracji i deprecacji istniejących integracji,
- utrzymywania runbooków i odpowiedzialności operacyjnej.

## Research Basis

Specyfikacja opiera się na obecnym stanie OpenOMS i na zewnętrznych wzorcach produkcyjnych:

- Google SRE Production Readiness Review traktuje produkcyjną gotowość jako proces analizy, checklist specyficznych dla usługi, identyfikacji braków oraz planu usprawnień.
- AWS Well-Architected wskazuje sześć filarów: operational excellence, security, reliability, performance efficiency, cost optimization i sustainability.
- Azure Well-Architected opisuje workload jako resilient, available, recoverable, secure, operationally supportable i oceniany przez iteracyjne checklisty.
- OWASP ASVS daje mierzalny standard weryfikacji technicznych kontroli bezpieczeństwa aplikacji.
- Supabase/Postgres best practices dla OpenOMS oznaczają: świadomy model RLS, indeksy pod access patterns, immutable history tam gdzie wymaga tego audyt, brak Redis jako durable truth.
- Dojrzałe platformy integracyjne, np. TD SYNNEX Digital Bridge, pokazują istotne wzorce: connector catalog, sandbox/production keys, sample payloads, audit logs, review przed publikacją i utrzymanie connectorów, gdy partner API się zmienia.
- Oficjalne dokumentacje providerów potwierdzają, że pierwsze testy muszą objąć różne klasy integracji: marketplace API, carrier labels/tracking, full dropshipping API, hybrid feed/API, SOAP B2B, webhooks, EDI/business documents i self-hosted shop APIs.

Źródła:

- Google SRE: `https://sre.google/sre-book/evolving-sre-engagement-model/`
- AWS Well-Architected: `https://docs.aws.amazon.com/wellarchitected/latest/framework/the-pillars-of-the-framework.html`
- Azure Well-Architected: `https://learn.microsoft.com/en-us/azure/well-architected/what-is-well-architected-framework`
- OWASP ASVS: `https://owasp.org/www-project-application-security-verification-standard/`
- InPost Developers: `https://developers.inpost-group.com/`
- Allegro Developer Portal: `https://developer.allegro.pl/documentation`
- BigBuy API: `https://www.bigbuy.eu/en/api_bigbuy.html`
- Matterhorn API: `https://help.matterhorn-wholesale.com/CMS2/article/api/`
- MALFINI SOAP B2B: `https://shop.malfini.com/file/pdf/pdf/b2b/B2B_soap_EN.pdf`
- GS1 EANCOM: `https://support.gs1.org/support/solutions/articles/43000734250-what-kind-of-messages-does-gs1-eancom-provide-`
- Shopify Webhooks/Admin API: `https://shopify.dev/docs/api/admin-rest/latest/resources/webhook`
- WooCommerce REST API: `https://woocommerce.github.io/woocommerce-rest-api-docs/`
- TD SYNNEX Digital Bridge: `https://www.tdsynnex.com/na/us/digital-bridge/`

## Production Readiness Pillars

Provider Integration Studio must satisfy these pillars before it can become part of the OpenOMS production operating model.

| Pillar | Production requirement |
| --- | --- |
| Governance | Every provider has owner, lifecycle, version, publication state, review history, and deprecation path |
| Security | Platform-admin access is separate from tenant RBAC; secrets and evidence are encrypted/redacted; SSRF and destructive probes are controlled |
| Reliability | Validation probes are deterministic, retry/backoff is explicit, provider outages degrade safely, publication is gated |
| Operability | Runbooks, audit, metrics, validation runs, gaps, dashboards, incident response and support ownership exist |
| Data Integrity | Provider definitions, versions, schemas, capabilities, mappings and validation runs are durable and immutable where required |
| Extensibility | New provider classes can be added through schema/capability/probe contracts without one-off UI/backend exceptions |
| Customer Safety | Customers see only published or explicitly enabled providers; unsupported capabilities never appear as supported |
| Migration Safety | Existing providers move into registry without breaking tenant integrations or losing auditability |

## Required Specification Sections

The production-ready specification for Provider Integration Studio must contain all sections below. A later implementation plan should map every section to code, tests, docs, and rollout tasks.

### 1. Product Scope And Non-Customer Boundary

Must define:

- Studio is internal OpenOMS platform-admin tooling.
- Tenant/customer users do not access `/platform/providers`.
- Customer-visible setup reads only published provider versions.
- Private beta visibility is explicit per tenant.
- AI-assisted research is external and redacted until privacy boundaries are mature.

Acceptance criteria:

- No normal tenant role can access Studio routes or APIs.
- Customer setup cannot reference draft/internal provider versions.

### 2. Platform-Admin Identity And Authorization

Must define:

- separate `platform_admins` model,
- permissions: `providers:read`, `providers:write`, `providers:validate`, `providers:publish`, `providers:secrets`,
- 2FA/session requirements,
- access revocation,
- audit of every sensitive action,
- distinction between OpenOMS platform admin and tenant owner/admin.

Acceptance criteria:

- Backend middleware enforces platform-admin claims.
- Frontend route isolation is not trusted as the only control.
- Sensitive actions require explicit permission, not generic admin.

### 3. Data Model Contract

Must define platform tables:

- `platform_admins`,
- `provider_definitions`,
- `provider_versions`,
- `provider_field_schemas`,
- `provider_capability_profiles`,
- `provider_status_mappings`,
- `provider_validation_probes`,
- `provider_validation_runs`,
- `provider_validation_results`,
- `provider_integration_gaps`,
- `provider_publication_events`,
- `provider_tenant_enables`,
- `provider_test_credentials`.

Must define tenant links:

- `integrations.provider_version_id`,
- tenant-specific capability profile,
- tenant-specific validation runs,
- migration compatibility with existing `provider` string.

Acceptance criteria:

- Platform tables are not tenant-scoped.
- Tenant tables remain protected by RLS.
- Published provider versions are immutable except controlled publication transitions.
- Validation runs and publication events are append-only.

### 4. Provider Lifecycle Contract

Lifecycle states:

```text
research -> designed -> adapter_in_progress -> internal_validation -> private_beta -> available -> deprecated -> retired
```

Must define:

- allowed transitions,
- required permissions,
- blocking gaps per transition,
- immutable fields after publication,
- deprecation and retirement rules,
- rollback to previous version,
- emergency disable.

Acceptance criteria:

- State transitions are audited.
- `available` cannot be reached with blocking validation gaps.
- `retired` provider cannot be used for new tenant setup.

### 5. Credential And Settings Schema Contract

Must define:

- field types,
- secret vs non-secret storage,
- field validation,
- environment modes,
- dynamic field visibility,
- help text and customer labels,
- mapping from schema to backend validation,
- generated tenant setup forms.

Acceptance criteria:

- Tenant UI fields are generated from published schema.
- Adapter-required fields cannot be missing from schema.
- Secret fields never return raw values through API.
- Internal test credentials are stored separately from tenant credentials.

### 6. Capability Profile Contract

Must define:

- support values: `supported`, `partially_supported`, `manual_supported`, `not_supported`, `unknown`,
- channel values: REST, SOAP, EDI, cXML, webhook, XML/IOF/CSV/XLSX feed, portal, email, manual,
- mode values: synchronous, polling, webhook, scheduled, batch, manual, hybrid,
- freshness requirements,
- required inputs,
- provided outputs,
- evidence references,
- tenant-specific downgrades.

Acceptance criteria:

- No capability can be customer-visible as supported without evidence.
- Tenant account probes can downgrade provider defaults.
- Manual support is explicit, not hidden as an error.

### 7. Status Mapping Contract

Must define:

- raw status storage,
- canonical status per domain,
- status level: order, line, shipment, package, invoice, return,
- confidence,
- terminal flag,
- automation blocking flag,
- unknown mapping gap creation.

Acceptance criteria:

- Unknown status never silently maps to success.
- Terminal statuses require explicit high-confidence mapping.
- Shipment status does not automatically overwrite commercial order status.

### 8. Validation Engine Contract

Must define:

- probe types,
- sync vs async execution,
- worker behavior,
- retries and backoff,
- destructive-probe controls,
- sandbox vs production behavior,
- validation result schema,
- blocking vs warning gaps,
- validation expiration and revalidation.

Required probe classes:

- `auth.check`,
- `schema.validate`,
- `catalog.sample`,
- `catalog.full.dry_run`,
- `stock.sample`,
- `stock.write.sample`,
- `stock.increase.sample`,
- `stock.decrease.sample`,
- `stock.acknowledgement.sample`,
- `price.sample`,
- `order.preflight.sample`,
- `order.create.test`,
- `order.status.sample`,
- `shipment.tracking.sample`,
- `invoice.sample`,
- `webhook.signature.test`,
- `malformed.payload.test`,
- `rate_limit.behavior`,
- `evidence.redaction.test`.

Acceptance criteria:

- Probe outcomes are deterministic and immutable.
- Long probes run through workers.
- Production order-creating probes require explicit platform-admin confirmation and idempotency.

### 9. Evidence, Audit, And Retention Contract

Must define:

- what evidence is stored,
- what is hashed only,
- field-level redaction rules,
- retention by evidence type,
- viewer permissions,
- linkage to provider version, validation run, tenant integration and order where relevant.

Acceptance criteria:

- Raw payload storage requires provider-specific redaction policy.
- Safe hashes exist even when raw payloads are discarded.
- Evidence explains provider decisions without exposing secrets or unnecessary PII.

### 10. API Contract

Must define:

- endpoint list under `/v1/platform/providers`,
- request/response payloads,
- error model,
- pagination/filtering,
- idempotency,
- optimistic concurrency/version handling,
- permission requirements,
- audit behavior,
- OpenAPI updates.

Acceptance criteria:

- Every state-changing endpoint creates audit entry.
- API errors distinguish validation, auth, permission, conflict, provider business error, transport error and system error.
- API contract is documented before implementation.

### 11. UI/UX Contract

Must define:

- Provider Registry,
- Provider Overview,
- Schema Builder,
- Capability Matrix,
- Status Mapping,
- Validation Runner,
- Evidence/Gaps,
- Publication Panel,
- Private Beta Tenant Enablement,
- Migration View.

Acceptance criteria:

- Internal UI never appears in customer navigation.
- UI shows blocking gaps before publication.
- Generated setup preview matches tenant-facing setup schema.
- Validation history and evidence are inspectable by platform admins.

### 12. Observability And SLO Contract

Must define:

- metrics,
- structured logs,
- dashboards,
- alerts,
- SLOs for validation worker and provider catalog availability,
- provider health degradation behavior.

Required metrics:

- `provider_validation_runs_total`,
- `provider_validation_probe_duration_seconds`,
- `provider_validation_probe_failures_total`,
- `provider_publication_state_changes_total`,
- `provider_integration_gaps_open_total`,
- `provider_catalog_requests_total`,
- `provider_catalog_errors_total`.

Acceptance criteria:

- Platform admin can see validation failure trends.
- Provider outage does not create uncontrolled retry storms.
- Critical gaps are visible without reading application logs.

### 13. Security Review Package

Must include:

- platform-admin threat model,
- SSRF review,
- secrets handling review,
- PII and partner-confidential data review,
- AI prompt boundary review,
- destructive probe review,
- tenant isolation review,
- ASVS-aligned application security checklist.

Acceptance criteria:

- Security review is completed before customer-visible publication.
- No evidence or AI prompt path can contain raw credentials.
- Feed/API fetches use safe clients and bounded reads.

### 14. Certification Test Suite

Must define standard tests for every provider class:

- valid credentials,
- invalid credentials,
- missing setup field,
- provider timeout,
- rate limit,
- malformed payload,
- unknown status,
- terminal status,
- stale stock/price,
- stock increase propagation,
- stock decrease propagation,
- stock write unsupported,
- stock write accepted without acknowledgement,
- manual stock override active,
- business rejection,
- duplicate order submit,
- partial order/shipment,
- missing tracking after SLA,
- webhook signature failure,
- credential rotation.

Acceptance criteria:

- Each provider version lists selected tests and test status.
- Test fixtures are redacted and versioned.
- Certification result controls publication state.

### 15. Migration And Backward Compatibility Contract

Must define:

- how existing tenant integrations receive `provider_version_id`,
- how static frontend readiness maps are replaced,
- fallback period,
- migration ledger,
- rollback if provider version mapping fails,
- old provider keys compatibility.

Acceptance criteria:

- Existing tenants keep working during migration.
- Static maps stop being source of truth after provider registry is active.
- At least one marketplace, one carrier and one supplier are migrated before broad rollout.

### 16. Operational Runbooks

Must include runbooks for:

- provider API response changes,
- auth failures,
- stale feed,
- unknown status,
- missing tracking,
- rate limit,
- provider business rejection,
- provider outage,
- endpoint deprecation,
- credential leak suspicion.

Acceptance criteria:

- Every runbook has detection, immediate system behavior, owner, customer impact, recovery, evidence and rollback/disable path.
- Runbooks are linked from provider version.

### 17. Rollout And Rollback Contract

Must define:

- feature flags,
- internal rollout,
- private beta rollout,
- tenant allowlist,
- emergency disable,
- provider version rollback,
- migration rollback,
- data retention on rollback.

Acceptance criteria:

- Studio can be deployed without changing customer-visible integrations.
- Provider publication can be reverted without deleting tenant data.
- Emergency disable prevents new actions while preserving audit/history.

## Production Readiness Gates

Provider Integration Studio is ready for production use only when all gates below are met.

| Gate | Requirement |
| --- | --- |
| G1 Platform security | Platform-admin auth, permissions, audit, 2FA/session policy and route isolation exist |
| G2 Data foundation | Provider registry, versions, schemas, capabilities, mappings, validation runs and gaps exist in Postgres |
| G3 Validation engine | Probes run, persist immutable results and produce blocking gaps |
| G4 Publication control | Customer visibility is controlled by backend publication state |
| G5 Evidence safety | Redaction, hashing, retention and viewer rules are enforced |
| G6 Migration safety | Existing providers can be linked to provider versions without breaking tenants |
| G7 Operational readiness | Runbooks, metrics, dashboards and alerts exist |
| G8 Certification | Initial representative providers pass class-specific certification |
| G9 Security review | Threat model and ASVS-aligned review completed |

## Capability-Class Proof Strategy

Provider names in this specification are test vectors, not architecture boundaries. The Studio must first prove reusable integration classes, then certify concrete providers against those classes.

Architecture rules:

- Provider-specific logic belongs inside adapters, probes, mappings and versioned provider definitions.
- Core registry, orchestration, fulfillment, validation, evidence, publication and dashboard models must operate on capability classes, canonical states and declared gaps.
- Passing one provider cannot certify a whole category. It certifies the baseline class contract plus the exact gaps and variances observed for that provider.
- Every class proof must produce a reusable checklist, schema pattern, status mapping pattern, evidence pattern, runbook pattern and tenant-visibility pattern.

Required class proofs before broad provider expansion:

| Capability class | Required proof |
| --- | --- |
| Marketplace platform | OAuth or app credential flow, order import, external status mapping, bidirectional stock propagation proof or declared unsupported capability, shipment/tracking push or declared unsupported capability, webhook/poller behavior and offer/listing boundary |
| Carrier network | Shipment creation, label retrieval, tracking events, pickup-point capability, cancellation capability, validation probes and destructive-action safety |
| Hybrid supplier feed/API | Catalog/feed ingestion, stock/price freshness, supplier availability converted to policy-adjusted `available_to_sell`, API inventory/order capability where available, explicit fallback when order/status/tracking API is missing |
| Full dropshipping supplier | Order creation, supplier acceptance/rejection, shipment/tracking retrieval, stock freshness, preflight/reservation capability where available, multi-warehouse or lead-time representation |
| Variable account/region supplier | Same provider can expose different capabilities by tenant, country, contract or account type without hard-coded assumptions |
| B2B SOAP/XML supplier | XML/SOAP transport, document IDs, delivery/payment option mapping, test mode, invoice/expedition status and evidence capture |
| Hosted shop platform | Versioned API, webhook setup, app token schema, order/product sync and controlled deprecation across API versions |
| Self-hosted shop platform | Tenant-controlled URL validation, SSRF protection, credential test, response-size limits and environment variability |
| Enterprise distributor connector | Sandbox/production credentials, connector review/governance, catalog/pricing/order/status/invoice capability groups and audit requirements |
| EDI/business-document profile | Partner profile versioning, ORDERS/ORDRSP/DESADV/INVOIC document lifecycle, acknowledgements, mapping gaps and evidence retention |

## Initial Integration Test Portfolio

The first provider portfolio must be class-first. The concrete provider is selected because it stresses a capability class that OpenOMS must support long term.

### Wave 1 — Required Class Representatives

| Capability class | Representative provider/profile | Why this representative is useful | Class-level proof output |
| --- | --- | --- | --- |
| Marketplace platform | Allegro | Existing broad marketplace surface: orders, offers, shipment/tracking-related flows, messages, returns, disputes, OAuth, sandbox/status concepts | Reusable marketplace class contract for order pull, auth, status mapping, event/polling model and publication gates |
| Carrier network | InPost | Existing important carrier; official docs cover OAuth2.1, shipment creation, label download, tracking and request IDs | Reusable carrier class contract for labels, tracking, pickup points, auth, label/evidence handling and destructive probe controls |
| Hybrid supplier feed/API | BTP.pro | Existing OpenOMS supplier integration; hybrid XML catalog plus API inventory/order | Reusable supplier capability split for feed/API hybrid, missing status/tracking gaps and supplier setup schema |
| Full dropshipping supplier | BigBuy | Public docs describe product data, prices, stock, order creation, tracking and shipping status | Reusable dropship class contract for preflight, order submission, tracking, stock freshness and supplier order evidence |
| Variable account/region supplier | Matterhorn | Public docs show API for product import/order/status, while other regional docs point to feed/manual models | Reusable variance model for per-account, region and version differences without provider-name branching |
| B2B SOAP/XML supplier | MALFINI | SOAP docs include test order mode, delivery/payment IDs, invoice options and item requirements | Reusable SOAP/XML class contract for B2B documents, test mode, invoice and expedition capabilities |
| Hosted shop platform | Shopify | API versioning and webhooks are central; official docs show webhook subscription and Admin API access token patterns | Reusable hosted-shop class contract for versioned APIs, webhook setup, shop event ingestion and app/custom-token credential schemas |
| Self-hosted shop platform | WooCommerce | REST API uses site URL plus consumer key/secret; real customer environments vary strongly | Reusable self-hosted class contract for URL validation, SSRF safety, credential schema and order/product API checks |
| Enterprise distributor connector | TD SYNNEX | Digital Bridge exposes catalog/connectors, sandbox/production keys, product/pricing/orders/status/invoices and connector review model | Reusable enterprise-distributor class contract for connector governance, sandbox keys, audit logs and catalog/PO/status/invoice capability groups |
| EDI business-document profile | GS1 EANCOM profile | Not a company, but needed to validate business-document integrations: ORDERS, ORDRSP, DESADV, INVOIC | Reusable EDI profile contract for order response, despatch advice, invoice, partner-specific mapping and evidence retention |

### Wave 1 Coverage Matrix

| Capability class | Initial representative | Required reusable artifact |
| --- | --- |
| Marketplace OAuth and order import | Allegro | Marketplace capability profile, status map template and publication gate checklist |
| Carrier label/tracking/pickup points | InPost | Carrier capability profile, label/tracking probe set and pickup-point setup schema |
| Existing supplier hybrid feed/API | BTP.pro | Hybrid supplier capability profile and missing-data/gap taxonomy |
| Full dropshipping API | BigBuy | Dropship order lifecycle, supplier evidence and rejection/blocker taxonomy |
| Account/region capability variance | Matterhorn | Tenant/account capability override model |
| SOAP/XML B2B | MALFINI | XML/SOAP transport, document evidence and test-mode contract |
| Webhooks and API versioning | Shopify | Hosted-shop API version lifecycle and webhook validation contract |
| Self-hosted REST variability | WooCommerce | Self-hosted URL safety and environment variability checklist |
| Enterprise connector governance | TD SYNNEX | Connector review, sandbox/production credential and audit contract |
| EDI business documents | GS1 EANCOM profile | EDI profile version, document lifecycle and acknowledgement contract |

### Wave 2 — After Studio Mechanics Are Proven

| Provider | Reason to defer after Wave 1 |
| --- | --- |
| Amazon SP-API | Adds async feeds, strict authorization and certification complexity; valuable after core publication/validation engine exists |
| eBay | OAuth and marketplace order/listing patterns overlap with Allegro but add international marketplace complexity |
| GLS/DPD/DHL | Important carrier expansion after InPost proves carrier schema/probe model |
| Fakturownia / wFirma / inFakt | Invoice-provider class after provider registry and evidence model exist |
| PrestaShop / Shoper | Store-platform expansion after Shopify/WooCommerce validate shop patterns |
| AB / ACTION / ALSO | Polish/EU B2B distributor patterns after EDI and enterprise connector foundations are stable |

## Implementation Planning Envelope

The later implementation plan should be split into independent epics:

1. Platform-admin auth and route isolation.
2. Provider registry and versioned schema data model.
3. Capability, status mapping, gaps and publication state.
4. Validation engine and worker execution.
5. Evidence, redaction and retention.
6. Internal Studio UI.
7. Tenant setup integration and generated provider forms.
8. Canonical logistics state model and capability-class proof gates.
9. Existing provider migration.
10. Representative provider certification after class proofs.
11. Observability, runbooks and security review.

No implementation should start until this production readiness specification is reviewed and accepted.

## Self-Review

- The document defines target production requirements, not an interim scope.
- It includes research-backed production readiness pillars.
- It lists the required specification sections and acceptance criteria.
- It includes security, data, API, UI, validation, evidence, operations, migration, rollout and certification contracts.
- It includes an initial provider portfolio chosen to prove integration classes rather than anchoring the architecture to current providers.
- It separates company providers from the GS1 EANCOM standard profile while still including EDI validation coverage.
- It avoids unsupported claims by tying provider rationale to official documentation or current OpenOMS code context.
