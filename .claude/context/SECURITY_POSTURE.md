# Security Posture
Last full audit: 2026-02-17

## Unfixed Findings

### HIGH
1. **WebSocket CheckOrigin allows all origins** — `apps/api-server/internal/handler/ws_handler.go:17`
   - Risk: Cross-Site WebSocket Hijacking (CSWSH)
   - Fix: Validate Origin header against configured FrontendURL
   - Effort: S

2. **Automation webhook action — no SSRF protection** — `apps/api-server/internal/automation/actions.go:330`
   - Risk: SSRF to internal services, cloud metadata endpoints
   - Fix: Use `noPrivateDialer()` transport (already exists in webhook_dispatch_service.go)
   - Effort: S

### MEDIUM
3. **Allegro webhook body not size-limited** — `apps/api-server/internal/handler/allegro_webhook_handler.go:38`
   - Risk: OOM DoS via large request body
   - Fix: `http.MaxBytesReader(w, r.Body, 1<<20)` before `io.ReadAll`
   - Effort: XS

4. **InPost webhook body not size-limited** — `apps/api-server/internal/handler/inpost_webhook_handler.go:28`
   - Risk: Same as #3
   - Fix: Same as #3
   - Effort: XS

### PLANNED (from CSRF plan)
5. **CSRF cookie not accessible cross-subdomain** — `apps/api-server/internal/middleware/csrf.go`
   - Risk: CSRF protection broken for app.openoms.org → api.openoms.org
   - Fix: Add Domain=.openoms.org to CSRF cookie, change SameSite to Lax
   - Plan: fancy-cuddling-snail.md

## Recently Fixed
- 2026-02-20: JSONBCodec nil Marshal/Unmarshal → panic on Scan (production login broken)
- 2026-02-20: json.RawMessage sent as bytea hex in simple_protocol mode
- 2026-02-19: RLS policies missing_ok for transaction mode pooler
- 2026-02-17: NetworkPolicy egress rules for Supabase

## Threat Model (Priority Order)
1. **Multi-tenant data leak** via RLS bypass — CATASTROPHIC
2. **Integration credential exposure** via logging or error messages — HIGH
3. **SSRF** via automation webhooks / supplier feed URLs — HIGH
4. **CSRF** on mutation endpoints (cross-subdomain) — MEDIUM
5. **XSS** via product descriptions from external sources — LOW (React escapes, pgx parameterizes)
6. **DDoS** via webhook endpoints without body limits — LOW (Cloudflare WAF mitigates)

## Security Architecture
- JWT: Ed25519 signed, 1h access, 30d refresh (httpOnly cookie)
- Passwords: bcrypt cost 12
- 2FA: TOTP (Google Authenticator), encrypted secret in DB
- Encryption: AES-256-GCM for integration credentials
- Multi-tenant: PostgreSQL RLS per transaction
- RBAC: Custom roles with granular permissions
- SSRF: noPrivateDialer for webhook dispatch (NOT yet for automation actions)
- Webhooks outgoing: HMAC-SHA256 signed
- K8s: PSS enforce:restricted, NetworkPolicies default-deny
- Headers: CSP, X-Frame-Options:DENY, X-Content-Type-Options:nosniff
