package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// InvoiceRepository handles persistence for invoices.
type InvoiceRepository struct{}

// NewInvoiceRepository creates a new InvoiceRepository.
func NewInvoiceRepository() *InvoiceRepository {
	return &InvoiceRepository{}
}

// invoiceSelectColumns is the canonical list of columns selected from invoices.
const invoiceSelectColumns = `id, tenant_id, order_id, provider, external_id, external_number,
		        status, invoice_type, total_net, total_gross, currency,
		        issue_date, due_date, pdf_url, metadata, error_message,
		        ksef_number, ksef_status, ksef_sent_at, ksef_response,
		        created_at, updated_at`

// scanInvoice scans a row into a model.Invoice using the invoiceSelectColumns column order.
func scanInvoice(row pgx.Row) (model.Invoice, error) {
	var inv model.Invoice
	err := row.Scan(
		&inv.ID, &inv.TenantID, &inv.OrderID, &inv.Provider,
		&inv.ExternalID, &inv.ExternalNumber,
		&inv.Status, &inv.InvoiceType, &inv.TotalNet, &inv.TotalGross,
		&inv.Currency, &inv.IssueDate, &inv.DueDate, &inv.PDFURL,
		&inv.Metadata, &inv.ErrorMessage,
		&inv.KSeFNumber, &inv.KSeFStatus, &inv.KSeFSentAt, &inv.KSeFResponse,
		&inv.CreatedAt, &inv.UpdatedAt,
	)
	return inv, err
}

// scanInvoices scans multiple rows into a slice of model.Invoice using the invoiceSelectColumns column order.
func scanInvoices(rows pgx.Rows) ([]model.Invoice, error) {
	var invoices []model.Invoice
	for rows.Next() {
		inv, err := scanInvoice(rows)
		if err != nil {
			return nil, err
		}
		invoices = append(invoices, inv)
	}
	return invoices, rows.Err()
}

// List returns a paginated list of invoices matching the filter.
func (r *InvoiceRepository) List(ctx context.Context, tx pgx.Tx, filter model.InvoiceListFilter) ([]model.Invoice, int, error) {
	qb := NewQueryBuilder()

	if filter.Status != nil {
		qb.Add("status = $%d", *filter.Status)
	}
	if filter.Provider != nil {
		qb.Add("provider = $%d", *filter.Provider)
	}
	if filter.OrderID != nil {
		qb.Add("order_id = $%d", *filter.OrderID)
	}

	where := qb.WhereClause()

	var total int
	countQuery := "SELECT COUNT(*) FROM invoices " + where
	if err := tx.QueryRow(ctx, countQuery, qb.Args()...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count invoices: %w", err)
	}

	allowedSortColumns := map[string]string{
		"created_at":  "created_at",
		"status":      "status",
		"total_gross": "total_gross",
		"issue_date":  "issue_date",
	}
	orderByClause := model.BuildOrderByClause(filter.SortBy, filter.SortOrder, allowedSortColumns)

	argIdx := qb.AddArgs(filter.Limit, filter.Offset)
	query := fmt.Sprintf(
		`SELECT %s
		 FROM invoices %s
		 %s
		 LIMIT $%d OFFSET $%d`,
		invoiceSelectColumns, where, orderByClause, argIdx, argIdx+1,
	)

	rows, err := tx.Query(ctx, query, qb.Args()...)
	if err != nil {
		return nil, 0, fmt.Errorf("list invoices: %w", err)
	}
	defer rows.Close()

	invoices, err := scanInvoices(rows)
	if err != nil {
		return nil, 0, fmt.Errorf("scan invoice: %w", err)
	}
	return invoices, total, nil
}

// FindByID returns an invoice by its ID.
func (r *InvoiceRepository) FindByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.Invoice, error) {
	inv, err := scanInvoice(tx.QueryRow(ctx,
		fmt.Sprintf(`SELECT %s FROM invoices WHERE id = $1`, invoiceSelectColumns), id,
	))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find invoice by id: %w", err)
	}
	return &inv, nil
}

// FindByOrderID returns all invoices for the given order.
func (r *InvoiceRepository) FindByOrderID(ctx context.Context, tx pgx.Tx, orderID uuid.UUID) ([]model.Invoice, error) {
	rows, err := tx.Query(ctx,
		fmt.Sprintf(`SELECT %s FROM invoices WHERE order_id = $1 ORDER BY created_at DESC`, invoiceSelectColumns), orderID,
	)
	if err != nil {
		return nil, fmt.Errorf("find invoices by order_id: %w", err)
	}
	defer rows.Close()

	invoices, err := scanInvoices(rows)
	if err != nil {
		return nil, fmt.Errorf("scan invoice: %w", err)
	}
	return invoices, nil
}

// Create inserts a new invoice.
func (r *InvoiceRepository) Create(ctx context.Context, tx pgx.Tx, inv *model.Invoice) error {
	return tx.QueryRow(ctx,
		`INSERT INTO invoices (
			id, tenant_id, order_id, provider, external_id, external_number,
			status, invoice_type, total_net, total_gross, currency,
			issue_date, due_date, pdf_url, metadata, error_message,
			ksef_number, ksef_status, ksef_sent_at, ksef_response
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)
		RETURNING created_at, updated_at`,
		inv.ID, inv.TenantID, inv.OrderID, inv.Provider,
		inv.ExternalID, inv.ExternalNumber,
		inv.Status, inv.InvoiceType, inv.TotalNet, inv.TotalGross,
		inv.Currency, inv.IssueDate, inv.DueDate, inv.PDFURL,
		inv.Metadata, inv.ErrorMessage,
		inv.KSeFNumber, inv.KSeFStatus, inv.KSeFSentAt, inv.KSeFResponse,
	).Scan(&inv.CreatedAt, &inv.UpdatedAt)
}

// Update overwrites mutable fields on an invoice.
func (r *InvoiceRepository) Update(ctx context.Context, tx pgx.Tx, inv *model.Invoice) error {
	ct, err := tx.Exec(ctx,
		`UPDATE invoices SET
			external_id = $2, external_number = $3, status = $4,
			total_net = $5, total_gross = $6, pdf_url = $7,
			error_message = $8, metadata = $9,
			ksef_number = $10, ksef_status = $11, ksef_sent_at = $12, ksef_response = $13,
			updated_at = NOW()
		WHERE id = $1`,
		inv.ID, inv.ExternalID, inv.ExternalNumber, inv.Status,
		inv.TotalNet, inv.TotalGross, inv.PDFURL,
		inv.ErrorMessage, inv.Metadata,
		inv.KSeFNumber, inv.KSeFStatus, inv.KSeFSentAt, inv.KSeFResponse,
	)
	if err != nil {
		return fmt.Errorf("update invoice: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("invoice not found")
	}
	return nil
}

// Delete removes an invoice by its ID.
func (r *InvoiceRepository) Delete(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	ct, err := tx.Exec(ctx, "DELETE FROM invoices WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete invoice: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("invoice not found")
	}
	return nil
}

// FindPendingKSeF returns all invoices with ksef_status = 'pending'.
func (r *InvoiceRepository) FindPendingKSeF(ctx context.Context, tx pgx.Tx) ([]model.Invoice, error) {
	rows, err := tx.Query(ctx,
		fmt.Sprintf(`SELECT %s FROM invoices WHERE ksef_status = 'pending' ORDER BY ksef_sent_at ASC LIMIT 100`, invoiceSelectColumns),
	)
	if err != nil {
		return nil, fmt.Errorf("find pending ksef invoices: %w", err)
	}
	defer rows.Close()

	invoices, err := scanInvoices(rows)
	if err != nil {
		return nil, fmt.Errorf("scan invoice: %w", err)
	}
	return invoices, nil
}

// FindErrorKSeF returns all invoices with ksef_status = 'error' (eligible for retry).
func (r *InvoiceRepository) FindErrorKSeF(ctx context.Context, tx pgx.Tx) ([]model.Invoice, error) {
	rows, err := tx.Query(ctx,
		fmt.Sprintf(`SELECT %s FROM invoices WHERE ksef_status = 'error' ORDER BY updated_at ASC LIMIT 100`, invoiceSelectColumns),
	)
	if err != nil {
		return nil, fmt.Errorf("find error ksef invoices: %w", err)
	}
	defer rows.Close()

	invoices, err := scanInvoices(rows)
	if err != nil {
		return nil, fmt.Errorf("scan invoice: %w", err)
	}
	return invoices, nil
}

// FindRetryableKSeF returns invoices with ksef_status = 'retrying' that have a retry_count > 0
// in their ksef_response JSONB (i.e., invoices that were reset for retry, not fresh invoices).
func (r *InvoiceRepository) FindRetryableKSeF(ctx context.Context, tx pgx.Tx) ([]model.Invoice, error) {
	rows, err := tx.Query(ctx,
		fmt.Sprintf(`SELECT %s FROM invoices
			WHERE ksef_status = 'retrying'
			  AND ksef_response IS NOT NULL
			  AND ksef_response ? 'retry_count'
			  AND (ksef_response->>'retry_count')::int > 0
			ORDER BY updated_at ASC LIMIT 50`, invoiceSelectColumns),
	)
	if err != nil {
		return nil, fmt.Errorf("find retryable ksef invoices: %w", err)
	}
	defer rows.Close()

	invoices, err := scanInvoices(rows)
	if err != nil {
		return nil, fmt.Errorf("scan invoice: %w", err)
	}
	return invoices, nil
}

// UpdateKSeFStatus updates only the KSeF-related fields of an invoice.
func (r *InvoiceRepository) UpdateKSeFStatus(ctx context.Context, tx pgx.Tx, id uuid.UUID, ksefNumber *string, ksefStatus string, ksefResponse []byte) error {
	ct, err := tx.Exec(ctx,
		`UPDATE invoices SET
			ksef_number = $2, ksef_status = $3, ksef_response = $4, updated_at = NOW()
		WHERE id = $1`,
		id, ksefNumber, ksefStatus, ksefResponse,
	)
	if err != nil {
		return fmt.Errorf("update ksef status: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("invoice not found")
	}
	return nil
}
