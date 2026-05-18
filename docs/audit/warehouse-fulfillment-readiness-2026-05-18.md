# OPE-252: Readiness magazynu i istniejących flow fulfillment

Data: 2026-05-18
Zakres: istniejące ekrany dashboardu, route readiness, statyczny przegląd hooków UI i tras API.
Tryb: konserwatywny audit przed ekspozycją klientom; bez wykonywania operacji magazynowych na live danych.

## Granica scope

Ten dokument dotyczy wyłącznie istniejących modułów magazynu, pakowania, pick-pack, stocktake i stock sync.

Nie rozpoczyna prac z OPE-403 ani żadnego child issue pod gated epikiem Provider Integration Studio & Fulfillment Orchestration. Nie dodaje nowego modelu orkiestracji, nowych tabel, nowych workerów ani nowych kontraktów providerów.

## Decyzja

Domyślna powierzchnia `client-ready` pozostaje wąska:

- `shipments` i `carriers` zostają widoczne, bo aktualny produktowy kierunek zakłada pracę z przesyłkami i setupem kuriera.
- Magazyny, dokumenty magazynowe, inwentaryzacja, pakowanie, Pick & Pack, stock sync i kontrola magazynowa pozostają ukryte do czasu testów end-to-end.

Powód: te moduły mogą zmieniać stany magazynowe, statusy zamówień i proces operacyjny klienta. Bez sprawdzonego recovery path łatwo doprowadzić do złych stanów magazynowych albo niespójnego procesu pakowania.

## Aktualna macierz

| Obszar | Trasa | Obecny status | Client-ready | Decyzja | Dowód wymagany przed odblokowaniem |
|---|---|---:|---:|---|---|
| Przesyłki | `/shipments`, `/shipments/new`, `/shipments/[id]` | `ready` | Tak | Widoczne | Smoke utworzenia przesyłki z gotowym providerem, label download, tracking/error state. |
| Kurierzy | `/carriers`, `/carriers/new`, `/carriers/[id]` | `ready` | Tak | Widoczne | Provider picker musi filtrować beta/blocked; InPost jako certyfikowany provider, reszta controlled/beta według readiness. |
| Pakowanie | `/packing` | `verify` | Nie | Ukryte | Skanowanie SKU/EAN, obsługa braków, status zamówienia, błędy skanera i puste stany. |
| Pick & Pack | `/pick-pack`, `/pick-pack/[id]` | `verify` | Nie | Ukryte | Utworzenie sesji, pick, move-to-packing, pack item, complete/cancel, uprawnienia i recovery. |
| Magazyny | `/settings/warehouses`, `/settings/warehouses/[id]` | `controlled` | Nie | Ukryte domyślnie | CRUD, default warehouse, adres nadawcy dla etykiet, stock per warehouse, owner/admin permissions. |
| Dokumenty magazynowe | `/settings/warehouse-documents`, `/settings/warehouse-documents/new`, `/settings/warehouse-documents/[id]` | `verify` | Nie | Ukryte | PZ/WZ/MM create, confirm, cancel, wpływ na stock, walidacja target warehouse dla MM. |
| Inwentaryzacja | `/stocktakes`, `/stocktakes/new`, `/stocktakes/[id]` | `verify` | Nie | Ukryte | Draft/start/count/complete/cancel/delete, różnice stocku, brak podwójnego complete, obsługa pustego magazynu. |
| Stock sync | `/stock-sync`, `/stock-sync/events` | `beta` | Nie | Ukryte | Kanały sync, dry-run/push, idempotencja, błędy providerów, brak masowych zmian bez potwierdzenia. |
| Kontrola magazynowa | `/settings/inventory` | `verify` | Nie | Ukryte | Strict mode, min stock, reservations, manual stock update policy i komunikacja skutków dla klienta. |

## Sprawdzone zabezpieczenia ekspozycji

- Nawigacja dashboardu filtruje logistics/settings route przez readiness registry.
- Settings index korzysta z `getVisibleNavItems`, więc nie pokazuje ukrytych ustawień magazynowych w `client-ready`.
- Command palette korzysta z readiness-filtered nav i filtruje quick actions przez `isRouteAccessible`.
- Direct route access jest zabezpieczony przez `ReadinessRouteGuard`.
- Regresje w `apps/dashboard/src/lib/__tests__/readiness.test.ts` obejmują widoczność `/shipments` i `/carriers` oraz blokadę warehouse/pick-pack/stock route w `client-ready`.

## Statyczne obserwacje techniczne

- Hooki magazynowe (`use-warehouses`, `use-warehouse-documents`, `use-stocktakes`, `use-pick-pack`) korzystają z centralnego `apiClient` albo `apiFetch`.
- Router API grupuje warehouse, warehouse documents i stocktakes pod uprawnieniami magazynowymi.
- Pick-pack ma osobne endpointy sesji i statusów; UI jest ukryty w `client-ready`, ale API `/v1/pick-pack/sessions` jest obecnie dostępne dla każdego zalogowanego użytkownika. Przed ekspozycją klientom wymagane jest OPE-436 z jawnym modelem uprawnień.
- Stock sync jest obszarem wysokiego ryzyka operacyjnego, bo propaguje stany do kanałów sprzedaży; bez dry-run i jasnych alertów powinien zostać `beta`.

## Follow-upy

- Jeżeli pierwszy klient potrzebuje magazynu, zacząć od kontrolowanej walidacji `/settings/warehouses`, bo default warehouse wpływa także na adres nadawcy w etykietach.
- OPE-436: dodać jawne uprawnienia do API Pick & Pack przed odblokowaniem `/pick-pack` dla klientów.
- Pick-pack i stocktakes powinny dostać osobne testy browser smoke dopiero po decyzji, że pierwszy klient realnie ich używa.
- Stock sync wymaga osobnego hardeningu operacyjnego przed ekspozycją: dry-run, batch limits, czytelny audit log i recovery.

## Walidacja lokalna

Dodano regresje w `apps/dashboard/src/lib/__tests__/readiness.test.ts`, które sprawdzają:

- `shipments` i `carriers` pozostają dostępne w `client-ready`,
- packing, pick-pack, warehouses, stocktakes, warehouse documents, stock sync i inventory są blokowane w `client-ready`,
- te same nieblokowane route są dostępne w `full` validation mode,
- klasyfikacja route używa oczekiwanych statusów `ready`, `controlled`, `verify` i `beta`.
