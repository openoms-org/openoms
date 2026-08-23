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
- **API REST** -- ~540 endpointow z OpenAPI 3.1
- **Dashboard** -- Next.js 16 + React 19, 139 stron, dark mode, PWA
- **AI** -- auto-kategoryzacja, opis, ulepszanie i tlumaczenie produktow (OpenAI)
- **Inwentaryzacja** -- pelny cykl zycia stocktake z liczeniem pozycji
- **Rate shopping** -- porownywanie stawek przewoznikow
- **Marketplace Listings** -- dynamiczny picker marketplace'ow + wizardy per marketplace (Allegro 4-krokowy, eBay 3-krokowy, OLX z drzewem kategorii, WooCommerce, Erli)
- **Kanban board** -- widok zamowien w formie tablicy Kanban
- **Import CSV** -- import produktow i zamowien z podgladem
- **Command Palette** -- Cmd+K do szybkiej nawigacji i wyszukiwania
- **Dostawcy (dropship)** -- import katalogow XML/IOF, hybrid sync (XML+API), auto-submit zamowien, wizardy konfiguracji
- **Hierarchiczne kategorie** -- drzewo kategorii produktow z mapowaniem kategorii dostawcow
- **Token refresh rotation** -- rotacja refresh tokenow z detekcja ponownego uzycia
- **Billing/Stripe** -- integracja platnosci (Stripe Checkout, subskrypcje, webhooks), plany konfigurowane runtime
- **Onboarding wizard** -- 4-krokowy kreator konfiguracji dla nowych firm
- **License tokens** -- Ed25519 JWT z ochrona przed powtorzeniem (enterprise)

### Licencja

- `apps/` -- Elastic License 2.0 (core)
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
|  | 89 tabel |  |  RLS Policy  |  |  SECURITY DEFINER     |  |
|  | 50 migr. |  | (per tenant) |  |  (auth bypass)        |  |
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
    USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);
```

Tabele tenant-scoped maja wlaczone `FORCE ROW LEVEL SECURITY`, a CI sprawdza baze po migracjach pod katem regresji w politykach RLS.

### CI/CD i Deployment

#### Pipeline

1. Push do `main` -> `release.yml`:
   - Buduje 3 obrazy Docker -> GHCR
   - `ghcr.io/openoms-org/openoms-api`
   - `ghcr.io/openoms-org/openoms-dashboard`
   - `ghcr.io/openoms-org/openoms-migrate`
   - Skanuje obrazy (Trivy, CRITICAL+HIGH)
   - Generuje i waliduje CycloneDX SBOM dla kazdego obrazu, zapisujac zbiorczy artefakt `openoms-sbom-<sha>`
   - Opcjonalnie: wysyla `repository_dispatch` do prywatnego repo deploymentu (patrz komentarz w `release.yml`)

2. Deployment (blue-green z Argo Rollouts):
   - Uzyj Helm chart z `deploy/helm/openoms/` + wlasny `values-production.yaml`
   - `helm upgrade --install openoms ./deploy/helm/openoms -f values-production.yaml`
   - API i Dashboard deploya sa jako Argo Rollouts z blue-green strategia
   - Pre-promotion: `AnalysisTemplate` uruchamia smoke tests (health check + 5 kluczowych endpointow); job smoke-test jest zgodny z PSS `restricted` (`runAsNonRoot`, drop ALL capabilities, read-only root filesystem, seccomp RuntimeDefault)
   - Preview service (`openoms-api-preview`, `openoms-dashboard-preview`) do weryfikacji przed przelaczeniem
   - Automatyczny rollback jesli smoke tests nie przejda
   - Albo uzyj `docker-compose.prod.yml` dla prostszych setupow (bez Argo)

#### Helm Chart

- Chart: `deploy/helm/openoms/`
- Domyslne wartosci: `values.yaml` (generyczne, example.com)
- Produkcyjne wartosci: utworz wlasny `values-production.yaml` z domenami i sekretami
- Migration job: pre-upgrade hook, `activeDeadlineSeconds: 600`
- Worker deployment uzywa osobnych `worker.podSecurityContext` i `worker.securityContext`, zeby produkcyjny overlay mogl twardo konfigurowac hardening workerow niezaleznie od API.
- Daily backup CronJob zapisuje dump SQL do pliku, kompresuje go osobno, waliduje `gzip -t` i minimalny rozmiar, a dopiero potem wysyla plik do S3-compatible storage przez MinIO client (`mcli`).

#### Obrazy Docker

| Obraz | Dockerfile | Zawartosc |
|-------|-----------|-----------|
| `openoms-api` | `apps/api-server/Dockerfile` | Go binary (distroless) |
| `openoms-dashboard` | `apps/dashboard/Dockerfile` | Next.js standalone (distroless Node 22 Debian 13 runtime, non-root, bez shella) |
| `openoms-migrate` | `deploy/Dockerfile.migrate` | golang-migrate + SQL |

Obrazy sa publiczne na GHCR -- nie wymagaja `imagePullSecrets`.
Dashboard runtime `FROM` jest pinowany do digestu indeksu `gcr.io/distroless/nodejs22-debian13:nonroot@sha256:22d2f0480e59548ad14cf10d8921b24ef809780e7a61b162838f3d15a4a92e3d`, ktory dostarcza `libssl3t64` `>= 3.5.6-1~deb13u2` (CVE-2026-45447 / DSA-6335-1); Renovate aktualizuje digest.
Dashboard image jest walidowany w CI/release przez `scripts/check-dashboard-image.sh`: runtime musi startowac jako non-root Node/distroless, nie moze zawierac `/bin/sh`, placeholderow runtime (`NEXT_PUBLIC_API_URL_PLACEHOLDER`, `WS_CSP_HOST_PLACEHOLDER`, `SENTRY_DSN_PLACEHOLDER`) ani `http://localhost:8080` w bundle, i musi przejsc read-only smoke test `/login`.
Sentry source map upload dla dashboardu uzywa BuildKit secret mount (`sentry_auth_token`) tylko podczas `next build`; `SENTRY_AUTH_TOKEN` nie moze wracac jako Docker `ARG`, `ENV` ani release `build-args`. Production release dashboardu ma dodatkowy preflight `scripts/check-dashboard-release-config.sh`, ktory wymaga prawdziwego `NEXT_PUBLIC_SENTRY_DSN`, `SENTRY_ORG`, `SENTRY_PROJECT` i `SENTRY_AUTH_TOKEN`.
Sentry release identity jest spinane z tagiem obrazu/SHA commita przez Helm `sentry.release` i runtime env `SENTRY_RELEASE` dla API, workerow oraz dashboardu. Go API przekazuje ten identyfikator do `sentry-go` jako `ClientOptions.Release`. Worker deployment dostaje `SENTRY_DSN` z tego samego `openoms-secrets` co API, zeby panic recovery i worker-level captures trafialy do projektu `openoms-api`.
API Sentry request context jest scrubowany przed przypieciem do scope oraz ponownie w `BeforeSend`: naglowki auth/cookie/CSRF/API-key, cookies, body, remote address env oraz token/email-like query values nie sa wysylane do Sentry. Failed-login audit entries nie zapisuja plaintext emaila w `audit_log.changes`; zostaje tylko nie-PII marker powodu.
Release pipeline instaluje przypieta wersje Syft z weryfikacja checksumu release, generuje CycloneDX JSON SBOM-y dla obrazow `openoms-api`, `openoms-dashboard` i `openoms-migrate`, zapisuje digest kazdego wypchnietego obrazu oraz publikuje zweryfikowany artefakt `openoms-sbom-<sha>`. Dashboard SBOM wyklucza metadane Next.js `node_modules/next/dist/compiled/**/package.json`, poniewaz opisuja wewnetrzne bundlowane pakiety bez wersji. Guard release blokuje dashboard SBOM, jesli zawiera komponenty npm z pusta albo `UNKNOWN` wersja. Prywatny import do systemu monitorowania podatnosci i alert routing nalezy do warstwy enterprise.
Deploy produkcyjny i powiazane artefakty release sa pinowane do niezmiennego SHA commita. Promocja obrazow do tagu `:latest` jest best-effort convenience dla ludzi i narzedzi pomocniczych; jej blad powinien zostac zalogowany jako warning, ale nie moze blokowac dispatcha deployu opartego o tagi SHA.
Publiczne workflowy GitHub Actions pinują zewnetrzne akcje do pelnych commit SHA zamiast mutowalnych tagow semver. `scripts/validate-github-actions-pinning.sh` jest uruchamiany lokalnie przez `scripts/local-ci.sh` oraz w publicznym CI, zeby blokowac regresje do tagow typu `@v4`/`@v7`.

#### Konfiguracja produkcyjna

Sekrety (hasla DB, JWT, klucze API) sa wstrzykiwane przez K8s Secrets w runtime -- nie sa w obrazach Docker.

Domyslna liczba replik: 1 (API, Dashboard, Worker). Skalowanie przez `replicaCount` w values overlay.

---

## 3. Stos technologiczny

### Backend

| Komponent | Technologia | Wersja |
|-----------|------------|--------|
| Jezyk | Go | 1.25 |
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
| Framework | Next.js | 16.2.6 |
| UI Library | React | 19.2.6 |
| Komponenty | shadcn/ui + Radix UI | latest |
| Styl | Tailwind CSS | v4 |
| Walidacja | Zod | v4 |
| Formularze | react-hook-form | 7.x |
| State | Zustand | 5.x |
| Data fetching | TanStack React Query | 5.x |
| Wykresy | Recharts | 3.x |
| Ikony | Lucide React | 0.563 |
| Testy E2E | Playwright | 1.60 |
| Testy jednostkowe | Vitest + Testing Library | 4.x |

### Monorepo

```
OpenOMS/
+-- apps/
|   +-- api-server/          <- Go backend (ELv2)
|   |   +-- cmd/server/      <- punkt wejscia
|   |   +-- internal/        <- logika aplikacji (105 handlerow, 104 serwisy, 63 repozytoria)
|   |   +-- migrations/      <- 50 wersji SQL (000001-000050)
|   +-- dashboard/           <- Next.js frontend (ELv2)
|       +-- src/app/         <- 139 stron (App Router)
|       +-- src/components/  <- 151 komponentow React (prod .tsx)
|       +-- src/hooks/       <- 83 custom hooks
|       +-- src/lib/         <- utils, API client, auth
|       +-- e2e/             <- 26 specow E2E Playwright
+-- packages/                <- SDK-i (MIT)
|   +-- order-engine/        <- maszyna stanow zamowien (uzywana przez shipmenty)
|   +-- allegro-go-sdk/      <- Allegro REST API
|   +-- inpost-go-sdk/       <- InPost ShipX API
|   +-- ksef-go-sdk/         <- KSeF e-Faktur API
|   +-- ...                  <- 28 pakietow SDK
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

### Tabele (89 lacznie; ponizej rdzen)

| Tabela | Cel | Kluczowe kolumny |
|--------|-----|-----------------|
| `tenants` | Konta firm | name, slug, plan, settings JSONB |
| `users` | Uzytkownicy | email, name, role, role_id, password_hash, totp_secret, totp_enabled |
| `roles` | Role RBAC | name, permissions TEXT[], is_system |
| `orders` | Zamowienia | status, items JSONB, total_amount, tags[], custom_fields, priority, internal_notes; unikalne `(tenant_id, source, external_id)` dla niepustych external_id |
| `shipments` | Przesylki | carrier, tracking_number, label_url, status, warehouse_id |
| `returns` | Zwroty/RMA | status, reason, refund_amount, return_token, customer_email |
| `products` | Produkty | sku, ean, price, stock_quantity, images JSONB, description, dimensions |
| `product_variants` | Warianty | attributes JSONB, sku, price_override |
| `product_listings` | Oferty marketplace | integration_id, external_id, sync_status, price_override. `sync_status=synced` tylko po realnym pushu/potwierdzeniu feedu, nie po pullu bez danych z marketplace |
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
| `suppliers` | Dostawcy | name, feed_url, feed_format, feed_type (xml/api/hybrid), integration_id, last_sync_at |
| `supplier_products` | Katalog dostawcy | external_id, price, stock_quantity, ean, weight, images JSONB, metadata JSONB, product_id (link do OMS) |
| `automation_rules` | Reguly automatyzacji | trigger_event, conditions JSONB, actions JSONB, priority |
| `automation_rule_logs` | Logi regul | conditions_met, actions_executed, error |
| `automation_delayed_actions` | Opoznione akcje | rule_id, order_id, execute_at, executed, attempt_count, last_attempt_at, action_data JSONB |
| `price_lists` | Cenniki B2B | discount_type, valid_from, valid_to, currency |
| `price_list_items` | Pozycje cennika | product_id, price, min_quantity, discount |
| `exchange_rates` | Kursy walut | base_currency, target_currency, rate, source |
| `order_groups` | Grupy zamowien | group_type (merge/split), source/target_order_ids |
| `sync_jobs` | Logi synchronizacji | job_type, status, items_processed |
| `webhook_events` | Eventy (przychodzace) | provider, event_type, payload JSONB |
| `webhook_deliveries` | Dostawy (wychodzace) | url, event_type, response_code |
| `audit_log` | Dziennik audytu | action, entity_type, entity_id, ip_address |
| `product_categories` | Hierarchiczne kategorie | name, parent_id, path, position |
| `allegro_parameter_mappings` | Mapowania parametrow Allegro per dostawca | supplier_id, allegro_category_id, parameter_id, source_field |
| `supplier_category_mappings` | Mapowania kategorii dostawca->OMS | supplier_id, supplier_category, product_category_id |
| `supplier_messages` | Komunikacja z dostawcami | supplier_id, direction, subject, body |
| `supplier_portal_tokens` | Tokeny portalu dostawcy | supplier_id, token_hash, expires_at |
| `dropship_orders` | Zamowienia dropship | order_id, supplier_id, status, external_id |
| `dropship_order_items` | Pozycje zamowien dropship | dropship_order_id, supplier_product_id, quantity |
| `listing_sync_configs` | Konfiguracja sync listingow | integration_id, product_id, sync_price, sync_stock |
| `listing_sync_log` | Log synchronizacji listingow | listing_id, event_type, status |
| `customer_segments` | Segmenty klientow | name, conditions JSONB |
| `customer_segment_members` | Czlonkowie segmentow | segment_id, customer_id |
| `customer_loyalty` | Lojalnosc klienta | customer_id, points, tier |
| `loyalty_programs` | Programy lojalnosciowe | name, rules JSONB |
| `payment_transactions` | Transakcje platnosci | order_id, amount, provider, status |
| `payment_settlements` | Rozliczenia platnosci | provider, period, total_amount |
| `pick_pack_sessions` | Sesje pick & pack | warehouse_id, status, started_by |
| `pick_pack_items` | Pozycje pick & pack | session_id, order_id, product_id, quantity |
| `purchase_orders` | Zamowienia zakupu | supplier_id, status, total_amount |
| `purchase_order_items` | Pozycje zamowien zakupu | product_id, quantity, unit_price |
| `recurring_orders` | Zamowienia cykliczne | customer_id, schedule, next_execution |
| `recurring_order_items` | Pozycje zamowien cyklicznych | product_id, quantity, price |
| `repricing_rules` | Reguly automatycznego pricingu | product_id, strategy, min_price, max_price |
| `repricing_log` | Log zmian cen | rule_id, old_price, new_price |
| `stock_sync_channels` | Kanaly synchronizacji stanow | integration_id, warehouse_id, direction |
| `stock_sync_events` | Eventy sync stanow | channel_id, product_id, quantity_change |
| `invitations` | Zaproszenia uzytkownikow | email, role_id, token, expires_at |
| `schema_migrations` | Wersja migracji DB | version, dirty |
| `billing_customers` | Klienci Stripe | tenant_id (UNIQUE), stripe_customer_id (UNIQUE) |
| `billing_subscriptions` | Subskrypcje | stripe_subscription_id (UNIQUE), plan, billing_interval, status, trial_end |
| `billing_checkout_sessions` | Sesje checkout | stripe_session_id (UNIQUE), plan, billing_interval, email, status, tenant_id, stripe_customer_id, stripe_subscription_id, subscription_status, trial_end, current_period_start, current_period_end |
| `used_license_tokens` | Uzyte tokeny licencji | jti (UNIQUE), email, plan, used_at |
| `listing_description_html` | Opisy HTML listingow | listing_id, html_content |
| `api_tokens` | Tokeny API (owner) | token_hash, name, revoked_at |

Lista powyzej to **podzbior rdzeniowy**. Pelny schemat (89 tabel, 50 wersji migracji `000001`–`000050`) obejmuje dodatkowo:

- fulfillment: `fulfillment_processes`, `fulfillment_units`, `fulfillment_steps`, `fulfillment_blockers`, `fulfillment_provider_attempts`
- orchestracja: `orchestration_outbox`, `orchestration_attempts`
- Provider Studio: `provider_definitions`, `provider_versions`, `provider_publication_events`, `provider_tenant_enables`, `provider_field_schemas`, `provider_capability_profiles`, `provider_status_mappings`, `provider_integration_gaps`, `provider_validation_probes`, `provider_validation_runs`, `provider_validation_results`
- platform: `platform_admins`, `platform_audit_log`
- pozostale: `loyalty_transactions`, `supplier_availability`, `supplier_availability_policy`, `external_workflow_tokens`, `message_templates`, `marketplace_category_mappings`

**Stan magazynowy:** `products.stock_quantity` jest polem legacy — nie jest zmniejszane przy wysylce. Kanoniczny stan to `warehouse_stock.quantity - reserved`. `Product.AvailableStock` jest wyliczane (`AvailableStockBatch`); 0 na surowym `List` bez populate oznacza "nie pobrano", nie "brak towaru".

Rezerwacja, wysylka i anulowanie sa **swiadome wariantow**: pozycja zamowienia jest kluczowana para `(product_id, variant_id)`, a instrukcja `UPDATE` trafia w konkretny wiersz `warehouse_stock` po jego `id`. Pozycja z wariantem korzysta z wierszy tego wariantu, a gdy wariant nie ma wlasnych wierszy — z wiersza produktowego (`variant_id IS NULL`), i odwrotnie. Ten fallback jest zgodny z dostepnoscia, na ktora zamowienie zostalo przyjete: `AvailableStockBatch` sumuje wszystkie wiersze produktu. Dopisanie towaru ze zwrotu (`received`) wybiera wiersz tak samo, preferujac magazyn domyslny; produkt bez zadnego wiersza magazynowego jest pomijany.

### Funkcje SECURITY DEFINER (bypass RLS)

`PUBLIC EXECUTE` jest cofany dla funkcji omijajacych RLS. Dostep jest nadawany jawnie tylko rolom aplikacyjnym, a CI sprawdza baze po migracjach pod katem regresji. 22 funkcje (nazwy w kodzie / migracjach):

| Funkcja | Cel |
|---------|-----|
| `find_tenant_by_slug` | Login: tenant po slug; `settings` zredagowane |
| `find_user_for_auth` | Login: user z haslem + TOTP |
| `find_order_tenant_id` | Publiczny formularz zwrotu |
| `find_return_by_token` | Status zwrotu po tokenie |
| `find_invitation_by_token` / `use_invitation` | Rejestracja z zaproszenia |
| `check_license_token_used` / `mark_license_token_used` / `update_license_token_tenant` | Licencja Ed25519, ochrona JTI |
| `get_tenant_plan` | Plan tenanta (settings bez zaszyfrowanych blobow) |
| `billing_create_checkout_session` | Checkout pre-rejestracja |
| `billing_complete_checkout_session` / `billing_complete_checkout_session_with_refs` | Zakonczenie sesji checkout |
| `billing_get_checkout_session` / `billing_get_checkout_session_with_refs` | Status sesji |
| `billing_claim_checkout_session` / `billing_update_checkout_tenant` | Claim sesji dla tenanta |
| `billing_create_customer` / `billing_get_customer_by_stripe_id` | Klient billing |
| `billing_upsert_subscription` / `billing_update_subscription_by_stripe_id` | Subskrypcja |
| `billing_sync_tenant_plan` | Sync planu tenanta z subskrypcji |

Pelne `tenants.settings` sa odczytywane tylko przez tenant-scoped sciezki repozytorium, nie przez publiczny/loginowy lookup sluga.

---

## 5. Backend API

### Middleware Stack

```
Globalnie (router, w tej kolejnosci):
  RequestID -> TrustedRealIP -> [Sentry] -> [Prometheus] -> SecurityHeaders
    -> Logging -> Recoverer -> CORS -> CSRF
  HSTS ustawia SecurityHeaders (gdy https), nie osobny middleware.

Route-scoped (NIE globalne):
  JWTAuth (Ed25519 access + opcjonalny hashed API token na /v1; blacklist JWT)
    -> [SentryContext] -> TenantPlanGuard -> MaxBodySize
    -> RequirePermission -> RateLimit* -> Handler

Nie ma RequireRole. RBAC to wylacznie RequirePermission (uprawnienia z JWT).
/v1/docs i /v1/openapi.yaml sa wylaczone poza developmentem, dopoki ENABLE_API_DOCS=true.
```

`TrustedRealIP` honoruje `X-Forwarded-For` / `X-Real-IP` tylko wtedy, gdy bezposredni peer IP jest w `TRUSTED_PROXY_CIDRS`. `X-Forwarded-For` jest rozwiazywany od prawej do lewej i wybiera pierwszy nie-zaufany hop, zeby lewy wpis podstawiony przez klienta nie mogl zmienic rate-limit key. Przy pustej liscie naglowki forwarded sa ignorowane, a rate limity i logi uzywaja TCP peer address. Produkcyjne values powinny zawierac tylko CIDR-y ingress/cloudflared podow, ktore kontroluja lub sanitizuja forwarded headers.

### Endpointy (~540; spec `/v1/openapi.yaml`)

#### Autentykacja (publiczne, rate limit 10/min login, 60/min refresh)

| Metoda | Sciezka | Opis |
|--------|---------|------|
| POST | `/v1/auth/register` | Rejestracja (nowy tenant + owner, opcjonalnie: invite_token, license_token, checkout_session_id) |
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

#### Tokeny API (owner)

| Metoda | Sciezka | Opis |
|--------|---------|------|
| GET | `/v1/api-tokens` | Lista wlasnych tokenow bez sekretow; tylko owner (403 dla innych rol) |
| POST | `/v1/api-tokens` | Tworzenie tokena; tylko owner (403); surowy sekret zwracany raz, przy utworzeniu |
| DELETE | `/v1/api-tokens/{id}` | Odwolanie tokena; tylko owner (403); dziala od nastepnego requestu |

Token uwierzytelnia na `/v1` przez `Authorization: Bearer`, nie wygasa i nie dziala na `/uploads`, `/v1/auth/2fa`, `/v1/auth/ws-ticket` ani `/v1/platform`.

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

#### Oferty marketplace (admin)

| Metoda | Sciezka | Opis |
|--------|---------|------|
| GET | `/v1/products/{pid}/listings` | Lista ofert produktu (wszystkie marketplace'y) |
| POST | `/v1/products/{pid}/listings/allegro` | Tworzenie oferty Allegro |
| POST | `/v1/products/{pid}/listings/ebay` | Tworzenie oferty eBay (3-step: inventory item -> offer -> publish) |
| POST | `/v1/products/{pid}/listings/olx` | Tworzenie ogloszenia OLX |
| POST | `/v1/products/{pid}/listings/woocommerce` | Tworzenie produktu WooCommerce |
| POST | `/v1/products/{pid}/listings/erli` | Tworzenie oferty Erli |
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
| GET | `/v1/integrations/allegro/auth-url` | URL OAuth Allegro; state jest powiazany z tenantem, userem i providerem |
| POST | `/v1/integrations/allegro/callback` | Callback OAuth Allegro; odrzuca state z innego tenanta/usera/providera; zapisuje access token i refresh token oraz aktywuje integracje przed zwroceniem sukcesu, a blad zapisu daje 5xx zamiast 200 |
| POST | `/v1/integrations/amazon/setup` | Setup Amazon SP-API |
| GET | `/v1/integrations/amazon/auth-url` | URL OAuth Amazon; state jest powiazany z tenantem, userem i providerem |
| POST | `/v1/integrations/amazon/callback` | Callback OAuth Amazon; odrzuca state z innego tenanta/usera/providera |
| GET | `/v1/integrations/ebay/auth-url` | URL OAuth eBay; state jest powiazany z tenantem, userem i providerem |
| POST | `/v1/integrations/ebay/callback` | Callback OAuth eBay; odrzuca state z innego tenanta/usera/providera |
| GET | `/v1/integrations/olx/auth-url` | URL OAuth OLX; state jest powiazany z tenantem, userem i providerem |
| POST | `/v1/integrations/olx/callback` | Callback OAuth OLX; odrzuca state z innego tenanta/usera/providera |
| PUT | `/v1/integrations/amazon/credentials` | Aktualizacja danych aplikacji Amazon (wymaga ponownego OAuth) |

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
| GET | `/v1/integrations/allegro/delivery-proposals/{oid}` | Oficjalne, wstepnie wypelnione body create-commands dla zamowienia |
| POST | `/v1/integrations/allegro/shipments` | Tworzenie przesylki; jeden POST create-commands na request, brak idempotencji, wiec klient nie moze ponawiac; polling do 60 s; 409 gdy brak metody dostawy albo gdy `deliveryMethodId` nie jest metoda z propozycji |
| POST | `/v1/integrations/allegro/orders/{oid}/wza-shipments/import` | Import juz utworzonej przesylki WzA; tylko odczyt, nigdy nie wysyla create-commands; 409 gdy w Allegro nie ma przesylki |
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

#### eBay -- Fulfillment, tracking, refunds (admin)

| Metoda | Sciezka | Opis |
|--------|---------|------|
| GET | `/v1/integrations/ebay/carriers` | Lista przewoznikow eBay |
| POST | `/v1/integrations/ebay/orders/{oid}/tracking` | Dodanie trackingu (carrier + tracking number) |
| POST | `/v1/integrations/ebay/orders/{oid}/refund` | Wydanie refundu (reason, items, amount) |
| GET | `/v1/integrations/ebay/policies` | Polityki sprzedawcy (fulfillment, return, payment) |
| GET | `/v1/integrations/ebay/offers` | Lista ofert eBay (filtr po SKU, paginacja) |
| POST | `/v1/integrations/ebay/import-offers` | Import ofert eBay do OpenOMS (match SKU -> link/create) |

#### OLX -- Kategorie, miasta, listingi (admin)

| Metoda | Sciezka | Opis |
|--------|---------|------|
| GET | `/v1/integrations/olx/categories` | Lista kategorii OLX (z parent_id dla sub-kategorii) |
| GET | `/v1/integrations/olx/categories/{cid}/attributes` | Wymagane atrybuty kategorii |
| GET | `/v1/integrations/olx/categories/suggest` | Sugestia kategorii na podstawie tytulu |
| GET | `/v1/integrations/olx/cities` | Wyszukiwanie miast OLX (autocomplete) |
| GET | `/v1/integrations/olx/cities/{cid}/districts` | Dzielnice miasta |

#### Dostawcy (admin)

| Metoda | Sciezka | Opis |
|--------|---------|------|
| GET | `/v1/suppliers` | Lista dostawcow |
| POST | `/v1/suppliers` | Dodanie |
| GET | `/v1/suppliers/{id}` | Szczegoly |
| PATCH | `/v1/suppliers/{id}` | Aktualizacja |
| DELETE | `/v1/suppliers/{id}` | Usuniecie |
| POST | `/v1/suppliers/{id}/sync` | Synchronizacja (XML/IOF/CSV/API) |
| GET | `/v1/suppliers/{id}/products` | Produkty dostawcy |
| GET | `/v1/suppliers/{id}/products/categories` | Kategorie zrodlowe dostawcy |
| POST | `/v1/suppliers/{id}/products/import` | Import produktow do katalogu OMS |
| POST | `/v1/suppliers/{id}/products/bulk-delete` | Masowe usuwanie produktow dostawcy |
| POST | `/v1/suppliers/{id}/products/{spid}/link` | Powiazanie z katalogiem |
| POST | `/v1/suppliers/{id}/products/{spid}/unlink` | Odlaczenie od katalogu |
| POST | `/v1/suppliers/{id}/products/{spid}/import-single` | Import pojedynczego produktu |
| DELETE | `/v1/suppliers/{id}/products/{spid}` | Usuniecie produktu dostawcy |
| GET | `/v1/suppliers/{id}/category-mappings` | Mapowania kategorii |
| PUT | `/v1/suppliers/{id}/category-mappings` | Upsert mapowania kategorii |
| DELETE | `/v1/suppliers/{id}/category-mappings/{mid}` | Usuniecie mapowania |
| GET | `/v1/suppliers/{id}/allegro-mappings` | Mapowania parametrow Allegro |
| PUT | `/v1/suppliers/{id}/allegro-mappings` | Bulk upsert mapowania Allegro |
| DELETE | `/v1/suppliers/{id}/allegro-mappings/{mid}` | Usuniecie mapowania Allegro |
| GET | `/v1/suppliers/{id}/allegro-mappings/categories` | Kategorie z mapowaniami Allegro |
| GET | `/v1/suppliers/{id}/attributes` | Atrybuty dostawcy (z XML) |
| POST | `/v1/suppliers/btp-wizard/start` | BTP wizard -- start importu |
| GET | `/v1/suppliers/btp-wizard/{id}/progress` | BTP wizard -- postep importu |
| POST | `/v1/suppliers/btp-wizard/{id}/api-keys` | BTP wizard -- klucze API |
| POST | `/v1/suppliers/btp-wizard/{id}/sync-settings` | BTP wizard -- ustawienia sync |
| POST | `/v1/suppliers/{id}/portal/generate-link` | Wygenerowanie linku portalu dostawcy (`/supplier-portal#token=...`) |
| POST | `/v1/suppliers/{id}/portal/revoke` | Cofniecie dostepu do portalu |
| GET | `/v1/suppliers/{id}/portal/status` | Status portalu dostawcy |

#### Portal dostawcy (publiczny, token w `Authorization: Bearer ...`)

| Metoda | Sciezka | Opis |
|--------|---------|------|
| GET | `/v1/supplier-portal/orders` | Lista zamowien dla dostawcy |
| GET | `/v1/supplier-portal/orders/{id}` | Szczegoly zamowienia |
| POST | `/v1/supplier-portal/orders/{id}/confirm` | Potwierdzenie zamowienia |
| POST | `/v1/supplier-portal/orders/{id}/ship` | Oznaczenie jako wyslane |
| POST | `/v1/supplier-portal/orders/{id}/messages` | Dodanie wiadomosci |
| GET | `/v1/supplier-portal/orders/{id}/messages` | Lista wiadomosci |

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

Rate shopping zwraca tylko stawki z providerow z zaakceptowana implementacja wyceny. InPost, DPD, GLS, UPS, Poczta Polska, Orlen Paczka, FedEx i DHL zwracaja `ErrCarrierRatesNotImplemented`, a endpoint pomija je zamiast pokazywac placeholderowe albo szacunkowe ceny.

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
| PATCH | `/v1/users/me/password` | Zmiana wlasnego hasla po podaniu obecnego hasla; rate limit 5/min na uzytkownika, z fallbackiem do IP |
| GET/POST/PATCH/DELETE | `/v1/users/...` | CRUD userow (admin); tworzenie wymaga hasla startowego |
| GET | `/v1/audit` | Dziennik audytu (admin) |
| GET | `/v1/webhooks` | Konfiguracja webhookow (admin) |
| GET | `/v1/webhook-deliveries` | Log dostaw webhookow (admin) |
| GET | `/v1/sync-jobs` | Logi synchronizacji (admin) |
| GET | `/v1/sync-jobs/{id}` | Szczegoly sync job |
| GET | `/v1/helpdesk/tickets` | Tickety Freshdesk |
| GET | `/v1/order-statuses` | Statusy zamowien (read-only) |
| GET | `/v1/custom-fields` | Pola niestandardowe (read-only) |
| GET | `/v1/product-categories` | Kategorie produktow (read-only) |
| POST | `/v1/webhooks/{provider}/{tenant_id}` | Webhook przychodzacy; znani providerzy wymagaja skonfigurowanego sekretu HMAC |
| POST | `/v1/webhooks/allegro` | Legacy webhook Allegro (HMAC) oparty o sekret z env |
| POST | `/v1/webhooks/allegro/{integration_id}` | Tenant-scoped webhook Allegro (HMAC); sekret `webhook_secret` jest zapisany per integracja w zaszyfrowanym `integrations.credentials` |
| POST | `/v1/webhooks/inpost` | Legacy webhook InPost (HMAC-SHA256) oparty o sekret z env |
| POST | `/v1/webhooks/inpost/{integration_id}` | Tenant-scoped webhook InPost (HMAC-SHA256); sekret `webhook_secret` jest zapisany per integracja w zaszyfrowanym `integrations.credentials`, lookup statusu uzywa jawnego privileged `WORKER_DATABASE_URL`, a aktualizacja jest wykonywana w tenant context |
| POST | `/v1/webhooks/stripe` | Webhook Stripe (Stripe-Signature) |
| GET | `/health` | Health check (no version disclosed) |
| GET | `/metrics` | Prometheus metrics (requires Bearer token) |
| GET | `/v1/openapi.yaml` | Specyfikacja OpenAPI |
| GET | `/v1/docs` | Swagger UI |

Allegro i InPost powinny uzywac scoped callbackow z `integration_id`, gdy klient ma wlasny sekret webhookow. Sekret jest opcjonalnym polem konfiguracji integracji w UI i nie wymaga zmiany sekretow Kubernetes ani redeployu. Legacy endpointy bez `integration_id` pozostaja jako fallback dla istniejacych callbackow skonfigurowanych przez operatora.

#### Billing (publiczne, bez auth)

| Metoda | Sciezka | Opis |
|--------|---------|------|
| GET | `/v1/billing/plans` | Lista planow (bez Stripe Price IDs) |
| POST | `/v1/billing/checkout` | Tworzenie sesji Stripe Checkout |
| GET | `/v1/billing/checkout/{session_id}` | Status sesji checkout |

#### Billing (tenant-scoped, wymaga JWT)

| Metoda | Sciezka | Opis |
|--------|---------|------|
| GET | `/v1/billing/subscription` | Aktualny plan, status subskrypcji i limity tenanta |

Plan guard egzekwuje status subskrypcji Stripe po stronie backendu: `past_due`/`unpaid`/`incomplete` oraz `canceled`/`paused`/`incomplete_expired` blokuja mutacje HTTP 402, a `suspended` blokuje uwierzytelniony dostep HTTP 402.

#### Onboarding (wymaga JWT)

| Metoda | Sciezka | Opis |
|--------|---------|------|
| GET | `/v1/onboarding/status` | Status onboardingu tenanta |
| PUT | `/v1/onboarding/step/{step}` | Oznaczenie kroku jako ukonczony/pominiety |
| POST | `/v1/onboarding/complete` | Zakonczenie onboardingu |

#### Konfiguracja publiczna

| Metoda | Sciezka | Opis |
|--------|---------|------|
| GET | `/v1/config` | Konfiguracja publiczna (registration_mode: open/invite/closed/disabled, billing_enabled, stripe_public_key) |

---

## 6. Frontend Dashboard

### Mapa stron (139 `page.tsx`)

#### Publiczne (bez logowania)

| Sciezka | Strona |
|---------|--------|
| `/login` | Formularz logowania (z obsluga 2FA) |
| `/register` | Rejestracja firmy |
| `/return-request` | Formularz zwrotu (klient) |
| `/return-request/[token]` | Status zwrotu (klient) |
| `/register/complete` | Formularz po platnosci Stripe |
| `/register/invite` | Rejestracja z tokenem zaproszenia |
| `/onboarding` | Wizard onboardingu (4 kroki) |
| `/track/[tenant_slug]` | Publiczne sledzenie zamowienia (numer + email). Teksty przez next-intl (`messages/pl/track.json`, `messages/en/track.json`). Bez logowania. |

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
| `/products/[id]/listings` | Oferty na marketplace'ach (dynamiczny picker + wizardy per marketplace) |
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
| `/marketplaces/ebay` | Setup eBay (OAuth2) |
| `/marketplaces/olx` | Setup OLX (OAuth2) |
| `/suppliers` | Dostawcy |
| `/suppliers/new` | Nowy dostawca (wizard) |
| `/suppliers/[id]` | Szczegoly dostawcy |
| `/suppliers/[id]/products` | Katalog produktow dostawcy |
| `/suppliers/[id]/categories` | Mapowania kategorii dostawcy |
| `/suppliers/[id]/allegro-mappings` | Mapowania parametrow Allegro |
| `/suppliers/[id]/settings` | Ustawienia synchronizacji dostawcy |

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
  Allegro (podstrony: dashboard, oferty, wiadomosci, zwroty, dostawa, spory, finanse, polityki, promocje, oceny, katalog, wysylam z allegro)
  eBay (setup OAuth2)
  OLX (setup OAuth2)
  Amazon (setup SP-API)
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

### Kluczowe komponenty (98)

| Komponent | Opis |
|-----------|------|
| `DataTable` | Generyczna tabela z sortowaniem, paginacja, selekcja, inline edit |
| `CommandPalette` | Cmd+K -- szybkie wyszukiwanie i nawigacja |
| `StatusTransitionDialog` | Dialog zmiany statusu (zamowienia, przesylki) |
| `PaczkomatSelector` | Wybor paczkomatu InPost (mapa/search/inline) |
| `OrderForm` | Formularz zamowienia (klient, pozycje, adres, custom fields) |
| `OrderKanbanBoard` | Widok Kanban zamowien z drag & drop |
| `AllegroListingWizard` | 4-krokowy kreator oferty Allegro |
| `CreateEbayListingDialog` | Dialog tworzenia oferty eBay (polityki, kategoria, condition) |
| `CreateOLXListingDialog` | Dialog tworzenia OLX (drzewo kategorii, miasto, atrybuty, kontakt) |
| `CreateListingWizard` | Dynamiczny picker marketplace'ow (z aktywnych integracji) |
| `CategoryTreePicker` | Picker drzewa kategorii (Allegro, OLX) |
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
| isAuth       |     | ...79 hooks  |     |              |
+-------------+     +--------------+     +------+-------+
                                                 |
                                          Auto-refresh
                                          on 401 (mutex)
```

---

## 7. Pakiety SDK (28)

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
| allegro-go-sdk | Allegro.pl | OAuth 2.0 | Zamowienia, oferty, eventy, katalog; bledy odswiezania tokenu sa zwracane wywolujacemu; przy terminalnym bledzie OAuth workery oznaczaja integracje jako wymagajaca ponownej autoryzacji, a handlery HTTP zwracaja 422 bez zmiany statusu integracji |
| amazon-sp-sdk | Amazon | LWA OAuth + AWS Signing | Zamowienia, inventory, pricing; terminalne bledy OAuth oznaczaja integracje jako wymagajaca ponownej autoryzacji |
| woocommerce-go-sdk | WooCommerce | REST API | Zamowienia, produkty, webhooks; `on-hold` mapuje platnosc jako `pending` (nie `paid`); malformed monetary fields reject order import |
| ebay-go-sdk | eBay | OAuth 2.0 | Zamowienia (OrderService), fulfillment + refundy (FulfillmentService), inventory CRUD + bulk (InventoryService), oferty lifecycle (OfferService), polityki konta (AccountService); malformed monetary fields reject order import; terminalne bledy OAuth oznaczaja integracje jako wymagajaca ponownej autoryzacji |
| kaufland-go-sdk | Kaufland | Feed API | Import CSV/XML; adapter MarketplaceProvider **bez** order pollera. To **nie** jest zywy kanal zamowien: `main.go` nie rejestruje `*OrderPoller`, a UI (picker, listing-sync, filtr zrodla zamowien) ukrywa `kaufland` |
| olx-go-sdk | OLX | OAuth 2.0 | Ogloszenia CRUD + komendy (AdvertService), kategorie + atrybuty + sugestie (CategoryService), miasta + dzielnice (CityService), transakcje (TransactionService); `invalid_grant` z endpointu tokenow jest zwracany jako terminalny blad OAuth, a workery oznaczaja integracje jako wymagajaca ponownej autoryzacji |
| mirakl-go-sdk | Mirakl/Empik | REST | Seller network; adapter MarketplaceProvider **bez** order pollera. Empik/Mirakl tez **nie** sa zywym kanalem zamowien (brak pollera, ukryte w UI) |
| erli-go-sdk | Erli | REST | Zamowienia, oferty |
| shoper-go-sdk | Shoper | REST | Zamowienia, produkty |
| prestashop-go-sdk | PrestaShop | REST | Zamowienia, produkty |
| shopify-go-sdk | Shopify | REST | Zamowienia, produkty; `PushOffer` tworzy produkt i zapisuje realny Shopify `variant_id` jako listing `external_id`, a stock sync rozwiązuje go do `inventory_item_id`; pozycje zamowien uzywaja `variant_id` jako `external_id` z fallbackiem do `product_id`; malformed monetary fields reject order import |
| btp-go-sdk | BTP.pro | Basic Auth | Katalog dostawcy, zamowienia dropship |

### Carrier SDK-i

| SDK | Provider | Auth | Glowne operacje |
|-----|----------|------|----------------|
| inpost-go-sdk | InPost | Bearer | Paczki, etykiety, tracking, paczkomaty, webhooks |
| dhl-go-sdk | DHL | API Key | Przesylki, etykiety, tracking |
| dpd-go-sdk | DPD | REST + InfoServices SOAP | Przesylki (Polska), etykiety, tracking |
| gls-go-sdk | GLS | API | Przesylki (Europa) |
| ups-go-sdk | UPS | XML/REST | Miedzynarodowe |
| poczta-polska-go-sdk | Poczta Polska | REST | Paczki pocztowe |
| orlen-paczka-go-sdk | Orlen Paczka | REST | Paczkomaty Orlen |
| fedex-go-sdk | FedEx | REST | Miedzynarodowe |
| apaczka-go-sdk | Apaczka (broker) | REST v2 + HMAC-SHA256 | Tworzenie przesylek, etykiety (waybill PDF), tracking, wycena (rate-shopping), punkty odbioru; broker fronting InPost/DPD/DHL/UPS/GLS/Poczta Polska/Orlen Paczka |

### Inne SDK-i

| SDK | Provider | Cel |
|-----|----------|-----|
| fakturownia-go-sdk | Fakturownia | Faktury |
| infakt-go-sdk | InFakt | Faktury |
| wfirma-go-sdk | wFirma | Faktury |
| ksef-go-sdk | KSeF | Krajowy System e-Faktur |
| smsapi-go-sdk | SMSAPI | Powiadomienia SMS |
| iof-parser | IOF/CSV/XML | Parser feedow dostawcow (IOF 3.0, XML, CSV) |

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
| Refresh Token | 30 dni | Cookie httpOnly (sciezka `/v1/auth`). Rotacja one-time; reuse kasuje cala rodzine sesji (`ErrRefreshTokenReuse`) |
| Token API (owner) | Do odwolania | Header `Authorization: Bearer oms_...` na `/v1` |

Dashboard po 60 min bezczynnosci operatora wywoluje best-effort `POST /v1/auth/logout` (`useSessionTimeout`), zeby uniewaznic rodzine refresh, a potem czysci Zustand i cookie `has_session`. Cookie `has_session` to tylko bramka UX, nie sesja.

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

Zablokowane konto (za duzo blednych kodow TOTP) na `POST /v1/auth/2fa/login` zwraca 429, tak samo jak `POST /v1/auth/login`. Blad zapisu last-used TOTP step jest fail-closed: tokeny nie sa wydawane, zeby jeden kod nie otworzyl dwoch sesji.

Sekret TOTP szyfrowany w kolumnie `users.totp_secret`. Kompatybilny z Google Authenticator, Authy i innymi aplikacjami TOTP.

### Szyfrowanie AES-256-GCM

Credentials integracji szyfrowane w bazie:
```
Plaintext -> AES-256-GCM(key, random_nonce) -> Base64 -> DB
DB -> Base64 decode -> AES-256-GCM decrypt -> Plaintext
```

Pola sekretow w `tenants.settings` sa szyfrowane tym samym mechanizmem przed zapisem do JSONB: haslo SMTP, token SMS, token KSeF, credentials providerow fakturowania oraz sekrety endpointow webhookow. API zwraca je jako wartosci zamaskowane.

Klucz: `ENCRYPTION_KEY` (64-char hex = 32 bajty)

### Hasla -- bcrypt (cost 12)

```
password -> bcrypt(cost=12) -> $2a$12$... -> DB
```

Tworzenie uzytkownika przez panel administracyjny wymaga jawnego hasla startowego w `POST /v1/users`; API zapisuje tylko bcrypt hash i nie zwraca hasla ani hasha. Zalogowany uzytkownik moze zmienic wlasne haslo przez `PATCH /v1/users/me/password` po podaniu obecnego hasla. Endpoint jest limitowany do 5 prob/min na uzytkownika, z fallbackiem do IP, zeby ograniczyc zgadywanie obecnego hasla. Zmiana hasla zapisuje event audytowy bez wartosci hasla.

### RBAC

```
Stare role:  owner > admin > member (fallback dla tokenow bez permissions)
Nowe role:   Custom roles z permissions[] egzekwowane przez backendowe RequirePermission

Access token zawiera role_id oraz efektywne permissions. Dla systemowych rol permissions sa wyliczane z domyslnych zestawow; Administrator nie ma users.manage, Owner ma pelny dostep. Dla custom role permissions sa ladowane z roles.permissions przy login/refresh/2FA.

Uprawnienia np.:
  orders.view, orders.create, orders.edit, orders.delete, orders.export
  products.view, products.create, products.edit, products.delete
  settings.manage
  users.manage
```

### Platform-Admin Boundary (OPE-404)

Warstwa platform-admin (administracja providerami, międzytenantowa) jest **całkowicie oddzielona od tenant-RBAC** — fundament pod Provider Integration Studio (OPE-405+).

```
Tożsamość:   tabela platform_admins (user_id UNIQUE -> users, ON DELETE RESTRICT),
             BEZ tenant_id i BEZ RLS (platform transcenduje tenanty).
Uprawnienia: providers:read, providers:write, providers:validate, providers:publish,
             providers:secrets (styl z dwukropkiem — NIGDY nadawane przez role tenanta).
Middleware:  RequirePlatformAdmin (po JWTAuth; fail-closed: 403 przy braku/niedostępności
             wpisu, 401 bez usera) + RequirePlatformPermission(perm). NIE reużywa
             tenant RequirePermission.
Routing:     grupa /v1/platform/* zamontowana jako rodzeństwo /v1 — NIE dziedziczy
             TenantPlanGuard ani tenant-RBAC. Backend jest autorytatywny; widoczność
             we frontendzie nigdy nie jest autoryzacją. Rate-limit 60/min.
Endpoint:    GET /v1/platform/me (whoami: user_id + permissions).
Audyt:       dedykowana tabela platform_audit_log (actor_user_id ON DELETE SET NULL,
             append-only) — osobny strumień od tenantowego audit_log.
Auth:        platform-admin loguje się istniejącym loginem (JWT -> user_id); oddzielona
             jest wyłącznie AUTORYZACJA, nie uwierzytelnianie.
```

### Provider Integration Studio — registry (OPE-405)

Rejestr providerów (rdzeń Studio), platform-managed, BEZ RLS — pod bramką OPE-404.

```
Tabele:      provider_definitions (rodzina providera: provider_key, type, regions[],
             business_domains[], latest_published_version_id),
             provider_versions (wersjonowana konfiguracja; UNIQUE(definition_id, version)),
             provider_publication_events (append-only audyt cyklu życia),
             provider_tenant_enables (allowlista private-beta per tenant).
Cykl życia:  research → designed → adapter_in_progress → internal_validation →
             private_beta → available → deprecated → retired
             (+ internal_validation→designed, private_beta/available→internal_validation).
             Przejścia egzekwowane w ProviderRegistryService; nielegalne → 422.
Niemutowalność: wersje w stanie published (private_beta/available/deprecated/retired)
             nie pozwalają na edycję metadanych (422). available ustawia
             definition.latest_published_version_id + published_at.
Emergency disable: available/private_beta → internal_validation z rollbackiem
             wskaźnika latest_published do poprzedniej available (lub NULL).
Endpointy:   /v1/platform/providers (+ /{id}, /{id}/versions, /{id}/versions/{vid}/
             {schema,publish,disable,enable-tenant,publication-events}); każdy strzeżony przez
             RequirePlatformPermission (providers:read|write|publish). Mutacje audytowane.
```

### Provider field schema — contract (OPE-406)

Wersjonowany schemat pól credential/settings (jedno źródło prawdy generujące formularze admina i klienta), 1:1 z `provider_versions`.

```
Tabela:      provider_field_schemas (UNIQUE provider_version_id, schema jsonb, no RLS).
Grupy pól:   secret_credentials, settings, environment, sync, feature_toggles, provider_options.
Pole:        key, label, type (string|password|number|boolean|enum|url|textarea), required,
             secret (cel zapisu), environment_scope (all|production|sandbox), help_text,
             validation (enum/regex/min/max/min_length/max_length), capability_enabled,
             test_connection_dependency.
Walidacja:   ValidateFieldSchema — znane grupy/typy/scope, unikalne klucze pól w całym
             schemacie, enum z wartościami, kompilowalny regex, min<=max. Błąd → 422.
Niemutowalność: edycja schematu zablokowana gdy wersja published (private_beta+) → 422.
Endpointy:   GET/PATCH /v1/platform/providers/{id}/versions/{vid}/schema (read|write).
Uwaga:       schemat tylko DEKLARUJE które pola są secret; wartości credentials klienta
             pozostają zaszyfrowane AES w integrations.credentials. Renderowanie formularzy
             (frontend) i ścieżka tenant-setup = późniejsze zadania (Studio UI).
```

### Provider capabilities / status mappings / gaps (OPE-407)

Deklaracje per provider-version: co wersja potrafi, jak mapują się statusy, jakie luki blokują publikację. Platform-managed, bez RLS.

```
Tabele:      provider_capability_profiles (capability_key, support_status, channel/mode/
             freshness, required_inputs[]/provided_outputs[], latency_sla_seconds),
             provider_status_mappings (status_domain order|shipment|line, raw_status →
             canonical_status/event_type/step_key, confidence, is_terminal),
             provider_integration_gaps (gap_type, severity info|warning|action_required|
             system_error, status open|acknowledged|resolved).
Support:     supported|configured|unsupported|requires_manual|degraded|unknown.
Replace-set: capabilities i status-mappings ustawiane hurtowo (DELETE+INSERT w transakcji),
             walidowane (unikalne klucze/pary, znane enumy), zamrożone gdy wersja published.
Gaps:        tworzone/listowane/rozwiązywane; OPE-408 (validation engine) będzie je auto-tworzyć
             (np. nieznany status → missing_status_mapping).
Endpointy:   GET/PATCH /…/versions/{vid}/{capabilities,status-mappings}, GET/POST /…/gaps,
             PATCH /…/gaps/{gap_id} — RequirePlatformPermission (read|write), audytowane.
```

### Provider validation engine & evidence (OPE-408)

Silnik walidacji wersji providera: definicje probe'ów + niemutowalne validation runs/results + evidence + auto-tworzenie luk. Platform-managed, bez RLS.

```
Tabele:      provider_validation_probes (probe_type [15 typów], label, destructive,
             required, config jsonb), provider_validation_runs (environment sandbox|
             production, verdict pending|passed|failed|error, started/finished, actor),
             provider_validation_results (status passed|failed|skipped|error, observation
             [redacted], payload_hash, findings; UNIQUE(run_id,label)).
Stan run:    StartRun (pending) → RecordResult (tylko gdy pending) → CompleteRun (verdict z
             VerdictFromResults; finalizacja niemutowalna). Wszystko po publikacji wersji
             zamrożone (probes).
Safety:      probe'y destructive (np. sandbox_order_create) wymagają allow_destructive=true
             przy StartRun, inaczej 422.
Evidence:    results trzymają tylko redacted observation + payload_hash (HashPayload sha256);
             nigdy surowych sekretów/PII. Faktyczne wykonywanie probe'ów na realnych
             providerach = pluggable/odłożone (adaptery+creds, OPE-413+).
Auto-gaps:   CompleteRun atomowo (transakcja) finalizuje verdykt i tworzy luki dla
             failed/error (auth_check→auth_failure, feed_parse/malformed→parser_failure,
             order_preflight→missing_order_preflight, shipment_tracking_read→missing_tracking,
             reszta→provider_business_error; severity error→system_error).
Endpointy:   GET/PATCH /…/versions/{vid}/probes, POST /…/validate, GET /…/validation-runs[/{id}],
             POST /…/validation-runs/{id}/{results,complete} — providers:read|write|validate.
```

### Migracja istniejących providerów do registry (OPE-412)

Seed rejestru draftowymi definicjami istniejących providerów (class-first, §526). Additive — integracje najemców nietknięte; frontendowe statyczne mapy zostają jako fallback.

```
Katalog:     internal/service/provider_catalog.go — reprezentanci klas zdolności:
             allegro (marketplace), inpost (carrier), btp (supplier),
             fakturownia (invoicing), shopify (shop). Każdy: provider_type + schema
             credentiali (z realnych pól) + capability-stuby.
Seeder:      ProviderRegistrySeeder.Seed — idempotentny (klucz provider_key); tworzy
             definicję + wersję `research` (draft) + schema + capabilities. Pomija
             providerów już obecnych. Forward-only, nie modyfikuje istniejących.
Uczciwość:   wszystkie capabilities seedowane jako `unknown` — NIGDY fabrykowane jako
             supported; weryfikacja/upgrade przez Studio (OPE-407 workbench / OPE-408
             validation). Migracja status-map = autoring w workbenchu (świadomie odłożone).
Uruchomienie: env SEED_PROVIDERS=true → seed przy starcie (jak backfill szyfrowania).
             Publikacja seedowanych wersji = osobny gated krok (OPE-413).
```

### Fulfillment Orchestration — model danych (OPE-414, tor B)

Kanoniczny, TENANT-SCOPED model realizacji (ADR-424) — fundament orchestracji (OPE-415+). W przeciwieństwie do tabel Studio: RLS + `database.WithTenant`.

```
Tabele:      fulfillment_processes (per-order: aggregate_status [new|validating|ready|
             in_progress|waiting_external|blocked|completed|cancelled] + health_status
             [ok|warning|action_required|system_error], niezależne),
             fulfillment_units (typ warehouse|dropship|backorder|mixed_child|manual; status
             pending..skipped; parent_unit_id dla splitu),
             fulfillment_steps (22 kanoniczne, provider-agnostyczne step_key; status; attempts),
             fulfillment_blockers (12 kodów w kategoriach integration|supplier|operator|
             capability|mapping; status open|acknowledged|resolved).
RLS:         FORCE ROW LEVEL SECURITY + tenant_isolation (current_setting('app.current_tenant_id'))
             na wszystkich 4 tabelach — jak orders/shipments. Read+write izolowane per-najemca.
Repo:        FulfillmentRepository — metody przyjmują pgx.Tx z database.WithTenant (wzorzec
             tenant-scoped, jak AuditRepository). CRUD procesów/unitów/kroków/blockerów.
Zakres:      tylko model+RLS+repo (foundation). Orchestracja (outbox/worker/commands),
             inventory availability i stock-propagation = OPE-415+. Repo podłączy OPE-415.
```

### Orchestration outbox + idempotent worker (OPE-415, tor B)

Transactional outbox: trwała kolejka efektów ubocznych fulfillmentu + idempotentny worker. TENANT-SCOPED (RLS).

```
Tabele:      orchestration_outbox (event_type, payload, status pending|claimed|succeeded|failed,
             idempotency_key UNIQUE per tenant, attempts/max_attempts, next_attempt_at, last_error),
             orchestration_attempts (per-próba: attempt_number, status running|succeeded|failed, error).
Enqueue:     EnqueueEvent w TEJ SAMEJ transakcji co zmiana stanu (database.WithTenant) — efekt
             uboczny nie ginie przy crashu. Idempotentny (ON CONFLICT tenant+idempotency_key).
Worker:      OrchestrationWorker (Manager/Worker, gated ORCHESTRATION_WORKER_ENABLED, domyślnie
             OFF). Claimuje due rows MIĘDZY najemcami przez privileged workerPool (bypass RLS)
             z FOR UPDATE SKIP LOCKED — brak podwójnego claimu. Per event: StartAttempt →
             Dispatch → FinishAttempt.
Dispatcher:  OrchestrationDispatcher (mapa event_type → handler). OPE-415 dostarcza pusty (+ test);
             realne handlery = OPE-416/417/418. Nieznany event_type → PermanentError.
Retry:       sukces→succeeded; błąd retryable→pending + next_attempt_at = now+backoff (30s×2^n,
             cap 1h); permanent (PermanentError) lub wyczerpane attempts→failed + fulfillment_blocker.
Reaper:      każdy tick zaczyna się od reap-pass (OPE-534): wiersze 'claimed' starsze niż
             10 min (crash workera między claim a mark) wracają do 'pending' z backoffem
             (przerwana próba liczy się do attempts); wyczerpane attempts → failed +
             fulfillment_blocker; wiszące 'running' attempts zamykane jako failed.
```

### Routing tworzenia zamówień przez fulfillment (OPE-416, tor B)

Most między tworzeniem zamówienia a orchestracją — **addytywny, gated `FULFILLMENT_PROCESS_ENABLED`** (domyślnie OFF → zero zmian w zachowaniu).

`FULFILLMENT_PROCESS_ENABLED` jest flaga **recording**. OFF: `EnsureProcessForOrder` jest no-op, home (`/`) zostaje przy dashboardzie heurystycznym, a panel control tower na home i strona `/operations/fulfillment` pokazuja puste/zerowe stany (endpointy read sa zarejestrowane, RLS-scoped). `/operations/fulfillment` jest `ready` w obu surface'ach, zeby operator mogl czytac parity **zanim** ktokolwiek wlaczy recording. ON: tworzenie zamowienia zapisuje proces + event `order.created` w tej samej transakcji.

Produkcyjny Helm (`deploy/helm/openoms/values.yaml` + ConfigMap) **nie** ustawia `FULFILLMENT_PROCESS_ENABLED`. Wykres nie ma tej zmiennej. Domysl `envDefault:"false"` w `internal/config` zostaje. Tej fali **nie** wolno flipowac produkcji przez Helm. Wlaczenie jest recznym env na srodowisku, po backfillu i progu parity (kolejnosc nizej).

```
Hook:        FulfillmentService.EnsureProcessForOrder(ctx, tx, tenantID, orderID) wołany W TEJ
             SAMEJ transakcji co insert zamówienia (atomowo) w: OrderService.Create (manual/API),
             ImportService, BaseLinkerImportService. No-op gdy flaga OFF; idempotentny
             (GetProcessByOrder→ErrNoRows→CreateProcess), enqueue eventu order.created
             (idempotency_key order.created:<orderID>).
Handler:     OrderCreatedHandler (OrchestrationHandler) — proces new→validating w kontekście
             najemcy; rejestrowany gdy ORCHESTRATION_WORKER_ENABLED. Pierwszy realny handler outboxa.
Odroczone:   marketplace_order_poller (followup — flaga OFF, nic nie ginie).
```

### Atrybuty providerów przesyłka/etykieta/tracking (OPE-417, tor B)

Efekty uboczne carrierów w modelu orchestracji jako **provider attempts** + step events + typed blockers — addytywne, gated, **best-effort** (nigdy nie przerywa operacji bazowej).

```
Migracja:    000038 fulfillment_provider_attempts (TENANT-SCOPED RLS): provider, operation,
             status, request_id, correlation_id, raw_provider_status (surowy status carriera),
             result_redacted (jsonb, nigdy sekretów), payload_hash (sha256), error_code.
Hooki:       ShipmentService.Create, LabelService.GenerateLabel (carrier CreateShipment + GetLabel),
             TrackingPoller.Run — każdy własny database.WithTenant, log-and-continue.
Mapowanie:   carrier-error → istniejące kody blockerów (rate_limit/auth/outage/missing_data/
             rejection); kanoniczny mapping statusu trackingu (raw zachowany); carrier bez API
             trackingu → jawny outcome "unsupported capability" (nie błąd).
```

### Jednostki realizacji warehouse/pick-pack/dropship/backorder (OPE-418, tor B)

Mieszane ścieżki realizacji jako **fulfillment units + steps** + typed supplier blockers — addytywne, gated, best-effort (reużywa tabel 000036, bez migracji).

```
Serwis:      FulfillmentService — EnsureUnit (idempotentny per process/typ/dedupe-key),
             RecordUnitTransition, RecordStep, CreateSupplierBlocker (kody ADR-424),
             AggregateMixedOrder + czysty AggregateProcessState (agregat stanu z jednostek),
             ClassifySupplierCapability (api/portal/manual/unsupported z feed_format + integracji).
Hooki:       pick_pack (pick/pack→step), dropship (jednostka per dostawca; manual/portal→waiting
             preflight; unsupported→blocked unit + integration_capability_missing blocker),
             warehouse_document (PZ zamknięcie backorder + wznowienie warehouse; WZ dispatch),
             purchase_order (setter; receive = udokumentowany no-op do czasu linku PO↔order).
```

### Silnik zamówień do dostawcy — prepare/preflight/submit/reconcile (OPE-418/Phase-7, tor B)

Cykl życia złożenia zamówienia dropship u dostawcy przez orchestration outbox — addytywny, gated `SUPPLIER_ORDER_ENABLED` (domyślnie OFF → brama dropship zachowuje obecne zachowanie, zero auto-submitu).

```
Adapter:     integration.SupplierProvider (ProviderName/FetchProducts/FetchInventory/CreateOrder)
             + opcjonalne sub-interfejsy: SupplierPreflighter (Preflight) i SupplierStatusReader
             (GetOrderStatus) — wykrywane przez type-assert, brak = krok pomijany/no-op.
Brama:       DropshipService.recordDropshipUnit — capability api + routable → enqueue eventu
             supplier.order.submit (idempotency key supplier.order.submit:<order>:<supplier>);
             portal/manual → krok preflight waiting_external + blocker supplier_manual_submission_
             _required (operatorski, nie cichy auto-submit). Best-effort: enqueue nigdy nie cofa routingu.
Handler:     SupplierOrderHandler (OrchestrationHandler dla supplier.order.submit) — prepare
             (resolve EAN/SKU per linia; brak identyfikacji → supplier_order_missing_data) →
             preflight (gdy adapter wspiera) → submit (CreateOrder). Idempotentny: UPDATE
             istniejącego pending dropship_orders supplier_reference (fallback Create); pomija gdy
             reference już ustawiony. PermanentError (odrzucenie biznesowe) → blocker, nie retry.
Reconcile:   SupplierOrderStatusPoller — kanoniczny mapping statusu dostawcy na status dropship
             (pending/sent/confirmed/shipped/delivered/cancelled); raw zachowany; nieznany →
             external_status_unmapped. Legacy auto-submit UpdateStatus pending→sent pomijany gdy
             supplier_reference już obecny (brak podwójnego submitu).
Blockery:    supplier_order_missing_data, supplier_order_ambiguous_sku, supplier_order_rejected,
             supplier_payment_awaiting, supplier_partial_fulfillment, supplier_manual_submission_required.
```

### Operations + fulfillment READ API i dashboard (OPE-419, tor B)

Strona READ — tenant-safe widok nad danymi 414–418 (procesy/jednostki/kroki/blockery/attempts) + sterownia operacyjna w dashboardzie.

```
Serwis:      FulfillmentReadService — KAŻDY odczyt/zapis przez database.WithTenant (RLS); brak
             cross-tenant pool. Bucket-y operatora: ready/processing/stuck/blocked/provider_issue/
             missing_data (classifyProcessBucket).
Migracja:    000039 index fulfillment_processes(tenant_id, aggregate_status, health_status).
Endpointy:   GET  /v1/fulfillment/processes (filtr status/health, paginacja),
             GET  /v1/fulfillment/processes/{id} (detal: units+steps+blockers+attempts),
             GET  /v1/fulfillment/orders/{orderID},
             GET  /v1/operations/summary (liczniki bucketów),
             GET  /v1/operations/exceptions (akcjonowalne procesy + top blocker),
             GET  /v1/operations/integration-capability-summary,
             POST /v1/fulfillment/blockers/{id}/resolve,  POST /v1/fulfillment/steps/{id}/retry.
             GET-y: uprawnienie orders.view; akcje: orders.edit. Najemca z kontekstu (nie z param).
Dashboard:   /operations/fulfillment (6 bucketów + exceptions + drilldown do detalu) +
             FulfillmentControlTowerPanel na home (additywnie, operator-guarded). Puste stany gdy
             flaga recording OFF — nie zastępuje legacy dashboardu (cutover = OPE-423).
```

### Automation v2 — routing set_status przez outbox (OPE-421, tor B)

Akcja automatyzacji `set_status` routowana przez orchestration outbox — addytywne, gated `AUTOMATION_ORCHESTRATION_ENABLED` (OFF = bezpośredni TransitionStatus jak dotąd).

```
Enqueue:     executeSetStatus gdy flaga ON: ensure-process (idempotentny, spełnia NOT NULL
             outbox process_id FK) + enqueue eventu automation.set_status (idempotency key
             rule+order+target; engine stempluje action.RuleID/ActionIndex).
Handler:     AutomationStatusTransitionHandler — idempotentny: no-op gdy zamówienie już w target
             statusie (zrywa burzę efektów ubocznych: email/SMS/webhook/invoice/rekurencyjny
             order.status_changed). Błędny payload/order/status → PermanentError.
Blocker:     permanentny fail → kod automation_action_failed (widoczny w ops dashboardzie).
Dual-flag:   enqueue wymaga AUTOMATION_ORCHESTRATION_ENABLED; przetwarzanie też ORCHESTRATION_WORKER_ENABLED.
```

### Obserwowalność orchestracji (OPE-422)

Addytywna instrumentacja ścieżek 414–419 — best-effort (nigdy nie przerywa/nie spowalnia operacji).

```
Metryki:     pakiet internal/obsmetrics (ręczne atomowe liczniki + Prometheus text exposition,
             BEZ klienta prometheus — wzorzec middleware.MetricsCollector): provider attempts,
             blockers per kategoria, outbox events + queue depth, unit/step transitions, validation
             runs/failures, publication transitions. DYSCYPLINA KARDYNALNOŚCI: tylko bounded-enum
             labelki, nieznane→"other"; NIGDY id (tenant/order/process/...) jako labelka. Przez /metrics.
Logi:        correlation_id (idempotency key) + kontekst eventu w logach workera; payloady nie logowane.
Audit:       akcje operatora resolve/retry → wpis audit (fulfillment.blocker.resolved /
             .step.retried) we własnej transakcji po commicie (fail audytu nie cofa akcji).
```

### Rollout / cutover (OPE-423)

Bezpieczne przejście produkcji z legacy stanu na model process-backed — wszystko addytywne + gated, rollback = wyłączenie flagi.

```
Backfill:    FulfillmentBackfillService/Worker — procesy dla niefinalnych zamówień bez procesu
             (LEFT JOIN ... fp.id IS NULL), tą samą idempotentną ścieżką co 416. PODWÓJNA brama:
             FULFILLMENT_BACKFILL_ENABLED(false) + FULFILLMENT_BACKFILL_DRY_RUN(true) — zapis wymaga
             OBU; default = total no-op. Resumable (bez ledgera), per-tenant WithTenant, batch.
Parity:      GET /v1/operations/parity (orders.view) — read-only raport legacy vs process-backed:
             non_terminal_orders, fulfillment_processes, orders_missing_process (gap→0 po backfillu),
             process_coverage, process_coverage_met (werdykt bramki cutoveru, próg domyślny 0.99).
Cutover:     flaga build-time NEXT_PUBLIC_OPENOMS_FULFILLMENT_DASHBOARD (heuristic[default]|
             process-backed) — dual-read home: default = legacy, flip → control tower primary.
             Parity-readiness indicator (konsumuje /operations/parity). Cutover = flip flagi po parity.
Migracja:    000040 UNIQUE index fulfillment_processes(tenant_id, order_id) (addytywnie) +
             CreateProcess INSERT ... ON CONFLICT DO NOTHING — duplikat procesu strukturalnie niemożliwy.
```

**Flagi orchestracji** (wszystkie domyślnie OFF/bezpieczne; kolejność włączania w rolloucie):
`SEED_PROVIDERS` → `FULFILLMENT_PROCESS_ENABLED` (recording) → `FULFILLMENT_BACKFILL_ENABLED`+`FULFILLMENT_BACKFILL_DRY_RUN=false` (backfill) → `ORCHESTRATION_WORKER_ENABLED` (przetwarzanie outboxa) → próg parity → `NEXT_PUBLIC_OPENOMS_FULFILLMENT_DASHBOARD=process-backed` (cutover UI) → `AUTOMATION_ORCHESTRATION_ENABLED` (na końcu, najwrażliwsze).

### Zabezpieczenia

| Zagrozenie | Mitygacja |
|-----------|-----------|
| SQL Injection | Parametryzowane zapytania (pgx driver) |
| XSS | React auto-escape + sanityzacja inputow (strip tags) + CSP header. dangerouslySetInnerHTML usuniete. |
| CSRF | Double-submit cookie (csrf_token cookie + X-CSRF-Token header, SameSite=Lax). Domain cookie CSRF jest eTLD+1 z `FrontendURL` (lista public suffix): `app.openoms.org` -> `.openoms.org`, `app.example.com.pl` -> `.example.com.pl` (nie `.com.pl`), `app.example.co.uk` -> `.example.co.uk` (nie `.co.uk`). Host-only (pusty Domain) dla localhost, adresow IP i nazw, ktore same sa public suffix. CSRF jest pomijany, gdy request niesie niepusty credential `Authorization: Bearer` (JWT lub token API), wiec klienci bez cookies dzialaja; sesje cookie nadal wymagaja naglowka. |
| Clickjacking | X-Frame-Options: DENY + CSP frame-ancestors 'none' |
| Tenant leakage | RLS + FORCE ROW LEVEL SECURITY |
| Token theft | SHA-256 hash w blacklist, httpOnly cookies |
| Token revocation | Redis-backed composite blacklist; poza developmentem Redis jest wymagany, a in-memory fallback jest tylko lokalny/explicit single-node. Tokeny API sa odwolywane w bazie przez `revoked_at`, nie przez blacklist JWT; odwolanie dziala od nastepnego requestu. |
| SSRF | noPrivateDialer na wszystkich polaczeniach wychodzacych (webhooks, automation, supplier feeds). IPv4 + IPv6 (w tym ::/128, ff00::/8). |
| SSRF (WebSocket) | Walidacja Origin header + ticket-only auth (JWT w URL usuniety) |
| Brute force | Rate limiting (10/min login, 5/min zmiana hasla, 60/min refresh, 30/min public). Atomowy Lua script (INCR+EXPIRE). Invalid login paths wykonuja dummy bcrypt compare, zeby ograniczyc timing oracle dla tenant/email/password. Lockout konta (haslo i 2FA) zwraca 429. |
| DoS / webhook poisoning | Max body size (1MB default, 10MB upload). MaxBytesReader na webhook handlerach. BTP XML catalogue feeds maja limit 200 MiB; IOF feeds 50 MiB; 50 000 produktow na import. Tenant-configured shop SDK JSON responses (WooCommerce/PrestaShop/Shoper/Shopify) maja limit 10 MiB. Webhooki znanych providerow fail-closed przy braku sekretu HMAC. |
| Account takeover | 2FA/TOTP, bcrypt, Ed25519 JWT |
| Info disclosure | Brak wersji w /health, brak X-Powered-By, /metrics chroniony tokenem |
| MIME sniffing | X-Content-Type-Options: nosniff |
| Referrer leak | Referrer-Policy: strict-origin-when-cross-origin |
| HSTS | Strict-Transport-Security w produkcji |
| Supply chain | Swagger UI CDN pinned do dokladnej wersji (5.18.2) |
| Observability PII | Sentry `SendDefaultPII=false`, request scrubber + `BeforeSend` backstop; failed-login audit payload bez plaintext emaila |
| Ekspozycja niegotowych funkcji | Serwerowy gate gotowosci (`OPENOMS_API_SURFACE`, domyslnie `client-ready`): grupy tras niegotowych funkcji zwracaja `404 feature_not_available`, a tworzenie integracji/przesylek z niegotowym providerem `422 provider_not_available`. Rejestr `internal/readiness/readiness.json` jest jedynym zrodlem prawdy wspoldzielonym z dashboardem; `OPENOMS_API_SURFACE=full` wylacza gating. |

### Bezpieczenstwo infrastruktury (Kubernetes)

| Warstwa | Mechanizm |
|---------|-----------|
| Secrets encryption at rest | AES-CBC w k3s (EncryptionConfiguration) |
| K8s audit logging | Audit policy z logowaniem zmian w secrets, RBAC, write ops |
| Pod Security Standards | PSS enforce: restricted (OpenOMS app namespace). Alloy monitoring dziala jako restricted Deployment i tailuje logi przez Kubernetes API zamiast hostPath/root DaemonSet. System/storage namespaces moga miec osobne baseline/privileged profile. |
| NetworkPolicies | Default-deny ingress na wszystkich 15 namespacach. Publiczny chart OpenOMS ma konfigurowalne klasy egress (`dns`, `https`, `database`, `redis`) z szerokimi domyslnymi destinationami dla self-hosted compatibility. Enterprise staging/production overlays zawężają DNS do CoreDNS w `kube-system`, PostgreSQL i Redis do podow w `apps-core`, a HTTPS pozostaje destination-unscoped na TCP/443 dla zewnetrznych API providerow, S3, Sentry i SaaS endpointow o dynamicznych zakresach. |
| State DB permissions | chmod 600 (wylacznie root) |
| TLS | Mutual TLS do API servera k3s, TLS 1.2+ z strong cipher suites |
| Image scanning | Trivy CRITICAL+HIGH w CI/CD pipeline |
| SBOM | CycloneDX JSON generowany z obrazow release i publikowany jako `openoms-sbom-<sha>`; dashboard SBOM blokuje npm components bez znanej wersji |
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

Zywy graf statusow zamowienia to `tenants.settings.order_statuses` albo `DefaultOrderStatusConfig()`. Pakiet `order-engine` jest uzywany przez **shipmenty**; `OrderService` go nie importuje.

`OrderService.TransitionStatus` po commicie odpala: email, webhook + WebSocket, auto-fakture, SMS, automatyzacje, sync fulfillment Allegro, stock przy pierwszym `shipped`, zwolnienie rezerwacji przy `cancelled`, lojalnosc przy `completed`.

Zapis statusu i efekty uboczne sa rozdzielone na dwie metody, `WriteOrderStatusInTx` (zapis + wpis audytowy, w transakcji wolajacego) i `FireOrderStatusEffects` (efekty po commicie), wystawione jako interfejs `OrderStatusSyncer`. `ShipmentService` uzywa obu: status zamowienia wynikajacy ze zdarzenia kuriera zapisuje sie atomowo z wierszami przesylki, a caly zestaw efektow odpala po commicie tej samej transakcji. Zdarzenie kuriera nie przechodzi walidacji grafu statusow — odebrana paczka jest faktem, nie wnioskiem. Dekrement stocku jest bramkowany przez `shipped_at` (`COALESCE` przy zapisie, nigdy nie czyszczone), wiec powtorzony `picked_up` nie zdejmuje stocku po raz drugi.

Rezerwacja stocku wykonuje sie **w tej samej transakcji** co zapis zamowienia, w `Create` i w `Duplicate`. Niemoznosc zapisania rezerwacji przewraca cale zamowienie; zarezerwowanie mniej niz zamowiono (brak wierszy magazynowych albo mniej na stanie) nie jest bledem i przepuszcza zamowienie, jak dotychczas.

Publiczny `POST /v1/public/returns` przechodzi przez `ReturnService.CreatePublic` — handler odpowiada tylko za sprawdzenie wlasnosci zamowienia (dopasowanie e-maila przez funkcje omijajaca RLS), a wiersz, audyt, webhook i zdarzenie automatyzacji sa wspolne z tworzeniem wewnetrznym. Zwrot przy statusie `received` **dopisuje towar do stocku magazynowego** (fizyczny powrot towaru); `refunded` pozostaje zdarzeniem finansowym i ustawia tylko `payment_status` zamowienia. Graf zwrotow nie ma sciezki powrotnej do `received`, wiec dopisanie nie moze zadzialac dwa razy.

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
    |     |    +- set_status -> "confirmed"  (graf statusow obowiazuje; params.force = opt-in)
    |     |    +- send_email -> sales@company.com
    |     |    +- add_tag -> "auto-confirmed"
    |     |    +- delay(30m) -> kolejna akcja  <- OPOZNIENIE!
    |     |         |
    |     |         v
    |     |    Zapis w automation_delayed_actions (execute_at = NOW() + 30m)
    |     |         |
    |     |         v
    |     |    DelayedActionWorker (co 30s) -> execute_at <= NOW() -> wykonaj akcje
    |     |      +- blad wykonania -> retry z exponential backoff (max 5 prob)
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

### Flow 8: Tworzenie oferty eBay (3-step)

```
Krok 1: Wybor polityk i kategorii
    |  GET /v1/integrations/ebay/policies?marketplace_id=EBAY_PL
    |  -> fulfillment, return, payment policies
    v
Krok 2: Konfiguracja oferty
    |  Kategoria eBay, condition, tytul, opis, cena
    v
Krok 3: Publikacja
    |  POST /v1/products/{pid}/listings/ebay
    |  -> Backend: PUT inventory_item/{sku}   (Inventory API)
    |  -> Backend: POST offer                  (Offer API)
    |  -> Backend: POST offer/{id}/publish     (Offer API)
    v
Oferta opublikowana na eBay (3 wywolania API w jednej transakcji)
```

### Flow 9: Tworzenie ogloszenia OLX

```
Krok 1: Wybor kategorii (drzewo)
    |  GET /v1/integrations/olx/categories
    |  -> Nawigacja: top-level -> sub -> leaf (wymagana kategoria koncowa)
    v
Krok 2: Atrybuty kategorii
    |  GET /v1/integrations/olx/categories/{cid}/attributes
    |  -> Formularz z wymaganymi atrybutami (typ, walidacja, wartosci)
    v
Krok 3: Lokalizacja i kontakt
    |  GET /v1/integrations/olx/cities?query=Warsz (autocomplete)
    |  GET /v1/integrations/olx/cities/{cid}/districts
    |  -> city_id, district_id, contact_name, contact_phone
    v
Krok 4: Publikacja
    |  POST /v1/products/{pid}/listings/olx
    |  -> Backend: POST /adverts (OLX Partner API)
    v
Ogloszenie opublikowane na OLX
```

### Flow 10: Import ofert eBay

```
POST /v1/integrations/ebay/import-offers
    |
    v
EbayImportService.ImportOffers()
    |
    +- Per-tenant mutex (TryLock, jedna operacja na raz)
    +- Odszyfruj credentials -> eBay provider
    +- Fetch ofert z eBay (paginacja 100/strone, max 50 stron)
    |
    +- Dla kazdej oferty:
    |     +- Sprawdz czy listing istnieje (skip jesli tak)
    |     +- Szukaj produktu po SKU
    |     +- Jesli znaleziony -> linkuj (ProductListing)
    |     +- Jesli nie -> utworz produkt z danych oferty + linkuj
    |
    +- Wynik: { total, created, linked, skipped, errors, details[] }
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
              Polling (45s-2min, zalezy od marketplace'u)
                           |
                           v
+----------------------------------------------+
|            MarketplaceProvider                |
|                                              |
|  interface {                                 |
|    ProviderName() -> string                  |
|    PollOrders(ctx, cursor) -> orders         |
|    GetOrder(ctx, externalID) -> order        |
|    PushOffer(ctx, product) -> externalID     |
|    UpdateStock(ctx, offerID, qty)            |
|    UpdatePrice(ctx, offerID, price)          |
|  }                                           |
|                                              |
|  Opcjonalne interfejsy (mixin pattern):      |
|  +- BulkStockUpdater   (batch stock update)  |
|  +- BulkPriceUpdater   (batch price update)  |
|  +- ListingActivator   (publish offer)       |
|  +- ListingDeactivator (withdraw offer)      |
|  +- AsyncStockUpdater  (Amazon feeds)        |
|  +- AsyncPriceUpdater  (Amazon feeds)        |
|  +- AsyncFeedResult    (feed submission ID)  |
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
|    GetRates(ctx, req) -> rates or            |
|      ErrCarrierRatesNotImplemented           |
|  }                                           |
+----------------------------------------------+
```

### Obslugiwane integracje

| Kategoria | Provider | Status |
|-----------|----------|--------|
| **Marketplace** | Allegro | OAuth 2.0, polling, oferty, katalog, messaging, zwroty, spory, oceny, promocje, dostawa |
| | Amazon | SP-API, OAuth2, polling, async feeds (stock/price), feed status polling |
| | WooCommerce | REST API, webhooks, listings, stock sync |
| | eBay | OAuth 2.0, polling, fulfillment, tracking, refundy, 3-step listings, bulk stock/price, import ofert, activate/deactivate |
| | Kaufland | SDK + adapter; **nie** kanal zamowien (brak pollera, ukryte w UI) |
| | OLX | OAuth 2.0, listings (kategorie, miasta, dzielnice, atrybuty), activate/deactivate |
| | Mirakl/Empik | SDK + adapter; **nie** kanal zamowien (brak pollera, ukryte w UI) |
| | Erli | REST API, listings |
| **Carrier** | InPost | Paczkomaty, kurier, Geowidget, webhook, dispatch orders |
| | DHL | Krajowe i miedzynarodowe, adres nadawcy (shipper), DHL24 SOAP WebAPI2 |
| | DPD | Polska, etykiety przez DPD Services REST, tracking przez DPD InfoServices SOAP `getEventsForWaybillV1` |
| | GLS | Europa, adres nadawcy (shipper), ShipIT REST API with Basic Auth |
| | UPS | Miedzynarodowe |
| | Poczta Polska | Paczki |
| | Orlen Paczka | Paczkomaty |
| | FedEx | Miedzynarodowe |
| | Apaczka (broker) | Broker meta-carrier: jedna integracja REST API v2 (HMAC-SHA256) fronting wielu przewoznikow (InPost, DPD, DHL, UPS, GLS, Poczta Polska, Orlen Paczka); tworzenie przesylek, etykiety (waybill PDF), tracking, rate-shopping (service_structure + order_valuation), punkty odbioru |
| **Fakturowanie** | Fakturownia | Faktury VAT |
| **e-Fakturowanie** | KSeF | Krajowy System e-Faktur (wysylka, UPO, status) |
| **Marketing** | Mailchimp | Sync klientow, kampanie |
| **Helpdesk** | Freshdesk | Tickety |
| **Powiadomienia** | SMTP | Email |
| | Twilio/SMSAPI | SMS |
| **AI** | OpenAI | Kategoryzacja, opisy, ulepszanie, tlumaczenie |
| **Kursy walut** | NBP | Narodowy Bank Polski |

Rate shopping dla DPD, GLS, UPS, Poczta Polska, Orlen Paczka, FedEx i DHL jest wylaczony do czasu realnej wyceny kontraktowej/rating API. Providerzy pozostaja dostepni dla obslugiwanych przeplywow, np. etykiet, trackingu lub punktow odbioru, ale nie zwracaja sztucznych stawek. Apaczka jako broker meta-carrier zwraca rzeczywiste stawki dla wszystkich aktywnych uslug (service_structure + order_valuation) i jest w pelni uwzgledniany w endpoint rate-shopping `/v1/shipping/rates`.

DPD uzywa dwoch powierzchni API: DPD Services REST do tworzenia przesylek i etykiet oraz DPD InfoServices SOAP do sledzenia przesylek. Opcjonalne dane `info_login`, `info_password` i `info_channel` pozwalaja zapisac osobne dane InfoServices; gdy sa puste, OpenOMS uzywa glownego loginu/hasla DPD oraz `master_fid` jako kanalu InfoServices.

### Macierz interfejsow marketplace

| Provider | MarketplaceProvider | BulkStock | BulkPrice | Activator | Deactivator | AsyncStock | AsyncPrice |
|----------|:--:|:--:|:--:|:--:|:--:|:--:|:--:|
| Allegro | + | + | + | + | + | — | — |
| eBay | + | + | + | + | + | — | — |
| Amazon | + | — | — | — | — | + | + |
| OLX | + | — | — | + | + | — | — |
| WooCommerce | + | — | — | — | — | — | — |
| Kaufland | + | — | — | — | — | — | — |
| Shoper | + | — | — | — | — | — | — |
| Shopify | + | — | — | — | — | — | — |
| PrestaShop | + | — | — | — | — | — | — |
| Mirakl/Empik | + | — | — | — | — | — | — |
| Erli | + | — | — | — | — | — | — |

---

## 11. Background Workers (32 pliki zrodlowe)

Workery to **tickery w procesie API** (ten sam obraz `openoms-api`, flaga `WORKERS_ENABLED`), nie osobna kolejka asynq. Redis `SET NX` (`openoms:worker-lock:<name>`); blad Redis **pomija** tick. Produkcja wymaga Redis (`ALLOW_IN_MEMORY_STATE=false`). Deployment workera: 1 replica, `Recreate`. Kaufland i Mirakl maja adaptery marketplace, ale **nie maja** order pollera.

### Workery bezwarunkowe (21)

| Worker | Interwal | Cel |
|--------|----------|-----|
| AllegroOrderPoller | 45s | Polling zamowien z Allegro |
| ErliOrderPoller | 45s | Polling zamowien z Erli |
| OLXOrderPoller | 45s | Polling zamowien z OLX |
| WooCommerceOrderPoller | 60s | Polling zamowien z WooCommerce |
| ShoperOrderPoller | 60s | Polling zamowien z Shoper |
| ShopifyOrderPoller | 60s | Polling zamowien z Shopify |
| PrestaShopOrderPoller | 90s | Polling zamowien z PrestaShop |
| EbayOrderPoller | 90s | Polling zamowien z eBay |
| AmazonOrderPoller | 2min | Polling zamowien z Amazon |
| AmazonFeedStatusWorker | 2min | Sprawdzanie statusu feedow Amazon SP-API |
| StockSyncWorker | 5min | Sync stanow do marketplace'ow (BulkStockUpdater / AsyncStockUpdater) |
| PriceSyncWorker | 5min | Sync cen do marketplace'ow |
| ListingSyncWorker | 5min | Push listingow (oferty/ceny/stan). Tick 5min; `SyncIntervalMinutes` jest wewnetrzny. Pull **nie** ustawia `sync_status=synced` bez danych z marketplace: `MarketplaceProvider` nie ma list-offers, wiec `PullListings` zapisuje skip + `last_error` (`provider does not expose offer listing`) |
| KSeFStatusWorker | 5min | Status faktur wyslanych do KSeF |
| TrackingPoller | 10min | Aktualizacja statusu przesylek |
| RepricingWorker | 15min | Automatyczna zmiana cen wg regul |
| DelayedActionWorker | 30s | Opoznione akcje automatyzacji |
| OAuthRefresher | 30min | Tokeny OAuth (Allegro, OLX, Amazon, eBay); credentials AES-256-GCM |
| SegmentRefreshWorker | 1h | Przeliczanie czlonkostwa segmentow |
| RecurringOrderWorker | 1h | Tworzenie zamowien cyklicznych |
| ExchangeRateWorker | 24h | Kursy NBP |

`AllegroWebhookSyncer` jest podpiety do handlera webhookow (event), nie do managera.

#### Workery gated (nie w 21)

| Worker | Interwal | Flaga | Cel |
|--------|----------|-------|-----|
| SupplierSyncWorker | 1min | `SUPPLIER_SYNC_ENABLED` | Sync katalogow dostawcow (XML/IOF/CSV/API) |
| BillingReconciliationWorker | 15min | billing enabled (`STRIPE_SECRET_KEY` + `BILLING_PLANS`) | Naprawia sesje checkout bez rekordow billing |
| OrchestrationWorker | 15s | `ORCHESTRATION_WORKER_ENABLED` | Drenuje orchestration_outbox (FOR UPDATE SKIP LOCKED), dispatch, retry/backoff |
| SupplierOrderStatusPoller | 5min | orchestration + `SUPPLIER_ORDER_ENABLED` | Reconcile statusu zamowien u dostawcy |
| FulfillmentBackfillWorker | 5min | `FULFILLMENT_BACKFILL_ENABLED` (+ dry-run default true) | Backfill procesow dla niefinalnych zamowien |
| FulfillmentGaugeSweeper | 1min | `FULFILLMENT_PROCESS_ENABLED` | Metryki fulfillment |

### Infrastruktura workerow

| Plik | Cel |
|------|-----|
| `manager.go` | Menedzer workerow (rejestracja, start, stop, graceful shutdown) |
| `marketplace_order_poller.go` | Bazowy poller zamowien (wspolna logika dla Allegro/Amazon/WooCommerce/eBay/Shoper/PrestaShop/Shopify) |
| `tenant_iterator.go` | Iterator tenantow -- wykonuje logike per-tenant |
| `distributed_lock.go` | Odnawialne lease Redis (`SET NX`) dla multi-instance: UUID ownership, Lua renewal podczas aktywnego runu i Lua release |
| `integration_error.go` | Wspolna obsluga bledow integracji (np. oznaczanie integracji jako wymagajacej ponownej autoryzacji) |

### Cechy

- Panic recovery: wykonania workerow są chronione przez `safeRun`, a pozostałe asynchroniczne goroutines w API (`webhooks`, WebSocket pumps, memory cleanups, rate shopping, automation dispatch) są uruchamiane przez `asyncutil.SafeGo` z recover + Sentry
- Graceful shutdown: workery sprawdzaja context cancellation miedzy iteracjami tenantow/integracji/listingow i przy rate-limit sleep, zeby shutdown nie startowal kolejnych tenant tasks
- Logowanie bledow per worker (slog)
- Interfejs Worker: `Name()`, `Interval()`, `Run(ctx)`
- Iteracja per-tenant (kazdy worker dziala dla wszystkich aktywnych tenantow)
- Idempotentny import zamowien marketplace/BaseLinker: atomowy insert-or-skip oparty o czesciowy unikalny indeks `orders(tenant_id, source, external_id)` dla niepustych `external_id`
- Cross-tenant queries uzywaja jawnego `WORKER_DATABASE_URL` poza developmentem; API `DATABASE_URL` pozostaje least-privilege/RLS-scoped; privileged worker pool ma konserwatywny limit polaczen, zeby nie wyczerpywac session poolera podczas blue-green deployow

---

## 12. Automatyzacja

### Trigger events

| Event | Kiedy |
|-------|-------|
| `order.created` | Nowe zamowienie |
| `order.updated` | Aktualizacja zamowienia |
| `order.status_changed` | Zmiana statusu zamowienia |
| `shipment.created` | Nowa przesylka |
| `shipment.status_changed` | Zmiana statusu przesylki |
| `return.created` | Nowy zwrot |
| `return.status_changed` | Zmiana statusu zwrotu |
| `product.created` / `product.updated` | Katalog |
| `product.stock_restored` / `product.out_of_stock` | Stan magazynowy |

### Warunki (conditions)

```json
[
  { "field": "total_amount", "operator": "gte", "value": 500 },
  { "field": "tags", "operator": "contains", "value": "vip" },
  { "field": "source", "operator": "eq", "value": "allegro" },
  { "field": "priority", "operator": "eq", "value": "high" }
]
```

Operatory: `eq`, `neq`, `gt`, `gte`, `lt`, `lte`, `contains`, `not_contains`, `in`, `not_in`, `starts_with`. Puste warunki = true (AND wszystkich).

### Akcje (actions)

| Typ akcji | Opis |
|-----------|------|
| `set_status` | Zmiana statusu zamowienia. Domyslnie **respektuje graf statusow**; pominiecie grafu wymaga jawnego `params.force = true` (przyjmowane tez jako string `"true"`) i jest przenoszone przez payload outboxa. Alias w UI: `transition_status` |
| `send_email` | Wyslanie emaila |
| `add_tag` | Dodanie tagu |
| `create_invoice` | Utworzenie faktury |
| `activate_listing` / `deactivate_listing` | Publikacja / wycofanie oferty |
| `send_marketplace_message` | Wiadomosc marketplace (Allegro) |
| `webhook` | Wywolanie custom webhook |
| `external_workflow` | Gated (`EXTERNAL_WORKFLOW_ENABLED`) |
| `delay` | Zapis do `automation_delayed_actions` (wykonuje DelayedActionWorker) |

### Routing akcji zmieniających stan przez orchestrator (OPE-421, opcjonalne)

Gdy `AUTOMATION_ORCHESTRATION_ENABLED=true`, akcja `set_status` (`transition_status` dla zamówień) nie woła bezpośrednio `TransitionStatus`, lecz **enqueue'uje** event `automation.set_status` do orchestration outboxa (sekcja 8). Transition wykonuje idempotentnie `AutomationStatusTransitionHandler` (no-op gdy zamówienie już w target statusie → zrywa burzę efektów ubocznych i rekurencję `order.status_changed`); permanentny fail → blocker `automation_action_failed` widoczny w ops dashboardzie. Domyślnie OFF → akcja działa bezpośrednio jak dotąd. Pozostałe typy akcji są niezmienione.

### Opoznione akcje (delayed actions)

Akcja `delay` w regule automatyzacji tworzy wpis w tabeli `automation_delayed_actions` z polem `execute_at`. Worker `DelayedActionWorker` co 30 sekund sprawdza, czy sa akcje do wykonania i je realizuje. Bledy wykonania akcji sa zapisywane w polu `error`, zwiekszaja `attempt_count` i requeue'uja akcje z exponential backoff (1m, 2m, 4m, 8m) do maksymalnie 5 prob; dopiero po wyczerpaniu prob wpis jest oznaczany jako wykonany z bledem. Pozwala to na scenariusze typu:

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
ENV=production|staging|development
REDIS_URL=redis://redis-master.apps-core.svc.cluster.local:6379
ALLOW_IN_MEMORY_STATE=false  # true tylko dla jawnego single-node/self-host bez Redis

# -- Baza danych ------------------
DATABASE_URL=postgres://openoms_app:pass@localhost:5433/openoms        # app role, RLS-scoped
WORKER_DATABASE_URL=postgres://openoms_worker:pass@localhost:5433/openoms  # privileged, cross-tenant worker/webhook queries

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
# Dashboard domyslnie wywoluje API po tym samym originie (/v1).
# NEXT_PUBLIC_API_URL ustawiaj tylko dla niestandardowych wdrozen z osobnym originem API.
NEXT_PUBLIC_API_URL=

# -- Workers ----------------------
WORKERS_ENABLED=true

# -- Fulfillment orchestration (tor B, wszystkie domyslnie OFF/bezpieczne) --
# Kolejnosc wlaczania w rolloucie (sekcja 8, OPE-423):
SEED_PROVIDERS=false                  # seed rejestru providerow (OPE-412), idempotentny po provider_key
FULFILLMENT_PROCESS_ENABLED=false     # recording: proces + outbox event przy tworzeniu zamowienia (OPE-416)
FULFILLMENT_BACKFILL_ENABLED=false    # worker backfillu procesow dla istniejacych zamowien (OPE-423)
FULFILLMENT_BACKFILL_DRY_RUN=true     # backfill tylko liczy; zapis wymaga ENABLED=true I DRY_RUN=false
ORCHESTRATION_WORKER_ENABLED=false    # OrchestrationWorker drenuje outbox + dispatch (OPE-415)
SUPPLIER_ORDER_ENABLED=false          # silnik zamowien do dostawcy: submit handler + status poller (OPE-418/Phase-7)
AUTOMATION_ORCHESTRATION_ENABLED=false # set_status automatyzacji przez outbox (OPE-421, na koncu)
# Dashboard (build-time, cutover UI po osiagnieciu parity — OPE-423):
NEXT_PUBLIC_OPENOMS_FULFILLMENT_DASHBOARD=heuristic  # heuristic[default]|process-backed

# -- Monitoring -------------------
METRICS_TOKEN=...                    # Bearer token dla /metrics (openssl rand -base64 32)

# -- Integracje (opcjonalne) ------
INPOST_API_TOKEN=...
INPOST_ORG_ID=...
ALLEGRO_WEBHOOK_SECRET=...           # Legacy fallback dla /v1/webhooks/allegro; preferowany scoped sekret per integracja
INPOST_WEBHOOK_SECRET=...            # Legacy fallback dla /v1/webhooks/inpost; preferowany scoped sekret per integracja
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

| Metryka | Wartosc (stan na 2026-08-23, `origin/main`) |
|---------|--------|
| **Tabele DB** | 89 |
| **Migracje SQL** | 50 wersji (`000001`–`000050`) |
| **Endpointy API** | ~540 (`/v1/openapi.yaml`) |
| **Strony frontend** | 139 `page.tsx` |
| **Komponenty React** | 151 plikow prod `.tsx` |
| **Custom hooks** | 83 (`use-*`) |
| **Handlery Go** | 105 plikow (non-test) |
| **Serwisy Go** | 104 plikow (non-test) |
| **Repozytoria Go** | 63 plikow (non-test) |
| **Background workers** | 21 bezwarunkowych + gated (32 pliki zrodlowe) |
| **Middleware** | 23 pliki |
| **Pakiety SDK** | 28 |
| **Testy Go (api-server)** | 286 `*_test.go` |
| **E2E Playwright** | 26 specow |
| **Jezyki** | Go, TypeScript, SQL |
| **Licencja** | Elastic License 2.0 (apps) + MIT (packages) |

### Testy

| Typ testu | Status |
|-----------|--------|
| Backend Go tests | PASS |
| API contract (TS <-> Go) | PASS |
| Load testing | 0 bledow, 1000-1800 req/s |
| RLS isolation | PASS |
| Clean migration | PASS |

---

*Dokument zaktualizowany: 2026-08-23 (odswiezenie liczb i faktow z kodu na `origin/main`)*
*Wersja: OpenOMS v3.5*
