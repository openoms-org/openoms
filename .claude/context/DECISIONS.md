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
