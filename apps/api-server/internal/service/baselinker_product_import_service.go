package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

// BaseLinkerProductImportService handles CSV import of products from BaseLinker exports,
// including BL column aliases, variant grouping by product_id, and image URL parsing.
type BaseLinkerProductImportService struct {
	productRepo  repository.ProductRepo
	variantRepo  repository.VariantRepo
	categoryRepo repository.ProductCategoryRepo
	auditRepo    repository.AuditRepo
	pool         *pgxpool.Pool
}

// NewBaseLinkerProductImportService creates a new BaseLinkerProductImportService.
func NewBaseLinkerProductImportService(
	productRepo repository.ProductRepo,
	variantRepo repository.VariantRepo,
	categoryRepo repository.ProductCategoryRepo,
	auditRepo repository.AuditRepo,
	pool *pgxpool.Pool,
) *BaseLinkerProductImportService {
	return &BaseLinkerProductImportService{
		productRepo:  productRepo,
		variantRepo:  variantRepo,
		categoryRepo: categoryRepo,
		auditRepo:    auditRepo,
		pool:         pool,
	}
}

// blProductAliases maps BaseLinker CSV column names to canonical field names.
var blProductAliases = map[string]string{
	"product_id":           "external_id",
	"name":                 "name",
	"product_name":         "name",
	"sku":                  "sku",
	"product_sku":          "sku",
	"ean":                  "ean",
	"product_ean":          "ean",
	"price_brutto":         "price",
	"product_price_brutto": "price",
	"quantity":             "stock_quantity",
	"stock":                "stock_quantity",
	"product_quantity":     "stock_quantity",
	"description":          "description_short",
	"description_extra":    "description_long",
	"category":             "category",
	"category_name":        "category",
	"weight":               "weight",
	"image_url":            "images",
	"images":               "images",
	"variant_id":           "variant_id",
	"variant_name":         "variant_name",
}

// BLProductImportPreview is the response for the BL product import preview endpoint.
type BLProductImportPreview struct {
	Headers      []string                    `json:"headers"`
	TotalRows    int                         `json:"total_rows"`
	ProductCount int                         `json:"product_count"`
	VariantCount int                         `json:"variant_count"`
	SampleRows   []model.ImportPreviewRow    `json:"sample_rows"`
	Mappings     []model.ImportColumnMapping `json:"mappings,omitempty"`
}

// BLProductImportResult is the summary returned after a BL product import.
type BLProductImportResult struct {
	ProductsCreated   int                 `json:"products_created"`
	ProductsUpdated   int                 `json:"products_updated"`
	VariantsCreated   int                 `json:"variants_created"`
	CategoriesCreated int                 `json:"categories_created"`
	Errors            []model.ImportError `json:"errors"`
}

// parseBLProductCSV reads raw CSV bytes, resolves BL aliases, and returns headers,
// records, canonical field index, and the auto-detected mappings.
func parseBLProductCSV(raw []byte) (
	headers []string,
	records [][]string,
	canonIdx map[string]int,
	mappings []model.ImportColumnMapping,
	err error,
) {
	raw = stripBOM(raw)

	reader := csv.NewReader(bytes.NewReader(raw))
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true

	records, err = reader.ReadAll()
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("parse csv: %w", err)
	}

	if len(records) < 1 {
		return nil, nil, nil, nil, fmt.Errorf("csv file is empty")
	}

	headers = records[0]

	// Build canonical field index using BL aliases.
	canonIdx = make(map[string]int, len(headers))
	for i, h := range headers {
		key := strings.ToLower(strings.TrimSpace(h))
		if canon, ok := blProductAliases[key]; ok {
			// First match wins for each canonical field.
			if _, exists := canonIdx[canon]; !exists {
				canonIdx[canon] = i
				mappings = append(mappings, model.ImportColumnMapping{
					CSVColumn:  h,
					OrderField: canon,
				})
			}
		} else {
			// Also store raw header name for non-aliased columns.
			if _, exists := canonIdx[key]; !exists {
				canonIdx[key] = i
			}
		}
	}

	return headers, records, canonIdx, mappings, nil
}

// getCanonVal retrieves a trimmed value from a row using the canonical field index.
func getCanonVal(row []string, canonIdx map[string]int, field string) string {
	idx, ok := canonIdx[field]
	if !ok || idx >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[idx])
}

// parseBLImages splits semicolon-separated image URLs into a JSON array.
func parseBLImages(raw string) json.RawMessage {
	if raw == "" {
		return json.RawMessage("[]")
	}
	var urls []string
	for u := range strings.SplitSeq(raw, ";") {
		u = strings.TrimSpace(u)
		if u != "" {
			urls = append(urls, u)
		}
	}
	if len(urls) == 0 {
		return json.RawMessage("[]")
	}
	b, _ := json.Marshal(urls)
	return b
}

// blProductRow holds parsed data from a single CSV row, classified as parent or variant.
type blProductRow struct {
	ExternalID string // from product_id column
	VariantID  string // from variant_id column (empty = parent product)
	Row        []string
	RowIndex   int // 1-based row number
}

// classifyBLRows separates CSV rows into parent products and variant rows,
// grouped by external_id (product_id).
func classifyBLRows(records [][]string, canonIdx map[string]int) (
	parentRows []*blProductRow,
	variantRows []*blProductRow,
	parentOrder []string, // unique product_ids in order
) {
	seen := make(map[string]bool)

	for rowNum := 1; rowNum < len(records); rowNum++ {
		row := records[rowNum]
		externalID := getCanonVal(row, canonIdx, "external_id")
		variantID := getCanonVal(row, canonIdx, "variant_id")

		parsed := &blProductRow{
			ExternalID: externalID,
			VariantID:  variantID,
			Row:        row,
			RowIndex:   rowNum,
		}

		if variantID != "" {
			variantRows = append(variantRows, parsed)
		} else {
			parentRows = append(parentRows, parsed)
			if externalID != "" && !seen[externalID] {
				seen[externalID] = true
				parentOrder = append(parentOrder, externalID)
			}
		}
	}

	return parentRows, variantRows, parentOrder
}

// PreviewCSV parses a BaseLinker product CSV and returns a preview with stats.
func (s *BaseLinkerProductImportService) PreviewCSV(ctx context.Context, tenantID uuid.UUID, file io.Reader) (*BLProductImportPreview, error) {
	raw, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("read csv: %w", err)
	}

	headers, records, canonIdx, mappings, err := parseBLProductCSV(raw)
	if err != nil {
		return nil, err
	}

	totalRows := len(records) - 1
	parentRows, variantRows, _ := classifyBLRows(records, canonIdx)

	// Build sample rows (up to 10).
	sampleCount := min(totalRows, 10)
	sampleRows := make([]model.ImportPreviewRow, 0, sampleCount)
	for i := 1; i <= sampleCount; i++ {
		row := records[i]
		data := make(map[string]any, len(headers))
		for j, h := range headers {
			if j < len(row) {
				data[h] = row[j]
			} else {
				data[h] = ""
			}
		}
		sampleRows = append(sampleRows, model.ImportPreviewRow{
			Row:  i,
			Data: data,
		})
	}

	return &BLProductImportPreview{
		Headers:      headers,
		TotalRows:    totalRows,
		ProductCount: len(parentRows),
		VariantCount: len(variantRows),
		SampleRows:   sampleRows,
		Mappings:     mappings,
	}, nil
}

// ImportCSV performs a batch import of products from a BaseLinker CSV export.
func (s *BaseLinkerProductImportService) ImportCSV(
	ctx context.Context,
	tenantID uuid.UUID,
	file io.Reader,
	userID uuid.UUID,
	ip string,
) (*BLProductImportResult, error) {
	raw, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("read csv: %w", err)
	}

	_, records, canonIdx, _, err := parseBLProductCSV(raw)
	if err != nil {
		return nil, err
	}

	if len(records) < 2 {
		return nil, fmt.Errorf("csv file must have a header row and at least one data row")
	}

	// Check that at least name is present.
	if _, ok := canonIdx["name"]; !ok {
		return nil, fmt.Errorf("CSV must have a product name column (recognized: name, product_name)")
	}

	parentRows, variantRows, _ := classifyBLRows(records, canonIdx)

	result := &BLProductImportResult{
		Errors: []model.ImportError{},
	}

	// Map external_id -> created/updated product ID for variant linking.
	productByExternalID := make(map[string]uuid.UUID)

	// Track which external_ids have variants.
	externalIDsWithVariants := make(map[string]bool)
	for _, vr := range variantRows {
		if vr.ExternalID != "" {
			externalIDsWithVariants[vr.ExternalID] = true
		}
	}

	// Category cache: name -> *model.ProductCategory.
	categoryCache := make(map[string]*model.ProductCategory)

	err = database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		// Phase 1: Import parent products.
		for _, pr := range parentRows {
			productID, importErr := s.importParentProduct(ctx, tx, tenantID, pr, canonIdx, result, externalIDsWithVariants, categoryCache)
			if importErr != nil {
				result.Errors = append(result.Errors, *importErr)
				continue
			}
			if pr.ExternalID != "" && productID != uuid.Nil {
				productByExternalID[pr.ExternalID] = productID
			}
		}

		// Phase 2: Import variants.
		variantCounter := make(map[string]int) // external_id -> auto-increment counter
		for _, vr := range variantRows {
			importErr := s.importVariantRow(ctx, tx, tenantID, vr, canonIdx, result, productByExternalID, variantCounter)
			if importErr != nil {
				result.Errors = append(result.Errors, *importErr)
			}
		}

		// Audit log.
		_ = s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     userID,
			Action:     "baselinker.product_import",
			EntityType: "product",
			EntityID:   uuid.Nil,
			Changes: map[string]string{
				"products_created":   strconv.Itoa(result.ProductsCreated),
				"products_updated":   strconv.Itoa(result.ProductsUpdated),
				"variants_created":   strconv.Itoa(result.VariantsCreated),
				"categories_created": strconv.Itoa(result.CategoriesCreated),
				"errors":             strconv.Itoa(len(result.Errors)),
			},
			IPAddress: ip,
		})

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("import baselinker products: %w", err)
	}

	return result, nil
}

// importParentProduct processes a single parent product row.
func (s *BaseLinkerProductImportService) importParentProduct(
	ctx context.Context,
	tx pgx.Tx,
	tenantID uuid.UUID,
	pr *blProductRow,
	canonIdx map[string]int,
	result *BLProductImportResult,
	externalIDsWithVariants map[string]bool,
	categoryCache map[string]*model.ProductCategory,
) (uuid.UUID, *model.ImportError) {
	row := pr.Row
	rowNum := pr.RowIndex

	name := getCanonVal(row, canonIdx, "name")
	if name == "" {
		return uuid.Nil, &model.ImportError{Row: rowNum, Field: "name", Message: "product name is required"}
	}

	sku := getCanonVal(row, canonIdx, "sku")
	ean := getCanonVal(row, canonIdx, "ean")
	categoryName := getCanonVal(row, canonIdx, "category")
	shortDesc := getCanonVal(row, canonIdx, "description_short")
	longDesc := getCanonVal(row, canonIdx, "description_long")
	imagesRaw := getCanonVal(row, canonIdx, "images")
	externalID := pr.ExternalID
	hasVariants := externalIDsWithVariants[externalID]

	// Parse price.
	var price float64
	if v := getCanonVal(row, canonIdx, "price"); v != "" {
		v = strings.ReplaceAll(v, ",", ".")
		parsed, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return uuid.Nil, &model.ImportError{Row: rowNum, Field: "price", Message: fmt.Sprintf("invalid number: %s", v)}
		}
		if parsed < 0 {
			return uuid.Nil, &model.ImportError{Row: rowNum, Field: "price", Message: "price must not be negative"}
		}
		price = parsed
	}

	// Parse stock_quantity.
	var stockQty int
	if v := getCanonVal(row, canonIdx, "stock_quantity"); v != "" {
		parsed, err := strconv.Atoi(v)
		if err != nil {
			return uuid.Nil, &model.ImportError{Row: rowNum, Field: "stock_quantity", Message: fmt.Sprintf("invalid integer: %s", v)}
		}
		if parsed < 0 {
			return uuid.Nil, &model.ImportError{Row: rowNum, Field: "stock_quantity", Message: "stock_quantity must not be negative"}
		}
		stockQty = parsed
	}

	// Parse weight.
	var weight *float64
	if v := getCanonVal(row, canonIdx, "weight"); v != "" {
		v = strings.ReplaceAll(v, ",", ".")
		parsed, err := strconv.ParseFloat(v, 64)
		if err == nil {
			weight = &parsed
		}
	}

	// Parse images.
	images := parseBLImages(imagesRaw)

	// Resolve category.
	var categoryPtr *string
	var categoryID *uuid.UUID
	if categoryName != "" {
		categoryPtr = &categoryName
		cat, err := s.resolveCategory(ctx, tx, tenantID, categoryName, categoryCache, result)
		if err == nil && cat != nil {
			categoryID = &cat.ID
		}
	}

	// Check if product exists by SKU (upsert logic).
	if sku != "" {
		existing, err := s.productRepo.FindBySKU(ctx, tx, sku)
		if err != nil {
			return uuid.Nil, &model.ImportError{Row: rowNum, Message: fmt.Sprintf("error looking up SKU: %s", err.Error())}
		}
		if existing != nil {
			// Update existing product.
			source := "baselinker"
			req := model.UpdateProductRequest{
				Name:             &name,
				Price:            &price,
				StockQuantity:    &stockQty,
				Source:           &source,
				DescriptionShort: &shortDesc,
				DescriptionLong:  &longDesc,
				Images:           &images,
				Weight:           weight,
			}
			if externalID != "" {
				req.ExternalID = &externalID
			}
			if ean != "" {
				req.EAN = &ean
			}
			if categoryPtr != nil {
				req.Category = categoryPtr
			}
			if categoryID != nil {
				req.CategoryID = categoryID
			}

			// Set has_variants if any variant rows reference this product.
			if hasVariants {
				if _, err := tx.Exec(ctx, "UPDATE products SET has_variants = true WHERE id = $1", existing.ID); err != nil {
					return uuid.Nil, &model.ImportError{Row: rowNum, Message: fmt.Sprintf("failed to set has_variants: %s", err.Error())}
				}
			}

			if err := s.productRepo.Update(ctx, tx, existing.ID, req); err != nil {
				return uuid.Nil, &model.ImportError{Row: rowNum, Message: fmt.Sprintf("failed to update product: %s", err.Error())}
			}
			result.ProductsUpdated++
			return existing.ID, nil
		}
	}

	// Create new product.
	var skuPtr *string
	if sku != "" {
		skuPtr = &sku
	}
	var eanPtr *string
	if ean != "" {
		eanPtr = &ean
	}
	var externalIDPtr *string
	if externalID != "" {
		externalIDPtr = &externalID
	}

	product := &model.Product{
		ID:               uuid.New(),
		TenantID:         tenantID,
		ExternalID:       externalIDPtr,
		Source:           "baselinker",
		Name:             name,
		SKU:              skuPtr,
		EAN:              eanPtr,
		Price:            price,
		StockQuantity:    stockQty,
		Metadata:         json.RawMessage("{}"),
		Tags:             []string{},
		DescriptionShort: shortDesc,
		DescriptionLong:  longDesc,
		Weight:           weight,
		Category:         categoryPtr,
		CategoryID:       categoryID,
		Images:           images,
		HasVariants:      hasVariants,
	}

	if err := s.productRepo.Create(ctx, tx, product); err != nil {
		return uuid.Nil, &model.ImportError{Row: rowNum, Message: fmt.Sprintf("failed to create product: %s", err.Error())}
	}
	result.ProductsCreated++
	return product.ID, nil
}

// importVariantRow processes a single variant row and creates a ProductVariant.
func (s *BaseLinkerProductImportService) importVariantRow(
	ctx context.Context,
	tx pgx.Tx,
	tenantID uuid.UUID,
	vr *blProductRow,
	canonIdx map[string]int,
	result *BLProductImportResult,
	productByExternalID map[string]uuid.UUID,
	variantCounter map[string]int,
) *model.ImportError {
	row := vr.Row
	rowNum := vr.RowIndex

	// Find parent product by external_id.
	parentID, ok := productByExternalID[vr.ExternalID]
	if !ok {
		return &model.ImportError{
			Row:     rowNum,
			Field:   "product_id",
			Message: fmt.Sprintf("parent product not found for product_id: %s", vr.ExternalID),
		}
	}

	// Variant name: use variant_name column, or auto-generate.
	variantName := getCanonVal(row, canonIdx, "variant_name")
	if variantName == "" {
		variantCounter[vr.ExternalID]++
		variantName = fmt.Sprintf("Wariant %d", variantCounter[vr.ExternalID])
	}

	// SKU.
	var skuPtr *string
	if sku := getCanonVal(row, canonIdx, "sku"); sku != "" {
		skuPtr = &sku
	}

	// EAN.
	var eanPtr *string
	if ean := getCanonVal(row, canonIdx, "ean"); ean != "" {
		eanPtr = &ean
	}

	// Price override.
	var priceOverride *float64
	if v := getCanonVal(row, canonIdx, "price"); v != "" {
		v = strings.ReplaceAll(v, ",", ".")
		parsed, err := strconv.ParseFloat(v, 64)
		if err == nil && parsed >= 0 {
			priceOverride = &parsed
		}
	}

	// Stock quantity.
	stockQty := 0
	if v := getCanonVal(row, canonIdx, "stock_quantity"); v != "" {
		parsed, err := strconv.Atoi(v)
		if err == nil && parsed >= 0 {
			stockQty = parsed
		}
	}

	// Weight.
	var weight *float64
	if v := getCanonVal(row, canonIdx, "weight"); v != "" {
		v = strings.ReplaceAll(v, ",", ".")
		parsed, err := strconv.ParseFloat(v, 64)
		if err == nil {
			weight = &parsed
		}
	}

	variant := &model.ProductVariant{
		ID:            uuid.New(),
		TenantID:      tenantID,
		ProductID:     parentID,
		SKU:           skuPtr,
		EAN:           eanPtr,
		Name:          variantName,
		Attributes:    json.RawMessage("{}"),
		PriceOverride: priceOverride,
		StockQuantity: stockQty,
		Weight:        weight,
		Active:        true,
	}

	if err := s.variantRepo.Create(ctx, tx, variant); err != nil {
		return &model.ImportError{Row: rowNum, Message: fmt.Sprintf("failed to create variant: %s", err.Error())}
	}
	result.VariantsCreated++
	return nil
}

// resolveCategory finds or creates a ProductCategory by name.
func (s *BaseLinkerProductImportService) resolveCategory(
	ctx context.Context,
	tx pgx.Tx,
	tenantID uuid.UUID,
	categoryName string,
	cache map[string]*model.ProductCategory,
	result *BLProductImportResult,
) (*model.ProductCategory, error) {
	// Check cache first.
	if cat, ok := cache[categoryName]; ok {
		return cat, nil
	}

	// Try fuzzy match.
	matches, err := s.categoryRepo.FuzzyMatch(ctx, tx, categoryName)
	if err != nil {
		return nil, err
	}

	// Look for exact match first.
	for i := range matches {
		if strings.EqualFold(matches[i].Name, categoryName) {
			cache[categoryName] = &matches[i]
			return &matches[i], nil
		}
	}

	// Create new category.
	slug := model.GenerateSlug(categoryName)
	if slug == "" {
		slug = "import-" + uuid.New().String()[:8]
	}

	// Ensure slug uniqueness.
	count, err := s.categoryRepo.CountBySlug(ctx, tx, slug)
	if err != nil {
		return nil, err
	}
	if count > 0 {
		slug = fmt.Sprintf("%s-%d", slug, count+1)
	}

	cat := &model.ProductCategory{
		ID:       uuid.New(),
		TenantID: tenantID,
		Name:     categoryName,
		Slug:     slug,
		Color:    "#6B7280",
	}

	if err := s.categoryRepo.Create(ctx, tx, cat); err != nil {
		return nil, fmt.Errorf("create category: %w", err)
	}

	cache[categoryName] = cat
	result.CategoriesCreated++
	return cat, nil
}
