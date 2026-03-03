# Test Coverage Gap Analysis

**Analysis Date:** 2026-03-03

---

## Overall Summary

| Component | Total | Tested | Coverage | Risk |
|-----------|-------|--------|----------|------|
| **Middleware** | 19 | 20 | 100% | LOW |
| **Automation** | 4 | 3 | 75% | LOW |
| **Worker** | 20 | 12 | 60% | HIGH |
| **Service** | 66 | 30 | 45.5% | HIGH |
| **Handler** | 83 | 29 | 34.9% | CRITICAL |
| **Integration** | 36 | 7 | 19.4% | CRITICAL |
| **Repository** | 46 | 1 | 2.2% | CRITICAL |
| **TOTAL** | **274** | **102** | **37.2%** | **CRITICAL** |

---

## 1. Repository Layer -- 2.2% Coverage (CRITICAL)

Only 1 of 46 repository files has tests (`license_repository_test.go`).

### Critical Untested Repositories

| Repository | Notes |
|-----------|-------|
| `order_repository.go` | Core business -- orders |
| `product_repository.go` | Catalog data |
| `user_repository.go` | Authentication |
| `tenant_repository.go` | Multi-tenancy isolation |
| `billing_repository.go` | Stripe billing data |
| `invoice_repository.go` | KSeF, invoicing |
| `shipment_repository.go` | Shipping |
| `return_repository.go` | Returns/RMA |
| `integration_repository.go` | Encrypted credentials |
| `payment_repository.go` | Payment transactions |
| `role_repository.go` | RBAC roles |

---

## 2. Handler Layer -- 34.9% Coverage

54 of 83 handlers untested.

### Critical Untested Handlers

| Handler | Notes |
|---------|-------|
| `ksef_handler.go` | Polish e-invoice endpoints |
| `tracking_handler.go` | Shipment tracking |
| `ws_handler.go` | WebSocket connections |
| `allegro_catalog_handler.go` | Allegro catalog sync |
| `public_return_handler.go` | Public return form (no auth) |
| `role_handler.go` | RBAC role management |
| `stats_handler.go` | Dashboard KPIs |
| `store_auth_handler.go` | Store authentication |

---

## 3. Service Layer -- 45.5% Coverage

36 of 66 services untested.

### Critical Untested Services

| Service | Notes |
|---------|-------|
| `allegro_sync_service.go` | Allegro order sync |
| `tracking_service.go` | Carrier tracking updates |
| `listing_sync_service.go` | Marketplace listing sync |
| `invitation_service.go` | User invitations |
| `role_service.go` | RBAC management |
| `ai_service.go` | OpenAI integration |

### Well-Tested Services

Auth, checkout, order, shipment, invoice, KSeF, license, product, return, webhook, stock sync -- all have tests.

---

## 4. Integration Providers -- 19.4% Coverage

29 of 36 providers untested.

### Critical Untested Providers

| Provider | Type | Notes |
|----------|------|-------|
| `amazon/provider.go` | Marketplace | SP-API |
| `ebay/provider.go` | Marketplace | API |
| `shopify/provider.go` | Marketplace | API |
| `woocommerce/provider.go` | Marketplace | API |
| `olx/provider.go` | Marketplace | API |
| `carriers/fedex.go` | Carrier | International |
| `carriers/ups.go` | Carrier | API |
| `carriers/poczta_polska.go` | Carrier | Polish National Post |
| `accounting/infakt_provider.go` | Accounting | Invoicing |
| `accounting/wfirma_provider.go` | Accounting | Accounting |
| `fakturownia/provider.go` | Invoicing | API |

### Tested Providers (7)

DHL, DPD, GLS, InPost (carriers), Allegro (marketplace)

---

## 5. Worker Layer -- 60% Coverage

8 of 20 workers untested.

### Critical Untested Workers

| Worker | Notes |
|--------|-------|
| `ksef_status_worker.go` | Tax compliance |
| `oauth_refresher.go` | Token refresh |
| `listing_sync_worker.go` | Marketplace sync |
| `delayed_action_worker.go` | Automation actions |
| `supplier_sync_worker.go` | Supplier catalog |

---

## 6. Middleware -- 100% Coverage

All 19 middleware files have comprehensive tests. Includes: JWT auth, RBAC, CSRF, rate limiting, tenant isolation, token blacklist, security headers.

---

## 7. Automation Engine -- 75% Coverage

Core logic (engine, conditions, actions) fully tested. Only `types.go` (type definitions) lacks tests.

---

## Priority Recommendations

### P0 -- Revenue/Compliance Impact (First Sprint)

1. Repository layer: `order_repository`, `billing_repository`, `invoice_repository`, `tenant_repository`, `user_repository`
2. Handlers: `ksef_handler`, `tracking_handler`, `role_handler`, `public_return_handler`
3. Workers: `ksef_status_worker`, `oauth_refresher`
4. Integration: `allegro/provider`, carrier providers

### P1 -- Data Integrity (Second Sprint)

1. Repository layer: `product_repository`, `shipment_repository`, `return_repository`, `payment_repository`
2. Services: `allegro_sync_service`, `tracking_service`, `listing_sync_service`
3. Workers: `listing_sync_worker`, `delayed_action_worker`

### P2 -- Feature Completeness

1. Remaining repositories
2. Remaining handlers
3. Remaining integration providers
