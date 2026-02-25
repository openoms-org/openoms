# API Contracts
Version: 4 (bump after every endpoint change)
Updated: 2026-02-25

## Recently Changed
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
POST /v1/auth/register     {tenant_name, tenant_slug, name, email, password} → {access_token, refresh_token_cookie, user, tenant}
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

### Public (no auth, rate limited)
```
POST   /v1/public/returns/submit     {order_id, items[], reason, email} → Return
GET    /v1/public/returns/:token     → ReturnStatus
```

### Webhooks Incoming
```
POST   /v1/webhooks/:provider/:tenant_id   → 200 (generic webhook)
POST   /v1/webhooks/allegro                 → 200 (HMAC verified)
POST   /v1/webhooks/inpost                  → 200 (HMAC-SHA256 verified)
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
