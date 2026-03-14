# Prompt: Przygotowanie kodu OpenOMS pod ekspansję międzynarodową

## Kontekst

OpenOMS to open-source'owy system zarządzania zamówieniami (OMS) dla polskiego e-commerce. Obecnie działa wyłącznie po polsku — hardcoded stringi w UI, domyślna waluta PLN, polskie integracje (Allegro, InPost, KSeF). Celem tej sesji jest przygotowanie architektury kodu tak, aby w przyszłości (6-18 miesięcy) możliwe było:

1. Uruchomienie dashboardu w języku angielskim (i kolejnych)
2. Obsługa tenantów z innych krajów (Czechy, Rumunia, Słowacja, Niemcy)
3. Pluggowalne integracje lokalne (marketplace, carrier, invoicing) per kraj
4. Multi-currency z prawidłowym formatowaniem i przelicznikami

**WAŻNE**: To NIE jest sesja na pełną implementację i18n. To jest sesja na przygotowanie fundamentów architektonicznych — interfejsy, abstrakcje, struktura plików — żeby przyszła praca translacyjna była mechaniczna, a nie architektoniczna.

## Zasady

- Kod i konfiguracja w języku angielskim, komentarze techniczne po angielsku
- Żadnych danych biznesowych (ceny, plany, klucze Stripe) w publicznym repo
- Żadnych atrybutów AI (Co-Authored-By, "Generated with Claude Code") w commitach i PR-ach
- Zawsze branch + PR, nigdy push do main
- Minimalne zmiany — nie refaktoryzuj kodu który nie wymaga zmian
- Nie dodawaj features których nikt nie zamówił — tylko fundamenty architektoniczne

## Stan obecny (audyt)

### Frontend (`apps/dashboard/`)

| Aspekt | Status | Szczegóły |
|--------|--------|-----------|
| Biblioteka i18n | Brak | Żadna zainstalowana (brak next-intl, react-i18next) |
| Pliki tłumaczeń | Brak | Żadne .json/.yaml z locale |
| Stringi w komponentach | Hardcoded PL | ~240 plików TSX z polskimi stringami |
| Stałe statusów | Scentralizowane | `src/lib/constants.ts` (464 linii) — 20+ map z polskimi labelami |
| Toast / powiadomienia | Rozproszone | `toast.error("Polski tekst")` w wielu komponentach |
| Walidacja formularzy | Rozproszona | Polskie komunikaty bezpośrednio w komponentach |
| HTML lang | Hardcoded | `lang="pl"` w `src/app/layout.tsx:19` |
| Routing | Brak i18n | Brak segmentu `[locale]`, brak middleware locale |

### Backend (`apps/api-server/`)

| Aspekt | Status | Szczegóły |
|--------|--------|-----------|
| Błędy API | EN, generyczne | `writeError(w, 400, "invalid order ID")` — brak kodów błędów |
| Locale w modelu | Brak | User i Tenant nie mają pola language/locale |
| Waluta domyślna | PLN hardcoded | Order defaults "PLN", Price Lists default "PLN" |
| VAT | EU-ready | `vat_oss_service.go` ma stawki dla 27 krajów EU |
| Provider interfaces | Neutralne językowo | MarketplaceProvider, CarrierProvider — brak parametru language |
| Provider factory | Registry pattern | `factory.go` — `RegisterMarketplaceProvider("allegro", ...)` |
| Kraj domyślny | PL hardcoded | `vat_oss_handler.go:109`: `homeCountry = "PL"` |
| NIP w CompanySettings | Polska-specific | Pole `NIP` zamiast generycznego `tax_id` |

### SDK packages (`packages/`)

| Kategoria | Pakiety | Specyficzność |
|-----------|---------|---------------|
| Marketplace (10) | allegro, amazon, ebay, shopify, woocommerce... | Generyczne (poza Allegro) |
| Carrier (7) | inpost, dhl, dpd, gls, ups, fedex, poczta-polska | InPost, Poczta Polska, Orlen = PL-only |
| Invoicing (4) | ksef, fakturownia, wfirma, infakt | Wszystkie PL-only |
| Supplier (2) | btp, iof-parser | PL-only |

### Hardcoded "PL" / "PLN" w kodzie

| Plik | Linia | Kontekst |
|------|-------|----------|
| `handler/vat_oss_handler.go` | 109 | `homeCountry = "PL"` |
| `handler/allegro_account_handler.go` | 41, 80 | `body.Currency = "PLN"` |
| `model/feed.go` | - | `DefaultCurrency: "PLN"` |
| `integration/carriers/inpost.go` | 162 | Domestic check `== "PL"` |
| `integration/carriers/fedex.go` | - | Domestic check `== "PL"` |
| `integration/carriers/ups.go` | - | Domestic check `== "PL"` |
| `integration/carriers/orlen_paczka.go` | - | Domestic check `== "PL"` |
| `service/vat_oss_service.go` | 73 | `"PL": "Polska"` (country name) |
| `app/layout.tsx` | 19 | `lang="pl"` |
| `lib/constants.ts` | 1-464 | 20+ map ze statusami po polsku |

## Zadania (w kolejności priorytetów)

### FAZA 1: Fundamenty backend (bez zmian w behavior)

#### 1.1 Dodaj `locale` i `country` do modelu Tenant

**Plik**: `apps/api-server/internal/model/user.go`

Dodaj do `Tenant` struct:
```go
Locale  string `json:"locale"`   // "pl", "en", "cs", "ro" — UI language
Country string `json:"country"`  // "PL", "CZ", "RO", "SK" — business country (tax, carriers)
```

Domyślne wartości: `locale = "pl"`, `country = "PL"`. Istniejące tenanty nie powinny się zepsuć (COALESCE w SQL).

**Migracja**: Nowy plik w `apps/api-server/migrations/` — `ALTER TABLE tenants ADD COLUMN locale VARCHAR(10) DEFAULT 'pl', ADD COLUMN country VARCHAR(5) DEFAULT 'PL'`.

#### 1.2 Dodaj `preferred_locale` do modelu User

**Plik**: `apps/api-server/internal/model/user.go`

```go
PreferredLocale string `json:"preferred_locale,omitempty"` // overrides tenant locale
```

User locale > Tenant locale > "pl" (fallback chain).

#### 1.3 Zamień `NIP` na `TaxID` w CompanySettings

**Plik**: `apps/api-server/internal/model/user.go`

```go
type CompanySettings struct {
    CompanyName string `json:"company_name"`
    TaxID       string `json:"tax_id"`       // was: NIP — now generic: NIP (PL), IČO (CZ), CUI (RO)
    TaxIDType   string `json:"tax_id_type"`  // "nip", "ico", "cui", "ust_id"
    // ... rest unchanged
}
```

**UWAGA**: Zachowaj backward compatibility — jeśli w bazie istnieje `nip`, zmapuj na `tax_id`. Nie kasuj danych.

#### 1.4 Wyciągnij domyślny kraj/walutę z hardcodu

Zamiast `homeCountry = "PL"` w handlerach, czytaj z tenant settings:
```go
tenant := middleware.TenantFromContext(r.Context())
homeCountry := tenant.Country // defaults "PL" from DB
```

Analogicznie dla domyślnej waluty — zamiast `"PLN"`:
```go
func DefaultCurrencyForCountry(country string) string {
    switch country {
    case "PL": return "PLN"
    case "CZ": return "CZK"
    case "RO": return "RON"
    case "SK": return "EUR"
    case "DE": return "EUR"
    default:   return "EUR"
    }
}
```

Umieść w `internal/i18n/defaults.go` lub `internal/locale/defaults.go`.

#### 1.5 Utwórz system kodów błędów API

**Nowy plik**: `apps/api-server/internal/apierror/codes.go`

Zamiast `writeError(w, 400, "invalid order ID")`:
```go
// Error codes are stable identifiers — frontend translates them to user language
type ErrorCode string

const (
    ErrCodeInvalidInput    ErrorCode = "INVALID_INPUT"
    ErrCodeNotFound        ErrorCode = "NOT_FOUND"
    ErrCodeUnauthorized    ErrorCode = "UNAUTHORIZED"
    ErrCodeForbidden       ErrorCode = "FORBIDDEN"
    ErrCodeConflict        ErrorCode = "CONFLICT"
    ErrCodeLimitExceeded   ErrorCode = "LIMIT_EXCEEDED"
    ErrCodeValidation      ErrorCode = "VALIDATION_ERROR"
    // ...
)

func WriteError(w http.ResponseWriter, status int, code ErrorCode, message string) {
    // JSON response: {"error": {"code": "INVALID_INPUT", "message": "invalid order ID"}}
}
```

**NIE zmieniaj** istniejących handlerów w tej sesji — to jest fundament. Migracja handlerów to osobne zadanie (mechaniczne).

### FAZA 2: Fundamenty frontend

#### 2.1 Zainstaluj `next-intl`

```bash
cd apps/dashboard && npm install next-intl
```

#### 2.2 Utwórz strukturę plików tłumaczeń

```
apps/dashboard/
├── messages/
│   ├── pl.json          # Polish (primary, complete)
│   └── en.json          # English (start empty, fill incrementally)
├── src/
│   ├── i18n/
│   │   ├── config.ts    # supported locales, default locale
│   │   ├── request.ts   # next-intl getRequestConfig
│   │   └── navigation.ts # createNavigation with localePrefix
│   ├── middleware.ts     # locale detection middleware
│   └── app/
│       └── [locale]/    # locale segment in routing
│           └── layout.tsx
```

#### 2.3 Wyekstrahuj `constants.ts` do pliku tłumaczeń

Zamień:
```typescript
// constants.ts
export const ORDER_STATUSES = {
  new: { label: "Nowe", color: "blue" },
  // ...
}
```

Na:
```typescript
// constants.ts — colors only (language-neutral)
export const ORDER_STATUS_COLORS = {
  new: "blue",
  confirmed: "green",
  // ...
}
```

```json
// messages/pl.json
{
  "orderStatus": {
    "new": "Nowe",
    "confirmed": "Potwierdzone",
    "processing": "W realizacji"
  }
}
```

#### 2.4 Zmień `lang="pl"` na dynamiczny

**Plik**: `src/app/layout.tsx`
```tsx
// Before: <html lang="pl">
// After:  <html lang={locale}>
```

#### 2.5 NIE tłumacz jeszcze komponentów

Wyekstrahuj TYLKO:
1. `constants.ts` statusy/labele → `messages/pl.json`
2. Layout root → dynamiczny `lang`
3. Middleware → locale detection

Translacja 240 plików TSX to osobna sesja (mechaniczna praca, nie architektoniczna).

### FAZA 3: Pluggowalna architektura provider

#### 3.1 Dodaj `InvoicingProvider` registry per country

**Obecny stan**: KSeF, Fakturownia, wFirma, inFakt — wszystkie polskie, hardcoded.

**Cel**: Tenant z `country = "CZ"` automatycznie widzi czeskie opcje fakturowania (np. ISDOC), nie polskie.

```go
// internal/integration/invoicing_registry.go
var countryInvoicingProviders = map[string][]string{
    "PL": {"ksef", "fakturownia", "wfirma", "infakt"},
    "CZ": {}, // future: isdoc
    "RO": {}, // future: e-factura
}

func AvailableInvoicingProviders(country string) []string {
    return countryInvoicingProviders[country]
}
```

#### 3.2 Dodaj `country` filter do carrier registry

Nie wszyscy przewoźnicy działają we wszystkich krajach:

```go
// internal/integration/carrier_registry.go
var countryCarriers = map[string][]string{
    "PL": {"inpost", "dhl", "dpd", "gls", "ups", "fedex", "poczta-polska", "orlen-paczka"},
    "CZ": {"dhl", "dpd", "gls", "ups", "fedex"},  // future: zasilkovna, ceska-posta
    "DE": {"dhl", "dpd", "gls", "ups", "fedex"},   // future: hermes
}
```

#### 3.3 Marketplace nie wymaga zmian

Marketplace'y (Allegro, Amazon, eBay, Shopify...) są globalnie dostępne lub per-tenant configurable. Nie filtruj ich per country — tenant sam wybiera swoje integracje.

### FAZA 4: Multi-currency UX

#### 4.1 Formatowanie walut

**Nowy plik**: `apps/dashboard/src/lib/currency.ts`

```typescript
export function formatCurrency(amount: number, currency: string, locale: string): string {
  return new Intl.NumberFormat(locale, {
    style: "currency",
    currency,
    minimumFractionDigits: 2,
  }).format(amount / 100) // amounts stored in cents/grosze
}
```

Użyj `Intl.NumberFormat` zamiast hardcoded "zł" czy "PLN".

#### 4.2 Formatowanie dat

```typescript
export function formatDate(date: string | Date, locale: string): string {
  return new Intl.DateTimeFormat(locale, {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
  }).format(new Date(date))
}
```

Zastąp hardcoded `dd.MM.yyyy` (polski format) na locale-aware.

## Czego NIE robić w tej sesji

1. **Nie tłumacz 240 plików TSX** — to mechaniczna praca na osobną sesję
2. **Nie dodawaj nowych marketplace/carrier integracji** — tylko struktura registry
3. **Nie zmieniaj istniejących handlerów na ErrorCode** — tylko stwórz fundament
4. **Nie dodawaj nowych języków** — tylko `pl.json` + pusty `en.json` scaffold
5. **Nie modyfikuj SDK packages** — są wystarczająco generyczne
6. **Nie ruszaj Helm chart / Terraform / CI** — infrastruktura nie wymaga zmian na tym etapie
7. **Nie dodawaj feature flags** — za wcześnie, nie ma co flagować

## Quality gate (checklist na każde zadanie)

Przed commitem każdego zadania sprawdź:

- [ ] `go test ./...` przechodzi w `apps/api-server/`
- [ ] `go vet ./...` przechodzi
- [ ] Istniejące API nie zmieniło behavior (backward compatible)
- [ ] Nowe pola w modelu mają domyślne wartości (nie łamią istniejących tenantów)
- [ ] Migracja ma plik `.up.sql` i `.down.sql`
- [ ] Migracja jest backward-compatible (add column, nie drop/rename)
- [ ] Brak hardcoded "PL"/"PLN" w nowym kodzie (czytaj z tenant/config)
- [ ] Frontend: `npm run lint` przechodzi
- [ ] Frontend: `npm run build` przechodzi (no type errors)
- [ ] Brak polskich stringów w nowym kodzie (używaj translation keys)
- [ ] Brak danych biznesowych (ceny, plany, klucze) w publicznym repo

## Kolejność pracy

```
1.1 Tenant locale/country ──→ 1.2 User preferred_locale ──→ 1.3 TaxID refactor
                                                                      │
1.4 Default country/currency from tenant ◄────────────────────────────┘
         │
         ▼
1.5 Error code system (apierror package)
         │
         ▼
2.1 Install next-intl ──→ 2.2 File structure ──→ 2.3 Extract constants ──→ 2.4 Dynamic lang
         │
         ▼
3.1 Invoicing registry ──→ 3.2 Carrier registry
         │
         ▼
4.1 Currency formatting ──→ 4.2 Date formatting
```

Każdy krok to osobny commit na feature branch. PR po zakończeniu całej fazy (1-4).
