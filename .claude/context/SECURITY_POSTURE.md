# Security Posture
Last full audit: 2026-02-25 (3 rounds: PR #36, #38, #42; test hardening: PRs #43-56)

## Unfixed Findings

### BACKLOG (low priority, separate PRs)
1. **`writeError` → `writeServerError` migration** — 371 call sites use generic `writeError` for internal server errors
   - Risk: inconsistent error responses
   - Fix: batch rename + add structured error codes
   - Effort: L (mechanical but wide-reaching)

2. **Bare `return err` without wrapping** — 580+ sites in services
   - Risk: poor error context in logs
   - Fix: wrap with `fmt.Errorf("operation: %w", err)` incrementally
   - Effort: XL

3. **CSP `unsafe-inline` removal** — `apps/api-server/internal/middleware/security.go`
   - Risk: weakened Content Security Policy
   - Fix: nonce-based CSP (requires Next.js integration for script nonces)
   - Effort: L

## Recently Fixed
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
- Headers: CSP, X-Frame-Options:DENY, X-Content-Type-Options:nosniff, Referrer-Policy:strict-origin-when-cross-origin
