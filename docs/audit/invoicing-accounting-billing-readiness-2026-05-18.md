# OPE-253: Readiness fakturowania, księgowości i billing

Data: 2026-05-18
Zakres: dashboard public repo, route readiness, provider readiness, statyczny przegląd API/UI.
Tryb: konserwatywny audit przed ekspozycją klientom; bez użycia live credentiali Fakturowni, wFirma, inFakt, KSeF ani Stripe.

## Decyzja

Moduły prawno-księgowe i SaaS billing pozostają poza domyślną powierzchnią `client-ready`.

Powód jest prosty: te ekrany dotykają faktur, danych firmowych, rozliczeń podatkowych, płatności i zaufania klienta. Sam fakt, że kod istnieje, nie jest wystarczający. Do ekspozycji potrzeba dowodu end-to-end na realnym albo bezpiecznym testowym koncie providera, czytelnych błędów, manualnego recovery path oraz przetestowanej obsługi uprawnień.

## Aktualna macierz

| Obszar | Trasa / provider | Obecny status | Client-ready | Decyzja | Dowód wymagany przed odblokowaniem |
|---|---|---:|---:|---|---|
| Faktury | `/invoices`, `/invoices/[id]` | `controlled` | Nie | Ukryte domyślnie | Utworzenie faktury z zamówienia, PDF, anulowanie/korekta, puste stany, błędy providera, uprawnienia owner/admin/member. |
| Fakturowanie | `/invoicing` | `controlled` | Nie | Ukryte domyślnie | Lista integracji, dodanie/edycja providera, test credentiali, brak martwych CTA i bezpieczne błędy. |
| Fakturownia | provider `fakturownia` | `controlled` | Nie | Wewnętrzna walidacja po credentialach | Subdomena + API token; create invoice, get invoice, PDF, cancel/test correction, czytelny błąd złych credentiali. |
| wFirma | provider `wfirma` | `beta` | Nie | Ukryte | Konto/API key, create/get/PDF, jawnie opisany brak cancel albo poprawka cancel flow. |
| inFakt | provider `infakt` | `beta` | Nie | Ukryte | Konto/API key, create/get/PDF, jawnie opisany brak cancel albo poprawka cancel flow. |
| Księgowość | `/settings/accounting` | `controlled` | Nie | Ukryte domyślnie | Spójność z `/invoicing`, konfiguracja providera, walidacja formularzy i brak możliwości zapisania pozornie gotowej integracji. |
| KSeF | `/settings/ksef` | `blocked` | Nie | Zablokowane w każdym trybie | Testowe konto/certyfikat/token, wysyłka, status, UPO, błędy autoryzacji, decyzja prawna i operacyjna. |
| VAT OSS settings | `/settings/vat-oss` | `beta` | Nie | Ukryte | Kraje UE, waluty, źródło danych faktur, poprawność eksportu i review księgowe. |
| VAT OSS report | `/reports/vat-oss` | `beta` | Nie | Ukryte | Spójne dane z fakturami i kursami, eksport, puste/error states, weryfikacja z księgowością. |
| Billing SaaS | `/settings/billing` | `controlled` | Nie | Ukryte dla pierwszego klienta do decyzji OPE-246 | Stripe checkout/subscription/webhook E2E, statusy active/trial/past_due/canceled/manual contract, czytelny UX dla klienta. |

## Zasady odblokowania

- Nie przenosimy żadnego z tych obszarów na `ready` bez browser smoke evidence.
- Provider fakturowania wymaga testu na kontrolowanym koncie, nie tylko mocka albo statycznego przeglądu.
- KSeF nie może być traktowany jako zwykły toggle UI; wymaga osobnej walidacji prawno-operacyjnej.
- Billing klienta jest decyzją produktową i operacyjną, nie tylko ekranem ustawień. Zakres ekspozycji powinien zostać domknięty w OPE-246.
- `full` dashboard mode może służyć do walidacji operatora, ale nie jest powierzchnią dla pierwszego klienta.

## Sprawdzone zabezpieczenia ekspozycji

- Nawigacja dashboardu filtruje elementy przez readiness registry.
- Command palette korzysta z readiness-filtered nav i nie powinna odkrywać ukrytych route.
- Direct route access jest zabezpieczony przez `ReadinessRouteGuard`.
- Provider readiness ukrywa Fakturownię, wFirma i inFakt w `client-ready`.
- Ekran `/invoicing` ma własny wybór Fakturownia/wFirma/inFakt, ale sama trasa jest `controlled`, więc w `client-ready` nie jest dostępna ani z menu, ani przez direct route.
- `/settings/ksef` pozostaje `blocked`, więc nie otwiera się nawet w `full`.

## Follow-upy

- OPE-246: decyzja, jak klient ma widzieć billing/subskrypcję w SaaS.
- Potrzebne przed odblokowaniem Fakturowni: kontrolowane credentiale i smoke flow create/PDF/cancel.
- Potrzebne przed wFirma/inFakt: osobna certyfikacja providera i decyzja, co robimy z brakiem cancel.
- Potrzebne przed KSeF/VAT OSS: test env, review księgowe i jasny runbook recovery.

## Walidacja lokalna

Dodano regresje w `apps/dashboard/src/lib/__tests__/readiness.test.ts`, które sprawdzają:

- brak faktur, fakturowania, księgowości, VAT OSS i billing w client-ready nav,
- blokadę direct route access w client-ready,
- dostępność kontrolowanych route w full mode tylko do walidacji,
- trwałą blokadę KSeF,
- brak providerów fakturowania w client-ready.
