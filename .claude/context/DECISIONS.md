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
