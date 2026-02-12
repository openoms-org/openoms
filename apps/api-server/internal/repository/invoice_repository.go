package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

type InvoiceRepository struct{}

func NewInvoiceRepository() *InvoiceRepository {
	return &InvoiceRepository{}
}

func (r *InvoiceRepository) List(ctx context.Context, tx pgx.Tx, filter model.InvoiceListFilter) ([]model.Invoice, int, error) {
	where := "WHERE 1=1"
	args := []any{}
	argIdx := 1

	if filter.Status != nil {
		where += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, *filter.Status)
		argIdx++
	}
	if filter.Provider != nil {
		where += fmt.Sprintf(" AND provider = $%d", argIdx)
		args = append(args, *filter.Provider)
		argIdx++
	}
	if filter.OrderID != nil {
		where += fmt.Sprintf(" AND order_id = $%d", argIdx)
		args = append(args, *filter.OrderID)
		argIdx++
	}

	var total int
	countQuery := "SELECT COUNT(*) FROM invoices " + where
	if err := tx.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count invoices: %w", err)
	}

	allowedSortColumns := map[string]string{
		"created_at":  "created_at",
		"status":      "status",
		"total_gross": "total_gross",
		"issue_date":  "issue_date",
	}
	orderByClause := model.BuildOrderByClause(filter.SortBy, filter.SortOrder, allowedSortColumns)

	query := fmt.Sprintf(
		`SELECT id, tenant_id, order_id, provider, external_id, external_number,
		        status, invoice_type, total_net, total_gross, currency,
		        issue_date, due_date, pdf_url, metadata, error_message,
		        ksef_number, ksef_status, ksef_sent_at, ksef_response,
		        created_at, updated_at
		 FROM invoices %s
		 %s
		 LIMIT $%d OFFSET $%d`,
		where, orderByClause, argIdx, argIdx+1,
	)
	args = append(args, filter.Limit, filter.Offset)

	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list invoices: %w", err)
	}
	defer rows.Close()

	var invoices []model.Invoice
	for rows.Next() {
		var inv model.Invoice
		if err := rows.Scan(
			&inv.ID, &inv.TenantID, &inv.OrderID, &inv.Provider,
			&inv.ExternalID, &inv.ExternalNumber,
			&inv.Status, &inv.InvoiceType, &inv.TotalNet, &inv.TotalGross,
			&inv.Currency, &inv.IssueDate, &inv.DueDate, &inv.PDFURL,
			&inv.Metadata, &inv.ErrorMessage,
			&inv.KSeFNumber, &inv.KSeFStatus, &inv.KSeFSentAt, &inv.KSeFResponse,
			&inv.CreatedAt, &inv.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan invoice: %w", err)
		}
		invoices = append(invoices, inv)
	}
	return invoices, total, rows.Err()
}

func (r *InvoiceRepository) FindByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.Invoice, error) {
	var inv model.Invoice
	err := tx.QueryRow(ctx,
		`SELECT id, tenant_id, order_id, provider, external_id, external_number,
		        status, invoice_type, total_net, total_gross, currency,
		        issue_date, due_date, pdf_url, metadata, error_message,
		        ksef_number, ksef_status, ksef_sent_at, ksef_response,
		        created_at, updated_at
		 FROM invoices WHERE id = $1`, id,
	).Scan(
		&inv.ID, &inv.TenantID, &inv.OrderID, &inv.Provider,
		&inv.ExternalID, &inv.ExternalNumber,
		&inv.Status, &inv.InvoiceType, &inv.TotalNet, &inv.TotalGross,
		&inv.Currency, &inv.IssueDate, &inv.DueDate, &inv.PDFURL,
		&inv.Metadata, &inv.ErrorMessage,
		&inv.CreatedAt, &inv.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find invoice by id: %w", err)
	}
	return &inv, nil
}

func (r *InvoiceRepository) FindByOrderID(ctx context.Context, tx pgx.Tx, orderID uuid.UUID) ([]model.Invoice, error) {
	rows, err := tx.Query(ctx,
		`SELECT id, tenant_id, order_id, provider, external_id, external_number,
		        status, invoice_type, total_net, total_gross, currency,
		        issue_date, due_date, pdf_url, metadata, error_message,
		        ksef_number, ksef_status, ksef_sent_at, ksef_response,
		        created_at, updated_at
		 FROM invoices WHERE order_id = $1 ORDER BY created_at DESC`, orderID,
	)
	if err != nil {
		return nil, fmt.Errorf("find invoices by order_id: %w", err)
	}
	defer rows.Close()

	var invoices []model.Invoice
	for rows.Next() {
		var inv model.Invoice
		if err := rows.Scan(
			&inv.ID, &inv.TenantID, &inv.OrderID, &inv.Provider,
			&inv.ExternalID, &inv.ExternalNumber,
			&inv.Status, &inv.InvoiceType, &inv.TotalNet, &inv.TotalGross,
			&inv.Currency, &inv.IssueDate, &inv.DueDate, &inv.PDFURL,
			&inv.Metadata, &inv.ErrorMessage,
			&inv.KSeFNumber, &inv.KSeFStatus, &inv.KSeFSentAt, &inv.KSeFResponse,
			&inv.CreatedAt, &inv.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan invoice: %w", err)
		}
		invoices = append(invoices, inv)
	}
	return invoices, rows.Err()
}

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
		`SELECT id, tenant_id, order_id, provider, external_id, external_number,
		        status, invoice_type, total_net, total_gross, currency,
		        issue_date, due_date, pdf_url, metadata, error_message,
		        ksef_number, ksef_status, ksef_sent_at, ksef_response,
		        created_at, updated_at
		 FROM invoices WHERE ksef_status = 'pending' ORDER BY ksef_sent_at ASC LIMIT 100`,
	)
	if err != nil {
		return nil, fmt.Errorf("find pending ksef invoices: %w", err)
	}
	defer rows.Close()

	var invoices []model.Invoice
	for rows.Next() {
		var inv model.Invoice
		if err := rows.Scan(
			&inv.ID, &inv.TenantID, &inv.OrderID, &inv.Provider,
			&inv.ExternalID, &inv.ExternalNumber,
			&inv.Status, &inv.InvoiceType, &inv.TotalNet, &inv.TotalGross,
			&inv.Currency, &inv.IssueDate, &inv.DueDate, &inv.PDFURL,
			&inv.Metadata, &inv.ErrorMessage,
			&inv.KSeFNumber, &inv.KSeFStatus, &inv.KSeFSentAt, &inv.KSeFResponse,
			&inv.CreatedAt, &inv.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan invoice: %w", err)
		}
		invoices = append(invoices, inv)
	}
	return invoices, rows.Err()
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
