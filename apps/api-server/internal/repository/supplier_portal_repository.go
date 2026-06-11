package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// SupplierPortalTokenRepository handles persistence for supplier portal tokens.
type SupplierPortalTokenRepository struct{}

// NewSupplierPortalTokenRepository creates a new SupplierPortalTokenRepository.
func NewSupplierPortalTokenRepository() *SupplierPortalTokenRepository {
	return &SupplierPortalTokenRepository{}
}

// Create inserts a new supplier portal token.
func (r *SupplierPortalTokenRepository) Create(ctx context.Context, tx pgx.Tx, token *model.SupplierPortalToken) error {
	return tx.QueryRow(ctx,
		`INSERT INTO supplier_portal_tokens (id, tenant_id, supplier_id, token_hash, expires_at)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING created_at`,
		token.ID, token.TenantID, token.SupplierID, token.TokenHash, token.ExpiresAt,
	).Scan(&token.CreatedAt)
}

// DeleteBySupplier removes all portal tokens for the given supplier.
func (r *SupplierPortalTokenRepository) DeleteBySupplier(ctx context.Context, tx pgx.Tx, supplierID uuid.UUID) error {
	_, err := tx.Exec(ctx,
		`DELETE FROM supplier_portal_tokens WHERE supplier_id = $1`, supplierID,
	)
	if err != nil {
		return fmt.Errorf("delete portal tokens by supplier: %w", err)
	}
	return nil
}

// FindBySupplier returns all portal tokens for the given supplier.
func (r *SupplierPortalTokenRepository) FindBySupplier(ctx context.Context, tx pgx.Tx, supplierID uuid.UUID) ([]model.SupplierPortalToken, error) {
	rows, err := tx.Query(ctx,
		`SELECT id, tenant_id, supplier_id, token_hash, expires_at, last_used_at, created_at
		 FROM supplier_portal_tokens WHERE supplier_id = $1 ORDER BY created_at DESC`,
		supplierID,
	)
	if err != nil {
		return nil, fmt.Errorf("list portal tokens by supplier: %w", err)
	}
	defer rows.Close()

	var tokens []model.SupplierPortalToken
	for rows.Next() {
		var t model.SupplierPortalToken
		if err := rows.Scan(&t.ID, &t.TenantID, &t.SupplierID, &t.TokenHash, &t.ExpiresAt, &t.LastUsedAt, &t.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan portal token: %w", err)
		}
		tokens = append(tokens, t)
	}
	return tokens, rows.Err()
}

// SupplierMessageRepository handles persistence for supplier messages.
type SupplierMessageRepository struct{}

// NewSupplierMessageRepository creates a new SupplierMessageRepository.
func NewSupplierMessageRepository() *SupplierMessageRepository {
	return &SupplierMessageRepository{}
}

// Create inserts a new supplier message.
func (r *SupplierMessageRepository) Create(ctx context.Context, tx pgx.Tx, msg *model.SupplierMessage) error {
	return tx.QueryRow(ctx,
		`INSERT INTO supplier_messages (id, tenant_id, purchase_order_id, supplier_id, user_id, message, is_from_supplier)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING created_at`,
		msg.ID, msg.TenantID, msg.PurchaseOrderID, msg.SupplierID, msg.UserID, msg.Message, msg.IsFromSupplier,
	).Scan(&msg.CreatedAt)
}

// ListByPO returns all messages for the given purchase order.
func (r *SupplierMessageRepository) ListByPO(ctx context.Context, tx pgx.Tx, poID uuid.UUID) ([]model.SupplierMessage, error) {
	rows, err := tx.Query(ctx,
		`SELECT id, tenant_id, purchase_order_id, supplier_id, user_id, message, is_from_supplier, created_at
		 FROM supplier_messages WHERE purchase_order_id = $1 ORDER BY created_at ASC`,
		poID,
	)
	if err != nil {
		return nil, fmt.Errorf("list supplier messages by PO: %w", err)
	}
	defer rows.Close()

	var messages []model.SupplierMessage
	for rows.Next() {
		var m model.SupplierMessage
		if err := rows.Scan(&m.ID, &m.TenantID, &m.PurchaseOrderID, &m.SupplierID, &m.UserID, &m.Message, &m.IsFromSupplier, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan supplier message: %w", err)
		}
		messages = append(messages, m)
	}
	return messages, rows.Err()
}
