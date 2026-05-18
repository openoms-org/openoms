# OPE-247: Audyt gotowości providerów i funkcji

Data: 2026-05-11
Zakres: publiczny dashboard, warstwa integracji API, SDK
Tryb: audyt statyczny. Nie wykonywałem operacji na prawdziwych credentialach klientów, płatnych przesyłek, faktur ani zapisów do marketplace.

## Podsumowanie

Nie powinniśmy pokazywać pierwszym klientom wszystkiego, co istnieje w kodzie. Projekt ma już dużo adapterów, tras i stron dashboardu, ale "jest zaimplementowane" nie znaczy jeszcze "gotowe dla klienta". W kilku miejscach provider wygląda w UI na dostępny, mimo że ma niepełne akcje, placeholderowe tworzenie ofert, stawki tylko szacunkowe, brak automatycznego trackingu albo niespójne nazwy między UI i backendem.

Rekomendowana powierzchnia dla pierwszego klienta:

- Domyślnie gotowe: podstawowe moduły OMS, rdzeń Allegro, InPost.
- Kontrolowane / tylko po checkliście: DHL, DPD, GLS, BTP, Fakturownia, OLX oraz zaawansowane podmoduły Allegro.
- Ukryte domyślnie: Amazon, eBay, WooCommerce, Erli, Kaufland, Empik/Mirakl, Shoper, PrestaShop, Shopify, UPS, FedEx, Poczta Polska, Orlen Paczka, wFirma, inFakt, KSeF, SMS/marketing/helpdesk, listing sync, stock sync, automatyzacje/workflow builder i raporty zaawansowane.
- Wyłączone w UI i zabezpieczone w API: ścieżki tworzenia listingów, które zwracają sztuczne external ID albo jawny błąd "not implemented".

## Walidacja w produkcyjnym dashboardzie

Data sprawdzenia: 2026-05-11
Adres: `https://app.openoms.org/`
Tryb: read-only browser audit na zalogowanej sesji. Nie wykonywałem akcji tworzących, aktualizujących ani usuwających dane.

Obserwacje potwierdzają ryzyko z audytu statycznego:

- Główne menu pokazuje bardzo szeroką powierzchnię: faktury, fakturowanie, szablony druku, pakowanie, pick & pack, magazyny, inwentaryzację, dokumenty magazynowe, sync magazynu, marketplace, feedy, synchronizację ofert, raporty zaawansowane, forecast, carbon, VAT OSS, reconciliation, repricing, suppliers, purchase orders, dropshipping, automation, workflow builder, importy, background removal, waluty, subskrypcje, loyalty, segmenty, billing, accounting, SMS, webhooki, sync jobs i audit log.
- Ekran główny ma dobry, wąski onboarding: firma, Allegro, produkt, pierwsze zamówienie. Problemem jest kontrast z pełnym menu, które równocześnie sugeruje, że prawie cały system jest gotowy.
- `/marketplaces/new` pokazuje jako wybieralne: Allegro, Amazon, WooCommerce, Shopify, PrestaShop, Shoper, eBay, Kaufland, OLX, Erli i Empik (Mirakl). Część ma badge `Beta`, ale nadal wygląda jak dostępna akcja. Opisy renderują surowe klucze tłumaczeń, np. `providers.amazon.description`, `providers.empik.description`.
- `/integrations/new` pokazuje `allegro`, `amazon DEV` i `olx`; Amazon jest dostępny z poziomu UI mimo statusu beta.
- `/carriers` jest widoczne i ma empty state "Brak kurierow"; brakuje provider readiness i polskiego znaku w tekście.
- `/listing-sync`, `/settings/feeds`, `/reports/forecast`, `/reports/carbon`, `/repricing`, `/workflows` są widoczne i mają akcje typu "Nowa konfiguracja", "Zapisz ustawienia", "Konfiguracja", "Zastosuj teraz", mimo że nie są rekomendowane jako client-ready.
- `/suppliers` pokazuje heading `Tytuł`; to wygląda jak nieusunięty placeholder na produkcji.
- `/settings/sms` przekierowuje do `/settings/notifications`, ale w treści nadal pojawia się `Tytuł` i nieoczekiwany tekst `Hasło DHL`; to wygląda jak pomieszany albo roboczy ekran.

Wniosek UX: dashboard jest technicznie bogaty, ale produktowo zbyt głośny na pierwszych klientów. Największy problem nie jest wizualny, tylko informacyjny: użytkownik nie wie, które ścieżki są produkcyjne, które są beta, a które są tylko szkieletem.

## Poziomy gotowości

| Status | Znaczenie | Ekspozycja klientowi |
|---|---|---|
| `ready` | Zweryfikowane wystarczająco dla pierwszego klienta w danym flow. | Widoczne i używalne. |
| `controlled` | Technicznie użyteczne, ale wymaga per-klientowego włączenia, credentiali i checklisty. | Ukryte z domyślnej nawigacji albo dostępne tylko świadomie dla owner/admin. |
| `beta` | Zaimplementowane częściowo albo bez end-to-end certyfikacji. | Ukryte domyślnie w produkcji. |
| `disabled` | Znany brak, placeholder albo niespójność z backendem. | Nie do wyboru; backend powinien odrzucać bezpośrednie użycie. |

## Brak jednego źródła prawdy

Dziś readiness jest rozproszony.

- Nawigacja ma tylko `hidden?: boolean`, więc nie da się wyrazić `ready`, `controlled`, `beta` ani `disabled` (`apps/dashboard/src/lib/nav-items.ts:70-74`).
- Sidebar i command palette filtrują tylko `hidden`, nie filtrują modułów beta/internal (`apps/dashboard/src/components/layout/sidebar.tsx`, `apps/dashboard/src/components/layout/mobile-nav.tsx`, `apps/dashboard/src/components/shared/command-palette.tsx`).
- Listy providerów są niespójne:
  - `ORDER_SOURCES` pokazuje `amazon`, `empik`, `erli`, `ebay`, `kaufland`, `olx`, `woocommerce` (`apps/dashboard/src/lib/constants.ts:101`).
  - `SHIPMENT_PROVIDERS` pokazuje wszystkich kurierów, także beta (`apps/dashboard/src/lib/constants.ts:119`).
  - `INTEGRATION_PROVIDERS` zawiera `empik`, ale backend rejestruje `mirakl`, nie `empik` (`apps/dashboard/src/lib/constants.ts:120`, `apps/api-server/internal/integration/mirakl/provider.go:18-21`).
  - `PROVIDER_CATEGORIES.marketplace` pomija `kaufland`, `empik`, `shoper`, `prestashop`, `shopify`, a `provider-info.ts` pokazuje je w marketplace pickerze (`apps/dashboard/src/lib/constants.ts:296-300`, `apps/dashboard/src/lib/provider-info.ts:21-31`).
  - `INVOICING_PROVIDERS` zawiera `wfirma` i `infakt`, ale generic integration form wystawia tylko `fakturownia` w kategorii invoicing (`apps/dashboard/src/lib/constants.ts:186`, `apps/dashboard/src/lib/constants.ts:299`).
- Status "w trakcie rozwoju" tylko dopisuje etykietę i nie blokuje wyboru (`apps/dashboard/src/components/integrations/integration-form.tsx:324-329`).

## Macierz providerów

### Marketplace

| Provider | Dowód | Rekomendowany status | Akcja dla klienta |
|---|---|---|---|
| Allegro | Istnieją trasy OAuth, polling, oferty, fulfillment, przesyłki, wiadomości, zwroty, katalog, polityki, promocje, spory i oceny (`apps/api-server/internal/router/router.go:570-691`). | `ready` dla rdzenia, `controlled` dla zaawansowanych podmodułów | Pokazać core: setup, zamówienia, podstawowe oferty. Zaawansowane zakładki po checkliście. |
| OLX | OAuth i listing flow istnieją, ale stock update nie ma sensu dla ogłoszeń (`apps/api-server/internal/integration/olx/provider.go:248-250`). Ostatni problem `invalid_grant` został naprawiony operacyjnie. | `controlled` | Pokazać tylko jeśli klient realnie potrzebuje OLX; bez obietnicy stock sync. |
| Amazon | Zamówienia i asynchroniczne feedy stock/price istnieją, ale `PushOffer` jest jawnie niezaimplementowany, a feedy wymagają `selling_partner_id` (`apps/api-server/internal/integration/amazon/provider.go:190-200`). | `beta` | Ukryć w client-ready mode. |
| eBay | Jest OAuth/listing/import/stock/price, ale provider jest oznaczony beta i ten audyt nie ma dowodu produkcyjnego E2E. | `beta` | Ukryć domyślnie. |
| WooCommerce | Kod CRUD/listing istnieje, ale brak dowodu produkcyjnej gotowości; generic form może go wystawić. | `beta` | Ukryć do czasu E2E na demo sklepie. |
| Erli | Provider ma realne ścieżki ofert i zamówień, ale sandbox wymaga jawnego base URL i wcześniejszy ADR notuje follow-upy. | `beta` | Ukryć; certyfikować przez sandbox/live test. |
| Kaufland | Backend rejestruje provider, ale `PushOffer` jest niezaimplementowany (`apps/api-server/internal/integration/kaufland/provider.go:127-129`). | `disabled` dla listingów, `beta` dla order/stock | Nie pokazywać w pickerze klienta. |
| Empik/Mirakl | UI pokazuje `empik`, backend factory jest `mirakl`; `empik` skończyłby jako unknown provider (`apps/api-server/internal/integration/factory.go:24-34`, `apps/api-server/internal/integration/mirakl/provider.go:18-21`). | `disabled` | Najpierw naprawić alias/nazewnictwo. |
| Shoper | Setup weryfikuje credentiale, ale `PushOffer` zwraca sztuczne `shoper-{productID}` (`apps/api-server/internal/integration/shoper/provider.go:121-131`). | `disabled` dla listingów, `beta` dla order sync | Ukryć stronę z domyślnego marketplace pickera. |
| PrestaShop | Setup weryfikuje credentiale, ale `PushOffer` zwraca sztuczne `prestashop-{productID}` i część parse błędów money jest ignorowana (`apps/api-server/internal/integration/prestashop/provider.go:118-128`, `apps/api-server/internal/integration/prestashop/provider.go:183`). | `disabled` dla listingów, `beta` dla order sync | Ukryć stronę z domyślnego marketplace pickera. |
| Shopify | Setup weryfikuje credentiale, ale `PushOffer` zwraca sztuczne `shopify-{productID}` (`apps/api-server/internal/integration/shopify/provider.go:128-138`). | `disabled` dla listingów, `beta` dla order sync | Ukryć stronę z domyślnego marketplace pickera. |

### Kurierzy

| Provider | Dowód | Rekomendowany status | Akcja dla klienta |
|---|---|---|---|
| InPost | Create shipment, label, tracking, cancel, dispatch order, punkty odbioru, webhook i Geowidget są podłączone. Rate shopping jest ukryty, dopóki nie ma kontraktowego źródła wyceny. | `ready` dla labels/tracking/points, `blocked` dla rates | Pokazać labels/tracking/points, jeśli credentiale są poprawne; nie pokazywać rate-shopping jako wyceny InPost. |
| DHL | SOAP integration istnieje i ma production-style testy; rates są szacunkowe, brak pickup points (`apps/api-server/internal/integration/carriers/dhl.go:224-328`). | `controlled` | Włączyć po checkliście create/label/tracking na credentialach klienta. |
| DPD | Create/label istnieje, ale tracking zwraca "not available via DPD REST API"; rates są szacunkowe (`apps/api-server/internal/integration/carriers/dpd.go:150-168`). | `controlled` z caveatem trackingu | Włączyć etykiety; nie obiecywać automatycznego trackingu DPD. |
| GLS | Create/label ze storage/tracking/cancel istnieją; rates są szacunkowe, pickup search niezaimplementowany (`apps/api-server/internal/integration/carriers/gls.go:224-267`). | `controlled` | Włączyć po checkliście label storage + tracking. |
| UPS | Adapter istnieje; rates są TODO/estimate i brak dowodu produkcyjnego w tym audycie. | `beta` | Ukryć domyślnie. |
| FedEx | Adapter istnieje; rates są TODO/estimate i brak dowodu produkcyjnego w tym audycie. | `beta` | Ukryć domyślnie. |
| Poczta Polska | Adapter istnieje; rates są TODO/estimate, pickup search unsupported. | `beta` | Ukryć domyślnie. |
| Orlen Paczka | Adapter istnieje i ma pickup search; rates są TODO/estimate. | `beta` | Ukryć do certyfikacji. |

### Dostawcy i dropship

| Provider / funkcja | Dowód | Rekomendowany status | Akcja dla klienta |
|---|---|---|---|
| BTP.pro | Supplier provider ma product, inventory i order methods; limity XML/parserów doszły w OPE-218. | `controlled` | Ukryć z domyślnego menu, chyba że pierwszy klient używa BTP; wtedy włączyć po checkliście feed/order. |
| Generic suppliers / purchase orders / dropship | Duże moduły widoczne w nav; OPE-241 już znalazło placeholder copy i niejasną wartość dla pierwszego klienta. | `beta` | Ukryć domyślnie do certyfikacji jednego flow dostawcy. |

### Fakturowanie i księgowość

| Provider / funkcja | Dowód | Rekomendowany status | Akcja dla klienta |
|---|---|---|---|
| Fakturownia | Provider implementuje create/get/pdf/cancel. | `controlled` | Ukryć, dopóki nie przejdzie test invoice flow. |
| wFirma | Create/get/pdf istnieją; cancel jest jawnie unsupported. | `beta` | Ukryć domyślnie. |
| inFakt | Create/get/pdf istnieją; cancel jest jawnie unsupported. | `beta` | Ukryć domyślnie. |
| KSeF | Już ukryte w nav. | `beta` | Zostawić ukryte. |

### Moduły przekrojowe

| Moduł | Obecna ekspozycja | Rekomendacja |
|---|---|---|
| Zamówienia, produkty, klienci, zwroty, firma, użytkownicy, role, bezpieczeństwo, pomoc | Core surface z OPE-241. | `ready` po polishu UI. |
| Faktury/fakturowanie | Widoczne dziś (`apps/dashboard/src/lib/nav-items.ts:116-117`). | Ukryć, chyba że Fakturownia przejdzie certyfikację. |
| Przesyłki/kurierzy | Widoczne dziś (`apps/dashboard/src/lib/nav-items.ts:125-126`). | Pokazać przesyłki; setup kurierów tylko przez ready/controlled providerów. |
| Marketplace | Widoczne dziś (`apps/dashboard/src/lib/nav-items.ts:135`). | Pokazać tylko ready/controlled providerów. |
| Forecast, carbon, VAT OSS, reconciliation, repricing | Widoczne dziś (`apps/dashboard/src/lib/nav-items.ts:140-145`). | Ukryć domyślnie. |
| Dostawcy, purchase orders, dropship | Widoczne dziś (`apps/dashboard/src/lib/nav-items.ts:148-150`). | Ukryć domyślnie. |
| Automation/workflows/importy/bg removal/waluty/subskrypcje/loyalty/segmenty | Widoczne dziś (`apps/dashboard/src/lib/nav-items.ts:153-163`). | Ukryć domyślnie, poza importem jeśli będzie przetestowany dla pierwszego klienta. |
| Billing, zaawansowane settings, SMS, webhooks, sync jobs, audit log | Widoczne dziś (`apps/dashboard/src/lib/nav-items.ts:166-183`). | Ukryć domyślnie poza firma/users/roles/security. |

## Najwyższe ryzyka

1. `empik` jest widoczny w dashboard constants, ale backend rejestruje `mirakl`; to może tworzyć integracje, które później failują przy factory resolution.
2. `Shoper`, `PrestaShop` i `Shopify` mogą stworzyć w bazie pozornie aktywny listing bez realnego utworzenia oferty w marketplace.
3. `Amazon`, `Kaufland` i `Mirakl` jawnie nie obsługują direct `PushOffer`, więc product listing flow musi mieć guard.
4. Rate shopping u kurierów często zwraca twardo zaszyte szacunki. To trzeba ukryć albo jasno oznaczyć jako estimate-only.
5. DPD jest oznaczone jako verified w frontend maturity metadata, ale automatyczny tracking nie jest dostępny przez DPD REST API.
6. Readiness jest rozproszony między `constants.ts`, `provider-info.ts`, `integration-status.ts`, nav items, backend factory names i router.

## Rekomendowana implementacja dla OPE-247

1. Dodać jedno źródło prawdy, np. `apps/dashboard/src/lib/readiness.ts`, z:
   - readiness modułów,
   - readiness providerów per kategoria,
   - capability flags: `orders`, `listingCreate`, `stockSync`, `priceSync`, `labels`, `tracking`, `rates`, `pickupPoints`, `invoicing`.
2. Zastąpić rozproszone listy providerów w widocznym UI filtrowanymi selectorami:
   - integration form,
   - marketplace picker,
   - shipment provider selector,
   - order source filters/forms,
   - product listing provider choices,
   - rate shopping provider choices.
3. Domyślny client-ready production mode powinien pokazywać tylko `ready`. `controlled` wymaga jawnego włączenia wewnętrznego albo feature flagi.
4. Dodać backend guards dla disabled provider/actions, szczególnie listing creation:
   - odrzucać `shoper`, `prestashop`, `shopify`, dopóki nie mają realnego `PushOffer`,
   - odrzucać `amazon`, `kaufland`, `mirakl/empik`, gdzie provider zwraca not implemented,
   - zwracać bezpieczne, czytelne błędy.
5. Dodać testy:
   - unit testy registry/filteringu readiness,
   - component testy provider selectorów,
   - regresję, że sidebar i command palette nie odkrywają hidden/beta modułów w client-ready mode,
   - backend tests dla disabled listing routes.
6. Przed przejściem providera na `ready` wymagać checklisty:
   - walidacja credentiali,
   - create/read/update/cancel albo udokumentowane unsupported operations,
   - label/PDF download, jeśli dotyczy,
   - webhook/tracking, jeśli obiecujemy,
   - idempotency i duplicate behavior,
   - sandbox albo niskoryzykowna operacja live,
   - notatka operatora: rollback/manual recovery.

## Proponowany pierwszy allowlist

Celowo konserwatywny:

- Marketplace: domyślnie `allegro`; opcjonalnie `olx` jako controlled.
- Kurierzy: domyślnie `inpost`; opcjonalnie `dhl`, `dpd`, `gls` jako controlled po sprawdzeniu credentiali.
- Dostawcy: domyślnie nic; opcjonalnie `btp`, jeśli pierwszy klient go używa.
- Fakturowanie: domyślnie nic; opcjonalnie `fakturownia` po testowym invoice flow.
- Źródła zamówień w UI: `manual`, `allegro`; opcjonalnie `olx`, jeśli włączony.
- Providerzy przesyłek w UI: `inpost`; opcjonalnie `dhl`, `dpd`, `gls`, jeśli włączone.

## Pytania otwarte

- Czy `controlled` włączamy przez env var, tenant setting, czy wewnętrzny owner-only toggle?
- Które integracje realnie ma pierwszy klient: Allegro, InPost, BTP, Fakturownia, DHL/DPD/GLS?
- Czy backend guard ma obowiązywać tylko w SaaS production, czy też w domyślnym open-source self-hosted?
- Czy bezpośrednie URL-e do beta stron mają pokazywać "niedostępne dla tego tenanta", czy redirect/404-like state?
