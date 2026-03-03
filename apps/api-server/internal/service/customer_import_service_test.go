package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// --- Mocks ---

// mockCustomerRepo implements repository.CustomerRepo for unit tests.
type mockCustomerRepo struct {
	customers    map[string]*model.Customer // keyed by email
	created      []*model.Customer
	updated      []mockUpdateCall
	findByIDFunc func(id uuid.UUID) (*model.Customer, error)
}

type mockUpdateCall struct {
	ID  uuid.UUID
	Req model.UpdateCustomerRequest
}

func newMockCustomerRepo() *mockCustomerRepo {
	return &mockCustomerRepo{
		customers: make(map[string]*model.Customer),
	}
}

func (m *mockCustomerRepo) List(_ context.Context, _ pgx.Tx, _ model.CustomerListFilter) ([]model.Customer, int, error) {
	return nil, 0, nil
}

func (m *mockCustomerRepo) FindByID(_ context.Context, _ pgx.Tx, id uuid.UUID) (*model.Customer, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(id)
	}
	return nil, nil
}

func (m *mockCustomerRepo) FindByEmail(_ context.Context, _ pgx.Tx, email string) (*model.Customer, error) {
	if c, ok := m.customers[email]; ok {
		return c, nil
	}
	return nil, nil
}

func (m *mockCustomerRepo) Create(_ context.Context, _ pgx.Tx, customer *model.Customer) error {
	m.created = append(m.created, customer)
	return nil
}

func (m *mockCustomerRepo) Update(_ context.Context, _ pgx.Tx, id uuid.UUID, req model.UpdateCustomerRequest) error {
	m.updated = append(m.updated, mockUpdateCall{ID: id, Req: req})
	return nil
}

func (m *mockCustomerRepo) Delete(_ context.Context, _ pgx.Tx, _ uuid.UUID) error {
	return nil
}

func (m *mockCustomerRepo) IncrementOrderStats(_ context.Context, _ pgx.Tx, _ uuid.UUID, _ float64) error {
	return nil
}

func (m *mockCustomerRepo) ListOrdersByCustomerID(_ context.Context, _ pgx.Tx, _ uuid.UUID, _ model.OrderListFilter) ([]model.Order, int, error) {
	return nil, 0, nil
}

// mockAuditRepo implements repository.AuditRepo for unit tests.
type mockAuditRepo struct {
	entries []model.AuditEntry
}

func (m *mockAuditRepo) Log(_ context.Context, _ pgx.Tx, entry model.AuditEntry) error {
	m.entries = append(m.entries, entry)
	return nil
}

func (m *mockAuditRepo) ListByEntity(_ context.Context, _ pgx.Tx, _ string, _ uuid.UUID) ([]model.AuditLogEntry, error) {
	return nil, nil
}

func (m *mockAuditRepo) List(_ context.Context, _ pgx.Tx, _ model.AuditListFilter) ([]model.AuditLogEntry, int, error) {
	return nil, 0, nil
}

// --- CSV parsing tests ---

func TestParseCustomerCSV_StandardHeaders(t *testing.T) {
	csv := []byte("name,email,phone,company_name,nip,tags,notes\nJan Kowalski,jan@example.com,+48123456789,Firma Sp. z o.o.,1234567890,\"vip,hurtowy\",Notatka\n")

	headers, records, fieldIdx, err := parseCustomerCSV(csv)
	require.NoError(t, err)

	assert.Equal(t, []string{"name", "email", "phone", "company_name", "nip", "tags", "notes"}, headers)
	assert.Equal(t, 2, len(records)) // header + 1 data row
	assert.Equal(t, 0, fieldIdx["name"])
	assert.Equal(t, 1, fieldIdx["email"])
	assert.Equal(t, 2, fieldIdx["phone"])
	assert.Equal(t, 3, fieldIdx["company_name"])
	assert.Equal(t, 4, fieldIdx["nip"])
	assert.Equal(t, 5, fieldIdx["tags"])
	assert.Equal(t, 6, fieldIdx["notes"])
}

func TestParseCustomerCSV_BaseLinkerAliases(t *testing.T) {
	csv := []byte("buyer_name,buyer_email,buyer_phone,invoice_company,invoice_nip\nAnna Nowak,anna@test.pl,+48111222333,FirmaBL,9876543210\n")

	_, _, fieldIdx, err := parseCustomerCSV(csv)
	require.NoError(t, err)

	// BL aliases should resolve to canonical fields.
	assert.Equal(t, 0, fieldIdx["name"])
	assert.Equal(t, 1, fieldIdx["email"])
	assert.Equal(t, 2, fieldIdx["phone"])
	assert.Equal(t, 3, fieldIdx["company_name"])
	assert.Equal(t, 4, fieldIdx["nip"])
}

func TestParseCustomerCSV_CustomerNameAlias(t *testing.T) {
	csv := []byte("customer_name,customer_email,customer_phone\nTest,test@x.com,111\n")

	_, _, fieldIdx, err := parseCustomerCSV(csv)
	require.NoError(t, err)

	assert.Equal(t, 0, fieldIdx["name"])
	assert.Equal(t, 1, fieldIdx["email"])
	assert.Equal(t, 2, fieldIdx["phone"])
}

func TestParseCustomerCSV_DeliveryFullname(t *testing.T) {
	csv := []byte("delivery_fullname,email\nMaria Wisniewska,maria@example.com\n")

	_, _, fieldIdx, err := parseCustomerCSV(csv)
	require.NoError(t, err)

	assert.Equal(t, 0, fieldIdx["name"])
	assert.Equal(t, 1, fieldIdx["email"])
}

func TestParseCustomerCSV_EmptyFile(t *testing.T) {
	csv := []byte("")

	_, _, _, err := parseCustomerCSV(csv)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
}

func TestParseCustomerCSV_StripsBOM(t *testing.T) {
	// UTF-8 BOM prefix + standard CSV.
	csv := append([]byte{0xEF, 0xBB, 0xBF}, []byte("name,email\nTest,test@x.com\n")...)

	headers, _, fieldIdx, err := parseCustomerCSV(csv)
	require.NoError(t, err)

	assert.Equal(t, "name", headers[0])
	assert.Equal(t, 0, fieldIdx["name"])
}

func TestParseCustomerCSV_FirstAliasWins(t *testing.T) {
	// If CSV has both "name" and "customer_name", the first one (by column order) should win.
	csv := []byte("name,customer_name,email\nFirst,Second,e@x.com\n")

	_, records, fieldIdx, err := parseCustomerCSV(csv)
	require.NoError(t, err)

	// "name" at index 0 is mapped first; "customer_name" also maps to "name" but index 0 already set.
	assert.Equal(t, 0, fieldIdx["name"])
	assert.Equal(t, "First", extractField(records[1], fieldIdx, "name"))
}

func TestExtractField(t *testing.T) {
	row := []string{"  Jan Kowalski  ", "jan@example.com", "+48123"}
	fieldIdx := map[string]int{"name": 0, "email": 1, "phone": 2}

	assert.Equal(t, "Jan Kowalski", extractField(row, fieldIdx, "name"))
	assert.Equal(t, "jan@example.com", extractField(row, fieldIdx, "email"))
	assert.Equal(t, "+48123", extractField(row, fieldIdx, "phone"))
	assert.Equal(t, "", extractField(row, fieldIdx, "nonexistent"))
}

func TestExtractField_IndexOutOfRange(t *testing.T) {
	row := []string{"value"}
	fieldIdx := map[string]int{"name": 0, "email": 5}

	assert.Equal(t, "value", extractField(row, fieldIdx, "name"))
	assert.Equal(t, "", extractField(row, fieldIdx, "email")) // index 5 >= len(row)
}

func TestParseCustomerCSV_SpacesInHeaders(t *testing.T) {
	// Headers with spaces (e.g. "customer name", "company name") should be recognized.
	csv := []byte("customer name,customer email,customer phone,company name\nTest,t@x.com,111,Firma\n")

	_, _, fieldIdx, err := parseCustomerCSV(csv)
	require.NoError(t, err)

	assert.Equal(t, 0, fieldIdx["name"])
	assert.Equal(t, 1, fieldIdx["email"])
	assert.Equal(t, 2, fieldIdx["phone"])
	assert.Equal(t, 3, fieldIdx["company_name"])
}

func TestParseCustomerCSV_TaxIdAlias(t *testing.T) {
	csv := []byte("name,tax_id\nTest,1234567890\n")

	_, records, fieldIdx, err := parseCustomerCSV(csv)
	require.NoError(t, err)

	assert.Equal(t, 1, fieldIdx["nip"])
	assert.Equal(t, "1234567890", extractField(records[1], fieldIdx, "nip"))
}

func TestParseCustomerCSV_CompanyAlias(t *testing.T) {
	csv := []byte("name,company\nTest,My Company\n")

	_, records, fieldIdx, err := parseCustomerCSV(csv)
	require.NoError(t, err)

	assert.Equal(t, 1, fieldIdx["company_name"])
	assert.Equal(t, "My Company", extractField(records[1], fieldIdx, "company_name"))
}

func TestParseCustomerCSV_MultipleRows(t *testing.T) {
	csv := []byte("name,email\nAlice,alice@x.com\nBob,bob@x.com\nCharlie,charlie@x.com\n")

	_, records, _, err := parseCustomerCSV(csv)
	require.NoError(t, err)

	assert.Equal(t, 4, len(records)) // header + 3 data rows
}

func TestParseCustomerCSV_TagsParsing(t *testing.T) {
	// Verify the tag extraction logic from a real row.
	csv := []byte("name,tags\nTest,\"vip,hurtowy,premium\"\n")

	_, records, fieldIdx, err := parseCustomerCSV(csv)
	require.NoError(t, err)

	tagsStr := extractField(records[1], fieldIdx, "tags")
	assert.Equal(t, "vip,hurtowy,premium", tagsStr)
}

// --- importCustomerRow tests (mock-based) ---

func TestImportCustomerRow_CreatesNewCustomer(t *testing.T) {
	repo := newMockCustomerRepo()
	svc := &CustomerImportService{customerRepo: repo}

	tenantID := uuid.New()
	row := []string{"Jan Kowalski", "jan@example.com", "+48123456789", "Firma", "1234567890", "vip,premium", "Good customer"}
	fieldIdx := map[string]int{
		"name":         0,
		"email":        1,
		"phone":        2,
		"company_name": 3,
		"nip":          4,
		"tags":         5,
		"notes":        6,
	}
	result := &CustomerImportResult{Errors: []model.ImportError{}}

	rowErr := svc.importCustomerRow(context.Background(), nil, tenantID, row, fieldIdx, 1, result)

	assert.Nil(t, rowErr)
	assert.Equal(t, 1, result.Created)
	assert.Equal(t, 0, result.Updated)
	assert.Equal(t, 0, result.Skipped)

	// Verify the customer was created with correct fields.
	require.Len(t, repo.created, 1)
	c := repo.created[0]
	assert.Equal(t, "Jan Kowalski", c.Name)
	assert.Equal(t, tenantID, c.TenantID)
	assert.NotEqual(t, uuid.Nil, c.ID)
	require.NotNil(t, c.Email)
	assert.Equal(t, "jan@example.com", *c.Email)
	require.NotNil(t, c.Phone)
	assert.Equal(t, "+48123456789", *c.Phone)
	require.NotNil(t, c.CompanyName)
	assert.Equal(t, "Firma", *c.CompanyName)
	require.NotNil(t, c.NIP)
	assert.Equal(t, "1234567890", *c.NIP)
	assert.Equal(t, []string{"vip", "premium"}, c.Tags)
	require.NotNil(t, c.Notes)
	assert.Equal(t, "Good customer", *c.Notes)
}

func TestImportCustomerRow_UpdatesExistingByEmail(t *testing.T) {
	existingID := uuid.New()
	repo := newMockCustomerRepo()
	repo.customers["jan@example.com"] = &model.Customer{
		ID:   existingID,
		Name: "Old Name",
	}
	svc := &CustomerImportService{customerRepo: repo}

	row := []string{"Jan Kowalski Updated", "jan@example.com", "+48999888777", "New Firma"}
	fieldIdx := map[string]int{
		"name":         0,
		"email":        1,
		"phone":        2,
		"company_name": 3,
	}
	result := &CustomerImportResult{Errors: []model.ImportError{}}

	rowErr := svc.importCustomerRow(context.Background(), nil, uuid.New(), row, fieldIdx, 1, result)

	assert.Nil(t, rowErr)
	assert.Equal(t, 0, result.Created)
	assert.Equal(t, 1, result.Updated)
	assert.Equal(t, 0, result.Skipped)

	// Verify Update was called with the existing customer's ID.
	require.Len(t, repo.updated, 1)
	assert.Equal(t, existingID, repo.updated[0].ID)
	require.NotNil(t, repo.updated[0].Req.Name)
	assert.Equal(t, "Jan Kowalski Updated", *repo.updated[0].Req.Name)
	require.NotNil(t, repo.updated[0].Req.Phone)
	assert.Equal(t, "+48999888777", *repo.updated[0].Req.Phone)
	require.NotNil(t, repo.updated[0].Req.CompanyName)
	assert.Equal(t, "New Firma", *repo.updated[0].Req.CompanyName)

	// Create should not have been called.
	assert.Empty(t, repo.created)
}

func TestImportCustomerRow_SkipsEmptyName(t *testing.T) {
	repo := newMockCustomerRepo()
	svc := &CustomerImportService{customerRepo: repo}

	row := []string{"", "jan@example.com", "+48123"}
	fieldIdx := map[string]int{"name": 0, "email": 1, "phone": 2}
	result := &CustomerImportResult{Errors: []model.ImportError{}}

	rowErr := svc.importCustomerRow(context.Background(), nil, uuid.New(), row, fieldIdx, 1, result)

	require.NotNil(t, rowErr)
	assert.Equal(t, 1, rowErr.Row)
	assert.Equal(t, "name", rowErr.Field)
	assert.Contains(t, rowErr.Message, "name is required")
	assert.Equal(t, 1, result.Skipped)
	assert.Equal(t, 0, result.Created)
	assert.Equal(t, 0, result.Updated)

	// Neither Create nor Update should have been called.
	assert.Empty(t, repo.created)
	assert.Empty(t, repo.updated)
}

func TestImportCustomerRow_CreatesWhenNoEmail(t *testing.T) {
	repo := newMockCustomerRepo()
	svc := &CustomerImportService{customerRepo: repo}

	// Row with name but no email -- should create (no dedup possible).
	row := []string{"Jan Kowalski", ""}
	fieldIdx := map[string]int{"name": 0, "email": 1}
	result := &CustomerImportResult{Errors: []model.ImportError{}}

	rowErr := svc.importCustomerRow(context.Background(), nil, uuid.New(), row, fieldIdx, 1, result)

	assert.Nil(t, rowErr)
	assert.Equal(t, 1, result.Created)
	assert.Equal(t, 0, result.Updated)
	require.Len(t, repo.created, 1)
	assert.Equal(t, "Jan Kowalski", repo.created[0].Name)
	assert.Nil(t, repo.created[0].Email) // empty string → nil
}

func TestImportCustomerRow_BaseLinkerAliasesEndToEnd(t *testing.T) {
	repo := newMockCustomerRepo()
	svc := &CustomerImportService{customerRepo: repo}

	// Simulate a row from a CSV with BL headers, after alias resolution.
	csvData := []byte("buyer_name,buyer_email,buyer_phone,invoice_company,invoice_nip\nAnna Nowak,anna@test.pl,+48111222333,FirmaBL,9876543210\n")
	_, records, fieldIdx, err := parseCustomerCSV(csvData)
	require.NoError(t, err)

	result := &CustomerImportResult{Errors: []model.ImportError{}}
	rowErr := svc.importCustomerRow(context.Background(), nil, uuid.New(), records[1], fieldIdx, 1, result)

	assert.Nil(t, rowErr)
	assert.Equal(t, 1, result.Created)
	require.Len(t, repo.created, 1)
	c := repo.created[0]
	assert.Equal(t, "Anna Nowak", c.Name)
	require.NotNil(t, c.Email)
	assert.Equal(t, "anna@test.pl", *c.Email)
	require.NotNil(t, c.Phone)
	assert.Equal(t, "+48111222333", *c.Phone)
	require.NotNil(t, c.CompanyName)
	assert.Equal(t, "FirmaBL", *c.CompanyName)
	require.NotNil(t, c.NIP)
	assert.Equal(t, "9876543210", *c.NIP)
}
