# OPE-247: Feature Readiness Matrix

Data: 2026-05-11
Zakres: dashboard OpenOMS, moduły widoczne w nawigacji, integracje, providery, funkcje wymagające zewnętrznych kont.
Tryb: audyt statyczny + read-only browser audit na `https://app.openoms.org/`.

## Cel

Ta macierz ma uporządkować bałagan funkcjonalny przed wejściem do klientów. Każda funkcja dostaje status, decyzję ekspozycji i konkretny test odblokowujący. Domyślna zasada: jeśli nie mamy dowodu, że flow działa end-to-end, funkcja nie jest widoczna dla klienta produkcyjnego.

## Statusy

| Status | Znaczenie | Decyzja UI |
|---|---|---|
| `ready` | Flow jest sensowne dla pierwszego klienta i ma wystarczający dowód działania. | Widoczne. |
| `controlled` | Funkcja może działać, ale wymaga per-klientowej konfiguracji, credentiali albo checklisty operatora. | Ukryte domyślnie; włączane świadomie. |
| `verify` | Kod istnieje, ale trzeba przejść test E2E przed decyzją. | Ukryte do czasu walidacji. |
| `beta` | Częściowe, eksperymentalne albo bez produkcyjnej certyfikacji. | Ukryte w client-ready mode. |
| `blocked` | Znany placeholder, brak implementacji albo niespójność frontend/backend. | Wyłączone w UI i zabezpieczone w API. |

## Minimalna checklista dla każdej funkcji

Funkcja przechodzi na `ready` albo `controlled` dopiero po odhaczeniu:

- UI: ekran nie ma placeholderów typu `Tytuł`, surowych kluczy tłumaczeń, martwych CTA ani niejasnych pustych stanów.
- API: wszystkie akcje widoczne w UI mają działające endpointy albo czytelny, zamierzony disabled state.
- Dane: flow działa na tenant-scoped danych, bez przecieków między tenantami.
- Uprawnienia: owner/admin/member widzą tylko to, co powinni.
- Błędy: błąd credentiali, limitu API, timeoutu i braku danych ma jasny recovery path.
- Test: minimum smoke E2E na dev/staging albo sandboxie; dla providerów zewnętrznych także test credentiali.
- Operacje: wiadomo, jak cofnąć, odtworzyć albo ręcznie naprawić nieudaną akcję.

## Macierz modułów dashboardu

### Core i onboarding

| Funkcja | Trasa | Status | Ekspozycja | Co sprawdzić / odblokowuje |
|---|---|---|---|---|
| Pulpit | `/` | `ready` | Pokazać | Dashboard ładuje KPI, onboarding i empty states bez błędów konsoli. |
| Onboarding pierwszych kroków | `/`, `/onboarding` | `ready` | Pokazać | Firma -> Allegro -> produkt -> zamówienie, bez martwych linków. |
| Pomoc | `/help` | `ready` | Pokazać | Linki prowadzą do aktualnych instrukcji, bez obietnic beta funkcji. |
| Command palette | global | `verify` | Ograniczyć | Musi respektować readiness, bo dziś odkrywa moduły ukrywane w menu. |

### Sprzedaż

| Funkcja | Trasa | Status | Ekspozycja | Co sprawdzić / odblokowuje |
|---|---|---|---|---|
| Zamówienia | `/orders`, `/orders/new`, `/orders/[id]` | `ready` | Pokazać | CRUD manual order, status transitions, audit trail, export, bulk status. |
| Import zamówień CSV | `/orders/import` | `verify` | Ukryć lub pokazać po teście | Preview, mapowanie kolumn, import, rollback błędnych wierszy. |
| Klienci | `/customers`, `/customers/new`, `/customers/[id]` | `ready` | Pokazać | CRUD, historia zamówień, brak błędów przy pustych danych. |
| Segmenty klientów | `/customers/segments` | `beta` | Ukryć | Reguły segmentów, podgląd członków, aktualizacja po zamówieniach. |
| Zwroty | `/returns`, `/returns/new`, `/returns/[id]` | `ready` | Pokazać | Utworzenie RMA, zmiana statusu, public return token. |
| Faktury | `/invoices`, `/invoices/[id]` | `controlled` | Ukryć domyślnie | Utworzenie faktury, PDF, anulowanie, relacja z zamówieniem. |
| Fakturowanie providerów | `/invoicing`, `/settings/accounting` | `controlled` | Ukryć domyślnie | Test Fakturownia; wFirma/inFakt osobno jako beta. |

### Katalog

| Funkcja | Trasa | Status | Ekspozycja | Co sprawdzić / odblokowuje |
|---|---|---|---|---|
| Produkty | `/products`, `/products/new`, `/products/[id]` | `ready` | Pokazać | CRUD, warianty, zdjęcia, stock, relacje z kategoriami. |
| Import produktów CSV | `/products/import` | `verify` | Ukryć lub pokazać po teście | Preview, mapowanie pól, import, deduplikacja SKU/EAN. |
| Kategorie produktów | `/settings/product-categories` | `ready` | Pokazać | Tree CRUD, przypisanie do produktu, brak pustych placeholderów. |
| Warianty produktów | `/products/[id]/variants` | `verify` | Ukryć z onboarding path | Dodawanie wariantu, cena override, stock, listing compatibility. |
| Listing produktu | `/products/[id]/listings` | `controlled` dla Allegro/OLX, `blocked` dla części providerów | Ograniczyć providerami | Guard na Shoper/PrestaShop/Shopify/Amazon/Kaufland/Mirakl; test Allegro offer. |
| Szablony druku | `/settings/print-templates` | `verify` | Ukryć domyślnie | Edycja, podgląd, druk zamówienia/etykiety bez błędów. |

### Logistyka i magazyn

| Funkcja | Trasa | Status | Ekspozycja | Co sprawdzić / odblokowuje |
|---|---|---|---|---|
| Przesyłki | `/shipments`, `/shipments/new`, `/shipments/[id]` | `controlled` | Pokazać po skonfigurowaniu kuriera | Create shipment, label download, tracking, cancel. |
| Kurierzy | `/carriers`, `/carriers/new`, `/carriers/[id]` | `controlled` | Ukryć domyślnie; włączać per provider | Provider readiness, credential test, brak providerów beta. |
| InPost | provider | `ready` dla labels/tracking/points, `blocked` dla rates | Pokazać po credentialach; rate shopping ukryty | Manager InPost, token, organization_id, label, tracking, paczkomat; kontraktowe źródło wyceny wymagane przed pokazaniem stawek. |
| DHL | provider | `controlled` | Włączyć po checkliście | Konto DHL, SOAP/API login, hasło, account number, label, tracking. |
| DPD | provider | `controlled` z caveatem | Włączyć po checkliście | Konto DPD, login, hasło, master_fid; nie obiecywać REST tracking. |
| GLS | provider | `controlled` | Włączyć po checkliście | Konto GLS/API key, label storage, tracking. |
| UPS/FedEx/Poczta/Orlen | provider | `beta` | Ukryć | Konta partnerskie + test create/label/tracking/rates. |
| Pakowanie | `/packing` | `verify` | Ukryć lub pilotaż | Skanowanie SKU/EAN, potwierdzenie pakowania, zachowanie przy brakach. |
| Pick & Pack | `/pick-pack` | `verify` | Ukryć lub pilotaż | Lista picking, skanowanie, kompletacja, statusy zamówień. |
| Magazyny | `/settings/warehouses` | `controlled` | Ukryć domyślnie | CRUD magazynów, default warehouse, stock per warehouse. |
| Inwentaryzacja | `/stocktakes` | `verify` | Ukryć | Start, count items, complete/cancel, korekta stocku. |
| Dokumenty magazynowe | `/settings/warehouse-documents` | `verify` | Ukryć | PZ/WZ/MM create, confirm, cancel, wpływ na stock. |
| Sync magazynu | `/stock-sync` | `beta` | Ukryć | Rules, events, marketplace stock push, idempotencja. |

### Kanały sprzedaży i providery marketplace

| Funkcja/provider | Trasa | Status | Ekspozycja | Co sprawdzić / odblokowuje |
|---|---|---|---|---|
| Marketplace hub | `/marketplaces`, `/marketplaces/new` | `controlled` | Pokazać tylko ready/controlled providers | Picker musi filtrować beta/blocked i nie renderować kluczy `providers.*.description`. |
| Allegro core | `/marketplaces/allegro` | `ready` core, `controlled` advanced | Pokazać core | Developer app, OAuth, import orders/offers, basic offer flow. |
| Allegro advanced | `/marketplaces/allegro/*` | `controlled` | Ukryć zakładki do checklist | Fulfillment, shipments, returns, disputes, ratings, promotions, policies. |
| OLX | `/marketplaces/olx` | `controlled` | Włączyć per klient | Developer app/OAuth, listing import/create; brak stock sync promise. |
| Amazon | `/marketplaces/amazon` | `beta` | Ukryć | Seller Central/SP-API, refresh token, seller/marketplace IDs, feed verification; direct PushOffer nie działa. |
| eBay | `/marketplaces/ebay` | `beta` | Ukryć | Developer app, OAuth, listing/order E2E. |
| WooCommerce | generic provider | `beta` | Ukryć | Demo Woo store, consumer key/secret, order/product/listing E2E. |
| Shopify | `/marketplaces/shopify` | `blocked` dla listing create, `beta` order sync | Ukryć | Real PushOffer zamiast synthetic ID. |
| PrestaShop | `/marketplaces/prestashop` | `blocked` dla listing create, `beta` order sync | Ukryć | Real PushOffer + poprawa parse błędów money. |
| Shoper | `/marketplaces/shoper` | `blocked` dla listing create, `beta` order sync | Ukryć | Real PushOffer zamiast synthetic ID. |
| Kaufland | picker/backend | `blocked` direct listing, `beta` reszta | Ukryć | PushOffer implementacja albo UI guard. |
| Erli | provider | `beta` | Ukryć | Sandbox/live base URL, order/listing E2E. |
| Empik/Mirakl | picker/backend | `blocked` | Ukryć | Decyzja nazewnictwa i alias `empik`/`mirakl`; factory musi rozpoznawać provider. |
| Feedy produktowe | `/settings/feeds` | `beta` | Ukryć | Feed generation/import E2E i jasne formaty. |
| Synchronizacja ofert | `/listing-sync` | `beta` | Ukryć | Provider allowlist, dry-run, idempotent sync, event log. |

### Raporty i automaty

| Funkcja | Trasa | Status | Ekspozycja | Co sprawdzić / odblokowuje |
|---|---|---|---|---|
| Raporty podstawowe | `/reports` | `verify` | Ukryć lub tylko admin | Dane liczbowe zgodne z API, empty/error states. |
| Prognoza popytu | `/reports/forecast` | `beta` | Ukryć | Dane sprzedaży, algorytm, generowanie zamówienia zakupu. |
| Ślad węglowy | `/reports/carbon` | `beta` | Ukryć | Źródło danych emisji, eksport CSV, brak pustych wykresów. |
| VAT OSS report | `/reports/vat-oss` | `beta` | Ukryć | Kraje UE, waluty, faktury, eksport, zgodność księgowa. |
| Rozliczenia | `/reconciliation` | `beta` | Ukryć | Integracja payment gateway, match z zamówieniem, manual review. |
| Repricing | `/repricing` | `beta` | Ukryć | Reguły, dry-run, guard przed masową zmianą ceny. |
| Automatyzacja rules | `/settings/automation` | `controlled` | Ukryć domyślnie | Trigger -> condition -> action, delayed actions, audit log. |
| Workflow Builder | `/workflows` | `beta` | Ukryć | Czy zapisany workflow faktycznie wykonuje akcje; brak martwego edytora. |

### Zaopatrzenie

| Funkcja | Trasa | Status | Ekspozycja | Co sprawdzić / odblokowuje |
|---|---|---|---|---|
| Dostawcy | `/suppliers`, `/suppliers/new`, `/suppliers/[id]` | `controlled` | Ukryć domyślnie | Naprawić placeholder `Tytuł`; test BTP/XML/import produktów. |
| BTP.pro | `/suppliers/new/btp` | `controlled` | Włączyć tylko jeśli klient używa BTP | Konto BTP, XML URL, public/private key, import, API stock/price. |
| Supplier portal | `/supplier-portal` | `verify` | Ukryć z głównej nawigacji | Token handoff, potwierdzenie PO, wiadomości, brak token leak. |
| Zamówienia zakupu | `/purchase-orders` | `verify` | Ukryć domyślnie | Create/send/receive/cancel, wpływ na stock. |
| Dropshipping | `/dropship-orders` | `beta` | Ukryć | Forward order to supplier, tracking, supplier portal, status updates. |

### Narzędzia i ustawienia

| Funkcja | Trasa | Status | Ekspozycja | Co sprawdzić / odblokowuje |
|---|---|---|---|---|
| Firma | `/settings/company` | `ready` | Pokazać | Dane firmy, NIP, adres, użycie w fakturach/przesyłkach. |
| Użytkownicy | `/settings/users` | `ready` | Pokazać owner/admin | Invite/create/edit/deactivate, role assignment. |
| Role | `/settings/roles` | `ready` | Pokazać owner/admin | RBAC permissions, owner lockout guard. |
| Bezpieczeństwo | `/settings/security` | `ready` | Pokazać | Password/2FA/session settings, brak placeholderów. |
| Statusy zamówień | `/settings/order-statuses` | `controlled` | Ukryć domyślnie | Custom statuses + state machine impact. |
| Pola niestandardowe | `/settings/custom-fields` | `controlled` | Ukryć domyślnie | Field definition, validation, render in forms/tables. |
| Cenniki | `/settings/price-lists` | `verify` | Ukryć | Price list application in orders/products. |
| Kontrola magazynowa | `/settings/inventory` | `verify` | Ukryć | Min stock, reservations, warehouse stock. |
| Waluty | `/settings/currencies` | `verify` | Ukryć | NBP fetch, conversion, stale-rate handling. |
| Subskrypcja/billing | `/settings/billing` | `controlled` | Ukryte dla pierwszego klienta; dostęp tylko w `full`/operator validation | Stripe checkout/subscription/webhook E2E, manual/enterprise contract copy, statusy inactive/payment, brak linków do ukrytego route z bannerów. |
| SMS | `/settings/sms`, `/settings/notifications` | `blocked` | Ukryć | Naprawić redirect/copy `Tytuł`/`Hasło DHL`; SMSAPI credential test. |
| Webhooki | `/settings/webhooks` | `controlled` | Ukryć domyślnie | Signing, SSRF guard, delivery retry, test endpoint. |
| Dostawy webhooków | `/settings/webhooks/deliveries` | `controlled` | Ukryć domyślnie | Retry, response logging, redaction. |
| Sync jobs | `/settings/sync-jobs` | `controlled` | Ukryć domyślnie | Czytelne statusy, retry, operator-only. |
| Dziennik aktywności | `/audit` | `ready` dla admin/operator | Ukryć dla zwykłych userów | Filtrowanie, pagination, brak danych wrażliwych. |
| KSeF | `/settings/ksef` | `beta` | Zostawić ukryte | Certyfikat/token, test env, status/UPO. |
| Marketing | `/settings/marketing` | `beta` | Zostawić ukryte | Mailchimp account, campaign flow. |
| Helpdesk | `/settings/helpdesk` | `beta` | Zostawić ukryte | Freshdesk account, ticket creation, order link. |
| Usuwanie tła | `/tools/bg-removal` | `beta` | Ukryć | Provider AI/API, koszty, limity, file handling. |
| Subskrypcje/recurring orders | `/recurring-orders` | `beta` | Ukryć | Scheduler, generated order, cancellation. |
| Loyalty | `/loyalty` | `beta` | Ukryć | Rules, customer rewards, order integration. |

## Konta i credentiale wymagane do certyfikacji

| Provider | Wymagane konto/dostęp | Minimalny test bezpieczny | Status startowy |
|---|---|---|---|
| Allegro | Konto sprzedawcy + aplikacja Allegro Developer + redirect URI + client ID/secret | OAuth, import zamówień/ofert, odświeżenie tokenu, jedna niskoryzykowna operacja oferty | `ready` core / `controlled` advanced |
| OLX | Konto OLX + aplikacja OAuth + client ID/secret | OAuth, odczyt ogłoszeń, utworzenie/edycja testowego ogłoszenia | `controlled` |
| Amazon | Seller Central + SP-API app + refresh token + marketplace/seller IDs | Token refresh, order read, feed dry-run albo sandbox | `beta` |
| eBay | eBay Developer app + app/cert/dev IDs + OAuth refresh token | OAuth, order/listing read, sandbox offer | `beta` |
| WooCommerce | Testowy sklep WooCommerce + consumer key/secret | Order/product/listing CRUD na demo sklepie | `beta` |
| Shopify | Shopify store + access token/scopes | Credential verification + order read; listing disabled do real PushOffer | `blocked` listing / `beta` sync |
| PrestaShop | Sklep PrestaShop + Webservice key/uprawnienia | Credential verification + order read; listing disabled do real PushOffer | `blocked` listing / `beta` sync |
| Shoper | Sklep Shoper + WebAPI login/hasło | Credential verification + order read; listing disabled do real PushOffer | `blocked` listing / `beta` sync |
| Kaufland | Konto seller + API key/secret | Product/order read; listing disabled do PushOffer | `blocked` listing / `beta` |
| Erli | Konto seller + API token + sandbox/live base URL | Product/order/listing E2E | `beta` |
| Empik/Mirakl | Konto Mirakl/Empik + API key + base URL | Najpierw naprawić alias providerów, potem order/listing read | `blocked` |
| InPost | Manager Paczek + API token + organization ID + opcjonalnie GeoWidget token | Create shipment, label PDF, tracking, paczkomat | `ready` labels/tracking |
| DHL | Konto biznesowe + API/SOAP username/password + account number | Create shipment, label, tracking na sandbox/live | `controlled` |
| DPD | Konto DPD + login/password + master_fid | Create shipment i label; tracking jako unsupported albo inna integracja | `controlled` |
| GLS | Konto GLS/API key | Create shipment, label storage, tracking | `controlled` |
| UPS | UPS Developer app + client ID/secret + account | Create shipment, label, tracking, rates | `beta` |
| FedEx | FedEx Developer app + client ID/secret + account number | Create shipment, label, tracking, rates | `beta` |
| Poczta Polska | Konto partnerskie/API key + partner ID | Create shipment, label, tracking jeśli dostępne | `beta` |
| Orlen Paczka | Konto partnerskie/API key + partner ID | Create shipment, pickup point, label, tracking | `beta` |
| BTP.pro | Panel BTP + XML URL + public/private key + opcjonalnie API URL | Import XML, mapowanie, API stock/price | `controlled` |
| Fakturownia | Subdomena + API token | Create invoice, PDF, cancel/test correction | `controlled` |
| wFirma | API key/account | Create/get/PDF; cancel unsupported | `beta` |
| inFakt | API key/account | Create/get/PDF; cancel unsupported | `beta` |
| KSeF | Test/prod auth token/certyfikat + dane firmy | Send/status/UPO w środowisku testowym | `beta` |
| SMSAPI | Konto SMSAPI + token + sender name | Send test SMS do kontrolowanego numeru | `blocked` do naprawy UI |
| Mailchimp | Account + API key/list ID | Sync customer + create test campaign | `beta` |
| Freshdesk | Account + API key/domain | Create ticket from order + status sync | `beta` |

## Kolejność pracy

1. P0: natychmiast ukryć `blocked` i oczywiste `beta` z menu, command palette i pickerów.
2. P0: naprawić produkcyjne placeholdery `Tytuł`, surowe `providers.*.description`, `Brak kurierow`, `Hasło DHL` na ekranie SMS/notifications.
3. P1: zbudować jeden readiness registry w dashboardzie i użyć go w nawigacji, pickerach providerów oraz direct route access.
4. P1: dodać backend guards na niedostępne akcje providerów, szczególnie listing create.
5. P2: przejść checklisty core: zamówienia, produkty, klienci, zwroty, firma, użytkownicy, role, security.
6. P2: przejść provider certification: Allegro, InPost, potem DHL/DPD/GLS/BTP/Fakturownia według potrzeb pierwszego klienta.
7. P3: każdy kolejny moduł odblokowywać dopiero po wypełnieniu checklisty i dopisaniu dowodu do tej macierzy.

## Proponowany client-ready allowlist

Na dziś:

- Pulpit
- Pomoc
- Zamówienia
- Klienci
- Zwroty
- Produkty
- Kategorie
- Przesyłki tylko po skonfigurowaniu `InPost`
- Marketplace tylko `Allegro core`
- Firma
- Użytkownicy
- Role
- Bezpieczeństwo

Opcjonalne, jeśli pierwszy klient ich potrzebuje i przejdą checklistę:

- OLX
- DHL
- DPD
- GLS
- BTP.pro
- Fakturownia
- Audit log dla admin/operatora
