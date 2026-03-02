# Security Posture
Last full audit: 2026-02-25 (4 rounds: PR #36, #38, #42, #58; test hardening: PRs #43-56)
Carrier SDK audit (DHL, DPD, GLS): 2026-03-01 (PASS — all critical issues fixed: SOAP response parsing corrected, GLS model fields aligned, DHL service types validated, DPD COD form added)
Carrier fields fix security audit: 2026-03-01 (PASS — zero XSS/injection vectors, hardcoded values only, React auto-escape, commits 4ef72f9 + 62eef14)
Billing/Stripe integration security audit: 2026-03-02 (PASS — Stripe webhook signature verification, checkout session anti-replay, SECURITY DEFINER for pre-registration, no Stripe Price IDs exposed to frontend, runtime config only)
Onboarding wizard security audit: 2026-03-01 (PASS — JWT auth on all backend endpoints, Next.js middleware protecting routes, no credential exposure, backward-compatible JSONB migration)

## Unfixed Findings

### HIGH (should fix immediately — affects production)
1. **`carrier-fields.tsx:248-249` — Invalid DHL service types** — Frontend offers `"dhl_parcel"` and `"dhl_courier"` which are NOT valid DHL24 SOAP service types. DHL24 uses: `AH` (domestic standard), `09` (before 9:00), `12` (before 12:00), `EK` (Express), `PI` (Parcel International). Backend passes strings through to SOAP without validation → SOAP fault at DHL
   - Risk: user selects DHL, shipment creation fails at carrier API, confusing error message
   - Files: `apps/dashboard/src/components/shipments/carrier-fields.tsx:248-249`
   - Fix: Change to valid DHL codes (`AH`, `09`, `12`, `EK`, `PI`) with proper labels, or add backend validation/mapping
   - Effort: S

2. **`carrier-fields.tsx:DPDFields` — Missing COD/Insurance form fields** — `DPDFields` renders only `ParcelDimensionFields` but NOT `CODAndInsuranceFields`. Backend (`dpd.go:78-99`) fully supports COD (paymentType: "COD") and insurance (`insuranceValue`). Users cannot set COD for DPD shipments from the UI
   - Risk: DPD COD feature unavailable to users despite backend support; lost sales for COD orders
   - Files: `apps/dashboard/src/components/shipments/carrier-fields.tsx` (DPDFields component)
   - Fix: Add `<CODAndInsuranceFields values={values} onChange={onChange} />` to `DPDFields` render
   - Effort: S

### BACKLOG (low priority, separate PRs)
1. **Carrier SDK MEDIUM findings** (deferred to follow-up)
   - `dhl-go-sdk/client.go:119`, `dpd-go-sdk/client.go:122`, `gls-go-sdk/client.go:113` — Unbounded `io.ReadAll` in response parsing. A compromised/malicious carrier API endpoint could cause OOM. Typical carrier responses <1MB.
     - Fix: Use `io.LimitReader(resp.Body, 10*1024*1024)` (10MB cap) on all response body reads
   - `dhl-go-sdk/client.go:14` — Sandbox no-op. `WithSandbox()` is silently ignored. A developer setting `sandbox: true` in credentials expects safety but gets production. Documented in comment but could surprise.
     - Fix: Return error or log warning when sandbox is requested but not fully supported
   - `dhl.go:159-233`, `dpd.go:157-177`, `gls.go:193-213` — Hardcoded placeholder rates. All 3 carriers return hardcoded PLN prices from `GetRates()`. Marked as TODO but if reachable in production, users see fabricated pricing.
     - Fix: Ensure UI/API layer gates these behind a "rates_configured" flag, or return empty until real API integration
   - `dhl-go-sdk/models.go` — Dual serialization concern. Models have JSON tags but SDK uses SOAP/XML transport. JSON tags serve integration layer API but confusing and could cause issues if SDK used directly with JSON.
     - Fix: Document clearly that JSON tags are for integration layer only; SOAP uses XML marshaling

2. **`writeError` → `writeServerError` migration** — 371 call sites use generic `writeError` for internal server errors
   - Risk: inconsistent error responses
   - Fix: batch rename + add structured error codes
   - Effort: L (mechanical but wide-reaching)

3. **Bare `return err` without wrapping** — 580+ sites in services
   - Risk: poor error context in logs
   - Fix: wrap with `fmt.Errorf("operation: %w", err)` incrementally
   - Effort: XL

4. **CSP `unsafe-inline` removal** — `apps/api-server/internal/middleware/security.go`
   - Risk: weakened Content Security Policy
   - Fix: nonce-based CSP (requires Next.js integration for script nonces)
   - Effort: L

5. **Erli SDK MEDIUM findings** (deferred to follow-up)
   - `client.go:109` — no scheme validation on `WithBaseURL()` (allow http on localhost only, require https for others)
   - `offers.go:61-64` — global `slog.Warn` in library package (thread logger through OfferService)
   - `provider.go:186-188` — `RawData` nil on unknown status (set fallback map with erli_status + oms_status=unknown)
   - `go.mod:1` — Go 1.25.0 behind on patches (bump to 1.25.7+ for CVE-2025-47910, CVE-2025-58186, CVE-2025-61726)

## Recently Fixed
- 2026-03-02: Billing/Stripe integration security review COMPLETE:
  - **Stripe webhooks:** Signature verification via stripe.ConstructEvent, raw body parsing, webhook secret from env
  - **Checkout sessions:** Anti-replay via atomic status transitions (pending→completed→registered), session claimed once per tenant
  - **Pre-registration ops:** SECURITY DEFINER functions for checkout session CRUD (bypass RLS before tenant exists)
  - **Credential isolation:** Stripe Price IDs never exposed to frontend (ListPlans strips them), secrets in env vars only
  - **Registration:** checkout_session_id verified against Stripe, email must match session email
  - **Verdict:** PASS — no credential exposure, proper isolation, anti-replay protection
- 2026-03-02: Onboarding status 401 fix — added isAuthenticated guard to prevent unauthenticated API calls (PR #86)
- 2026-03-02: Shipments NULL scan fix — nullable DB columns now use pointer types in Go structs (PR #83)
- 2026-03-01: Onboarding wizard security review COMPLETE:
  - **Backend endpoints:** All 3 endpoints (`GET/PUT/POST /v1/onboarding/*`) behind JWT auth middleware, tenant-scoped via RLS context
  - **Frontend routes:** `/onboarding` protected by Next.js middleware (unauthenticated users redirected to `/login`), checked on every request
  - **State storage:** JSONB in tenants.settings, backward compatible (existing tenants treated as completed), no PII or credentials exposed
  - **Form submission:** All steps use existing secure endpoints (PUT /v1/settings/company, POST /v1/warehouses, POST /v1/integrations, etc.), CSRF protection inherited
  - **Verdict:** PASS — zero new security concerns identified
- 2026-03-01: Carrier SDK audit remediation COMPLETE:
  - **DHL24 SOAP WebAPI2**: Replaced fictional REST API (commit 9859edb) with correct SOAP envelope marshaling; corrected 5 service types (AH, 09, 12, EK, PI); test suite rewritten for XML responses
  - **DPD REST API**: Corrected to official dpdservices.dpd.com.pl endpoint (commit 92727d7); fixed session-based auth; implemented two-phase label flow; added COD/Insurance frontend fields
  - **GLS ShipIT API**: Fixed Basic Auth (was Bearer), tracking GET→POST (commit 80a8663), cancel DELETE→POST (commit f4b9419); aligned models to API spec; corrected test assertions (Service/Product fields); added COD propagation
  - **Specification tests**: Added comprehensive test suites for all 3 carriers (commit 2943d6b) verifying SDK responses match official documentation
  - **Frontend**: DHL service types corrected from arbitrary strings to valid codes; DPD COD/Insurance form fields added
- 2026-03-01: Carrier SDK audit (DHL, DPD, GLS) completed — verified base URLs, auth methods, endpoints, response models, and status mappings against official API documentation. Found 4 CRITICAL test suite issues, 2 HIGH frontend bugs, 4 MEDIUM best practices. All CRITICAL+HIGH issues now fixed.
- 2026-03-01: Carrier fields fix — FedEx/UPS/GLS/PP service type corrections, GLS backend wiring (PR #77)
- 2026-02-28: Erli SDK rebuild + sandbox fail-open fix — hard error on sandbox=true without base_url (PR #76)
- 2026-02-25: PR #58 — Allegro competitive parity with full security audit: SendToKSeF three-phase refactor (no DB during external calls), unique index on product_listings(external_id, integration_id), per-tenant ImportOffers concurrency guard, KSeF session defer-terminate, message template body max length validation, raw error leak fix in PushListing, proper Allegro provider in automation (token refresh), warehouse doc stock quantities fix
- 2026-02-25: PRs #43-56 — Massive test coverage expansion: SDK tests for all 27 packages (100%), middleware tests for all 18 files (100%), worker tests for all 19 files (100%), handler validation tests for all handlers, model Validate() coverage for 54 request types, service-layer tests (carbon, repricing, webhook, dropship, invoice, automation conditions)
- 2026-02-25: PR #52 — SafeGo helper for goroutine panic recovery (prevents silent worker crashes)
- 2026-02-25: PR #42 — Audit remediation v3: additional security fixes, error handling improvements, code quality
- 2026-02-25: PR #41 — Weight propagation: supplier sync writes product weight, shipment auto-calculates weight from order items
- 2026-02-25: PR #40 — Allegro integration hardening: retry with backoff, bulk sync, order deduplication
- 2026-02-24: PR #38 — SSRF IPv6 bypass (::/128, ff00::/8), atomic rate limiter (Lua script), WS ticket-only auth, Swagger CDN pinned to 5.18.2, XSS dangerouslySetInnerHTML removed, Polish errors translated, discarded DB errors logged, settings validation (Email/SMS/Invoicing), 12 dead code items removed, 12 hooks migrated to createCrudHooks
- 2026-02-24: PR #36 — CSRF double-submit cookie middleware, composite token blacklist (Redis + in-memory), WebSocket Origin validation + ticket-only auth, automation SSRF fix (noPrivateDialer), webhook body size limits (1MB), HSTS header, security headers hardening, input sanitization (StripTags), response helpers (writeServerError, writeValidationError)
- 2026-02-20: JSONBCodec nil Marshal/Unmarshal → panic on Scan (production login broken)
- 2026-02-20: json.RawMessage sent as bytea hex in simple_protocol mode
- 2026-02-19: RLS policies missing_ok for transaction mode pooler
- 2026-02-17: NetworkPolicy egress rules for Supabase

## Threat Model (Priority Order)
1. **Multi-tenant data leak** via RLS bypass — CATASTROPHIC (mitigated: RLS + FORCE ROW LEVEL SECURITY)
2. **Integration credential exposure** via logging or error messages — HIGH (mitigated: AES-256-GCM, no plaintext in logs)
3. **SSRF** via automation webhooks / supplier feed URLs — HIGH → **MITIGATED** (noPrivateDialer on all outbound, IPv4+IPv6 ranges blocked)
4. **CSRF** on mutation endpoints (cross-subdomain) — MEDIUM → **MITIGATED** (double-submit cookie + X-CSRF-Token header, Domain=.openoms.org)
5. **XSS** via product descriptions from external sources — LOW → **MITIGATED** (dangerouslySetInnerHTML removed, React auto-escape, input sanitization)
6. **DDoS** via webhook endpoints without body limits — LOW → **MITIGATED** (MaxBytesReader 1MB on all webhook handlers)

## Security Architecture
- JWT: Ed25519 signed, 1h access, 30d refresh (httpOnly cookie)
- Passwords: bcrypt cost 12
- 2FA: TOTP (Google Authenticator), encrypted secret in DB
- Encryption: AES-256-GCM for integration credentials
- Multi-tenant: PostgreSQL RLS per transaction
- RBAC: Custom roles with granular permissions
- CSRF: double-submit cookie (X-CSRF-Token header + csrf_token cookie, SameSite=Lax, Domain configurable)
- Token blacklist: composite (Redis primary + in-memory fallback, prevents fail-open)
- Rate limiting: atomic Redis Lua script (INCR+EXPIRE in single operation)
- SSRF: noPrivateDialer on all outbound connections (webhooks, automation, supplier feeds) — IPv4 + IPv6 private ranges
- Webhooks outgoing: HMAC-SHA256 signed
- HSTS: Strict-Transport-Security in production
- K8s: PSS enforce:restricted, NetworkPolicies default-deny
- Stripe: webhook signature verification (Stripe-Signature header), checkout session anti-replay (atomic DB status transitions)
- License tokens: Ed25519 signed JWTs with JTI replay protection (used_license_tokens table)
- Headers: CSP, X-Frame-Options:DENY, X-Content-Type-Options:nosniff, Referrer-Policy:strict-origin-when-cross-origin
- DB connection safety: three-phase pattern for external API calls (no DB held during HTTP), deferred KSeF session cleanup
- Concurrency: per-tenant mutex on ImportOffers, unique index on product_listings(external_id, integration_id)
- Input validation: message template body max 50k chars, name max 200 chars
