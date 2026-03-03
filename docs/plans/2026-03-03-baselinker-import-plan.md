# BaseLinker CSV Import — Implementation Plan

> **For Claude:** Implement tasks sequentially. After each task: run tests, verify build, commit. Branch: `feat/baselinker-import` (create from main).

**Goal:** Enable one-time data migration from BaseLinker via CSV upload — orders (with row-per-item grouping), products (with variants and images), and customers (with dedup by email).

**Architecture:** Extend existing CSV import infrastructure with BL-specific parsers. New customer import endpoint pair (preview + import). BL order parser groups rows by order_id into orders with items[]. BL product parser handles variant grouping and image URL extraction. Separate image re-download background job uploads to S3.

**Tech Stack:** Go 1.25, pgx/v5, chi/v5, Next.js 16, React 19, TypeScript, Tailwind v4, React Query, Playwright

**Conventions:**
- No Co-Authored-By in commits
- Go tests: `httptest` + `testify/assert`
- Frontend: TypeScript, Tailwind v4, shadcn/ui, React Query
- Code/config in English, user-facing Polish text OK
- All DB operations via `database.WithTenant(ctx, pool, tenantID, callback)`

---

## Task 1: Customer Import — Backend Service

**Context:** Order and product import exist. Customer import does not. Follow the exact pattern from `product_import_service.go`.

**Files:**
- Create: `apps/api-server/internal/service/customer_import_service.go`
- Create: `apps/api-server/internal/service/customer_import_service_test.go`

### 1a: Write CustomerImportService with PreviewCSV

```go
// customer_import_service.go
package service

import (
	"encoding/csv"
	"fmt"
	"io"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
	"github.com/openoms-org/openoms/apps/api-server/pkg/database"
)

type CustomerImportService struct {
	customerRepo repository.CustomerRepo
	auditRepo    repository.AuditRepo
	pool         *pgxpool.Pool
}

func NewCustomerImportService(
	customerRepo repository.CustomerRepo,
	auditRepo repository.AuditRepo,
	pool *pgxpool.Pool,
) *CustomerImportService {
	return &CustomerImportService{
		customerRepo: customerRepo,
		auditRepo:    auditRepo,
		pool:         pool,
	}
}

type CustomerImportPreview struct {
	Headers     []string                 `json:"headers"`
	TotalRows   int                      `json:"total_rows"`
	SampleRows  []model.ImportPreviewRow `json:"sample_rows"`
	NewCount    int                      `json:"new_count"`
	UpdateCount int                      `json:"update_count"`
	Mappings    []model.ImportColumnMapping `json:"mappings,omitempty"`
}

type CustomerImportResult struct {
	Created int                 `json:"created"`
	Updated int                 `json:"updated"`
	Skipped int                 `json:"skipped"`
	Errors  []model.ImportError `json:"errors"`
}

// customerFieldAliases maps known CSV header names → canonical customer field.
var customerFieldAliases = map[string]string{
	"name":           "name",
	"customer_name":  "name",
	"customer name":  "name",
	"buyer_name":     "name",
	"delivery_fullname": "name",
	"email":          "email",
	"customer_email": "email",
	"customer email": "email",
	"buyer_email":    "email",
	"phone":          "phone",
	"customer_phone": "phone",
	"customer phone": "phone",
	"buyer_phone":    "phone",
	"company_name":   "company_name",
	"company":        "company_name",
	"company name":   "company_name",
	"invoice_company": "company_name",
	"nip":            "nip",
	"invoice_nip":    "nip",
	"tax_id":         "nip",
	"tags":           "tags",
	"notes":          "notes",
}

var validCustomerFields = map[string]bool{
	"name": true, "email": true, "phone": true,
	"company_name": true, "nip": true, "tags": true, "notes": true,
}

func (s *CustomerImportService) PreviewCSV(ctx context.Context, tenantID uuid.UUID, file io.Reader) (*CustomerImportPreview, error) {
	reader := csv.NewReader(file)
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true

	headers, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("read CSV headers: %w", err)
	}
	if len(headers) > 0 {
		headers[0] = stripBOM(headers[0])
	}

	// Auto-detect mappings
	var mappings []model.ImportColumnMapping
	for _, h := range headers {
		lower := strings.ToLower(strings.TrimSpace(h))
		if field, ok := customerFieldAliases[lower]; ok {
			mappings = append(mappings, model.ImportColumnMapping{
				CSVColumn:  h,
				OrderField: field, // reuse OrderField JSON tag for compat
			})
		}
	}

	// Check "name" column is present
	hasName := false
	for _, m := range mappings {
		if m.OrderField == "name" {
			hasName = true
			break
		}
	}
	if !hasName {
		return nil, fmt.Errorf("CSV must contain a name column (name, customer_name, buyer_name)")
	}

	// Read all rows for counting
	var sampleRows []model.ImportPreviewRow
	headerIdx := make(map[string]int)
	for i, h := range headers {
		headerIdx[strings.ToLower(strings.TrimSpace(h))] = i
	}

	totalRows := 0
	newCount, updateCount := 0, 0

	err = database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		for {
			row, err := reader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				continue
			}
			totalRows++

			if len(sampleRows) < 10 {
				data := make(map[string]any)
				for i, val := range row {
					if i < len(headers) {
						data[headers[i]] = val
					}
				}
				sampleRows = append(sampleRows, model.ImportPreviewRow{
					Row:  totalRows,
					Data: data,
				})
			}

			// Count new vs update by email
			email := extractField(row, headerIdx, "email", "customer_email", "buyer_email")
			if email != "" {
				existing, _ := s.customerRepo.FindByEmail(ctx, tx, email)
				if existing != nil {
					updateCount++
				} else {
					newCount++
				}
			} else {
				newCount++
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &CustomerImportPreview{
		Headers:     headers,
		TotalRows:   totalRows,
		SampleRows:  sampleRows,
		NewCount:    newCount,
		UpdateCount: updateCount,
		Mappings:    mappings,
	}, nil
}

// extractField tries multiple header aliases and returns the first non-empty value.
func extractField(row []string, headerIdx map[string]int, aliases ...string) string {
	for _, alias := range aliases {
		if idx, ok := headerIdx[strings.ToLower(alias)]; ok && idx < len(row) {
			val := strings.TrimSpace(row[idx])
			if val != "" {
				return val
			}
		}
	}
	return ""
}
```

### 1b: Write ImportCSV method

```go
func (s *CustomerImportService) ImportCSV(ctx context.Context, tenantID uuid.UUID, file io.Reader, userID uuid.UUID, ip string) (*CustomerImportResult, error) {
	reader := csv.NewReader(file)
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true

	headers, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("read CSV headers: %w", err)
	}
	if len(headers) > 0 {
		headers[0] = stripBOM(headers[0])
	}

	headerIdx := make(map[string]int)
	for i, h := range headers {
		headerIdx[strings.ToLower(strings.TrimSpace(h))] = i
	}

	result := &CustomerImportResult{}

	err = database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		rowNum := 0
		for {
			row, err := reader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				continue
			}
			rowNum++

			importErr := s.importCustomerRow(ctx, tx, tenantID, row, headerIdx, rowNum, result)
			if importErr != nil {
				result.Errors = append(result.Errors, *importErr)
			}
		}

		// Audit log
		s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     userID,
			Action:     "customer.import",
			EntityType: "customer",
			Changes: map[string]string{
				"created": fmt.Sprintf("%d", result.Created),
				"updated": fmt.Sprintf("%d", result.Updated),
				"skipped": fmt.Sprintf("%d", result.Skipped),
				"errors":  fmt.Sprintf("%d", len(result.Errors)),
			},
			IPAddress: ip,
		})

		return nil
	})

	return result, err
}

func (s *CustomerImportService) importCustomerRow(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID, row []string, headerIdx map[string]int, rowNum int, result *CustomerImportResult) *model.ImportError {
	name := extractField(row, headerIdx, "name", "customer_name", "customer name", "buyer_name", "delivery_fullname")
	if name == "" {
		result.Skipped++
		return &model.ImportError{Row: rowNum, Field: "name", Message: "name is required"}
	}

	email := extractField(row, headerIdx, "email", "customer_email", "buyer_email")
	phone := extractField(row, headerIdx, "phone", "customer_phone", "buyer_phone")
	companyName := extractField(row, headerIdx, "company_name", "company", "invoice_company")
	nip := extractField(row, headerIdx, "nip", "invoice_nip", "tax_id")
	tags := extractField(row, headerIdx, "tags")
	notes := extractField(row, headerIdx, "notes")

	// Dedup by email
	if email != "" {
		existing, _ := s.customerRepo.FindByEmail(ctx, tx, email)
		if existing != nil {
			// Update existing customer
			req := model.UpdateCustomerRequest{Name: &name}
			if phone != "" {
				req.Phone = &phone
			}
			if companyName != "" {
				req.CompanyName = &companyName
			}
			if nip != "" {
				req.NIP = &nip
			}
			if err := s.customerRepo.Update(ctx, tx, existing.ID, req); err != nil {
				return &model.ImportError{Row: rowNum, Message: fmt.Sprintf("update customer: %v", err)}
			}
			result.Updated++
			return nil
		}
	}

	// Create new customer
	customer := &model.Customer{
		ID:       uuid.New(),
		TenantID: tenantID,
		Name:     name,
	}
	if email != "" {
		customer.Email = &email
	}
	if phone != "" {
		customer.Phone = &phone
	}
	if companyName != "" {
		customer.CompanyName = &companyName
	}
	if nip != "" {
		customer.NIP = &nip
	}
	if notes != "" {
		customer.Notes = &notes
	}
	if tags != "" {
		customer.Tags = strings.Split(tags, ",")
		for i := range customer.Tags {
			customer.Tags[i] = strings.TrimSpace(customer.Tags[i])
		}
	}

	if err := s.customerRepo.Create(ctx, tx, customer); err != nil {
		return &model.ImportError{Row: rowNum, Message: fmt.Sprintf("create customer: %v", err)}
	}
	result.Created++
	return nil
}
```

### 1c: Write tests

Test file: `customer_import_service_test.go` with tests for:
- `TestCustomerImportService_PreviewCSV` — auto-detects headers, counts new/update
- `TestCustomerImportService_ImportCSV` — creates new, updates by email dedup
- `TestCustomerImportService_ImportCSV_SkipsEmptyName` — rows without name are skipped
- `TestCustomerImportService_ImportCSV_BaseLinkerAliases` — BL header names (`buyer_name`, `buyer_email`, etc.) are recognized

**Verify:**
```bash
cd apps/api-server && go build ./... && go test ./internal/service/ -run TestCustomerImport -count=1 2>&1 | tail -10
```

**Commit:** `feat(import): add customer CSV import service with BL aliases`

---

## Task 2: Customer Import — Backend Handler + Routes

**Context:** Follow the exact pattern from `import_handler.go` (order import).

**Files:**
- Modify: `apps/api-server/internal/handler/customer_handler.go` — add ImportPreview + ImportCSV methods
- Modify: `apps/api-server/internal/router/router.go` — add customer import routes
- Modify: `apps/api-server/cmd/server/main.go` — wire CustomerImportService into handler

### 2a: Add handler methods

In `customer_handler.go`, update constructor to accept `*service.CustomerImportService` and add:

```go
func (h *CustomerHandler) ImportPreview(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "invalid multipart form")
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing file field")
		return
	}
	defer file.Close()

	preview, err := h.customerImportService.PreviewCSV(r.Context(), tenantID, file)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, preview)
}

func (h *CustomerHandler) ImportCSV(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	userID := middleware.UserIDFromContext(r.Context())

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "invalid multipart form")
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing file field")
		return
	}
	defer file.Close()

	result, err := h.customerImportService.ImportCSV(r.Context(), tenantID, file, userID, clientIP(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}
```

### 2b: Add routes

In `router.go`, inside the customers route group, add:
```go
r.Post("/import/preview", deps.Customer.ImportPreview)
r.Post("/import", deps.Customer.ImportCSV)
```

### 2c: Wire service in main.go

Add `CustomerImportService` creation and pass to `CustomerHandler`.

**Verify:**
```bash
cd apps/api-server && go build ./... 2>&1 | tail -5
```

**Commit:** `feat(import): add customer import handler and routes`

---

## Task 3: BaseLinker Order Parser — Row Grouping

**Context:** BL exports one row per order item. Current order import creates one order per row. Need a new parser that groups rows by `order_id` → single order with `items[]`.

**Files:**
- Create: `apps/api-server/internal/service/baselinker_import_service.go`
- Create: `apps/api-server/internal/service/baselinker_import_service_test.go`

### 3a: Implement BL order parser

The key difference from regular import: **grouping**. Algorithm:
1. Read all CSV rows
2. Group by `order_id` column
3. For each group: first row provides customer data + order metadata, all rows provide items
4. Create one order per group with aggregated `items[]` and computed `total_amount`

```go
// baselinker_import_service.go
package service

// BaseLinkerImportService handles CSV imports in BaseLinker export format.
type BaseLinkerImportService struct {
	orderRepo    repository.OrderRepo
	customerRepo repository.CustomerRepo
	auditRepo    repository.AuditRepo
	tenantRepo   repository.TenantRepo
	pool         *pgxpool.Pool
}

// blOrderGroup collects all rows belonging to one BL order.
type blOrderGroup struct {
	OrderID       string
	Rows          [][]string // all CSV rows for this order_id
	FirstRowIndex int        // 1-based row number of first occurrence
}

func (s *BaseLinkerImportService) ImportOrders(ctx context.Context, tenantID uuid.UUID, file io.Reader, userID uuid.UUID, ip string) (*model.ImportResult, error) {
	// 1. Parse CSV, strip BOM
	// 2. Find order_id column index (aliases: order_id, order_number, bl_order_id)
	// 3. Group rows by order_id value → map[string]*blOrderGroup
	// 4. For each group:
	//    a. Extract customer data from first row (delivery_fullname, buyer_email, buyer_phone)
	//    b. Extract shipping address from first row (delivery_address, delivery_city, delivery_postcode, delivery_country)
	//    c. Extract billing address from first row (invoice_*)
	//    d. For each row in group: extract item (product_name, product_sku, product_quantity, product_price_brutto)
	//    e. Compute total_amount = sum(qty * price) for all items
	//    f. Parse date from date_add column
	//    g. Map status: order_status_name → tenant status config
	//    h. Check external_id dedup (order_id as external_id)
	//    i. Create Order with items[] as JSON
	// 5. Optionally extract unique customers → auto-create via customerRepo
	// 6. Audit log
}
```

### 3b: BL address extraction helper

```go
func extractBLShippingAddress(row []string, headerIdx map[string]int) json.RawMessage {
	addr := map[string]string{}
	if v := extractField(row, headerIdx, "delivery_fullname"); v != "" {
		addr["name"] = v
	}
	if v := extractField(row, headerIdx, "delivery_address"); v != "" {
		addr["street"] = v
	}
	if v := extractField(row, headerIdx, "delivery_city"); v != "" {
		addr["city"] = v
	}
	if v := extractField(row, headerIdx, "delivery_postcode", "delivery_zipcode"); v != "" {
		addr["postal_code"] = v
	}
	if v := extractField(row, headerIdx, "delivery_country_code", "delivery_country"); v != "" {
		addr["country"] = v
	}
	if len(addr) == 0 {
		return nil
	}
	b, _ := json.Marshal(addr)
	return b
}
```

### 3c: Tests

- `TestBLImport_GroupsByOrderID` — 6 rows (2 orders × 3 items) → 2 orders created, each with 3 items
- `TestBLImport_AggregatesItems` — items have correct name, sku, qty, price
- `TestBLImport_ExtractsCustomerFromFirstRow` — customer_name, email, phone from first row of group
- `TestBLImport_ExtractsAddresses` — shipping/billing address JSONB populated
- `TestBLImport_ComputesTotalAmount` — total = sum(qty × price)
- `TestBLImport_DeduplicatesByOrderID` — existing external_id → skip
- `TestBLImport_ExtractsCustomers` — unique customers created/updated

**Verify:**
```bash
cd apps/api-server && go test ./internal/service/ -run TestBLImport -count=1 -v 2>&1 | tail -20
```

**Commit:** `feat(import): add BaseLinker order parser with row grouping`

---

## Task 4: BaseLinker Order Import — Handler + Routes

**Files:**
- Modify: `apps/api-server/internal/handler/import_handler.go` — add BL-specific handlers
- Modify: `apps/api-server/internal/router/router.go` — add BL import routes

### 4a: Add handlers

```go
func (h *ImportHandler) BaseLinkerPreview(w http.ResponseWriter, r *http.Request) {
	// Same pattern as Preview() but calls baseLinkerImportService.PreviewOrders()
}

func (h *ImportHandler) BaseLinkerImport(w http.ResponseWriter, r *http.Request) {
	// Same pattern as Import() but calls baseLinkerImportService.ImportOrders()
	// Also auto-imports customers if checkbox "import_customers" is set
}
```

### 4b: Add routes

```go
r.Route("/v1/import/baselinker", func(r chi.Router) {
	r.Post("/orders/preview", deps.Import.BaseLinkerPreview)
	r.Post("/orders", deps.Import.BaseLinkerImport)
})
```

**Verify:**
```bash
cd apps/api-server && go build ./... 2>&1 | tail -5
```

**Commit:** `feat(import): add BaseLinker order import endpoints`

---

## Task 5: BaseLinker Product Import — Aliases + Variants

**Context:** Extend existing `ProductImportService` with BL column aliases and variant grouping.

**Files:**
- Modify: `apps/api-server/internal/service/product_import_service.go` — add BL aliases, variant grouping mode
- Create: `apps/api-server/internal/service/product_import_bl_test.go`

### 5a: Add BL aliases to product auto-detection

Extend the header-to-field mapping in `product_import_service.go`:

```go
// Add to existing alias map or create separate BL alias map
var blProductAliases = map[string]string{
	"product_id":          "external_id",
	"name":                "name",
	"product_name":        "name",
	"sku":                 "sku",
	"product_sku":         "sku",
	"ean":                 "ean",
	"product_ean":         "ean",
	"price_brutto":        "price",
	"product_price_brutto": "price",
	"quantity":            "stock_quantity",
	"stock":               "stock_quantity",
	"product_quantity":     "stock_quantity",
	"description":         "short_description",
	"description_extra":   "long_description",
	"category":            "category",
	"category_name":       "category",
	"weight":              "weight",
	"image_url":           "images",
	"images":              "images",
	"variant_id":          "variant_id",
	"variant_name":        "variant_name",
}
```

### 5b: Add variant grouping mode

New method `ImportBLProductsCSV` that:
1. Reads all rows
2. Identifies rows with `variant_id` → these are variants of a parent product (`product_id`)
3. Groups: parent rows (no variant_id) create products, child rows create variants
4. For each parent product:
   - Create/update product via existing upsert-by-SKU logic
   - Set `has_variants = true` if variants exist
5. For each variant row:
   - Find parent product by `product_id` (external_id)
   - Create `ProductVariant` with: name from `variant_name`, SKU from variant row, price from variant row
   - Attributes from any extra columns (color, size, etc.)
6. Image URLs: parse semicolon-separated list → `images` JSONB array

### 5c: Tests

- `TestBLProductImport_Aliases` — BL headers auto-detected
- `TestBLProductImport_VariantGrouping` — parent + children correctly created
- `TestBLProductImport_ImageURLs` — semicolon-separated URLs → JSON array
- `TestBLProductImport_CategoryCreation` — new categories auto-created

**Verify:**
```bash
cd apps/api-server && go test ./internal/service/ -run TestBLProduct -count=1 -v 2>&1 | tail -20
```

**Commit:** `feat(import): add BaseLinker product import with variants and image URLs`

---

## Task 6: Image Re-download Job

**Context:** After import, product images are external URLs. Users can trigger a background job to download and re-host them on S3.

**Files:**
- Create: `apps/api-server/internal/service/image_download_service.go`
- Create: `apps/api-server/internal/service/image_download_service_test.go`
- Modify: `apps/api-server/internal/handler/product_handler.go` — add RedownloadImages handler
- Modify: `apps/api-server/internal/router/router.go` — add route

### 6a: Image download service

```go
type ImageDownloadService struct {
	productRepo repository.ProductRepo
	pool        *pgxpool.Pool
	s3Client    *s3.Client // or Supabase Storage client
	bucketName  string
}

type ImageDownloadResult struct {
	Total      int `json:"total"`
	Downloaded int `json:"downloaded"`
	Failed     int `json:"failed"`
	Skipped    int `json:"skipped"` // already local URLs
}

func (s *ImageDownloadService) RedownloadImages(ctx context.Context, tenantID uuid.UUID) (*ImageDownloadResult, error) {
	// 1. List all products with external image URLs (http:// or https://)
	// 2. For each product:
	//    a. For each image URL:
	//       - Skip if already points to our S3/storage
	//       - Download with timeout (10s) and size limit (10MB)
	//       - Upload to S3: {bucket}/{tenant_id}/products/{product_id}/{filename}
	//       - Replace URL in images JSONB
	//    b. Update product with new image URLs
	// 3. Return result
}
```

### 6b: Handler + route

```go
// POST /v1/products/redownload-images
func (h *ProductHandler) RedownloadImages(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	result, err := h.imageDownloadService.RedownloadImages(r.Context(), tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}
```

**Note:** For MVP this runs synchronously. If slow (>30s for many images), convert to async job with polling endpoint later.

### 6c: Tests

- `TestImageDownload_DownloadsExternalURLs` — mock HTTP server, verify download + upload
- `TestImageDownload_SkipsLocalURLs` — URLs already on our domain are skipped
- `TestImageDownload_HandlesFailedDownloads` — 404/timeout → counted as failed, doesn't block others

**Verify:**
```bash
cd apps/api-server && go build ./... && go test ./internal/service/ -run TestImageDownload -count=1 -v 2>&1 | tail -10
```

**Commit:** `feat(import): add image re-download service for S3 upload`

---

## Task 7: Frontend — Customer Import Page

**Context:** Follow exact pattern from `/products/import/page.tsx` (simplified 2-step: upload → results).

**Files:**
- Create: `apps/dashboard/src/hooks/use-customer-import.ts`
- Create: `apps/dashboard/src/app/(dashboard)/customers/import/page.tsx`
- Modify: `apps/dashboard/src/app/(dashboard)/customers/page.tsx` — add "Importuj CSV" button

### 7a: Create hooks

```typescript
// use-customer-import.ts
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiFetch } from "@/lib/api-client";

interface CustomerImportPreview {
  headers: string[];
  total_rows: number;
  sample_rows: { row: number; data: Record<string, unknown>; errors?: string[] }[];
  new_count: number;
  update_count: number;
  mappings?: { csv_column: string; order_field: string }[];
}

interface CustomerImportResult {
  created: number;
  updated: number;
  skipped: number;
  errors: { row: number; field?: string; message: string }[];
}

export function useCustomerImportPreview() {
  return useMutation({
    mutationFn: async (file: File): Promise<CustomerImportPreview> => {
      const fd = new FormData();
      fd.append("file", file);
      const resp = await apiFetch("/v1/customers/import/preview", {
        method: "POST",
        body: fd,
      });
      return resp.json();
    },
  });
}

export function useCustomerImport() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (file: File): Promise<CustomerImportResult> => {
      const fd = new FormData();
      fd.append("file", file);
      const resp = await apiFetch("/v1/customers/import", {
        method: "POST",
        body: fd,
      });
      return resp.json();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["customers"] });
    },
  });
}
```

### 7b: Create import page

Copy structure from `/products/import/page.tsx`:
- Drag-and-drop file upload
- Preview with badges (total, new, update)
- Sample rows table
- Import button → results with created/updated/errors cards
- Polish labels: "Importuj klientów z pliku CSV", "Analizowanie...", "Importuj", etc.

### 7c: Add button to customers page

In `/customers/page.tsx`, add next to the "Nowy klient" button:
```tsx
<Link href="/customers/import">
  <Button variant="outline">
    <Upload className="h-4 w-4 mr-2" />
    Importuj CSV
  </Button>
</Link>
```

**Verify:**
```bash
cd apps/dashboard && npx tsc --noEmit 2>&1 | tail -10
```

**Commit:** `feat(dashboard): add customer CSV import page`

---

## Task 8: Frontend — Image Re-download Button

**Files:**
- Create: `apps/dashboard/src/hooks/use-image-redownload.ts`
- Modify: `apps/dashboard/src/app/(dashboard)/products/page.tsx` — add "Pobierz zdjęcia" button

### 8a: Hook

```typescript
export function useRedownloadImages() {
  return useMutation({
    mutationFn: async (): Promise<ImageDownloadResult> => {
      const resp = await apiFetch("/v1/products/redownload-images", {
        method: "POST",
      });
      return resp.json();
    },
  });
}
```

### 8b: Button on products page

Add button with loading state, toast on success showing results.

**Verify:**
```bash
cd apps/dashboard && npx tsc --noEmit 2>&1 | tail -10
```

**Commit:** `feat(dashboard): add image re-download button on products page`

---

## Task 9: E2E Tests

**Files:**
- Create: `apps/dashboard/e2e/customer-import.spec.ts`

### Test cases:

```typescript
test.describe('Customer Import', () => {
  test('customer import page loads', async ({ page }) => {
    await gotoWithAuth(page, '/customers/import');
    await expect(page.getByRole('heading', { name: /Importuj klientów/ })).toBeVisible();
  });

  test('upload CSV shows preview', async ({ page }) => {
    await gotoWithAuth(page, '/customers/import');
    // Upload test CSV file
    // Verify preview badges visible (total, new, update)
  });

  test('customers page has import button', async ({ page }) => {
    await gotoWithAuth(page, '/customers');
    await expect(page.getByRole('link', { name: /Importuj CSV/ })).toBeVisible();
  });
});
```

**Verify:**
```bash
cd apps/dashboard && npx playwright test customer-import.spec.ts --project=chromium --reporter=list 2>&1 | tail -10
```

**Commit:** `test(e2e): add customer import tests`

---

## Execution Order

1. Task 1: Customer import backend service (+ tests)
2. Task 2: Customer import handler + routes
3. Task 3: BL order parser with row grouping (+ tests)
4. Task 4: BL order import handler + routes
5. Task 5: BL product import with variants + images (+ tests)
6. Task 6: Image re-download service (+ tests)
7. Task 7: Frontend customer import page
8. Task 8: Frontend image re-download button
9. Task 9: E2E tests

## Verification

```bash
# Backend build + tests
cd apps/api-server && go build ./... && go test ./... 2>&1 | tail -5

# Frontend type check
cd apps/dashboard && npx tsc --noEmit 2>&1 | tail -10

# E2E
cd apps/dashboard && npx playwright test customer-import.spec.ts --project=chromium --reporter=list
```

## Key Reference Files

| File | Purpose |
|------|---------|
| `internal/service/import_service.go` | Order CSV import — pattern reference |
| `internal/service/product_import_service.go` | Product CSV import — pattern reference |
| `internal/handler/import_handler.go` | Import handler pattern |
| `internal/handler/product_handler.go` | Product import handler (ImportPreview, ImportCSV) |
| `internal/handler/customer_handler.go` | Customer handler — add import methods |
| `internal/model/import.go` | Import models (reuse ImportError, ImportPreviewRow) |
| `internal/model/customer.go` | Customer model + validation |
| `internal/model/variant.go` | ProductVariant model |
| `internal/repository/interfaces.go` | Repo interfaces (CustomerRepo, VariantRepo) |
| `internal/router/router.go` | Route registration |
| `cmd/server/main.go` | Service wiring / DI |
| `dashboard/src/hooks/use-product-import.ts` | Frontend import hook pattern |
| `dashboard/src/app/(dashboard)/products/import/page.tsx` | Frontend import page pattern |
