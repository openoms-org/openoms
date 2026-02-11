# QA Report: OpenOMS — Test z perspektywy sprzedawcy

**Data**: 2026-02-11
**Tester**: AI (rola: owner MercPart, rafal@mercpart.pl)
**Metoda**: API curl + analiza kodu frontend
**Zakres**: Wszystkie moduły systemu

---

## Podsumowanie

| Severity | Ilość | Opis |
|----------|-------|------|
| CRITICAL | 8 | Crash, utrata danych, niedziałające kluczowe funkcje |
| HIGH | 9 | Poważne bugi, złe walidacje, 500 errors |
| MEDIUM | 14 | UX, niespójności, brakujące guardy |
| LOW | 12 | Drobne UX, kosmetyka, edge cases |
| **TOTAL** | **43** | |

---

## CRITICAL (8)

### C1. Merge orders zwraca 500 — NOT NULL constraint violation
- **Endpoint**: `POST /v1/orders/merge`
- **Opis**: `MergeOrders()` tworzy nowy Order bez ustawienia `Metadata` (jsonb NOT NULL) i `OrderedAt`
- **Plik**: `apps/api-server/internal/service/order_group_service.go:94-115`

### C2. Split order zwraca 500 — ten sam problem co merge
- **Endpoint**: `POST /v1/orders/{id}/split`
- **Plik**: `apps/api-server/internal/service/order_group_service.go:207-222`

### C3. Import zamówień — "commit unexpectedly resulted in rollback"
- **Endpoint**: `POST /v1/orders/import`
- **Opis**: Preview działa, ale faktyczny import kończy się 500

### C4. Public return self-service kompletnie nie działa — RLS bypass broken
- **Endpoint**: `POST /v1/public/returns/`, `GET /v1/public/returns/{token}`
- **Opis**: `queryWithoutRLS()` nie ustawia `app.current_tenant_id`, więc RLS FORCE blokuje wszystkie zapytania
- **Plik**: `apps/api-server/internal/handler/public_return_handler.go:204-215`

### C5. Company settings PUT nadpisuje WSZYSTKO — brak merge
- **Endpoint**: `PUT /v1/settings/company`
- **Opis**: Wysłanie `{"name":"X"}` kasuje company_name, address, NIP, phone, email, website
- **Plik**: `apps/api-server/internal/handler/settings_handler.go:171-200`

### C6. Helpdesk/Marketing save niszczy dane firmy
- **Plik**: `apps/dashboard/src/app/(dashboard)/settings/helpdesk/page.tsx:56-73`
- **Opis**: `PUT /v1/settings/company` z `{"freshdesk": form}` nadpisuje WSZYSTKIE inne pola

### C7. Stored XSS przez Product PATCH — brak sanityzacji
- **Endpoint**: `PATCH /v1/products/{id}`
- **Opis**: Create stripuje HTML tagi, ale Update nie — `<script>` tagowane w description_long trafiają do DB
- **Plik**: `apps/api-server/internal/service/product_service.go:154-203` (brak strippera)

### C8. AuthProvider — infinite loading na network error / 429
- **Plik**: `apps/dashboard/src/components/providers/auth-provider.tsx:25-33`
- **Opis**: Przy błędzie sieci lub 429, `isLoading` nigdy nie jest ustawiane na `false` — wieczny skeleton

---

## HIGH (9)

### H1. Top products endpoint zwraca 500
- **Endpoint**: `GET /v1/stats/products/top`
- **Opis**: `jsonb_to_recordset(orders.items)` failuje na zamówieniach z niestandardowym formatem items

### H2. Duplikat SKU akceptowany — brak UNIQUE constraint
- **Endpoint**: `POST /v1/products/`
- **Opis**: `idx_products_sku` to zwykły INDEX, nie UNIQUE — `isDuplicateKeyError()` nigdy nie triggeruje
- **Plik**: `apps/api-server/migrations/000001_init_schema.up.sql`

### H3. Allegro OAuth — field name mismatch (url vs auth_url)
- **Opis**: API zwraca `{"url": "..."}`, frontend oczekuje `{"auth_url": "..."}`
- **Pliki**: `apps/api-server/internal/handler/allegro_auth_handler.go:67` vs `apps/dashboard/src/components/integrations/allegro-connect.tsx:29`

### H4. Allegro OAuth redirect_uri ma extra `/api` prefix
- **Opis**: `redirect_uri=.../api/v1/integrations/allegro/callback` — router ma `/v1/...` bez `/api`

### H5. Helpdesk tickets — 500 zamiast pustej listy gdy Freshdesk nie skonfigurowany
- **Endpoint**: `GET /v1/helpdesk/tickets`

### H6. Warehouse stock/docs — 500 na FK violation zamiast 400
- **Opis**: Nieistniejący product_id zwraca generyczny 500 zamiast "product not found"

### H7. Backend default order status labels — brak polskich ogonków
- **Plik**: `apps/api-server/internal/model/order.go:330-343`
- **Opis**: "Wyslane", "Zakonczone", "Zwrocone" zamiast "Wysłane", "Zakończone", "Zwrócone"

### H8. Shipment form Zod schema — brak "fedex" provider
- **Plik**: `apps/dashboard/src/components/shipments/shipment-form.tsx:25`
- **Opis**: Dropdown pokazuje FedEx, ale Zod walidacja go odrzuca

### H9. Invoice type — API field `invoice_type`, ale `INVOICE_TYPE_LABELS` nie ma "standard"
- **Opis**: Faktury z `invoice_type: "standard"` wyświetlają surowy tekst w UI

---

## MEDIUM (14)

### M1. Sidebar double-highlight /orders i /orders/import
- **Plik**: `apps/dashboard/src/components/layout/sidebar.tsx:33-35`

### M2. Breadcrumbs — brakuje 17 tłumaczeń segmentów
- **Plik**: `apps/dashboard/src/components/layout/breadcrumbs.tsx:7-27`

### M3. Niespójne nazwy audit page: "Dziennik" (nav) vs "Audyt" (breadcrumb) vs "Dziennik aktywności" (heading)

### M4. Login response — zero-value dates (0001-01-01T00:00:00Z) i null settings
- **Opis**: `FindForAuth` i `FindBySlug` nie selectują timestamps — refresh zwraca poprawne dane

### M5. Warehouse documents filter wysyła literalny string "all" do API
- **Plik**: `apps/dashboard/src/app/(dashboard)/settings/warehouse-documents/page.tsx:106-128`

### M6. Brak AdminGuard na: Price Lists, Sync Jobs, Webhook Deliveries, Reports, Packing, Audit
- **Opis**: 6 stron dostępnych dla zwykłych userów, mimo że zawierają dane admin-only

### M7. Customer email — brak walidacji formatu (akceptuje "not-an-email")
- **Plik**: `apps/api-server/internal/model/customer.go:43-66`

### M8. Customer address field ignorowany — API ma `default_shipping_address` (JSON), nie `address`

### M9. Warehouse documents new — ręczne wpisywanie UUID produktu zamiast search/selector

### M10. Warehouse document detail — surowe UUID zamiast nazw (magazyn, dostawca, zamówienie)

### M11. Order form source selector — 3 opcje zamiast 9
- **Plik**: `apps/dashboard/src/components/orders/order-form.tsx:303-315`

### M12. Merge dialog nie waliduje czy zamówienia są od tego samego klienta

### M13. Print endpoints otwierane przez `window.open()` — brak Bearer token w nowym oknie

### M14. Marketing settings — `loadSettings()` w useEffect nic nie robi (martwy kod)
- **Plik**: `apps/dashboard/src/app/(dashboard)/settings/marketing/page.tsx:49-60`

---

## LOW (12)

### L1. "Dashboard" w nav — angielski, reszta po polsku
- **Plik**: `apps/dashboard/src/lib/nav-items.ts:45`

### L2. has_session cookie bez Max-Age — znika po zamknięciu przeglądarki
- **Opis**: refresh_token ma 30 dni, has_session to session cookie

### L3. Pie chart — 3 kolory na 9 źródeł zamówień
- **Plik**: `apps/dashboard/src/components/dashboard/order-source-chart.tsx:21`

### L4. Revenue chart "ostatnie 30 dni" — ale API nie filtruje po dacie

### L5. Trends data — brak punktów dla dni bez zamówień (gap w wykresie)

### L6. Bulk actions — zawsze "zamówień" (brak odmiany polskiej: zamówienie/zamówienia/zamówień)
- **Plik**: `apps/dashboard/src/components/orders/bulk-actions.tsx:94`

### L7. Listings page — stub redirect, zero funkcjonalności
- **Plik**: `apps/dashboard/src/app/(dashboard)/products/[id]/listings/page.tsx`

### L8. Bundle stock dla non-bundle — zwraca `{"stock": 0}` zamiast błędu

### L9. Return detail — 3-kolumnowy grid z pustą trzecią kolumną
- **Plik**: `apps/dashboard/src/app/(dashboard)/returns/[id]/page.tsx:223`

### L10. Invoice cancel zwraca 204 bez body — niespójne z innymi endpointami

### L11. Logout bez auth — zwraca 200 zamiast 401

### L12. Refresh token nie invalidowany po logout — session hijacking risk

---

## Brakujące polskie ogonki (COSMETIC — ~30 plików)

Liczne pliki mają tekst polski bez znaków diakrytycznych ("Usun" zamiast "Usuń", "Brak magazynow" zamiast "Brak magazynów", itp.). Dotyczy głównie:
- customers/ (cały moduł)
- settings/warehouses, warehouse-documents, price-lists
- settings/sms, currencies, marketing, helpdesk
- suppliers/[id]
- products/ (częściowo)
- packing, reports
- return-request/ (formularz publiczny)
- connection-status component
- breadcrumbs component

Strony z poprawnymi ogonkami (wzorcowe): integrations/, audit/, shipments/, returns/ (moduł główny)

---

## Co działa dobrze

1. **Zamówienia CRUD** — tworzenie, edycja, usuwanie, zmiana statusu (cały lifecycle)
2. **Walidacja formularzy** — Zod + backend walidacja z polskimi komunikatami
3. **Eksport CSV** — działa z BOM, polskimi nagłówkami
4. **Produkty CRUD** — tworzenie, warianty, bundles, kategorie
5. **Przesyłki** — pełny cykl: created → label_ready → picked_up → in_transit → delivered
6. **Zwroty (autentykowane)** — pełny cykl: requested → approved → received → refunded
7. **Klienci** — CRUD, szukanie, historia zamówień
8. **Faktury** — tworzenie, podgląd, anulowanie
9. **Magazyny** — CRUD, stany, dokumenty PZ/WZ/MM z auto-aktualizacją stanów
10. **Cenniki B2B** — pełny CRUD z pozycjami
11. **Automatyzacja** — tworzenie reguł, logi
12. **Role RBAC** — macierz uprawnień, tworzenie ról
13. **Audit log** — 410+ wpisów, filtrowanie, pełna historia
14. **InPost** — wyszukiwanie paczkomatów (prawdziwe dane)
15. **Webhooks** — konfiguracja, historia dostarczeń (38 pozycji)
16. **XSS ochrona na Create** — stripowanie HTML tagów
17. **RLS tenant isolation** — dane widoczne tylko w ramach tenanta
18. **Loading/Error/Empty states** — obecne na większości stron
