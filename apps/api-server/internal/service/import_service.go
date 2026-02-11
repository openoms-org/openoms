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
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

// ImportService handles CSV import of orders.
type ImportService struct {
	orderRepo repository.OrderRepo
	auditRepo repository.AuditRepo
	pool      *pgxpool.Pool
}

// NewImportService creates a new ImportService.
func NewImportService(orderRepo repository.OrderRepo, auditRepo repository.AuditRepo, pool *pgxpool.Pool) *ImportService {
	return &ImportService{
		orderRepo: orderRepo,
		auditRepo: auditRepo,
		pool:      pool,
	}
}

// stripBOM removes the UTF-8 BOM (byte order mark) from the beginning of data if present.
func stripBOM(data []byte) []byte {
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		return data[3:]
	}
	return data
}

// ParseCSV reads a CSV file, returns headers and up to 10 sample rows.
func (s *ImportService) ParseCSV(file io.Reader) (*model.ImportPreviewResponse, error) {
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
	totalRows := len(records) - 1 // exclude header row

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

	// Auto-detect mappings based on known header names
	mappings := autoDetectMappings(headers)

	return &model.ImportPreviewResponse{
		Headers:    headers,
		TotalRows:  totalRows,
		SampleRows: sampleRows,
		Mappings:   mappings,
	}, nil
}

// knownFields maps common CSV column names to order fields for auto-detection.
var knownFields = map[string]string{
	"customer_name":  "customer_name",
	"customer name":  "customer_name",
	"name":           "customer_name",
	"customer_email": "customer_email",
	"customer email": "customer_email",
	"email":          "customer_email",
	"customer_phone": "customer_phone",
	"customer phone": "customer_phone",
	"phone":          "customer_phone",
	"total_amount":   "total_amount",
	"total amount":   "total_amount",
	"total":          "total_amount",
	"amount":         "total_amount",
	"currency":       "currency",
	"source":         "source",
	"external_id":    "external_id",
	"external id":    "external_id",
	"notes":          "notes",
	"status":         "status",
	"ordered_at":     "ordered_at",
	"ordered at":     "ordered_at",
	"order_date":     "ordered_at",
	"order date":     "ordered_at",
	"payment_status": "payment_status",
	"payment status": "payment_status",
	"payment_method": "payment_method",
	"payment method": "payment_method",
	"items":          "items",
	"tags":           "tags",
}

func autoDetectMappings(headers []string) []model.ImportColumnMapping {
	var mappings []model.ImportColumnMapping
	for _, h := range headers {
		lower := strings.ToLower(strings.TrimSpace(h))
		if field, ok := knownFields[lower]; ok {
			mappings = append(mappings, model.ImportColumnMapping{
				CSVColumn:  h,
				OrderField: field,
			})
		}
	}
	return mappings
}

// validOrderFields is the set of order fields that can be mapped from CSV.
var validOrderFields = map[string]bool{
	"customer_name":  true,
	"customer_email": true,
	"customer_phone": true,
	"total_amount":   true,
	"currency":       true,
	"source":         true,
	"external_id":    true,
	"notes":          true,
	"status":         true,
	"ordered_at":     true,
	"payment_status": true,
	"payment_method": true,
	"items":          true,
	"tags":           true,
}

// ImportOrders performs a batch import of orders from CSV data using provided column mappings.
func (s *ImportService) ImportOrders(
	ctx context.Context,
	tenantID uuid.UUID,
	file io.Reader,
	mappings []model.ImportColumnMapping,
	userID uuid.UUID,
	ip string,
) (*model.ImportResult, error) {
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

	// Build header index: header name -> column index
	headerIdx := make(map[string]int, len(headers))
	for i, h := range headers {
		headerIdx[h] = i
	}

	// Build effective mappings: order field -> column index
	fieldToCol := make(map[string]int)
	for _, m := range mappings {
		if !validOrderFields[m.OrderField] {
			continue
		}
		idx, ok := headerIdx[m.CSVColumn]
		if !ok {
			continue
		}
		fieldToCol[m.OrderField] = idx
	}

	result := &model.ImportResult{
		TotalRows: len(records) - 1,
		Errors:    []model.ImportError{},
	}

	err = database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		for rowNum := 1; rowNum < len(records); rowNum++ {
			row := records[rowNum]
			rowErrors := s.importRow(ctx, tx, tenantID, row, fieldToCol, rowNum, result)
			if len(rowErrors) > 0 {
				result.Errors = append(result.Errors, rowErrors...)
				result.Skipped++
			}
		}

		// Audit log
		_ = s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     userID,
			Action:     "import",
			EntityType: "order",
			EntityID:   uuid.Nil,
			Changes: map[string]string{
				"total":    strconv.Itoa(result.TotalRows),
				"imported": strconv.Itoa(result.Imported),
				"skipped":  strconv.Itoa(result.Skipped),
				"errors":   strconv.Itoa(len(result.Errors)),
			},
			IPAddress: ip,
		})

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("import orders: %w", err)
	}

	return result, nil
}

// importRow processes a single CSV row. Returns any errors encountered.
func (s *ImportService) importRow(
	ctx context.Context,
	tx pgx.Tx,
	tenantID uuid.UUID,
	row []string,
	fieldToCol map[string]int,
	rowNum int,
	result *model.ImportResult,
) []model.ImportError {
	var rowErrors []model.ImportError

	getVal := func(field string) string {
		idx, ok := fieldToCol[field]
		if !ok || idx >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[idx])
	}

	customerName := getVal("customer_name")
	if customerName == "" {
		rowErrors = append(rowErrors, model.ImportError{
			Row:     rowNum,
			Field:   "customer_name",
			Message: "customer_name is required",
		})
		return rowErrors
	}

	// Parse total_amount
	var totalAmount float64
	if v := getVal("total_amount"); v != "" {
		// Replace comma with dot for European decimal format
		v = strings.ReplaceAll(v, ",", ".")
		parsed, err := strconv.ParseFloat(v, 64)
		if err != nil {
			rowErrors = append(rowErrors, model.ImportError{
				Row:     rowNum,
				Field:   "total_amount",
				Message: fmt.Sprintf("invalid number: %s", v),
			})
			return rowErrors
		}
		if parsed < 0 {
			rowErrors = append(rowErrors, model.ImportError{
				Row:     rowNum,
				Field:   "total_amount",
				Message: "total_amount must be non-negative",
			})
			return rowErrors
		}
		totalAmount = parsed
	}

	// Currency
	currency := getVal("currency")
	if currency == "" {
		currency = "PLN"
	}

	// Source
	source := getVal("source")
	if source == "" {
		source = "import"
	}

	// Status
	status := getVal("status")
	if status == "" {
		status = "new"
	}

	// Payment status
	paymentStatus := getVal("payment_status")
	if paymentStatus == "" {
		paymentStatus = "pending"
	}

	// External ID — deduplication
	externalID := getVal("external_id")
	if externalID != "" {
		// Check for duplicate by external_id column
		existing, err := s.findByExternalIDColumn(ctx, tx, externalID)
		if err != nil {
			rowErrors = append(rowErrors, model.ImportError{
				Row:     rowNum,
				Field:   "external_id",
				Message: fmt.Sprintf("error checking duplicate: %s", err.Error()),
			})
			return rowErrors
		}
		if existing {
			rowErrors = append(rowErrors, model.ImportError{
				Row:     rowNum,
				Field:   "external_id",
				Message: fmt.Sprintf("duplicate external_id: %s", externalID),
			})
			return rowErrors
		}
	}

	// Optional string fields
	customerEmail := getVal("customer_email")
	customerPhone := getVal("customer_phone")
	notes := getVal("notes")
	paymentMethod := getVal("payment_method")

	// Parse ordered_at
	var orderedAt *time.Time
	if v := getVal("ordered_at"); v != "" {
		t, err := parseFlexibleTime(v)
		if err != nil {
			rowErrors = append(rowErrors, model.ImportError{
				Row:     rowNum,
				Field:   "ordered_at",
				Message: fmt.Sprintf("invalid date: %s", v),
			})
			return rowErrors
		}
		orderedAt = &t
	}

	// Parse items (JSON string)
	var items json.RawMessage
	if v := getVal("items"); v != "" {
		if json.Valid([]byte(v)) {
			items = json.RawMessage(v)
		} else {
			rowErrors = append(rowErrors, model.ImportError{
				Row:     rowNum,
				Field:   "items",
				Message: "invalid JSON for items",
			})
			return rowErrors
		}
	}

	// Parse tags (comma-separated)
	var tags []string
	if v := getVal("tags"); v != "" {
		for _, t := range strings.Split(v, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				tags = append(tags, t)
			}
		}
	}
	if tags == nil {
		tags = []string{}
	}

	order := model.Order{
		ID:            uuid.New(),
		TenantID:      tenantID,
		Source:        source,
		Status:        status,
		CustomerName:  customerName,
		TotalAmount:   totalAmount,
		Currency:      currency,
		Tags:          tags,
		Items:         items,
		PaymentStatus: paymentStatus,
		OrderedAt:     orderedAt,
	}

	if externalID != "" {
		order.ExternalID = &externalID
	}
	if customerEmail != "" {
		order.CustomerEmail = &customerEmail
	}
	if customerPhone != "" {
		order.CustomerPhone = &customerPhone
	}
	if notes != "" {
		order.Notes = &notes
	}
	if paymentMethod != "" {
		order.PaymentMethod = &paymentMethod
	}

	if err := s.orderRepo.Create(ctx, tx, &order); err != nil {
		rowErrors = append(rowErrors, model.ImportError{
			Row:     rowNum,
			Message: fmt.Sprintf("failed to create order: %s", err.Error()),
		})
		return rowErrors
	}

	result.Imported++
	return nil
}

// findByExternalIDColumn checks if an order with the given external_id column value
// already exists in the current tenant scope.
func (s *ImportService) findByExternalIDColumn(ctx context.Context, tx pgx.Tx, externalID string) (bool, error) {
	var exists bool
	err := tx.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM orders WHERE external_id = $1)",
		externalID,
	).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

// parseFlexibleTime tries several common date/time formats.
func parseFlexibleTime(s string) (time.Time, error) {
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
		"02.01.2006",
		"02-01-2006",
		"02/01/2006",
		"01/02/2006",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognized date format: %s", s)
}
