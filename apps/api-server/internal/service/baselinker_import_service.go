package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
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

// BaseLinkerImportService handles CSV import of orders from BaseLinker exports,
// where each CSV row represents a single order item and rows must be grouped
// by order_id to produce one order with multiple items.
type BaseLinkerImportService struct {
	orderRepo    repository.OrderRepo
	customerRepo repository.CustomerRepo
	auditRepo    repository.AuditRepo
	tenantRepo   repository.TenantRepo
	pool         *pgxpool.Pool
}

// NewBaseLinkerImportService creates a new BaseLinkerImportService.
func NewBaseLinkerImportService(
	orderRepo repository.OrderRepo,
	customerRepo repository.CustomerRepo,
	auditRepo repository.AuditRepo,
	tenantRepo repository.TenantRepo,
	pool *pgxpool.Pool,
) *BaseLinkerImportService {
	return &BaseLinkerImportService{
		orderRepo:    orderRepo,
		customerRepo: customerRepo,
		auditRepo:    auditRepo,
		tenantRepo:   tenantRepo,
		pool:         pool,
	}
}

// blOrderGroup holds all CSV rows belonging to a single order_id.
type blOrderGroup struct {
	OrderID       string     // order_id value
	Rows          [][]string // all CSV rows for this order_id
	FirstRowIndex int        // 1-based row number of first occurrence
}

// BLOrderImportPreview is the response for the BL order import preview endpoint.
type BLOrderImportPreview struct {
	Headers    []string                    `json:"headers"`
	TotalRows  int                         `json:"total_rows"`
	OrderCount int                         `json:"order_count"`
	SampleRows []model.ImportPreviewRow    `json:"sample_rows"`
	Mappings   []model.ImportColumnMapping `json:"mappings,omitempty"`
}

// BLOrderImportResult is the summary returned after a BL order import.
type BLOrderImportResult struct {
	OrdersCreated    int                 `json:"orders_created"`
	CustomersCreated int                 `json:"customers_created"`
	Skipped          int                 `json:"skipped"`
	Errors           []model.ImportError `json:"errors"`
}

// blOrderIDAliases are the recognized column names for the order ID field.
var blOrderIDAliases = []string{"order_id", "order_number", "bl_order_id"}

// extractBLField tries multiple header name aliases and returns the first non-empty value.
func extractBLField(row []string, headerIdx map[string]int, aliases ...string) string {
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

// parseBLOrderCSV reads raw CSV bytes and returns headers, records, and a header-to-index map
// with lowercase-normalized keys.
func parseBLOrderCSV(raw []byte) (headers []string, records [][]string, headerIdx map[string]int, err error) {
	raw = stripBOM(raw)

	reader := csv.NewReader(bytes.NewReader(raw))
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true

	records, err = reader.ReadAll()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("parse csv: %w", err)
	}

	if len(records) < 1 {
		return nil, nil, nil, fmt.Errorf("csv file is empty")
	}

	headers = records[0]

	// Build header index: lowercase header name → column index.
	headerIdx = make(map[string]int, len(headers))
	for i, h := range headers {
		key := strings.ToLower(strings.TrimSpace(h))
		if _, exists := headerIdx[key]; !exists {
			headerIdx[key] = i
		}
	}

	return headers, records, headerIdx, nil
}

// findOrderIDColumnIndex returns the column index for the order_id field,
// trying all known aliases.
func findOrderIDColumnIndex(headerIdx map[string]int) (int, bool) {
	for _, alias := range blOrderIDAliases {
		if idx, ok := headerIdx[alias]; ok {
			return idx, true
		}
	}
	return -1, false
}

// groupByOrderID groups CSV data rows by order_id value.
// Returns groups in insertion order.
func groupByOrderID(records [][]string, headerIdx map[string]int) ([]*blOrderGroup, error) {
	orderIDIdx, found := findOrderIDColumnIndex(headerIdx)
	if !found {
		return nil, fmt.Errorf("CSV must have an order_id column (recognized: %s)", strings.Join(blOrderIDAliases, ", "))
	}

	groupMap := make(map[string]*blOrderGroup)
	var groupOrder []string

	for rowNum := 1; rowNum < len(records); rowNum++ {
		row := records[rowNum]
		if orderIDIdx >= len(row) {
			continue
		}

		orderID := strings.TrimSpace(row[orderIDIdx])
		if orderID == "" {
			continue
		}

		if g, ok := groupMap[orderID]; ok {
			g.Rows = append(g.Rows, row)
		} else {
			groupMap[orderID] = &blOrderGroup{
				OrderID:       orderID,
				Rows:          [][]string{row},
				FirstRowIndex: rowNum, // 1-based (row 0 = header)
			}
			groupOrder = append(groupOrder, orderID)
		}
	}

	groups := make([]*blOrderGroup, 0, len(groupOrder))
	for _, id := range groupOrder {
		groups = append(groups, groupMap[id])
	}

	return groups, nil
}

// extractBLShippingAddress builds a shipping address JSON from the first row of a group.
func extractBLShippingAddress(row []string, headerIdx map[string]int) json.RawMessage {
	addr := map[string]string{}
	if v := extractBLField(row, headerIdx, "delivery_fullname"); v != "" {
		addr["name"] = v
	}
	if v := extractBLField(row, headerIdx, "delivery_address", "delivery_street"); v != "" {
		addr["street"] = v
	}
	if v := extractBLField(row, headerIdx, "delivery_city"); v != "" {
		addr["city"] = v
	}
	if v := extractBLField(row, headerIdx, "delivery_postcode", "delivery_zipcode"); v != "" {
		addr["postal_code"] = v
	}
	if v := extractBLField(row, headerIdx, "delivery_country_code", "delivery_country"); v != "" {
		addr["country"] = v
	}
	if len(addr) == 0 {
		return nil
	}
	b, _ := json.Marshal(addr)
	return b
}

// extractBLBillingAddress builds a billing address JSON from the first row of a group.
func extractBLBillingAddress(row []string, headerIdx map[string]int) json.RawMessage {
	addr := map[string]string{}
	if v := extractBLField(row, headerIdx, "invoice_fullname"); v != "" {
		addr["name"] = v
	}
	if v := extractBLField(row, headerIdx, "invoice_address"); v != "" {
		addr["street"] = v
	}
	if v := extractBLField(row, headerIdx, "invoice_city"); v != "" {
		addr["city"] = v
	}
	if v := extractBLField(row, headerIdx, "invoice_postcode"); v != "" {
		addr["postal_code"] = v
	}
	if v := extractBLField(row, headerIdx, "invoice_country"); v != "" {
		addr["country"] = v
	}
	if v := extractBLField(row, headerIdx, "invoice_company"); v != "" {
		addr["company"] = v
	}
	if v := extractBLField(row, headerIdx, "invoice_nip"); v != "" {
		addr["nip"] = v
	}
	if len(addr) == 0 {
		return nil
	}
	b, _ := json.Marshal(addr)
	return b
}

// blItem represents a single line item extracted from a CSV row.
type blItem struct {
	Name     string  `json:"name"`
	SKU      string  `json:"sku"`
	Quantity int     `json:"quantity"`
	Price    float64 `json:"price"`
}

// extractBLItems extracts items from all rows in a group and computes total_amount.
func extractBLItems(rows [][]string, headerIdx map[string]int) (json.RawMessage, float64) {
	var items []blItem
	var totalAmount float64

	for _, row := range rows {
		name := extractBLField(row, headerIdx, "product_name")
		sku := extractBLField(row, headerIdx, "product_sku")

		qtyStr := extractBLField(row, headerIdx, "product_quantity")
		qty := 1
		if qtyStr != "" {
			if parsed, err := strconv.Atoi(qtyStr); err == nil && parsed > 0 {
				qty = parsed
			}
		}

		priceStr := extractBLField(row, headerIdx, "product_price_brutto")
		priceStr = strings.ReplaceAll(priceStr, ",", ".")
		var price float64
		if priceStr != "" {
			if parsed, err := strconv.ParseFloat(priceStr, 64); err == nil {
				price = parsed
			}
		}

		items = append(items, blItem{
			Name:     name,
			SKU:      sku,
			Quantity: qty,
			Price:    price,
		})

		totalAmount += float64(qty) * price
	}

	// Round to 2 decimal places to avoid floating point drift.
	totalAmount = math.Round(totalAmount*100) / 100

	if len(items) == 0 {
		return nil, 0
	}

	b, _ := json.Marshal(items)
	return b, totalAmount
}

// PreviewOrders parses a BaseLinker CSV and returns a preview with order count.
func (s *BaseLinkerImportService) PreviewOrders(_ context.Context, _ uuid.UUID, file io.Reader) (*BLOrderImportPreview, error) {
	raw, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("read csv: %w", err)
	}

	headers, records, headerIdx, err := parseBLOrderCSV(raw)
	if err != nil {
		return nil, err
	}

	groups, err := groupByOrderID(records, headerIdx)
	if err != nil {
		return nil, err
	}

	totalRows := len(records) - 1

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

	return &BLOrderImportPreview{
		Headers:    headers,
		TotalRows:  totalRows,
		OrderCount: len(groups),
		SampleRows: sampleRows,
	}, nil
}

// ImportOrders performs a batch import of orders from a BaseLinker CSV export.
// Rows are grouped by order_id; each group produces one order with aggregated items.
// If importCustomers is true, customers are auto-created from order data.
func (s *BaseLinkerImportService) ImportOrders(
	ctx context.Context,
	tenantID uuid.UUID,
	file io.Reader,
	userID uuid.UUID,
	ip string,
	importCustomers bool,
) (*BLOrderImportResult, error) {
	raw, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("read csv: %w", err)
	}

	_, records, headerIdx, err := parseBLOrderCSV(raw)
	if err != nil {
		return nil, err
	}

	if len(records) < 2 {
		return nil, fmt.Errorf("csv file must have a header row and at least one data row")
	}

	groups, err := groupByOrderID(records, headerIdx)
	if err != nil {
		return nil, err
	}

	result := &BLOrderImportResult{
		Errors: []model.ImportError{},
	}

	err = database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		statusConfig := s.loadStatusConfig(ctx, tx, tenantID)

		for _, group := range groups {
			groupErrors := s.importOrderGroup(ctx, tx, tenantID, group, headerIdx, result, statusConfig, importCustomers)
			if len(groupErrors) > 0 {
				result.Errors = append(result.Errors, groupErrors...)
				result.Skipped++
			}
		}

		// Audit log.
		_ = s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     userID,
			Action:     "baselinker.import",
			EntityType: "order",
			EntityID:   uuid.Nil,
			Changes: map[string]string{
				"orders_created":    strconv.Itoa(result.OrdersCreated),
				"customers_created": strconv.Itoa(result.CustomersCreated),
				"skipped":           strconv.Itoa(result.Skipped),
				"errors":            strconv.Itoa(len(result.Errors)),
			},
			IPAddress: ip,
		})

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("import baselinker orders: %w", err)
	}

	return result, nil
}

// loadStatusConfig loads the tenant's order status configuration.
func (s *BaseLinkerImportService) loadStatusConfig(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID) *model.OrderStatusConfig {
	settings, err := s.tenantRepo.GetSettings(ctx, tx, tenantID)
	if err != nil {
		slog.Warn("bl-import: failed to load tenant settings for status validation", "error", err)
		cfg := model.DefaultOrderStatusConfig()
		return &cfg
	}

	if settings != nil {
		var allSettings map[string]json.RawMessage
		if err := json.Unmarshal(settings, &allSettings); err == nil {
			if raw, ok := allSettings["order_statuses"]; ok {
				var config model.OrderStatusConfig
				if err := json.Unmarshal(raw, &config); err != nil {
					slog.Warn("bl-import: failed to unmarshal order status config", "error", err)
				} else if len(config.Statuses) > 0 {
					return &config
				}
			}
		}
	}

	cfg := model.DefaultOrderStatusConfig()
	return &cfg
}

// importOrderGroup processes a single order group (all rows sharing one order_id).
func (s *BaseLinkerImportService) importOrderGroup(
	ctx context.Context,
	tx pgx.Tx,
	tenantID uuid.UUID,
	group *blOrderGroup,
	headerIdx map[string]int,
	result *BLOrderImportResult,
	statusConfig *model.OrderStatusConfig,
	importCustomers bool,
) []model.ImportError {
	var errs []model.ImportError
	firstRow := group.Rows[0]
	rowNum := group.FirstRowIndex

	// Customer data from first row.
	customerName := extractBLField(firstRow, headerIdx, "delivery_fullname", "buyer_name")
	if customerName == "" {
		errs = append(errs, model.ImportError{
			Row:     rowNum,
			Field:   "customer_name",
			Message: "customer name is required (delivery_fullname or buyer_name)",
		})
		return errs
	}

	customerEmail := extractBLField(firstRow, headerIdx, "buyer_email")
	customerPhone := extractBLField(firstRow, headerIdx, "buyer_phone")

	// Addresses from first row.
	shippingAddress := extractBLShippingAddress(firstRow, headerIdx)
	billingAddress := extractBLBillingAddress(firstRow, headerIdx)

	// Items from all rows in the group.
	items, totalAmount := extractBLItems(group.Rows, headerIdx)

	// Date.
	var orderedAt *time.Time
	if v := extractBLField(firstRow, headerIdx, "date_add"); v != "" {
		t, err := parseFlexibleTime(v)
		if err != nil {
			errs = append(errs, model.ImportError{
				Row:     rowNum,
				Field:   "date_add",
				Message: fmt.Sprintf("invalid date: %s", v),
			})
			return errs
		}
		orderedAt = &t
	}

	// Payment info.
	paymentMethod := extractBLField(firstRow, headerIdx, "payment_method")
	paymentDone := extractBLField(firstRow, headerIdx, "payment_done")
	paymentStatus := "pending"
	if paymentDone != "" {
		paymentDone = strings.ReplaceAll(paymentDone, ",", ".")
		if parsed, err := strconv.ParseFloat(paymentDone, 64); err == nil && parsed > 0 {
			paymentStatus = "paid"
		}
	}

	// Currency.
	currency := extractBLField(firstRow, headerIdx, "currency")
	if currency == "" {
		currency = "PLN"
	}

	// Status.
	status := extractBLField(firstRow, headerIdx, "order_status_name")
	if status == "" {
		status = "new"
	}
	if statusConfig != nil && !statusConfig.IsValidStatus(status) {
		// Use the raw status name as-is; fall back to "new" if not recognized.
		status = "new"
	}

	// External ID deduplication (order_id as external_id).
	externalID := group.OrderID
	existing, err := s.orderRepo.FindByExternalID(ctx, tx, "baselinker", externalID)
	if err != nil {
		errs = append(errs, model.ImportError{
			Row:     rowNum,
			Field:   "order_id",
			Message: fmt.Sprintf("error checking duplicate: %s", err.Error()),
		})
		return errs
	}
	if existing != nil {
		errs = append(errs, model.ImportError{
			Row:     rowNum,
			Field:   "order_id",
			Message: fmt.Sprintf("duplicate order_id: %s", externalID),
		})
		return errs
	}

	// Optionally create customer.
	var customerID *uuid.UUID
	if importCustomers && customerEmail != "" {
		existingCustomer, findErr := s.customerRepo.FindByEmail(ctx, tx, customerEmail)
		switch {
		case findErr != nil:
			slog.Warn("bl-import: error looking up customer by email",
				"error", findErr,
				"tenant_id", tenantID,
				"order_id", externalID,
			)
		case existingCustomer != nil:
			customerID = &existingCustomer.ID
		default:
			newCustomer := &model.Customer{
				ID:       uuid.New(),
				TenantID: tenantID,
				Name:     customerName,
				Email:    strPtr(customerEmail),
				Phone:    strPtr(customerPhone),
				Tags:     []string{},
			}
			if createErr := s.customerRepo.Create(ctx, tx, newCustomer); createErr != nil {
				slog.Warn("bl-import: failed to create customer",
					"error", createErr,
					"tenant_id", tenantID,
					"order_id", externalID,
				)
			} else {
				customerID = &newCustomer.ID
				result.CustomersCreated++
			}
		}
	}

	order := model.Order{
		ID:              uuid.New(),
		TenantID:        tenantID,
		ExternalID:      &externalID,
		Source:          "baselinker",
		Status:          status,
		CustomerName:    customerName,
		TotalAmount:     totalAmount,
		Currency:        currency,
		Tags:            []string{},
		Items:           items,
		ShippingAddress: shippingAddress,
		BillingAddress:  billingAddress,
		PaymentStatus:   paymentStatus,
		OrderedAt:       orderedAt,
		Metadata:        json.RawMessage(`{}`),
		CustomerID:      customerID,
		Priority:        "normal",
	}

	if order.OrderedAt == nil {
		now := time.Now()
		order.OrderedAt = &now
	}
	if customerEmail != "" {
		order.CustomerEmail = &customerEmail
	}
	if customerPhone != "" {
		order.CustomerPhone = &customerPhone
	}
	if paymentMethod != "" {
		order.PaymentMethod = &paymentMethod
	}

	created, err := s.orderRepo.CreateIfExternalIDNotExists(ctx, tx, &order)
	if err != nil {
		errs = append(errs, model.ImportError{
			Row:     rowNum,
			Message: fmt.Sprintf("failed to create order: %s", err.Error()),
		})
		return errs
	}
	if !created {
		errs = append(errs, model.ImportError{
			Row:     rowNum,
			Field:   "order_id",
			Message: fmt.Sprintf("duplicate order_id: %s", externalID),
		})
		return errs
	}

	result.OrdersCreated++
	return nil
}

// strPtr returns a pointer to s if non-empty, or nil otherwise.
func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
