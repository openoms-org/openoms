package service

import (
	"bytes"
	"context"
	"encoding/csv"
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

// ProductImportService handles CSV import/export of products.
type ProductImportService struct {
	productRepo repository.ProductRepo
	auditRepo   repository.AuditRepo
	pool        *pgxpool.Pool
}

// NewProductImportService creates a new ProductImportService.
func NewProductImportService(
	productRepo repository.ProductRepo,
	auditRepo repository.AuditRepo,
	pool *pgxpool.Pool,
) *ProductImportService {
	return &ProductImportService{
		productRepo: productRepo,
		auditRepo:   auditRepo,
		pool:        pool,
	}
}

// ProductImportPreview is the response for the import preview endpoint.
type ProductImportPreview struct {
	Headers    []string                    `json:"headers"`
	TotalRows  int                         `json:"total_rows"`
	SampleRows []model.ImportPreviewRow    `json:"sample_rows"`
	NewCount   int                         `json:"new_count"`
	UpdateCount int                        `json:"update_count"`
}

// ProductImportResult is the response for the import endpoint.
type ProductImportResult struct {
	Created int                `json:"created"`
	Updated int                `json:"updated"`
	Errors  []model.ImportError `json:"errors"`
}

// requiredProductCSVHeaders lists the expected CSV header names.
var requiredProductCSVHeaders = map[string]bool{
	"name": true,
}

// PreviewCSV parses a CSV and returns a preview with stats.
func (s *ProductImportService) PreviewCSV(ctx context.Context, tenantID uuid.UUID, file io.Reader) (*ProductImportPreview, error) {
	raw, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("read csv: %w", err)
	}
	raw = stripBOM(raw)

	reader := csv.NewReader(bytes.NewReader(raw))
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true

	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("parse csv: %w", err)
	}

	if len(records) < 1 {
		return nil, fmt.Errorf("csv file is empty")
	}

	headers := records[0]
	totalRows := len(records) - 1

	// Validate that "name" header exists
	hasName := false
	for _, h := range headers {
		if strings.ToLower(strings.TrimSpace(h)) == "name" {
			hasName = true
			break
		}
	}
	if !hasName {
		return nil, fmt.Errorf("CSV must have a 'name' column header")
	}

	// Build sample rows (up to 10)
	sampleCount := totalRows
	if sampleCount > 10 {
		sampleCount = 10
	}

	sampleRows := make([]model.ImportPreviewRow, 0, sampleCount)
	for i := 1; i <= sampleCount; i++ {
		row := records[i]
		data := make(map[string]interface{})
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

	// Build header index
	headerIdx := make(map[string]int, len(headers))
	for i, h := range headers {
		headerIdx[strings.ToLower(strings.TrimSpace(h))] = i
	}

	// Count new vs updates by SKU match
	newCount := 0
	updateCount := 0

	err = database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		skuIdx, hasSKU := headerIdx["sku"]
		for rowNum := 1; rowNum < len(records); rowNum++ {
			row := records[rowNum]
			if hasSKU && skuIdx < len(row) {
				sku := strings.TrimSpace(row[skuIdx])
				if sku != "" {
					existing, err := s.productRepo.FindBySKU(ctx, tx, sku)
					if err != nil {
						return err
					}
					if existing != nil {
						updateCount++
						continue
					}
				}
			}
			newCount++
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("preview analysis: %w", err)
	}

	return &ProductImportPreview{
		Headers:     headers,
		TotalRows:   totalRows,
		SampleRows:  sampleRows,
		NewCount:    newCount,
		UpdateCount: updateCount,
	}, nil
}

// ImportCSV performs a batch import of products from CSV, upserting by SKU.
func (s *ProductImportService) ImportCSV(
	ctx context.Context,
	tenantID uuid.UUID,
	file io.Reader,
	userID uuid.UUID,
	ip string,
) (*ProductImportResult, error) {
	raw, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("read csv: %w", err)
	}
	raw = stripBOM(raw)

	reader := csv.NewReader(bytes.NewReader(raw))
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true

	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("parse csv: %w", err)
	}

	if len(records) < 2 {
		return nil, fmt.Errorf("csv file must have a header row and at least one data row")
	}

	headers := records[0]

	// Build header index (lowercase)
	headerIdx := make(map[string]int, len(headers))
	for i, h := range headers {
		headerIdx[strings.ToLower(strings.TrimSpace(h))] = i
	}

	// Validate name column exists
	if _, ok := headerIdx["name"]; !ok {
		return nil, fmt.Errorf("CSV must have a 'name' column header")
	}

	result := &ProductImportResult{
		Errors: []model.ImportError{},
	}

	err = database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		for rowNum := 1; rowNum < len(records); rowNum++ {
			row := records[rowNum]
			rowErr := s.importProductRow(ctx, tx, tenantID, row, headerIdx, rowNum, result)
			if rowErr != nil {
				result.Errors = append(result.Errors, *rowErr)
			}
		}

		// Audit log
		_ = s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     userID,
			Action:     "product.import",
			EntityType: "product",
			EntityID:   uuid.Nil,
			Changes: map[string]string{
				"created": strconv.Itoa(result.Created),
				"updated": strconv.Itoa(result.Updated),
				"errors":  strconv.Itoa(len(result.Errors)),
			},
			IPAddress: ip,
		})

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("import products: %w", err)
	}

	return result, nil
}

func (s *ProductImportService) importProductRow(
	ctx context.Context,
	tx pgx.Tx,
	tenantID uuid.UUID,
	row []string,
	headerIdx map[string]int,
	rowNum int,
	result *ProductImportResult,
) *model.ImportError {
	getVal := func(field string) string {
		idx, ok := headerIdx[field]
		if !ok || idx >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[idx])
	}

	name := getVal("name")
	if name == "" {
		return &model.ImportError{Row: rowNum, Field: "name", Message: "name is required"}
	}

	sku := getVal("sku")
	ean := getVal("ean")
	category := getVal("category")
	shortDesc := getVal("short_description")
	tagsStr := getVal("tags")

	// Parse price
	var price float64
	if v := getVal("price"); v != "" {
		v = strings.ReplaceAll(v, ",", ".")
		parsed, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return &model.ImportError{Row: rowNum, Field: "price", Message: fmt.Sprintf("invalid number: %s", v)}
		}
		if parsed < 0 {
			return &model.ImportError{Row: rowNum, Field: "price", Message: "price must not be negative"}
		}
		price = parsed
	}

	// Parse stock_quantity
	var stockQty int
	if v := getVal("stock_quantity"); v != "" {
		parsed, err := strconv.Atoi(v)
		if err != nil {
			return &model.ImportError{Row: rowNum, Field: "stock_quantity", Message: fmt.Sprintf("invalid integer: %s", v)}
		}
		if parsed < 0 {
			return &model.ImportError{Row: rowNum, Field: "stock_quantity", Message: "stock_quantity must not be negative"}
		}
		stockQty = parsed
	}

	// Parse tags (comma-separated)
	var tags []string
	if tagsStr != "" {
		for _, t := range strings.Split(tagsStr, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				tags = append(tags, t)
			}
		}
	}
	if tags == nil {
		tags = []string{}
	}

	// Parse dimensions
	parseFloat := func(field string) *float64 {
		v := getVal(field)
		if v == "" {
			return nil
		}
		v = strings.ReplaceAll(v, ",", ".")
		parsed, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return nil
		}
		return &parsed
	}

	weight := parseFloat("weight")
	width := parseFloat("width")
	height := parseFloat("height")
	depth := parseFloat("length")

	// Check if product exists by SKU (upsert logic)
	if sku != "" {
		existing, err := s.productRepo.FindBySKU(ctx, tx, sku)
		if err != nil {
			return &model.ImportError{Row: rowNum, Message: fmt.Sprintf("error looking up SKU: %s", err.Error())}
		}
		if existing != nil {
			// Update existing product
			req := model.UpdateProductRequest{
				Name:          &name,
				Price:         &price,
				StockQuantity: &stockQty,
				Tags:          &tags,
			}
			if ean != "" {
				req.EAN = &ean
			}
			if category != "" {
				req.Category = &category
			}
			if shortDesc != "" {
				req.DescriptionShort = &shortDesc
			}
			if weight != nil {
				req.Weight = weight
			}
			if width != nil {
				req.Width = width
			}
			if height != nil {
				req.Height = height
			}
			if depth != nil {
				req.Depth = depth
			}

			if err := s.productRepo.Update(ctx, tx, existing.ID, req); err != nil {
				return &model.ImportError{Row: rowNum, Message: fmt.Sprintf("failed to update product: %s", err.Error())}
			}
			result.Updated++
			return nil
		}
	}

	// Create new product
	var skuPtr *string
	if sku != "" {
		skuPtr = &sku
	}
	var eanPtr *string
	if ean != "" {
		eanPtr = &ean
	}
	var categoryPtr *string
	if category != "" {
		categoryPtr = &category
	}

	product := &model.Product{
		ID:               uuid.New(),
		TenantID:         tenantID,
		Source:           "import",
		Name:             name,
		SKU:              skuPtr,
		EAN:              eanPtr,
		Price:            price,
		StockQuantity:    stockQty,
		Metadata:         []byte("{}"),
		Tags:             tags,
		DescriptionShort: shortDesc,
		Category:         categoryPtr,
		Weight:           weight,
		Width:            width,
		Height:           height,
		Depth:            depth,
		Images:           []byte("[]"),
	}

	if err := s.productRepo.Create(ctx, tx, product); err != nil {
		return &model.ImportError{Row: rowNum, Message: fmt.Sprintf("failed to create product: %s", err.Error())}
	}
	result.Created++
	return nil
}
