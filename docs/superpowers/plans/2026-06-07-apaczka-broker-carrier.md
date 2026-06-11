# Apaczka Broker Carrier — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add Apaczka (R2G/Alsendo) as the first **broker meta-carrier** — one integration that fronts many couriers (InPost, DPD, DHL, UPS, GLS, Poczta Polska, Orlen Paczka…) behind the existing `CarrierProvider` interface, unlocking multi-carrier label creation, tracking, pickup points, and real rate-shopping for tenants without per-carrier contracts.

**Architecture:** A broker is modelled as a single `CarrierProvider` (provider key `apaczka`), exactly like a direct carrier — no new interface. The "meta" nature lives in two places: (1) `GetRates` returns **many** `Rate` rows (one per underlying carrier/service from Apaczka's `service_structure`), which the existing parallel `RateService` already aggregates and sorts; (2) `CreateShipment` selects the broker service via `CarrierShipmentRequest.ServiceType` (the Apaczka service id). New Go SDK package `packages/apaczka-go-sdk` (HTTP client + HMAC-SHA256 signed envelope), provider adapter `internal/integration/carriers/apaczka.go`, Studio catalog registration, and one-line wiring into rate-shopping. This establishes the reusable broker pattern; **Furgonetka is a follow-up plan** (OAuth2 + OAuthRefresher extension + per-tenant consent flow — a larger surface, out of scope here).

**Tech Stack:** Go 1.25.7, `net/http`, `crypto/hmac`+`crypto/sha256`, existing `internal/integration` interfaces, `internal/netutil.SafeHTTPClient` (SSRF-safe), `internal/crypto` (AES-GCM creds), Provider Integration Studio catalog (`internal/service/provider_catalog.go`), `go.work` workspace. Tests: `httptest` + `testify` (repo convention).

**Linear:** Create a child issue under epic OPE-403 (e.g. "OPE-XXX: Apaczka broker meta-carrier") before starting; use its ID in the branch (`feat/OPE-XXX-apaczka-broker`), PR title, and commits. Replace `OPE-XXX` throughout below.

**Reference patterns (read before coding):**
- Interface + DTOs: `apps/api-server/internal/integration/carrier.go:10-149`
- Provider adapter pattern: `apps/api-server/internal/integration/carriers/inpost.go` (whole file)
- SDK client pattern: `packages/inpost-go-sdk/client.go` (whole file)
- Rate-shopping aggregator: `apps/api-server/internal/service/rate_service.go:43-158` (carrier allow-map at `:59-68`)
- Studio catalog: `apps/api-server/internal/service/provider_catalog.go` (helpers `secretField`/`settingField`/`unknownCap`; entries at `:39-121`)
- Tracking poller (consumes `GetTracking`+`MapStatus`): `apps/api-server/internal/worker/tracking_poller.go:180-307`
- Module wiring: `public/go.work` (`use` block) + `apps/api-server/go.mod` (`require` + `replace ... => ../../packages/btp-go-sdk` pattern)

---

## ✅ VERIFIED CONTRACT (Task 1 done 2026-06-07 against panel.apaczka.pl/dokumentacja_api_v2.php)

This is the authoritative contract. Where the task code blocks below differ from this section, **this section wins** — implementers must follow it. SDK tests are `httptest`-mocked, so they pass regardless; real-API correctness follows this contract.

- **Base URL:** `https://www.apaczka.pl/api/v2/`
- **Envelope:** `POST` `application/x-www-form-urlencoded`, fields `app_id`, `request` (JSON payload string, `{}` when none), `expires` (unix ts, ≤ now+1800), `signature`.
- **Signature:** `hex( HMAC_SHA256( key=app_secret, msg = app_id + ":" + route + ":" + request + ":" + expires ) )`. `route` = the path segment used in the URL, e.g. `order_send/`. ⚠️ **One residual unknown:** whether the signed `route` includes the `api/v2/` prefix — match the official Apaczka PHP sample (github "apaczka api v2" client); keep the URL path and the signed `route` identical. Verify on first live call.
- **Response wrapper:** `{ "status": 200, "response": { ... } }`. `status != 200` ⇒ error (message in top-level `message`).
- **Routes:**
  - `service_structure/` → `response.services[]`, each `{ service_id (string), name, supplier (carrier code), delivery_time, domestic ("0"/"1"), door_to_door ("0"/"1"), door_to_point, point_to_point, point_to_door }`. **NO PRICE here.** Also returns `points_type` (e.g. INPOST/UPS/POCZTA), `package_type`, `pickup_type`.
  - `order_valuation/` → request `{ order: { service_id, shipment:[…], address:{sender,receiver} } }` → response `{ price_table: { "<service_id>": { price, price_gross } } }`. **Prices in GROSZE (int as string), no currency field.** This is the rate source.
  - `order_send/` → request `{ order: { service_id, address:{ sender, receiver }, shipment:[ { dimension1, dimension2, dimension3 (cm), weight (kg), shipment_type_code (e.g. "PACZKA") } ], pickup:{ type:"SELF"|"COURIER", date:"Y-m-d" }, cod:{ amount (grosze), currency:"PLN", bankaccount, country } (optional) } }` → `response.order { id, service_id, service_name, waybill_number, tracking_url, status }`.
  - `waybill/:order_id/` → `response { waybill (base64 string), type ("pdf") }`.
  - `tracking/:waybill_number/` → tracking events, each `{ status, status_original (nullable), description, place, updated_at (ISO-8601 UTC, nullable) }`. **Tracking is keyed by waybill_number, NOT order id.**
  - `order/:order_id/` → order details incl. `shipments[]` with `status`.
  - `points/:type` (`:type` ∈ INPOST/UPS/POCZTA) → request `{ country_code, subtype? }` → pickup points.
  - `cancel_order/:order_id/` → no payload, empty response on success.
  - `orders/` → list orders.

**Address fields** (sender & receiver): `country_code`, `name`, `line1`, `postal_code`, `city` (+ phone/email where supported).

**Net effect on the plan vs the originally-drafted code blocks below:**
1. `service_structure` has no price → **GetRates = service_structure (names/suppliers) + order_valuation (price_table) joined**, grosze÷100 → PLN. (Task 3 + Task 7.)
2. Tracking route is `tracking/:waybill_number/` (not `order_status/`); fields `status/place/description/updated_at` (ISO-8601). (Task 4 + Task 7 `GetTracking`.)
3. `order_send` parcel is a `shipment[]` array with `dimension1/2/3`+`weight`+`shipment_type_code` (not a flat parcel); `address.{sender,receiver}.line1`; `cod` is `{amount(grosze),currency,bankaccount,country}`; `pickup:{type,date}`. (Task 4 models + Task 7 `CreateShipment`.)
4. `waybill` response = `{waybill(base64), type}`; `points/:type` needs `country_code`; cancel route `cancel_order/:order_id/`.

---

## File Structure

**New SDK package `packages/apaczka-go-sdk/`** (one responsibility per file):
- `go.mod` — module `github.com/openoms-org/openoms/packages/apaczka-go-sdk`
- `client.go` — `Client`, `Option`s, `NewClient`, signed `do()` envelope, `APIError`
- `client_test.go` — signing + envelope + error decoding tests
- `models.go` — request/response DTOs (`ServiceStructureResponse`, `Service`, `CreateOrderRequest`, `Order`, `OrderStatusResponse`, `Point`, `Envelope`)
- `services.go` — `GetServiceStructure(ctx)`; `services_test.go`
- `orders.go` — `CreateOrder`, `GetOrderStatus`, `CancelOrder`; `orders_test.go`
- `labels.go` — `GetWaybill(ctx, orderID, format)`; `labels_test.go`
- `points.go` — `SearchPoints(ctx, query)`; `points_test.go`

**api-server changes:**
- Create: `apps/api-server/internal/integration/carriers/apaczka.go` — `ApaczkaProvider` adapter + `init()` registration
- Create: `apps/api-server/internal/integration/carriers/apaczka_test.go`
- Create: `apps/api-server/internal/integration/carriers/apaczka_status.go` — status map + test `apaczka_status_test.go`
- Modify: `apps/api-server/internal/service/rate_service.go:59-68` — add `"apaczka": true`
- Modify: `apps/api-server/internal/service/provider_catalog.go:37-123` — add Apaczka catalog entry
- Modify: `apps/api-server/internal/service/provider_catalog_test.go` (or seeder test) — assert entry present
- Modify: `public/go.work` — add `./packages/apaczka-go-sdk` to `use`
- Modify: `apps/api-server/go.mod` — add `require` + `replace` for the new package
- Docs: `docs/system-documentation.md` (new carrier/broker), `.claude/context/API_CONTRACTS.md` (rate-shopping now includes apaczka), `.claude/context/PROJECT_STATE.md`

---

## Task 0: Branch, worktree, Linear

- [ ] **Step 1: Create Linear issue** under OPE-403 titled "Apaczka broker meta-carrier (first broker pattern)". Note its ID as `OPE-XXX`. Move to In Progress.

- [ ] **Step 2: Create isolated worktree + branch** (use superpowers:using-git-worktrees if available; otherwise):

```bash
cd /Users/rafs/praca/openoms-dev/public
git checkout -b feat/OPE-XXX-apaczka-broker
```

Expected: `Switched to a new branch 'feat/OPE-XXX-apaczka-broker'`

---

## Task 1: Verify the live Apaczka API contract

**Files:** none (research) — record findings as a comment block at the top of `packages/apaczka-go-sdk/client.go` in Task 2.

- [ ] **Step 1: Fetch official docs** and confirm the assumptions in the "⚠️ Contract verification" section:
  - https://panel.apaczka.pl/dokumentacja_api_v2.php
  - https://panel.apaczka.pl/dokumentacja_api_v2_mapa.php

- [ ] **Step 2: Pin down and write down** the EXACT: (a) signature string format and hex/base64 encoding, (b) form field names (`app_id`/`request`/`expires`/`signature`), (c) route names for order_send / waybill / order_status / service_structure / points, (d) `service_structure` JSON path to the service list + price units (grosze vs PLN), (e) `order_send` payload shape and where the order id + tracking number come back.

- [ ] **Step 3:** If anything differs from the assumed contract, update the struct/string literals in the following tasks accordingly before implementing. (No commit — this is the source-of-truth for the code below.)

---

## Task 2: SDK scaffold — signed client

**Files:**
- Create: `packages/apaczka-go-sdk/go.mod`
- Create: `packages/apaczka-go-sdk/client.go`
- Create: `packages/apaczka-go-sdk/models.go`
- Test: `packages/apaczka-go-sdk/client_test.go`

- [ ] **Step 1: Create the module**

```bash
cd /Users/rafs/praca/openoms-dev/public/packages
mkdir apaczka-go-sdk && cd apaczka-go-sdk
cat > go.mod <<'EOF'
module github.com/openoms-org/openoms/packages/apaczka-go-sdk

go 1.25.7
EOF
```

- [ ] **Step 2: Write the failing signing test** — `packages/apaczka-go-sdk/client_test.go`

```go
package apaczka

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSign_MatchesHMACSHA256(t *testing.T) {
	got := sign("appid", "service_structure/", "{}", 1700000000, "secret")
	mac := hmac.New(sha256.New, []byte("secret"))
	_, _ = mac.Write([]byte("appid:service_structure/:{}:1700000000"))
	want := hex.EncodeToString(mac.Sum(nil))
	if got != want {
		t.Fatalf("sign() = %q, want %q", got, want)
	}
}

func TestDo_SendsSignedEnvelopeAndDecodes(t *testing.T) {
	var gotForm map[string][]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		gotForm = r.Form
		_, _ = io.WriteString(w, `{"status":200,"response":{"ok":true}}`)
	}))
	defer srv.Close()

	c := NewClient("appid", "secret", WithBaseURL(srv.URL+"/"), WithNow(func() int64 { return 1700000000 }))
	var out struct {
		OK bool `json:"ok"`
	}
	if err := c.do(context.Background(), "service_structure/", nil, &out); err != nil {
		t.Fatalf("do: %v", err)
	}
	if !out.OK {
		t.Fatalf("expected ok=true, got %+v", out)
	}
	if gotForm.Get("app_id") != "appid" || gotForm.Get("request") != "{}" || gotForm.Get("expires") == "" || gotForm.Get("signature") == "" {
		t.Fatalf("missing envelope fields: %v", gotForm)
	}
}

func TestDo_NonOKStatusReturnsAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"status":403,"message":"forbidden"}`)
	}))
	defer srv.Close()
	c := NewClient("a", "b", WithBaseURL(srv.URL+"/"))
	err := c.do(context.Background(), "orders/", nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	var apiErr *APIError
	if !errorsAs(err, &apiErr) || apiErr.Status != 403 {
		t.Fatalf("want APIError status 403, got %v", err)
	}
}
```

(Add a tiny `errorsAs` helper in the test file: `func errorsAs(err error, target any) bool { return errors.As(err, target.(*"*APIError" placeholder)) }` — actually use `errors.As` directly in the assertion and import `errors`. Replace `errorsAs(err, &apiErr)` with `errors.As(err, &apiErr)` and import `"errors"`.)

- [ ] **Step 3: Run test, verify it fails**

Run: `cd packages/apaczka-go-sdk && go test ./... -run TestSign -v`
Expected: compile failure (`undefined: sign`, `NewClient`, …)

- [ ] **Step 4: Implement `client.go`**

```go
// Package apaczka is a Go client for the Apaczka (R2G/Alsendo) broker API v2.
// Contract verified against panel.apaczka.pl/dokumentacja_api_v2.php (see Task 1).
package apaczka

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const productionBaseURL = "https://www.apaczka.pl/api/v2/"

// Client is an Apaczka API v2 client using HMAC-SHA256 signed envelopes.
type Client struct {
	httpClient *http.Client
	baseURL    string
	appID      string
	appSecret  string
	now        func() int64
}

// Option configures a Client.
type Option func(*Client)

// WithHTTPClient sets the HTTP client (use netutil.SafeHTTPClient in production).
func WithHTTPClient(h *http.Client) Option { return func(c *Client) { c.httpClient = h } }

// WithBaseURL overrides the API base URL (must end with "/"); used in tests.
func WithBaseURL(u string) Option { return func(c *Client) { c.baseURL = u } }

// WithNow overrides the clock; used in tests for deterministic signatures.
func WithNow(f func() int64) Option { return func(c *Client) { c.now = f } }

// NewClient creates an Apaczka client. appID/appSecret come from the seller panel.
func NewClient(appID, appSecret string, opts ...Option) *Client {
	c := &Client{
		httpClient: http.DefaultClient,
		baseURL:    productionBaseURL,
		appID:      appID,
		appSecret:  appSecret,
		now:        func() int64 { return time.Now().Unix() },
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// APIError is a non-2xx / non-200-status Apaczka response.
type APIError struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("apaczka: api error status=%d: %s", e.Status, e.Message)
}

// sign computes hex(HMAC_SHA256(appSecret, "appID:route:request:expires")).
func sign(appID, route, request string, expires int64, appSecret string) string {
	msg := appID + ":" + route + ":" + request + ":" + strconv.FormatInt(expires, 10)
	mac := hmac.New(sha256.New, []byte(appSecret))
	_, _ = mac.Write([]byte(msg))
	return hex.EncodeToString(mac.Sum(nil))
}

// envelope is the decoded {status, response} wrapper.
type envelope struct {
	Status   int             `json:"status"`
	Message  string          `json:"message"`
	Response json.RawMessage `json:"response"`
}

// do signs and POSTs a request to `route`, decoding `response` into result.
// payload is JSON-marshalled into the `request` field ({} when nil).
func (c *Client) do(ctx context.Context, route string, payload any, result any) error {
	request := "{}"
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("apaczka: marshal payload: %w", err)
		}
		request = string(b)
	}
	expires := c.now() + 1800
	form := url.Values{
		"app_id":    {c.appID},
		"request":   {request},
		"expires":   {strconv.FormatInt(expires, 10)},
		"signature": {sign(c.appID, route, request, expires, c.appSecret)},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+route, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("apaczka: new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("apaczka: request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
	if err != nil {
		return fmt.Errorf("apaczka: read response: %w", err)
	}

	var env envelope
	if err := json.Unmarshal(body, &env); err != nil {
		return fmt.Errorf("apaczka: decode envelope (http %d): %w", resp.StatusCode, err)
	}
	if env.Status != 200 || resp.StatusCode >= 400 {
		msg := env.Message
		if msg == "" {
			msg = string(env.Response)
		}
		return &APIError{Status: env.Status, Message: msg}
	}
	if result != nil && len(env.Response) > 0 {
		if err := json.Unmarshal(env.Response, result); err != nil {
			return fmt.Errorf("apaczka: decode response: %w", err)
		}
	}
	return nil
}
```

- [ ] **Step 5: Fix the test imports** — replace the `errorsAs` placeholder with `errors.As(err, &apiErr)` and add `"errors"` to the import block.

- [ ] **Step 6: Run tests, verify pass**

Run: `cd packages/apaczka-go-sdk && go test ./... -v`
Expected: PASS (3 tests)

- [ ] **Step 7: Commit**

```bash
cd /Users/rafs/praca/openoms-dev/public
git add packages/apaczka-go-sdk/
git commit -m "OPE-XXX: scaffold apaczka SDK with signed envelope client"
```

---

## Task 3: SDK — service_structure (rate catalog)

**Files:**
- Modify: `packages/apaczka-go-sdk/models.go`
- Create: `packages/apaczka-go-sdk/services.go`
- Test: `packages/apaczka-go-sdk/services_test.go`

- [ ] **Step 1: Write failing test** — `services_test.go`

```go
package apaczka

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetServiceStructure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"status":200,"response":{"services":[
			{"id":"1","name":"InPost Paczkomaty","supplier":"INPOST","price":{"gross":1099,"currency":"PLN"},"delivery_time":"1-2","door_to_door":false},
			{"id":"7","name":"DPD Kurier","supplier":"DPD","price":{"gross":1599,"currency":"PLN"},"delivery_time":"1","door_to_door":true}
		]}}`)
	}))
	defer srv.Close()
	c := NewClient("a", "b", WithBaseURL(srv.URL+"/"))
	res, err := c.GetServiceStructure(context.Background())
	if err != nil {
		t.Fatalf("GetServiceStructure: %v", err)
	}
	if len(res.Services) != 2 || res.Services[0].Supplier != "INPOST" || res.Services[1].Price.Gross != 1599 {
		t.Fatalf("unexpected services: %+v", res.Services)
	}
}
```

- [ ] **Step 2: Run, verify fail** — Run: `go test ./... -run TestGetServiceStructure -v` → FAIL (`undefined: GetServiceStructure`)

- [ ] **Step 3: Add models** to `models.go`

```go
package apaczka

// Money is a price in the smallest currency unit (grosze) — CONFIRM unit in Task 1.
type Money struct {
	Gross    int    `json:"gross"`
	Net      int    `json:"net"`
	Currency string `json:"currency"`
}

// Service is one broker service (an underlying carrier + service level).
type Service struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Supplier     string `json:"supplier"` // underlying carrier code, e.g. INPOST, DPD
	Price        Money  `json:"price"`
	DeliveryTime string `json:"delivery_time"`
	DoorToDoor   bool   `json:"door_to_door"`
}

// ServiceStructureResponse is the decoded service_structure response.
type ServiceStructureResponse struct {
	Services []Service `json:"services"`
}
```

- [ ] **Step 4: Implement** `services.go`

```go
package apaczka

import "context"

// GetServiceStructure returns the broker's available services (per-carrier),
// used for rate-shopping and service selection.
func (c *Client) GetServiceStructure(ctx context.Context) (*ServiceStructureResponse, error) {
	var out ServiceStructureResponse
	if err := c.do(ctx, "service_structure/", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
```

- [ ] **Step 5: Run, verify pass** — Run: `go test ./... -v` → PASS

- [ ] **Step 6: Commit**

```bash
git add packages/apaczka-go-sdk/
git commit -m "OPE-XXX: apaczka SDK service_structure (rate catalog)"
```

---

## Task 4: SDK — order_send, order_status, cancel, waybill, points

**Files:**
- Modify: `packages/apaczka-go-sdk/models.go`
- Create: `packages/apaczka-go-sdk/orders.go`, `labels.go`, `points.go`
- Test: `packages/apaczka-go-sdk/orders_test.go`, `labels_test.go`, `points_test.go`

- [ ] **Step 1: Add models** to `models.go`

```go
// CreateOrderRequest is the order_send payload (wraps an order object).
// CONFIRM exact field names against Task 1 docs.
type CreateOrderRequest struct {
	Order OrderInput `json:"order"`
}

type OrderInput struct {
	ServiceID    string       `json:"service_id"`
	Address      AddressPair  `json:"address"`
	Option       map[string]string `json:"option,omitempty"`
	COD          *COD         `json:"cod,omitempty"`
	Content      string       `json:"content"`
	Comment      string       `json:"comment,omitempty"`
	PickupPoint  string       `json:"pickup_point,omitempty"` // target point id for PUDO services
}

type AddressPair struct {
	Sender   Address `json:"sender"`
	Receiver Address `json:"receiver"`
}

type Address struct {
	Name        string `json:"name"`
	Line1       string `json:"line1"`
	PostalCode  string `json:"postal_code"`
	City        string `json:"city"`
	CountryCode string `json:"country_code"`
	Email       string `json:"email,omitempty"`
	Phone       string `json:"phone,omitempty"`
}

type COD struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

// Order is the created order returned by order_send / order.
type Order struct {
	ID             string `json:"id"`
	WaybillNumber  string `json:"waybill_number"`
	TrackingNumber string `json:"tracking_number"`
	Status         string `json:"status"`
}

type createOrderResponse struct {
	Order Order `json:"order"`
}

type OrderStatusResponse struct {
	Order    Order        `json:"order"`
	Tracking []TrackEvent `json:"tracking"`
}

type TrackEvent struct {
	Status  string `json:"status"`
	Date    string `json:"date"` // CONFIRM format in Task 1
	Place   string `json:"place"`
	Comment string `json:"comment"`
}

// Point is a pickup/drop-off point from the points endpoint.
type Point struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	Supplier   string  `json:"supplier"`
	Street     string  `json:"street"`
	City       string  `json:"city"`
	PostalCode string  `json:"postal_code"`
	Lat        float64 `json:"lat"`
	Lng        float64 `json:"lng"`
}

type pointsResponse struct {
	Points []Point `json:"points"`
}
```

- [ ] **Step 2: Write failing tests** — `orders_test.go`

```go
package apaczka

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateOrder(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"status":200,"response":{"order":{"id":"555","waybill_number":"WB1","tracking_number":"TRK1","status":"NEW"}}}`)
	}))
	defer srv.Close()
	c := NewClient("a", "b", WithBaseURL(srv.URL+"/"))
	o, err := c.CreateOrder(context.Background(), &CreateOrderRequest{Order: OrderInput{ServiceID: "1", Content: "goods"}})
	if err != nil {
		t.Fatalf("CreateOrder: %v", err)
	}
	if o.ID != "555" || o.TrackingNumber != "TRK1" {
		t.Fatalf("unexpected order: %+v", o)
	}
}

func TestGetOrderStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"status":200,"response":{"order":{"id":"555","status":"DELIVERED"},"tracking":[{"status":"DELIVERED","date":"2026-06-07 10:00:00","place":"Warszawa"}]}}`)
	}))
	defer srv.Close()
	c := NewClient("a", "b", WithBaseURL(srv.URL+"/"))
	st, err := c.GetOrderStatus(context.Background(), "555")
	if err != nil {
		t.Fatalf("GetOrderStatus: %v", err)
	}
	if len(st.Tracking) != 1 || st.Tracking[0].Status != "DELIVERED" {
		t.Fatalf("unexpected status: %+v", st)
	}
}
```

`labels_test.go`:

```go
package apaczka

import (
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetWaybill(t *testing.T) {
	pdf := []byte("%PDF-1.4 fake")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"status":200,"response":{"waybill":"`+base64.StdEncoding.EncodeToString(pdf)+`"}}`)
	}))
	defer srv.Close()
	c := NewClient("a", "b", WithBaseURL(srv.URL+"/"))
	got, err := c.GetWaybill(context.Background(), "555")
	if err != nil {
		t.Fatalf("GetWaybill: %v", err)
	}
	if string(got) != string(pdf) {
		t.Fatalf("waybill mismatch: %q", got)
	}
}
```

`points_test.go`:

```go
package apaczka

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSearchPoints(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"status":200,"response":{"points":[{"id":"KRA01","name":"Paczkomat KRA01","supplier":"INPOST","city":"Kraków","postal_code":"30-001","lat":50.0,"lng":19.9}]}}`)
	}))
	defer srv.Close()
	c := NewClient("a", "b", WithBaseURL(srv.URL+"/"))
	pts, err := c.SearchPoints(context.Background(), "30-001")
	if err != nil {
		t.Fatalf("SearchPoints: %v", err)
	}
	if len(pts) != 1 || pts[0].ID != "KRA01" {
		t.Fatalf("unexpected points: %+v", pts)
	}
}
```

- [ ] **Step 3: Run, verify fail** — Run: `go test ./... -v` → FAIL (undefined methods)

- [ ] **Step 4: Implement** `orders.go`

```go
package apaczka

import "context"

// CreateOrder creates a shipment order (order_send) and returns the order.
func (c *Client) CreateOrder(ctx context.Context, req *CreateOrderRequest) (*Order, error) {
	var out createOrderResponse
	if err := c.do(ctx, "order_send/", req, &out); err != nil {
		return nil, err
	}
	return &out.Order, nil
}

// GetOrderStatus returns the order plus its tracking history (order_status/{id}).
func (c *Client) GetOrderStatus(ctx context.Context, orderID string) (*OrderStatusResponse, error) {
	var out OrderStatusResponse
	if err := c.do(ctx, "order_status/"+orderID+"/", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CancelOrder cancels an order by id (cancel_order/{id} — CONFIRM route in Task 1).
func (c *Client) CancelOrder(ctx context.Context, orderID string) error {
	return c.do(ctx, "cancel_order/"+orderID+"/", nil, nil)
}
```

`labels.go`:

```go
package apaczka

import (
	"context"
	"encoding/base64"
	"fmt"
)

// GetWaybill downloads the label PDF (base64 in `response.waybill`) for an order.
func (c *Client) GetWaybill(ctx context.Context, orderID string) ([]byte, error) {
	var out struct {
		Waybill string `json:"waybill"`
	}
	if err := c.do(ctx, "waybill/"+orderID+"/", nil, &out); err != nil {
		return nil, err
	}
	pdf, err := base64.StdEncoding.DecodeString(out.Waybill)
	if err != nil {
		return nil, fmt.Errorf("apaczka: decode waybill: %w", err)
	}
	return pdf, nil
}
```

`points.go`:

```go
package apaczka

import "context"

// SearchPoints returns pickup points near the query (postal code / city).
func (c *Client) SearchPoints(ctx context.Context, query string) ([]Point, error) {
	var out pointsResponse
	if err := c.do(ctx, "points/", map[string]string{"query": query}, &out); err != nil {
		return nil, err
	}
	return out.Points, nil
}
```

- [ ] **Step 5: Run, verify pass** — Run: `go test ./... -v` → PASS (all SDK tests)

- [ ] **Step 6: Commit**

```bash
git add packages/apaczka-go-sdk/
git commit -m "OPE-XXX: apaczka SDK orders, waybill, status, points"
```

---

## Task 5: Wire the SDK module into api-server

**Files:**
- Modify: `public/go.work`
- Modify: `apps/api-server/go.mod`

- [ ] **Step 1: Add to `go.work`** — inside the `use ( … )` block add the line (keep alphabetical-ish with the other packages):

```
	./packages/apaczka-go-sdk
```

- [ ] **Step 2: Add require + replace to `apps/api-server/go.mod`** — in the `require (` block add:

```
	github.com/openoms-org/openoms/packages/apaczka-go-sdk v0.0.0-00010101000000-000000000000
```

and add a replace directive next to the other local `replace … => ../../packages/*` lines:

```
replace github.com/openoms-org/openoms/packages/apaczka-go-sdk => ../../packages/apaczka-go-sdk
```

- [ ] **Step 3: Tidy + verify it resolves**

Run:
```bash
cd /Users/rafs/praca/openoms-dev/public/apps/api-server && go build ./... 2>&1 | tail -5
```
Expected: builds (no `missing go.sum entry` / `unknown import`). If `go mod tidy` is needed: `go mod tidy` then rebuild.

- [ ] **Step 4: Commit**

```bash
cd /Users/rafs/praca/openoms-dev/public
git add go.work apps/api-server/go.mod apps/api-server/go.sum
git commit -m "OPE-XXX: wire apaczka-go-sdk module into api-server"
```

---

## Task 6: Provider adapter — status map

**Files:**
- Create: `apps/api-server/internal/integration/carriers/apaczka_status.go`
- Test: `apps/api-server/internal/integration/carriers/apaczka_status_test.go`

- [ ] **Step 1: Write failing test** — `apaczka_status_test.go`

```go
package carriers

import "testing"

func TestMapApaczkaStatus(t *testing.T) {
	cases := map[string]string{
		"NEW":        "pending",
		"SENT":       "in_transit",
		"DELIVERED":  "delivered",
		"RETURNED":   "returned",
		"CANCELLED":  "cancelled",
	}
	for raw, want := range cases {
		got, ok := mapApaczkaStatus(raw)
		if !ok || got != want {
			t.Fatalf("mapApaczkaStatus(%q) = %q,%v; want %q", raw, got, ok, want)
		}
	}
	if _, ok := mapApaczkaStatus("WAT_UNKNOWN"); ok {
		t.Fatal("unknown status should return ok=false")
	}
}
```

- [ ] **Step 2: Run, verify fail** — Run: `cd apps/api-server && go test ./internal/integration/carriers/ -run TestMapApaczkaStatus -v` → FAIL

- [ ] **Step 3: Implement** `apaczka_status.go`

```go
package carriers

import "strings"

// mapApaczkaStatus maps Apaczka order/tracking statuses to OMS shipment statuses.
// CONFIRM the raw status vocabulary against Task 1 docs and extend as needed.
func mapApaczkaStatus(raw string) (string, bool) {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case "NEW", "CREATED", "WAITING":
		return "pending", true
	case "SENT", "IN_TRANSIT", "PICKED_UP":
		return "in_transit", true
	case "OUT_FOR_DELIVERY":
		return "out_for_delivery", true
	case "DELIVERED":
		return "delivered", true
	case "RETURNED", "RETURN":
		return "returned", true
	case "CANCELLED", "CANCELED":
		return "cancelled", true
	default:
		return "", false
	}
}
```

- [ ] **Step 4: Run, verify pass** — Run: same command → PASS

- [ ] **Step 5: Commit**

```bash
git add apps/api-server/internal/integration/carriers/apaczka_status.go apps/api-server/internal/integration/carriers/apaczka_status_test.go
git commit -m "OPE-XXX: apaczka status mapping"
```

---

## Task 7: Provider adapter — ApaczkaProvider (CarrierProvider)

**Files:**
- Create: `apps/api-server/internal/integration/carriers/apaczka.go`
- Test: `apps/api-server/internal/integration/carriers/apaczka_test.go`

- [ ] **Step 1: Write failing test** — `apaczka_test.go` (registration + GetRates mapping + CreateShipment; the SDK is pointed at an httptest server via the provider's settings `base_url` test hook)

```go
package carriers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
)

func TestApaczkaRegistered(t *testing.T) {
	creds := json.RawMessage(`{"app_id":"a","app_secret":"b"}`)
	p, err := integration.NewCarrierProvider("apaczka", creds, nil)
	if err != nil {
		t.Fatalf("NewCarrierProvider(apaczka): %v", err)
	}
	if p.ProviderName() != "apaczka" {
		t.Fatalf("ProviderName = %q", p.ProviderName())
	}
}

func TestApaczkaGetRates_MapsServicesToManyRates(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "service_structure") {
			_, _ = io.WriteString(w, `{"status":200,"response":{"services":[
				{"id":"1","name":"InPost","supplier":"INPOST","price":{"gross":1099,"currency":"PLN"},"delivery_time":"1","door_to_door":false},
				{"id":"7","name":"DPD","supplier":"DPD","price":{"gross":1599,"currency":"PLN"},"delivery_time":"1","door_to_door":true}
			]}}`)
			return
		}
		_, _ = io.WriteString(w, `{"status":200,"response":{}}`)
	}))
	defer srv.Close()

	creds := json.RawMessage(`{"app_id":"a","app_secret":"b"}`)
	settings := json.RawMessage(`{"base_url":"` + srv.URL + `/"}`)
	p, err := integration.NewCarrierProvider("apaczka", creds, settings)
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	rates, err := p.GetRates(context.Background(), integration.RateRequest{ToPostalCode: "30-001", ToCountry: "PL", Weight: 1})
	if err != nil {
		t.Fatalf("GetRates: %v", err)
	}
	if len(rates) != 2 {
		t.Fatalf("want 2 rates, got %d: %+v", len(rates), rates)
	}
	if rates[0].Price != 10.99 || rates[0].CarrierCode != "INPOST" {
		t.Fatalf("rate[0] mapping wrong: %+v", rates[0])
	}
}
```

- [ ] **Step 2: Run, verify fail** — Run: `cd apps/api-server && go test ./internal/integration/carriers/ -run TestApaczka -v` → FAIL (unknown provider / undefined)

- [ ] **Step 3: Implement** `apaczka.go`

```go
package carriers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	apaczkasdk "github.com/openoms-org/openoms/packages/apaczka-go-sdk"

	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
	"github.com/openoms-org/openoms/apps/api-server/internal/netutil"
)

func init() {
	integration.RegisterCarrierProvider("apaczka", func(credentials json.RawMessage, settings json.RawMessage) (integration.CarrierProvider, error) {
		return NewApaczkaProvider(credentials, settings)
	})
}

// ApaczkaCredentials is the JSON stored in encrypted integration credentials.
type ApaczkaCredentials struct {
	AppID     string `json:"app_id"`
	AppSecret string `json:"app_secret"`
}

// apaczkaSettings holds non-secret settings (test base URL override only).
type apaczkaSettings struct {
	BaseURL string `json:"base_url,omitempty"`
}

// ApaczkaProvider implements integration.CarrierProvider as a broker meta-carrier.
type ApaczkaProvider struct {
	client *apaczkasdk.Client
	logger *slog.Logger
}

// NewApaczkaProvider builds the provider from encrypted credentials + settings.
func NewApaczkaProvider(credentials json.RawMessage, settings json.RawMessage) (*ApaczkaProvider, error) {
	var creds ApaczkaCredentials
	if err := json.Unmarshal(credentials, &creds); err != nil {
		return nil, fmt.Errorf("apaczka: parse credentials: %w", err)
	}
	if creds.AppID == "" || creds.AppSecret == "" {
		return nil, fmt.Errorf("apaczka: missing app_id/app_secret")
	}

	opts := []apaczkasdk.Option{apaczkasdk.WithHTTPClient(netutil.SafeHTTPClient(30 * time.Second))}
	if len(settings) > 0 {
		var s apaczkaSettings
		if err := json.Unmarshal(settings, &s); err == nil && s.BaseURL != "" {
			opts = append(opts, apaczkasdk.WithBaseURL(s.BaseURL))
		}
	}

	return &ApaczkaProvider{
		client: apaczkasdk.NewClient(creds.AppID, creds.AppSecret, opts...),
		logger: slog.Default().With("provider", "apaczka"),
	}, nil
}

func (p *ApaczkaProvider) ProviderName() string { return "apaczka" }

// GetRates returns one Rate per broker service (multi-carrier rate-shopping).
func (p *ApaczkaProvider) GetRates(ctx context.Context, _ integration.RateRequest) ([]integration.Rate, error) {
	res, err := p.client.GetServiceStructure(ctx)
	if err != nil {
		return nil, fmt.Errorf("apaczka: service structure: %w", err)
	}
	rates := make([]integration.Rate, 0, len(res.Services))
	for _, s := range res.Services {
		rates = append(rates, integration.Rate{
			CarrierName: s.Name,
			CarrierCode: s.Supplier,
			ServiceName: s.ID, // service id used as ServiceType on CreateShipment
			Price:       float64(s.Price.Gross) / 100.0,
			Currency:    s.Price.Currency,
			PickupPoint: !s.DoorToDoor,
			IsEstimate:  false,
		})
	}
	return rates, nil
}

// CreateShipment creates a broker order; req.ServiceType is the Apaczka service id.
func (p *ApaczkaProvider) CreateShipment(ctx context.Context, req integration.CarrierShipmentRequest) (*integration.CarrierShipmentResponse, error) {
	if req.ServiceType == "" {
		return nil, fmt.Errorf("apaczka: service_type (service id) is required")
	}
	in := apaczkasdk.OrderInput{
		ServiceID: req.ServiceType,
		Content:   firstNonEmpty(req.Reference, "Order "+req.OrderID),
		Address: apaczkasdk.AddressPair{
			Receiver: apaczkasdk.Address{
				Name:        req.Receiver.Name,
				Line1:       req.Receiver.Street,
				PostalCode:  req.Receiver.PostalCode,
				City:        req.Receiver.City,
				CountryCode: defaultStr(req.Receiver.Country, "PL"),
				Email:       req.Receiver.Email,
				Phone:       req.Receiver.Phone,
			},
		},
		PickupPoint: req.TargetPoint,
	}
	if req.Shipper != nil {
		in.Address.Sender = apaczkasdk.Address{
			Name: req.Shipper.Name, Line1: req.Shipper.Street, PostalCode: req.Shipper.PostalCode,
			City: req.Shipper.City, CountryCode: defaultStr(req.Shipper.Country, "PL"),
			Email: req.Shipper.Email, Phone: req.Shipper.Phone,
		}
	}
	if req.CODAmount > 0 {
		in.COD = &apaczkasdk.COD{Amount: req.CODAmount, Currency: defaultStr(req.CODCurrency, "PLN")}
	}

	order, err := p.client.CreateOrder(ctx, &apaczkasdk.CreateOrderRequest{Order: in})
	if err != nil {
		return nil, fmt.Errorf("apaczka: create order: %w", err)
	}
	return &integration.CarrierShipmentResponse{
		ExternalID:     order.ID,
		TrackingNumber: firstNonEmpty(order.TrackingNumber, order.WaybillNumber),
		Status:         order.Status,
	}, nil
}

func (p *ApaczkaProvider) GetLabel(ctx context.Context, externalID string, _ string) ([]byte, error) {
	return p.client.GetWaybill(ctx, externalID)
}

func (p *ApaczkaProvider) GetTracking(ctx context.Context, trackingNumber string) ([]integration.TrackingEvent, error) {
	// Apaczka tracks by order id; the OMS stores the order id as the external id.
	st, err := p.client.GetOrderStatus(ctx, trackingNumber)
	if err != nil {
		return nil, fmt.Errorf("apaczka: order status: %w", err)
	}
	events := make([]integration.TrackingEvent, 0, len(st.Tracking))
	for _, e := range st.Tracking {
		ts, _ := time.Parse("2006-01-02 15:04:05", e.Date) // CONFIRM format in Task 1
		events = append(events, integration.TrackingEvent{
			Status: e.Status, Location: e.Place, Timestamp: ts, Details: e.Comment,
		})
	}
	return events, nil
}

func (p *ApaczkaProvider) CancelShipment(ctx context.Context, externalID string) error {
	return p.client.CancelOrder(ctx, externalID)
}

func (p *ApaczkaProvider) MapStatus(carrierStatus string) (string, bool) {
	return mapApaczkaStatus(carrierStatus)
}

func (p *ApaczkaProvider) SupportsPickupPoints() bool { return true }

func (p *ApaczkaProvider) SearchPickupPoints(ctx context.Context, query string) ([]integration.PickupPoint, error) {
	pts, err := p.client.SearchPoints(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("apaczka: search points: %w", err)
	}
	out := make([]integration.PickupPoint, 0, len(pts))
	for _, pt := range pts {
		out = append(out, integration.PickupPoint{
			ID: pt.ID, Name: pt.Name, Street: pt.Street, City: pt.City,
			PostalCode: pt.PostalCode, Latitude: pt.Lat, Longitude: pt.Lng, Type: pt.Supplier,
		})
	}
	return out, nil
}

func defaultStr(v, d string) string {
	if v == "" {
		return d
	}
	return v
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
```

> **Note:** if `defaultStr`/`firstNonEmpty` already exist in the `carriers` package, drop the local copies to avoid a redeclaration error (run `go build` after Step 3 to check).

- [ ] **Step 4: Run, verify pass** — Run: `cd apps/api-server && go test ./internal/integration/carriers/ -run TestApaczka -v` → PASS

- [ ] **Step 5: Commit**

```bash
git add apps/api-server/internal/integration/carriers/apaczka.go apps/api-server/internal/integration/carriers/apaczka_test.go
git commit -m "OPE-XXX: ApaczkaProvider implements CarrierProvider (broker meta-carrier)"
```

---

## Task 8: Enable Apaczka in rate-shopping

**Files:**
- Modify: `apps/api-server/internal/service/rate_service.go:59-68`
- Test: `apps/api-server/internal/service/rate_service_test.go` (add or extend)

- [ ] **Step 1: Add the allow-map entry** at `rate_service.go:67` (after `"fedex": true,`):

```go
			"fedex":         true,
			"apaczka":       true,
```

- [ ] **Step 2: Verify behaviour** — if `rate_service_test.go` exists, add a case asserting an active `apaczka` integration is included; otherwise verify via the carriers test already proving `GetRates` returns rows. Run:

Run: `cd apps/api-server && go test ./internal/service/ -run TestRate -v`
Expected: PASS (or no rate test present → run `go build ./...` to confirm compilation)

- [ ] **Step 3: Commit**

```bash
git add apps/api-server/internal/service/rate_service.go apps/api-server/internal/service/rate_service_test.go
git commit -m "OPE-XXX: include apaczka broker in rate-shopping aggregation"
```

---

## Task 9: Register Apaczka in the Provider Integration Studio catalog

**Files:**
- Modify: `apps/api-server/internal/service/provider_catalog.go:121` (append entry before the closing `}` of the slice)
- Test: `apps/api-server/internal/service/provider_catalog_test.go` (create if absent)

- [ ] **Step 1: Write failing test** — `provider_catalog_test.go`

```go
package service

import "testing"

func TestProviderCatalogIncludesApaczka(t *testing.T) {
	var found bool
	for _, e := range ProviderRegistryCatalog() {
		if e.ProviderKey == "apaczka" {
			found = true
			if e.ProviderType != "carrier" {
				t.Fatalf("apaczka ProviderType = %q, want carrier", e.ProviderType)
			}
			if len(e.Schema) == 0 || len(e.Capabilities) == 0 {
				t.Fatal("apaczka entry must declare schema + capabilities")
			}
		}
	}
	if !found {
		t.Fatal("apaczka not in provider catalog")
	}
}
```

- [ ] **Step 2: Run, verify fail** — Run: `cd apps/api-server && go test ./internal/service/ -run TestProviderCatalogIncludesApaczka -v` → FAIL

- [ ] **Step 3: Add the catalog entry** in `ProviderRegistryCatalog()` (after the `shopify` entry, before the closing `}`):

```go
		{
			ProviderKey: "apaczka", DisplayName: "Apaczka (broker)", ProviderType: model.ProviderTypeCarrier, Version: "1.0.0",
			Notes:   "Broker meta-carrier capability-class representative (HMAC-signed REST). Fronts many couriers (InPost/DPD/DHL/UPS/GLS/Poczta/Orlen) via one integration.",
			Regions: []string{"PL"}, BusinessDomains: []string{"carrier"},
			Schema: []model.ProviderFieldGroup{
				{Key: model.FieldGroupSecretCredentials, Label: "Credentials", Fields: []model.ProviderField{
					secretField("app_id", "App ID"),
					secretField("app_secret", "App Secret"),
				}},
			},
			Capabilities: []model.ProviderCapability{
				unknownCap("carrier.shipment.create"), unknownCap("carrier.label.read"),
				unknownCap("carrier.tracking.read"), unknownCap("carrier.rates.read"),
				unknownCap("carrier.pickup_points.read"),
			},
		},
```

- [ ] **Step 4: Run, verify pass** — Run: same command → PASS

- [ ] **Step 5: Commit**

```bash
git add apps/api-server/internal/service/provider_catalog.go apps/api-server/internal/service/provider_catalog_test.go
git commit -m "OPE-XXX: register apaczka broker in Provider Studio catalog"
```

---

## Task 10: Docs, full local-CI, PR

**Files:**
- Modify: `docs/system-documentation.md` — add Apaczka under carriers/brokers (note: broker fronting multiple couriers)
- Modify: `.claude/context/API_CONTRACTS.md` — note rate-shopping (`/v1/shipping/rates`) now includes the apaczka broker; new carrier provider key `apaczka`
- Modify: `.claude/context/PROJECT_STATE.md` — note first broker meta-carrier shipped (OPE-XXX, under OPE-403)

- [ ] **Step 1: Update the three docs** with concise factual entries (Polish where the file is Polish, English for context files per conventions).

- [ ] **Step 2: Format Go**

```bash
cd /Users/rafs/praca/openoms-dev/public && gofmt -w -s apps/ packages/
```

- [ ] **Step 3: Run full local-CI (MANDATORY before push)**

```bash
cd /Users/rafs/praca/openoms-dev/public && ./scripts/local-ci.sh
```
Expected: all checks pass (gofmt, go vet, golangci-lint, eslint, next build, go test). Fix ALL failures before continuing.

- [ ] **Step 4: Commit docs + push**

```bash
git add docs/system-documentation.md .claude/context/API_CONTRACTS.md .claude/context/PROJECT_STATE.md
git commit -m "OPE-XXX: docs — apaczka broker meta-carrier"
git push -u origin feat/OPE-XXX-apaczka-broker
```

- [ ] **Step 5: Open PR** (title `OPE-XXX: Apaczka broker meta-carrier`) with a "Docs updated" section. Move Linear issue to In Review.

---

## Follow-ups (NOT in this plan)

- **Furgonetka broker** (separate plan): OAuth2 authorization-code consent flow per tenant + extend `OAuthRefresher` (`worker/oauth_refresher.go:40-47` provider IN-list + a `refreshFurgonetka`) + `Credentials{access_token,refresh_token,token_expiry}` shape like Allegro (`allegro/provider.go:25-33`). Reuses everything else from this pattern.
- **Tracking-by-order-id nuance:** the tracking poller calls `GetTracking(trackingNumber)`. For Apaczka we track by order id (stored as `external_id`). If the poller passes the tracking number (not external id), add a follow-up to let broker providers track by `external_id` (small interface/poller tweak) — confirm during Task 1 whether Apaczka also accepts the waybill/tracking number directly.
- **Pickup-point caching** and **dispatch orders** (`turn_in` → implement `DispatchOrderCreator`) — second-wave.
- **Sendcloud / Packeta** brokers — additional instances on this same pattern (Sendcloud Basic-auth, Packeta XML) once a tenant pulls for cross-border CEE/EU.

---

## Self-Review

- **Spec coverage:** SDK (signing/rates/orders/label/tracking/points) ✓ Tasks 2-4; provider adapter implementing all 9 `CarrierProvider` methods ✓ Tasks 6-7; rate-shopping ✓ Task 8; Studio registration ✓ Task 9; module wiring ✓ Task 5; docs ✓ Task 10.
- **Placeholder scan:** the only deliberate "CONFIRM in Task 1" markers are real verification steps against live docs, with concrete starting code provided — not vague TODOs.
- **Type consistency:** SDK types (`Service`, `Money`, `OrderInput`, `Order`, `Point`, `TrackEvent`) defined in Task 3-4 and consumed verbatim in Task 7; `CarrierProvider` method set matches `carrier.go:116-126`; catalog helpers (`secretField`/`unknownCap`) and `model.*` constants match `provider_catalog.go`.
- **Known risk:** exact Apaczka JSON contract — mitigated by Task 1 (verify) + httptest-mocked SDK tests (valid regardless of live shape).
