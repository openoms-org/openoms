# Security Audit: Rate Limiting & RBAC

**Audit Date:** 2026-03-03

---

## 1. RATE LIMITING AUDIT

### Routes WITH explicit rate limits

| Route | Limit | File:Line |
|-------|-------|-----------|
| `POST /v1/auth/register` | 10/min/IP | `router.go:178` |
| `POST /v1/auth/login` | 10/min/IP | `router.go:179` |
| `POST /v1/auth/2fa/login` | 10/min/IP | `router.go:180` |
| `POST /v1/auth/refresh` | 60/min/IP | `router.go:183` |
| `POST /v1/auth/logout` | 60/min/IP | `router.go:184` |
| `GET /v1/config/public` | 60/min/IP | `router.go:202` |
| `GET /v1/billing/plans` | 60/min/IP | `router.go:208` |
| `POST /v1/billing/checkout` | 10/min/IP | `router.go:210` |
| `GET /v1/billing/checkout/{id}` | 30/min/IP | `router.go:212` |
| `POST /v1/webhooks/{provider}/{id}` | 120/min/IP | `router.go:217` |
| `POST /v1/webhooks/allegro` | 120/min/IP | `router.go:222` |
| `POST /v1/webhooks/inpost` | 120/min/IP | `router.go:228` |
| `POST /v1/webhooks/stripe` | 120/min/IP | `router.go:234` |
| `/v1/public/returns/*` (all) | 30/min/IP | `router.go:240` |
| `GET /v1/tracking/{slug}/{id}` | 10/min/IP | `router.go:248` |
| `/v1/supplier-portal/*` (all) | 30/min/IP | `router.go:254` |
| `/v1/feeds/*` (all) | 60/hour/IP | `router.go:268` |
| `GET /v1/ws` | 30/min/IP | `router.go:275` |
| `GET /v1/barcode/{code}` | 120/min/IP | `router.go:927` |

### Findings

**WARNING [W-RL-1]: Rate limiter key collision**

File: `apps/api-server/internal/middleware/ratelimit.go:29`

The rate limiter key is `fmt.Sprintf("%s:%d", ip, maxRequests)`. Endpoints with the same `maxRequests` value share a counter:
- `POST /v1/auth/register` (10/min) and `POST /v1/auth/login` (10/min) and `POST /v1/auth/2fa/login` (10/min) share the same counter
- An attacker making 10 login attempts would also exhaust the registration limit for that IP

**WARNING [W-RL-2]: No rate limits on any authenticated endpoint (except barcode lookup)**

The entire authenticated route group (`/v1` at line 279) has zero rate limiting. Affected endpoints:

- **AI endpoints** (`/v1/ai/*`) - lines 986-992: `categorize`, `describe`, `bulk-categorize`, `improve`, `translate` -- calls OpenAI APIs, incurs cost per request
- **Bulk operations**: `bulk-status`, `import` (orders, products, customers, suppliers), `bulk-delete`
- **Export endpoints**: orders export, products export, VAT OSS report
- **File upload**: `POST /v1/uploads`, `POST /v1/images/remove-background`
- **External API proxies**: exchange rate fetch, shipping rates, Allegro import, stock sync push

**WARNING [W-RL-3]: Rate limits are per-IP only, not per-user**

All rate limits use IP as the sole key (`ratelimit.go:22-25`). Behind a corporate NAT or shared proxy, all users share a single counter.

**WARNING [W-RL-4]: Rate limiter fails open on error**

File: `apps/api-server/internal/middleware/ratelimit.go:31-34`

If the rate limiter (Redis) returns an error, the request is allowed through. Logged as `Warn` -- should be `Error` with monitoring.

---

## 2. RBAC AUDIT

### Routes with RequireRole("admin") -- Properly Gated

All properly gated as admin-only:
- Onboarding step update/complete (`router.go:316`)
- Settings -- all (`router.go:324`)
- Audit log, webhook deliveries (`router.go:368`)
- Sync jobs (`router.go:379`)
- User management CRUD (`router.go:389`)
- Invitations (`router.go:399`)
- Product listings (`router.go:523`)
- Integrations -- all (`router.go:546`)
- Suppliers -- all (`router.go:699`)
- Categories (`router.go:743`)
- Purchase orders write (`router.go:760`)
- Dropship orders write (`router.go:777`)
- Warehouses (`router.go:792`)
- Automation rules (`router.go:861`)
- Workflows (`router.go:876`)
- Forecast config update (`router.go:901`)
- VAT OSS (`router.go:915`)
- Price lists (`router.go:932`)
- Message templates write (`router.go:951`)
- Warehouse documents (`router.go:961`)
- Stocktakes (`router.go:972`)
- Marketing (`router.go:996`)
- Exchange rates (`router.go:1007`)
- Roles RBAC (`router.go:1019`)
- Repricing (`router.go:1049`)
- Stock sync (`router.go:1064`)
- Listing sync (`router.go:1084`)
- Reconciliation (`router.go:1099`)

### Routes accessible to ALL authenticated users (including "member" role)

**WARNING [W-RBAC-1]: Orders -- full CRUD including DELETE accessible to any member**

File: `router.go:407-431`

Any authenticated user (including "member" role) can:
- **Delete orders** (`DELETE /v1/orders/{id}`) -- line 417
- Merge orders (`POST /v1/orders/merge`) -- line 412
- Split orders (`POST /v1/orders/{id}/split`) -- line 420
- Duplicate orders (`POST /v1/orders/{id}/duplicate`) -- line 419
- Bulk status transitions (`POST /v1/orders/bulk-status`) -- line 411
- Import orders from CSV (`POST /v1/orders/import`) -- line 414

**WARNING [W-RBAC-2]: Invoices -- cancel and KSeF send accessible to any member**

File: `router.go:440-452`

Any authenticated user can:
- **Cancel invoices** (`DELETE /v1/invoices/{id}`) -- line 447
- Send invoices to KSeF (`POST /v1/invoices/{id}/ksef/send`) -- line 448
- **Bulk send to KSeF** (`POST /v1/invoices/ksef/bulk-send`) -- line 443

KSeF is the Polish national e-invoicing system. Legal and tax implications.

**WARNING [W-RBAC-3]: Shipments -- full CRUD including DELETE accessible to any member**

File: `router.go:455-466`

- **Delete shipments** (`DELETE /v1/shipments/{id}`) -- line 462
- Generate labels -- calls carrier APIs, incurs cost
- Create batch labels, dispatch orders

**WARNING [W-RBAC-4]: Returns -- DELETE accessible to any member**

File: `router.go:469-479`

**WARNING [W-RBAC-5]: Products -- full CRUD including DELETE and import accessible to any member**

File: `router.go:482-542`

**WARNING [W-RBAC-6]: Customers -- full CRUD including DELETE and import accessible to any member**

File: `router.go:803-812`

GDPR implications for customer data deletion.

**WARNING [W-RBAC-7]: Customer segments and loyalty programs -- full CRUD accessible to any member**

File: `router.go:816-845`

Any member can award/redeem loyalty points.

**WARNING [W-RBAC-8]: Recurring orders -- full CRUD accessible to any member**

File: `router.go:848-857`

**WARNING [W-RBAC-9]: AI endpoints -- no RBAC, no rate limit**

File: `router.go:986-992`

Any authenticated user can call all AI endpoints including `bulk-categorize` which can be expensive.

**WARNING [W-RBAC-10]: Pick & Pack sessions -- full CRUD accessible to any member**

File: `router.go:1033-1042`

Potentially intentional for warehouse workers.

**INFO [RBAC-OK-1]: RequirePermission middleware exists but is not used in the router**

File: `apps/api-server/internal/middleware/role.go:55`

The fine-grained `RequirePermission` middleware is implemented but not used anywhere in `router.go`. All authorization is done via the coarser `RequireRole("admin")`. The RBAC roles/permissions management UI creates roles with permissions that are never enforced.

---

## 3. PUBLIC ENDPOINT DATA LEAKAGE AUDIT

**OK [PUB-OK-1]: POST /v1/public/returns -- no tenant data leaked**

- Requires `order_id` (UUID) + `email` match
- Uses SECURITY DEFINER function
- No tenant_id exposed in response

**OK [PUB-OK-2]: GET /v1/public/returns/{token} -- tokens are not enumerable**

- Token is 128-bit entropy
- Returns 404 for unknown tokens
- Rate limited at 30/min/IP

**WARNING [W-PUB-1]: GET /v1/public/returns/{token} -- exposes customer_email and refund_amount**

File: `public_return_handler.go:55-66`

The `GetByToken` endpoint returns `order_id`, `customer_email`, and `refund_amount`. While the token is unguessable, anyone with the token can see these fields. Consider returning the minimal set like `GetStatusByToken`.

**OK [PUB-OK-3]: GET /v1/tracking/{tenant_slug}/{order_id} -- requires email verification**

- Requires `email` query parameter that must match order's `customer_email`
- Rate limited at 10/min/IP (very strict)

**OK [PUB-OK-4]: Supplier portal -- proper token authentication**

- SHA-256 hashed, constant-time compared, has expiry
- Portal can be disabled per-supplier
- Rate limited at 30/min/IP

**OK [PUB-OK-5]: Product feeds (Ceneo, Google) -- no cost/margin data exposed**

- Token validation uses `subtle.ConstantTimeCompare`
- Feed tokens are 128-bit entropy
- Rate limited at 60/hour/IP
- Cached for 15 minutes

---

## 4. AUTH BYPASS AUDIT

**OK [AUTH-OK-1]: No auth bypass in JWT-protected group**

JWT middleware applied at group level. All routes inside inherit it.

**OK [AUTH-OK-2]: WebSocket ticket is single-use and expires in 30 seconds**

256-bit entropy, atomic consume, rate limited.

**OK [AUTH-OK-3]: Refresh tokens cannot be reused after logout**

Access token blacklisted, refresh family deleted, lastLogoutAt check.

**WARNING [W-AUTH-1]: Refresh token rotation fails open when store lookup fails**

File: `apps/api-server/internal/service/auth_service.go:616-625`

When `refreshStore.GetToken()` returns error or token not found (e.g., after server restart with in-memory store), rotation detection is completely bypassed. JWT signature alone is validated.

**OK [AUTH-OK-4]: Non-access tokens rejected by JWT middleware**

**OK [AUTH-OK-5]: Uploaded files are tenant-scoped**

---

## Recommendations (priority order)

1. **Fix rate limiter key to include route path** (W-RL-1): Change key to `fmt.Sprintf("rl:%s:%s:%d", ip, r.URL.Path, maxRequests)` or assign each route a unique name.

2. **Add per-user rate limits for authenticated AI endpoints** (W-RL-2, W-RBAC-9): Add `RequireRole("admin")` to AI routes and a rate limit of ~20/min per tenant.

3. **Gate destructive operations behind admin role** (W-RBAC-1 through W-RBAC-8): For DELETE operations on orders, invoices, shipments, returns, products, and customers, add `RequireRole("admin")` or use the existing `RequirePermission` infrastructure.

4. **Consider per-user/tenant rate limiting for authenticated routes** (W-RL-3): Add a global per-tenant rate limit (e.g., 600/min) on the authenticated group.

5. **Log rate limiter failopen at Error level and monitor** (W-RL-4).

6. **Reduce data in GetByToken response** (W-PUB-1).
