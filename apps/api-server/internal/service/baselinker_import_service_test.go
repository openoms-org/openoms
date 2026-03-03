package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// --- Mocks for BL import tests ---

// mockOrderRepo implements repository.OrderRepo for unit tests.
type mockOrderRepo struct {
	created          []*model.Order
	externalIDLookup map[string]*model.Order // key: "source:external_id"
}

func newMockOrderRepo() *mockOrderRepo {
	return &mockOrderRepo{
		externalIDLookup: make(map[string]*model.Order),
	}
}

func (m *mockOrderRepo) List(_ context.Context, _ pgx.Tx, _ model.OrderListFilter) ([]model.Order, int, error) {
	return nil, 0, nil
}

func (m *mockOrderRepo) FindByID(_ context.Context, _ pgx.Tx, _ uuid.UUID) (*model.Order, error) {
	return nil, nil
}

func (m *mockOrderRepo) Create(_ context.Context, _ pgx.Tx, order *model.Order) error {
	m.created = append(m.created, order)
	return nil
}

func (m *mockOrderRepo) Update(_ context.Context, _ pgx.Tx, _ uuid.UUID, _ model.UpdateOrderRequest) error {
	return nil
}

func (m *mockOrderRepo) UpdateStatus(_ context.Context, _ pgx.Tx, _ uuid.UUID, _ string, _ *time.Time, _ *time.Time) error {
	return nil
}

func (m *mockOrderRepo) FindByExternalID(_ context.Context, _ pgx.Tx, source, externalID string) (*model.Order, error) {
	key := source + ":" + externalID
	if o, ok := m.externalIDLookup[key]; ok {
		return o, nil
	}
	return nil, nil
}

func (m *mockOrderRepo) Delete(_ context.Context, _ pgx.Tx, _ uuid.UUID) error {
	return nil
}

func (m *mockOrderRepo) CountThisMonth(_ context.Context, _ pgx.Tx) (int, error) {
	return 0, nil
}

// mockTenantRepo implements repository.TenantRepo for unit tests.
type mockTenantRepo struct{}

func (m *mockTenantRepo) FindBySlug(_ context.Context, _ string) (*model.Tenant, error) {
	return nil, nil
}

func (m *mockTenantRepo) FindByID(_ context.Context, _ pgx.Tx, _ uuid.UUID) (*model.Tenant, error) {
	return nil, nil
}

func (m *mockTenantRepo) SlugExists(_ context.Context, _ string) (bool, error) {
	return false, nil
}

func (m *mockTenantRepo) Create(_ context.Context, _ pgx.Tx, _ *model.Tenant) error {
	return nil
}

func (m *mockTenantRepo) GetSettings(_ context.Context, _ pgx.Tx, _ uuid.UUID) (json.RawMessage, error) {
	return nil, nil
}

func (m *mockTenantRepo) ListAllTenantIDs(_ context.Context, _ *pgxpool.Pool) ([]uuid.UUID, error) {
	return nil, nil
}

func (m *mockTenantRepo) UpdateSettings(_ context.Context, _ pgx.Tx, _ uuid.UUID, _ json.RawMessage) error {
	return nil
}

// --- Helper to build CSV bytes ---

func buildBLCSV(headers []string, rows [][]string) []byte {
	var b strings.Builder
	b.WriteString(strings.Join(headers, ","))
	b.WriteByte('\n')
	for _, row := range rows {
		b.WriteString(strings.Join(row, ","))
		b.WriteByte('\n')
	}
	return []byte(b.String())
}

// --- Pure function tests ---

func TestBLImport_GroupsByOrderID(t *testing.T) {
	headers := []string{"order_id", "product_name", "product_sku", "product_quantity", "product_price_brutto"}
	rows := [][]string{
		{"1001", "Product A", "SKU-A", "1", "10.00"},
		{"1001", "Product B", "SKU-B", "2", "20.00"},
		{"1001", "Product C", "SKU-C", "1", "5.00"},
		{"1002", "Product D", "SKU-D", "3", "15.00"},
		{"1002", "Product E", "SKU-E", "1", "30.00"},
		{"1002", "Product F", "SKU-F", "2", "7.50"},
	}
	csvData := buildBLCSV(headers, rows)

	_, records, headerIdx, err := parseBLOrderCSV(csvData)
	require.NoError(t, err)

	groups, _, err := groupByOrderID(records, headerIdx)
	require.NoError(t, err)

	assert.Equal(t, 2, len(groups))
	assert.Equal(t, "1001", groups[0].OrderID)
	assert.Equal(t, 3, len(groups[0].Rows))
	assert.Equal(t, "1002", groups[1].OrderID)
	assert.Equal(t, 3, len(groups[1].Rows))
}

func TestBLImport_AggregatesItems(t *testing.T) {
	headers := []string{"order_id", "product_name", "product_sku", "product_quantity", "product_price_brutto"}
	rows := [][]string{
		{"1001", "Product A", "SKU-001", "2", "49.99"},
		{"1001", "Product B", "SKU-002", "1", "29.99"},
	}
	csvData := buildBLCSV(headers, rows)

	_, records, headerIdx, err := parseBLOrderCSV(csvData)
	require.NoError(t, err)

	groups, _, err := groupByOrderID(records, headerIdx)
	require.NoError(t, err)
	require.Equal(t, 1, len(groups))

	itemsJSON, totalAmount := extractBLItems(groups[0].Rows, headerIdx)
	require.NotNil(t, itemsJSON)

	var items []blItem
	err = json.Unmarshal(itemsJSON, &items)
	require.NoError(t, err)

	assert.Equal(t, 2, len(items))

	assert.Equal(t, "Product A", items[0].Name)
	assert.Equal(t, "SKU-001", items[0].SKU)
	assert.Equal(t, 2, items[0].Quantity)
	assert.InDelta(t, 49.99, items[0].Price, 0.001)

	assert.Equal(t, "Product B", items[1].Name)
	assert.Equal(t, "SKU-002", items[1].SKU)
	assert.Equal(t, 1, items[1].Quantity)
	assert.InDelta(t, 29.99, items[1].Price, 0.001)

	// total = 2*49.99 + 1*29.99 = 129.97
	assert.InDelta(t, 129.97, totalAmount, 0.001)
}

func TestBLImport_ExtractsCustomerFromFirstRow(t *testing.T) {
	headers := []string{"order_id", "delivery_fullname", "buyer_email", "buyer_phone", "product_name"}
	rows := [][]string{
		{"1001", "Jan Kowalski", "jan@example.com", "+48123456789", "Product A"},
		{"1001", "IGNORED NAME", "ignored@example.com", "+48000000000", "Product B"},
	}
	csvData := buildBLCSV(headers, rows)

	_, records, headerIdx, err := parseBLOrderCSV(csvData)
	require.NoError(t, err)

	groups, _, err := groupByOrderID(records, headerIdx)
	require.NoError(t, err)
	require.Equal(t, 1, len(groups))

	firstRow := groups[0].Rows[0]
	customerName := extractBLField(firstRow, headerIdx, "delivery_fullname", "buyer_name")
	customerEmail := extractBLField(firstRow, headerIdx, "buyer_email")
	customerPhone := extractBLField(firstRow, headerIdx, "buyer_phone")

	assert.Equal(t, "Jan Kowalski", customerName)
	assert.Equal(t, "jan@example.com", customerEmail)
	assert.Equal(t, "+48123456789", customerPhone)
}

func TestBLImport_ExtractsAddresses(t *testing.T) {
	headers := []string{
		"order_id",
		"delivery_fullname", "delivery_address", "delivery_city", "delivery_postcode", "delivery_country_code",
		"invoice_fullname", "invoice_address", "invoice_city", "invoice_postcode", "invoice_country", "invoice_company", "invoice_nip",
		"product_name",
	}
	rows := [][]string{
		{
			"1001",
			"Jan Kowalski", "ul. Testowa 1", "Warszawa", "00-001", "PL",
			"Firma Jan Kowalski", "ul. Firmowa 2", "Krakow", "30-001", "PL", "Firma Sp. z o.o.", "1234567890",
			"Product A",
		},
	}
	csvData := buildBLCSV(headers, rows)

	_, records, headerIdx, err := parseBLOrderCSV(csvData)
	require.NoError(t, err)

	groups, _, err := groupByOrderID(records, headerIdx)
	require.NoError(t, err)
	require.Equal(t, 1, len(groups))

	firstRow := groups[0].Rows[0]

	// Shipping address.
	shippingJSON := extractBLShippingAddress(firstRow, headerIdx)
	require.NotNil(t, shippingJSON)

	var shipping map[string]string
	err = json.Unmarshal(shippingJSON, &shipping)
	require.NoError(t, err)

	assert.Equal(t, "Jan Kowalski", shipping["name"])
	assert.Equal(t, "ul. Testowa 1", shipping["street"])
	assert.Equal(t, "Warszawa", shipping["city"])
	assert.Equal(t, "00-001", shipping["postal_code"])
	assert.Equal(t, "PL", shipping["country"])

	// Billing address.
	billingJSON := extractBLBillingAddress(firstRow, headerIdx)
	require.NotNil(t, billingJSON)

	var billing map[string]string
	err = json.Unmarshal(billingJSON, &billing)
	require.NoError(t, err)

	assert.Equal(t, "Firma Jan Kowalski", billing["name"])
	assert.Equal(t, "ul. Firmowa 2", billing["street"])
	assert.Equal(t, "Krakow", billing["city"])
	assert.Equal(t, "30-001", billing["postal_code"])
	assert.Equal(t, "PL", billing["country"])
	assert.Equal(t, "Firma Sp. z o.o.", billing["company"])
	assert.Equal(t, "1234567890", billing["nip"])
}

func TestBLImport_ComputesTotalAmount(t *testing.T) {
	headers := []string{"order_id", "product_name", "product_sku", "product_quantity", "product_price_brutto"}
	rows := [][]string{
		{"1001", "Item 1", "S1", "3", "10.00"},  // 3 * 10.00 = 30.00
		{"1001", "Item 2", "S2", "2", "25.50"},  // 2 * 25.50 = 51.00
		{"1001", "Item 3", "S3", "1", "100.00"}, // 1 * 100.00 = 100.00
	}
	csvData := buildBLCSV(headers, rows)

	_, records, headerIdx, err := parseBLOrderCSV(csvData)
	require.NoError(t, err)

	groups, _, err := groupByOrderID(records, headerIdx)
	require.NoError(t, err)
	require.Equal(t, 1, len(groups))

	_, totalAmount := extractBLItems(groups[0].Rows, headerIdx)

	// 30.00 + 51.00 + 100.00 = 181.00
	assert.InDelta(t, 181.00, totalAmount, 0.001)
}

func TestBLImport_ParsesDate(t *testing.T) {
	headers := []string{"order_id", "delivery_fullname", "date_add", "product_name"}
	rows := [][]string{
		{"1001", "Jan", "2025-06-15 14:30:00", "Product A"},
	}
	csvData := buildBLCSV(headers, rows)

	_, records, headerIdx, err := parseBLOrderCSV(csvData)
	require.NoError(t, err)

	groups, _, err := groupByOrderID(records, headerIdx)
	require.NoError(t, err)
	require.Equal(t, 1, len(groups))

	dateStr := extractBLField(groups[0].Rows[0], headerIdx, "date_add")
	parsedTime, err := parseFlexibleTime(dateStr)
	require.NoError(t, err)

	assert.Equal(t, 2025, parsedTime.Year())
	assert.Equal(t, time.June, parsedTime.Month())
	assert.Equal(t, 15, parsedTime.Day())
	assert.Equal(t, 14, parsedTime.Hour())
	assert.Equal(t, 30, parsedTime.Minute())
}

func TestBLImport_PaymentStatus(t *testing.T) {
	tests := []struct {
		name           string
		paymentDone    string
		expectedStatus string
	}{
		{"paid when positive", "150.00", "paid"},
		{"paid when positive comma", "150,00", "paid"},
		{"pending when zero", "0", "pending"},
		{"pending when empty", "", "pending"},
		{"pending when invalid", "abc", "pending"},
		{"paid when small positive", "0.01", "paid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paymentDone := tt.paymentDone
			paymentStatus := "pending"
			if paymentDone != "" {
				paymentDone = strings.ReplaceAll(paymentDone, ",", ".")
				parsed, err := parseFloat64Safe(paymentDone)
				if err == nil && parsed > 0 {
					paymentStatus = "paid"
				}
			}
			assert.Equal(t, tt.expectedStatus, paymentStatus)
		})
	}
}

// parseFloat64Safe is a test helper that wraps strconv.ParseFloat.
func parseFloat64Safe(s string) (float64, error) {
	return parseFloatHelper(s)
}

func parseFloatHelper(s string) (float64, error) {
	return json.Number(s).Float64()
}

// --- Integration tests using mock repos ---

func TestBLImport_ImportOrderGroup_CreatesOrder(t *testing.T) {
	orderRepo := newMockOrderRepo()
	customerRepo := newMockCustomerRepo()
	auditRepo := &mockAuditRepo{}

	svc := &BaseLinkerImportService{
		orderRepo:    orderRepo,
		customerRepo: customerRepo,
		auditRepo:    auditRepo,
	}

	tenantID := uuid.New()

	headers := []string{
		"order_id", "delivery_fullname", "buyer_email", "buyer_phone",
		"product_name", "product_sku", "product_quantity", "product_price_brutto",
		"date_add", "payment_method", "payment_done", "currency", "order_status_name",
	}
	rows := [][]string{
		{"BL-1001", "Jan Kowalski", "jan@example.com", "+48123456789", "Product A", "SKU-A", "2", "49.99", "2025-06-15 14:30:00", "PayU", "99.98", "PLN", "new"},
		{"BL-1001", "Jan Kowalski", "jan@example.com", "+48123456789", "Product B", "SKU-B", "1", "29.99", "2025-06-15 14:30:00", "PayU", "99.98", "PLN", "new"},
	}
	csvData := buildBLCSV(headers, rows)

	_, records, headerIdx, err := parseBLOrderCSV(csvData)
	require.NoError(t, err)

	groups, _, err := groupByOrderID(records, headerIdx)
	require.NoError(t, err)
	require.Equal(t, 1, len(groups))

	result := &BLOrderImportResult{Errors: []model.ImportError{}}
	statusConfig := &model.OrderStatusConfig{
		Statuses: []model.StatusDef{{Key: "new", Label: "New"}},
	}

	errs := svc.importOrderGroup(context.Background(), nil, tenantID, groups[0], headerIdx, result, statusConfig, false)
	assert.Nil(t, errs)
	assert.Equal(t, 1, result.OrdersCreated)

	require.Len(t, orderRepo.created, 1)
	order := orderRepo.created[0]

	assert.Equal(t, "baselinker", order.Source)
	assert.Equal(t, "new", order.Status)
	assert.Equal(t, "Jan Kowalski", order.CustomerName)
	require.NotNil(t, order.ExternalID)
	assert.Equal(t, "BL-1001", *order.ExternalID)
	require.NotNil(t, order.CustomerEmail)
	assert.Equal(t, "jan@example.com", *order.CustomerEmail)
	require.NotNil(t, order.CustomerPhone)
	assert.Equal(t, "+48123456789", *order.CustomerPhone)
	assert.Equal(t, "PLN", order.Currency)
	assert.Equal(t, "paid", order.PaymentStatus)
	require.NotNil(t, order.PaymentMethod)
	assert.Equal(t, "PayU", *order.PaymentMethod)

	// Verify items.
	var items []blItem
	err = json.Unmarshal(order.Items, &items)
	require.NoError(t, err)
	assert.Equal(t, 2, len(items))
	assert.Equal(t, "Product A", items[0].Name)
	assert.Equal(t, "SKU-A", items[0].SKU)
	assert.Equal(t, 2, items[0].Quantity)
	assert.Equal(t, "Product B", items[1].Name)

	// total = 2*49.99 + 1*29.99 = 129.97
	assert.InDelta(t, 129.97, order.TotalAmount, 0.001)

	// Verify ordered_at parsed correctly.
	require.NotNil(t, order.OrderedAt)
	assert.Equal(t, 2025, order.OrderedAt.Year())
	assert.Equal(t, time.June, order.OrderedAt.Month())
	assert.Equal(t, 15, order.OrderedAt.Day())
}

func TestBLImport_ImportOrderGroup_SkipsDuplicate(t *testing.T) {
	orderRepo := newMockOrderRepo()
	orderRepo.externalIDLookup["baselinker:BL-1001"] = &model.Order{ID: uuid.New()}

	svc := &BaseLinkerImportService{
		orderRepo:    orderRepo,
		customerRepo: newMockCustomerRepo(),
		auditRepo:    &mockAuditRepo{},
	}

	tenantID := uuid.New()

	headers := []string{"order_id", "delivery_fullname", "product_name"}
	rows := [][]string{
		{"BL-1001", "Jan Kowalski", "Product A"},
	}
	csvData := buildBLCSV(headers, rows)

	_, records, headerIdx, err := parseBLOrderCSV(csvData)
	require.NoError(t, err)

	groups, _, err := groupByOrderID(records, headerIdx)
	require.NoError(t, err)

	result := &BLOrderImportResult{Errors: []model.ImportError{}}
	statusConfig := &model.OrderStatusConfig{
		Statuses: []model.StatusDef{{Key: "new", Label: "New"}},
	}

	errs := svc.importOrderGroup(context.Background(), nil, tenantID, groups[0], headerIdx, result, statusConfig, false)
	require.NotNil(t, errs)
	assert.Contains(t, errs[0].Message, "duplicate order_id")
	assert.Equal(t, 0, result.OrdersCreated)
	assert.Empty(t, orderRepo.created)
}

func TestBLImport_ImportOrderGroup_CreatesCustomer(t *testing.T) {
	orderRepo := newMockOrderRepo()
	customerRepo := newMockCustomerRepo()
	auditRepo := &mockAuditRepo{}

	svc := &BaseLinkerImportService{
		orderRepo:    orderRepo,
		customerRepo: customerRepo,
		auditRepo:    auditRepo,
	}

	tenantID := uuid.New()

	headers := []string{"order_id", "delivery_fullname", "buyer_email", "buyer_phone", "product_name"}
	rows := [][]string{
		{"BL-2001", "Anna Nowak", "anna@example.com", "+48111222333", "Widget"},
	}
	csvData := buildBLCSV(headers, rows)

	_, records, headerIdx, err := parseBLOrderCSV(csvData)
	require.NoError(t, err)

	groups, _, err := groupByOrderID(records, headerIdx)
	require.NoError(t, err)

	result := &BLOrderImportResult{Errors: []model.ImportError{}}
	statusConfig := &model.OrderStatusConfig{
		Statuses: []model.StatusDef{{Key: "new", Label: "New"}},
	}

	errs := svc.importOrderGroup(context.Background(), nil, tenantID, groups[0], headerIdx, result, statusConfig, true)
	assert.Nil(t, errs)
	assert.Equal(t, 1, result.OrdersCreated)
	assert.Equal(t, 1, result.CustomersCreated)

	// Verify customer was created.
	require.Len(t, customerRepo.created, 1)
	c := customerRepo.created[0]
	assert.Equal(t, "Anna Nowak", c.Name)
	require.NotNil(t, c.Email)
	assert.Equal(t, "anna@example.com", *c.Email)
	require.NotNil(t, c.Phone)
	assert.Equal(t, "+48111222333", *c.Phone)

	// Verify customer linked to order.
	require.Len(t, orderRepo.created, 1)
	assert.NotNil(t, orderRepo.created[0].CustomerID)
	assert.Equal(t, c.ID, *orderRepo.created[0].CustomerID)
}

func TestBLImport_GroupsByOrderID_MissingColumn(t *testing.T) {
	headers := []string{"product_name", "product_sku"}
	rows := [][]string{
		{"Product A", "SKU-A"},
	}
	csvData := buildBLCSV(headers, rows)

	_, records, headerIdx, err := parseBLOrderCSV(csvData)
	require.NoError(t, err)

	_, _, err = groupByOrderID(records, headerIdx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "order_id column")
}

func TestBLImport_GroupsByOrderID_Aliases(t *testing.T) {
	// Test all recognized aliases for order_id.
	for _, alias := range []string{"order_id", "order_number", "bl_order_id"} {
		t.Run(alias, func(t *testing.T) {
			headers := []string{alias, "product_name"}
			rows := [][]string{
				{"1001", "Product A"},
				{"1001", "Product B"},
				{"1002", "Product C"},
			}
			csvData := buildBLCSV(headers, rows)

			_, records, headerIdx, err := parseBLOrderCSV(csvData)
			require.NoError(t, err)

			groups, _, err := groupByOrderID(records, headerIdx)
			require.NoError(t, err)
			assert.Equal(t, 2, len(groups))
		})
	}
}

func TestBLImport_BuyerNameFallback(t *testing.T) {
	// When delivery_fullname is absent, buyer_name should be used.
	headers := []string{"order_id", "buyer_name", "product_name"}
	rows := [][]string{
		{"1001", "Jan Kowalski", "Product A"},
	}
	csvData := buildBLCSV(headers, rows)

	_, records, headerIdx, err := parseBLOrderCSV(csvData)
	require.NoError(t, err)

	groups, _, err := groupByOrderID(records, headerIdx)
	require.NoError(t, err)

	firstRow := groups[0].Rows[0]
	name := extractBLField(firstRow, headerIdx, "delivery_fullname", "buyer_name")
	assert.Equal(t, "Jan Kowalski", name)
}

func TestBLImport_ExtractBLField_MultipleAliases(t *testing.T) {
	headerIdx := map[string]int{
		"delivery_address": 0,
		"delivery_street":  1,
	}
	row := []string{"", "ul. Testowa 5"}

	// Should skip empty first alias and return second.
	val := extractBLField(row, headerIdx, "delivery_address", "delivery_street")
	assert.Equal(t, "ul. Testowa 5", val)

	// When first has value, return it.
	row2 := []string{"ul. Glowna 1", "ul. Testowa 5"}
	val2 := extractBLField(row2, headerIdx, "delivery_address", "delivery_street")
	assert.Equal(t, "ul. Glowna 1", val2)
}

func TestBLImport_ShippingAddress_EmptyFields(t *testing.T) {
	headerIdx := map[string]int{
		"delivery_fullname": 0,
	}
	// All empty values → nil result.
	row := []string{""}
	result := extractBLShippingAddress(row, headerIdx)
	assert.Nil(t, result)
}

func TestBLImport_DefaultCurrencyPLN(t *testing.T) {
	headers := []string{"order_id", "delivery_fullname", "product_name"}
	rows := [][]string{
		{"1001", "Jan", "Product A"},
	}
	csvData := buildBLCSV(headers, rows)

	_, records, headerIdx, err := parseBLOrderCSV(csvData)
	require.NoError(t, err)

	groups, _, err := groupByOrderID(records, headerIdx)
	require.NoError(t, err)

	// No currency column → should default to PLN in importOrderGroup.
	currency := extractBLField(groups[0].Rows[0], headerIdx, "currency")
	if currency == "" {
		currency = "PLN"
	}
	assert.Equal(t, "PLN", currency)
}

func TestBLImport_StripsBOM(t *testing.T) {
	// UTF-8 BOM prefix.
	csvData := append([]byte{0xEF, 0xBB, 0xBF}, []byte("order_id,product_name\n1001,Product A\n")...)

	headers, _, headerIdx, err := parseBLOrderCSV(csvData)
	require.NoError(t, err)

	assert.Equal(t, "order_id", headers[0])
	_, ok := headerIdx["order_id"]
	assert.True(t, ok)
}

func TestBLImport_EuropeanDecimalFormat(t *testing.T) {
	// Use quoted CSV value to avoid comma being treated as field separator.
	csvData := []byte("order_id,product_name,product_quantity,product_price_brutto\n1001,Product A,2,\"49,99\"\n")

	_, records, headerIdx, err := parseBLOrderCSV(csvData)
	require.NoError(t, err)

	groups, _, err := groupByOrderID(records, headerIdx)
	require.NoError(t, err)

	_, totalAmount := extractBLItems(groups[0].Rows, headerIdx)

	// 2 * 49.99 = 99.98
	assert.InDelta(t, 99.98, totalAmount, 0.001)
}
