package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

func TestParseCSV_BasicFile(t *testing.T) {
	csv := "customer_name,total_amount,currency\nJan Kowalski,100.50,PLN\nAnna Nowak,200,EUR\n"
	svc := &ImportService{}

	result, err := svc.ParseCSV(strings.NewReader(csv))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Headers) != 3 {
		t.Fatalf("expected 3 headers, got %d", len(result.Headers))
	}
	if result.Headers[0] != "customer_name" {
		t.Errorf("expected header 'customer_name', got '%s'", result.Headers[0])
	}
	if result.TotalRows != 2 {
		t.Errorf("expected 2 total rows, got %d", result.TotalRows)
	}
	if len(result.SampleRows) != 2 {
		t.Errorf("expected 2 sample rows, got %d", len(result.SampleRows))
	}

	// Check first sample row data
	row1 := result.SampleRows[0]
	if row1.Row != 1 {
		t.Errorf("expected row number 1, got %d", row1.Row)
	}
	if row1.Data["customer_name"] != "Jan Kowalski" {
		t.Errorf("expected 'Jan Kowalski', got '%v'", row1.Data["customer_name"])
	}
	if row1.Data["total_amount"] != "100.50" {
		t.Errorf("expected '100.50', got '%v'", row1.Data["total_amount"])
	}
}

func TestParseCSV_WithBOM(t *testing.T) {
	// UTF-8 BOM + CSV data
	csv := "\xEF\xBB\xBFcustomer_name,total_amount\nTest User,50\n"
	svc := &ImportService{}

	result, err := svc.ParseCSV(strings.NewReader(csv))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Headers[0] != "customer_name" {
		t.Errorf("expected 'customer_name' after BOM strip, got '%s'", result.Headers[0])
	}
	if result.TotalRows != 1 {
		t.Errorf("expected 1 total row, got %d", result.TotalRows)
	}
}

func TestParseCSV_EmptyFile(t *testing.T) {
	svc := &ImportService{}
	_, err := svc.ParseCSV(strings.NewReader(""))
	if err == nil {
		t.Fatal("expected error for empty file")
	}
}

func TestParseCSV_HeaderOnly(t *testing.T) {
	csv := "customer_name,total_amount\n"
	svc := &ImportService{}

	result, err := svc.ParseCSV(strings.NewReader(csv))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.TotalRows != 0 {
		t.Errorf("expected 0 total rows, got %d", result.TotalRows)
	}
	if len(result.SampleRows) != 0 {
		t.Errorf("expected 0 sample rows, got %d", len(result.SampleRows))
	}
}

func TestParseCSV_SampleRowsLimitedTo10(t *testing.T) {
	var sb strings.Builder
	sb.WriteString("name\n")
	for range 25 {
		sb.WriteString("row\n")
	}
	svc := &ImportService{}

	result, err := svc.ParseCSV(strings.NewReader(sb.String()))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.TotalRows != 25 {
		t.Errorf("expected 25 total rows, got %d", result.TotalRows)
	}
	if len(result.SampleRows) != 10 {
		t.Errorf("expected 10 sample rows (capped), got %d", len(result.SampleRows))
	}
}

func TestAutoDetectMappings(t *testing.T) {
	headers := []string{"customer_name", "Email", "Total Amount", "unknown_col", "tags"}
	mappings := autoDetectMappings(headers)

	found := make(map[string]string)
	for _, m := range mappings {
		found[m.CSVColumn] = m.OrderField
	}

	if found["customer_name"] != "customer_name" {
		t.Errorf("expected customer_name mapping, got '%s'", found["customer_name"])
	}
	if found["Email"] != "customer_email" {
		t.Errorf("expected customer_email mapping for 'Email', got '%s'", found["Email"])
	}
	if found["tags"] != "tags" {
		t.Errorf("expected tags mapping, got '%s'", found["tags"])
	}
	if _, ok := found["unknown_col"]; ok {
		t.Error("expected no mapping for 'unknown_col'")
	}
}

func TestParseFlexibleTime(t *testing.T) {
	tests := []struct {
		input string
		valid bool
	}{
		{"2024-01-15T10:30:00Z", true},
		{"2024-01-15T10:30:00", true},
		{"2024-01-15 10:30:00", true},
		{"2024-01-15", true},
		{"15.01.2024", true},
		{"15-01-2024", true},
		{"not-a-date", false},
	}

	for _, tt := range tests {
		_, err := parseFlexibleTime(tt.input)
		if tt.valid && err != nil {
			t.Errorf("expected valid date for '%s', got error: %v", tt.input, err)
		}
		if !tt.valid && err == nil {
			t.Errorf("expected error for '%s', got nil", tt.input)
		}
	}
}

func TestStripBOM(t *testing.T) {
	bom := []byte{0xEF, 0xBB, 0xBF}
	data := append(bom, []byte("hello")...) //nolint:gocritic // bom is intentionally not reused

	result := stripBOM(data)
	if string(result) != "hello" {
		t.Errorf("expected 'hello', got '%s'", string(result))
	}

	noBom := []byte("hello")
	result2 := stripBOM(noBom)
	if string(result2) != "hello" {
		t.Errorf("expected 'hello' (no change), got '%s'", string(result2))
	}
}

func TestImportService_EnforceMonthlyOrderLimit_AllowsCreateThatReachesLimit(t *testing.T) {
	tx := &importLimitTx{}
	orderCounter := importLimitOrderCounter{count: 8}

	err := EnforceMonthlyOrderLimit(context.Background(), tx, orderCounter, uuid.New(), 10, 1)

	require.NoError(t, err)
	require.True(t, tx.lockedTenantRow)
}

func TestImportService_EnforceMonthlyOrderLimit_RejectsWhenNextCreateWouldExceedLimit(t *testing.T) {
	tx := &importLimitTx{}
	orderCounter := importLimitOrderCounter{count: 9}

	err := EnforceMonthlyOrderLimit(context.Background(), tx, orderCounter, uuid.New(), 10, 1)

	require.ErrorIs(t, err, ErrOrderLimitExceeded)
	require.True(t, tx.lockedTenantRow)
}

func TestImportService_EnforceMonthlyOrderLimit_PropagatesLockError(t *testing.T) {
	lockErr := errors.New("lock timeout")
	tx := &importLimitTx{execErr: lockErr}
	orderCounter := importLimitOrderCounter{count: 5}

	err := EnforceMonthlyOrderLimit(context.Background(), tx, orderCounter, uuid.New(), 10, 0)

	require.Error(t, err)
	require.ErrorIs(t, err, lockErr)
}

func TestImportService_EnforceMonthlyOrderLimit_PropagatesCountError(t *testing.T) {
	countErr := errors.New("database timeout")
	tx := &importLimitTx{}
	orderCounter := importLimitOrderCounter{countErr: countErr}

	err := EnforceMonthlyOrderLimit(context.Background(), tx, orderCounter, uuid.New(), 10, 0)

	require.Error(t, err)
	require.ErrorIs(t, err, countErr)
	require.True(t, tx.lockedTenantRow)
}

type importLimitOrderCounter struct {
	count    int
	countErr error
}

func (r importLimitOrderCounter) CountThisMonth(context.Context, pgx.Tx) (int, error) {
	return r.count, r.countErr
}

type importLimitTx struct {
	lockedTenantRow bool
	execErr         error
}

func (tx *importLimitTx) Begin(context.Context) (pgx.Tx, error) {
	return nil, nil
}

func (tx *importLimitTx) Commit(context.Context) error {
	return nil
}

func (tx *importLimitTx) Rollback(context.Context) error {
	return nil
}

func (tx *importLimitTx) CopyFrom(context.Context, pgx.Identifier, []string, pgx.CopyFromSource) (int64, error) {
	return 0, nil
}

func (tx *importLimitTx) SendBatch(context.Context, *pgx.Batch) pgx.BatchResults {
	return nil
}

func (tx *importLimitTx) LargeObjects() pgx.LargeObjects {
	return pgx.LargeObjects{}
}

func (tx *importLimitTx) Prepare(context.Context, string, string) (*pgconn.StatementDescription, error) {
	return nil, nil
}

func (tx *importLimitTx) Exec(_ context.Context, sql string, _ ...any) (pgconn.CommandTag, error) {
	if strings.Contains(sql, "SELECT 1 FROM tenants WHERE id = $1 FOR UPDATE") {
		tx.lockedTenantRow = true
	}
	return pgconn.CommandTag{}, tx.execErr
}

func (tx *importLimitTx) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, nil
}

func (tx *importLimitTx) QueryRow(context.Context, string, ...any) pgx.Row {
	return nil
}

func (tx *importLimitTx) Conn() *pgx.Conn {
	return nil
}

var _ MonthlyOrderCounter = importLimitOrderCounter{}
var _ pgx.Tx = (*importLimitTx)(nil)
