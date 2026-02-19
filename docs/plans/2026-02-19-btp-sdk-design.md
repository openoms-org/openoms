# BTP.pro SDK + Supplier Provider Integration

**Date:** 2026-02-19
**Branch:** feature/btp-sdk
**License:** MIT (packages/)

## Context

BTP.pro (Business Trading Platform) is a B2B platform used by Polish wholesalers/distributors.
It exposes a REST API with Basic Auth for product catalogue, inventory, order placement, invoices, and waybills.

API spec: https://ext.btp.pro/swagger/clientApi/swagger.json

## Package: `packages/btp-go-sdk/`

Full SDK covering all 18 BTP.pro Client API endpoints.

### File Structure

```
packages/btp-go-sdk/
  go.mod              # module github.com/openoms-org/openoms/packages/btp-go-sdk; go 1.25
  doc.go              # package btp
  client.go           # Client struct, NewClient, do/doMultipart helpers, Option funcs
  models.go           # All API types from swagger spec
  errors.go           # APIError + sentinel errors + Unwrap()
  inventory.go        # InventoryService — GET InventoryReportGet
  catalogue.go        # CatalogueService — GET ProductCatalogueGet
  orders.go           # OrderService — POST OrderCreate, POST OrderAttachmentsSet
  invoices.go         # InvoiceService — POST InvoicesGet, GET InvoiceImageGet, POST InvoiceNotifyConfigure
  waybills.go         # WaybillService — POST WaybillsGet, POST WaybillNotifyConfigure
  client_info.go      # ClientInfoService — GET ClientGet, DeliveryModesGet, CarriersGet, PaymentMethodsGet, CountryGovAreasGet
  files.go            # FileService — POST FileAdd, DELETE FilesDelete, DELETE FilesClear
  health.go           # GET HealthCheck
  client_test.go      # Auth + error handling + options tests
  inventory_test.go   # httptest tests per service
  catalogue_test.go
  orders_test.go
  invoices_test.go
  waybills_test.go
  client_info_test.go
  files_test.go
```

### Client Design

```go
type Client struct {
    httpClient *http.Client
    baseURL    string
    username   string
    password   string

    Inventory  *InventoryService
    Catalogue  *CatalogueService
    Orders     *OrderService
    Invoices   *InvoiceService
    Waybills   *WaybillService
    ClientInfo *ClientInfoService
    Files      *FileService
}

func NewClient(username, password string, opts ...Option) *Client
```

Options: WithBaseURL(url), WithHTTPClient(c), WithSandbox()

Authentication: HTTP Basic Auth header on every request.

### Endpoints → Methods

| BTP Endpoint | SDK Method | HTTP |
|---|---|---|
| InventoryReportGet | Inventory.GetReport(ctx, opts) | GET |
| ProductCatalogueGet | Catalogue.GetCatalogue(ctx, opts) | GET |
| ClientGet | ClientInfo.GetClient(ctx) | GET |
| DeliveryModesGet | ClientInfo.GetDeliveryModes(ctx) | GET |
| CarriersGet | ClientInfo.GetCarriers(ctx) | GET |
| PaymentMethodsGet | ClientInfo.GetPaymentMethods(ctx) | GET |
| CountryGovAreasGet | ClientInfo.GetCountryGovAreas(ctx, countryID) | GET |
| OrderCreate | Orders.Create(ctx, req) | POST |
| OrderAttachmentsSet | Orders.SetAttachments(ctx, req) | POST |
| FileAdd | Files.Add(ctx, file, description) | POST multipart |
| FilesDelete | Files.Delete(ctx, fileIDs) | DELETE |
| FilesClear | Files.Clear(ctx) | DELETE |
| InvoicesGet | Invoices.Get(ctx, req) | POST |
| InvoiceImageGet | Invoices.GetImage(ctx, opts) | GET |
| InvoiceNotifyConfigure | Invoices.ConfigureNotify(ctx, cfg) | POST |
| WaybillsGet | Waybills.Get(ctx, req) | POST |
| WaybillNotifyConfigure | Waybills.ConfigureNotify(ctx, cfg) | POST |
| HealthCheck | Client.HealthCheck(ctx) | GET |

### Error Handling

```go
var (
    ErrUnauthorized = errors.New("btp: unauthorized")
    ErrForbidden    = errors.New("btp: forbidden")
    ErrNotFound     = errors.New("btp: not found")
    ErrValidation   = errors.New("btp: validation error")
    ErrServerError  = errors.New("btp: server error")
)
```

BTP returns ApiResult with resultType (INFORMATION/WARNING/ERROR) and messages array.
APIError.Unwrap() maps HTTP status → sentinel.

## Provider: `apps/api-server/internal/integration/btp/`

### provider.go

Implements a `SupplierProvider` interface (new) that the SupplierService can call:

- `SyncCatalogue(ctx)` → calls Catalogue.GetCatalogue + upserts supplier_products
- `SyncInventory(ctx)` → calls Inventory.GetReport + updates stock/prices
- `CreateOrder(ctx, req)` → calls Orders.Create for dropship fulfillment

Registered via init() + factory pattern, matching existing carrier/marketplace providers.

### Credentials

```go
type BTPCredentials struct {
    Username string `json:"username"`
    Password string `json:"password"`
    BaseURL  string `json:"base_url,omitempty"`
}
```

Stored encrypted in integrations table like other providers.
