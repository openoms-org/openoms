# Domain Model — OpenOMS

## Core Entities

### Tenant
- Company account (multi-tenant isolation via RLS)
- Fields: name, slug (URL-friendly), plan (standard/plus/pro), settings (JSONB)
- Secret fields stored in settings are encrypted at field level before DB write: email.smtp_pass, sms.api_token, ksef.token, invoicing.credentials, and webhooks.endpoints[].secret
- One tenant = one e-commerce business

### User
- Belongs to one tenant
- Fields: email, name, role_id, password_hash (bcrypt 12), totp_secret, totp_enabled
- RBAC: Role → permissions[] (e.g., "orders:read", "orders:write", "settings:manage")

### Order
- Central entity — most connected in the system
- Fields: status, items (JSONB array), shipping/billing address (JSONB), total_amount, tags[], custom_fields, priority, source (manual/allegro/amazon/woocommerce)
- Imported external orders are unique per `(tenant_id, source, external_id)` when `external_id` is non-empty; marketplace and BaseLinker import paths use atomic insert-or-skip semantics to avoid duplicate side effects.
- State machine: see below
- Relations: → shipments (1:N), → returns (1:N), → invoices (1:N), → audit_log

### Product
- SKU-based catalog
- Fields: sku, ean, name, price, stock_quantity, images (JSONB), metadata (JSONB)
- Relations: → variants (1:N), → listings (1:N per marketplace), → bundles (N:M), → warehouse_stock (1:N per warehouse)

### Shipment
- Physical package sent to customer
- Fields: carrier, tracking_number, label_url, status, warehouse_id
- Carriers: InPost, DHL, DPD, GLS, UPS, FedEx, Poczta Polska, Orlen Paczka

### Return (RMA)
- Customer return request
- Fields: status, reason, items (JSONB), refund_amount, return_token (public access), customer_email
- Public form: customer submits via token-based URL (no auth required, rate limited 30/min)

### Integration
- Third-party connection (marketplace, carrier, invoicing)
- Fields: provider, credentials (JSONB, AES-256-GCM encrypted), settings, status
- Providers: allegro, amazon, woocommerce, shopify, inpost, dhl, fakturownia, ksef, etc.

### Invoice
- Financial document
- Fields: provider (fakturownia/ksef), external_number, ksef_number, ksef_status, pdf_url
- KSeF statuses: pending → sent → accepted/rejected

### Warehouse
- Physical location for stock
- Fields: name, address, is_default
- Relations: → warehouse_stock (product quantities per warehouse), → warehouse_documents (PZ/WZ/MM)

### BillingCustomer
- Links tenant to Stripe customer
- Fields: tenant_id (UNIQUE), stripe_customer_id (UNIQUE)
- 1:1 with Tenant

### BillingSubscription
- Stripe subscription state
- Fields: tenant_id, stripe_subscription_id (UNIQUE), plan, billing_interval (month/year), status (incomplete/incomplete_expired/trialing/active/past_due/canceled/unpaid/paused), trial_end, current_period_start, current_period_end, canceled_at
- Status machine follows Stripe subscription states; trialing/active allow normal API access, past_due/unpaid/incomplete/canceled/paused/incomplete_expired allow reads but block mutations, and suspended blocks authenticated API access via the tenant plan guard.

### BillingCheckoutSession
- Pre-registration checkout state
- Fields: stripe_session_id (UNIQUE), plan, billing_interval, email, status (pending/completed/registered), tenant_id (set after registration)
- Anti-replay: status transitions are atomic (pending→completed→registered)

### OnboardingSettings
- Stored in tenants.settings JSONB (not a separate table)
- Fields: completed (bool), current_step (int), completed_steps (int[]), skipped_steps (int[]), completed_at (timestamp), dismissed (bool)
- 4 steps: Company details, Warehouse, Integration, Team invite
- Backward compatible: existing tenants default to completed=true

## Order State Machine

```
new → confirmed → processing → ready_to_ship → shipped → in_transit
  → out_for_delivery → delivered → completed

Any state → cancelled
cancelled → refunded (terminal)
completed → on_hold → refunded (terminal)
```

### State Transition Side Effects
Each status change triggers ALL of:
1. Audit log entry (who, what, when, IP)
2. Webhook dispatch to subscribers (HMAC-SHA256 signed)
3. Email notification to customer (if enabled)
4. SMS notification to customer (if enabled)
5. Automation rule evaluation (matching triggers)
6. Delayed action scheduling (if automation has delay)
7. WebSocket broadcast to connected dashboard users

## Multi-Tenant Isolation

```sql
-- Set tenant context per transaction
SELECT set_config('app.current_tenant_id', $1::text, true);

-- RLS policy on every tenant-scoped table
CREATE POLICY xxx_tenant ON xxx
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
```

### Bypass RLS (SECURITY DEFINER functions only)
- `PUBLIC EXECUTE` is revoked from bypass functions; app roles receive explicit grants, and CI checks migrated databases for regressions.
- `find_tenant_by_slug(slug)` — Login flow
- `find_user_for_auth(email, tenant_id)` — Login flow
- `find_order_tenant_id(order_id)` — Public return form
- `find_return_by_token(token)` — Public return status
- `create_checkout_session(...)` — Pre-registration billing checkout (bypass RLS)
- `complete_checkout_session(...)` — Mark checkout session completed (bypass RLS)
- `get_checkout_session(...)` — Get checkout session status (bypass RLS)
- `claim_checkout_session(...)` — Claim session for tenant (bypass RLS)
- `validate_license_token(...)` — Validate license token (bypass RLS)

## Business Rules

- Stock reserved on order creation, released on cancellation
- Allegro orders auto-confirm after import (configurable per integration)
- Invoice auto-generated on status → shipped (if invoicing enabled)
- Low stock alert: triggers `product.stock_low` automation event when quantity < min_stock
- Order merging: combine multiple orders to same customer into one shipment
- Order splitting: split one order into multiple shipments (e.g., different warehouses)
