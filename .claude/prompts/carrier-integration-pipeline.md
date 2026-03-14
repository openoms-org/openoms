# Prompt: Carrier Integration Pipeline

## Cel

Doprowadź integrację carrier **{CARRIER}** do stanu produkcyjnego. Każdy carrier w OpenOMS składa się z dwóch warstw:

1. **SDK** (`packages/{carrier}-go-sdk/`) — niskopoziomowy klient HTTP do API przewoźnika
2. **Adapter** (`apps/api-server/internal/integration/carriers/{carrier}.go`) — implementacja interfejsu `CarrierProvider`

Integracja InPost jest wzorcowa i działa w produkcji. Nowe integracje muszą osiągnąć ten sam poziom jakości.

## Architektura (nie zmieniaj)

```
Handler (shipment_handler.go)
  → Service (shipment_service.go)
    → integration.NewCarrierProvider("dhl", credentials, settings)
      → factory.go registry (init() auto-register)
        → carriers/dhl.go adapter
          → dhl-go-sdk/client.go
            → DHL HTTP API
```

### Interfejs CarrierProvider (MUSISZ zaimplementować WSZYSTKIE metody)

```go
// Plik: apps/api-server/internal/integration/carrier.go
type CarrierProvider interface {
    ProviderName() string
    CreateShipment(ctx context.Context, req CarrierShipmentRequest) (*CarrierShipmentResponse, error)
    GetLabel(ctx context.Context, externalID string, format string) ([]byte, error)
    GetTracking(ctx context.Context, trackingNumber string) ([]TrackingEvent, error)
    CancelShipment(ctx context.Context, externalID string) error
    MapStatus(carrierStatus string) (omsStatus string, ok bool)
    SupportsPickupPoints() bool
    SearchPickupPoints(ctx context.Context, query string) ([]PickupPoint, error)
    GetRates(ctx context.Context, req RateRequest) ([]Rate, error)
}
```

Opcjonalny interfejs (jeśli carrier wspiera pickup/odbiór kurierski):
```go
type DispatchOrderCreator interface {
    CreateDispatchOrder(ctx context.Context, shipmentExternalIDs []int64, address DispatchOrderAddress, contact DispatchOrderContact) (int64, error)
}
```

### Kluczowe typy danych

```go
type CarrierShipmentRequest struct {
    OrderID       string          // OpenOMS order ID
    ServiceType   string          // carrier-specific service code
    Receiver      CarrierReceiver // name, email, phone, street, city, postal_code, country
    Parcel        CarrierParcel   // size_code, weight_kg, width_cm, height_cm, depth_cm
    TargetPoint   string          // pickup point ID (if applicable)
    SendingMethod string          // e.g. "parcel_locker", "dispatch_order"
    CODAmount     float64         // cash on delivery (0 = none)
    CODCurrency   string          // "PLN", "EUR", etc.
    InsuredValue  float64         // declared value for insurance
    Reference     string          // order reference number
}

type CarrierShipmentResponse struct {
    ExternalID     string  // carrier's shipment ID
    TrackingNumber string  // tracking number
    Status         string  // initial status
    LabelURL       string  // label URL if available
}

type RateRequest struct {
    FromPostalCode, FromCountry string
    ToPostalCode, ToCountry     string
    Weight                      float64 // kg
    Width, Height, Length       float64 // cm
    COD                         float64 // 0 = none
    IsPickupPoint               bool
}

type Rate struct {
    CarrierName, CarrierCode, ServiceName string
    Price                                 float64
    Currency                              string
    EstimatedDays                         int
    PickupPoint                           bool
}
```

## Wzorzec: InPost (produkcyjny, 386 linii adapter, 1631 linii SDK)

### SDK (`packages/inpost-go-sdk/`)

Struktura plików (wzorcowa):
```
client.go          — Client struct, NewClient(), do()/doRaw(), WithSandbox(), WithBaseURL()
models.go          — request/response structs, API-specific types
shipments.go       — ShipmentService (Create, Get, Buy, Delete)
labels.go          — LabelService (Get — PDF/ZPL/EPL)
tracking.go        — TrackingService (Get)
points.go          — PointService (Search — jeśli carrier wspiera pickup points)
dispatch_orders.go — DispatchOrderService (Create — jeśli carrier wspiera)
statusmap.go       — MapStatus() function — carrier statuses → OpenOMS statuses
validation.go      — request validation (opcjonalne)
errors.go          — APIError struct, typed errors
webhook.go         — webhook signature verification (jeśli carrier wysyła webhooks)
doc.go             — package documentation
*_test.go          — unit tests z httptest.NewServer mockiem
```

### Adapter (`carriers/inpost.go`)

Kluczowe elementy:
1. `init()` — rejestracja w factory: `integration.RegisterCarrierProvider("inpost", ...)`
2. Credentials struct — mapuje JSON z zaszyfrowanych credentials w DB
3. `NewXxxProvider(credentials, settings)` — unmarshal credentials, create SDK client
4. Wszystkie 9 metod interfejsu `CarrierProvider`
5. Helpery wewnętrzne (mapParcelTemplate, etc.)

## Procedura wdrożenia (krok po kroku)

### KROK 1: Przeczytaj dokumentację API przewoźnika

Zanim cokolwiek kodujesz, przeczytaj oficjalną dokumentację API:
- Jakie endpointy są dostępne?
- Jaki model auth? (Basic Auth, API key, OAuth2, session token?)
- Jakie formaty labelek? (PDF, ZPL?)
- Czy jest sandbox/testowe środowisko?
- Jakie statusy przesyłek zwraca? (potrzebne do MapStatus)
- Czy wspiera pickup points / punkty odbioru?

### KROK 2: Sprawdź istniejący stan SDK i adaptera

Wiele SDK jest już częściowo napisanych. Sprawdź:
```bash
ls packages/{carrier}-go-sdk/
cat packages/{carrier}-go-sdk/client.go
cat packages/{carrier}-go-sdk/models.go
cat apps/api-server/internal/integration/carriers/{carrier}.go
```

Zidentyfikuj:
- Co jest zaimplementowane?
- Co jest stubbem / TODO?
- Czy modele request/response pasują do prawdziwego API? (porównaj z dokumentacją)
- Czy są testy?

### KROK 3: Zweryfikuj/popraw SDK

Sprawdź i popraw (w tej kolejności):

**3a. client.go — HTTP client**
- [ ] Base URL (production + sandbox) jest poprawny wg dokumentacji API
- [ ] Auth mechanizm jest poprawny (Basic Auth? Bearer token? API key w header?)
- [ ] `doRaw()` poprawnie obsługuje błędy HTTP (4xx/5xx → APIError)
- [ ] `WithSandbox()` i `WithBaseURL()` opcje działają
- [ ] Content-Type i Accept headers poprawne

**3b. models.go — request/response structs**
- [ ] `CreateShipmentRequest` — pola i JSON tagi pasują do dokumentacji API
- [ ] `ShipmentResponse` — pola odpowiadają response body z API
- [ ] Sender/Shipper struct — **MUSI być wypełniony** (częsty bug: adapter nie przekazuje nadawcy)
- [ ] Label response — jak API zwraca labelkę? (base64? URL? binary?)
- [ ] Tracking response — struktura zdarzeń śledzenia

**3c. shipments.go — ShipmentService**
- [ ] `Create()` — endpoint path, HTTP method, request body format
- [ ] `GetLabel()` — endpoint, response format (base64 → decode do []byte)
- [ ] `GetTracking()` — endpoint, response parsing
- [ ] `Cancel()` — endpoint (niektóre API nie wspierają cancel — zwróć sensowny error)

**3d. statusmap.go — status mapping**
- [ ] Zmapuj WSZYSTKIE statusy carrier → OpenOMS:
  ```
  OpenOMS statuses:
    created, label_ready, picked_up, in_transit, out_for_delivery,
    delivered, returned, cancelled, exception
  ```
- [ ] Nieznany status → `("", false)` (nie zgaduj)

**3e. Testy**
- [ ] `client_test.go` z `httptest.NewServer` mockującym API
- [ ] Test happy path dla: Create, GetLabel, GetTracking, Cancel
- [ ] Test error handling (401, 400, 500)
- [ ] Test status mapping (każdy znany status)

### KROK 4: Zweryfikuj/popraw adapter

**4a. init() i factory registration**
```go
func init() {
    integration.RegisterCarrierProvider("{carrier}", func(credentials json.RawMessage, settings json.RawMessage) (integration.CarrierProvider, error) {
        return New{Carrier}Provider(credentials, settings)
    })
}
```

**4b. Credentials struct**
```go
type {Carrier}Credentials struct {
    // Pola zależne od carrier auth model
    APIKey  string `json:"api_key"`
    Sandbox bool   `json:"sandbox,omitempty"`
}
```

**4c. CreateShipment — najważniejsza metoda**

Checklist:
- [ ] Service type — domyślny jeśli pusty (np. "AH" dla DHL domestic)
- [ ] Receiver — wszystkie pola przekazane (name, phone, street, city, postal, country)
- [ ] **Sender/Shipper — MUSI być przekazany** jeśli API wymaga (częsty bug!)
- [ ] Parcel — weight + dimensions LUB size code
- [ ] COD — jeśli `req.CODAmount > 0`, dodaj do żądania
- [ ] Insurance — jeśli `req.InsuredValue > 0`
- [ ] Reference — pass-through
- [ ] Target point — jeśli carrier wspiera pickup points i `req.TargetPoint != ""`
- [ ] Response mapping → `CarrierShipmentResponse{ExternalID, TrackingNumber, Status}`
- [ ] Error wrapping: `fmt.Errorf("{carrier}: create shipment: %w", err)`

**4d. GetLabel**
- [ ] Parse externalID (string → int jeśli API wymaga)
- [ ] Obsłuż format parameter (pdf/zpl/epl) — jeśli carrier nie wspiera formatu, użyj domyślnego
- [ ] Zwróć `[]byte` (raw PDF/ZPL)
- [ ] Jeśli API zwraca base64: `base64.StdEncoding.Decode()`

**4e. GetTracking**
- [ ] Parse tracking events z response
- [ ] Timestamp parsing — RFC3339? Unix? Custom format?
- [ ] Zwróć `[]TrackingEvent{Status, Location, Timestamp, Details}`

**4f. GetRates**
- [ ] Domestic check: `(req.FromCountry == "" || req.FromCountry == "PL") && ...`
- [ ] Jeśli carrier nie ma Rate API — hardcoded tiers (jak InPost/DHL). To jest OK na start
- [ ] Dodaj `COD` surcharge jeśli `req.COD > 0`
- [ ] Currency = "PLN" dla polskich domestic

**4g. SupportsPickupPoints / SearchPickupPoints**
- [ ] `SupportsPickupPoints()` — `true` jeśli carrier ma punkty odbioru
- [ ] `SearchPickupPoints()` — implementacja lub `return nil, nil` jeśli carrier nie wspiera

### KROK 5: Uruchom testy

```bash
# SDK tests
cd packages/{carrier}-go-sdk && go test ./... -v -count=1

# Full API server tests (upewnij się że nic nie zepsułeś)
cd apps/api-server && go test ./... 2>&1 | tail -5

# Vet
cd apps/api-server && go vet ./...
```

### KROK 6: Test z sandbox API (jeśli dostępny)

Jeśli masz credentials do sandbox:
1. Utwórz plik testowy (nie commituj credentials!)
2. Stwórz przesyłkę testową
3. Pobierz labelkę
4. Sprawdź tracking
5. Anuluj przesyłkę

Jeśli NIE masz sandbox — skończ na unit testach z mockami. Sandbox test to zadanie dla użytkownika.

## Znane pułapki

| Problem | Rozwiązanie |
|---------|-------------|
| Adapter nie przekazuje Sender/Shipper | ZAWSZE wypełniaj Shipper z tenant company settings (lub dummy na start) |
| API zwraca label jako base64 string | `base64.StdEncoding.DecodeString(resp.LabelData)` |
| DPD wymaga session token auth | Implementuj token refresh na 401 (patrz dpd-go-sdk/client.go) |
| Status mapping niekompletny | Sprawdź dokumentację — zmapuj WSZYSTKIE statusy, nie zgaduj |
| Rates hardcoded | OK na start, dodaj TODO komentarz z linkiem do Rate API docs |
| Insurance currency hardcoded "PLN" | Użyj `req.CODCurrency` lub tenant default currency |
| Pickup points nie zaimplementowane | Dla door-to-door carrierów (DHL) — `return nil, nil` jest OK |
| go test fails: missing go.sum | `cd packages/{carrier}-go-sdk && go mod tidy` |

## Kolejność carrierów do wdrożenia

1. **DHL** — 65% gotowy, basic auth, brak pickup points, najprostszy
2. **DPD** — 70% gotowy, ALE: bug z Sender, session token auth, pickup points wymagane
3. **GLS** — sprawdź stan SDK
4. **UPS** — sprawdź stan SDK
5. **FedEx** — sprawdź stan SDK
6. **Poczta Polska** — sprawdź stan SDK (PL-only)
7. **Orlen Paczka** — sprawdź stan SDK (PL-only, pickup points)

## Konwencje kodu

- Go fmt enforced
- Error wrapping: `fmt.Errorf("{carrier}: {action}: %w", err)`
- Logging: `slog.Default().With("provider", "{carrier}")` — NIGDY nie loguj credentials
- JSON tags: camelCase dla API requests (zgodne z dokumentacją carrier), snake_case dla OpenOMS structs
- Testy: `httptest.NewServer` + `testify/assert` — NIGDY nie testuj z prawdziwym API w CI
- Nie dodawaj Co-Authored-By ani atrybutów AI w commitach

## Definition of Done

Integracja jest "done" gdy:
- [ ] Wszystkie 9 metod `CarrierProvider` zaimplementowane (nawet jeśli niektóre zwracają nil)
- [ ] `init()` rejestruje provider w factory
- [ ] Credentials struct odpowiada temu co user podaje w UI (API key / username+password / token)
- [ ] Unit testy przechodzą: `go test ./... -count=1`
- [ ] `go vet ./...` bez błędów w SDK i api-server
- [ ] Status mapping pokrywa przynajmniej: created, in_transit, delivered, returned, cancelled
- [ ] CreateShipment tworzy przesyłkę z prawidłowym receiver, parcel, i opcjonalnym COD
- [ ] GetLabel zwraca raw bytes (PDF) — nie base64 string
- [ ] Error messages mają prefix carrier name: `"dhl: ..."`, `"dpd: ..."`
- [ ] Brak hardcoded credentials w kodzie
- [ ] Brak polskich stringów w error messages (poza UI-facing messages jak InPost payment error)
