# Backend Workers, Middleware, Router & SDKs Audit

---

## WORKER SYSTEM (16 Workers)

### Worker Manager (`manager.go`)
- Interface: `Worker { Name() string, Interval() time.Duration, Run(ctx) error }`
- guardedRun() prevents concurrent execution via atomic.Bool
- safeRun() catches panics + logs to Sentry

### Order Polling Workers (MarketplaceOrderPoller pattern)
| Worker | Name | Interval | Provider |
|--------|------|----------|----------|
| AllegroOrderPoller | `allegro_order_poller` | 45s | Allegro REST API |
| AmazonOrderPoller | `amazon_order_poller` | 2min | Amazon SP-API |
| WooCommerceOrderPoller | `woocommerce_order_poller` | 45s | WooCommerce REST API |
| StoreOrderPoller | varies | 45s | Shoper, PrestaShop, Shopify |
| ErliOrderPoller | `erli_order_poller` | 45s | Erli REST API |

All use: Dedupe by external_id, create orders, auto-create shipments & labels if settings enabled.

### Sync Workers
| Worker | Name | Interval | Purpose |
|--------|------|----------|---------|
| StockSyncWorker | `stock_sync` | 5min | Push stock to marketplace listings |
| PriceSyncWorker | `price_sync` | 5min | Push prices to marketplace listings |
| SupplierSyncWorker | `supplier-sync` | 1min | Sync supplier catalogs (XML/CSV/IOF/API) |
| ListingSyncWorker | `listing_sync` | 5min | Full marketplace catalog sync |

### Tracking & Status
| Worker | Name | Interval | Purpose |
|--------|------|----------|---------|
| TrackingPoller | `tracking_poller` | 10min | Update shipment tracking from carriers |
| DelayedActionWorker | `delayed_action_executor` | 30s | Execute delayed automation actions |

### Exchange & Invoice
| Worker | Name | Interval | Purpose |
|--------|------|----------|---------|
| ExchangeRateWorker | `exchange_rate_fetcher` | 24h | Fetch rates from NBP (Polish National Bank) |
| KSeFStatusWorker | `ksef_status_checker` | 5min | Check KSeF invoice status + retry errors |

### OAuth & Subscriptions
| Worker | Name | Interval | Purpose |
|--------|------|----------|---------|
| OAuthRefresher | `oauth_refresher` | 30min | Refresh expiring Allegro/Amazon tokens |
| RecurringOrderWorker | `recurring_order_processor` | 1h | Create orders from subscriptions |

### Pricing
| Worker | Name | Interval | Purpose |
|--------|------|----------|---------|
| RepricingWorker | `repricing_engine` | 15min | Apply dynamic pricing rules |

---

## MIDDLEWARE STACK (16 middlewares)

**Applied order** (router.go):
1. `RequestID` (chi) — X-Request-ID
2. `RealIP` (chi) — client IP extraction
3. `SentryMiddleware` — panic capture (conditional)
4. `MetricsCollector.Middleware()` — Prometheus HTTP metrics (conditional)
5. `SecurityHeaders` — CSP, X-Frame-Options:DENY, X-Content-Type-Options, Referrer-Policy
6. `Logging` — structured slog JSON to stdout
7. `Recoverer` (chi) — panic recovery
8. `CORS` — origin whitelist (FrontendURL), credentials
9. `CSRF` — double-submit signed cookies, exempt GET/HEAD/OPTIONS/public
10. `JWTAuth` — Bearer token validation, sets claims/tenant/user in context
11. `SentryContext` — enriches Sentry with tenant + user
12. `TenantPlanGuard` — blocks 402 for suspended, past_due blocks mutations
13. `RequireRole` — hierarchy: member < admin < owner
14. `RateLimiter` — token bucket per IP:endpoint (Redis or memory)
15. `MaxBodySize` — http.MaxBytesReader (1MB default)
16. `TokenBlacklist` — revoked token cache (composite: Redis-first + memory fallback)

### Middleware Files
- `auth.go` — JWTAuth token validation, context keys
- `context.go` — Context key constants
- `tenant.go` — TenantID extraction
- `role.go` — RequireRole() hierarchy check
- `bodylimit.go` — MaxBodySize wrapper
- `csrf.go` — CSRF validation, exempt list
- `cors.go` — CORS headers
- `security_headers.go` — CSP, X-Frame-Options, HSTS
- `logging.go` — JSON logging (supports Hijacker/Flusher for WebSocket/SSE)
- `metrics.go` — Prometheus metrics (request duration, status, method)
- `metrics_auth.go` — Bearer token auth for /metrics
- `ratelimit.go` — RateLimiter interface + factory
- `ratelimit_redis.go` — Redis Lua INCR+EXPIRE
- `plan_guard.go` — Plan status + limits from settings
- `sentry.go` — Sentry panic capture
- `sentry_context.go` — Sentry enrichment
- `token_blacklist.go` — Interface + MemoryTokenBlacklist
- `token_blacklist_redis.go` — Redis SET with TTL
- `token_blacklist_composite.go` — Redis-first + memory fallback

---

## ROUTER (router.go)

### Public Routes (no JWT)
```
GET  /health
GET  /metrics                          [Bearer token]
GET  /v1/openapi.yaml
GET  /v1/docs
GET  /v1/config/public                 [60 req/min]
GET  /v1/billing/plans                 [60 req/min]
POST /v1/billing/checkout              [10 req/min]
GET  /v1/billing/checkout/{session_id} [30 req/min]
```

### Webhook Routes (signature-verified, no JWT)
```
POST /v1/webhooks/{provider}/{tenant_id}  [120 req/min]
POST /v1/webhooks/allegro                 [120 req/min, HMAC]
POST /v1/webhooks/inpost                  [120 req/min, HMAC-SHA256]
POST /v1/webhooks/stripe                  [120 req/min, Stripe-Signature]
```

### Public Customer Routes (no JWT)
```
POST /v1/public/returns                [30 req/min]
GET  /v1/public/returns/{token}
GET  /v1/public/returns/{token}/status
GET  /v1/tracking/{tenant_slug}/{order_id}  [10 req/min]
```

### Auth Routes (rate-limited, no JWT)
```
POST /v1/auth/register     [10 req/min]
POST /v1/auth/login        [10 req/min]
POST /v1/auth/2fa/login    [10 req/min]
POST /v1/auth/refresh      [60 req/min]
POST /v1/auth/logout       [60 req/min]
```

### Supplier Portal (token auth, no JWT)
```
/v1/supplier-portal/*      [30 req/min]
```

### Product Feeds (no JWT)
```
GET /v1/feeds/ceneo/{tenant_id}/{token}   [60 req/h]
GET /v1/feeds/google/{tenant_id}/{token}  [60 req/h]
```

### WebSocket
```
GET /v1/ws   [30 req/min, ticket-based auth]
```

### Authenticated Routes (JWT + PlanGuard)
All under `/v1` with 1MB body limit.

**Any Auth:**
- Orders CRUD + export, bulk-status, merge, split, duplicate, import, pack, print
- Products CRUD + export, import (CSV + BaseLinker), variants, bundles, listings
- Shipments CRUD + batch-labels, dispatch, label, tracking
- Returns CRUD + status transitions, print
- Customers CRUD + import, orders
- Invoices CRUD + PDF, KSeF send/status/UPO, bulk-send
- Warehouses CRUD + stock
- Stats (dashboard, top products, revenue, trends, payment methods)
- Pick/pack sessions
- Recurring orders
- Shipping rates comparison
- AI (categorize, describe, improve, translate, bulk-categorize)
- InPost points + geowidget token
- Barcode lookup

**Admin Only:**
- Settings (all subsections)
- Users + invitations
- Roles + permissions
- Integrations CRUD + Allegro (~80 sub-endpoints)
- Suppliers + BTP wizard + portal
- Automation rules + delayed actions + workflows
- Warehouse documents (PZ/WZ/MM)
- Stocktakes
- Stock sync channels
- Listing sync configs
- Repricing rules
- Price lists
- Exchange rates
- Categories
- Purchase orders (create/update/delete)
- Dropship orders (create/update)
- Segments + loyalty
- Marketing
- Audit log
- Webhook deliveries
- Sync jobs
- Onboarding management
- Billing subscription

---

## SDK PACKAGES (27 Go modules)

### Marketplace SDKs (11)
| Package | Provider | Status |
|---------|----------|--------|
| `allegro-go-sdk` | Allegro REST API | Verified (OAuth2) |
| `amazon-sp-sdk` | Amazon SP-API | Implemented |
| `woocommerce-go-sdk` | WooCommerce REST API v3 | Implemented |
| `prestashop-go-sdk` | PrestaShop Web Service | Implemented |
| `shoper-go-sdk` | Shoper REST API | Implemented |
| `shopify-go-sdk` | Shopify Admin REST API | Implemented |
| `ebay-go-sdk` | eBay RESTful API v1 | Implemented |
| `kaufland-go-sdk` | Kaufland Seller API v2 | Implemented |
| `olx-go-sdk` | OLX Partner API | Implemented |
| `mirakl-go-sdk` | Mirakl REST API (Empik) | Implemented |
| `erli-go-sdk` | Erli.pl REST API | Verified (sandbox tests) |

### Carrier SDKs (10)
| Package | Provider | Status |
|---------|----------|--------|
| `inpost-go-sdk` | InPost ShipX API | Verified |
| `dhl-go-sdk` | DHL24 SOAP API | In development |
| `dpd-go-sdk` | DPD Poland REST API | Implemented |
| `gls-go-sdk` | GLS Poland REST API | Implemented |
| `ups-go-sdk` | UPS REST API (OAuth2) | In development |
| `fedex-go-sdk` | FedEx REST API v1 | In development |
| `orlen-paczka-go-sdk` | Orlen Paczka REST API | In development |
| `poczta-polska-go-sdk` | Poczta Polska eNadawca | In development |
| `smsapi-go-sdk` | SMSAPI.pl SMS gateway | Implemented |

### Invoice/Accounting SDKs (4)
| Package | Provider | Status |
|---------|----------|--------|
| `fakturownia-go-sdk` | Fakturownia API | Implemented |
| `infakt-go-sdk` | inFakt API | Implemented |
| `ksef-go-sdk` | KSeF (Polish e-invoicing) | Verified |
| `wfirma-go-sdk` | wFirma API | Implemented |

### B2B/Supplier SDKs (1)
| Package | Provider | Status |
|---------|----------|--------|
| `btp-go-sdk` | BTP.pro Business Platform | Verified (Basic Auth) |

### Utility Packages (2)
| Package | Purpose |
|---------|---------|
| `iof-parser` | IOF XML feed parser (Polish wholesalers) |
| `order-engine` | Order state machine + domain events |
