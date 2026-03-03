# Architectural Decisions

Append-only log. Each entry is immutable once written.

---

## ADR-001: PostgreSQL RLS for Multi-Tenant Isolation
- **Date:** 2025 (initial architecture)
- **Context:** Need tenant isolation without separate databases per tenant.
- **Decision:** Row-Level Security with `set_config('app.current_tenant_id')` per transaction.
- **Consequences:** Every query must go through `WithTenant()`. SECURITY DEFINER functions needed for auth bypass. Workers need superuser pool for cross-tenant operations.

## ADR-002: Ed25519 JWT for Authentication
- **Date:** 2025 (initial architecture)
- **Context:** Need stateless auth tokens. RSA is slow, HMAC requires shared secret.
- **Decision:** Ed25519 signed JWT. 1h access token in Authorization header, 30d refresh in httpOnly cookie.
- **Consequences:** Key rotation requires invalidating all tokens. Can't use HS256 libraries.

## ADR-003: AES-256-GCM for Integration Credentials
- **Date:** 2025 (initial architecture)
- **Context:** Integration API keys/secrets stored in DB. Must be encrypted at rest.
- **Decision:** AES-256-GCM encryption. 64-char hex ENCRYPTION_KEY env var (32 bytes).
- **Consequences:** Key loss = credential loss. Key rotation requires re-encrypting all credentials.

## ADR-004: Supabase Migration (from self-hosted PostgreSQL)
- **Date:** 2026-02-19
- **Context:** Self-hosted PostgreSQL on Hetzner volume. No HA, manual backups, operational burden.
- **Decision:** Migrate to Supabase Pro (managed PostgreSQL).
- **Consequences:**
  - PgBouncer in transaction mode requires `simple_protocol` in DATABASE_URL
  - `simple_protocol` sends []byte as bytea hex → breaks JSONB columns
  - Required: AfterConnect type registration for json.RawMessage → JSONB
  - Migration URL uses direct connection (session mode, port 5432) — IPv6 only, needs session pooler
  - Cost: ~$25/month for Pro plan

## ADR-005: json.RawMessage JSONB Type Registration
- **Date:** 2026-02-20
- **Context:** pgx v5 `simple_protocol` sends `json.RawMessage` (which is `[]byte`) as bytea hex. PostgreSQL cannot cast bytea hex to JSONB → "invalid input syntax for type json".
- **Decision:** Register `json.RawMessage` as JSONB type in `database.go` AfterConnect callback with explicit `json.Marshal`/`json.Unmarshal` functions.
- **Alternatives rejected:**
  - `string()` cast in every repository (21+ locations, error-prone for new code)
  - Abandon Supabase (too costly to revert)
  - Use `pgtype.JSONB` wrapper type in models (invasive refactor)
- **Consequences:** Global fix — all `json.RawMessage` automatically sent as JSONB OID. Must provide `Marshal`/`Unmarshal` functions (nil causes panic on Scan).

## ADR-006: Subscription Pricing Model
- **Date:** 2026-02-20
- **Context:** Need revenue model for SaaS.
- **Decision:** Standard / Plus / Pro tiers based on order volume.
- **Details:** TBD (pricing, limits, feature gates not yet defined)
- **Consequences:** Need billing integration (Stripe), feature flag system, tenant plan enforcement.

## ADR-007: CSRF Double-Submit Cookie
- **Date:** 2026-02-24
- **Context:** Cross-subdomain setup (api.openoms.org + app.openoms.org) needs CSRF protection. SameSite cookies alone insufficient for cross-origin mutations.
- **Decision:** Double-submit cookie pattern. Server sets `csrf_token` cookie (SameSite=Lax, Domain=.openoms.org, HttpOnly=false), client reads cookie and echoes value as `X-CSRF-Token` header on mutations.
- **Exempt paths:** login, register, refresh, ws-ticket, public/*, webhooks/*, health, metrics.
- **Consequences:** Frontend must read cookie and attach header. Cookie Domain is configurable (empty for dev/localhost).

## ADR-008: Composite Token Blacklist (Redis + In-Memory)
- **Date:** 2026-02-24
- **Context:** Token blacklist stored in Redis. Redis outage = fail-open (revoked tokens accepted).
- **Decision:** Composite blacklist: write to both Redis + in-memory, read with fallback (Redis first, in-memory if unavailable).
- **File:** `apps/api-server/internal/middleware/token_blacklist_composite.go`
- **Consequences:** Slightly more memory usage. Graceful degradation on Redis failure. No single point of failure for token revocation.

## ADR-009: createCrudHooks Factory Pattern
- **Date:** 2026-02-24
- **Context:** 17 React hook files with identical CRUD boilerplate (useList/useGet/useCreate/useUpdate/useDelete). ~600 lines of duplicated code.
- **Decision:** Generic `createCrudHooks<T, CreateReq, UpdateReq, ListParams>` factory in `hooks/create-crud-hooks.ts`. Custom hooks stay in individual files alongside the factory-generated CRUD hooks.
- **File:** `apps/dashboard/src/hooks/create-crud-hooks.ts`
- **Consequences:** ~600 lines removed. New resources auto-get CRUD hooks. ListParams requires `as unknown as` cast for interface compatibility with generic constraint.

## ADR-010: SafeGo Helper for Goroutine Panic Recovery
- **Date:** 2026-02-25
- **Context:** Uncaught panics in goroutines (workers, background tasks) crash the entire process silently. Go runtime terminates on unrecovered goroutine panics.
- **Decision:** `SafeGo(logger, fn)` wrapper in `internal/util/safego.go` — recovers panics, logs stack trace with slog, prevents process termination.
- **File:** `apps/api-server/internal/util/safego.go`
- **Consequences:** All background goroutines should use `SafeGo` instead of bare `go func()`. Panic is logged, not re-raised.

## ADR-011: Weight Propagation (Supplier → Product → Shipment)
- **Date:** 2026-02-25
- **Context:** Shipments need weight for carrier label generation. Manual weight entry is error-prone. Supplier feeds (IOF/CSV) include product weights.
- **Decision:** Three-step propagation: (1) supplier sync writes weight to products table, (2) shipment creation auto-sums product weights from order items, (3) manual weight override still takes precedence.
- **Files:** `supplier_service.go` (sync weight), `shipment_service.go` (calculateOrderWeight)
- **Consequences:** ShipmentService now depends on ProductRepo. Weight calculated only when not explicitly provided.

## ADR-012: Three-Phase Pattern for External API Calls
- **Date:** 2026-02-25
- **Context:** SendToKSeF and SyncPendingStatuses held DB connections during external HTTP calls (KSeF API, marketplace APIs). Under load, this exhausts pgx connection pool.
- **Decision:** Three-phase pattern: (1) short DB transaction to read/validate, (2) external API calls with no DB connection, (3) short DB transaction to persist results.
- **Files:** `ksef_service.go` (SendToKSeF, SyncPendingStatuses, RetryErroredInvoices)
- **Consequences:** More complex code (data passed between phases via local vars). Error handling split across phases. KSeF session cleanup via `defer Terminate`.

## ADR-013: KSeF Auto-Send with Retry and Exponential Backoff
- **Date:** 2026-02-25
- **Context:** KSeF (Polish national e-invoice system) mandatory April 2026. Manual sending is impractical for high-volume tenants.
- **Decision:** Auto-send on invoice creation (async via SafeGo). Failed sends marked "error" (retryable, max 3 retries with 5/15/45 min backoff). Intermediate "retrying" status prevents duplicate submissions. Terminal "rejected" for KSeF UPO rejections.
- **Status flow:** `not_sent` → `retrying` → `pending` → `accepted`/`rejected`; `error` (retryable) → max 3 → `rejected` (terminal)
- **Files:** `ksef_service.go`, `invoice_service.go` (SetKSeFService setter), `ksef_status_worker.go`
- **Consequences:** InvoiceService depends on KSeFService via setter injection (avoids circular imports). Worker runs retry after status sync per tenant.

## ADR-014: Allegro Offer Import with SKU Matching
- **Date:** 2026-02-25
- **Context:** Sellers have existing Allegro offers. Need to link them to OMS products without manual mapping.
- **Decision:** Import endpoint fetches all seller offers via `ListAll()` (auto-pagination, max 1000/page). SKU matching: if offer has EAN/GTIN, match to product.ean; else match offer title words to product.sku. Creates ProductListing records for linked products.
- **Files:** `allegro_import_service.go`, `allegro-go-sdk/offers.go` (ListAll)
- **Consequences:** Per-tenant mutex prevents concurrent imports. Unique index on `product_listings(external_id, integration_id)` prevents duplicates at DB level.

## ADR-015: Message Templates with Variable Substitution
- **Date:** 2026-02-25
- **Context:** Automation engine needs to send marketplace messages (e.g., Allegro buyer notifications). Templates must support dynamic content (order ID, buyer name, tracking number).
- **Decision:** `message_templates` table with `{{variable}}` placeholder syntax. `substituteVariables` replaces placeholders from event data map. Templates are per-tenant, channel-scoped (allegro, email, sms), admin-guarded writes.
- **Files:** `message_template_service.go`, `message_template_handler.go`, `automation/actions.go` (executeSendMarketplaceMessage)
- **Consequences:** Body max 50k chars validation. Frontend uses `enabled` field (not `is_active`). Template variables are not validated against schema — invalid placeholders pass through unchanged.

## ADR-016: Erli SDK Rebuild Against Official API Docs
- **Date:** 2026-02-28
- **Context:** API Audit revealed entire `packages/erli-go-sdk/` was built on wrong assumptions — none of the endpoints matched official docs at https://erli.pl/svc/shop-api/doc/. Base URL was incorrect (api.erli.pl → erli.pl), endpoints wrong (POST /offers → POST /products/{externalId}), status mapping (6→3), polling parameter name (cursor → after).
- **Decision:** Full SDK rebuild against official Erli docs: (1) base URL: `productionBaseURL = "https://erli.pl/svc/shop-api"`, sandbox via `WithBaseURL()` (no hardcoded URL), (2) offers → products/{externalId} endpoints, externalId in path (URL-escaped), (3) status map 6→3 (pending/purchased/cancelled only), (4) polling param "after" instead of "cursor", (5) handle 202 Accepted response (async product validation), (6) provider.go: SKU as externalId, Create(ctx, externalID, req).
- **Files:** `packages/erli-go-sdk/client.go` (base URL), `offers.go` (endpoints), `statusmap.go` (3 statuses), `orders.go` (polling), `provider.go` (integration), `*_test.go` (updated mocks)
- **Security audit findings:** 1 HIGH (provider.go:47-49 — sandbox flag silent fail-open to production), 4 MEDIUM (deferred to follow-up: scheme validation, global logger, nil RawData fallback, Go 1.25.0 patches)
- **Consequences:** All Erli integrations fixed. 202 handling requires client code to detect async creation. Sandbox tenants must set `ERLI_SANDBOX_URL` or use `WithBaseURL()`; without it, sandbox flag is now a hard error (pending security fix merge).

## ADR-017: DHL24 SOAP Integration (WebAPI2)
- **Date:** 2026-03-01
- **Context:** Initial DHL SDK used fictional REST API endpoints. Official DHL Parcel Poland provides WebAPI2 (SOAP). Audit verified against https://dhl24.com.pl/en/webapi2/doc.html.
- **Decision:** Full SOAP integration via `packages/dhl-go-sdk/`: (1) base URL `dhl24.com.pl/webapi2` (prod) + `dhl24-test.dpd.com.pl/webapi2` (DHL has no official sandbox, test via credentials), (2) Auth via `AuthData` struct in SOAP body (username + password), (3) SOAP methods: createShipments, getLabels, getTrackAndTraceInfo, deleteShipment (all verified in WSDL), (4) Service type "AH" (domestic standard) or "09"/"12"/"EK"/"PI" for speed/express variants, (5) no sandbox URL (hard error if sandbox flag requested without base_url override).
- **Files:** `packages/dhl-go-sdk/client.go` (SOAP transport, auth), `shipments.go` (SOAP methods), `models.go` (SOAP request/response types with XML tags), `statusmap.go` (8 DHL24 event types), `provider.go` (integration layer), `*_test.go` (SOAP XML mocks)
- **Audit findings:** 4 CRITICAL (test suite — mock server returns JSON but SDK expects XML; GLS assertion fields stale), 1 HIGH (frontend offers invalid DHL service types "dhl_parcel"/"dhl_courier" not valid in SOAP API), 1 MEDIUM (unbounded io.ReadAll in response parsing)
- **Consequences:** DHL tracking requires separate portal login (not available via API). Statuses from WSDL are event-based (PICKED_UP, IN_TRANSIT, etc.) not order-based. Tests require SOAP XML envelopes, not JSON. Frontend service_type dropdown must use SOAP codes or map UI names to codes.

## ADR-018: DPD Poland REST API Alignment
- **Date:** 2026-03-01
- **Context:** DPD SDK base URLs were hardcoded as `dpd.com.pl/api/v1` (wrong — no such endpoint). Audit verified against https://dpdservices.dpd.com.pl/api-docs.
- **Decision:** Full REST API rebuild via `packages/dpd-go-sdk/`: (1) base URL `dpdservices.dpd.com.pl` (prod) + `dpdservicesdemo.dpd.com.pl` (sandbox), (2) Auth via Basic Auth (username:password base64) + `x-dpd-fid: {{masterfid}}` header, (3) endpoints verified: `/public/shipment/v1/generatePackagesNumbers` (create), `/public/shipment/v1/generateSpedLabels` (labels), (4) tracking and cancel NOT available via API (documented in statusmap), (5) COD support via `paymentType: "COD"` in request, insurance via `insuranceValue` field.
- **Files:** `packages/dpd-go-sdk/client.go` (REST client, auth headers), `shipments.go` (endpoints), `models.go` (DPD REST request/response models), `statusmap.go` (DPD statuses), `provider.go` (integration layer), `*_test.go` (REST HTTP mocks)
- **Audit findings:** 1 HIGH (frontend DPDFields missing `<CODAndInsuranceFields>` component — users cannot set COD from UI despite backend support), 1 MEDIUM (unbounded io.ReadAll in response parsing)
- **Consequences:** Tracking and cancel require separate manual portal access (not OMS-integrated). Labels returned as PDF base64 in response, embedded in response (not URL). Two-step label flow: generatePackagesNumbers → generateSpedLabels. Frontend form must include COD checkbox and insurance amount field.

## ADR-019: GLS ShipIT REST API Verification
- **Date:** 2026-03-01
- **Context:** GLS SDK claimed Bearer token auth with centralized ShipIT API. Audit verified against https://shipit.gls-group.eu/webservices/3_4_19/doxygen/WS-REST-API/index.html. Base URL `api.gls-group.eu/public/v1` is per-contract (varies by region/customer).
- **Decision:** Confirmed GLS implementation via `packages/gls-go-sdk/` is compliant: (1) Auth: HTTP Basic Auth (username:password base64, NOT Bearer token — corrected from initial bearer assumption), (2) Content-Type `application/glsVersion1+json` (custom header required), (3) endpoints: POST `/shipments` (create, labels inline in response), POST `/shipments/parceldetails` (tracking via `TrackID` in body), POST `/shipments/cancel/{trackID}` (cancel), (4) products PARCEL/EXPRESS verified valid, (5) COD via `service_cash` (service object, not boolean), insurance via `service_addonliability`.
- **Files:** `packages/gls-go-sdk/client.go` (REST client, auth), `shipments.go` (endpoints, proper path escaping on cancel), `models.go` (GLS request/response types), `statusmap.go` (GLS event statuses), `provider.go` (integration layer), `*_test.go` (REST HTTP mocks)
- **Audit findings:** 3 CRITICAL (test suite — wrong field assertions: `Services` vs `Service`, `ServiceType` vs `Product`, stale GetLabel test expecting success), 1 MEDIUM (unbounded io.ReadAll)
- **Consequences:** Labels returned embedded in create response (not separate getLabel call). Tracking requires POST with TrackID in body (not GET). Status model response changed to include `CancelStatus` with enum. Tests require assertions matching actual API field names (singular Service, not plural Services).

## ADR-020: Carrier SDK Audit Pipeline (Multi-Carrier Verification Pattern)
- **Date:** 2026-03-01
- **Context:** Erli audit (ADR-016) uncovered wholesale SDK misconceptions. Needed systematic verification for DHL/DPD/GLS to prevent production failures.
- **Decision:** Establish carrier SDK audit checklist for future integrations: (1) **Faza 1 — Documentation verification:** official API docs search, base URL + auth method + endpoints + models + status list verification, (2) **Faza 2 — Integration audit:** shipment creation field mapping, label retrieval format, tracking response parsing, cancel semantics, error handling, (3) **Faza 3 — Frontend audit:** service type options, COD/insurance form fields, rate display, (4) **Faza 4 — Test quality:** mock servers must match actual API transport (JSON vs XML/SOAP), assertion fields match API response structure, end-to-end tests catch integration breaks.
- **Files:** Audit script/checklist (future: `scripts/carrier-audit.sh` or wiki page)
- **Findings summary:** 4 CRITICAL (test suite), 2 HIGH (frontend), 4 MEDIUM (best practices). Verdict: FAIL (CI blocks merge until tests fixed).
- **Consequences:** All future carrier integrations must pass this audit before merge. Test suite quality raised to production standard. Frontend field mapping must be verified against backend API contracts. No "fictional" endpoints/codes — docs-first approach mandatory.

## ADR-021: Onboarding Wizard State in JSONB (Tenant-Scoped Setup Flow)
- **Date:** 2026-03-01
- **Context:** New tenants land on empty dashboard post-registration with no data. Need guided setup (company details → warehouse → integration → team invite) before they can operate effectively.
- **Decision:** Store onboarding progress in tenants.settings JSONB under key `"onboarding"` with structure: `{completed, current_step, completed_steps[], skipped_steps[], completed_at}`. Three new endpoints (backend handler methods):
  1. `GET /v1/onboarding/status` — read current state (returns defaults if missing: current_step=1, empty slices)
  2. `PUT /v1/onboarding/step/{step}` — idempotent mark step as completed/skipped (step 1 non-skippable, auto-advance current_step)
  3. `POST /v1/onboarding/complete` — mark onboarding done and set completed_at timestamp
- **Frontend:** Dedicated `/onboarding` route with 4-step stepper (Company, Warehouse, Integration, Team). Each step calls existing API endpoints (PUT /v1/settings/company, POST /v1/warehouses, POST /v1/integrations, etc.) then marks step done via onboarding endpoint. Dashboard auto-redirects to /onboarding if onboarding.completed==false (checked via Next.js middleware and auth context).
- **Auth:** All endpoints JWT-required (under /v1 route with JWTAuth middleware), RLS-scoped to current tenant. Frontend route protected by Next.js middleware (unauthenticated users redirected to /login).
- **Backward compatibility:** Existing tenants (onboarding key missing or `{dismissed: true}`) treated as completed=true → no redirect. New tenants (registered after feature) start with current_step=1, completed=false.
- **Files:** Backend `settings_handler.go` (3 new methods), frontend `app/(dashboard)/onboarding/page.tsx` (4-step form), `app/(dashboard)/layout.tsx` (redirect check), model `user.go` (OnboardingSettings struct)
- **Consequences:** Tenants must complete step 1 (company details) before advancing, but can skip steps 2-4. Onboarding state is atomic per tenant (shared JSONB prevents race conditions via PostgreSQL transaction isolation). Dashboard banner reminds users to finish if they choose "Finish later" before completion.

## ADR-022: Shipper Address Resolution from Warehouse + CompanySettings Fallback
- **Date:** 2026-03-02
- **Context:** DHL24 SOAP API requires shipper address (sender name, street, city, postal code, phone) in every shipment creation request. Users create multiple warehouses; each has an address.
- **Decision:** (1) Add `CarrierSender` struct with fields: `Name`, `Street`, `City`, `PostalCode`, `Phone`, `Email`, `Country`. (2) Add optional `Shipper *CarrierSender` to `CarrierShipmentRequest`. (3) `LabelService.GenerateLabel()` resolves shipper via: (a) warehouse.Address JSON (preferred) → unmarshal to `CarrierSender`, or (b) tenant CompanySettings fallback (company name, country defaults to "PL" for DHL24-only context). (4) `WarehouseRepo.FindDefault()` returns the tenant's default active warehouse by `is_default=true AND active=true`.
- **Files:** `integration/carrier.go` (CarrierSender struct), `service/label_service.go` (resolveShipper method), `repository/warehouse_repo.go` (FindDefault method), `carriers/dhl.go` (shipper SOAP mapping). DPD/GLS ignore Shipper for now.
- **Alternatives rejected:**
  - Store shipper in each shipment request (redundant, requires separate shipper_address column)
  - Hardcode warehouse 1 as shipper (breaks multi-warehouse setups)
  - Skip shipper if not provided (carriers error/use DHL defaults, unreliable)
- **Consequences:**
  - Shipper is resolved lazily per label generation (reads warehouse + tenant once per request)
  - Fallback to CompanySettings is expected behavior — no shipper resolution warning needed for this path (may be added in future for international carrier support)
  - Polish street/house number splitting required for DHL24/DPD SOAP — added `splitStreetHouseNo()` helper to parse "ul. Warszawska 10" → street="ul. Warszawska", houseNo="10"
  - Warehouse address must be valid JSON or field mapping fails (silent nil return, carrier provides error on missing shipper)
  - International carrier support (FedEx, UPS) will reuse this pattern but add country field to CompanySettings for tenant default country

## ADR-023: Service Type Mapping (Frontend Codes → Carrier API Codes)
- **Date:** 2026-03-02
- **Context:** Frontend offers `dhl_parcel`, `dhl_courier`, etc. (human-readable UI labels). DHL24 SOAP API requires specific codes: `AH` (domestic standard), `09` (before 9:00), `12` (before 12:00), `EK` (Express), `PI` (Parcel International). Old code passed frontend strings directly to SOAP (caused SOAP faults).
- **Decision:** Each carrier provider implements `mapServiceType(serviceName string) (string, error)` function. (1) DHL: `dhl_parcel→AH`, `dhl_courier→DR`, unknown→error. (2) DPD: `dpd_classic→standard`, `dpd_pickup→pickup`, with validation. (3) GLS: `gls_parcel→PARCEL`, `gls_express→EXPRESS`. DHL rejects unknown service types with a clear error; DPD validates against allowlist; GLS uses passthrough.
- **Security note:** Service type strings come from `carrier_data` JSON in DB (populated from frontend). `encoding/xml` auto-escapes values when marshaling to SOAP XML, so there's no injection risk. DHL and DPD now validate against allowlist and return error for unknown types.
- **Files:** `carriers/dhl.go:mapDHLServiceType()`, `carriers/dpd.go:mapDPDServiceType()`, `carriers/gls.go:mapGLSServiceType()`
- **Consequences:** All 3 carriers now map service types correctly. Frontend users cannot accidentally send SOAP faults. Future carriers must implement mapping or face passthrough risk.

## ADR-024: DPD Carrier Production-Ready Implementation
- **Date:** 2026-03-03
- **Context:** ADR-018 documented DPD REST API audit findings. DPD adapter required: (1) SDK field additions for service type/target point support, (2) shipper address mapping (warehouse + CompanySettings fallback, following ADR-022 pattern), (3) service type mapping via `mapDPDServiceType()` (ADR-023), (4) production specification tests verifying against official API contract, (5) cleanup test mock handlers.
- **Decision:** Complete DPD production-ready implementation via 5-step plan: (1) `packages/dpd-go-sdk/models.go`: add `ServiceType` and `TargetPoint` fields to `CreateParcelRequest`; (2) `packages/dpd-go-sdk/doc.go`: remove "In Development" status marking SDK production-verified; (3) `carriers/dpd.go`: implement `mapDPDServiceType()` (dpd_classic/dpd_pickup allowlist), shipper resolution via `resolveShipper()` helper, pickup point validation; (4) `carriers/dpd_test.go`: remove dead `/auth/login` mock handler; (5) `carriers/dpd_production_test.go`: add 7 new test cases (service type mapping, shipper resolution, tracking/cancel error handling verifying REST API does not support these, COD/Insurance fields).
- **Files:** `packages/dpd-go-sdk/models.go` (+ServiceType, +TargetPoint), `packages/dpd-go-sdk/doc.go` (removed "In Development"), `carriers/dpd.go` (+mapDPDServiceType, +shipper integration), `carriers/dpd_test.go` (cleanup), `carriers/dpd_production_test.go` (+7 tests)
- **Security audit:** PASS — no CRITICAL/HIGH findings. Carrier SDK matches official dpdservices.dpd.com.pl REST API contract (Basic Auth + x-dpd-fid header, two-phase label flow). 1 MEDIUM item noted: hardcoded placeholder rates in GetRates (tracked in SECURITY_POSTURE backlog).
- **Consequences:** DPD carrier now production-ready. Service type validation prevents invalid API calls. Shipper address properly resolved from warehouse (first) or CompanySettings (fallback). Frontend users can set COD/Insurance for DPD shipments (backend already supported, form fields were pre-added in previous audit fix).

## ADR-025: GLS Carrier Production-Ready Implementation
- **Date:** 2026-03-03
- **Context:** ADR-019 documented GLS ShipIT REST API verification. GLS adapter required: (1) SDK hardening with response body limits, (2) label retrieval from inline PrintData in CreateShipment response (not separate API call), (3) service type mapping via `mapGLSServiceType()` (ADR-023), (4) shipper/ContactID documentation (GLS uses pre-registered contacts, not inline addresses), (5) production specification tests verifying against official API contract.
- **Decision:** Complete GLS production-ready implementation via 4-step plan: (1) `packages/gls-go-sdk/client.go`: add `io.LimitReader(resp.Body, 10*1024*1024)` (10MB cap) on response body reads; (2) `packages/gls-go-sdk/doc.go`: remove "In Development" status marking SDK production-verified; (3) `carriers/gls.go`: implement `mapGLSServiceType()` (standard/express_10/express_12 allowlist), extract and cache label data from CreateShipment PrintData field (base64 decode), return clear error from GetLabel explaining labels are inline, document Shipper/ContactID limitation; (4) `carriers/gls_production_test.go`: add 7 new test cases (service type mapping, unknown type error, label retrieval error semantics, COD/insurance mapping, shipper isolation, reference propagation).
- **Files:** `packages/gls-go-sdk/client.go` (+io.LimitReader), `packages/gls-go-sdk/doc.go` (removed "In Development"), `carriers/gls.go` (+mapGLSServiceType, +label caching, +GetLabel error), `carriers/gls_production_test.go` (+7 tests)
- **Label retrieval design:** GLS ShipIT API returns labels embedded in CreateShipment response as base64-encoded PrintData array. No separate getLabel call. Adapter: (1) decode PrintData[0] to PDF bytes, (2) cache in per-instance map, (3) GetLabel returns cached bytes. This differs from DHL (separate getLabels call) and DPD (labels in response but mapped differently).
- **Security audit:** PASS — no CRITICAL/HIGH findings. SDK uses Basic Auth (over HTTPS) + io.LimitReader preventing OOM. Service type validation prevents invalid API calls. Shipper is resolved per warehouse (not hardcoded). 1 MEDIUM item noted: hardcoded placeholder rates in GetRates (tracked in SECURITY_POSTURE backlog, pre-existing from carrier SDK audit).
- **Consequences:** GLS carrier now production-ready. Service type validation prevents invalid API calls. Labels correctly retrieved from CreateShipment response (no separate API call needed). Shipper address determined by warehouse; GLS-specific ContactID configured in integration settings (limitation documented in adapter code). All 3 carriers (DHL, DPD, GLS) now follow consistent production-ready pattern: SDK verified against official docs, service type mapping with validation, shipper resolution (where carrier supports it), production tests covering integration contracts.
