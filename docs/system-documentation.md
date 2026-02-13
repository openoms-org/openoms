# OpenOMS - Kompletna Dokumentacja Systemu

## Spis tresci

1. [Podsumowanie](#1-podsumowanie)
2. [Architektura](#2-architektura)
3. [Stos technologiczny](#3-stos-technologiczny)
4. [Baza danych](#4-baza-danych)
5. [Backend API](#5-backend-api)
6. [Frontend Dashboard](#6-frontend-dashboard)
7. [Pakiety SDK](#7-pakiety-sdk)
8. [Bezpieczenstwo](#8-bezpieczenstwo)
9. [Kluczowe flow](#9-kluczowe-flow)
10. [Integracje](#10-integracje)
11. [Background Workers](#11-background-workers)
12. [Automatyzacja](#12-automatyzacja)
13. [Konfiguracja](#13-konfiguracja)
14. [Statystyki projektu](#14-statystyki-projektu)

---

## 1. Podsumowanie

**OpenOMS** to open-source'owy system zarzadzania zamowieniami (OMS) dla polskiego e-commerce. Jest to aplikacja multi-tenant SaaS z izolacja danych na poziomie bazy danych (PostgreSQL RLS).

### Glowne cechy

- **Multi-tenant** -- pelna izolacja danych miedzy firmami
- **Multi-marketplace** -- Allegro, Amazon, eBay, Kaufland, OLX, WooCommerce, Empik/Mirakl, Erli
- **Multi-carrier** -- InPost, DHL, DPD, GLS, UPS, Poczta Polska, Orlen Paczka, FedEx
- **Automatyzacja** -- silnik regul (trigger -> warunki -> akcje) z obsluga opoznionych akcji
- **Fakturowanie** -- integracja z Fakturownia + KSeF (Krajowy System e-Faktur)
- **Powiadomienia** -- Email (SMTP) + SMS (Twilio/SMSAPI)
- **RBAC** -- role z granularnymi uprawnieniami
- **2FA/TOTP** -- dwuskladnikowe uwierzytelnianie (Google Authenticator)
- **API REST** -- ~296 endpointow z OpenAPI 3.1
- **Dashboard** -- Next.js 16 + React 19, 81 stron, dark mode, PWA
- **AI** -- auto-kategoryzacja, opis, ulepszanie i tlumaczenie produktow (OpenAI)
- **Inwentaryzacja** -- pelny cykl zycia stocktake z liczeniem pozycji
- **Rate shopping** -- porownywanie stawek przewoznikow
- **Allegro Listings** -- kreator ofert z 4-krokowym wizardem
- **Kanban board** -- widok zamowien w formie tablicy Kanban
- **Import CSV** -- import produktow i zamowien z podgladem
- **Command Palette** -- Cmd+K do szybkiej nawigacji i wyszukiwania

### Licencja

- `apps/` -- AGPLv3 (core)
- `packages/` -- MIT (SDK-i)

---

## 2. Architektura

### Diagram wysokopoziomowy

```
+-------------------------------------------------------------+
|                        KLIENCI                               |
|  +----------+  +----------+  +----------+  +------------+   |
|  | Dashboard |  |  API     |  | Webhook  |  | Public     |   |
|  | (Next.js) |  | Consumer |  | Sender   |  | Return     |   |
|  +-----+----+  +-----+----+  +-----+----+  +------+-----+   |
+---------+-------------+-------------+---------------+--------+
          |             |             |               |
          v             v             v               v
+-------------------------------------------------------------+
|                    API SERVER (Go)                            |
|  +---------+  +----------+  +-----------+  +------------+    |
|  |  Router  |->|Middleware|->| Handlers  |->|  Services  |   |
|  |  (chi)   |  |JWT+CORS |  | (HTTP)    |  | (logika)   |   |
|  +---------+  +----------+  +-----------+  +------+------+  |
|                                                   |          |
|  +-------------+  +----------+  +--------------+  |          |
|  |  Workers    |  |WebSocket |  |  Automation  |  |          |
|  | (bg jobs)   |  |  Hub     |  |  Engine      |  |          |
|  +------+------+  +-----+----+  +------+------+  |          |
|         |              |               |          |          |
|         v              v               v          v          |
|  +-----------------------------------------------------+    |
|  |              Repositories (Data Access)               |   |
|  +------------------------+------------------------------+   |
+----------------------------+--------------------------------+
                             |
                             v
+-------------------------------------------------------------+
|                   PostgreSQL 16                               |
|  +----------+  +--------------+  +-----------------------+   |
|  | 32 tabel |  |  RLS Policy  |  |  SECURITY DEFINER     |  |
|  | 46 migr. |  | (per tenant) |  |  (auth bypass)        |  |
|  +----------+  +--------------+  +-----------------------+   |
+-------------------------------------------------------------+
```

### Wzorzec warstwowy

```
HTTP Request
    |
    v
+----------+     +------------+     +--------------+     +----------+
| Handler  | --> |  Service   | --> | Repository   | --> |   DB     |
| (waliduj |     | (logika    |     | (SQL query)  |     | (pgx +   |
|  + HTTP) |     |  biznesowa)|     |              |     |  RLS)    |
+----------+     +------------+     +--------------+     +----------+
```

### Multi-tenancy

```
Request -> JWT Token -> TenantID extraction
    |
    v
database.WithTenant(ctx, pool, tenantID, func(tx) {
    SET app.current_tenant_id = $1  <- parametryzowane!
    ...query...                      <- RLS filtruje automatycznie
})
```

Kazda tabela ma polityke RLS:
```sql
CREATE POLICY tenant_isolation ON orders
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
```

---

## 3. Stos technologiczny

### Backend

| Komponent | Technologia | Wersja |
|-----------|------------|--------|
| Jezyk | Go | 1.24 |
| Router HTTP | chi/v5 | 5.x |
| Baza danych | PostgreSQL | 16 |
| Driver DB | pgx/v5 | 5.x |
| JWT | golang-jwt/v5 + Ed25519 | 5.x |
| Logowanie | log/slog | std |
| Konfiguracja | caarlos0/env/v11 | 11.x |
| Migracje | golang-migrate | 4.x |
| WebSocket | gorilla/websocket | 1.x |
| Metryki | prometheus/client_golang | 1.x |
| 2FA/TOTP | pquerna/otp | 1.x |

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
+-- apps/
|   +-- api-server/          <- Go backend (AGPLv3)
|   |   +-- cmd/server/      <- punkt wejscia
|   |   +-- internal/        <- logika aplikacji (386 plikow Go, 71 testow)
|   |   +-- migrations/      <- 46 migracji SQL (000001-000046)
|   +-- dashboard/           <- Next.js frontend (AGPLv3)
|       +-- src/app/         <- 81 stron (App Router)
|       +-- src/components/  <- 81 komponentow React
|       +-- src/hooks/       <- 45 custom hooks
|       +-- src/lib/         <- utils, API client, auth
|       +-- e2e/             <- 12 specow E2E Playwright
+-- packages/                <- SDK-i (MIT)
|   +-- order-engine/        <- maszyna stanow zamowien
|   +-- allegro-go-sdk/      <- Allegro REST API
|   +-- inpost-go-sdk/       <- InPost ShipX API
|   +-- ksef-go-sdk/         <- KSeF e-Faktur API
|   +-- ...                  <- 21 pakietow SDK
+-- docs/                    <- dokumentacja
```

---

## 4. Baza danych

### Diagram ERD (uproszczony)

```
+----------+       +----------+       +----------+
| tenants  |------<|  users   |       |  roles   |
|          |       |          |>------|          |
+----+-----+       | totp_*   |       +----------+
     |              +----------+
     |
     |  +--------------+   +------------+   +--------------+
     +-<|   orders     |--<| shipments  |   |   returns    |
     |  |              |   |            |   |              |
     |  |  items[]     |   | carrier    |   |  status      |
     |  |  status      |--<| tracking#  |   |  refund_amt  |
     |  |  tags[]      |   +------------+   |  return_token|
     |  |  custom_flds |                    +--------------+
     |  |  priority    |
     |  |  int_notes   |
     |  +------+-------+
     |         |
     |         +------------------<+--------------+
     |         |                   |  invoices    |
     |         |                   |  ksef_*      |
     |         |                   +--------------+
     |
     |  +--------------+   +----------------+   +--------------+
     +-<|  products    |--<| prod_variants  |   | prod_bundles |
     |  |              |   +----------------+   +--------------+
     |  |  sku, ean    |--<+----------------+
     |  |  images[]    |   | prod_listings  |
     |  |  tags[]      |   +----------------+
     |  +------+-------+
     |         |
     |         +---------<+------------------+
     |         |          | warehouse_stock   |
     |         |          +--------+---------+
     |         |                   |
     |  +--------------+   +------+------+   +------------------+
     +-<|  warehouses  |   | wh_documents|--<| wh_document_items|
     |  +--------------+   +-------------+   +------------------+
     |
     |  +--------------+   +------------------+
     +-<| stocktakes   |--<| stocktake_items  |
     |  +--------------+   +------------------+
     |
     |  +--------------+   +----------------+
     +-<| integrations |   |  sync_jobs     |
     |  |  (encrypted) |--<|  (append-only) |
     |  +--------------+   +----------------+
     |
     |  +--------------+   +--------------+
     +-<|  customers   |   |  suppliers   |--<+-----------------+
     |  |  tags[]      |   |              |   |supplier_products|
     |  +--------------+   +--------------+   +-----------------+
     |
     |  +--------------+   +----------------+   +--------------+
     +-<| automation   |--<| auto_rule_logs |   | price_lists  |
     |  | _rules       |   +----------------+   |              |--<+---------+
     |  +--------------+                        +--------------+   | pl_items|
     |         |                                                    +---------+
     |         +--<+------------------------+
     |             | auto_delayed_actions   |
     |             +------------------------+
     |
     |  +--------------+   +----------------+   +--------------+
     +-<|  audit_log   |   | webhook_events |   | wh_deliveries|
     |  +--------------+   +----------------+   +--------------+
     |
     +-<+--------------+   +----------------+
        | order_groups |   | exchange_rates |
        +--------------+   +----------------+
```

### Wszystkie tabele (32)

| Tabela | Cel | Kluczowe kolumny |
|--------|-----|-----------------|
| `tenants` | Konta firm | name, slug, plan, settings JSONB |
| `users` | Uzytkownicy | email, name, role, role_id, password_hash, totp_secret, totp_enabled |
| `roles` | Role RBAC | name, permissions TEXT[], is_system |
| `orders` | Zamowienia | status, items JSONB, total_amount, tags[], custom_fields, priority, internal_notes |
| `shipments` | Przesylki | carrier, tracking_number, label_url, status, warehouse_id |
| `returns` | Zwroty/RMA | status, reason, refund_amount, return_token, customer_email |
| `products` | Produkty | sku, ean, price, stock_quantity, images JSONB, description, dimensions |
| `product_variants` | Warianty | attributes JSONB, sku, price_override |
| `product_listings` | Oferty marketplace | integration_id, external_id, sync_status, price_override |
| `product_bundles` | Zestawy | bundle_product_id, component_product_id, quantity |
| `customers` | Klienci | email, phone, name, company_name, nip, total_orders, total_spent |
| `integrations` | Integracje | provider, credentials JSONB (szyfrowane AES), settings |
| `invoices` | Faktury | provider, external_number, pdf_url, total_gross, ksef_number, ksef_status |
| `warehouses` | Magazyny | name, address, is_default, active |
| `warehouse_stock` | Stany mag. | product_id, warehouse_id, quantity, reserved, min_stock |
| `warehouse_documents` | Dok. mag. (PZ/WZ/MM) | document_type, status, warehouse_id, target_warehouse_id |
| `warehouse_document_items` | Pozycje dok. | product_id, quantity, unit_price |
| `stocktakes` | Inwentaryzacja | warehouse_id, status, started_at, completed_at, created_by |
| `stocktake_items` | Pozycje inwent. | product_id, expected_quantity, counted_quantity, difference |
| `suppliers` | Dostawcy | name, feed_url, feed_format, last_sync_at |
| `supplier_products` | Katalog dostawcy | external_id, price, stock_quantity, ean |
| `automation_rules` | Reguly automatyzacji | trigger_event, conditions JSONB, actions JSONB, priority |
| `automation_rule_logs` | Logi regul | conditions_met, actions_executed, error |
| `automation_delayed_actions` | Opoznione akcje | rule_id, order_id, execute_at, executed, action_data JSONB |
| `price_lists` | Cenniki B2B | discount_type, valid_from, valid_to, currency |
| `price_list_items` | Pozycje cennika | product_id, price, min_quantity, discount |
| `exchange_rates` | Kursy walut | base_currency, target_currency, rate, source |
| `order_groups` | Grupy zamowien | group_type (merge/split), source/target_order_ids |
| `sync_jobs` | Logi synchronizacji | job_type, status, items_processed |
| `webhook_events` | Eventy (przychodzace) | provider, event_type, payload JSONB |
| `webhook_deliveries` | Dostawy (wychodzace) | url, event_type, response_code |
| `audit_log` | Dziennik audytu | action, entity_type, entity_id, ip_address |

### Funkcje SECURITY DEFINER (bypass RLS)

| Funkcja | Cel |
|---------|-----|
| `find_tenant_by_slug(slug)` | Login: znalezienie tenanta po slug |
| `find_user_for_auth(email, tenant_id)` | Login: pobranie usera z haslem + TOTP |
| `find_order_tenant_id(order_id)` | Publiczny formularz zwrotu |
| `find_return_by_token(token)` | Status zwrotu po tokenie |

---

## 5. Backend API

### Middleware Stack (12 middleware)

```
Request -> RequestID -> RealIP -> Prometheus -> SecurityHeaders -> Logger -> Recoverer -> CORS
    -> JWTAuth -> TokenBlacklist -> RequireRole -> RequirePermission
    -> RateLimit -> MaxBodySize -> MetricsAuth -> Handler
```

### Wszystkie endpointy (~296)

#### Autentykacja (publiczne, rate limit 10/min login, 60/min refresh)

| Metoda | Sciezka | Opis |
|--------|---------|------|
| POST | `/v1/auth/register` | Rejestracja (nowy tenant + owner) |
| POST | `/v1/auth/login` | Logowanie -> access + refresh token |
| POST | `/v1/auth/refresh` | Odswiezenie access tokena |
| POST | `/v1/auth/logout` | Wylogowanie (blacklist tokena) |
| POST | `/v1/auth/2fa/login` | Logowanie z kodem TOTP (2FA) |

#### 2FA/TOTP (wymaga JWT)

| Metoda | Sciezka | Opis |
|--------|---------|------|
| POST | `/v1/auth/2fa/setup` | Generowanie sekretu TOTP + QR code |
| POST | `/v1/auth/2fa/verify` | Weryfikacja kodu i wlaczenie 2FA |
| POST | `/v1/auth/2fa/disable` | Wylaczenie 2FA |
| GET | `/v1/auth/2fa/status` | Sprawdzenie statusu 2FA |

#### Zamowienia

| Metoda | Sciezka | Opis |
|--------|---------|------|
| GET | `/v1/orders` | Lista z filtrowaniem, sortowaniem, paginacja |
| POST | `/v1/orders` | Tworzenie zamowienia |
| GET | `/v1/orders/export` | Eksport CSV |
| POST | `/v1/orders/bulk-status` | Masowa zmiana statusu |
| POST | `/v1/orders/merge` | Scalenie zamowien |
| POST | `/v1/orders/import/preview` | Podglad importu CSV |
| POST | `/v1/orders/import` | Import zamowien z CSV |
| GET | `/v1/orders/{id}` | Szczegoly zamowienia |
| PATCH | `/v1/orders/{id}` | Aktualizacja |
| DELETE | `/v1/orders/{id}` | Usuniecie |
| POST | `/v1/orders/{id}/status` | Zmiana statusu (triggeruje webhooki, email, SMS) |
| POST | `/v1/orders/{id}/duplicate` | Duplikowanie zamowienia |
| POST | `/v1/orders/{id}/split` | Podzial zamowienia |
| GET | `/v1/orders/{id}/groups` | Grupy zamowien |
| GET | `/v1/orders/{id}/audit` | Historia zmian |
| GET | `/v1/orders/{id}/invoices` | Faktury zamowienia |
| GET | `/v1/orders/{id}/packing-slip` | List przewozowy |
| GET | `/v1/orders/{id}/print` | Wydruk zamowienia |
| POST | `/v1/orders/{id}/pack` | Pakowanie (barcode) |
| GET | `/v1/orders/{id}/tickets` | Tickety helpdesk |
| POST | `/v1/orders/{id}/tickets` | Nowy ticket |

#### Produkty

| Metoda | Sciezka | Opis |
|--------|---------|------|
| GET | `/v1/products` | Lista z wyszukiwaniem full-text |
| POST | `/v1/products` | Tworzenie produktu |
| GET | `/v1/products/export` | Eksport CSV |
| POST | `/v1/products/import/preview` | Podglad importu CSV |
| POST | `/v1/products/import` | Import produktow z CSV |
| GET | `/v1/products/{id}` | Szczegoly |
| PATCH | `/v1/products/{id}` | Aktualizacja |
| DELETE | `/v1/products/{id}` | Usuniecie |
| GET | `/v1/products/{id}/stock` | Stany w magazynach |
| GET/POST/PUT/DELETE | `/v1/products/{id}/bundle/...` | Zestawy (bundles) |
| GET | `/v1/products/{id}/bundle/stock` | Dostepnosc zestawu |
| GET/POST/PATCH/DELETE | `/v1/products/{pid}/variants/...` | Warianty |

#### Oferty Allegro (admin)

| Metoda | Sciezka | Opis |
|--------|---------|------|
| GET | `/v1/products/{pid}/listings` | Lista ofert produktu |
| POST | `/v1/products/{pid}/listings/allegro` | Tworzenie oferty Allegro |
| GET | `/v1/products/{pid}/listings/{lid}` | Szczegoly oferty |
| PATCH | `/v1/products/{pid}/listings/{lid}` | Aktualizacja oferty |
| DELETE | `/v1/products/{pid}/listings/{lid}` | Usuniecie oferty |
| POST | `/v1/products/{pid}/listings/{lid}/sync` | Synchronizacja oferty |

#### Przesylki

| Metoda | Sciezka | Opis |
|--------|---------|------|
| GET | `/v1/shipments` | Lista przesylek |
| POST | `/v1/shipments` | Tworzenie (z wyborem przewoznika) |
| POST | `/v1/shipments/batch-labels` | Wsadowe generowanie etykiet |
| POST | `/v1/shipments/dispatch-order` | Tworzenie zlecenia odbioru |
| GET | `/v1/shipments/{id}` | Szczegoly |
| PATCH | `/v1/shipments/{id}` | Aktualizacja |
| DELETE | `/v1/shipments/{id}` | Usuniecie |
| POST | `/v1/shipments/{id}/status` | Zmiana statusu |
| POST | `/v1/shipments/{id}/label` | Generowanie etykiety |
| GET | `/v1/shipments/{id}/tracking` | Sledzenie przesylki |

#### Zwroty

| Metoda | Sciezka | Opis |
|--------|---------|------|
| GET | `/v1/returns` | Lista zwrotow |
| POST | `/v1/returns` | Tworzenie |
| GET | `/v1/returns/{id}` | Szczegoly |
| PATCH | `/v1/returns/{id}` | Aktualizacja |
| DELETE | `/v1/returns/{id}` | Usuniecie |
| POST | `/v1/returns/{id}/status` | Zmiana statusu |
| GET | `/v1/returns/{id}/print` | Wydruk |

#### Publiczne zwroty (rate limit 30/min, bez auth)

| Metoda | Sciezka | Opis |
|--------|---------|------|
| POST | `/v1/public/returns` | Zgloszenie zwrotu (klient) |
| GET | `/v1/public/returns/{token}` | Status zwrotu (klient) |
| GET | `/v1/public/returns/{token}/status` | Krotki status |

#### Klienci

| Metoda | Sciezka | Opis |
|--------|---------|------|
| GET | `/v1/customers` | Lista klientow |
| POST | `/v1/customers` | Tworzenie |
| GET | `/v1/customers/{id}` | Szczegoly |
| PATCH | `/v1/customers/{id}` | Aktualizacja |
| DELETE | `/v1/customers/{id}` | Usuniecie |
| GET | `/v1/customers/{id}/orders` | Zamowienia klienta |

#### Faktury + KSeF

| Metoda | Sciezka | Opis |
|--------|---------|------|
| GET | `/v1/invoices` | Lista faktur |
| POST | `/v1/invoices` | Tworzenie |
| POST | `/v1/invoices/ksef/bulk-send` | Masowe wysylanie do KSeF |
| GET | `/v1/invoices/{id}` | Szczegoly |
| GET | `/v1/invoices/{id}/pdf` | Pobranie PDF |
| DELETE | `/v1/invoices/{id}` | Anulowanie |
| POST | `/v1/invoices/{id}/ksef/send` | Wyslanie do KSeF |
| GET | `/v1/invoices/{id}/ksef/status` | Sprawdzenie statusu KSeF |
| GET | `/v1/invoices/{id}/ksef/upo` | Pobranie UPO z KSeF |

#### Integracje (admin)

| Metoda | Sciezka | Opis |
|--------|---------|------|
| GET | `/v1/integrations` | Lista integracji |
| POST | `/v1/integrations` | Dodanie |
| GET | `/v1/integrations/{id}` | Szczegoly |
| PATCH | `/v1/integrations/{id}` | Aktualizacja |
| DELETE | `/v1/integrations/{id}` | Usuniecie |
| GET | `/v1/integrations/allegro/auth-url` | URL OAuth Allegro |
| POST | `/v1/integrations/allegro/callback` | Callback OAuth |
| POST | `/v1/integrations/amazon/setup` | Setup Amazon SP-API |

#### Allegro -- Fulfillment i sledzenie (admin)

| Metoda | Sciezka | Opis |
|--------|---------|------|
| GET | `/v1/integrations/allegro/carriers` | Lista przewoznikow Allegro |
| POST | `/v1/integrations/allegro/sync` | Synchronizacja zamowien |
| POST | `/v1/integrations/allegro/orders/{oid}/fulfillment` | Aktualizacja fulfillment |
| POST | `/v1/integrations/allegro/orders/{oid}/tracking` | Dodanie trackingu |

#### Allegro -- Wysylam z Allegro (admin)

| Metoda | Sciezka | Opis |
|--------|---------|------|
| GET | `/v1/integrations/allegro/delivery-services` | Uslugi dostawy |
| POST | `/v1/integrations/allegro/shipments` | Tworzenie przesylki |
| GET | `/v1/integrations/allegro/shipments/{sid}/label` | Pobranie etykiety |
| DELETE | `/v1/integrations/allegro/shipments/{sid}` | Anulowanie |
| POST | `/v1/integrations/allegro/pickup-proposals` | Propozycje odbioru |
| POST | `/v1/integrations/allegro/pickups` | Zaplanowanie odbioru |
| POST | `/v1/integrations/allegro/protocol` | Generowanie protokolu |

#### Allegro -- Wiadomosci i zwroty (admin)

| Metoda | Sciezka | Opis |
|--------|---------|------|
| GET | `/v1/integrations/allegro/messages` | Lista watkow |
| GET | `/v1/integrations/allegro/messages/{tid}` | Wiadomosci w watku |
| POST | `/v1/integrations/allegro/messages/{tid}` | Wyslanie wiadomosci |
| GET | `/v1/integrations/allegro/returns` | Zwroty Allegro |
| GET | `/v1/integrations/allegro/returns/{rid}` | Szczegoly zwrotu |
| POST | `/v1/integrations/allegro/returns/{rid}/reject` | Odrzucenie zwrotu |
| POST | `/v1/integrations/allegro/refunds` | Tworzenie refundacji |
| GET | `/v1/integrations/allegro/refunds` | Lista refundacji |

#### Allegro -- Konto i oferty (admin)

| Metoda | Sciezka | Opis |
|--------|---------|------|
| GET | `/v1/integrations/allegro/account` | Dane konta |
| GET | `/v1/integrations/allegro/billing` | Rozliczenia |
| GET | `/v1/integrations/allegro/offers` | Lista ofert |
| POST | `/v1/integrations/allegro/offers/{oid}/deactivate` | Dezaktywacja |
| POST | `/v1/integrations/allegro/offers/{oid}/activate` | Aktywacja |
| PATCH | `/v1/integrations/allegro/offers/{oid}/stock` | Aktualizacja stanu |
| PATCH | `/v1/integrations/allegro/offers/{oid}/price` | Aktualizacja ceny |

#### Allegro -- Katalog i finanse (admin)

| Metoda | Sciezka | Opis |
|--------|---------|------|
| GET | `/v1/integrations/allegro/categories` | Lista kategorii |
| GET | `/v1/integrations/allegro/categories/search` | Wyszukiwanie kategorii |
| GET | `/v1/integrations/allegro/categories/{cid}` | Szczegoly kategorii |
| GET | `/v1/integrations/allegro/categories/{cid}/parameters` | Parametry kategorii |
| GET | `/v1/integrations/allegro/products/catalog` | Wyszukiwanie w katalogu |
| GET | `/v1/integrations/allegro/products/catalog/{pid}` | Produkt z katalogu |
| GET | `/v1/integrations/allegro/pricing/fees` | Podglad prowizji |
| GET | `/v1/integrations/allegro/pricing/commissions` | Tabela prowizji |

#### Allegro -- Polityki po-sprzedazowe (admin)

| Metoda | Sciezka | Opis |
|--------|---------|------|
| GET/POST | `/v1/integrations/allegro/return-policies` | Polityki zwrotow |
| GET/PUT | `/v1/integrations/allegro/return-policies/{pid}` | Edycja polityki |
| GET/POST | `/v1/integrations/allegro/warranties` | Gwarancje |
| GET/PUT | `/v1/integrations/allegro/warranties/{wid}` | Edycja gwarancji |
| GET/POST | `/v1/integrations/allegro/size-tables` | Tabele rozmiarow |
| GET/PUT/DELETE | `/v1/integrations/allegro/size-tables/{tid}` | Edycja tabeli |

#### Allegro -- Promocje (admin)

| Metoda | Sciezka | Opis |
|--------|---------|------|
| GET | `/v1/integrations/allegro/promotions` | Lista promocji |
| POST | `/v1/integrations/allegro/promotions` | Tworzenie promocji |
| GET | `/v1/integrations/allegro/promotions/{pid}` | Szczegoly |
| PUT | `/v1/integrations/allegro/promotions/{pid}` | Aktualizacja |
| DELETE | `/v1/integrations/allegro/promotions/{pid}` | Usuniecie |
| GET | `/v1/integrations/allegro/promotion-badges` | Odznaki promocyjne |

#### Allegro -- Dostawa (admin)

| Metoda | Sciezka | Opis |
|--------|---------|------|
| GET | `/v1/integrations/allegro/delivery-settings` | Ustawienia dostawy |
| PUT | `/v1/integrations/allegro/delivery-settings` | Aktualizacja ustawien |
| GET | `/v1/integrations/allegro/shipping-rates` | Cenniki dostawy |
| POST | `/v1/integrations/allegro/shipping-rates` | Tworzenie cennika |
| POST | `/v1/integrations/allegro/shipping-rates/auto-generate` | Auto-generowanie |
| GET | `/v1/integrations/allegro/shipping-rates/{rid}` | Szczegoly cennika |
| PUT | `/v1/integrations/allegro/shipping-rates/{rid}` | Aktualizacja cennika |
| GET | `/v1/integrations/allegro/delivery-methods` | Metody dostawy |

#### Allegro -- Spory (admin)

| Metoda | Sciezka | Opis |
|--------|---------|------|
| GET | `/v1/integrations/allegro/disputes` | Lista sporow |
| GET | `/v1/integrations/allegro/disputes/{did}` | Szczegoly sporu |
| GET | `/v1/integrations/allegro/disputes/{did}/messages` | Wiadomosci sporu |
| POST | `/v1/integrations/allegro/disputes/{did}/messages` | Odpowiedz w sporze |

#### Allegro -- Oceny (admin)

| Metoda | Sciezka | Opis |
|--------|---------|------|
| GET | `/v1/integrations/allegro/ratings` | Lista ocen |
| GET | `/v1/integrations/allegro/ratings/{rid}/answer` | Odpowiedz na ocene |
| PUT | `/v1/integrations/allegro/ratings/{rid}/answer` | Tworzenie odpowiedzi |
| DELETE | `/v1/integrations/allegro/ratings/{rid}/answer` | Usuniecie odpowiedzi |
| POST | `/v1/integrations/allegro/ratings/{rid}/removal` | Wniosek o usuniecie |

#### Dostawcy (admin)

| Metoda | Sciezka | Opis |
|--------|---------|------|
| GET | `/v1/suppliers` | Lista dostawcow |
| POST | `/v1/suppliers` | Dodanie |
| GET | `/v1/suppliers/{id}` | Szczegoly |
| PATCH | `/v1/suppliers/{id}` | Aktualizacja |
| DELETE | `/v1/suppliers/{id}` | Usuniecie |
| POST | `/v1/suppliers/{id}/sync` | Synchronizacja |
| GET | `/v1/suppliers/{id}/products` | Produkty dostawcy |
| POST | `/v1/suppliers/{id}/products/{spid}/link` | Powiazanie z katalogiem |

#### Magazyny (admin)

| Metoda | Sciezka | Opis |
|--------|---------|------|
| GET | `/v1/warehouses` | Lista magazynow |
| POST | `/v1/warehouses` | Dodanie |
| GET | `/v1/warehouses/{id}` | Szczegoly |
| PATCH | `/v1/warehouses/{id}` | Aktualizacja |
| DELETE | `/v1/warehouses/{id}` | Usuniecie |
| GET | `/v1/warehouses/{id}/stock` | Stany |
| PUT | `/v1/warehouses/{id}/stock` | Ustawienie stanu |

#### Dokumenty magazynowe (admin)

| Metoda | Sciezka | Opis |
|--------|---------|------|
| GET | `/v1/warehouse-documents` | Lista (PZ/WZ/MM) |
| POST | `/v1/warehouse-documents` | Tworzenie |
| GET | `/v1/warehouse-documents/{id}` | Szczegoly |
| PATCH | `/v1/warehouse-documents/{id}` | Aktualizacja |
| DELETE | `/v1/warehouse-documents/{id}` | Usuniecie |
| POST | `/v1/warehouse-documents/{id}/confirm` | Potwierdzenie |
| POST | `/v1/warehouse-documents/{id}/cancel` | Anulowanie |

#### Inwentaryzacja (admin)

| Metoda | Sciezka | Opis |
|--------|---------|------|
| GET | `/v1/stocktakes` | Lista inwentaryzacji |
| POST | `/v1/stocktakes` | Tworzenie nowej |
| GET | `/v1/stocktakes/{id}` | Szczegoly |
| DELETE | `/v1/stocktakes/{id}` | Usuniecie |
| POST | `/v1/stocktakes/{id}/start` | Rozpoczecie liczenia |
| POST | `/v1/stocktakes/{id}/items/{iid}/count` | Zapis policzonych sztuk |
| POST | `/v1/stocktakes/{id}/complete` | Zakonczenie i aktualizacja stanow |
| POST | `/v1/stocktakes/{id}/cancel` | Anulowanie |
| GET | `/v1/stocktakes/{id}/items` | Lista pozycji do policzenia |

#### Automatyzacja (admin)

| Metoda | Sciezka | Opis |
|--------|---------|------|
| GET | `/v1/automation/delayed` | Lista opoznionych akcji |
| GET | `/v1/automation/rules` | Lista regul |
| POST | `/v1/automation/rules` | Tworzenie |
| GET | `/v1/automation/rules/{id}` | Szczegoly |
| PATCH | `/v1/automation/rules/{id}` | Aktualizacja |
| DELETE | `/v1/automation/rules/{id}` | Usuniecie |
| GET | `/v1/automation/rules/{id}/logs` | Logi wykonan |
| POST | `/v1/automation/rules/{id}/test` | Test (dry-run) |

#### Ustawienia (admin)

| Metoda | Sciezka | Opis |
|--------|---------|------|
| GET | `/v1/settings/export` | Eksport ustawien |
| POST | `/v1/settings/import` | Import ustawien |
| GET/PUT | `/v1/settings/company` | Dane firmy + logo |
| GET/PUT | `/v1/settings/order-statuses` | Konfiguracja statusow |
| GET/PUT | `/v1/settings/custom-fields` | Pola niestandardowe |
| GET/PUT | `/v1/settings/product-categories` | Kategorie produktow |
| GET/PUT | `/v1/settings/webhooks` | Webhooki (endpointy) |
| GET/PUT | `/v1/settings/email` | SMTP |
| POST | `/v1/settings/email/test` | Test email |
| GET/PUT | `/v1/settings/sms` | SMS provider |
| POST | `/v1/settings/sms/test` | Test SMS |
| GET/PUT | `/v1/settings/invoicing` | Fakturowanie |
| GET/PUT | `/v1/settings/inventory` | Tryb scisly magazynu |
| GET/PUT | `/v1/settings/print-templates` | Szablony druku |
| GET/PUT | `/v1/settings/ksef` | Ustawienia KSeF |
| POST | `/v1/settings/ksef/test` | Test polaczenia KSeF |

#### Cenniki B2B (admin)

| Metoda | Sciezka | Opis |
|--------|---------|------|
| GET | `/v1/price-lists` | Lista cennikow |
| POST | `/v1/price-lists` | Tworzenie |
| GET | `/v1/price-lists/{id}` | Szczegoly |
| PATCH | `/v1/price-lists/{id}` | Aktualizacja |
| DELETE | `/v1/price-lists/{id}` | Usuniecie |
| GET | `/v1/price-lists/{id}/items` | Pozycje cennika |
| POST | `/v1/price-lists/{id}/items` | Dodanie pozycji |
| DELETE | `/v1/price-lists/{id}/items/{iid}` | Usuniecie pozycji |

#### Role RBAC (admin)

| Metoda | Sciezka | Opis |
|--------|---------|------|
| GET | `/v1/roles` | Lista rol |
| GET | `/v1/roles/permissions` | Dostepne uprawnienia |
| POST | `/v1/roles` | Tworzenie |
| GET | `/v1/roles/{id}` | Szczegoly |
| PATCH | `/v1/roles/{id}` | Aktualizacja |
| DELETE | `/v1/roles/{id}` | Usuniecie |

#### Kursy walut (admin)

| Metoda | Sciezka | Opis |
|--------|---------|------|
| GET | `/v1/exchange-rates` | Lista kursow |
| POST | `/v1/exchange-rates` | Dodanie |
| POST | `/v1/exchange-rates/fetch` | Pobranie z NBP |
| POST | `/v1/exchange-rates/convert` | Przeliczenie |
| GET/PATCH/DELETE | `/v1/exchange-rates/{id}` | CRUD |

#### Rate shopping (porownywanie stawek)

| Metoda | Sciezka | Opis |
|--------|---------|------|
| POST | `/v1/shipping/rates` | Porownanie stawek przewoznikow |

#### Statystyki

| Metoda | Sciezka | Opis |
|--------|---------|------|
| GET | `/v1/stats/dashboard` | KPI dashboardu |
| GET | `/v1/stats/products/top` | Top produkty |
| GET | `/v1/stats/revenue/by-source` | Przychody wg zrodla |
| GET | `/v1/stats/trends` | Trendy zamowien |
| GET | `/v1/stats/payment-methods` | Metody platnosci |

#### AI (wymaga klucza OpenAI)

| Metoda | Sciezka | Opis |
|--------|---------|------|
| POST | `/v1/ai/categorize` | Kategoryzacja produktu |
| POST | `/v1/ai/describe` | Generowanie opisu |
| POST | `/v1/ai/bulk-categorize` | Masowa kategoryzacja |
| POST | `/v1/ai/improve` | Ulepszanie opisu produktu |
| POST | `/v1/ai/translate` | Tlumaczenie produktu |

#### Marketing (admin)

| Metoda | Sciezka | Opis |
|--------|---------|------|
| POST | `/v1/marketing/sync` | Sync do Mailchimp |
| GET | `/v1/marketing/status` | Status sync |
| POST | `/v1/marketing/campaigns` | Tworzenie kampanii |

#### Inne

| Metoda | Sciezka | Opis |
|--------|---------|------|
| GET | `/v1/ws` | WebSocket (real-time) |
| GET | `/v1/barcode/{code}` | Lookup barcodu |
| POST | `/v1/uploads` | Upload pliku (10MB max) |
| GET | `/v1/inpost/points` | Wyszukiwanie paczkomatow |
| GET | `/v1/inpost/geowidget-token` | Token Geowidget InPost |
| GET | `/v1/users/me` | Aktualny user |
| GET/POST/PATCH/DELETE | `/v1/users/...` | CRUD userow (admin) |
| GET | `/v1/audit` | Dziennik audytu (admin) |
| GET | `/v1/webhooks` | Konfiguracja webhookow (admin) |
| GET | `/v1/webhook-deliveries` | Log dostaw webhookow (admin) |
| GET | `/v1/sync-jobs` | Logi synchronizacji (admin) |
| GET | `/v1/sync-jobs/{id}` | Szczegoly sync job |
| GET | `/v1/helpdesk/tickets` | Tickety Freshdesk |
| GET | `/v1/order-statuses` | Statusy zamowien (read-only) |
| GET | `/v1/custom-fields` | Pola niestandardowe (read-only) |
| GET | `/v1/product-categories` | Kategorie produktow (read-only) |
| POST | `/v1/webhooks/{provider}/{tenant_id}` | Webhook przychodzacy |
| POST | `/v1/webhooks/allegro` | Webhook Allegro (HMAC) |
| GET | `/health` | Health check (no version disclosed) |
| GET | `/metrics` | Prometheus metrics (requires Bearer token) |
| GET | `/v1/openapi.yaml` | Specyfikacja OpenAPI |
| GET | `/v1/docs` | Swagger UI |

---

## 6. Frontend Dashboard

### Mapa stron (81 stron)

#### Publiczne (bez logowania)

| Sciezka | Strona |
|---------|--------|
| `/login` | Formularz logowania (z obsluga 2FA) |
| `/register` | Rejestracja firmy |
| `/return-request` | Formularz zwrotu (klient) |
| `/return-request/[token]` | Status zwrotu (klient) |

#### Pulpit

| Sciezka | Strona |
|---------|--------|
| `/` | Dashboard -- KPI, wykresy, ostatnie zamowienia |

#### Sprzedaz

| Sciezka | Strona |
|---------|--------|
| `/orders` | Lista zamowien (filtrowanie, sortowanie, bulk actions, inline edit, Kanban) |
| `/orders/new` | Nowe zamowienie |
| `/orders/[id]` | Szczegoly zamowienia (timeline, przesylki, zwroty, faktury, notatki) |
| `/orders/import` | Import zamowien CSV |
| `/shipments` | Lista przesylek |
| `/shipments/new` | Nowa przesylka |
| `/shipments/[id]` | Szczegoly przesylki + etykieta |
| `/returns` | Lista zwrotow |
| `/returns/new` | Nowy zwrot |
| `/returns/[id]` | Szczegoly zwrotu |
| `/invoices` | Lista faktur (z obsluga KSeF) |
| `/invoices/[id]` | Szczegoly faktury + PDF + status KSeF |
| `/customers` | Lista klientow |
| `/customers/new` | Nowy klient |
| `/customers/[id]` | Profil klienta + historia zamowien |
| `/packing` | Stanowisko pakowania (barcode) |
| `/reports` | Raporty i analizy |

#### Katalog

| Sciezka | Strona |
|---------|--------|
| `/products` | Lista produktow (search, inline edit, AI kategoryzacja) |
| `/products/new` | Nowy produkt |
| `/products/[id]` | Szczegoly + bundles + AI opis/ulepszanie |
| `/products/[id]/variants` | Warianty produktu |
| `/products/[id]/listings` | Oferty na marketplace'ach (Allegro wizard) |
| `/products/import` | Import produktow CSV |

#### Inwentaryzacja

| Sciezka | Strona |
|---------|--------|
| `/stocktakes` | Lista inwentaryzacji |
| `/stocktakes/new` | Nowa inwentaryzacja |
| `/stocktakes/[id]` | Szczegoly + liczenie pozycji |

#### Integracje

| Sciezka | Strona |
|---------|--------|
| `/integrations` | Lista integracji |
| `/integrations/new` | Nowa integracja |
| `/integrations/[id]` | Ustawienia integracji |
| `/integrations/allegro` | Setup Allegro (OAuth) |
| `/integrations/allegro/messages` | Wiadomosci Allegro |
| `/integrations/allegro/returns` | Zwroty Allegro |
| `/integrations/allegro/offers` | Oferty Allegro |
| `/integrations/allegro/catalog` | Katalog Allegro |
| `/integrations/allegro/finance` | Finanse Allegro |
| `/integrations/allegro/promotions` | Promocje Allegro |
| `/integrations/allegro/disputes` | Spory Allegro |
| `/integrations/allegro/ratings` | Oceny Allegro |
| `/integrations/allegro/policies` | Polityki po-sprzedazowe |
| `/integrations/allegro/delivery` | Ustawienia dostawy Allegro |
| `/integrations/amazon` | Setup Amazon |
| `/suppliers` | Dostawcy |
| `/suppliers/new` | Nowy dostawca |
| `/suppliers/[id]` | Szczegoly dostawcy |

#### Ustawienia (admin)

| Sciezka | Strona |
|---------|--------|
| `/settings/company` | Dane firmy + logo |
| `/settings/users` | Zarzadzanie uzytkownikami |
| `/settings/roles` | Role RBAC |
| `/settings/roles/[id]` | Edycja roli |
| `/settings/security` | Bezpieczenstwo (2FA/TOTP) |
| `/settings/order-statuses` | Konfiguracja statusow |
| `/settings/custom-fields` | Pola niestandardowe |
| `/settings/notifications` | Powiadomienia (Email + SMS) |
| `/settings/email` | Ustawienia SMTP |
| `/settings/sms` | Ustawienia SMS |
| `/settings/webhooks` | Konfiguracja webhookow |
| `/settings/webhooks/deliveries` | Log dostaw |
| `/settings/invoicing` | Fakturowanie |
| `/settings/ksef` | KSeF e-Fakturowanie |
| `/settings/inventory` | Tryb scisly magazynu |
| `/settings/currencies` | Kursy walut |
| `/settings/print-templates` | Szablony druku |
| `/settings/product-categories` | Kategorie produktow |
| `/settings/price-lists` | Cenniki B2B |
| `/settings/price-lists/[id]` | Edycja cennika |
| `/settings/warehouses` | Magazyny |
| `/settings/warehouses/[id]` | Edycja magazynu |
| `/settings/warehouse-documents` | Dokumenty magazynowe |
| `/settings/warehouse-documents/new` | Nowy dokument |
| `/settings/warehouse-documents/[id]` | Edycja dokumentu |
| `/settings/automation` | Reguly automatyzacji |
| `/settings/automation/new` | Nowa regula |
| `/settings/automation/[id]` | Edycja reguly |
| `/settings/marketing` | Marketing (Mailchimp) |
| `/settings/helpdesk` | Helpdesk (Freshdesk) |
| `/settings/sync-jobs` | Historia synchronizacji |
| `/audit` | Dziennik aktywnosci |

### Nawigacja sidebar (grupy)

Sidebar jest zwijany (collapsible) i zapamietuje stan po stronach.

```
Pulpit (Dashboard)

--- Sprzedaz ---
  Zamowienia (z widokiem Kanban)
  Przesylki
  Zwroty
  Faktury
  Import
  Klienci
  Pakowanie
  Raporty

--- Katalog ---
  Produkty
  Import produktow
  Inwentaryzacja
  Kategorie
  Szablony druku

--- Ogolne (admin) ---
  Firma
  Uzytkownicy
  Role
  Bezpieczenstwo (2FA)

--- Sprzedaz - ustawienia (admin) ---
  Statusy zamowien
  Pola niestandardowe
  Cenniki
  Fakturowanie
  KSeF

--- Powiadomienia (admin) ---
  Powiadomienia
  Webhooki

--- Magazyn (admin) ---
  Magazyny
  Dokumenty magazynowe
  Tryb scisly

--- Integracje (admin) ---
  Integracje
  Allegro (podstrony)
  Automatyzacja
  Waluty
  Marketing
  Helpdesk
  Dostawcy

--- Monitoring (admin) ---
  Synchronizacja
  Dostawy webhookow
  Dziennik aktywnosci
```

### Kluczowe komponenty (81)

| Komponent | Opis |
|-----------|------|
| `DataTable` | Generyczna tabela z sortowaniem, paginacja, selekcja, inline edit |
| `CommandPalette` | Cmd+K -- szybkie wyszukiwanie i nawigacja |
| `StatusTransitionDialog` | Dialog zmiany statusu (zamowienia, przesylki) |
| `PaczkomatSelector` | Wybor paczkomatu InPost (mapa/search/inline) |
| `OrderForm` | Formularz zamowienia (klient, pozycje, adres, custom fields) |
| `OrderKanbanBoard` | Widok Kanban zamowien z drag & drop |
| `AllegroListingWizard` | 4-krokowy kreator oferty Allegro |
| `StocktakeCounter` | Interfejs liczenia pozycji inwentaryzacji |
| `RateShoppingCard` | Porownywanie stawek przewoznikow |
| `ProductImportPreview` | Podglad importu CSV z mapowaniem kolumn |
| `TagInput` | Multi-select tagow |
| `AdminGuard` | Wrapper wymuszajacy role admin |
| `ErrorBoundary` | Obsluga bledow z fallback UI |
| `EditableCell` | Edycja inline w tabeli |
| `TOTPSetupDialog` | Dialog konfiguracji 2FA z QR code |
| `KSeFStatusBadge` | Status wysylki faktury do KSeF |
| `CollapsibleSidebar` | Zwijany sidebar z zapamietywaniem stanu |
| `TypographySettings` | Ustawienia czcionki i gestosci interfejsu |
| `DensitySelector` | Wybor gestosci tabeli (compact/normal/spacious) |
| `PriorityBadge` | Odznaka priorytetu zamowienia |
| `InternalNotesEditor` | Edytor notatek wewnetrznych zamowienia |

### State management

```
+-------------+     +--------------+     +--------------+
|   Zustand    |     | React Query  |     |  API Client  |
|  (auth store)|     | (data cache) |     | (fetch+JWT)  |
|              |     |              |     |              |
| token        |     | useOrders()  |---->| GET /orders  |
| user         |     | useProducts()|---->| GET /products|
| tenant       |     | useDashboard |---->| GET /stats   |
| isAuth       |     | ...45 hooks  |     |              |
+-------------+     +--------------+     +------+-------+
                                                 |
                                          Auto-refresh
                                          on 401 (mutex)
```

---

## 7. Pakiety SDK (21)

### Order Engine (packages/order-engine/)

Standalone maszyna stanow zamowien i przesylek:

```
                  +---------+
          +------>|confirmed|----------+
          |       +----+----+          |
          |            |               |
     +----+---+  +-----v-----+        |
     |  new   |  | processing|        |
     +----+---+  +-----+-----+        |
          |            |               |
          |       +----v------+        |
          |       |ready_to   |        |
          |       |ship       |        |
          |       +----+------+        |
          |            |               v
          |       +----v----+    +----------+
          +------>| shipped |--->|cancelled |
          |       +----+----+    +----------+
          |            |               |
          |       +----v------+        |
          |       |in_transit |        |
          |       +----+------+        |
          |            |               |
          |       +----v----------+    |
          |       |out_for_       |    |
          |       |delivery       |    |
          |       +----+----------+    |
          |            |               |
          |       +----v-----+         |
          |       |delivered |         |
          |       +----+-----+         |
          |            |               |
          |       +----v-----+   +-----v----+
          |       |completed |--->| refunded |
          |       +----------+   +----------+
          |                            ^
          +----------------------------+
                    (via on_hold)
```

### Marketplace SDK-i

| SDK | Provider | Auth | Glowne operacje |
|-----|----------|------|----------------|
| allegro-go-sdk | Allegro.pl | OAuth 2.0 | Zamowienia, oferty, eventy, katalog |
| amazon-sp-sdk | Amazon | AWS Signing | Zamowienia, inventory, pricing |
| woocommerce-go-sdk | WooCommerce | REST API | Zamowienia, produkty, webhooks |
| ebay-go-sdk | eBay | OAuth 2.0 | Zamowienia, inventory |
| kaufland-go-sdk | Kaufland | Feed API | Import CSV/XML |
| olx-go-sdk | OLX | REST | Ogloszenia |
| mirakl-go-sdk | Mirakl/Empik | REST | Seller network |
| erli-go-sdk | Erli | REST | Zamowienia, oferty |

### Carrier SDK-i

| SDK | Provider | Auth | Glowne operacje |
|-----|----------|------|----------------|
| inpost-go-sdk | InPost | Bearer | Paczki, etykiety, tracking, paczkomaty, webhooks |
| dhl-go-sdk | DHL | API Key | Przesylki, etykiety, tracking |
| dpd-go-sdk | DPD | REST | Przesylki (Polska) |
| gls-go-sdk | GLS | API | Przesylki (Europa) |
| ups-go-sdk | UPS | XML/REST | Miedzynarodowe |
| poczta-polska-go-sdk | Poczta Polska | REST | Paczki pocztowe |
| orlen-paczka-go-sdk | Orlen Paczka | REST | Paczkomaty Orlen |
| fedex-go-sdk | FedEx | REST | Miedzynarodowe |

### Inne SDK-i

| SDK | Provider | Cel |
|-----|----------|-----|
| fakturownia-go-sdk | Fakturownia | Faktury |
| ksef-go-sdk | KSeF | Krajowy System e-Faktur |
| smsapi-go-sdk | SMSAPI | Powiadomienia SMS |
| iof-parser | IOF/CSV | Parser feedow dostawcow |

---

## 8. Bezpieczenstwo

### Autentykacja JWT Ed25519

```
JWT_SECRET (env)
    |
    v SHA-512 hash
    |
    v Pierwsze 32 bajty = Ed25519 seed
    |
    v Generowanie pary kluczy
    |
+---------------+     +---------------+
| Private Key   |     | Public Key    |
| (signing)     |     | (verify)      |
+---------------+     +---------------+
```

**Tokeny:**

| Typ | Czas zycia | Uzycie |
|-----|-----------|--------|
| Access Token | 1 godzina | Header `Authorization: Bearer ...` |
| Refresh Token | 30 dni | Cookie httpOnly (sciezka /v1/auth) |

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

### Dwuskladnikowe uwierzytelnianie (2FA/TOTP)

```
Uzytkownik wlacza 2FA:
    POST /v1/auth/2fa/setup -> sekret TOTP + QR code URL
    POST /v1/auth/2fa/verify -> weryfikacja kodu, wlaczenie 2FA

Logowanie z 2FA:
    POST /v1/auth/login -> 200 { requires_2fa: true, temp_token: "..." }
    POST /v1/auth/2fa/login -> { temp_token, code } -> access + refresh token
```

Sekret TOTP szyfrowany w kolumnie `users.totp_secret`. Kompatybilny z Google Authenticator, Authy i innymi aplikacjami TOTP.

### Szyfrowanie AES-256-GCM

Credentials integracji szyfrowane w bazie:
```
Plaintext -> AES-256-GCM(key, random_nonce) -> Base64 -> DB
DB -> Base64 decode -> AES-256-GCM decrypt -> Plaintext
```

Klucz: `ENCRYPTION_KEY` (64-char hex = 32 bajty)

### Hasla -- bcrypt (cost 12)

```
password -> bcrypt(cost=12) -> $2a$12$... -> DB
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

| Zagrozenie | Mitygacja |
|-----------|-----------|
| SQL Injection | Parametryzowane zapytania (pgx driver) |
| XSS | Sanityzacja HTML w inputach (strip tags) + CSP header |
| CSRF | SameSite cookies + CORS whitelist |
| Clickjacking | X-Frame-Options: DENY + CSP frame-ancestors 'none' |
| Tenant leakage | RLS + FORCE ROW LEVEL SECURITY |
| Token theft | SHA-256 hash w blacklist, httpOnly cookies |
| SSRF | Webhook dispatcher sprawdza private IP ranges |
| Brute force | Rate limiting (10/min login, 60/min refresh, 30/min public) |
| DoS | Max body size (1MB default, 10MB upload) |
| Account takeover | 2FA/TOTP, bcrypt, Ed25519 JWT |
| Info disclosure | Brak wersji w /health, brak X-Powered-By, /metrics chroniony tokenem |
| MIME sniffing | X-Content-Type-Options: nosniff |
| Referrer leak | Referrer-Policy: strict-origin-when-cross-origin |

### Bezpieczenstwo infrastruktury (Kubernetes)

| Warstwa | Mechanizm |
|---------|-----------|
| Secrets encryption at rest | AES-CBC w k3s (EncryptionConfiguration) |
| K8s audit logging | Audit policy z logowaniem zmian w secrets, RBAC, write ops |
| Pod Security Standards | PSS enforce: restricted (apps), baseline (system), privileged (storage) |
| NetworkPolicies | Default-deny ingress na wszystkich 15 namespacach |
| State DB permissions | chmod 600 (wylacznie root) |
| TLS | Mutual TLS do API servera k3s, TLS 1.2+ z strong cipher suites |
| Image scanning | Trivy CRITICAL+HIGH w CI/CD pipeline |
| Vulnerability scanning | govulncheck (Go) + npm audit (frontend) w CI |

---

## 9. Kluczowe flow

### Flow 1: Logowanie (z 2FA)

```
Uzytkownik                Dashboard              API Server              DB
    |                        |                       |                    |
    |  email + password      |                       |                    |
    |  + tenant_slug         |                       |                    |
    |----------------------->|                       |                    |
    |                        |  POST /v1/auth/login  |                    |
    |                        |---------------------->|                    |
    |                        |                       |  find_tenant_by_slug
    |                        |                       |------------------>|
    |                        |                       |<------------------|
    |                        |                       |  find_user_for_auth
    |                        |                       |------------------>|
    |                        |                       |<------------------|
    |                        |                       |  bcrypt.Compare()  |
    |                        |                       |  check totp_enabled|
    |                        |  {requires_2fa: true, |                    |
    |                        |   temp_token}         |                    |
    |                        |<----------------------|                    |
    |  Pokaz pole TOTP       |                       |                    |
    |<-----------------------|                       |                    |
    |  kod TOTP              |                       |                    |
    |----------------------->|                       |                    |
    |                        | POST /v1/auth/2fa/login                   |
    |                        |---------------------->|                    |
    |                        |                       |  TOTP.Validate()   |
    |                        |                       |  Ed25519 sign JWT  |
    |                        |  {access_token,       |                    |
    |                        |   user, tenant}       |                    |
    |                        |<----------------------|                    |
    |  Zustand: setAuth()    |                       |                    |
    |  Cookie: has_session=1 |                       |                    |
    |<-----------------------|                       |                    |
    |  Redirect -> /         |                       |                    |
```

### Flow 2: Cykl zycia zamowienia

```
+----------+     +-----------+     +-----------+     +-----------+
|  NEW     |---->| CONFIRMED |---->|PROCESSING |---->|READY TO   |
|          |     |           |     |           |     |SHIP       |
+----------+     +-----------+     +-----------+     +-----+-----+
                                                           |
                  Kazda zmiana statusu:                    |
                  +- Audit log                             |
                  +- Webhook dispatch                      |
                  +- Email/SMS klientowi                   v
                  +- Automation rules              +-----------+
                  +- Delayed actions (opcja)       | SHIPPED   |
                  +- WebSocket broadcast           +-----+-----+
                                                         |
+----------+     +-----------+     +-----------+        |
|COMPLETED |<----|DELIVERED  |<----|IN TRANSIT |<-------+
|          |     |           |     |           |
+----+-----+     +-----------+     +-----------+
     |
     v
+----------+     +-----------+
| REFUNDED |<----| CANCELLED |
|(terminal)|     |           |
+----------+     +-----------+
```

### Flow 3: Webhook dispatch

```
Event (np. order.confirmed)
    |
    v
WebhookDispatchService.Dispatch()
    |
    +- Zaladuj endpoints z tenant settings
    |
    +- Dla kazdego endpointu:
    |     |
    |     +- Serializuj payload -> JSON
    |     +- HMAC-SHA256(payload, endpoint.secret) -> signature
    |     +- Sprawdz SSRF (resolve DNS -> odrzuc private IP)
    |     +- POST endpoint.url
    |     |    Headers: X-Signature-256, X-OpenOMS-Event, X-Delivery-ID
    |     +- Zapisz wynik w webhook_deliveries
    |
    +- WebSocket broadcast do tenanta
```

### Flow 4: Automatyzacja (z opoznionymi akcjami)

```
Event "order.created"
    |
    v
AutomationEngine.ProcessEvent() [async]
    |
    +- Zaladuj reguly WHERE trigger = "order.created" AND enabled
    |
    +- Dla kazdej reguly (wg priority):
    |     |
    |     +- Ewaluuj warunki:
    |     |    total_amount >= 500? ok
    |     |    tags contains "bulk"? ok
    |     |
    |     +- Jesli wszystkie spelnione:
    |     |    +- transition_status -> "confirmed"
    |     |    +- send_email -> sales@company.com
    |     |    +- add_tag -> "auto-confirmed"
    |     |    +- delay(30m) -> create_shipment  <- OPOZNIENIE!
    |     |         |
    |     |         v
    |     |    Zapis w automation_delayed_actions (execute_at = NOW() + 30m)
    |     |         |
    |     |         v
    |     |    DelayedActionWorker (co 30s) -> execute_at <= NOW() -> wykonaj akcje
    |     |
    |     +- Zapisz log w automation_rule_logs
    |
    +- Zaktualizuj rule.fire_count, rule.last_fired_at
```

### Flow 5: Generowanie etykiety

```
Uzytkownik klika "Generuj etykiete"
    |
    v
POST /v1/shipments/{id}/label
    |
    v
LabelService.GenerateLabel()
    |
    +- Zaladuj shipment + order
    +- Zaladuj integration (credentials)
    +- Odszyfruj credentials (AES-256-GCM)
    +- Utworz CarrierProvider (np. InPost)
    +- provider.CreateShipment(request)
    |     +- POST do InPost API
    |        -> tracking_number, label_url
    +- Zapisz w shipment record
    +- Pobierz PDF etykiety
    +- Zapisz w storage (S3 lub local)
    +- Zwroc label URL
```

### Flow 6: Inwentaryzacja (stocktake)

```
Admin tworzy inwentaryzacje
    |
    v
POST /v1/stocktakes { warehouse_id, name }
    |  -> status: "draft", laduje produkty jako stocktake_items
    v
POST /v1/stocktakes/{id}/start
    |  -> status: "in_progress"
    v
POST /v1/stocktakes/{id}/items/{iid}/count { counted_quantity }
    |  -> zapisuje policzona ilosc, oblicza roznice
    |  (powtarzane dla kazdej pozycji)
    v
POST /v1/stocktakes/{id}/complete
    |  -> status: "completed"
    |  -> aktualizuje warehouse_stock (jesli roznice)
    |  -> generuje raport roznic
    v
Gotowe -- stany magazynowe zaktualizowane
```

### Flow 7: Tworzenie oferty Allegro (wizard)

```
Krok 1: Wybor kategorii
    |  GET /v1/integrations/allegro/categories/search
    |  GET /v1/integrations/allegro/categories/{cid}/parameters
    v
Krok 2: Parametry oferty
    |  Uzupelnienie atrybutow wymaganych przez kategorie
    v
Krok 3: Cena, dostawa, polityki
    |  GET /v1/integrations/allegro/shipping-rates
    |  GET /v1/integrations/allegro/return-policies
    |  GET /v1/integrations/allegro/warranties
    v
Krok 4: Podsumowanie i publikacja
    |  POST /v1/products/{pid}/listings/allegro
    v
Oferta opublikowana na Allegro
```

---

## 10. Integracje

### Marketplace -- flow synchronizacji

```
                    +--------------+
                    |  Marketplace |
                    | (Allegro,    |
                    |  Amazon...)  |
                    +------+-------+
                           |
              Polling co 45s (Worker)
                           |
                           v
+----------------------------------------------+
|            MarketplaceProvider                |
|                                              |
|  interface {                                 |
|    PollOrders(ctx, cursor) -> orders         |
|    GetOrder(ctx, externalID) -> order        |
|    PushOffer(ctx, product) -> externalID     |
|    UpdateStock(ctx, offerID, qty)            |
|    UpdatePrice(ctx, offerID, price)          |
|  }                                           |
+----------------------------------------------+
                           |
                           v
                    +--------------+
                    |   OpenOMS    |
                    |   Orders     |
                    +--------------+
```

### Carrier -- flow wysylki

```
                    +--------------+
                    |   Carrier    |
                    | (InPost,     |
                    |  DHL...)     |
                    +------+-------+
                           |
              Label + Tracking API
                           |
                           v
+----------------------------------------------+
|              CarrierProvider                  |
|                                              |
|  interface {                                 |
|    CreateShipment(ctx, req) -> response      |
|    GetLabel(ctx, id, format) -> PDF          |
|    GetTracking(ctx, tracking#) -> events     |
|    CancelShipment(ctx, id)                   |
|    SupportsPickupPoints() -> bool            |
|    SearchPickupPoints(ctx, query) -> points  |
|    GetRates(ctx, req) -> rates               |
|  }                                           |
+----------------------------------------------+
```

### Obslugiwane integracje

| Kategoria | Provider | Status |
|-----------|----------|--------|
| **Marketplace** | Allegro | OAuth 2.0, polling, oferty, katalog, messaging, zwroty, spory, oceny, promocje, dostawa |
| | Amazon | SP-API, polling |
| | WooCommerce | REST API, webhooks |
| | eBay | OAuth 2.0 |
| | Kaufland | Feed API |
| | OLX | REST API |
| | Mirakl/Empik | REST API |
| | Erli | REST API |
| **Carrier** | InPost | Paczkomaty, kurier, Geowidget |
| | DHL | Miedzynarodowe |
| | DPD | Polska |
| | GLS | Europa |
| | UPS | Miedzynarodowe |
| | Poczta Polska | Paczki |
| | Orlen Paczka | Paczkomaty |
| | FedEx | Miedzynarodowe |
| **Fakturowanie** | Fakturownia | Faktury VAT |
| **e-Fakturowanie** | KSeF | Krajowy System e-Faktur (wysylka, UPO, status) |
| **Marketing** | Mailchimp | Sync klientow, kampanie |
| **Helpdesk** | Freshdesk | Tickety |
| **Powiadomienia** | SMTP | Email |
| | Twilio/SMSAPI | SMS |
| **AI** | OpenAI | Kategoryzacja, opisy, ulepszanie, tlumaczenie |
| **Kursy walut** | NBP | Narodowy Bank Polski |

---

## 11. Background Workers (14 plikow)

### Workery (10 zarejestrowanych)

| Worker | Interwal | Cel |
|--------|----------|-----|
| AllegroOrderPoller | 45s | Polling zamowien z Allegro |
| AmazonOrderPoller | 45s | Polling zamowien z Amazon |
| WooCommerceOrderPoller | 45s | Polling zamowien z WooCommerce |
| TrackingPoller | 5min | Aktualizacja statusu przesylek |
| StockSyncWorker | konfigurowalny | Sync stanow magazynowych do marketplace'ow |
| SupplierSyncWorker | konfigurowalny | Sync katalogow dostawcow (IOF/CSV) |
| ExchangeRateWorker | 1/dzien | Pobranie kursow z NBP |
| OAuthRefresher | 1/dzien | Odswiezenie tokenow OAuth (Allegro, Amazon) |
| KSeFStatusWorker | 5min | Sprawdzanie statusu faktur wyslanych do KSeF |
| DelayedActionWorker | 30s | Wykonywanie opoznionych akcji automatyzacji |

### Infrastruktura workerow

| Plik | Cel |
|------|-----|
| `manager.go` | Menedzer workerow (rejestracja, start, stop, graceful shutdown) |
| `marketplace_order_poller.go` | Bazowy poller zamowien (wspolna logika dla Allegro/Amazon/WooCommerce) |
| `tenant_iterator.go` | Iterator tenantow -- wykonuje logike per-tenant |
| `distributed_lock.go` | Blokada rozproszona (SETNX) dla multi-instance |

### Cechy

- Panic recovery (safeRun wrapper)
- Graceful shutdown (context cancellation)
- Logowanie bledow per worker (slog)
- Interfejs Worker: `Name()`, `Interval()`, `Run(ctx)`
- Iteracja per-tenant (kazdy worker dziala dla wszystkich aktywnych tenantow)

---

## 12. Automatyzacja

### Trigger events

| Event | Kiedy |
|-------|-------|
| `order.created` | Nowe zamowienie |
| `order.status_changed` | Zmiana statusu zamowienia |
| `order.confirmed` | Zamowienie potwierdzone |
| `shipment.created` | Nowa przesylka |
| `shipment.status_changed` | Zmiana statusu przesylki |
| `return.created` | Nowy zwrot |
| `return.status_changed` | Zmiana statusu zwrotu |
| `product.stock_low` | Niski stan magazynowy |

### Warunki (conditions)

```json
[
  { "field": "total_amount", "operator": "gte", "value": 500 },
  { "field": "tags", "operator": "contains", "value": "vip" },
  { "field": "source", "operator": "eq", "value": "allegro" },
  { "field": "priority", "operator": "eq", "value": "high" }
]
```

Operatory: `eq`, `neq`, `gt`, `gte`, `lt`, `lte`, `contains`, `not_contains`, `in`, `not_in`

### Akcje (actions)

| Typ akcji | Opis |
|-----------|------|
| `transition_status` | Zmiana statusu zamowienia/przesylki |
| `send_email` | Wyslanie emaila |
| `send_sms` | Wyslanie SMS |
| `add_tag` | Dodanie tagu |
| `remove_tag` | Usuniecie tagu |
| `set_field` | Ustawienie pola |
| `create_shipment` | Auto-tworzenie przesylki |
| `webhook` | Wywolanie custom webhook |
| `delay` | Opoznienie nastepnych akcji (np. 30m, 2h, 1d) |

### Opoznione akcje (delayed actions)

Akcja `delay` w regule automatyzacji tworzy wpis w tabeli `automation_delayed_actions` z polem `execute_at`. Worker `DelayedActionWorker` co 30 sekund sprawdza, czy sa akcje do wykonania i je realizuje. Pozwala to na scenariusze typu:

- "Jesli zamowienie nie zostalo wyslane w ciagu 24h, wyslij przypomnienie"
- "Po potwierdzeniu zamowienia, po 30 minutach automatycznie utworz przesylke"

### Przyklad reguly

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
      "subject": "Nowe zamowienie VIP",
      "template": "vip_alert"
    }},
    { "type": "delay", "params": { "duration": "30m" } },
    { "type": "create_shipment", "params": { "carrier": "inpost" } }
  ]
}
```

---

## 13. Konfiguracja

### Zmienne srodowiskowe

```bash
# -- Serwer -----------------------
PORT=8080
ENV=production|development

# -- Baza danych ------------------
DATABASE_URL=postgres://openoms:pass@localhost:5433/openoms

# -- Bezpieczenstwo ---------------
JWT_SECRET=minimum-32-znaki-losowy-string
ENCRYPTION_KEY=64-znakowy-hex-string

# -- Storage ----------------------
STORAGE_TYPE=s3|local
UPLOAD_DIR=./uploads
MAX_UPLOAD_SIZE=10485760
BASE_URL=https://api.firma.pl

# -- S3 ---------------------------
S3_REGION=eu-central-1
S3_BUCKET=openoms-uploads
S3_ENDPOINT=https://s3.example.com
S3_ACCESS_KEY=...
S3_SECRET_KEY=...
S3_PUBLIC_URL=https://cdn.firma.pl

# -- Frontend ---------------------
FRONTEND_URL=https://app.firma.pl
NEXT_PUBLIC_API_URL=http://localhost:8080

# -- Workers ----------------------
WORKERS_ENABLED=true

# -- Monitoring -------------------
METRICS_TOKEN=...                    # Bearer token dla /metrics (openssl rand -base64 32)

# -- Integracje (opcjonalne) ------
INPOST_API_TOKEN=...
INPOST_ORG_ID=...
ALLEGRO_WEBHOOK_SECRET=...
OPENAI_API_KEY=...
OPENAI_MODEL=gpt-4

# -- KSeF (opcjonalne) ------------
# Konfiguracja w Settings -> KSeF (per tenant, w JSONB settings)
```

### Seed data (testowe)

| Tenant | Slug | Branza | Owner |
|--------|------|--------|-------|
| MercPart | mercpart | Czesci samochodowe | rafal@mercpart.pl |
| ElektroMax | elektromax | Elektronika | jan@elektromax.pl |
| ZielonyOgrod | zielonyogrod | Ogrodnictwo | maria@zielonyogrod.pl |

Haslo testowe: `password123`

---

## 14. Statystyki projektu

| Metryka | Wartosc |
|---------|--------|
| **Pliki Go** | 386 (w tym 71 testow) |
| **Pliki TypeScript/TSX** | 234 |
| **Tabele DB** | 32 |
| **Migracje SQL** | 46 (000001-000046) |
| **Endpointy API** | ~296 |
| **Strony frontend** | 81 |
| **Komponenty React** | 81 |
| **Custom hooks** | 45 |
| **Handlery Go** | 57 |
| **Serwisy Go** | 38 |
| **Repozytoria Go** | 28 |
| **Background workers** | 14 (10 zarejestrowanych + 4 infra) |
| **Middleware** | 12 |
| **Pakiety SDK** | 21 |
| **Testy E2E** | 12 specow Playwright |
| **Jezyki** | Go, TypeScript, SQL |
| **Licencja** | AGPLv3 (apps) + MIT (packages) |

### Testy

| Typ testu | Status |
|-----------|--------|
| E2E Playwright (12 specow) | PASS |
| Backend integration | PASS |
| API contract (TS <-> Go) | PASS |
| Load testing | 0 bledow, 1000-1800 req/s |
| RLS isolation | PASS |
| Clean migration | PASS |

---

*Dokument wygenerowany: 2026-02-13*
*Wersja: OpenOMS v3.0*
