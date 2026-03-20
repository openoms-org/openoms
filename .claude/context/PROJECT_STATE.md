# Project State
Updated: 2026-03-20

## Target
Open production for paying customers: **May 2026** (~8 weeks remaining)

## Pricing Model
Subscription tiers based on order volume. Plan names and pricing configured at runtime via BILLING_PLANS env var.

## Current Focus (P0 — must do next)
- [x] Billing/Subscription integration (Stripe) — **DONE** (PR #67+)
- [x] Monitoring/Alerting (Grafana/Sentry) — **DONE** (Alloy metrics, 9 Grafana alerts, Sentry connected)
- [x] Allegro competitive parity — **DONE** (PR #58)
- [x] Onboarding wizard — **DONE** (PR #80+)

## Completed
- **Linear: 109/119 issues done** across Production Readiness + Full System Audit
- Production Readiness project: 22/22 DONE
- Full System Audit (8 modules): 7 critical, 23 high, 57 medium, 63 low found → most fixed
- Sentry bugs (5/5 resolved): supplier sync, OLX scope, dashboard routing, CategorySuggestion
- CVE-2026-22184 (zlib), npm flatted + next 16.2.0 fixed

## In Progress / Remaining (10 tasks)
- OPE-45/46/47: SDK maturity decisions (13 unverified SDKs, hardcoded rates, Kaufland/Mirakl)
- OPE-102: N+1 query fixes (BulkTransitionStatus, MergeOrders, listing sync)
- OPE-120: Frontend API client code deduplication
- OPE-81: i18n — 29 namespaces still need JSON files
- OPE-53/54/55/56: Frontend quality (DevelopmentBanner, Polish keys, large components, handler tests)

## Recently Completed
- 2026-03-20: Full system audit fixes batch 5 — invoice HTTP-out-of-TX, SDK LimitReader (12 packages), Alloy PG/Redis scrape targets
- 2026-03-20: npm vulns fixed (flatted + next 16.2.0), cloudflared 2026.3.0, deploy debug timeout
- 2026-03-20: GLS labels persisted to S3 storage (OPE-112), LocalStorage.Get method
- 2026-03-20: Deploy pipeline fixed — staging RBAC bootstrap via ClusterRoleBinding
- 2026-03-17: Full 8-module system audit completed (docs/audit/2026-03-17-executive-summary.md)
- 2026-03-17: RLS on 20 missing tables + stocktakes fallback fix (migration 000016)
- 2026-03-17: FK indexes on orders.customer_id, returns.order_id, shipments.warehouse_id (migration 000017)
- 2026-03-17: Handler security — PII removed from URLs, OLX error sanitized, InPost webhook wired to DB
- 2026-03-17: Carrier HTTP timeouts (30s on all 8 providers), auth refresh fail-close, token reuse → Sentry
- 2026-03-16: Production readiness 22/22, i18n 8 pages translated, distributed locks, security hardening
- 2026-03-15: CSP hotfix, monitoring verified, Grafana alerts provisioned
- 2026-03-03: GLS carrier production-ready COMPLETE — SDK hardening + label retrieval + service type mapping:
  - **SDK fixes:** Added `io.LimitReader(resp.Body, 10*1024*1024)` (10MB cap) on response body reads; removed "In Development" status from doc.go
  - **Label retrieval:** `CreateShipment` response contains inline PrintData base64. Adapter decodes and caches per-provider instance. `GetLabel()` returns cached PDF bytes or clear error explaining labels are embedded, not separate API
  - **Service type mapping:** Added `mapGLSServiceType()` function mapping frontend `standard` → GLS PARCEL, `express_10` → EXPRESS_10, `express_12` → EXPRESS_12, with proper allowlist validation and error on unknown types
  - **Shipper/ContactID mapping:** Documented GLS limitation — GLS uses pre-registered ContactID (carrier-specific), not inline shipper address. Adapter ignores `Shipper` field; users configure default account/ContactID in integration settings
  - **COD/Insurance support:** Service objects mapped correctly (`service_cash` for COD, `service_addonliability` for insurance)
  - **Production tests:** Added 7 new test cases in `gls_production_test.go` covering service type mapping, unknown type error handling, label retrieval error semantics, COD/insurance mapping, shipper isolation, reference field propagation
  - **Security audit:** PASS (no CRITICAL/HIGH findings; 1 MEDIUM: hardcoded placeholder rates in GetRates, tracked in SECURITY_POSTURE backlog)
- 2026-03-03: DPD carrier production-ready COMPLETE — SDK alignment + service type mapping:
  - **SDK fixes:** Added `ServiceType` and `TargetPoint` fields to `CreateParcelRequest` in `packages/dpd-go-sdk/models.go`; removed "In Development" status from SDK doc.go
  - **Service type mapping:** Added `mapDPDServiceType()` function mapping frontend `dpd_classic` → DPD REST code, `dpd_pickup` → DPD pickup service, with proper validation
  - **Shipper support:** Integrated shipper mapping (follows DHL pattern), optional `Shipper *CarrierSender` field in adapter
  - **Pickup point validation:** Added validation for `dpd_pickup` service type requiring valid `TargetPoint`
  - **Production tests:** Added 7 new test cases in `dpd_production_test.go` covering service type mapping, shipper resolution, tracking/cancel error handling (DPD REST API does not support these via REST)
  - **Test cleanup:** Removed dead `/auth/login` mock handler from `dpd_test.go`
  - **API contract:** DPD REST API verified against official dpdservices.dpd.com.pl documentation (Basic Auth + x-dpd-fid header, two-phase label flow)
  - **Status:** Code merged, security audit PASS (no CRITICAL/HIGH findings; 1 MEDIUM item: hardcoded placeholder rates in GetRates, tracked in SECURITY_POSTURE backlog)
- 2026-03-02: DHL carrier production-ready COMPLETE — shipper address + SDK fixes:
  - **Shipper address support:** Added `CarrierSender` struct to `CarrierShipmentRequest`, resolved shipper from warehouse address or tenant `CompanySettings` fallback
  - **SDK fixes:** Corrected DHL24 doc.go (SOAP not REST), added missing Phone field to Shipper SOAP mapping, fixed duplicate Service/ServiceType in shipments.go, fixed silent timestamp parse errors
  - **Service type mapping:** Added `mapDHLServiceType()` function mapping frontend `dhl_parcel` → DHL24 code `AH` (domestic standard), `dhl_courier` → `DR` (courier domestic)
  - **Polish address splitting:** Added `splitStreetHouseNo()` helper for correct SOAP mapping (e.g. "ul. Warszawska 10" → street="ul. Warszawska", houseNo="10")
  - **LabelService enhancement:** Added `warehouseRepo` + `tenantRepo` dependencies, `FindDefault()` method on WarehouseRepo for default warehouse lookup
  - **Test coverage:** 6 new test cases covering shipper mapping, service type mapping, street splitting; specification tests verify against official DHL24 SOAP API
  - **Commits:** 6656870 (spec tests), 199cba1 (main implementation), 3d5fc39 (code review fixes)
  - **Status:** Code merged, security audit PASS (resolves existing HIGH finding about invalid DHL service types), 2 new HIGH items identified for future hardening (hardcoded country, service type passthrough)
- 2026-03-02: Billing/Stripe integration COMPLETE — full payment flow:
  - **Backend**: Stripe Checkout sessions, webhook handling (checkout.session.completed, subscription updates, payment failures), billing tables (customers, subscriptions, checkout_sessions), SECURITY DEFINER functions for pre-registration ops
  - **Frontend**: Plan selection page at /register (dynamic from API), /register/complete (post-Stripe form), invite flow preserved at /register/invite
  - **Auth**: Register with checkout_session_id, session claim with anti-replay, email verification
  - **Config**: Runtime via BILLING_PLANS env var (JSON), no hardcoded plans/prices in source
  - **Status**: All code merged, Stripe account setup pending (enterprise config)
- 2026-03-02: Onboarding status 401 fix (PR #86) — added isAuthenticated guard to useOnboardingStatus query
- 2026-03-02: Shipments NULL scan fix (PR #83) — nullable columns properly handled with pointer types
- 2026-03-01: Onboarding wizard COMPLETE — full multi-step setup flow for new tenants:
  - **Backend**: 3 new endpoints (`GET /v1/onboarding/status`, `PUT /v1/onboarding/step/{step}`, `POST /v1/onboarding/complete`), extended `OnboardingSettings` model (current_step, completed_steps, skipped_steps tracking in JSONB)
  - **Frontend**: Dedicated `/onboarding` route with 4-step stepper (Company details, Warehouse, Integration, Team invite), form-based flows reusing existing API endpoints, completion screen with redirect
  - **Auth protection**: JWT-required backend endpoints, Next.js middleware protecting `/onboarding` route, auto-redirect on login if onboarding incomplete
  - **State tracking**: Stored in tenants.settings JSONB, backward compatible with existing tenants (marked as completed)
  - **Status**: All 4 steps functional, step 1 required, steps 2-4 skippable, dashboard banner for "Finish later"
  - **Commits**: Multiple fixes after code review (0f3dc3e, 8f567b0) and security audit
- 2026-03-01: Carrier SDK audit & remediation COMPLETE (DHL, DPD, GLS verified):
  - **Audit findings**: 4 CRITICAL test failures, 2 HIGH frontend bugs, 4 MEDIUM best practices identified
  - **DHL24 SOAP WebAPI2**: Fictional REST API → correct SOAP marshaling, service types AH/09/12/EK/PI, XML response parsing
  - **DPD REST API**: Fictional URLs → dpdservices.dpd.com.pl, session auth, two-phase labels, COD+Insurance form fields
  - **GLS ShipIT API**: Bearer → Basic Auth, tracking GET→POST, cancel DELETE→POST, model alignment, test assertion fixes
  - **Status**: All 3 carriers now VERIFIED with specification tests (specs cover official API contracts)
  - **Commits**: 9859edb (DHL), 92727d7 (DPD), 80a8663+f4b9419 (GLS), 2943d6b (spec tests)
- 2026-03-01: Carrier fields fix — FedEx, UPS, GLS, Poczta Polska service type corrections + GLS backend wiring (PR #77)
- 2026-02-28: Erli SDK rebuild — base URL, endpoints, statuses, polling, pagination, 202 handling, sandbox fail-open fix (PR #76)
- 2026-02-25: Allegro competitive parity — offer import (SKU matching + auto-pagination), stock sync (per-channel push, error counts), message templates CRUD, send_marketplace_message action, KSeF auto-send + retry, activate_listing automation, full audit fixes (PR #58, 49 files, +5012/-139)
- 2026-02-25: Test coverage expansion — SDK 27/27, middleware 18/18, worker 19/19, handlers all covered, 54 model validators, service-layer tests (PRs #43-56)
- 2026-02-25: SafeGo panic recovery helper for goroutines (PR #52)
- 2026-02-25: Quiet CI/dev tooling — summary-only test output, eslint --quiet (PR #54)
- 2026-02-25: Audit remediation v3 — additional security, error handling, quality (PR #42)
- 2026-02-25: Weight propagation — supplier→product→shipment auto-calculation (PR #41)
- 2026-02-25: Allegro hardening — retry with backoff, bulk sync, order dedup (PR #40)
- 2026-02-24: Documentation update post-audit (PR #39)
- 2026-02-24: Audit remediation v2 — SSRF IPv6, atomic rate limiter, WS ticket-only auth, XSS fix, settings validation, dead code cleanup, 12 hook migrations (PR #38)
- 2026-02-24: Security hardening — CSRF middleware, composite token blacklist, WebSocket Origin validation, HSTS, automation SSRF fix, webhook body limits, input sanitization, response helpers (PR #36)
- 2026-02-20: JSONB type registration fix for pgx simple_protocol (AfterConnect + JSONBCodec)
- 2026-02-20: Tenant repository explicit jsonb cast
- 2026-02-19: Supplier product enrichment (PR #13)
- 2026-02-19: BTP.pro SDK and supplier integration
- 2026-02-17: Security audit completed (4 HIGH, 4 MEDIUM findings)
- 2026-02-17: Gap analysis vs competitors (BaseLinker, Sellasist, Apilo)

## Recent Deploys
- 2026-03-20: npm vulns (next 16.2.0), GLS S3 labels, staging RBAC fix, debug timeout
- 2026-03-17: RLS migration 016, FK indexes 017, handler security, carrier timeouts, CSV limits, Trivy blocking
- 2026-03-16: Full audit fixes — distributed lock, settings race, TestConnection, NetworkPolicy, ARC RBAC, Stripe secrets
- 2026-03-16: OPE-28 supplier sync async, OPE-30/31/32 OLX + dashboard fix
- 2026-03-15: Production readiness — HPA (PR #133), security hardening (PR #132), alerts provisioned, DR runbooks
- 2026-03-15: CSP strict-dynamic hotfix, i18n audit (PR #131), integration audit (PR #129), security audit (PR #130)
- 2026-03-02: PRs #83-86 merged — shipments NULL fix, billing tests, onboarding 401 fix
- 2026-03-01: PRs #76-82 merged — carrier SDK fixes (DHL, DPD, GLS, Erli), onboarding wizard, carrier fields
- 2026-02-25: PR #58 merged — Allegro competitive parity (offer import, stock sync, messaging, KSeF auto-send, audit fixes)
- 2026-02-25: PRs #40-56 merged — Allegro hardening, weight propagation, audit v3, test coverage expansion
- 2026-02-24: PR #38 merged — audit remediation v2
- 2026-02-24: PR #36 merged — security hardening + code quality
- 2026-02-20 14:00: `6bd6a7d` — JSONBCodec Marshal/Unmarshal fix (fixed login panic)
- 2026-02-20 13:50: `8dc4a19` — Tenant repository jsonb cast

## Active Blockers
- None currently blocking development

## MVP Critical Path
```
[DONE] Monitoring → [DONE] Allegro → Email templates → Regulamin → Soft launch
```
Note: Billing done, onboarding done, monitoring done, 4 carriers verified (InPost/DHL/DPD/GLS).

## Estimated Hours Remaining to MVP
~200h (8 weeks, ~25h/week capacity)
