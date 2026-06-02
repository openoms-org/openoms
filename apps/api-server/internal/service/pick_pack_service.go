package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

var (
	// ErrPickPackSessionNotFound is returned when a pick-pack session does not exist.
	ErrPickPackSessionNotFound = errors.New("pick-pack session not found")
	// ErrPickPackItemNotFound is returned when a pick-pack item does not exist.
	ErrPickPackItemNotFound = errors.New("pick-pack item not found")
	// ErrPickPackNotPicking is returned when an operation requires the session to be in picking status.
	ErrPickPackNotPicking = errors.New("session is not in picking status")
	// ErrPickPackNotPacking is returned when an operation requires the session to be in packing status.
	ErrPickPackNotPacking = errors.New("session is not in packing status")
	// ErrPickPackNotActive is returned when an operation requires an active pick-pack session.
	ErrPickPackNotActive = errors.New("session is not active")
	// ErrPickPackBarcodeNoMatch is returned when a scanned barcode does not match any session item.
	ErrPickPackBarcodeNoMatch = errors.New("barcode does not match any item in this session")
	// ErrPickPackQuantityExceeded is returned when a scan would exceed the required item quantity.
	ErrPickPackQuantityExceeded = errors.New("quantity would exceed required amount")
	// ErrPickPackNotAllPicked is returned when session completion is attempted before all items are picked.
	ErrPickPackNotAllPicked = errors.New("not all items have been picked")
	// ErrPickPackNotAllPacked is returned when session completion is attempted before all items are packed.
	ErrPickPackNotAllPacked = errors.New("not all items have been packed")
	// ErrOrderNotProcessing is returned when an operation requires the order to be in processing status.
	ErrOrderNotProcessing = errors.New("order is not in processing status")
)

// pickPackCompletedOrderStatus is the order status applied when a pick-pack
// session is completed. Sessions are created from orders in "processing" status,
// and the order state machine transitions "processing" -> "ready_to_ship", so a
// completed session advances its orders to "ready_to_ship". This must be a valid
// order state-machine status (see model.DefaultOrderStatusConfig); "packed" is
// not a valid order status.
const pickPackCompletedOrderStatus = "ready_to_ship"

// PickPackService provides business logic for pick-and-pack workflow.
type PickPackService struct {
	pickPackRepo *repository.PickPackRepository
	orderRepo    repository.OrderRepo
	productRepo  repository.ProductRepo
	variantRepo  repository.VariantRepo
	auditRepo    repository.AuditRepo
	pool         *pgxpool.Pool
}

// NewPickPackService creates a new PickPackService.
func NewPickPackService(
	pickPackRepo *repository.PickPackRepository,
	orderRepo repository.OrderRepo,
	productRepo repository.ProductRepo,
	variantRepo repository.VariantRepo,
	auditRepo repository.AuditRepo,
	pool *pgxpool.Pool,
) *PickPackService {
	return &PickPackService{
		pickPackRepo: pickPackRepo,
		orderRepo:    orderRepo,
		productRepo:  productRepo,
		variantRepo:  variantRepo,
		auditRepo:    auditRepo,
		pool:         pool,
	}
}

// pickPackOrderItem represents a parsed order item from the items JSONB array.
type pickPackOrderItem struct {
	ProductID *string `json:"product_id,omitempty"`
	SKU       string  `json:"sku"`
	Name      string  `json:"name"`
	Quantity  int     `json:"quantity"`
}

// CreateSession creates a new pick-pack session from the given order IDs.
func (s *PickPackService) CreateSession(ctx context.Context, tenantID uuid.UUID, req model.CreatePickPackSessionRequest, userID uuid.UUID, ip string) (*model.PickPackSession, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	var session *model.PickPackSession
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		// Validate all orders exist and are in "processing" status
		for _, orderID := range req.OrderIDs {
			order, err := s.orderRepo.FindByID(ctx, tx, orderID)
			if err != nil {
				return fmt.Errorf("find order %s: %w", orderID, err)
			}
			if order == nil {
				return fmt.Errorf("order %s not found", orderID)
			}
			if order.Status != "processing" {
				return fmt.Errorf("order %s: %w (current status: %s)", orderID, ErrOrderNotProcessing, order.Status)
			}
		}

		sessionType := "single"
		if len(req.OrderIDs) > 1 {
			sessionType = "batch"
		}

		session = &model.PickPackSession{
			ID:          uuid.New(),
			TenantID:    tenantID,
			SessionType: sessionType,
			Status:      "picking",
			AssignedTo:  &userID,
			Notes:       req.Notes,
		}

		if err := s.pickPackRepo.CreateSession(ctx, tx, session); err != nil {
			return fmt.Errorf("create session: %w", err)
		}

		// Extract items from each order and create pick-pack items
		for _, orderID := range req.OrderIDs {
			order, err := s.orderRepo.FindByID(ctx, tx, orderID)
			if err != nil {
				return err
			}

			var items []pickPackOrderItem
			if order.Items != nil {
				if err := json.Unmarshal(order.Items, &items); err != nil {
					return fmt.Errorf("parse order items for %s: %w", orderID, err)
				}
			}

			for _, item := range items {
				if item.SKU == "" {
					continue
				}

				ppItem := &model.PickPackItem{
					ID:               uuid.New(),
					TenantID:         tenantID,
					SessionID:        session.ID,
					OrderID:          orderID,
					SKU:              item.SKU,
					ProductName:      item.Name,
					QuantityRequired: item.Quantity,
				}

				// Try to resolve product ID and pick location
				if item.ProductID != nil && *item.ProductID != "" {
					if pid, err := uuid.Parse(*item.ProductID); err == nil {
						ppItem.ProductID = &pid
					}
				}

				// If no product_id from order items, look up by SKU
				if ppItem.ProductID == nil {
					product, err := s.productRepo.FindBySKU(ctx, tx, item.SKU)
					if err == nil && product != nil {
						ppItem.ProductID = &product.ID
					}
				}

				if err := s.pickPackRepo.CreateItem(ctx, tx, ppItem); err != nil {
					return fmt.Errorf("create pick_pack_item: %w", err)
				}
			}
		}

		// Get stats and items
		stats, err := s.pickPackRepo.GetSessionStats(ctx, tx, session.ID)
		if err != nil {
			return err
		}
		session.Stats = stats

		items, err := s.pickPackRepo.ListItemsBySession(ctx, tx, session.ID)
		if err != nil {
			return err
		}
		session.Items = items

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     userID,
			Action:     "pick_pack.session_created",
			EntityType: "pick_pack_session",
			EntityID:   session.ID,
			Changes:    map[string]string{"type": sessionType, "orders": fmt.Sprintf("%d", len(req.OrderIDs))},
			IPAddress:  ip,
		})
	})
	if err != nil {
		return nil, err
	}
	return session, nil
}

// GetSession retrieves a session with items and stats.
func (s *PickPackService) GetSession(ctx context.Context, tenantID, sessionID uuid.UUID) (*model.PickPackSession, error) {
	var session *model.PickPackSession
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var err error
		session, err = s.pickPackRepo.FindSessionByID(ctx, tx, sessionID)
		if err != nil {
			return err
		}
		if session == nil {
			return ErrPickPackSessionNotFound
		}

		items, err := s.pickPackRepo.ListItemsBySession(ctx, tx, sessionID)
		if err != nil {
			return err
		}
		session.Items = items

		stats, err := s.pickPackRepo.GetSessionStats(ctx, tx, sessionID)
		if err != nil {
			return err
		}
		session.Stats = stats

		return nil
	})
	if err != nil {
		return nil, err
	}
	return session, nil
}

// ListSessions lists pick-pack sessions with filtering and pagination.
func (s *PickPackService) ListSessions(ctx context.Context, tenantID uuid.UUID, filter model.PickPackSessionListFilter) (model.ListResponse[model.PickPackSession], error) {
	var resp model.ListResponse[model.PickPackSession]
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		sessions, total, err := s.pickPackRepo.ListSessions(ctx, tx, filter)
		if err != nil {
			return err
		}
		if sessions == nil {
			sessions = []model.PickPackSession{}
		}

		// Enrich each session with stats
		for i := range sessions {
			stats, err := s.pickPackRepo.GetSessionStats(ctx, tx, sessions[i].ID)
			if err != nil {
				return err
			}
			sessions[i].Stats = stats
		}

		resp = model.ListResponse[model.PickPackSession]{
			Items:  sessions,
			Total:  total,
			Limit:  filter.Limit,
			Offset: filter.Offset,
		}
		return nil
	})
	return resp, err
}

// ScanItem processes a barcode scan during the picking phase.
func (s *PickPackService) ScanItem(ctx context.Context, tenantID, sessionID uuid.UUID, req model.ScanItemRequest, _ uuid.UUID, _ string) (*model.ScanItemResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	var scanResp *model.ScanItemResponse
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		session, err := s.pickPackRepo.FindSessionByID(ctx, tx, sessionID)
		if err != nil {
			return err
		}
		if session == nil {
			return ErrPickPackSessionNotFound
		}
		if session.Status != "picking" {
			return ErrPickPackNotPicking
		}

		// Resolve barcode to SKU — try product SKU, then EAN, then variant SKU, then variant EAN
		sku := ""
		product, err := s.productRepo.FindBySKU(ctx, tx, req.Barcode)
		if err == nil && product != nil && product.SKU != nil {
			sku = *product.SKU
		}
		if sku == "" {
			product, err = s.productRepo.FindByEAN(ctx, tx, req.Barcode)
			if err == nil && product != nil && product.SKU != nil {
				sku = *product.SKU
			}
		}
		if sku == "" {
			variants, err := s.variantRepo.FindBySKU(ctx, tx, req.Barcode)
			if err == nil && len(variants) > 0 && variants[0].SKU != nil {
				sku = *variants[0].SKU
			}
		}
		if sku == "" {
			variants, err := s.variantRepo.FindByEAN(ctx, tx, req.Barcode)
			if err == nil && len(variants) > 0 && variants[0].SKU != nil {
				sku = *variants[0].SKU
			}
		}
		// Fallback: try using the barcode directly as SKU
		if sku == "" {
			sku = req.Barcode
		}

		// Find matching items in this session
		matchingItems, err := s.pickPackRepo.FindItemBySKUInSession(ctx, tx, sessionID, sku)
		if err != nil {
			return err
		}
		if len(matchingItems) == 0 {
			return ErrPickPackBarcodeNoMatch
		}

		// Find the first item that still needs picking
		var targetItem *model.PickPackItem
		for i := range matchingItems {
			if matchingItems[i].QuantityPicked < matchingItems[i].QuantityRequired {
				targetItem = &matchingItems[i]
				break
			}
		}
		if targetItem == nil {
			return ErrPickPackQuantityExceeded
		}

		// Increment picked count
		updatedItem, err := s.pickPackRepo.IncrementPicked(ctx, tx, targetItem.ID, 1)
		if err != nil {
			return err
		}

		// Calculate remaining items to pick across entire session
		stats, err := s.pickPackRepo.GetSessionStats(ctx, tx, sessionID)
		if err != nil {
			return err
		}
		remaining := stats.TotalRequired - stats.TotalPicked

		scanResp = &model.ScanItemResponse{
			Item:      *updatedItem,
			Remaining: remaining,
			Message:   fmt.Sprintf("Picked %s: %d/%d", updatedItem.ProductName, updatedItem.QuantityPicked, updatedItem.QuantityRequired),
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	return scanResp, nil
}

// MoveToPacking transitions the session from picking to packing.
func (s *PickPackService) MoveToPacking(ctx context.Context, tenantID, sessionID uuid.UUID, userID uuid.UUID, ip string) (*model.PickPackSession, error) {
	var session *model.PickPackSession
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		existing, err := s.pickPackRepo.FindSessionByID(ctx, tx, sessionID)
		if err != nil {
			return err
		}
		if existing == nil {
			return ErrPickPackSessionNotFound
		}
		if existing.Status != "picking" {
			return ErrPickPackNotPicking
		}

		// Verify all items are picked
		stats, err := s.pickPackRepo.GetSessionStats(ctx, tx, sessionID)
		if err != nil {
			return err
		}
		if !stats.AllPicked {
			return ErrPickPackNotAllPicked
		}

		if err := s.pickPackRepo.UpdateSessionStatus(ctx, tx, sessionID, "packing"); err != nil {
			return err
		}

		session, err = s.pickPackRepo.FindSessionByID(ctx, tx, sessionID)
		if err != nil {
			return err
		}

		items, err := s.pickPackRepo.ListItemsBySession(ctx, tx, sessionID)
		if err != nil {
			return err
		}
		session.Items = items
		session.Stats = stats

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     userID,
			Action:     "pick_pack.moved_to_packing",
			EntityType: "pick_pack_session",
			EntityID:   sessionID,
			IPAddress:  ip,
		})
	})
	if err != nil {
		return nil, err
	}
	return session, nil
}

// MarkItemPacked marks an item as packed (or partially packed).
func (s *PickPackService) MarkItemPacked(ctx context.Context, tenantID, sessionID, itemID uuid.UUID, req model.MarkPackedRequest, _ uuid.UUID, _ string) (*model.PickPackItem, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	var item *model.PickPackItem
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		session, err := s.pickPackRepo.FindSessionByID(ctx, tx, sessionID)
		if err != nil {
			return err
		}
		if session == nil {
			return ErrPickPackSessionNotFound
		}
		if session.Status != "packing" {
			return ErrPickPackNotPacking
		}

		existing, err := s.pickPackRepo.FindItemByID(ctx, tx, itemID)
		if err != nil {
			return err
		}
		if existing == nil || existing.SessionID != sessionID {
			return ErrPickPackItemNotFound
		}

		if existing.QuantityPacked+req.Quantity > existing.QuantityRequired {
			return ErrPickPackQuantityExceeded
		}

		item, err = s.pickPackRepo.IncrementPacked(ctx, tx, itemID, req.Quantity)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	return item, nil
}

// CompleteSession completes a session and transitions orders to the
// pickPackCompletedOrderStatus ("ready_to_ship") state.
func (s *PickPackService) CompleteSession(ctx context.Context, tenantID, sessionID uuid.UUID, userID uuid.UUID, ip string) (*model.PickPackSession, error) {
	var session *model.PickPackSession
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		existing, err := s.pickPackRepo.FindSessionByID(ctx, tx, sessionID)
		if err != nil {
			return err
		}
		if existing == nil {
			return ErrPickPackSessionNotFound
		}
		if existing.Status != "packing" {
			return ErrPickPackNotPacking
		}

		// Verify all items are packed
		stats, err := s.pickPackRepo.GetSessionStats(ctx, tx, sessionID)
		if err != nil {
			return err
		}
		if !stats.AllPacked {
			return ErrPickPackNotAllPacked
		}

		// Complete the session
		if err := s.pickPackRepo.CompleteSession(ctx, tx, sessionID); err != nil {
			return err
		}

		// Transition all orders to the pick-pack completed status (ready_to_ship)
		orderIDs, err := s.pickPackRepo.GetDistinctOrderIDs(ctx, tx, sessionID)
		if err != nil {
			return err
		}
		for _, orderID := range orderIDs {
			if err := s.orderRepo.UpdateStatus(ctx, tx, orderID, pickPackCompletedOrderStatus, nil, nil); err != nil {
				return fmt.Errorf("update order %s status: %w", orderID, err)
			}
		}

		session, err = s.pickPackRepo.FindSessionByID(ctx, tx, sessionID)
		if err != nil {
			return err
		}

		items, err := s.pickPackRepo.ListItemsBySession(ctx, tx, sessionID)
		if err != nil {
			return err
		}
		session.Items = items
		session.Stats = stats

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     userID,
			Action:     "pick_pack.session_completed",
			EntityType: "pick_pack_session",
			EntityID:   sessionID,
			Changes:    map[string]string{"orders_packed": fmt.Sprintf("%d", len(orderIDs))},
			IPAddress:  ip,
		})
	})
	if err != nil {
		return nil, err
	}
	return session, nil
}

// CancelSession cancels a session.
func (s *PickPackService) CancelSession(ctx context.Context, tenantID, sessionID uuid.UUID, userID uuid.UUID, ip string) (*model.PickPackSession, error) {
	var session *model.PickPackSession
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		existing, err := s.pickPackRepo.FindSessionByID(ctx, tx, sessionID)
		if err != nil {
			return err
		}
		if existing == nil {
			return ErrPickPackSessionNotFound
		}
		if existing.Status == "completed" || existing.Status == "cancelled" {
			return ErrPickPackNotActive
		}

		if err := s.pickPackRepo.UpdateSessionStatus(ctx, tx, sessionID, "cancelled"); err != nil {
			return err
		}

		session, err = s.pickPackRepo.FindSessionByID(ctx, tx, sessionID)
		if err != nil {
			return err
		}

		stats, err := s.pickPackRepo.GetSessionStats(ctx, tx, sessionID)
		if err != nil {
			return err
		}
		session.Stats = stats

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     userID,
			Action:     "pick_pack.session_cancelled",
			EntityType: "pick_pack_session",
			EntityID:   sessionID,
			IPAddress:  ip,
		})
	})
	if err != nil {
		return nil, err
	}
	return session, nil
}
