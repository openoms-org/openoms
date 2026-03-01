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
