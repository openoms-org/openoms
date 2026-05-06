# API Contracts
Version: 10 (bump after every endpoint change)
Updated: 2026-05-06

## Recently Changed
- 2026-05-06: Generic webhook fail-closed hardening:
  - `POST /v1/webhooks/{provider}/{tenant_id}` now rejects known providers with missing webhook secrets using `422 {"error":"webhook secret not configured"}`.
  - Generic Allegro/InPost webhook events require HMAC verification; an empty configured secret no longer disables signature checks.
- 2026-05-06: Registration mode hardening:
  - `REGISTRATION_MODE=closed` now denies `POST /v1/auth/register` with `403 {"error":"registration is disabled"}`.
  - Legacy `REGISTRATION_MODE=disabled` is accepted as a closed-registration alias.
  - Unknown runtime registration modes fail closed in the auth handler instead of allowing open registration.
  - Public config `registration_mode` values are `open`, `invite`, `closed`, or legacy `disabled`.
- 2026-03-03: GLS REST API integration finalized (internal SDK contract, no public API endpoint changes):
  - `packages/gls-go-sdk/client.go`: Added `io.LimitReader(resp.Body, 10*1024*1024)` (10MB cap) on response body reads to prevent OOM
  - `packages/gls-go-sdk/doc.go`: Removed "In Development" status marking SDK as production-verified
  - `CarrierShipmentRequest` (internal): GLS adapter resolves shipper from warehouse address or uses account default; GLS API requires pre-registered ContactID (not inline address mapping like DHL/DPD)
  - Service type codes verified: `standard` → GLS PARCEL, `express_10` → EXPRESS_10, `express_12` → EXPRESS_12
  - Label retrieval: PrintData base64 from `CreateShipment` response is decoded and cached per provider instance; `GetLabel` returns cached bytes or error
  - This is internal refactoring, public API endpoints unchanged
- 2026-03-03: DPD REST API integration finalized (internal SDK contract, no public API endpoint changes):
- 2026-03-03: DPD REST API integration finalized (internal SDK contract, no public API endpoint changes):
  - `packages/dpd-go-sdk/models.go`: Added `ServiceType` and `TargetPoint` fields to `CreateParcelRequest`
  - `packages/dpd-go-sdk/doc.go`: Removed "In Development" status marking SDK as production-verified
  - `CarrierShipmentRequest` (internal): DPD adapter now supports optional shipper mapping via `Shipper *CarrierSender` field
  - Service type codes verified: `dpd_classic` → standard parcel, `dpd_pickup` → pickup point delivery (requires `TargetPoint`)
  - This is internal refactoring, public API endpoints unchanged
- 2026-03-02: Internal carrier integration contract extended (no public API endpoint changes):
  - `CarrierShipmentRequest` struct now includes optional `Shipper *CarrierSender` field (internal, not exposed in public API)
  - `WarehouseRepo.FindDefault()` method added (returns tenant's default active warehouse by is_default flag)
  - `CarrierSender` struct added with fields: Name, Street, City, PostalCode, Phone, Email, Country (resolved from warehouse.Address JSONB or tenant CompanySettings fallback)
  - DHL provider maps shipper address to SOAP (DHL24 requires it); DPD and GLS ignore `Shipper` for now
  - This is internal refactoring, public API endpoints unchanged
- 2026-03-02: Billing endpoints added (public, no JWT):
  - `GET /v1/billing/plans` — list available plans (without Stripe Price IDs), rate limit 60/min
  - `POST /v1/billing/checkout` — create Stripe Checkout session, rate limit 10/min
  - `GET /v1/billing/checkout/{session_id}` — get session status (plan, email, limits)
- 2026-03-02: Stripe webhook endpoint: `POST /v1/webhooks/stripe` — Stripe-Signature verified, handles checkout.session.completed, customer.subscription.updated/deleted, invoice.payment_failed
- 2026-03-02: Auth register extended — `checkout_session_id` field added to RegisterRequest for Stripe-based registration
- 2026-03-02: Public config extended — `billing_enabled`, `stripe_public_key` fields added to GET /v1/config response
- 2026-03-01: Onboarding wizard endpoints finalized (tenant-scoped, JWT required):
  - `GET /v1/onboarding/status` — returns current onboarding state (current_step, completed_steps[], skipped_steps[], completed flag)
  - `PUT /v1/onboarding/step/{step}` — mark step completed or skipped (step 1 non-skippable), idempotent
  - `POST /v1/onboarding/complete` — mark onboarding as done (sets completed=true, completed_at timestamp)
  - Response: `{"completed": bool, "current_step": int, "completed_steps": [], "skipped_steps": [], "completed_at": "ISO8601"}` stored in tenants.settings JSONB
  - Backward compatible: existing tenants default to completed=true or dismissed=true → no redirect
- 2026-03-01: Carrier SDK audit completed (internal SDKs, not public API) — DHL24 migrated to SOAP WebAPI2 (dhl24.com.pl/webapi2), DPD aligned to official REST API (dpdservices.dpd.com.pl), GLS verified against ShipIT REST v3.4.19. All base URLs, auth methods, endpoints, and response models verified against official documentation. Fix PR pending.
- 2026-02-28: Erli carrier integration endpoints rebuilt (internal SDK, not public API) — base URL fix, product creation via /products/{externalId}, stock/price update via PATCH /products/{externalId}, status mapping fixed (3 statuses: pending/purchased/cancelled), polling parameter (after), 202 async handling
- 2026-02-25: `POST /v1/integrations/allegro/import-offers` — new endpoint: imports all Allegro seller offers as Product + ProductListing with SKU matching (PR #58)
- 2026-02-25: `POST /v1/stock-sync/push/channel/{channel_id}` — new endpoint: trigger stock sync for a single channel (PR #58)
- 2026-02-25: Message templates CRUD — `GET/POST /v1/message-templates`, `GET/PUT/DELETE /v1/message-templates/:id` (admin-guarded writes) (PR #58)
- 2026-02-25: KSeF auto-send — invoices auto-sent to KSeF on creation when `auto_send` enabled in KSeF settings (PR #58)
- 2026-02-25: New automation actions: `activate_listing`, `send_marketplace_message` (PR #58)
- 2026-02-25: New automation trigger: `product.stock_restored` (fires when stock goes 0→>0) (PR #58)
- 2026-02-25: `POST /v1/shipments` — auto-calculates weight from order items when weight not provided (PR #41)
- 2026-02-25: Supplier sync now propagates product weight to products table (PR #41)
- 2026-02-25: Allegro order polling: retry with backoff, bulk sync, order deduplication (PR #40)
- 2026-02-24: All mutation endpoints (POST/PUT/PATCH/DELETE) now require CSRF token (`X-CSRF-Token` header matching `csrf_token` cookie)
- 2026-02-24: `GET /v1/ws` — ticket-only auth (JWT query param fallback removed)
- 2026-02-24: Settings validation added: EmailSettings (smtp_port 0-65535), SMSSettings (from max 11 chars), InvoicingSettings (default_tax_rate 0-100)
- 2026-02-19: `POST /v1/suppliers` — added `feed_url`, `feed_format`, `feed_mapping` fields
- 2026-02-19: `GET /v1/suppliers/:id/products` — new endpoint (supplier product catalog)
- 2026-02-19: `POST /v1/suppliers/:id/products/enrich` — new endpoint (enrich products from supplier feed)

## Breaking Changes Queue
(none pending)

## Key Endpoint Groups

### Auth (no tenant context, rate limited)
```
POST /v1/auth/register     {tenant_name, tenant_slug, name, email, password, invite_token?, license_token?, checkout_session_id?} → {access_token, refresh_token_cookie, user, tenant}
                           registration_mode=closed|disabled or invalid runtime mode → 403 {"error":"registration is disabled"}
POST /v1/auth/login        {tenant_slug, email, password} → {access_token, user, tenant} | {requires_2fa, two_fa_token}
POST /v1/auth/login/2fa    {two_fa_token, code} → {access_token, user, tenant}
POST /v1/auth/refresh      (cookie: refresh_token) → {access_token}
POST /v1/auth/logout       → 204
POST /v1/auth/ws-ticket    (JWT required) → {ticket}  [CSRF exempt]
```

### Orders (tenant-scoped, requires auth)
```
GET    /v1/orders                    ?status=&page=&limit=&search=&sort= → {items[], total, page, limit}
POST   /v1/orders                    {items, shipping_address, ...} → Order
GET    /v1/orders/:id                → Order (with items, shipments, returns)
PUT    /v1/orders/:id                {fields to update} → Order
DELETE /v1/orders/:id                → 204
PATCH  /v1/orders/:id/status         {status} → Order
POST   /v1/orders/bulk-status        {order_ids[], status} → {updated, errors[]}
POST   /v1/orders/export             {format, filters} → CSV/XLSX
POST   /v1/orders/import             multipart/form-data → {imported, errors[]}
POST   /v1/orders/:id/duplicate      → Order
POST   /v1/orders/:id/merge          {target_order_id} → OrderGroup
POST   /v1/orders/:id/split          {items_to_split[]} → OrderGroup
```

### Products (tenant-scoped)
```
GET    /v1/products                  ?search=&category=&page=&limit= → paginated
POST   /v1/products                  {sku, name, price, ...} → Product
GET    /v1/products/:id              → Product (with variants, listings, stock)
PUT    /v1/products/:id              → Product
DELETE /v1/products/:id              → 204
POST   /v1/products/import           multipart/form-data → {imported, errors[]}
POST   /v1/products/export           → CSV
```

### Shipments (tenant-scoped)
```
GET    /v1/shipments                 → paginated
POST   /v1/shipments                 {order_id, carrier, ...} → Shipment
GET    /v1/shipments/:id             → Shipment
POST   /v1/shipments/:id/label       → {label_url, label_base64}
POST   /v1/shipments/:id/dispatch    → Shipment (dispatch to carrier)
POST   /v1/shipments/batch-labels    {shipment_ids[]} → {labels[]}
```

### Integrations (tenant-scoped)
```
GET    /v1/integrations              → Integration[]
POST   /v1/integrations              {provider, label, credentials, settings} → Integration
GET    /v1/integrations/:id          → Integration (credentials redacted)
PUT    /v1/integrations/:id          → Integration
DELETE /v1/integrations/:id          → 204
POST   /v1/integrations/allegro/import-offers → {created, linked, skipped, errors[]}
```

### Message Templates (tenant-scoped, admin-guarded writes)
```
GET    /v1/message-templates         ?channel=&enabled= → paginated
POST   /v1/message-templates         {name, channel, subject, body, enabled} → MessageTemplate
GET    /v1/message-templates/:id     → MessageTemplate
PUT    /v1/message-templates/:id     → MessageTemplate
DELETE /v1/message-templates/:id     → 204
```

### Stock Sync (tenant-scoped)
```
POST   /v1/stock-sync/push/channel/:channel_id → {status: "ok"}
```

### Billing (public, no JWT, rate limited)
```
GET    /v1/billing/plans                    → PlanInfo[] (name, features, limits, price — no Stripe IDs)
POST   /v1/billing/checkout                 {plan_id, interval: "month"|"year"} → {checkout_url, session_id}
GET    /v1/billing/checkout/{session_id}    → {plan, interval, email, status, limits}
```
Note: Disabled when STRIPE_SECRET_KEY not set. Plans configured via BILLING_PLANS env var (JSON).

### Onboarding (tenant-scoped, requires auth)
```
GET    /v1/onboarding/status             → {completed, current_step, completed_steps[], skipped_steps[], completed_at}
PUT    /v1/onboarding/step/{step}        {action: "completed"|"skipped"} → OnboardingSettings (idempotent)
POST   /v1/onboarding/complete           → {completed: true, completed_at}
```
Note: All state stored in tenants.settings JSONB. Backward compatible — existing tenants default to completed=true or dismissed=true.

### Public (no auth, rate limited)
```
POST   /v1/public/returns/submit     {order_id, items[], reason, email} → Return
GET    /v1/public/returns/:token     → ReturnStatus
```

### Webhooks Incoming
```
POST   /v1/webhooks/:provider/:tenant_id   → 200 (generic webhook; HMAC required for known providers) | 422 when provider secret is not configured
POST   /v1/webhooks/allegro                 → 200 (HMAC verified)
POST   /v1/webhooks/inpost                  → 200 (HMAC-SHA256 verified)
POST   /v1/webhooks/stripe                  → 200 (Stripe-Signature verified)
```

### WebSocket
```
GET    /v1/ws?ticket=xxx             → WebSocket upgrade (ticket from /v1/auth/ws-ticket)
```
Note: JWT query param auth removed in PR #38. Ticket-only.

## Response Conventions
- Success: `200 OK` (read), `201 Created` (create), `204 No Content` (delete)
- Error: `{"error": "human readable message"}`
- Pagination: `{"items": [...], "total": N, "page": N, "limit": N}`
- All timestamps: ISO 8601 (UTC)
- All IDs: UUID v4

## CSRF Protection
Mutation requests (POST/PUT/PATCH/DELETE) require:
- `csrf_token` cookie (set automatically on first GET request)
- `X-CSRF-Token` header with value matching the cookie

**Exempt paths:** `/v1/auth/login`, `/v1/auth/register`, `/v1/auth/refresh`, `/v1/auth/ws-ticket`, `/v1/public/*`, `/v1/webhooks/*`, `/health`, `/metrics`
