# Allegro Competitive Parity Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Close the critical feature gaps between OpenOMS and BaseLinker's Allegro integration — offer import, real-time stock sync, auto-relisting, messaging templates, and KSeF auto-send.

**Architecture:** Six independent features, each self-contained. Allegro Offer Import adds a new service + handler + frontend wizard. Stock sync completeness ensures all stock-changing paths trigger marketplace push. Auto-relisting adds a new automation trigger + action. Messaging templates add a new DB table + CRUD + automation action. KSeF auto-send hooks into invoice creation.

**Tech Stack:** Go 1.25, chi/v5, pgx/v5, allegro-go-sdk, ksef-go-sdk, Next.js 16, React 19, TypeScript, Tailwind v4, shadcn/ui

---

## Phase 1: Allegro Offer Import (P0 — onboarding blocker)

The #1 feature sellers need to migrate from BaseLinker. Without it, they'd have to manually recreate hundreds of offers.

### Task 1: Allegro SDK — Offer detail helper

**Files:**
- Modify: `packages/allegro-go-sdk/offers.go`
- Create: `packages/allegro-go-sdk/offers_test.go` (add tests)

**Context:** `Offers.List()` returns lightweight data (id, name, price, stock, status). For import we need full details (description, parameters, images, EAN). `Offers.Get()` exists but we need a batch-friendly helper that respects rate limits.

**Step 1: Write the failing test**

```go
func TestOfferService_ListAll_PaginatesAutomatically(t *testing.T) {
    // Mock server returns 2 pages of offers
    callCount := 0
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        callCount++
        offset := r.URL.Query().Get("offset")
        if offset == "" || offset == "0" {
            json.NewEncoder(w).Encode(map[string]any{
                "offers": []map[string]any{
                    {"id": "offer-1", "name": "Product 1"},
                    {"id": "offer-2", "name": "Product 2"},
                },
                "count":  3,
                "totalCount": 3,
            })
        } else {
            json.NewEncoder(w).Encode(map[string]any{
                "offers": []map[string]any{
                    {"id": "offer-3", "name": "Product 3"},
                },
                "count":  1,
                "totalCount": 3,
            })
        }
    }))
    defer srv.Close()
    client := NewClient("id", "secret", WithBaseURL(srv.URL), WithAccessToken("tok"))
    offers, err := client.Offers.ListAll(context.Background())
    assert.NoError(t, err)
    assert.Len(t, offers, 3)
    assert.Equal(t, "offer-1", offers[0].ID)
}
```

**Step 2: Run test to verify it fails**

Run: `cd packages/allegro-go-sdk && go test -run TestOfferService_ListAll -v`
Expected: FAIL — `ListAll` method doesn't exist

**Step 3: Implement ListAll**

Add to `offers.go`:

```go
// ListAll fetches all seller offers with automatic pagination.
// Respects rate limits via client.rateLimiter.
func (s *OfferService) ListAll(ctx context.Context) ([]Offer, error) {
    var all []Offer
    limit := 1000
    offset := 0
    for {
        params := &ListOffersParams{Limit: limit, Offset: offset}
        page, err := s.List(ctx, params)
        if err != nil {
            return all, fmt.Errorf("list offers offset=%d: %w", offset, err)
        }
        all = append(all, page.Offers...)
        if len(all) >= page.TotalCount || len(page.Offers) == 0 {
            break
        }
        offset += limit
    }
    return all, nil
}
```

**Step 4: Run test to verify it passes**

Run: `cd packages/allegro-go-sdk && go test -run TestOfferService_ListAll -v`
Expected: PASS

**Step 5: Commit**

```bash
git add packages/allegro-go-sdk/offers.go packages/allegro-go-sdk/offers_test.go
git commit -m "feat(allegro-sdk): add Offers.ListAll() with auto-pagination"
```

---

### Task 2: Allegro Import Service — core logic

**Files:**
- Create: `apps/api-server/internal/service/allegro_import_service.go`
- Create: `apps/api-server/internal/service/allegro_import_service_test.go`

**Context:** The service fetches all seller offers from Allegro, matches them to existing products by SKU/EAN, and creates Product + ProductListing records. It uses the existing `IntegrationService` to build the Allegro provider.

**Step 1: Write the model types**

Add to existing `model/product_listing.go` (or a new file if cleaner):

```go
// AllegroImportResult represents the outcome of an offer import job.
type AllegroImportResult struct {
    TotalOffers   int                    `json:"total_offers"`
    Created       int                    `json:"created"`
    Linked        int                    `json:"linked"`
    Skipped       int                    `json:"skipped"`
    Errors        int                    `json:"errors"`
    Details       []AllegroImportDetail  `json:"details,omitempty"`
}

type AllegroImportDetail struct {
    OfferID   string `json:"offer_id"`
    OfferName string `json:"offer_name"`
    Action    string `json:"action"` // "created", "linked", "skipped", "error"
    ProductID string `json:"product_id,omitempty"`
    Error     string `json:"error,omitempty"`
}
```

**Step 2: Write the failing test — service constructor**

```go
func TestAllegroImportService_New(t *testing.T) {
    svc := NewAllegroImportService(nil, nil, nil, nil, slog.Default())
    assert.NotNil(t, svc)
}
```

**Step 3: Implement the service struct + constructor**

```go
type AllegroImportService struct {
    integrationService *IntegrationService
    productRepo        repository.ProductRepo
    listingRepo        repository.ProductListingRepo
    pool               *pgxpool.Pool
    logger             *slog.Logger
}

func NewAllegroImportService(
    integrationService *IntegrationService,
    productRepo repository.ProductRepo,
    listingRepo repository.ProductListingRepo,
    pool *pgxpool.Pool,
    logger *slog.Logger,
) *AllegroImportService {
    return &AllegroImportService{
        integrationService: integrationService,
        productRepo:        productRepo,
        listingRepo:        listingRepo,
        pool:               pool,
        logger:             logger,
    }
}
```

**Step 4: Write test for offer-to-product mapping (pure function)**

```go
func TestMapAllegroOfferToProduct(t *testing.T) {
    offer := allegrosdk.Offer{
        ID:   "12345",
        Name: "Test Product",
        SellingMode: &allegrosdk.SellingMode{
            Price: allegrosdk.Amount{Amount: "29.99", Currency: "PLN"},
        },
        Stock: &allegrosdk.Stock{Available: 50},
        PrimaryImage: &allegrosdk.Image{URL: "https://img.allegro.pl/photo.jpg"},
    }

    product := mapAllegroOfferToProduct(offer, uuid.New())

    assert.Equal(t, "Test Product", product.Name)
    assert.Equal(t, 29.99, product.Price)
    assert.Equal(t, 50, product.StockQuantity)
    assert.Equal(t, "allegro", product.Source)
    assert.NotNil(t, product.ExternalID)
    assert.Equal(t, "12345", *product.ExternalID)
}
```

**Step 5: Implement mapAllegroOfferToProduct**

```go
func mapAllegroOfferToProduct(offer allegrosdk.Offer, tenantID uuid.UUID) model.Product {
    product := model.Product{
        ID:       uuid.New(),
        TenantID: tenantID,
        Name:     offer.Name,
        Source:   "allegro",
    }
    extID := offer.ID
    product.ExternalID = &extID

    if offer.SellingMode != nil {
        if p, err := strconv.ParseFloat(offer.SellingMode.Price.Amount, 64); err == nil {
            product.Price = p
        }
    }
    if offer.Stock != nil {
        product.StockQuantity = offer.Stock.Available
    }
    if offer.PrimaryImage != nil {
        product.ImageURL = &offer.PrimaryImage.URL
    }
    // Extract SKU from offer.External if available
    if offer.External != nil && offer.External.ID != "" {
        product.SKU = &offer.External.ID
    }
    return product
}
```

**Step 6: Write test for SKU/EAN matching logic**

```go
func TestAllegroImportService_matchProduct_BySKU(t *testing.T) {
    // Test that when a product with matching SKU exists, it's returned
    // (This is a unit test for the matching logic, not DB)
    existing := &model.Product{ID: uuid.New(), SKU: ptr("ABC123")}
    offer := allegrosdk.Offer{External: &allegrosdk.External{ID: "ABC123"}}

    matched := matchByIdentifier(existing, offer)
    assert.True(t, matched)
}

func TestAllegroImportService_matchProduct_NoMatch(t *testing.T) {
    offer := allegrosdk.Offer{External: &allegrosdk.External{ID: "XYZ999"}}
    existing := &model.Product{ID: uuid.New(), SKU: ptr("OTHER")}
    matched := matchByIdentifier(existing, offer)
    assert.False(t, matched)
}
```

**Step 7: Implement ImportOffers (main method)**

```go
// ImportOffers fetches all seller offers from Allegro and creates/links
// products + listings in OpenOMS. Idempotent — skips offers already linked.
func (s *AllegroImportService) ImportOffers(
    ctx context.Context,
    tenantID uuid.UUID,
    integrationID uuid.UUID,
) (*model.AllegroImportResult, error) {
    // 1. Build Allegro provider
    provider, err := s.integrationService.BuildAllegroProvider(ctx, tenantID, integrationID)
    if err != nil {
        return nil, fmt.Errorf("build provider: %w", err)
    }

    // 2. Fetch all offers
    client := provider.Client()
    offers, err := client.Offers.ListAll(ctx)
    if err != nil {
        return nil, fmt.Errorf("list offers: %w", err)
    }

    result := &model.AllegroImportResult{TotalOffers: len(offers)}

    // 3. Process each offer in a transaction
    for _, offer := range offers {
        detail := s.processOffer(ctx, tenantID, integrationID, offer)
        result.Details = append(result.Details, detail)
        switch detail.Action {
        case "created":
            result.Created++
        case "linked":
            result.Linked++
        case "skipped":
            result.Skipped++
        case "error":
            result.Errors++
        }
    }

    return result, nil
}

func (s *AllegroImportService) processOffer(
    ctx context.Context,
    tenantID, integrationID uuid.UUID,
    offer allegrosdk.Offer,
) model.AllegroImportDetail {
    detail := model.AllegroImportDetail{
        OfferID:   offer.ID,
        OfferName: offer.Name,
    }

    var resultProductID uuid.UUID
    err := database.WithTenant(ctx, s.pool, tenantID, func(ctx context.Context, tx pgx.Tx) error {
        // Check if listing already exists for this offer
        existing, _ := s.listingRepo.FindByExternalIDAndIntegration(ctx, tx, offer.ID, integrationID)
        if existing != nil {
            detail.Action = "skipped"
            detail.ProductID = existing.ProductID.String()
            return nil
        }

        // Try to match existing product by SKU
        var product *model.Product
        if offer.External != nil && offer.External.ID != "" {
            product, _ = s.productRepo.FindBySKU(ctx, tx, offer.External.ID)
        }

        if product == nil {
            // Create new product
            newProduct := mapAllegroOfferToProduct(offer, tenantID)
            if err := s.productRepo.Create(ctx, tx, &newProduct); err != nil {
                return fmt.Errorf("create product: %w", err)
            }
            product = &newProduct
            detail.Action = "created"
        } else {
            detail.Action = "linked"
        }

        resultProductID = product.ID

        // Create product listing
        listing := &model.ProductListing{
            ID:            uuid.New(),
            TenantID:      tenantID,
            ProductID:     product.ID,
            IntegrationID: integrationID,
            ExternalID:    &offer.ID,
            Status:        "active",
            SyncStatus:    "synced",
            StockSyncMode: "auto",
        }
        if offer.SellingMode != nil {
            if p, err := strconv.ParseFloat(offer.SellingMode.Price.Amount, 64); err == nil {
                listing.PriceOverride = &p
            }
        }
        return s.listingRepo.Create(ctx, tx, listing)
    })

    if err != nil {
        detail.Action = "error"
        detail.Error = err.Error()
    } else {
        detail.ProductID = resultProductID.String()
    }
    return detail
}
```

**Step 8: Run all tests**

Run: `cd apps/api-server && go test ./internal/service/ -run TestAllegroImport -v`
Expected: PASS

**Step 9: Commit**

```bash
git add apps/api-server/internal/service/allegro_import_service.go \
       apps/api-server/internal/service/allegro_import_service_test.go \
       apps/api-server/internal/model/product_listing.go
git commit -m "feat: Allegro offer import service with SKU matching"
```

---

### Task 3: Import handler + route

**Files:**
- Modify: `apps/api-server/internal/handler/allegro_handler.go`
- Modify: `apps/api-server/internal/router/router.go` (add route)
- Create: `apps/api-server/internal/handler/allegro_import_handler_test.go`

**Step 1: Write the failing test**

```go
func TestAllegroImportHandler_MissingIntegrationID(t *testing.T) {
    h := &AllegroHandler{} // minimal
    req := httptest.NewRequest("POST", "/v1/integrations/allegro/import-offers", nil)
    // No integration_id in URL
    rctx := chi.NewRouteContext()
    req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
    req = req.WithContext(middleware.WithTenantID(req.Context(), uuid.New()))
    rr := httptest.NewRecorder()
    h.ImportOffers(rr, req)
    assert.Equal(t, http.StatusBadRequest, rr.Code)
}
```

**Step 2: Implement handler**

```go
func (h *AllegroHandler) ImportOffers(w http.ResponseWriter, r *http.Request) {
    tenantID := middleware.TenantIDFromContext(r.Context())
    integrationIDStr := chi.URLParam(r, "integrationId")
    integrationID, err := uuid.Parse(integrationIDStr)
    if err != nil {
        writeValidationError(w, "integration_id", "invalid UUID")
        return
    }

    result, err := h.allegroImportService.ImportOffers(r.Context(), tenantID, integrationID)
    if err != nil {
        writeServerError(w, r, err)
        return
    }

    writeJSON(w, http.StatusOK, result)
}
```

**Step 3: Add route**

In `router.go`, inside the Allegro group:
```go
r.Post("/integrations/allegro/{integrationId}/import-offers", allegroHandler.ImportOffers)
```

**Step 4: Run tests, commit**

```bash
git add apps/api-server/internal/handler/allegro_handler.go \
       apps/api-server/internal/handler/allegro_import_handler_test.go \
       apps/api-server/internal/router/router.go
git commit -m "feat: POST /integrations/allegro/:id/import-offers endpoint"
```

---

### Task 4: Wire up in main.go

**Files:**
- Modify: `apps/api-server/cmd/server/main.go`

**Step 1: Add AllegroImportService to DI**

Find where other services are created and add:
```go
allegroImportService := service.NewAllegroImportService(
    integrationService, productRepo, listingRepo, pool, logger,
)
```

Pass it to the AllegroHandler constructor (or setter).

**Step 2: Run full test suite**

Run: `cd apps/api-server && go test ./... -count=1`
Expected: All tests pass

**Step 3: Commit**

```bash
git add apps/api-server/cmd/server/main.go
git commit -m "feat: wire AllegroImportService into DI"
```

---

### Task 5: Frontend — Import wizard page

**Files:**
- Create: `apps/dashboard/src/hooks/use-allegro-import.ts`
- Create: `apps/dashboard/src/app/(dashboard)/integrations/allegro/import/page.tsx`
- Modify: `apps/dashboard/src/app/(dashboard)/integrations/allegro/page.tsx` (add import button)

**Step 1: Create the hook**

```typescript
import { useMutation } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";

export interface AllegroImportResult {
  total_offers: number;
  created: number;
  linked: number;
  skipped: number;
  errors: number;
  details?: {
    offer_id: string;
    offer_name: string;
    action: "created" | "linked" | "skipped" | "error";
    product_id?: string;
    error?: string;
  }[];
}

export function useAllegroImportOffers(integrationId: string) {
  return useMutation({
    mutationFn: () =>
      apiClient<AllegroImportResult>(
        `/v1/integrations/allegro/${integrationId}/import-offers`,
        { method: "POST" }
      ),
  });
}
```

**Step 2: Create the import page**

A simple page with:
- "Import offers from Allegro" button
- Progress indicator while importing
- Results table: created / linked / skipped / errors
- Details expandable section
- Link back to products page

**Step 3: Add "Import from Allegro" button to integrations page**

In the Allegro integration card, add a button that links to `/integrations/allegro/import`.

**Step 4: Commit**

```bash
git add apps/dashboard/src/hooks/use-allegro-import.ts \
       apps/dashboard/src/app/\(dashboard\)/integrations/allegro/import/page.tsx \
       apps/dashboard/src/app/\(dashboard\)/integrations/allegro/page.tsx
git commit -m "feat: Allegro offer import wizard UI"
```

---

## Phase 2: Stock Sync Completeness + Auto-Relisting

### Task 6: Audit all stock-changing paths

**Files:**
- Modify: `apps/api-server/internal/service/product_service.go`
- Modify: `apps/api-server/internal/service/order_service.go`

**Context:** Ensure every code path that changes `stock_quantity` calls `StockSyncService.OnStockChange()`. Currently, some paths may update stock directly without triggering marketplace sync.

**Step 1: Search for all stock-changing SQL**

Look for: `stock_quantity` in UPDATE statements across all services/repos.
Paths to verify call `OnStockChange()`:
- Product update (manual stock change)
- Order creation (stock reservation)
- Order cancellation (stock release)
- Return processing (stock return)
- Warehouse document confirmation (PZ/WZ)
- Supplier sync (stock update from feed)
- Import (CSV/Excel)

**Step 2: Add OnStockChange calls to any missing paths**

For each missing path, add after the stock update:
```go
if stockChanged {
    go s.stockSyncService.OnStockChange(ctx, tenantID, productID, triggerType)
}
```

**Step 3: Write tests verifying sync is triggered**

**Step 4: Commit**

```bash
git commit -m "fix: ensure all stock-changing paths trigger marketplace sync"
```

---

### Task 7: Auto-relisting — new automation trigger + action

**Files:**
- Modify: `apps/api-server/internal/model/automation.go` (add trigger + action)
- Modify: `apps/api-server/internal/automation/actions.go` (implement activate_listing)
- Modify: `apps/api-server/internal/service/stock_sync_service.go` (dispatch trigger)
- Create: test files

**Step 1: Add new trigger event**

In `model/automation.go`, add to `ValidTriggerEvents`:
```go
"product.stock_restored": true, // stock was 0, now > 0
```

**Step 2: Add new action type**

In `automation/actions.go`, add `activate_listing` action:
```go
case "activate_listing":
    return s.executeActivateListing(ctx, tenantID, event)
```

Implementation:
- Find all inactive listings for the product
- Call `provider.Activate(ctx, externalID)` for each
- Update listing status to "active"

**Step 3: Dispatch trigger in OnStockChange**

In `stock_sync_service.go`, within `OnStockChange()`:
```go
// If previous stock was 0 and new stock > 0, fire product.stock_restored
if previousQty == 0 && newQty > 0 {
    s.automationEngine.ProcessEvent(ctx, AutomationEvent{
        Type:     "product.stock_restored",
        TenantID: tenantID,
        Data:     map[string]any{"product_id": productID, "new_quantity": newQty},
    })
}
```

**Step 4: Tests + commit**

```bash
git commit -m "feat: auto-relisting — product.stock_restored trigger + activate_listing action"
```

---

## Phase 3: Messaging Templates

### Task 8: Message templates — DB migration + model

**Files:**
- Create: `apps/api-server/migrations/000006_message_templates.up.sql`
- Create: `apps/api-server/migrations/000006_message_templates.down.sql`
- Modify: `apps/api-server/internal/model/message_template.go`

**Migration:**
```sql
CREATE TABLE IF NOT EXISTS message_templates (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name text NOT NULL,
    channel text NOT NULL DEFAULT 'allegro', -- 'allegro', 'email', 'sms'
    subject text,
    body text NOT NULL,
    variables text[] DEFAULT '{}', -- available variables: {order_id}, {buyer_name}, etc.
    is_autoresponder boolean DEFAULT false,
    trigger_event text, -- e.g., 'message.received'
    enabled boolean DEFAULT true,
    created_at timestamptz DEFAULT NOW(),
    updated_at timestamptz DEFAULT NOW()
);
ALTER TABLE message_templates ENABLE ROW LEVEL SECURITY;
CREATE POLICY message_templates_tenant ON message_templates
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid);
```

**Model:**
```go
type MessageTemplate struct {
    ID              uuid.UUID `json:"id"`
    TenantID        uuid.UUID `json:"tenant_id"`
    Name            string    `json:"name"`
    Channel         string    `json:"channel"`
    Subject         *string   `json:"subject,omitempty"`
    Body            string    `json:"body"`
    Variables       []string  `json:"variables"`
    IsAutoresponder bool      `json:"is_autoresponder"`
    TriggerEvent    *string   `json:"trigger_event,omitempty"`
    Enabled         bool      `json:"enabled"`
    CreatedAt       time.Time `json:"created_at"`
    UpdatedAt       time.Time `json:"updated_at"`
}
```

**Step: Commit**
```bash
git commit -m "feat: message_templates migration + model"
```

---

### Task 9: Message templates — repo + service + handler + routes

**Files:**
- Create: `apps/api-server/internal/repository/message_template_repository.go`
- Create: `apps/api-server/internal/service/message_template_service.go`
- Create: `apps/api-server/internal/handler/message_template_handler.go`
- Modify: `apps/api-server/internal/router/router.go`

Standard CRUD pattern (same as other resources):
- `GET /v1/message-templates` — list
- `POST /v1/message-templates` — create
- `GET /v1/message-templates/:id` — get
- `PUT /v1/message-templates/:id` — update
- `DELETE /v1/message-templates/:id` — delete

Variable substitution helper:
```go
func substituteVariables(template string, data map[string]string) string {
    result := template
    for key, value := range data {
        result = strings.ReplaceAll(result, "{"+key+"}", value)
    }
    return result
}
```

**Commit:**
```bash
git commit -m "feat: message templates CRUD — repo, service, handler, routes"
```

---

### Task 10: Automation action — send_marketplace_message

**Files:**
- Modify: `apps/api-server/internal/automation/actions.go`
- Modify: `apps/api-server/internal/model/automation.go` (add to valid actions)

Add new action type `send_marketplace_message`:
```go
case "send_marketplace_message":
    return s.executeSendMarketplaceMessage(ctx, tenantID, event, action.Config)
```

Config:
```json
{
    "type": "send_marketplace_message",
    "config": {
        "template_id": "uuid-of-template"
    }
}
```

Implementation:
1. Load template by ID
2. Extract order data for variable substitution
3. Find Allegro thread for the order (by external_id)
4. Call `provider.Messages.SendMessage(ctx, threadID, message)`

**Commit:**
```bash
git commit -m "feat: send_marketplace_message automation action"
```

---

## Phase 4: KSeF Auto-Send

### Task 11: KSeF auto-send setting + hook

**Files:**
- Modify: `apps/api-server/internal/service/ksef_service.go`
- Modify: `apps/api-server/internal/service/invoice_service.go`
- Modify: `apps/api-server/internal/model/settings.go` (add auto_send_ksef field)

**Context:** KSeF becomes mandatory April 1, 2026. Currently, KSeF send is manual (user clicks "Send to KSeF" per invoice). Auto-send hooks into invoice creation.

**Step 1: Add setting**

In tenant settings (or KSeF settings):
```go
type KSeFSettings struct {
    Environment string `json:"environment"` // "production" or "test"
    Token       string `json:"token"`
    NIP         string `json:"nip"`
    AutoSend    bool   `json:"auto_send"` // NEW
}
```

**Step 2: Hook into invoice creation**

In `invoice_service.go`, after successful invoice creation:
```go
// Auto-send to KSeF if enabled
if ksefSettings != nil && ksefSettings.AutoSend {
    go func() {
        if err := s.ksefService.SendToKSeF(ctx, tenantID, invoice.ID, actorID, ip); err != nil {
            s.logger.Error("auto ksef send failed", "invoice_id", invoice.ID, "error", err)
        }
    }()
}
```

**Step 3: Batch retry in KSeFStatusWorker**

Modify `KSeFStatusWorker` to also pick up invoices with `ksef_status = 'error'` and retry (with exponential backoff, max 3 retries).

**Step 4: Tests + commit**

```bash
git commit -m "feat: KSeF auto-send on invoice creation + retry"
```

---

## Phase 5: Frontend Pages

### Task 12: Messaging templates UI

**Files:**
- Create: `apps/dashboard/src/hooks/use-message-templates.ts`
- Create: `apps/dashboard/src/app/(dashboard)/settings/message-templates/page.tsx`

Standard CRUD page with DataTable, create/edit dialog, variable picker.

### Task 13: KSeF auto-send toggle in settings

**Files:**
- Modify: `apps/dashboard/src/app/(dashboard)/settings/ksef/page.tsx`

Add toggle switch for "Auto-send to KSeF" in KSeF settings page.

### Task 14: Stock sync dashboard improvements

**Files:**
- Modify: `apps/dashboard/src/app/(dashboard)/settings/stock-sync/page.tsx`

Show real-time sync status, last push timestamps, error counts per channel.

---

## Execution Order & Dependencies

```
Task 1 (SDK ListAll)
  └→ Task 2 (Import Service) — depends on Task 1
       └→ Task 3 (Handler + Route) — depends on Task 2
            └→ Task 4 (Wire main.go) — depends on Task 3
                 └→ Task 5 (Frontend Import) — depends on Task 4

Task 6 (Stock sync audit) — independent
Task 7 (Auto-relisting) — depends on Task 6

Task 8 (Templates migration) — independent
  └→ Task 9 (Templates CRUD) — depends on Task 8
       └→ Task 10 (Automation action) — depends on Task 9

Task 11 (KSeF auto-send) — independent

Task 12 (Templates UI) — depends on Task 9
Task 13 (KSeF UI) — depends on Task 11
Task 14 (Stock sync UI) — depends on Task 6
```

**Parallel execution possible:**
- Tasks 1-5 (Allegro Import) can run in parallel with Tasks 8-10 (Messaging)
- Task 6-7 (Stock) can run in parallel with Task 11 (KSeF)
- Frontend tasks (12-14) depend on their backend tasks

## Testing Strategy

- Every service method: unit test with mock repos
- Every handler: httptest with validation checks
- Pure mapping functions: table-driven tests
- Integration service: test with mock Allegro API (httptest.NewServer)
- Frontend: manual testing + Playwright for critical flows

## Risks & Mitigations

| Risk | Mitigation |
|------|-----------|
| Allegro rate limit (9000/min) during import | SDK rate limiter (150/min), pagination, async processing |
| Large seller (>10k offers) — slow import | Background job with progress reporting, not blocking HTTP |
| SKU collision between products | Match by SKU within tenant scope only |
| KSeF API downtime | Retry with backoff, queue failed sends |
| Allegro variant API deprecation (April 14, 2026) | Already using catalog-based variants |
