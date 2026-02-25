package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/asyncutil"
	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

var (
	ErrPurchaseOrderNotFound = errors.New("purchase order not found")
	ErrPOItemNotFound        = errors.New("purchase order item not found")
	ErrPONotDraft            = errors.New("purchase order is not in draft status")
	ErrPOInvalidTransition   = errors.New("invalid status transition")
	ErrPOAlreadyCancelled    = errors.New("purchase order is already cancelled")
)

// PurchaseOrderService handles business logic for purchase orders.
type PurchaseOrderService struct {
	poRepo          repository.PurchaseOrderRepo
	poItemRepo      repository.PurchaseOrderItemRepo
	warehouseStock  repository.WarehouseStockRepo
	auditRepo       repository.AuditRepo
	pool            *pgxpool.Pool
	webhookDispatch *WebhookDispatchService
	logger          *slog.Logger
}

// NewPurchaseOrderService creates a new PurchaseOrderService.
func NewPurchaseOrderService(
	poRepo repository.PurchaseOrderRepo,
	poItemRepo repository.PurchaseOrderItemRepo,
	warehouseStock repository.WarehouseStockRepo,
	auditRepo repository.AuditRepo,
	pool *pgxpool.Pool,
	webhookDispatch *WebhookDispatchService,
	logger *slog.Logger,
) *PurchaseOrderService {
	return &PurchaseOrderService{
		poRepo:          poRepo,
		poItemRepo:      poItemRepo,
		warehouseStock:  warehouseStock,
		auditRepo:       auditRepo,
		pool:            pool,
		webhookDispatch: webhookDispatch,
		logger:          logger,
	}
}

// List returns a paginated list of purchase orders.
func (s *PurchaseOrderService) List(ctx context.Context, tenantID uuid.UUID, filter model.PurchaseOrderListFilter) (model.ListResponse[model.PurchaseOrder], error) {
	var resp model.ListResponse[model.PurchaseOrder]
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		orders, total, err := s.poRepo.List(ctx, tx, filter)
		if err != nil {
			return err
		}
		if orders == nil {
			orders = []model.PurchaseOrder{}
		}
		resp = model.ListResponse[model.PurchaseOrder]{
			Items:  orders,
			Total:  total,
			Limit:  filter.Limit,
			Offset: filter.Offset,
		}
		return nil
	})
	return resp, err
}

// Get returns a single purchase order with its items.
func (s *PurchaseOrderService) Get(ctx context.Context, tenantID, poID uuid.UUID) (*model.PurchaseOrder, error) {
	var po *model.PurchaseOrder
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var err error
		po, err = s.poRepo.FindByID(ctx, tx, poID)
		if err != nil {
			return err
		}
		if po == nil {
			return ErrPurchaseOrderNotFound
		}
		items, err := s.poItemRepo.ListByPOID(ctx, tx, poID)
		if err != nil {
			return err
		}
		if items == nil {
			items = []model.PurchaseOrderItem{}
		}
		po.Items = items
		return nil
	})
	if err != nil {
		return nil, err
	}
	return po, nil
}

// Create creates a new purchase order with items.
func (s *PurchaseOrderService) Create(ctx context.Context, tenantID uuid.UUID, req model.CreatePurchaseOrderRequest, actorID uuid.UUID, ip string) (*model.PurchaseOrder, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	// Sanitize text fields
	req.SupplierName = model.StripHTMLTags(req.SupplierName)

	var po *model.PurchaseOrder
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		// Generate PO number
		poNumber, err := s.poRepo.GeneratePONumber(ctx, tx, tenantID)
		if err != nil {
			return fmt.Errorf("generate PO number: %w", err)
		}

		// Parse expected delivery date if provided
		var expectedDate *time.Time
		if req.ExpectedDeliveryDate != nil && *req.ExpectedDeliveryDate != "" {
			t, err := time.Parse("2006-01-02", *req.ExpectedDeliveryDate)
			if err != nil {
				return NewValidationError(fmt.Errorf("invalid expected_delivery_date format, use YYYY-MM-DD"))
			}
			expectedDate = &t
		}

		// Calculate total
		var totalAmount float64
		for _, item := range req.Items {
			totalAmount += float64(item.Quantity) * item.UnitCost
		}

		po = &model.PurchaseOrder{
			ID:                   uuid.New(),
			TenantID:             tenantID,
			PONumber:             poNumber,
			SupplierID:           req.SupplierID,
			SupplierName:         req.SupplierName,
			WarehouseID:          req.WarehouseID,
			Status:               "draft",
			Notes:                req.Notes,
			ExpectedDeliveryDate: expectedDate,
			TotalAmount:          totalAmount,
			Currency:             req.Currency,
			CreatedBy:            &actorID,
		}

		if err := s.poRepo.Create(ctx, tx, po); err != nil {
			return fmt.Errorf("create purchase order: %w", err)
		}

		// Create items
		var items []model.PurchaseOrderItem
		for _, itemReq := range req.Items {
			item := &model.PurchaseOrderItem{
				ID:              uuid.New(),
				TenantID:        tenantID,
				PurchaseOrderID: po.ID,
				ProductID:       itemReq.ProductID,
				SKU:             itemReq.SKU,
				ProductName:     model.StripHTMLTags(itemReq.ProductName),
				QuantityOrdered: itemReq.Quantity,
				UnitCost:        itemReq.UnitCost,
				Notes:           itemReq.Notes,
			}
			if err := s.poItemRepo.CreateItem(ctx, tx, item); err != nil {
				return fmt.Errorf("create PO item: %w", err)
			}
			items = append(items, *item)
		}
		po.Items = items

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "purchase_order.created",
			EntityType: "purchase_order",
			EntityID:   po.ID,
			Changes:    map[string]string{"po_number": poNumber, "supplier": req.SupplierName},
			IPAddress:  ip,
		})
	})
	if err != nil {
		return nil, err
	}

	if s.webhookDispatch != nil {
		asyncutil.SafeGo(func() { s.webhookDispatch.Dispatch(context.Background(), tenantID, "purchase_order.created", po) })
	}
	return po, nil
}

// Update updates a purchase order (only in draft status).
func (s *PurchaseOrderService) Update(ctx context.Context, tenantID, poID uuid.UUID, req model.UpdatePurchaseOrderRequest, actorID uuid.UUID, ip string) (*model.PurchaseOrder, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	var po *model.PurchaseOrder
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		existing, err := s.poRepo.FindByID(ctx, tx, poID)
		if err != nil {
			return err
		}
		if existing == nil {
			return ErrPurchaseOrderNotFound
		}
		if existing.Status != "draft" {
			return ErrPONotDraft
		}

		if err := s.poRepo.Update(ctx, tx, poID, req); err != nil {
			return err
		}

		// If items are provided, replace all items
		if req.Items != nil {
			if err := s.poItemRepo.DeleteByPOID(ctx, tx, poID); err != nil {
				return err
			}
			var totalAmount float64
			for _, itemReq := range req.Items {
				item := &model.PurchaseOrderItem{
					ID:              uuid.New(),
					TenantID:        tenantID,
					PurchaseOrderID: poID,
					ProductID:       itemReq.ProductID,
					SKU:             itemReq.SKU,
					ProductName:     model.StripHTMLTags(itemReq.ProductName),
					QuantityOrdered: itemReq.Quantity,
					UnitCost:        itemReq.UnitCost,
					Notes:           itemReq.Notes,
				}
				if err := s.poItemRepo.CreateItem(ctx, tx, item); err != nil {
					return fmt.Errorf("create PO item: %w", err)
				}
				totalAmount += float64(itemReq.Quantity) * itemReq.UnitCost
			}
			if err := s.poRepo.UpdateTotalAmount(ctx, tx, poID, totalAmount); err != nil {
				return err
			}
		}

		po, err = s.poRepo.FindByID(ctx, tx, poID)
		if err != nil {
			return err
		}
		items, err := s.poItemRepo.ListByPOID(ctx, tx, poID)
		if err != nil {
			return err
		}
		if items == nil {
			items = []model.PurchaseOrderItem{}
		}
		po.Items = items

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "purchase_order.updated",
			EntityType: "purchase_order",
			EntityID:   poID,
			IPAddress:  ip,
		})
	})
	if err != nil {
		return nil, err
	}

	if s.webhookDispatch != nil {
		asyncutil.SafeGo(func() { s.webhookDispatch.Dispatch(context.Background(), tenantID, "purchase_order.updated", po) })
	}
	return po, nil
}

// Delete deletes a purchase order (only in draft status).
func (s *PurchaseOrderService) Delete(ctx context.Context, tenantID, poID uuid.UUID, actorID uuid.UUID, ip string) error {
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		po, err := s.poRepo.FindByID(ctx, tx, poID)
		if err != nil {
			return err
		}
		if po == nil {
			return ErrPurchaseOrderNotFound
		}
		if po.Status != "draft" {
			return ErrPONotDraft
		}

		if err := s.poRepo.Delete(ctx, tx, poID); err != nil {
			return err
		}

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "purchase_order.deleted",
			EntityType: "purchase_order",
			EntityID:   poID,
			Changes:    map[string]string{"po_number": po.PONumber},
			IPAddress:  ip,
		})
	})
	if err == nil && s.webhookDispatch != nil {
		asyncutil.SafeGo(func() {
			s.webhookDispatch.Dispatch(context.Background(), tenantID, "purchase_order.deleted", map[string]any{"id": poID.String()})
		})
	}
	return err
}

// Send marks a purchase order as sent.
func (s *PurchaseOrderService) Send(ctx context.Context, tenantID, poID uuid.UUID, actorID uuid.UUID, ip string) (*model.PurchaseOrder, error) {
	var po *model.PurchaseOrder
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		existing, err := s.poRepo.FindByID(ctx, tx, poID)
		if err != nil {
			return err
		}
		if existing == nil {
			return ErrPurchaseOrderNotFound
		}
		if existing.Status != "draft" {
			return NewValidationError(fmt.Errorf("can only send draft purchase orders, current status: %s", existing.Status))
		}

		if err := s.poRepo.UpdateStatus(ctx, tx, poID, "sent"); err != nil {
			return err
		}

		po, err = s.poRepo.FindByID(ctx, tx, poID)
		if err != nil {
			return err
		}
		items, err := s.poItemRepo.ListByPOID(ctx, tx, poID)
		if err != nil {
			return err
		}
		if items == nil {
			items = []model.PurchaseOrderItem{}
		}
		po.Items = items

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "purchase_order.sent",
			EntityType: "purchase_order",
			EntityID:   poID,
			Changes:    map[string]string{"status": "sent"},
			IPAddress:  ip,
		})
	})
	if err != nil {
		return nil, err
	}

	if s.webhookDispatch != nil {
		asyncutil.SafeGo(func() { s.webhookDispatch.Dispatch(context.Background(), tenantID, "purchase_order.sent", po) })
	}
	return po, nil
}

// ReceiveItems receives items against a purchase order, updates stock, and manages status.
func (s *PurchaseOrderService) ReceiveItems(ctx context.Context, tenantID, poID uuid.UUID, req model.ReceiveItemsRequest, actorID uuid.UUID, ip string) (*model.PurchaseOrder, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	var po *model.PurchaseOrder
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		existing, err := s.poRepo.FindByID(ctx, tx, poID)
		if err != nil {
			return err
		}
		if existing == nil {
			return ErrPurchaseOrderNotFound
		}
		if existing.Status != "sent" && existing.Status != "partially_received" {
			return NewValidationError(fmt.Errorf("can only receive items for sent or partially_received orders, current status: %s", existing.Status))
		}

		// Process each received item
		for _, entry := range req.Items {
			item, err := s.poItemRepo.FindByID(ctx, tx, entry.ItemID)
			if err != nil {
				return err
			}
			if item == nil {
				return ErrPOItemNotFound
			}
			if item.PurchaseOrderID != poID {
				return NewValidationError(fmt.Errorf("item %s does not belong to this purchase order", entry.ItemID))
			}

			newReceived := item.QuantityReceived + entry.QuantityReceived
			if newReceived > item.QuantityOrdered {
				return NewValidationError(fmt.Errorf("received quantity (%d) would exceed ordered quantity (%d) for item %s",
					newReceived, item.QuantityOrdered, entry.ItemID))
			}

			if err := s.poItemRepo.UpdateReceived(ctx, tx, entry.ItemID, newReceived); err != nil {
				return err
			}

			// Update warehouse stock if warehouse_id is set and product_id is linked
			if existing.WarehouseID != nil && item.ProductID != nil {
				if err := s.warehouseStock.AdjustQuantity(ctx, tx, *existing.WarehouseID, *item.ProductID, nil, entry.QuantityReceived); err != nil {
					s.logger.Error("failed to adjust warehouse stock",
						"warehouse_id", existing.WarehouseID,
						"product_id", item.ProductID,
						"quantity", entry.QuantityReceived,
						"error", err)
					// Don't fail the whole operation for stock update failure
				}
			}
		}

		// Determine new status based on all items
		allItems, err := s.poItemRepo.ListByPOID(ctx, tx, poID)
		if err != nil {
			return err
		}

		allReceived := true
		anyReceived := false
		for _, item := range allItems {
			if item.QuantityReceived > 0 {
				anyReceived = true
			}
			if item.QuantityReceived < item.QuantityOrdered {
				allReceived = false
			}
		}

		var newStatus string
		if allReceived {
			newStatus = "received"
		} else if anyReceived {
			newStatus = "partially_received"
		} else {
			newStatus = existing.Status
		}

		if newStatus != existing.Status {
			if err := s.poRepo.UpdateStatus(ctx, tx, poID, newStatus); err != nil {
				return err
			}
		}

		po, err = s.poRepo.FindByID(ctx, tx, poID)
		if err != nil {
			return err
		}
		if allItems == nil {
			allItems = []model.PurchaseOrderItem{}
		}
		po.Items = allItems

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "purchase_order.items_received",
			EntityType: "purchase_order",
			EntityID:   poID,
			Changes:    map[string]string{"status": newStatus},
			IPAddress:  ip,
		})
	})
	if err != nil {
		return nil, err
	}

	if s.webhookDispatch != nil {
		asyncutil.SafeGo(func() {
			s.webhookDispatch.Dispatch(context.Background(), tenantID, "purchase_order.items_received", po)
		})
	}
	return po, nil
}

// Cancel cancels a purchase order.
func (s *PurchaseOrderService) Cancel(ctx context.Context, tenantID, poID uuid.UUID, actorID uuid.UUID, ip string) (*model.PurchaseOrder, error) {
	var po *model.PurchaseOrder
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		existing, err := s.poRepo.FindByID(ctx, tx, poID)
		if err != nil {
			return err
		}
		if existing == nil {
			return ErrPurchaseOrderNotFound
		}
		if existing.Status == "cancelled" {
			return ErrPOAlreadyCancelled
		}
		if existing.Status == "received" {
			return NewValidationError(fmt.Errorf("cannot cancel a fully received purchase order"))
		}

		if err := s.poRepo.UpdateStatus(ctx, tx, poID, "cancelled"); err != nil {
			return err
		}

		po, err = s.poRepo.FindByID(ctx, tx, poID)
		if err != nil {
			return err
		}
		items, err := s.poItemRepo.ListByPOID(ctx, tx, poID)
		if err != nil {
			return err
		}
		if items == nil {
			items = []model.PurchaseOrderItem{}
		}
		po.Items = items

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "purchase_order.cancelled",
			EntityType: "purchase_order",
			EntityID:   poID,
			Changes:    map[string]string{"status": "cancelled"},
			IPAddress:  ip,
		})
	})
	if err != nil {
		return nil, err
	}

	if s.webhookDispatch != nil {
		asyncutil.SafeGo(func() { s.webhookDispatch.Dispatch(context.Background(), tenantID, "purchase_order.cancelled", po) })
	}
	return po, nil
}
