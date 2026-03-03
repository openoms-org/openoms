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

// CustomerImportService handles CSV import of customers.
type CustomerImportService struct {
	customerRepo repository.CustomerRepo
	auditRepo    repository.AuditRepo
	pool         *pgxpool.Pool
}

// NewCustomerImportService creates a new CustomerImportService.
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

// CustomerImportPreview is the response for the customer import preview endpoint.
type CustomerImportPreview struct {
	Headers     []string                    `json:"headers"`
	TotalRows   int                         `json:"total_rows"`
	SampleRows  []model.ImportPreviewRow    `json:"sample_rows"`
	NewCount    int                         `json:"new_count"`
	UpdateCount int                         `json:"update_count"`
	Mappings    []model.ImportColumnMapping `json:"mappings,omitempty"`
}

// CustomerImportResult is the response for the customer import endpoint.
type CustomerImportResult struct {
	Created int                 `json:"created"`
	Updated int                 `json:"updated"`
	Skipped int                 `json:"skipped"`
	Errors  []model.ImportError `json:"errors"`
}

// customerFieldAliases maps CSV header names (lowercase) to canonical field names.
// Supports BaseLinker export column names.
var customerFieldAliases = map[string]string{
	"name":              "name",
	"customer_name":     "name",
	"customer name":     "name",
	"buyer_name":        "name",
	"delivery_fullname": "name",
	"email":             "email",
	"customer_email":    "email",
	"customer email":    "email",
	"buyer_email":       "email",
	"phone":             "phone",
	"customer_phone":    "phone",
	"customer phone":    "phone",
	"buyer_phone":       "phone",
	"company_name":      "company_name",
	"company":           "company_name",
	"company name":      "company_name",
	"invoice_company":   "company_name",
	"nip":               "nip",
	"invoice_nip":       "nip",
	"tax_id":            "nip",
	"tags":              "tags",
	"notes":             "notes",
}

// canonicalCustomerFields lists all canonical field names the importer understands.
var canonicalCustomerFields = []string{"name", "email", "phone", "company_name", "nip", "tags", "notes"}

// parseCustomerCSV reads raw CSV bytes and returns headers, records, and the header-to-index map
// with alias resolution applied.
func parseCustomerCSV(raw []byte) (headers []string, records [][]string, fieldIdx map[string]int, err error) {
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

	// Build field index: for each CSV header, resolve alias to canonical name.
	fieldIdx = make(map[string]int, len(headers))
	for i, h := range headers {
		key := strings.ToLower(strings.TrimSpace(h))
		if canonical, ok := customerFieldAliases[key]; ok {
			// First match wins — don't overwrite if already mapped.
			if _, exists := fieldIdx[canonical]; !exists {
				fieldIdx[canonical] = i
			}
		}
	}

	return headers, records, fieldIdx, nil
}

// extractField returns the trimmed value from a CSV row for the given canonical field name.
func extractField(row []string, fieldIdx map[string]int, field string) string {
	idx, ok := fieldIdx[field]
	if !ok || idx >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[idx])
}

// PreviewCSV parses a customer CSV and returns a preview with stats.
func (s *CustomerImportService) PreviewCSV(ctx context.Context, tenantID uuid.UUID, file io.Reader) (*CustomerImportPreview, error) {
	raw, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("read csv: %w", err)
	}

	headers, records, fieldIdx, err := parseCustomerCSV(raw)
	if err != nil {
		return nil, err
	}

	// Validate that we can resolve at least the "name" field.
	if _, ok := fieldIdx["name"]; !ok {
		return nil, fmt.Errorf("CSV must have a recognizable name column (e.g. 'name', 'customer_name', 'buyer_name')")
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

	// Build column mappings.
	var mappings []model.ImportColumnMapping
	for i, h := range headers {
		key := strings.ToLower(strings.TrimSpace(h))
		if canonical, ok := customerFieldAliases[key]; ok {
			// Only include if this header index is the one actually used.
			if fieldIdx[canonical] == i {
				mappings = append(mappings, model.ImportColumnMapping{
					CSVColumn:  h,
					OrderField: canonical,
				})
			}
		}
	}

	// Count new vs update by email match.
	newCount := 0
	updateCount := 0

	err = database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		for rowNum := 1; rowNum < len(records); rowNum++ {
			row := records[rowNum]
			name := extractField(row, fieldIdx, "name")
			if name == "" {
				continue // will be skipped during import
			}

			email := extractField(row, fieldIdx, "email")
			if email != "" {
				existing, findErr := s.customerRepo.FindByEmail(ctx, tx, email)
				if findErr != nil {
					return findErr
				}
				if existing != nil {
					updateCount++
					continue
				}
			}
			newCount++
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("preview analysis: %w", err)
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

// ImportCSV performs a batch import of customers from CSV, upserting by email.
func (s *CustomerImportService) ImportCSV(
	ctx context.Context,
	tenantID uuid.UUID,
	file io.Reader,
	userID uuid.UUID,
	ip string,
) (*CustomerImportResult, error) {
	raw, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("read csv: %w", err)
	}

	_, records, fieldIdx, err := parseCustomerCSV(raw)
	if err != nil {
		return nil, err
	}

	if len(records) < 2 {
		return nil, fmt.Errorf("csv file must have a header row and at least one data row")
	}

	// Validate that we can resolve at least the "name" field.
	if _, ok := fieldIdx["name"]; !ok {
		return nil, fmt.Errorf("CSV must have a recognizable name column (e.g. 'name', 'customer_name', 'buyer_name')")
	}

	result := &CustomerImportResult{
		Errors: []model.ImportError{},
	}

	err = database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		for rowNum := 1; rowNum < len(records); rowNum++ {
			row := records[rowNum]
			rowErr := s.importCustomerRow(ctx, tx, tenantID, row, fieldIdx, rowNum, result)
			if rowErr != nil {
				result.Errors = append(result.Errors, *rowErr)
			}
		}

		// Audit log.
		_ = s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     userID,
			Action:     "customer.import",
			EntityType: "customer",
			EntityID:   uuid.Nil,
			Changes: map[string]string{
				"created": strconv.Itoa(result.Created),
				"updated": strconv.Itoa(result.Updated),
				"skipped": strconv.Itoa(result.Skipped),
				"errors":  strconv.Itoa(len(result.Errors)),
			},
			IPAddress: ip,
		})

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("import customers: %w", err)
	}

	return result, nil
}

// importCustomerRow processes a single CSV row for customer import.
func (s *CustomerImportService) importCustomerRow(
	ctx context.Context,
	tx pgx.Tx,
	tenantID uuid.UUID,
	row []string,
	fieldIdx map[string]int,
	rowNum int,
	result *CustomerImportResult,
) *model.ImportError {
	name := extractField(row, fieldIdx, "name")
	if name == "" {
		return &model.ImportError{Row: rowNum, Field: "name", Message: "name is required"}
	}

	email := extractField(row, fieldIdx, "email")
	phone := extractField(row, fieldIdx, "phone")
	companyName := extractField(row, fieldIdx, "company_name")
	nip := extractField(row, fieldIdx, "nip")
	tagsStr := extractField(row, fieldIdx, "tags")
	notes := extractField(row, fieldIdx, "notes")

	// Parse comma-separated tags.
	var tags []string
	if tagsStr != "" {
		for t := range strings.SplitSeq(tagsStr, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				tags = append(tags, t)
			}
		}
	}
	if tags == nil {
		tags = []string{}
	}

	// Helper to create *string from non-empty value.
	strPtr := func(v string) *string {
		if v == "" {
			return nil
		}
		return &v
	}

	// Dedup by email: if email present and customer exists, update.
	if email != "" {
		existing, err := s.customerRepo.FindByEmail(ctx, tx, email)
		if err != nil {
			return &model.ImportError{Row: rowNum, Message: fmt.Sprintf("error looking up email: %s", err.Error())}
		}
		if existing != nil {
			req := model.UpdateCustomerRequest{
				Name:  &name,
				Tags:  &tags,
				Email: strPtr(email),
			}
			if phone != "" {
				req.Phone = strPtr(phone)
			}
			if companyName != "" {
				req.CompanyName = strPtr(companyName)
			}
			if nip != "" {
				req.NIP = strPtr(nip)
			}
			if notes != "" {
				req.Notes = strPtr(notes)
			}

			if err := s.customerRepo.Update(ctx, tx, existing.ID, req); err != nil {
				return &model.ImportError{Row: rowNum, Message: fmt.Sprintf("failed to update customer: %s", err.Error())}
			}
			result.Updated++
			return nil
		}
	}

	// Create new customer.
	customer := &model.Customer{
		ID:          uuid.New(),
		TenantID:    tenantID,
		Name:        name,
		Email:       strPtr(email),
		Phone:       strPtr(phone),
		CompanyName: strPtr(companyName),
		NIP:         strPtr(nip),
		Tags:        tags,
		Notes:       strPtr(notes),
	}

	if err := s.customerRepo.Create(ctx, tx, customer); err != nil {
		return &model.ImportError{Row: rowNum, Message: fmt.Sprintf("failed to create customer: %s", err.Error())}
	}
	result.Created++
	return nil
}
