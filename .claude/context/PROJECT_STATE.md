# Project State
Updated: 2026-05-06

## Target
Open production for paying customers: **May 2026**

## Pricing Model
Subscription tiers based on order volume. Plan names and pricing configured at runtime via BILLING_PLANS env var.

## Current Focus (P0 — must do next)
- [x] Billing/Subscription integration (Stripe) — **DONE** (PR #67+)
- [x] Monitoring/Alerting (Grafana/Sentry) — **DONE** (Alloy metrics, 9 Grafana alerts, Sentry connected)
- [x] Allegro competitive parity — **DONE** (PR #58)
- [x] Onboarding wizard — **DONE** (PR #80+)
- [x] Full System Audit #2 (6-module) — **DONE** (30 findings, all fixed)

## Completed
- **Linear: backlog clean** — all tracked issues resolved or closed
- Production Readiness project: 22/22 DONE
- System Audits: 2 complete rounds (2026-03-17 8-module + 2026-04-16 6-module)
- Audit #2 findings (5 CRITICAL + 14 HIGH + 5 MEDIUM): all fixed across PRs #233/235/251/253
- Security updates: Next.js 16.2.3, Go 1.25.9 (CVE-2026-32280/32282), Alpine base rebuild
- Sentry bugs (5/5 resolved)

## Recently Completed
- 2026-05-07: OPE-203 — marketplace/BaseLinker order imports now have a partial DB uniqueness guard for non-empty external IDs and use atomic insert-or-skip behavior to prevent duplicate downstream side effects under concurrent poller/webhook/import races.
- 2026-05-07: OPE-207 — backend plan guard now enforces Stripe subscription status, blocks writes for past_due/unpaid/incomplete/canceled/paused/incomplete_expired subscriptions, blocks authenticated API access for suspended subscriptions, and invalidates cached plan state on Stripe webhook sync.
- 2026-05-07: OPE-215 — API startup now requires Redis for shared auth/session/rate-limit/OAuth/WebSocket/worker-lock state outside development, with `ALLOW_IN_MEMORY_STATE` as an explicit single-node self-host override.
- 2026-05-06: OPE-214 — tenant settings secrets are encrypted at rest inside `tenants.settings`, legacy plaintext settings are backfilled by worker startup, and settings responses/export mask sensitive fields.
- 2026-05-06: OPE-213 — billing/license/tenant-plan SECURITY DEFINER functions now revoke default PUBLIC execute, with a CI database check preventing regressions.
- 2026-05-06: OPE-205 — generated supplier portal links now use URL fragments for raw portal tokens, and the public portal page rejects query-string token handoff.
- 2026-05-06: OPE-201 — generic Allegro/InPost webhook intake now rejects known providers when the webhook secret is missing instead of accepting unsigned events.
- 2026-05-06: OPE-202 — `REGISTRATION_MODE=closed` now blocks public registration; legacy `disabled` remains blocked and invalid runtime modes fail closed.
- 2026-05-06: OPE-200 — supplier portal message listing now enforces purchase-order ownership for the supplier token and hides draft/foreign orders as not found.
- 2026-05-05: OPE-209 — refresh token rotation hardened: atomic store consume, required non-revoked token family, non-current sibling invalidation, and no family resurrection after revocation.
- 2026-04-16: Security updates — Next.js 16.2.3, Go 1.25.9 (CVE-2026-32280/32282), Alpine libcrypto3/musl rebuild
- 2026-04-16: Audit #2 final batch (PR #253) — WebSocket hub race fix, refresh token fail-close on Redis, OLX pagination, batch shipments filter (order_ids)
- 2026-04-08: Audit #2 batch 3 (PR #251) — 7 io.LimitReader additions (Allegro/eBay SDK, GLS), AuthProvider 429 retry, lockout 30s→1m, worker cursor stall log
- 2026-04-02: Audit #2 batch 2 (PR #235) — rate limiter Close(), off-by-one, Freshdesk domain injection validation, useOnboarding auth guards, PII removed from logs, DataTable key, breadcrumb a11y
- 2026-03-30: Audit #2 batch 1 (PR #233) — token refresh mutex race, eBay HTTP timeout, Allegro cursor stall, MaxBytesReader panic, supplier SSRF
- 2026-03-27: OPE-54 — 360 Polish i18n keys renamed to English camelCase across 119 files
- 2026-03-25: OPE-56 — 963 lines of handler unit tests (8 handlers, 51 test cases)
- 2026-03-24: OPE-120 — API client deduplication via fetchWithAuth wrapper
- 2026-03-23: OPE-102 — N+1 batch queries via FindByIDs (BulkTransitionStatus, MergeOrders, ListingSync)
- 2026-03-23: OPE-53 — hide incomplete features (KSeF, Marketing, Helpdesk, Notifications) from navigation
- 2026-03-23: E2E infrastructure rework — 3/124 → 122/131 passing (SSR location fix, CSP eval, production build on CI, Polish locale, 15+ string fixes)
- 2026-03-22: OPE-45/46/47 — SDK maturity (DHL/DPD/GLS verified), carrier rates marked as estimates, Kaufland/Empik removed from marketplace picker
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
