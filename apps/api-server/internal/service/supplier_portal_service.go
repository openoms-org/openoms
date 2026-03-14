package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

var (
	// ErrPortalTokenInvalid is returned when a portal token is invalid or expired.
	ErrPortalTokenInvalid = errors.New("invalid or expired portal token")
	// ErrPortalTokenExpired is returned when a portal token has passed its expiry time.
	ErrPortalTokenExpired = errors.New("portal token has expired")
	// ErrPortalNotEnabled is returned when portal access is disabled for a supplier.
	ErrPortalNotEnabled = errors.New("portal access is not enabled for this supplier")
	// ErrPortalPONotFound is returned when a portal purchase order does not exist.
	ErrPortalPONotFound = errors.New("purchase order not found")
	// ErrPortalPONotOwned is returned when a PO belongs to a different supplier.
	ErrPortalPONotOwned = errors.New("purchase order does not belong to this supplier")
	// ErrPortalInvalidAction is returned when an action is not permitted for the current PO status.
	ErrPortalInvalidAction = errors.New("this action is not allowed for the current PO status")
)

// SupplierPortalService handles business logic for the supplier portal.
type SupplierPortalService struct {
	tokenRepo    *repository.SupplierPortalTokenRepository
	messageRepo  *repository.SupplierMessageRepository
	supplierRepo repository.SupplierRepo
	poRepo       repository.PurchaseOrderRepo
	poItemRepo   repository.PurchaseOrderItemRepo
	auditRepo    repository.AuditRepo
	pool         *pgxpool.Pool
	baseURL      string
	logger       *slog.Logger
}

// NewSupplierPortalService creates a new SupplierPortalService.
func NewSupplierPortalService(
	tokenRepo *repository.SupplierPortalTokenRepository,
	messageRepo *repository.SupplierMessageRepository,
	supplierRepo repository.SupplierRepo,
	poRepo repository.PurchaseOrderRepo,
	poItemRepo repository.PurchaseOrderItemRepo,
	auditRepo repository.AuditRepo,
	pool *pgxpool.Pool,
	baseURL string,
	logger *slog.Logger,
) *SupplierPortalService {
	return &SupplierPortalService{
		tokenRepo:    tokenRepo,
		messageRepo:  messageRepo,
		supplierRepo: supplierRepo,
		poRepo:       poRepo,
		poItemRepo:   poItemRepo,
		auditRepo:    auditRepo,
		pool:         pool,
		baseURL:      baseURL,
		logger:       logger,
	}
}

// GeneratePortalLink creates a time-limited access token for a supplier (30 days).
func (s *SupplierPortalService) GeneratePortalLink(ctx context.Context, tenantID, supplierID uuid.UUID, actorID uuid.UUID, ip string) (*model.SupplierPortalLinkResponse, error) {
	var resp *model.SupplierPortalLinkResponse

	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		// Verify supplier exists
		supplier, err := s.supplierRepo.FindByID(ctx, tx, supplierID)
		if err != nil {
			return err
		}
		if supplier == nil {
			return ErrSupplierNotFound
		}

		// Generate 32 random bytes, hex-encode to 64-char token
		rawBytes := make([]byte, 32)
		if _, err := rand.Read(rawBytes); err != nil {
			return fmt.Errorf("generate random token: %w", err)
		}
		rawToken := hex.EncodeToString(rawBytes)

		// Hash with SHA-256 for storage
		hash := sha256.Sum256([]byte(rawToken))
		tokenHash := hex.EncodeToString(hash[:])

		expiresAt := time.Now().Add(30 * 24 * time.Hour)

		token := &model.SupplierPortalToken{
			ID:         uuid.New(),
			TenantID:   tenantID,
			SupplierID: supplierID,
			TokenHash:  tokenHash,
			ExpiresAt:  expiresAt,
		}
		if err := s.tokenRepo.Create(ctx, tx, token); err != nil {
			return fmt.Errorf("create portal token: %w", err)
		}

		// Enable portal if not already
		if !supplier.PortalEnabled {
			portalEnabled := true
			if err := s.supplierRepo.Update(ctx, tx, supplierID, model.UpdateSupplierRequest{
				PortalEnabled: &portalEnabled,
			}); err != nil {
				return fmt.Errorf("enable portal: %w", err)
			}
		}

		// Audit log
		if err := s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "supplier_portal.link_generated",
			EntityType: "supplier",
			EntityID:   supplierID,
			Changes:    map[string]string{"expires_at": expiresAt.Format(time.RFC3339)},
			IPAddress:  ip,
		}); err != nil {
			return err
		}

		resp = &model.SupplierPortalLinkResponse{
			URL:       fmt.Sprintf("%s/supplier-portal?token=%s", s.baseURL, rawToken),
			ExpiresAt: expiresAt,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// ValidateToken validates a raw token and returns the supplier.
func (s *SupplierPortalService) ValidateToken(ctx context.Context, rawToken string) (*model.Supplier, uuid.UUID, error) {
	// Hash the raw token
	hash := sha256.Sum256([]byte(rawToken))
	tokenHash := hex.EncodeToString(hash[:])

	var supplier *model.Supplier
	var tenantID uuid.UUID

	// We need to find the token without tenant context first (token_hash is globally unique).
	// Use a non-tenant transaction to look up the token, then switch to tenant context.
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, uuid.Nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// Look up token by hash (no RLS since we don't know tenant yet)
	var tokenRecord model.SupplierPortalToken
	err = tx.QueryRow(ctx,
		`SELECT id, tenant_id, supplier_id, token_hash, expires_at, last_used_at, created_at
		 FROM supplier_portal_tokens WHERE token_hash = $1`,
		tokenHash,
	).Scan(&tokenRecord.ID, &tokenRecord.TenantID, &tokenRecord.SupplierID,
		&tokenRecord.TokenHash, &tokenRecord.ExpiresAt, &tokenRecord.LastUsedAt, &tokenRecord.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, uuid.Nil, ErrPortalTokenInvalid
		}
		return nil, uuid.Nil, fmt.Errorf("find portal token: %w", err)
	}

	// Constant-time comparison of the hash
	expectedHash := sha256.Sum256([]byte(rawToken))
	expectedHashHex := hex.EncodeToString(expectedHash[:])
	if subtle.ConstantTimeCompare([]byte(tokenHash), []byte(expectedHashHex)) != 1 {
		return nil, uuid.Nil, ErrPortalTokenInvalid
	}

	// Check expiry
	if time.Now().After(tokenRecord.ExpiresAt) {
		return nil, uuid.Nil, ErrPortalTokenExpired
	}

	tenantID = tokenRecord.TenantID

	// Update last_used_at
	if _, err := tx.Exec(ctx, `UPDATE supplier_portal_tokens SET last_used_at = NOW() WHERE id = $1`, tokenRecord.ID); err != nil {
		s.logger.Error("failed to update portal token last_used_at", "error", err)
	}

	// Set tenant context to find the supplier
	if _, err := tx.Exec(ctx, "SELECT set_config('app.current_tenant_id', $1, true)", tenantID.String()); err != nil {
		return nil, uuid.Nil, fmt.Errorf("set tenant context: %w", err)
	}

	supplier, err = s.supplierRepo.FindByID(ctx, tx, tokenRecord.SupplierID)
	if err != nil {
		return nil, uuid.Nil, fmt.Errorf("find supplier: %w", err)
	}
	if supplier == nil {
		return nil, uuid.Nil, ErrPortalTokenInvalid
	}
	if !supplier.PortalEnabled {
		return nil, uuid.Nil, ErrPortalNotEnabled
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, uuid.Nil, fmt.Errorf("commit: %w", err)
	}

	return supplier, tenantID, nil
}

// RevokeTokens revokes all portal tokens for a supplier.
func (s *SupplierPortalService) RevokeTokens(ctx context.Context, tenantID, supplierID uuid.UUID, actorID uuid.UUID, ip string) error {
	return database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		if err := s.tokenRepo.DeleteBySupplier(ctx, tx, supplierID); err != nil {
			return err
		}

		// Disable portal
		portalEnabled := false
		if err := s.supplierRepo.Update(ctx, tx, supplierID, model.UpdateSupplierRequest{
			PortalEnabled: &portalEnabled,
		}); err != nil {
			return fmt.Errorf("disable portal: %w", err)
		}

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "supplier_portal.access_revoked",
			EntityType: "supplier",
			EntityID:   supplierID,
			IPAddress:  ip,
		})
	})
}

// ListPurchaseOrders returns purchase orders for a supplier.
func (s *SupplierPortalService) ListPurchaseOrders(ctx context.Context, tenantID, supplierID uuid.UUID) ([]model.SupplierPortalPO, error) {
	var result []model.SupplierPortalPO
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		filter := model.PurchaseOrderListFilter{
			SupplierID: &supplierID,
			PaginationParams: model.PaginationParams{
				Limit:  100,
				Offset: 0,
			},
		}
		orders, _, err := s.poRepo.List(ctx, tx, filter)
		if err != nil {
			return err
		}

		for _, po := range orders {
			// Only show POs that have been sent or later
			if po.Status == "draft" {
				continue
			}
			result = append(result, model.SupplierPortalPO{
				ID:                   po.ID,
				PONumber:             po.PONumber,
				Status:               po.Status,
				Notes:                po.Notes,
				ExpectedDeliveryDate: po.ExpectedDeliveryDate,
				TotalAmount:          po.TotalAmount,
				Currency:             po.Currency,
				CreatedAt:            po.CreatedAt,
				UpdatedAt:            po.UpdatedAt,
			})
		}
		return nil
	})
	if result == nil {
		result = []model.SupplierPortalPO{}
	}
	return result, err
}

// GetPurchaseOrder returns a single PO with items, scoped to the supplier.
func (s *SupplierPortalService) GetPurchaseOrder(ctx context.Context, tenantID, supplierID, poID uuid.UUID) (*model.SupplierPortalPO, error) {
	var result *model.SupplierPortalPO
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		po, err := s.poRepo.FindByID(ctx, tx, poID)
		if err != nil {
			return err
		}
		if po == nil {
			return ErrPortalPONotFound
		}
		if po.SupplierID == nil || *po.SupplierID != supplierID {
			return ErrPortalPONotOwned
		}
		if po.Status == "draft" {
			return ErrPortalPONotFound // hide drafts from suppliers
		}

		items, err := s.poItemRepo.ListByPOID(ctx, tx, poID)
		if err != nil {
			return err
		}
		if items == nil {
			items = []model.PurchaseOrderItem{}
		}

		result = &model.SupplierPortalPO{
			ID:                   po.ID,
			PONumber:             po.PONumber,
			Status:               po.Status,
			Notes:                po.Notes,
			ExpectedDeliveryDate: po.ExpectedDeliveryDate,
			TotalAmount:          po.TotalAmount,
			Currency:             po.Currency,
			Items:                items,
			CreatedAt:            po.CreatedAt,
			UpdatedAt:            po.UpdatedAt,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// ConfirmOrder allows a supplier to confirm a PO (transition from sent to confirmed).
func (s *SupplierPortalService) ConfirmOrder(ctx context.Context, tenantID, supplierID, poID uuid.UUID, reference string) error {
	return database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		po, err := s.poRepo.FindByID(ctx, tx, poID)
		if err != nil {
			return err
		}
		if po == nil {
			return ErrPortalPONotFound
		}
		if po.SupplierID == nil || *po.SupplierID != supplierID {
			return ErrPortalPONotOwned
		}
		if po.Status != "sent" {
			return ErrPortalInvalidAction
		}

		if err := s.poRepo.UpdateStatus(ctx, tx, poID, "confirmed"); err != nil {
			return fmt.Errorf("update PO status to confirmed: %w", err)
		}

		// Optionally store reference in notes
		if reference != "" {
			newNotes := po.Notes
			if newNotes != "" {
				newNotes += "\n"
			}
			newNotes += fmt.Sprintf("[Supplier ref: %s]", model.StripHTMLTags(reference))
			notesPtr := &newNotes
			if err := s.poRepo.Update(ctx, tx, poID, model.UpdatePurchaseOrderRequest{
				Notes: notesPtr,
			}); err != nil {
				s.logger.Error("failed to update PO notes with supplier reference", "error", err)
			}
		}

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     uuid.Nil,
			Action:     "supplier_portal.order_confirmed",
			EntityType: "purchase_order",
			EntityID:   poID,
			Changes:    map[string]string{"status": "confirmed", "supplier_id": supplierID.String(), "reference": reference},
		})
	})
}

// MarkShipped allows a supplier to mark a PO as shipped.
func (s *SupplierPortalService) MarkShipped(ctx context.Context, tenantID, supplierID, poID uuid.UUID, trackingNumber, carrier string) error {
	return database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		po, err := s.poRepo.FindByID(ctx, tx, poID)
		if err != nil {
			return err
		}
		if po == nil {
			return ErrPortalPONotFound
		}
		if po.SupplierID == nil || *po.SupplierID != supplierID {
			return ErrPortalPONotOwned
		}
		if po.Status != "sent" && po.Status != "confirmed" {
			return ErrPortalInvalidAction
		}

		if err := s.poRepo.UpdateStatus(ctx, tx, poID, "shipped"); err != nil {
			return fmt.Errorf("update PO status to shipped: %w", err)
		}

		// Store tracking info in notes
		trackingNote := fmt.Sprintf("[Shipped: %s / %s]", model.StripHTMLTags(carrier), model.StripHTMLTags(trackingNumber))
		newNotes := po.Notes
		if newNotes != "" {
			newNotes += "\n"
		}
		newNotes += trackingNote
		notesPtr := &newNotes
		if err := s.poRepo.Update(ctx, tx, poID, model.UpdatePurchaseOrderRequest{
			Notes: notesPtr,
		}); err != nil {
			s.logger.Error("failed to update PO notes with tracking", "error", err)
		}

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     uuid.Nil,
			Action:     "supplier_portal.order_shipped",
			EntityType: "purchase_order",
			EntityID:   poID,
			Changes: map[string]string{
				"status":          "shipped",
				"supplier_id":     supplierID.String(),
				"tracking_number": trackingNumber,
				"carrier":         carrier,
			},
		})
	})
}

// AddMessage adds a message from the supplier on a PO.
func (s *SupplierPortalService) AddMessage(ctx context.Context, tenantID, supplierID, poID uuid.UUID, message string) (*model.SupplierMessage, error) {
	var msg *model.SupplierMessage
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		// Verify PO belongs to supplier
		po, err := s.poRepo.FindByID(ctx, tx, poID)
		if err != nil {
			return err
		}
		if po == nil {
			return ErrPortalPONotFound
		}
		if po.SupplierID == nil || *po.SupplierID != supplierID {
			return ErrPortalPONotOwned
		}

		msg = &model.SupplierMessage{
			ID:              uuid.New(),
			TenantID:        tenantID,
			PurchaseOrderID: poID,
			SupplierID:      &supplierID,
			Message:         model.StripHTMLTags(message),
			IsFromSupplier:  true,
		}
		return s.messageRepo.Create(ctx, tx, msg)
	})
	if err != nil {
		return nil, err
	}
	return msg, nil
}

// ListMessages returns all messages for a PO.
func (s *SupplierPortalService) ListMessages(ctx context.Context, tenantID, poID uuid.UUID) ([]model.SupplierMessage, error) {
	var messages []model.SupplierMessage
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var err error
		messages, err = s.messageRepo.ListByPO(ctx, tx, poID)
		return err
	})
	if messages == nil {
		messages = []model.SupplierMessage{}
	}
	return messages, err
}

// GetPortalStatus returns portal status for a supplier (for admin UI).
func (s *SupplierPortalService) GetPortalStatus(ctx context.Context, tenantID, supplierID uuid.UUID) (bool, *time.Time, error) {
	var portalEnabled bool
	var lastUsedAt *time.Time

	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		supplier, err := s.supplierRepo.FindByID(ctx, tx, supplierID)
		if err != nil {
			return err
		}
		if supplier == nil {
			return ErrSupplierNotFound
		}
		portalEnabled = supplier.PortalEnabled

		tokens, err := s.tokenRepo.FindBySupplier(ctx, tx, supplierID)
		if err != nil {
			return err
		}
		// Find most recent last_used_at
		for _, t := range tokens {
			if t.LastUsedAt != nil {
				if lastUsedAt == nil || t.LastUsedAt.After(*lastUsedAt) {
					lastUsedAt = t.LastUsedAt
				}
			}
		}
		return nil
	})
	return portalEnabled, lastUsedAt, err
}
