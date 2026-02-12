# OpenOMS - Kompletna Dokumentacja Systemu

## Spis treści

1. [Podsumowanie](#1-podsumowanie)
2. [Architektura](#2-architektura)
3. [Stos technologiczny](#3-stos-technologiczny)
4. [Baza danych](#4-baza-danych)
5. [Backend API](#5-backend-api)
6. [Frontend Dashboard](#6-frontend-dashboard)
7. [Pakiety SDK](#7-pakiety-sdk)
8. [Bezpieczeństwo](#8-bezpieczeństwo)
9. [Kluczowe flow](#9-kluczowe-flow)
10. [Integracje](#10-integracje)
11. [Background Workers](#11-background-workers)
12. [Automatyzacja](#12-automatyzacja)
13. [Konfiguracja](#13-konfiguracja)
14. [Statystyki projektu](#14-statystyki-projektu)

---

## 1. Podsumowanie

**OpenOMS** to open-source'owy system zarządzania zamówieniami (OMS) dla polskiego e-commerce. Jest to aplikacja multi-tenant SaaS z izolacją danych na poziomie bazy danych (PostgreSQL RLS).

### Główne cechy

- **Multi-tenant** — pełna izolacja danych między firmami
- **Multi-marketplace** — Allegro, Amazon, eBay, Kaufland, OLX, WooCommerce, Empik/Mirakl, Erli
- **Multi-carrier** — InPost, DHL, DPD, GLS, UPS, Poczta Polska, Orlen Paczka, FedEx
- **Automatyzacja** — silnik reguł (trigger → warunki → akcje)
- **Fakturowanie** — integracja z Fakturownia
- **Powiadomienia** — Email (SMTP) + SMS (Twilio)
- **RBAC** — role z granularnymi uprawnieniami
- **API REST** — 150+ endpointów z OpenAPI 3.1
- **Dashboard** — Next.js 16 + React 19, 64 strony, dark mode, PWA
- **AI** — auto-kategoryzacja i opis produktów (OpenAI)

### Licencja

- `apps/` — AGPLv3 (core)
- `packages/` — MIT (SDK-i)

---

## 2. Architektura

### Diagram wysokopoziomowy

```
┌─────────────────────────────────────────────────────────────┐
│                        KLIENCI                              │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌────────────┐  │
│  │ Dashboard │  │  API     │  │ Webhook  │  │ Public     │  │
│  │ (Next.js)│  │ Consumer │  │ Sender   │  │ Return     │  │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └──────┬─────┘  │
└───────┼──────────────┼─────────────┼───────────────┼────────┘
        │              │             │               │
        ▼              ▼             ▼               ▼
┌─────────────────────────────────────────────────────────────┐
│                    API SERVER (Go)                           │
│  ┌─────────┐  ┌──────────┐  ┌───────────┐  ┌────────────┐  │
│  │  Router  │→ │Middleware│→ │ Handlers  │→ │  Services  │  │
│  │  (chi)   │  │JWT+CORS │  │ (HTTP)    │  │ (logika)   │  │
│  └─────────┘  └──────────┘  └───────────┘  └─────┬──────┘  │
│                                                    │        │
│  ┌─────────────┐  ┌──────────┐  ┌──────────────┐  │        │
│  │  Workers    │  │WebSocket │  │  Automation   │  │        │
│  │ (bg jobs)   │  │  Hub     │  │  Engine       │  │        │
│  └──────┬──────┘  └────┬─────┘  └──────┬───────┘  │        │
│         │              │               │           │        │
│         ▼              ▼               ▼           ▼        │
│  ┌─────────────────────────────────────────────────────┐    │
│  │              Repositories (Data Access)              │    │
│  └──────────────────────┬──────────────────────────────┘    │
└─────────────────────────┼───────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│                   PostgreSQL 12+                            │
│  ┌──────────┐  ┌──────────────┐  ┌───────────────────────┐ │
│  │ 29 tabel │  │  RLS Policy  │  │  SECURITY DEFINER     │ │
│  │ 38 migr. │  │ (per tenant) │  │  (auth bypass)        │ │
│  └──────────┘  └──────────────┘  └───────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

### Wzorzec warstwowy

```
HTTP Request
    │
    ▼
┌──────────┐     ┌────────────┐     ┌──────────────┐     ┌──────────┐
│ Handler  │ ──→ │  Service   │ ──→ │ Repository   │ ──→ │   DB     │
│ (waliduj │     │ (logika    │     │ (SQL query)  │     │ (pgx +   │
│  + HTTP) │     │  biznesowa)│     │              │     │  RLS)    │
└──────────┘     └────────────┘     └──────────────┘     └──────────┘
```

### Multi-tenancy

```
Request → JWT Token → TenantID extraction
    │
    ▼
database.WithTenant(ctx, pool, tenantID, func(tx) {
    SET app.current_tenant_id = $1  ← parametryzowane!
    ...query...                      ← RLS filtruje automatycznie
})
```

Każda tabela ma politykę RLS:
```sql
CREATE POLICY tenant_isolation ON orders
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
```

---

## 3. Stos technologiczny

### Backend

| Komponent | Technologia | Wersja |
|-----------|------------|--------|
| Język | Go | 1.24 |
| Router HTTP | chi/v5 | 5.x |
| Baza danych | PostgreSQL | 12+ |
| Driver DB | pgx/v5 | 5.x |
| JWT | golang-jwt/v5 + Ed25519 | 5.x |
| Logowanie | log/slog | std |
| Konfiguracja | caarlos0/env/v11 | 11.x |
| Migracje | golang-migrate | 4.x |
| WebSocket | gorilla/websocket | 1.x |
| Metryki | prometheus/client_golang | 1.x |

### Frontend

| Komponent | Technologia | Wersja |
|-----------|------------|--------|
| Framework | Next.js | 16.1.6 |
| UI Library | React | 19.2.3 |
| Komponenty | shadcn/ui + Radix UI | latest |
| Styl | Tailwind CSS | v4 |
| Walidacja | Zod | v4 |
| Formularze | react-hook-form | 7.x |
| State | Zustand | 5.x |
| Data fetching | TanStack React Query | 5.x |
| Wykresy | Recharts | 3.x |
| Ikony | Lucide React | 0.563 |
| Testy E2E | Playwright | 1.58 |
| Testy jednostkowe | Vitest + Testing Library | 4.x |

### Monorepo

```
OpenOMS/
├── apps/
│   ├── api-server/          ← Go backend (AGPLv3)
│   │   ├── cmd/server/      ← punkt wejścia
│   │   ├── internal/        ← logika aplikacji
│   │   └── migrations/      ← 38 migracji SQL
│   └── dashboard/           ← Next.js frontend (AGPLv3)
│       ├── src/app/         ← 64 strony (App Router)
│       ├── src/components/  ← 70+ komponentów
│       ├── src/hooks/       ← 41 custom hooks
│       ├── src/lib/         ← utils, API client, auth
│       └── e2e/             ← 58 testów Playwright
├── packages/                ← SDK-i (MIT)
│   ├── order-engine/        ← maszyna stanów zamówień
│   ├── allegro-go-sdk/      ← Allegro REST API
│   ├── inpost-go-sdk/       ← InPost ShipX API
│   └── ...                  ← 15+ SDK-ów
└── docs/                    ← dokumentacja
```

---

## 4. Baza danych

### Diagram ERD (uproszczony)

```
┌──────────┐       ┌──────────┐       ┌──────────┐
│ tenants  │──────<│  users   │       │  roles   │
│          │       │          │>──────│          │
└────┬─────┘       └──────────┘       └──────────┘
     │
     │  ┌──────────────┐   ┌────────────┐   ┌──────────────┐
     ├─<│   orders     │──<│ shipments  │   │   returns    │
     │  │              │   │            │   │              │
     │  │  items[]     │   │ carrier    │   │  status      │
     │  │  status      │──<│ tracking#  │   │  refund_amt  │
     │  │  tags[]      │   └────────────┘   └──────────────┘
     │  │  custom_flds │
     │  └──────┬───────┘
     │         │
     │         ├──────────────────────<┌──────────────┐
     │         │                       │  invoices    │
     │         │                       └──────────────┘
     │
     │  ┌──────────────┐   ┌────────────────┐   ┌──────────────┐
     ├─<│  products    │──<│ prod_variants  │   │ prod_bundles │
     │  │              │   └────────────────┘   └──────────────┘
     │  │  sku, ean    │──<┌────────────────┐
     │  │  images[]    │   │ prod_listings  │
     │  │  tags[]      │   └────────────────┘
     │  └──────┬───────┘
     │         │
     │         ├─────────<┌──────────────────┐
     │         │          │ warehouse_stock   │
     │         │          └────────┬─────────┘
     │         │                   │
     │  ┌──────────────┐   ┌──────┴──────┐   ┌──────────────────┐
     ├─<│  warehouses  │   │ wh_documents│──<│ wh_document_items│
     │  └──────────────┘   └─────────────┘   └──────────────────┘
     │
     │  ┌──────────────┐   ┌────────────────┐
     ├─<│ integrations │   │  sync_jobs     │
     │  │  (encrypted) │──<│  (append-only) │
     │  └──────────────┘   └────────────────┘
     │
     │  ┌──────────────┐   ┌────────────────┐
     ├─<│  customers   │   │  suppliers     │──<┌─────────────────┐
     │  │  tags[]      │   │                │   │supplier_products│
     │  └──────────────┘   └────────────────┘   └─────────────────┘
     │
     │  ┌──────────────┐   ┌────────────────┐   ┌──────────────┐
     ├─<│ automation   │──<│ auto_rule_logs │   │ price_lists  │
     │  │ _rules       │   └────────────────┘   │              │──<┌──────────────┐
     │  └──────────────┘                        └──────────────┘   │ pl_items     │
     │                                                              └──────────────┘
     │  ┌──────────────┐   ┌────────────────┐   ┌──────────────┐
     ├─<│  audit_log   │   │ webhook_events │   │ wh_deliveries│
     │  └──────────────┘   └────────────────┘   └──────────────┘
     │
     └─<┌──────────────┐   ┌────────────────┐
        │ order_groups │   │exchange_rates  │
        └──────────────┘   └────────────────┘
```

### Wszystkie tabele (29)

| Tabela | Cel | Kluczowe kolumny |
|--------|-----|-----------------|
| `tenants` | Konta firm | name, slug, plan, settings JSONB |
| `users` | Użytkownicy | email, name, role, role_id, password_hash |
| `roles` | Role RBAC | name, permissions TEXT[], is_system |
| `orders` | Zamówienia | status, items JSONB, total_amount, tags[], custom_fields |
| `shipments` | Przesyłki | carrier, tracking_number, label_url, status |
| `returns` | Zwroty/RMA | status, reason, refund_amount, return_token |
| `products` | Produkty | sku, ean, price, stock_quantity, images JSONB |
| `product_variants` | Warianty | attributes JSONB, sku, price_override |
| `product_listings` | Oferty marketplace | integration_id, external_id, sync_status |
| `product_bundles` | Zestawy | bundle_product_id, component_product_id, quantity |
| `customers` | Klienci | email, phone, address, total_orders, total_spent |
| `integrations` | Integracje | provider, credentials JSONB (szyfrowane AES) |
| `invoices` | Faktury | provider, external_number, pdf_url, total_gross |
| `warehouses` | Magazyny | name, address, is_default |
| `warehouse_stock` | Stany mag. | product_id, warehouse_id, quantity, reserved |
| `warehouse_documents` | Dok. mag. (PZ/WZ/MM) | document_type, status, warehouse_id |
| `warehouse_document_items` | Pozycje dok. | product_id, quantity, unit_price |
| `suppliers` | Dostawcy | name, feed_url, feed_format |
| `supplier_products` | Katalog dostawcy | external_id, price, stock_quantity |
| `automation_rules` | Reguły automatyzacji | trigger_event, conditions JSONB, actions JSONB |
| `automation_rule_logs` | Logi reguł | conditions_met, actions_executed, error |
| `price_lists` | Cenniki B2B | discount_type, valid_from, valid_to |
| `price_list_items` | Pozycje cennika | product_id, price, min_quantity |
| `exchange_rates` | Kursy walut | base_currency, target_currency, rate |
| `order_groups` | Grupy zamówień | group_type (merge/split), source/target_order_ids |
| `sync_jobs` | Logi synchronizacji | job_type, status, items_processed |
| `webhook_events` | Eventy (przychodzące) | provider, event_type, payload JSONB |
| `webhook_deliveries` | Dostawy (wychodzące) | url, event_type, response_code |
| `audit_log` | Dziennik audytu | action, entity_type, entity_id, ip_address |

### Funkcje SECURITY DEFINER (bypass RLS)

| Funkcja | Cel |
|---------|-----|
| `find_tenant_by_slug(slug)` | Login: znalezienie tenanta po slug |
| `find_user_for_auth(email, tenant_id)` | Login: pobranie usera z hasłem |
| `find_order_tenant_id(order_id)` | Publiczny formularz zwrotu |
| `find_return_by_token(token)` | Status zwrotu po tokenie |

---

## 5. Backend API

### Middleware Stack

```
Request → RequestID → RealIP → Prometheus → Logger → Recoverer → CORS
    → JWTAuth → TokenBlacklist → RequireRole → RequirePermission
    → RateLimit → MaxBodySize → Handler
```

### Wszystkie endpointy (150+)

#### Autentykacja (publiczne, rate limit 100/min)

| Metoda | Ścieżka | Opis |
|--------|---------|------|
| POST | `/v1/auth/register` | Rejestracja (nowy tenant + owner) |
| POST | `/v1/auth/login` | Logowanie → access + refresh token |
| POST | `/v1/auth/refresh` | Odświeżenie access tokena |
| POST | `/v1/auth/logout` | Wylogowanie (blacklist tokena) |

#### Zamówienia

| Metoda | Ścieżka | Opis |
|--------|---------|------|
| GET | `/v1/orders` | Lista z filtrowaniem, sortowaniem, paginacją |
| POST | `/v1/orders` | Tworzenie zamówienia |
| GET | `/v1/orders/export` | Eksport CSV |
| POST | `/v1/orders/bulk-status` | Masowa zmiana statusu |
| POST | `/v1/orders/merge` | Scalenie zamówień |
| POST | `/v1/orders/import/preview` | Podgląd importu CSV |
| POST | `/v1/orders/import` | Import zamówień z CSV |
| GET | `/v1/orders/{id}` | Szczegóły zamówienia |
| PATCH | `/v1/orders/{id}` | Aktualizacja |
| DELETE | `/v1/orders/{id}` | Usunięcie |
| POST | `/v1/orders/{id}/status` | Zmiana statusu (triggeruje webhooki, email, SMS) |
| POST | `/v1/orders/{id}/split` | Podział zamówienia |
| GET | `/v1/orders/{id}/groups` | Grupy zamówień |
| GET | `/v1/orders/{id}/audit` | Historia zmian |
| GET | `/v1/orders/{id}/invoices` | Faktury zamówienia |
| GET | `/v1/orders/{id}/packing-slip` | List przewozowy |
| GET | `/v1/orders/{id}/print` | Wydruk zamówienia |
| POST | `/v1/orders/{id}/pack` | Pakowanie (barcode) |
| GET | `/v1/orders/{id}/tickets` | Tickety helpdesk |
| POST | `/v1/orders/{id}/tickets` | Nowy ticket |

#### Produkty

| Metoda | Ścieżka | Opis |
|--------|---------|------|
| GET | `/v1/products` | Lista z wyszukiwaniem full-text |
| POST | `/v1/products` | Tworzenie produktu |
| GET | `/v1/products/{id}` | Szczegóły |
| PATCH | `/v1/products/{id}` | Aktualizacja |
| DELETE | `/v1/products/{id}` | Usunięcie |
| GET | `/v1/products/{id}/stock` | Stany w magazynach |
| GET/POST/PUT/DELETE | `/v1/products/{id}/bundle/...` | Zestawy (bundles) |
| GET/POST/PATCH/DELETE | `/v1/products/{pid}/variants/...` | Warianty |

#### Przesyłki

| Metoda | Ścieżka | Opis |
|--------|---------|------|
| GET | `/v1/shipments` | Lista przesyłek |
| POST | `/v1/shipments` | Tworzenie (z wyborem przewoźnika) |
| GET | `/v1/shipments/{id}` | Szczegóły |
| PATCH | `/v1/shipments/{id}` | Aktualizacja |
| DELETE | `/v1/shipments/{id}` | Usunięcie |
| POST | `/v1/shipments/{id}/status` | Zmiana statusu |
| POST | `/v1/shipments/{id}/label` | Generowanie etykiety |

#### Zwroty

| Metoda | Ścieżka | Opis |
|--------|---------|------|
| GET | `/v1/returns` | Lista zwrotów |
| POST | `/v1/returns` | Tworzenie |
| GET | `/v1/returns/{id}` | Szczegóły |
| PATCH | `/v1/returns/{id}` | Aktualizacja |
| DELETE | `/v1/returns/{id}` | Usunięcie |
| POST | `/v1/returns/{id}/status` | Zmiana statusu |
| GET | `/v1/returns/{id}/print` | Wydruk |

#### Publiczne zwroty (rate limit 30/min, bez auth)

| Metoda | Ścieżka | Opis |
|--------|---------|------|
| POST | `/v1/public/returns` | Zgłoszenie zwrotu (klient) |
| GET | `/v1/public/returns/{token}` | Status zwrotu (klient) |
| GET | `/v1/public/returns/{token}/status` | Krótki status |

#### Klienci

| Metoda | Ścieżka | Opis |
|--------|---------|------|
| GET | `/v1/customers` | Lista klientów |
| POST | `/v1/customers` | Tworzenie |
| GET | `/v1/customers/{id}` | Szczegóły |
| PATCH | `/v1/customers/{id}` | Aktualizacja |
| DELETE | `/v1/customers/{id}` | Usunięcie |
| GET | `/v1/customers/{id}/orders` | Zamówienia klienta |

#### Faktury

| Metoda | Ścieżka | Opis |
|--------|---------|------|
| GET | `/v1/invoices` | Lista faktur |
| POST | `/v1/invoices` | Tworzenie |
| GET | `/v1/invoices/{id}` | Szczegóły |
| GET | `/v1/invoices/{id}/pdf` | Pobranie PDF |
| DELETE | `/v1/invoices/{id}` | Anulowanie |

#### Integracje (admin)

| Metoda | Ścieżka | Opis |
|--------|---------|------|
| GET | `/v1/integrations` | Lista integracji |
| POST | `/v1/integrations` | Dodanie |
| GET | `/v1/integrations/{id}` | Szczegóły |
| PATCH | `/v1/integrations/{id}` | Aktualizacja |
| DELETE | `/v1/integrations/{id}` | Usunięcie |
| GET | `/v1/integrations/allegro/auth-url` | URL OAuth Allegro |
| POST | `/v1/integrations/allegro/callback` | Callback OAuth |
| POST | `/v1/integrations/amazon/setup` | Setup Amazon SP-API |

#### Dostawcy (admin)

| Metoda | Ścieżka | Opis |
|--------|---------|------|
| GET | `/v1/suppliers` | Lista dostawców |
| POST | `/v1/suppliers` | Dodanie |
| GET | `/v1/suppliers/{id}` | Szczegóły |
| PATCH | `/v1/suppliers/{id}` | Aktualizacja |
| DELETE | `/v1/suppliers/{id}` | Usunięcie |
| POST | `/v1/suppliers/{id}/sync` | Synchronizacja |
| GET | `/v1/suppliers/{id}/products` | Produkty dostawcy |
| POST | `/v1/suppliers/{id}/products/{spid}/link` | Powiązanie z katalogiem |

#### Magazyny (admin)

| Metoda | Ścieżka | Opis |
|--------|---------|------|
| GET | `/v1/warehouses` | Lista magazynów |
| POST | `/v1/warehouses` | Dodanie |
| GET | `/v1/warehouses/{id}` | Szczegóły |
| PATCH | `/v1/warehouses/{id}` | Aktualizacja |
| DELETE | `/v1/warehouses/{id}` | Usunięcie |
| GET | `/v1/warehouses/{id}/stock` | Stany |
| PUT | `/v1/warehouses/{id}/stock` | Ustawienie stanu |

#### Dokumenty magazynowe (admin)

| Metoda | Ścieżka | Opis |
|--------|---------|------|
| GET | `/v1/warehouse-documents` | Lista (PZ/WZ/MM) |
| POST | `/v1/warehouse-documents` | Tworzenie |
| GET | `/v1/warehouse-documents/{id}` | Szczegóły |
| PATCH | `/v1/warehouse-documents/{id}` | Aktualizacja |
| DELETE | `/v1/warehouse-documents/{id}` | Usunięcie |
| POST | `/v1/warehouse-documents/{id}/confirm` | Potwierdzenie |
| POST | `/v1/warehouse-documents/{id}/cancel` | Anulowanie |

#### Automatyzacja (admin)

| Metoda | Ścieżka | Opis |
|--------|---------|------|
| GET | `/v1/automation/rules` | Lista reguł |
| POST | `/v1/automation/rules` | Tworzenie |
| GET | `/v1/automation/rules/{id}` | Szczegóły |
| PATCH | `/v1/automation/rules/{id}` | Aktualizacja |
| DELETE | `/v1/automation/rules/{id}` | Usunięcie |
| GET | `/v1/automation/rules/{id}/logs` | Logi wykonań |
| POST | `/v1/automation/rules/{id}/test` | Test (dry-run) |

#### Ustawienia (admin)

| Metoda | Ścieżka | Opis |
|--------|---------|------|
| GET/PUT | `/v1/settings/company` | Dane firmy + logo |
| GET/PUT | `/v1/settings/order-statuses` | Konfiguracja statusów |
| GET/PUT | `/v1/settings/custom-fields` | Pola niestandardowe |
| GET/PUT | `/v1/settings/product-categories` | Kategorie produktów |
| GET/PUT | `/v1/settings/webhooks` | Webhooki (endpointy) |
| GET/PUT | `/v1/settings/email` | SMTP |
| POST | `/v1/settings/email/test` | Test email |
| GET/PUT | `/v1/settings/sms` | SMS provider |
| POST | `/v1/settings/sms/test` | Test SMS |
| GET/PUT | `/v1/settings/invoicing` | Fakturowanie |
| GET/PUT | `/v1/settings/print-templates` | Szablony druku |

#### Cenniki B2B (admin)

| Metoda | Ścieżka | Opis |
|--------|---------|------|
| GET | `/v1/price-lists` | Lista cenników |
| POST | `/v1/price-lists` | Tworzenie |
| GET | `/v1/price-lists/{id}` | Szczegóły |
| PATCH | `/v1/price-lists/{id}` | Aktualizacja |
| DELETE | `/v1/price-lists/{id}` | Usunięcie |
| GET | `/v1/price-lists/{id}/items` | Pozycje cennika |
| POST | `/v1/price-lists/{id}/items` | Dodanie pozycji |
| DELETE | `/v1/price-lists/{id}/items/{iid}` | Usunięcie pozycji |

#### Role RBAC (admin)

| Metoda | Ścieżka | Opis |
|--------|---------|------|
| GET | `/v1/roles` | Lista ról |
| GET | `/v1/roles/permissions` | Dostępne uprawnienia |
| POST | `/v1/roles` | Tworzenie |
| GET | `/v1/roles/{id}` | Szczegóły |
| PATCH | `/v1/roles/{id}` | Aktualizacja |
| DELETE | `/v1/roles/{id}` | Usunięcie |

#### Kursy walut (admin)

| Metoda | Ścieżka | Opis |
|--------|---------|------|
| GET | `/v1/exchange-rates` | Lista kursów |
| POST | `/v1/exchange-rates` | Dodanie |
| POST | `/v1/exchange-rates/fetch` | Pobranie z NBP |
| POST | `/v1/exchange-rates/convert` | Przeliczenie |
| GET/PATCH/DELETE | `/v1/exchange-rates/{id}` | CRUD |

#### Statystyki

| Metoda | Ścieżka | Opis |
|--------|---------|------|
| GET | `/v1/stats/dashboard` | KPI dashboardu |
| GET | `/v1/stats/products/top` | Top produkty |
| GET | `/v1/stats/revenue/by-source` | Przychody wg źródła |
| GET | `/v1/stats/trends` | Trendy zamówień |
| GET | `/v1/stats/payment-methods` | Metody płatności |

#### AI (wymaga klucza OpenAI)

| Metoda | Ścieżka | Opis |
|--------|---------|------|
| POST | `/v1/ai/categorize` | Kategoryzacja produktu |
| POST | `/v1/ai/describe` | Generowanie opisu |
| POST | `/v1/ai/bulk-categorize` | Masowa kategoryzacja |

#### Inne

| Metoda | Ścieżka | Opis |
|--------|---------|------|
| GET | `/v1/ws` | WebSocket (real-time) |
| GET | `/v1/barcode/{code}` | Lookup barcodu |
| POST | `/v1/uploads` | Upload pliku (10MB max) |
| GET | `/v1/inpost/points` | Wyszukiwanie paczkomatów |
| GET | `/v1/users/me` | Aktualny user |
| GET/POST/PATCH/DELETE | `/v1/users/...` | CRUD userów (admin) |
| GET | `/v1/audit` | Dziennik audytu (admin) |
| GET | `/v1/webhooks` | Konfiguracja webhooków (admin) |
| GET | `/v1/webhook-deliveries` | Log dostaw webhooków (admin) |
| GET | `/v1/sync-jobs` | Logi synchronizacji (admin) |
| POST | `/v1/marketing/sync` | Sync do Mailchimp |
| GET | `/v1/marketing/status` | Status sync |
| GET | `/v1/helpdesk/tickets` | Tickety Freshdesk |
| GET | `/health` | Health check |
| GET | `/metrics` | Prometheus metrics |
| GET | `/v1/openapi.yaml` | Specyfikacja OpenAPI |
| GET | `/v1/docs` | Swagger UI |

---

## 6. Frontend Dashboard

### Mapa stron (64 strony)

#### Publiczne (bez logowania)

| Ścieżka | Strona |
|---------|--------|
| `/login` | Formularz logowania |
| `/register` | Rejestracja firmy |
| `/return-request` | Formularz zwrotu (klient) |
| `/return-request/[token]` | Status zwrotu (klient) |

#### Pulpit

| Ścieżka | Strona |
|---------|--------|
| `/` | Dashboard — KPI, wykresy, ostatnie zamówienia |

#### Sprzedaż

| Ścieżka | Strona |
|---------|--------|
| `/orders` | Lista zamówień (filtrowanie, sortowanie, bulk actions, inline edit) |
| `/orders/new` | Nowe zamówienie |
| `/orders/[id]` | Szczegóły zamówienia (timeline, przesyłki, zwroty, faktury) |
| `/orders/import` | Import zamówień CSV |
| `/shipments` | Lista przesyłek |
| `/shipments/new` | Nowa przesyłka |
| `/shipments/[id]` | Szczegóły przesyłki + etykieta |
| `/returns` | Lista zwrotów |
| `/returns/new` | Nowy zwrot |
| `/returns/[id]` | Szczegóły zwrotu |
| `/invoices` | Lista faktur |
| `/invoices/[id]` | Szczegóły faktury + PDF |
| `/customers` | Lista klientów |
| `/customers/new` | Nowy klient |
| `/customers/[id]` | Profil klienta + historia zamówień |
| `/packing` | Stanowisko pakowania (barcode) |
| `/reports` | Raporty i analizy |

#### Katalog

| Ścieżka | Strona |
|---------|--------|
| `/products` | Lista produktów (search, inline edit, AI kategoryzacja) |
| `/products/new` | Nowy produkt |
| `/products/[id]` | Szczegóły + bundles + AI opis |
| `/products/[id]/variants` | Warianty produktu |
| `/products/[id]/listings` | Oferty na marketplace'ach |

#### Integracje

| Ścieżka | Strona |
|---------|--------|
| `/integrations` | Lista integracji |
| `/integrations/new` | Nowa integracja |
| `/integrations/[id]` | Ustawienia integracji |
| `/integrations/allegro` | Setup Allegro (OAuth) |
| `/integrations/amazon` | Setup Amazon |
| `/suppliers` | Dostawcy |
| `/suppliers/new` | Nowy dostawca |
| `/suppliers/[id]` | Szczegóły dostawcy |

#### Ustawienia (admin)

| Ścieżka | Strona |
|---------|--------|
| `/settings/company` | Dane firmy + logo |
| `/settings/users` | Zarządzanie użytkownikami |
| `/settings/roles` | Role RBAC |
| `/settings/roles/[id]` | Edycja roli |
| `/settings/order-statuses` | Konfiguracja statusów |
| `/settings/custom-fields` | Pola niestandardowe |
| `/settings/notifications` | Email + SMS (tabs) |
| `/settings/webhooks` | Konfiguracja webhooków |
| `/settings/webhooks/deliveries` | Log dostaw |
| `/settings/invoicing` | Fakturowanie |
| `/settings/currencies` | Kursy walut |
| `/settings/print-templates` | Szablony druku |
| `/settings/product-categories` | Kategorie produktów |
| `/settings/price-lists` | Cenniki B2B |
| `/settings/price-lists/[id]` | Edycja cennika |
| `/settings/warehouses` | Magazyny |
| `/settings/warehouses/[id]` | Edycja magazynu |
| `/settings/warehouse-documents` | Dokumenty magazynowe |
| `/settings/warehouse-documents/new` | Nowy dokument |
| `/settings/warehouse-documents/[id]` | Edycja dokumentu |
| `/settings/automation` | Reguły automatyzacji |
| `/settings/automation/new` | Nowa reguła |
| `/settings/automation/[id]` | Edycja reguły |
| `/settings/marketing` | Marketing (Mailchimp) |
| `/settings/helpdesk` | Helpdesk (Freshdesk) |
| `/settings/sync-jobs` | Historia synchronizacji |
| `/audit` | Dziennik aktywności |

### Nawigacja sidebar (grupy)

```
Pulpit (Dashboard)

─── Sprzedaż ───
  Zamówienia
  Przesyłki
  Zwroty
  Faktury
  Import
  Klienci
  Pakowanie
  Raporty

─── Katalog ───
  Produkty
  Kategorie
  Szablony druku

─── Ogólne (admin) ───
  Firma
  Użytkownicy
  Role

─── Sprzedaż - ustawienia (admin) ───
  Statusy zamówień
  Pola niestandardowe
  Cenniki
  Fakturowanie

─── Powiadomienia (admin) ───
  Powiadomienia
  Webhooki

─── Magazyn (admin) ───
  Magazyny
  Dokumenty magazynowe

─── Integracje (admin) ───
  Integracje
  Automatyzacja
  Waluty
  Marketing
  Helpdesk
  Dostawcy

─── Monitoring (admin) ───
  Synchronizacja
  Dostawy webhooków
  Dziennik aktywności
```

### Kluczowe komponenty

| Komponent | Opis |
|-----------|------|
| `DataTable` | Generyczna tabela z sortowaniem, paginacją, selekcją, inline edit |
| `CommandPalette` | Cmd+K — szybkie wyszukiwanie i nawigacja |
| `StatusTransitionDialog` | Dialog zmiany statusu (zamówienia, przesyłki) |
| `PaczkomatSelector` | Wybór paczkomatu InPost (mapa/search/inline) |
| `OrderForm` | Formularz zamówienia (klient, pozycje, adres, custom fields) |
| `TagInput` | Multi-select tagów |
| `AdminGuard` | Wrapper wymuszający rolę admin |
| `ErrorBoundary` | Obsługa błędów z fallback UI |
| `EditableCell` | Edycja inline w tabeli |

### State management

```
┌─────────────┐     ┌──────────────┐     ┌──────────────┐
│   Zustand    │     │ React Query  │     │  API Client  │
│  (auth store)│     │ (data cache) │     │ (fetch+JWT)  │
│              │     │              │     │              │
│ token        │     │ useOrders()  │────→│ GET /orders  │
│ user         │     │ useProducts()│────→│ GET /products│
│ tenant       │     │ useDashboard │────→│ GET /stats   │
│ isAuth       │     │ ...41 hooks  │     │              │
└─────────────┘     └──────────────┘     └──────┬───────┘
                                                 │
                                          Auto-refresh
                                          on 401 (mutex)
```

---

## 7. Pakiety SDK

### Order Engine (packages/order-engine/)

Standalone maszyna stanów zamówień i przesyłek:

```
                  ┌─────────┐
          ┌──────→│confirmed│──────────┐
          │       └────┬────┘          │
          │            │               │
     ┌────┴───┐  ┌─────▼─────┐        │
     │  new   │  │ processing│        │
     └────┬───┘  └─────┬─────┘        │
          │            │               │
          │       ┌────▼──────┐        │
          │       │ready_to   │        │
          │       │ship       │        │
          │       └────┬──────┘        │
          │            │               ▼
          │       ┌────▼────┐    ┌──────────┐
          ├──────→│ shipped │───→│cancelled │
          │       └────┬────┘    └──────────┘
          │            │               │
          │       ┌────▼──────┐        │
          │       │in_transit │        │
          │       └────┬──────┘        │
          │            │               │
          │       ┌────▼──────────┐    │
          │       │out_for_       │    │
          │       │delivery       │    │
          │       └────┬──────────┘    │
          │            │               │
          │       ┌────▼─────┐         │
          │       │delivered │         │
          │       └────┬─────┘         │
          │            │               │
          │       ┌────▼─────┐   ┌─────▼────┐
          │       │completed │──→│ refunded │
          │       └──────────┘   └──────────┘
          │                            ▲
          └────────────────────────────┘
                    (via on_hold)
```

### Marketplace SDK-i

| SDK | Provider | Auth | Główne operacje |
|-----|----------|------|----------------|
| allegro-go-sdk | Allegro.pl | OAuth 2.0 | Zamówienia, oferty, eventy |
| amazon-sp-sdk | Amazon | AWS Signing | Zamówienia, inventory, pricing |
| woocommerce-go-sdk | WooCommerce | REST API | Zamówienia, produkty |
| ebay-go-sdk | eBay | OAuth 2.0 | Zamówienia, inventory |
| kaufland-go-sdk | Kaufland | Feed API | Import CSV/XML |
| olx-go-sdk | OLX | REST | Ogłoszenia |
| mirakl-go-sdk | Mirakl/Empik | REST | Seller network |

### Carrier SDK-i

| SDK | Provider | Auth | Główne operacje |
|-----|----------|------|----------------|
| inpost-go-sdk | InPost | Bearer | Paczki, etykiety, tracking, paczkomaty |
| dhl-go-sdk | DHL | API Key | Przesyłki, etykiety, tracking |
| dpd-go-sdk | DPD | REST | Przesyłki (Polska) |
| gls-go-sdk | GLS | API | Przesyłki (Europa) |
| ups-go-sdk | UPS | XML/REST | Międzynarodowe |
| poczta-polska-go-sdk | Poczta Polska | REST | Paczki pocztowe |
| orlen-paczka-go-sdk | Orlen Paczka | REST | Paczkomaty Orlen |

### Inne SDK-i

| SDK | Provider | Cel |
|-----|----------|-----|
| fakturownia-go-sdk | Fakturownia | Faktury |
| smsapi-go-sdk | SMSAPI | Powiadomienia SMS |
| iof-parser | IOF/CSV | Parser feedów dostawców |

---

## 8. Bezpieczeństwo

### Autentykacja JWT Ed25519

```
JWT_SECRET (env)
    │
    ▼ SHA-512 hash
    │
    ▼ Pierwsze 32 bajty = Ed25519 seed
    │
    ▼ Generowanie pary kluczy
    │
┌───────────────┐     ┌───────────────┐
│ Private Key   │     │ Public Key    │
│ (signing)     │     │ (verify)      │
└───────────────┘     └───────────────┘
```

**Tokeny:**

| Typ | Czas życia | Użycie |
|-----|-----------|--------|
| Access Token | 1 godzina | Header `Authorization: Bearer ...` |
| Refresh Token | 30 dni | Cookie httpOnly (ścieżka /v1/auth) |

**Claims JWT:**
```json
{
  "iss": "openoms",
  "sub": "user-uuid",
  "tid": "tenant-uuid",
  "email": "user@firma.pl",
  "role": "owner",
  "role_id": "role-uuid",
  "type": "access",
  "exp": 1234567890,
  "iat": 1234567890
}
```

### Szyfrowanie AES-256-GCM

Credentials integracji szyfrowane w bazie:
```
Plaintext → AES-256-GCM(key, random_nonce) → Base64 → DB
DB → Base64 decode → AES-256-GCM decrypt → Plaintext
```

Klucz: `ENCRYPTION_KEY` (64-char hex = 32 bajty)

### Hasła — bcrypt (cost 12)

```
password → bcrypt(cost=12) → $2a$12$... → DB
```

### RBAC

```
Stare role:  owner > admin > member
Nowe role:   Custom roles z permissions[]

Uprawnienia np.:
  orders:read, orders:write, orders:delete
  products:read, products:write
  settings:manage
  users:manage
```

### Zabezpieczenia

| Zagrożenie | Mitygacja |
|-----------|-----------|
| SQL Injection | Parametryzowane zapytania (pgx driver) |
| XSS | Sanityzacja HTML w inputach (strip tags) |
| CSRF | SameSite cookies + CORS whitelist |
| Tenant leakage | RLS + FORCE ROW LEVEL SECURITY |
| Token theft | SHA-256 hash w blacklist, httpOnly cookies |
| SSRF | Webhook dispatcher sprawdza private IP ranges |
| Brute force | Rate limiting (100/min auth, 30/min public) |
| DoS | Max body size (1MB default, 10MB upload) |

---

## 9. Kluczowe flow

### Flow 1: Logowanie

```
Użytkownik                Dashboard              API Server              DB
    │                        │                       │                    │
    │  email + password      │                       │                    │
    │  + tenant_slug         │                       │                    │
    │──────────────────────→│                       │                    │
    │                        │  POST /v1/auth/login  │                    │
    │                        │─────────────────────→│                    │
    │                        │                       │  find_tenant_by_slug
    │                        │                       │──────────────────→│
    │                        │                       │←──────────────────│
    │                        │                       │  find_user_for_auth
    │                        │                       │──────────────────→│
    │                        │                       │←──────────────────│
    │                        │                       │  bcrypt.Compare()  │
    │                        │                       │  Ed25519 sign JWT  │
    │                        │  {access_token,       │                    │
    │                        │   user, tenant}       │                    │
    │                        │←─────────────────────│                    │
    │  Zustand: setAuth()    │                       │                    │
    │  Cookie: has_session=1 │                       │                    │
    │←──────────────────────│                       │                    │
    │  Redirect → /          │                       │                    │
```

### Flow 2: Cykl życia zamówienia

```
┌──────────┐     ┌───────────┐     ┌───────────┐     ┌───────────┐
│  NEW     │────→│ CONFIRMED │────→│PROCESSING │────→│READY TO   │
│          │     │           │     │           │     │SHIP       │
└──────────┘     └───────────┘     └───────────┘     └─────┬─────┘
                                                           │
                  Każda zmiana statusu:                     │
                  ├─ Audit log                              │
                  ├─ Webhook dispatch                       │
                  ├─ Email/SMS klientowi                    ▼
                  ├─ Automation rules              ┌───────────┐
                  └─ WebSocket broadcast           │ SHIPPED   │
                                                   └─────┬─────┘
                                                         │
┌──────────┐     ┌───────────┐     ┌───────────┐        │
│COMPLETED │←────│ DELIVERED │←────│IN TRANSIT │←───────┘
│          │     │           │     │           │
└────┬─────┘     └───────────┘     └───────────┘
     │
     ▼
┌──────────┐     ┌───────────┐
│ REFUNDED │←────│ CANCELLED │
│(terminal)│     │           │
└──────────┘     └───────────┘
```

### Flow 3: Webhook dispatch

```
Event (np. order.confirmed)
    │
    ▼
WebhookDispatchService.Dispatch()
    │
    ├─ Załaduj endpoints z tenant settings
    │
    ├─ Dla każdego endpointu:
    │     │
    │     ├─ Serializuj payload → JSON
    │     ├─ HMAC-SHA256(payload, endpoint.secret) → signature
    │     ├─ Sprawdź SSRF (resolve DNS → odrzuć private IP)
    │     ├─ POST endpoint.url
    │     │    Headers: X-Signature-256, X-OpenOMS-Event, X-Delivery-ID
    │     └─ Zapisz wynik w webhook_deliveries
    │
    └─ WebSocket broadcast do tenanta
```

### Flow 4: Automatyzacja

```
Event "order.created"
    │
    ▼
AutomationEngine.ProcessEvent() [async]
    │
    ├─ Załaduj reguły WHERE trigger = "order.created" AND enabled
    │
    ├─ Dla każdej reguły (wg priority):
    │     │
    │     ├─ Ewaluuj warunki:
    │     │    total_amount >= 500? ✓
    │     │    tags contains "bulk"? ✓
    │     │
    │     ├─ Jeśli wszystkie spełnione:
    │     │    ├─ transition_status → "confirmed"
    │     │    ├─ send_email → sales@company.com
    │     │    └─ add_tag → "auto-confirmed"
    │     │
    │     └─ Zapisz log w automation_rule_logs
    │
    └─ Zaktualizuj rule.fire_count, rule.last_fired_at
```

### Flow 5: Generowanie etykiety

```
Użytkownik klika "Generuj etykietę"
    │
    ▼
POST /v1/shipments/{id}/label
    │
    ▼
LabelService.GenerateLabel()
    │
    ├─ Załaduj shipment + order
    ├─ Załaduj integration (credentials)
    ├─ Odszyfruj credentials (AES-256-GCM)
    ├─ Utwórz CarrierProvider (np. InPost)
    ├─ provider.CreateShipment(request)
    │     └─ POST do InPost API
    │        → tracking_number, label_url
    ├─ Zapisz w shipment record
    ├─ Pobierz PDF etykiety
    ├─ Zapisz w storage (S3 lub local)
    └─ Zwróć label URL
```

---

## 10. Integracje

### Marketplace — flow synchronizacji

```
                    ┌──────────────┐
                    │  Marketplace │
                    │ (Allegro,    │
                    │  Amazon...)  │
                    └──────┬───────┘
                           │
              Polling co 45s (Worker)
                           │
                           ▼
┌──────────────────────────────────────────────┐
│            MarketplaceProvider                │
│                                              │
│  interface {                                 │
│    PollOrders(ctx, cursor) → orders          │
│    GetOrder(ctx, externalID) → order         │
│    PushOffer(ctx, product) → externalID      │
│    UpdateStock(ctx, offerID, qty)            │
│    UpdatePrice(ctx, offerID, price)          │
│  }                                           │
└──────────────────────────────────────────────┘
                           │
                           ▼
                    ┌──────────────┐
                    │   OpenOMS    │
                    │   Orders     │
                    └──────────────┘
```

### Carrier — flow wysyłki

```
                    ┌──────────────┐
                    │   Carrier    │
                    │ (InPost,     │
                    │  DHL...)     │
                    └──────┬───────┘
                           │
              Label + Tracking API
                           │
                           ▼
┌──────────────────────────────────────────────┐
│              CarrierProvider                  │
│                                              │
│  interface {                                 │
│    CreateShipment(ctx, req) → response       │
│    GetLabel(ctx, id, format) → PDF           │
│    GetTracking(ctx, tracking#) → events      │
│    CancelShipment(ctx, id)                   │
│    SupportsPickupPoints() → bool             │
│    SearchPickupPoints(ctx, query) → points   │
│  }                                           │
└──────────────────────────────────────────────┘
```

### Obsługiwane integracje

| Kategoria | Provider | Status |
|-----------|----------|--------|
| **Marketplace** | Allegro | OAuth 2.0, polling, oferty |
| | Amazon | SP-API, polling |
| | WooCommerce | REST API, webhooks |
| | eBay | OAuth 2.0 |
| | Kaufland | Feed API |
| | OLX | REST API |
| | Mirakl/Empik | REST API |
| | Erli | REST API |
| **Carrier** | InPost | Paczkomaty, kurier |
| | DHL | Międzynarodowe |
| | DPD | Polska |
| | GLS | Europa |
| | UPS | Międzynarodowe |
| | Poczta Polska | Paczki |
| | Orlen Paczka | Paczkomaty |
| | FedEx | Międzynarodowe |
| **Fakturowanie** | Fakturownia | Faktury VAT |
| **Marketing** | Mailchimp | Sync klientów, kampanie |
| **Helpdesk** | Freshdesk | Tickety |
| **Powiadomienia** | SMTP | Email |
| | Twilio/SMSAPI | SMS |
| **AI** | OpenAI | Kategoryzacja, opisy |
| **Kursy walut** | NBP | Narodowy Bank Polski |

---

## 11. Background Workers

| Worker | Interwał | Cel |
|--------|----------|-----|
| AllegroOrderPoller | 45s | Polling zamówień z Allegro |
| AmazonOrderPoller | 45s | Polling zamówień z Amazon |
| WooCommerceOrderPoller | 45s | Polling zamówień z WooCommerce |
| TrackingPoller | 5min | Aktualizacja statusu przesyłek |
| StockSyncWorker | konfigurowalny | Sync stanów magazynowych |
| SupplierSyncWorker | konfigurowalny | Sync katalogów dostawców |
| ExchangeRateWorker | 1/dzień | Pobranie kursów z NBP |
| OAuthRefresher | 1/dzień | Odświeżenie tokenów OAuth |

Cechy:
- Panic recovery (safeRun wrapper)
- Graceful shutdown
- Logowanie błędów per worker
- Interfejs Worker: `Name()`, `Interval()`, `Run(ctx)`

---

## 12. Automatyzacja

### Trigger events

| Event | Kiedy |
|-------|-------|
| `order.created` | Nowe zamówienie |
| `order.status_changed` | Zmiana statusu zamówienia |
| `order.confirmed` | Zamówienie potwierdzone |
| `shipment.created` | Nowa przesyłka |
| `shipment.status_changed` | Zmiana statusu przesyłki |
| `return.created` | Nowy zwrot |
| `return.status_changed` | Zmiana statusu zwrotu |
| `product.stock_low` | Niski stan magazynowy |

### Warunki (conditions)

```json
[
  { "field": "total_amount", "operator": "gte", "value": 500 },
  { "field": "tags", "operator": "contains", "value": "vip" },
  { "field": "source", "operator": "eq", "value": "allegro" }
]
```

Operatory: `eq`, `neq`, `gt`, `gte`, `lt`, `lte`, `contains`, `not_contains`, `in`, `not_in`

### Akcje (actions)

| Typ akcji | Opis |
|-----------|------|
| `transition_status` | Zmiana statusu zamówienia/przesyłki |
| `send_email` | Wysłanie emaila |
| `send_sms` | Wysłanie SMS |
| `add_tag` | Dodanie tagu |
| `remove_tag` | Usunięcie tagu |
| `set_field` | Ustawienie pola |
| `create_shipment` | Auto-tworzenie przesyłki |
| `webhook` | Wywołanie custom webhook |

### Przykład reguły

```json
{
  "name": "VIP Fast Track",
  "trigger_event": "order.created",
  "conditions": [
    { "field": "total_amount", "operator": "gte", "value": 1000 },
    { "field": "tags", "operator": "contains", "value": "vip" }
  ],
  "actions": [
    { "type": "transition_status", "params": { "status": "confirmed" } },
    { "type": "add_tag", "params": { "tag": "auto-confirmed" } },
    { "type": "send_email", "params": {
      "to": "vip@firma.pl",
      "subject": "Nowe zamówienie VIP",
      "template": "vip_alert"
    }}
  ]
}
```

---

## 13. Konfiguracja

### Zmienne środowiskowe

```bash
# ── Serwer ───────────────────────────
PORT=8080
ENV=production|development

# ── Baza danych ──────────────────────
DATABASE_URL=postgres://openoms:pass@localhost:5433/openoms

# ── Bezpieczeństwo ──────────────────
JWT_SECRET=minimum-32-znaki-losowy-string
ENCRYPTION_KEY=64-znakowy-hex-string

# ── Storage ──────────────────────────
STORAGE_TYPE=s3|local
UPLOAD_DIR=./uploads
MAX_UPLOAD_SIZE=10485760
BASE_URL=https://api.firma.pl

# ── S3 ───────────────────────────────
S3_REGION=eu-central-1
S3_BUCKET=openoms-uploads
S3_ENDPOINT=https://s3.example.com
S3_ACCESS_KEY=...
S3_SECRET_KEY=...
S3_PUBLIC_URL=https://cdn.firma.pl

# ── Frontend ─────────────────────────
FRONTEND_URL=https://app.firma.pl
NEXT_PUBLIC_API_URL=http://localhost:8080

# ── Workers ──────────────────────────
WORKERS_ENABLED=true

# ── Integracje (opcjonalne) ──────────
INPOST_API_TOKEN=...
INPOST_ORG_ID=...
ALLEGRO_WEBHOOK_SECRET=...
OPENAI_API_KEY=...
OPENAI_MODEL=gpt-4
```

### Seed data (testowe)

| Tenant | Slug | Branża | Owner |
|--------|------|--------|-------|
| MercPart | mercpart | Części samochodowe | rafal@mercpart.pl |
| ElektroMax | elektromax | Elektronika | jan@elektromax.pl |
| ZielonyOgród | zielonyogrod | Ogrodnictwo | maria@zielonyogrod.pl |

Hasło testowe: `password123`

---

## 14. Statystyki projektu

| Metryka | Wartość |
|---------|--------|
| **Tabele DB** | 29 |
| **Migracje SQL** | 38 |
| **Endpointy API** | 150+ |
| **Strony frontend** | 64 |
| **Komponenty React** | 70+ |
| **Custom hooks** | 41 |
| **Handlery Go** | 40+ |
| **Serwisy Go** | 35+ |
| **Repozytoria Go** | 25+ |
| **Background workers** | 8 |
| **Marketplace SDK-i** | 8 |
| **Carrier SDK-i** | 8 |
| **Middleware** | 12 |
| **Testy E2E** | 58 |
| **Języki** | Go, TypeScript, SQL |
| **Licencja** | AGPLv3 (apps) + MIT (packages) |

### Testy

| Typ testu | Status |
|-----------|--------|
| E2E Playwright (Chromium) | 48/48 PASS |
| E2E Playwright (pełne) | 58/58 PASS |
| Backend integration | 33/33 PASS |
| API contract (TS ↔ Go) | 42/42 PASS |
| Load testing | 0 błędów, 1000-1800 req/s |
| RLS isolation | 7/7 PASS |
| Clean migration | PASS |

---

*Dokument wygenerowany: 2026-02-11*
*Wersja: OpenOMS v2.0*
