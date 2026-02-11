package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// WarehouseDocumentRepository implements WarehouseDocumentRepo.
type WarehouseDocumentRepository struct{}

// NewWarehouseDocumentRepository creates a new WarehouseDocumentRepository.
func NewWarehouseDocumentRepository() *WarehouseDocumentRepository {
	return &WarehouseDocumentRepository{}
}

func (r *WarehouseDocumentRepository) List(ctx context.Context, tx pgx.Tx, filter model.WarehouseDocumentListFilter) ([]model.WarehouseDocument, int, error) {
	var conditions []string
	var args []any
	argIdx := 1

	if filter.DocumentType != nil {
		conditions = append(conditions, fmt.Sprintf("document_type = $%d", argIdx))
		args = append(args, *filter.DocumentType)
		argIdx++
	}
	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *filter.Status)
		argIdx++
	}
	if filter.WarehouseID != nil {
		conditions = append(conditions, fmt.Sprintf("warehouse_id = $%d", argIdx))
		args = append(args, *filter.WarehouseID)
		argIdx++
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM warehouse_documents %s", where)
	if err := tx.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count warehouse_documents: %w", err)
	}

	allowedSortColumns := map[string]string{
		"created_at":      "created_at",
		"document_number": "document_number",
		"document_type":   "document_type",
		"status":          "status",
	}
	orderByClause := model.BuildOrderByClause(filter.SortBy, filter.SortOrder, allowedSortColumns)

	query := fmt.Sprintf(
		`SELECT id, tenant_id, document_number, document_type, status, warehouse_id,
		        target_warehouse_id, supplier_id, order_id, notes,
		        confirmed_at, confirmed_by, created_by, created_at, updated_at
		 FROM warehouse_documents %s %s LIMIT $%d OFFSET $%d`,
		where, orderByClause, argIdx, argIdx+1,
	)
	args = append(args, filter.Limit, filter.Offset)

	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list warehouse_documents: %w", err)
	}
	defer rows.Close()

	var docs []model.WarehouseDocument
	for rows.Next() {
		var d model.WarehouseDocument
		if err := rows.Scan(
			&d.ID, &d.TenantID, &d.DocumentNumber, &d.DocumentType, &d.Status,
			&d.WarehouseID, &d.TargetWarehouseID, &d.SupplierID, &d.OrderID, &d.Notes,
			&d.ConfirmedAt, &d.ConfirmedBy, &d.CreatedBy, &d.CreatedAt, &d.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan warehouse_document: %w", err)
		}
		docs = append(docs, d)
	}
	return docs, total, rows.Err()
}

func (r *WarehouseDocumentRepository) FindByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.WarehouseDocument, error) {
	var d model.WarehouseDocument
	err := tx.QueryRow(ctx,
		`SELECT id, tenant_id, document_number, document_type, status, warehouse_id,
		        target_warehouse_id, supplier_id, order_id, notes,
		        confirmed_at, confirmed_by, created_by, created_at, updated_at
		 FROM warehouse_documents WHERE id = $1`, id,
	).Scan(
		&d.ID, &d.TenantID, &d.DocumentNumber, &d.DocumentType, &d.Status,
		&d.WarehouseID, &d.TargetWarehouseID, &d.SupplierID, &d.OrderID, &d.Notes,
		&d.ConfirmedAt, &d.ConfirmedBy, &d.CreatedBy, &d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find warehouse_document by id: %w", err)
	}
	return &d, nil
}

func (r *WarehouseDocumentRepository) Create(ctx context.Context, tx pgx.Tx, doc *model.WarehouseDocument) error {
	return tx.QueryRow(ctx,
		`INSERT INTO warehouse_documents
		 (id, tenant_id, document_number, document_type, status, warehouse_id,
		  target_warehouse_id, supplier_id, order_id, notes, created_by)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		 RETURNING created_at, updated_at`,
		doc.ID, doc.TenantID, doc.DocumentNumber, doc.DocumentType, doc.Status,
		doc.WarehouseID, doc.TargetWarehouseID, doc.SupplierID, doc.OrderID, doc.Notes, doc.CreatedBy,
	).Scan(&doc.CreatedAt, &doc.UpdatedAt)
}

func (r *WarehouseDocumentRepository) Update(ctx context.Context, tx pgx.Tx, id uuid.UUID, req model.UpdateWarehouseDocumentRequest) error {
	var setClauses []string
	var args []any
	argIdx := 1

	if req.Notes != nil {
		setClauses = append(setClauses, fmt.Sprintf("notes = $%d", argIdx))
		args = append(args, *req.Notes)
		argIdx++
	}

	if len(setClauses) == 0 {
		return nil
	}

	setClauses = append(setClauses, "updated_at = NOW()")
	query := fmt.Sprintf("UPDATE warehouse_documents SET %s WHERE id = $%d",
		strings.Join(setClauses, ", "), argIdx)
	args = append(args, id)

	ct, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update warehouse_document: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("warehouse_document not found")
	}
	return nil
}

func (r *WarehouseDocumentRepository) Delete(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	ct, err := tx.Exec(ctx, "DELETE FROM warehouse_documents WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete warehouse_document: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("warehouse_document not found")
	}
	return nil
}

func (r *WarehouseDocumentRepository) Confirm(ctx context.Context, tx pgx.Tx, id uuid.UUID, confirmedBy uuid.UUID) error {
	ct, err := tx.Exec(ctx,
		`UPDATE warehouse_documents SET status = 'confirmed', confirmed_at = NOW(), confirmed_by = $1, updated_at = NOW()
		 WHERE id = $2 AND status = 'draft'`,
		confirmedBy, id,
	)
	if err != nil {
		return fmt.Errorf("confirm warehouse_document: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("warehouse_document not found or not in draft status")
	}
	return nil
}

func (r *WarehouseDocumentRepository) Cancel(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	ct, err := tx.Exec(ctx,
		`UPDATE warehouse_documents SET status = 'cancelled', updated_at = NOW()
		 WHERE id = $1 AND status = 'draft'`,
		id,
	)
	if err != nil {
		return fmt.Errorf("cancel warehouse_document: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("warehouse_document not found or not in draft status")
	}
	return nil
}

func (r *WarehouseDocumentRepository) NextDocumentNumber(ctx context.Context, tx pgx.Tx, docType string, year int) (int, error) {
	var count int
	err := tx.QueryRow(ctx,
		`SELECT COUNT(*) FROM warehouse_documents
		 WHERE document_type = $1 AND EXTRACT(YEAR FROM created_at) = $2`,
		docType, year,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("next document number: %w", err)
	}
	return count + 1, nil
}

// WarehouseDocItemRepository implements WarehouseDocItemRepo.
type WarehouseDocItemRepository struct{}

// NewWarehouseDocItemRepository creates a new WarehouseDocItemRepository.
func NewWarehouseDocItemRepository() *WarehouseDocItemRepository {
	return &WarehouseDocItemRepository{}
}

func (r *WarehouseDocItemRepository) ListByDocumentID(ctx context.Context, tx pgx.Tx, documentID uuid.UUID) ([]model.WarehouseDocItem, error) {
	rows, err := tx.Query(ctx,
		`SELECT id, tenant_id, document_id, product_id, variant_id, quantity, unit_price, notes, created_at
		 FROM warehouse_document_items WHERE document_id = $1
		 ORDER BY created_at ASC`,
		documentID,
	)
	if err != nil {
		return nil, fmt.Errorf("list warehouse_document_items: %w", err)
	}
	defer rows.Close()

	var items []model.WarehouseDocItem
	for rows.Next() {
		var item model.WarehouseDocItem
		if err := rows.Scan(
			&item.ID, &item.TenantID, &item.DocumentID, &item.ProductID,
			&item.VariantID, &item.Quantity, &item.UnitPrice, &item.Notes, &item.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan warehouse_document_item: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *WarehouseDocItemRepository) Create(ctx context.Context, tx pgx.Tx, item *model.WarehouseDocItem) error {
	return tx.QueryRow(ctx,
		`INSERT INTO warehouse_document_items
		 (id, tenant_id, document_id, product_id, variant_id, quantity, unit_price, notes)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 RETURNING created_at`,
		item.ID, item.TenantID, item.DocumentID, item.ProductID,
		item.VariantID, item.Quantity, item.UnitPrice, item.Notes,
	).Scan(&item.CreatedAt)
}
