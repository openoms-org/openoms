# Provider Integration Studio — obecne środowisko vs wymagane zmiany

- **Data:** 2026-05-17
- **Status:** analiza porównawcza i lista wymaganych zmian
- **Zakres:** OpenOMS public repo: API server, dashboard, PostgreSQL/Supabase schema, workers, provider adapters, integration setup, admin/security.
- **Powiązane dokumenty:** `2026-05-17-provider-integration-studio-design.md`, `2026-05-17-provider-integration-studio-production-readiness.md`, `2026-05-17-provider-integration-studio-ui-ux-design.md`, `../templates/provider-integration-builder.md`, `2026-05-17-supplier-integration-research.md`, `2026-05-17-fulfillment-orchestration-design.md`

## Cel

Ten dokument porównuje obecny stan OpenOMS z docelowym środowiskiem wymaganym do produkcyjnego Provider Integration Studio. Nie opisuje wariantu skróconego. Celem jest docelowy model, w którym integracje są wersjonowane, walidowane, publikowane kontrolowanie i utrzymywane operacyjnie.

## Wniosek główny

OpenOMS ma już solidny fundament do integracji:

- tenant-scoped `integrations`,
- szyfrowane credentials,
- provider factories,
- marketplace/carrier/supplier/invoicing adapter interfaces,
- RLS na tenant tables,
- audit log,
- sync jobs,
- readiness flags na dashboardzie,
- provider metadata w frontendzie,
- worker infrastructure,
- SSRF-safe HTTP patterns w części integracji.

Brakuje jednak platformowej warstwy zarządzania providerami:

- brak provider registry niezależnego od tenant integrations,
- brak wersjonowanych provider definitions,
- brak centralnego credential/settings schema,
- brak walidatora capabilities,
- brak immutable validation runs,
- brak evidence model dla provider readiness,
- brak platform-admin auth niezależnego od tenant admin/owner,
- brak publication gates sterujących widocznością integracji,
- brak centralnych status mappings i mapping gaps,
- brak tenant-specific capability downgrades,
- brak migracji statycznych frontend maps do danych z backendu.

Obecny model odpowiada na pytanie: "czy tenant ma zapisaną integrację z providerem?". Docelowy model musi odpowiadać na pytanie: "czy OpenOMS wie, co ta wersja providera potrafi, jak to zweryfikowano, kto to utrzymuje, komu wolno to zobaczyć i co stanie się, gdy provider zacznie zwracać inne dane?".

## Obecny stan techniczny

### Tenant Integrations

Obecna tabela `integrations` jest tenant-scoped i trzyma:

- `tenant_id`,
- `provider`,
- `status`,
- `credentials`,
- `settings`,
- `last_sync_at`,
- `sync_cursor`,
- `error_message`,
- `label`.

W kodzie backendowym:

- `model.Integration` eksponuje `has_credentials`, ale nie zwraca sekretów.
- `IntegrationService` szyfruje credentials przy tworzeniu i aktualizacji.
- `IntegrationRepository` przechowuje zaszyfrowane credentials jako JSON string w kolumnie JSONB.
- `IntegrationHandler` obsługuje tenant CRUD przez `/v1/integrations`.
- Operacje są wykonywane przez `database.WithTenant`.

Ocena:

- dobre jako tenant configuration store,
- niewystarczające jako provider governance layer,
- nie zawiera wersji providera, schema fields, capabilities, validation evidence ani publication state.

### Provider Interfaces

Obecnie istnieją osobne interfejsy:

- `MarketplaceProvider`,
- `CarrierProvider`,
- `SupplierProvider`,
- `InvoicingProvider`.

Marketplace ma już optional interfaces dla wybranych możliwości, np. stock/price bulk update, async feed, listing activate/deactivate. Carrier ma status mapping i optional dispatch order creator. SupplierProvider jest najwęższy: `FetchProducts`, `FetchInventory`, `CreateOrder`.

Ocena:

- dobra baza dla adapterów,
- optional interface pattern jest właściwym kierunkiem,
- capabilities są jednak wykrywane z kodu, nie zapisane i zweryfikowane jako provider/account profile,
- supplier domain wymaga rozbudowy o preflight, status, shipments, tracking, invoices, returns i capability reporter.

### Provider Registration

Provider registration jest runtime/hard-coded:

- `RegisterMarketplaceProvider`,
- `RegisterCarrierProvider`,
- `RegisterSupplierProvider`,
- `RegisterInvoicingProvider`.

Frontend provider metadata jest statyczne w `provider-info.ts`, readiness w `readiness.ts`, a dojrzałość integracji w `integration-status.ts`.

Ocena:

- dobre dla kompilacji i adapter wiring,
- niewystarczające dla publikacji i widoczności w produkcie,
- brak provider version i migration path,
- frontend może pokazywać stan inny niż backend capabilities.

### Security And Admin

Obecnie admin/owner jest tenant-level:

- `AdminGuard` w dashboardzie sprawdza `user.role === "admin" || "owner"`.
- Backend ma RBAC i `RequirePermission`.
- Nie ma osobnej platform-admin identity dla OpenOMS operatorów.

Ocena:

- dobre dla tenant administration,
- nie może być użyte do Provider Integration Studio,
- Studio wymaga osobnej platformowej autoryzacji, bo będzie zarządzało globalnym katalogiem providerów i internal test credentials.

### Observability

Obecne mechanizmy:

- `audit_log` dla tenant operations,
- `sync_jobs` dla background sync,
- `webhook_events` dla incoming webhooks,
- `integration.error_message` i `last_sync_at`,
- dashboard operational health z integracji w stanie `error`.

Ocena:

- dobra baza operacyjna,
- brakuje provider-level validation runs,
- brakuje probe-level results,
- brakuje evidence retention policy,
- brakuje centralnych integration gaps,
- brakuje safe raw observation store powiązanego z provider version.

## Porównanie obszarów

| Obszar | Obecnie | Wymagane docelowo | Zmiana |
| --- | --- | --- | --- |
| Provider identity | `provider` jako string na tenant integration i hard-coded key w kodzie | `provider_definitions` + `provider_versions` | Dodać platform registry |
| Provider versioning | Brak | Wersje z changelog, publication state, schema/capability mappings | Nowe tabele i API |
| Customer integration | Tenant `integrations` | Tenant integration wskazuje provider version i tenant capability profile | Migracja schema |
| Credentials | Tenant credentials szyfrowane | Tenant credentials + osobne internal test credentials dla platform validation | Nowe platform secret storage |
| Settings schema | Ad hoc per form/handler | Versioned `provider_field_schemas` generujące UI i backend validation | Schema builder |
| Capabilities | Wynikają z adapter interface albo statycznej wiedzy | `provider_capability_profiles` z evidence, support, channel, mode, freshness | Capability service + DB |
| Status mapping | W kodzie providerów, carrier `MapStatus`, ad hoc marketplace mapping | Centralne `provider_status_mappings` z confidence i gaps | Mapping engine |
| Readiness | Static frontend readiness maps | Backend publication state + validation verdict + tenant-specific downgrades | API-driven readiness |
| Admin access | Tenant owner/admin | Separate platform-admin auth and permissions | `platform_admins`, middleware, UI isolation |
| Validation | Provider-specific tests/manual checks | Deterministic validation probes and immutable runs | Validation runner + workers |
| Evidence | audit/sync/webhook logs | Provider version evidence, safe payload hashes, redaction, retention | Evidence model |
| Gaps | error_message/logs | Typed provider integration gaps with owner/severity/blocking | Gap service |
| Publication | Static visibility/readiness | Research -> designed -> validation -> private beta -> available -> deprecated -> retired | Publication workflow |
| Tenant setup UI | Hand-written provider forms/metadata | Generated from published provider schema | UI generator |
| Supplier support | BTP plus feed/supplier model | Capability-class supplier model: feed-only, hybrid feed/API, dropship API, B2B document, manual supplier portal | Interface expansion |
| Manual workflows | Existing supplier portal/manual areas | Manual as explicit provider capability with SLA/evidence | Manual task integration |
| Migration | No registry migration path | Existing providers registered and linked to versions | Migration project |

## Required Changes

### Database / Supabase

Add platform-level tables:

- `platform_admins`
- `provider_definitions`
- `provider_versions`
- `provider_field_schemas`
- `provider_capability_profiles`
- `provider_status_mappings`
- `provider_validation_probes`
- `provider_validation_runs`
- `provider_validation_results`
- `provider_integration_gaps`
- `provider_publication_events`
- `provider_tenant_enables`
- `provider_test_credentials`

Modify tenant integration model:

- add nullable `provider_version_id` to `integrations`,
- add tenant-specific capability profile or link table,
- keep `provider` string during migration for compatibility,
- add migration/backfill from existing provider keys.

Security requirements:

- platform tables are not tenant-scoped and must not be readable through tenant RLS,
- tenant integrations stay tenant-scoped and RLS-protected,
- internal test credentials encrypted separately,
- validation evidence redacted by policy,
- destructive provider probes audited.

### Backend API

Add platform route group:

```text
/v1/platform/providers
```

Required services:

- `PlatformAdminService`
- `ProviderDefinitionService`
- `ProviderVersionService`
- `ProviderSchemaService`
- `ProviderCapabilityService`
- `ProviderStatusMappingService`
- `ProviderValidationService`
- `ProviderPublicationService`
- `ProviderGapService`

Required middleware:

- `RequirePlatformAdmin`
- platform permission checks:
  - `providers:read`,
  - `providers:write`,
  - `providers:validate`,
  - `providers:publish`,
  - `providers:secrets`.

Required API behavior:

- no tenant user can call platform endpoints,
- provider schema mutations create or update draft versions only,
- published versions are immutable except publication state transitions,
- validation runs are immutable,
- publication transitions are audited,
- provider setup fields are served from backend, not frontend static maps.

### Provider Adapter Layer

Keep current factory pattern for compiled adapter code, but add provider metadata binding:

- factory registration remains code-level,
- provider definition/version defines publication and schema,
- adapter can expose optional capability reporter,
- validation probes call adapter capabilities through typed interfaces,
- unsupported capabilities must return typed `not_supported`, not generic error.

Supplier interfaces to add:

- `SupplierOrderChecker`
- `SupplierOrderStatusReader`
- `SupplierOrderLineStatusReader`
- `SupplierShipmentReader`
- `SupplierTrackingReader`
- `SupplierInvoiceReader`
- `SupplierReturnCreator`
- `SupplierCapabilityReporter`

Marketplace/carrier/invoicing should also gain capability reporter patterns where useful.

### Validation Workers

Add provider validation worker support:

- short auth probes can run synchronously,
- feed parsing, order probes, status/tracking checks run asynchronously,
- probes use safe HTTP clients,
- probe results write immutable validation records,
- destructive probes require explicit confirmation and sandbox/test mode where available,
- provider-level rate limits and backoff are enforced.

Required probe classes:

- `auth.check`,
- `schema.validate`,
- `catalog.sample`,
- `catalog.full.dry_run`,
- `stock.sample`,
- `price.sample`,
- `order.preflight.sample`,
- `order.create.test`,
- `order.status.sample`,
- `shipment.tracking.sample`,
- `invoice.sample`,
- `webhook.signature.test`,
- `malformed.payload.test`,
- `rate_limit.behavior`.

### Dashboard

Add internal route:

```text
/platform/providers
```

Required UI areas:

- Provider Registry,
- Provider Overview,
- Schema Builder,
- Capability Matrix,
- Status Mapping,
- Validation Runner,
- Evidence/Gaps view,
- Publication Panel,
- Private Beta tenant enablement.

Required frontend architecture changes:

- add platform-admin auth store or claims handling,
- hide route from tenant navigation,
- replace static provider readiness over time with backend-driven provider catalog,
- generate tenant setup forms from `provider_field_schemas`,
- keep static maps only as migration fallback.

### Existing Provider Migration

Migration candidates must be handled as representatives of capability classes. The required migration output is a reusable class contract plus the provider-specific version data.

| Capability class | Initial representative | Current state | Required migration |
| --- | --- | --- | --- |
| Marketplace platform | Allegro | Broad API surface and dedicated endpoints | Marketplace class profile, provider definition, version, capabilities, status mappings, validation probes, publication state |
| Carrier network | InPost | Carrier labels/tracking/pickup points | Carrier class profile for labels, tracking, pickup points, geowidget settings, status mapping evidence |
| Hybrid supplier feed/API | BTP | Supplier hybrid XML/API | Supplier class profile, feed/API split, missing status/tracking gaps, schema fields |
| Carrier expansion variance | GLS/DPD/DHL | Carrier providers with varying maturity | Carrier class variants for label/tracking/rates support and error mapping |
| Invoice provider | Fakturownia | Invoicing provider | Invoice class capabilities, credential schema, PDF/evidence validation |
| Hosted/self-hosted shop | Shopify/WooCommerce/Shoper | Store providers, some blocked/beta in readiness | Shop class profiles with true publication state and missing capability gaps |

## Required Documentation Additions

Keep these documents maintained:

- Provider Integration Studio design,
- Provider Integration Builder template,
- this gap analysis,
- provider-specific discovery briefs,
- provider-specific validation reports,
- provider-specific runbooks,
- migration ledger for existing providers,
- platform-admin security model,
- evidence retention policy.

## Production Acceptance Criteria

Provider Integration Studio is production-ready only when:

- platform-admin auth is separate from tenant roles,
- provider registry and versioning exist,
- provider schemas generate setup fields,
- validation runs are immutable,
- publication state controls customer visibility,
- evidence redaction policy is enforced,
- tenant-specific capability downgrades exist,
- existing static provider readiness has a migration plan,
- at least three capability classes are proven through representative providers:
  - one marketplace platform,
  - one carrier network,
  - one supplier/dropship class,
- one provider can move through the full lifecycle:
  - research,
  - designed,
  - internal validation,
  - private beta,
  - available,
  - deprecated.

## Risks

| Risk | Impact | Required mitigation |
| --- | --- | --- |
| Platform admin scope mixed with tenant RBAC | Tenant admin could access global provider controls | Separate platform auth and backend checks |
| Static frontend maps remain source of truth | Provider visibility drifts from validation evidence | Backend-driven provider catalog and migration fallback removal |
| Validation probes too shallow | Provider appears ready but fails in real fulfillment | Domain-specific probes and E2E validation |
| Evidence stores sensitive payloads | PII/secret exposure | Redaction policy, hashes, access controls |
| Existing providers not migrated | Two truths in system | Migration ledger and provider version backfill |
| Tenant account differs from provider default | Customer sees unsupported capability | Tenant-specific probes and capability downgrade |
| Provider changes API silently | Production drift | scheduled validation probes, runbooks, provider version updates |

## Self-Review

- This document compares actual code-backed capabilities with required Provider Integration Studio capabilities.
- It proposes only the target production model.
- It treats Supabase/PostgreSQL as durable source of truth.
- It separates tenant integrations from platform provider governance.
- It lists concrete backend, frontend, database, worker, migration, security, and documentation changes.
- It identifies current assets that should be reused instead of rebuilt.
