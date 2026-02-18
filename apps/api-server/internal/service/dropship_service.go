package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

var (
	ErrDropshipOrderNotFound     = errors.New("dropship order not found")
	ErrDropshipAlreadyCancelled  = errors.New("dropship order is already cancelled")
	ErrDropshipInvalidTransition = errors.New("invalid dropship status transition")
	ErrNoDropshipItems           = errors.New("no dropship items found for this order")
)

// DropshipService handles business logic for dropship orders.
type DropshipService struct {
	dropshipRepo     repository.DropshipOrderRepo
	dropshipItemRepo repository.DropshipOrderItemRepo
	orderRepo        repository.OrderRepo
	productRepo      repository.ProductRepo
	supplierRepo     repository.SupplierRepo
	auditRepo        repository.AuditRepo
	pool             *pgxpool.Pool
	webhookDispatch  *WebhookDispatchService
	logger           *slog.Logger
}

// NewDropshipService creates a new DropshipService.
func NewDropshipService(
	dropshipRepo repository.DropshipOrderRepo,
	dropshipItemRepo repository.DropshipOrderItemRepo,
	orderRepo repository.OrderRepo,
	productRepo repository.ProductRepo,
	supplierRepo repository.SupplierRepo,
	auditRepo repository.AuditRepo,
	pool *pgxpool.Pool,
	webhookDispatch *WebhookDispatchService,
	logger *slog.Logger,
) *DropshipService {
	return &DropshipService{
		dropshipRepo:     dropshipRepo,
		dropshipItemRepo: dropshipItemRepo,
		orderRepo:        orderRepo,
		productRepo:      productRepo,
		supplierRepo:     supplierRepo,
		auditRepo:        auditRepo,
		pool:             pool,
		webhookDispatch:  webhookDispatch,
		logger:           logger,
	}
}

// List returns a paginated list of dropship orders.
func (s *DropshipService) List(ctx context.Context, tenantID uuid.UUID, filter model.DropshipOrderListFilter) (model.ListResponse[model.DropshipOrder], error) {
	var resp model.ListResponse[model.DropshipOrder]
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		orders, total, err := s.dropshipRepo.List(ctx, tx, filter)
		if err != nil {
			return err
		}
		if orders == nil {
			orders = []model.DropshipOrder{}
		}
		resp = model.ListResponse[model.DropshipOrder]{
			Items:  orders,
			Total:  total,
			Limit:  filter.Limit,
			Offset: filter.Offset,
		}
		return nil
	})
	return resp, err
}

// Get returns a single dropship order with its items.
func (s *DropshipService) Get(ctx context.Context, tenantID, id uuid.UUID) (*model.DropshipOrder, error) {
	var d *model.DropshipOrder
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var err error
		d, err = s.dropshipRepo.FindByID(ctx, tx, id)
		if err != nil {
			return err
		}
		if d == nil {
			return ErrDropshipOrderNotFound
		}
		items, err := s.dropshipItemRepo.ListByDropshipOrderID(ctx, tx, id)
		if err != nil {
			return err
		}
		if items == nil {
			items = []model.DropshipOrderItem{}
		}
		d.Items = items
		return nil
	})
	if err != nil {
		return nil, err
	}
	return d, nil
}

// GetByOrderID returns all dropship orders for a customer order.
func (s *DropshipService) GetByOrderID(ctx context.Context, tenantID, orderID uuid.UUID) ([]model.DropshipOrder, error) {
	var orders []model.DropshipOrder
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var err error
		orders, err = s.dropshipRepo.FindByOrderID(ctx, tx, orderID)
		if err != nil {
			return err
		}
		if orders == nil {
			orders = []model.DropshipOrder{}
		}
		// Load items for each dropship order
		for i := range orders {
			items, err := s.dropshipItemRepo.ListByDropshipOrderID(ctx, tx, orders[i].ID)
			if err != nil {
				return err
			}
			if items == nil {
				items = []model.DropshipOrderItem{}
			}
			orders[i].Items = items
		}
		return nil
	})
	return orders, err
}

// dropshipOrderItemJSON is used internally for parsing order items from JSONB.
type dropshipOrderItemJSON struct {
	ProductID *string `json:"product_id"`
	SKU       string  `json:"sku"`
	Name      string  `json:"name"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"`
}

// AutoRouteOrder parses order items, checks which are dropship products,
// groups them by supplier, and creates one dropship order per supplier.
func (s *DropshipService) AutoRouteOrder(ctx context.Context, tenantID, orderID, actorID uuid.UUID, ip string) ([]model.DropshipOrder, error) {
	var result []model.DropshipOrder

	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		// 1. Get the customer order
		order, err := s.orderRepo.FindByID(ctx, tx, orderID)
		if err != nil {
			return err
		}
		if order == nil {
			return NewValidationError(fmt.Errorf("order not found"))
		}

		// 2. Parse items from order JSONB
		var items []dropshipOrderItemJSON
		if order.Items != nil {
			if err := json.Unmarshal(order.Items, &items); err != nil {
				return NewValidationError(fmt.Errorf("failed to parse order items: %w", err))
			}
		}
		if len(items) == 0 {
			return NewValidationError(fmt.Errorf("order has no items"))
		}

		// 3. For each item, check if the product is a dropship product
		type supplierGroup struct {
			supplierID   uuid.UUID
			supplierName string
			items        []model.DropshipOrderItem
			totalCost    float64
		}
		groups := make(map[uuid.UUID]*supplierGroup)

		for _, item := range items {
			if item.ProductID == nil || *item.ProductID == "" {
				continue
			}
			productID, err := uuid.Parse(*item.ProductID)
			if err != nil {
				continue
			}

			product, err := s.productRepo.FindByID(ctx, tx, productID)
			if err != nil || product == nil {
				continue
			}

			if !product.IsDropship || product.DropshipSupplierID == nil {
				continue
			}

			supplierID := *product.DropshipSupplierID

			// Lookup supplier name if not already in group
			if _, ok := groups[supplierID]; !ok {
				supplier, err := s.supplierRepo.FindByID(ctx, tx, supplierID)
				if err != nil || supplier == nil {
					s.logger.Warn("dropship supplier not found", "supplier_id", supplierID)
					continue
				}
				groups[supplierID] = &supplierGroup{
					supplierID:   supplierID,
					supplierName: supplier.Name,
				}
			}

			// Use supplier_products cost price if available, otherwise use order item price
			unitCost := item.Price
			sku := item.SKU
			if product.SKU != nil && *product.SKU != "" {
				sku = *product.SKU
			}

			dsItem := model.DropshipOrderItem{
				ID:          uuid.New(),
				TenantID:    tenantID,
				ProductID:   &productID,
				SKU:         sku,
				ProductName: item.Name,
				Quantity:    item.Quantity,
				UnitCost:    unitCost,
			}

			groups[supplierID].items = append(groups[supplierID].items, dsItem)
			groups[supplierID].totalCost += float64(item.Quantity) * unitCost
		}

		if len(groups) == 0 {
			return ErrNoDropshipItems
		}

		// 4. Create one dropship order per supplier
		for _, group := range groups {
			dsOrder := &model.DropshipOrder{
				ID:           uuid.New(),
				TenantID:     tenantID,
				OrderID:      orderID,
				SupplierID:   group.supplierID,
				SupplierName: group.supplierName,
				Status:       "pending",
				TotalCost:    group.totalCost,
				Currency:     order.Currency,
			}

			if err := s.dropshipRepo.Create(ctx, tx, dsOrder); err != nil {
				return fmt.Errorf("create dropship order: %w", err)
			}

			var createdItems []model.DropshipOrderItem
			for _, item := range group.items {
				item.DropshipOrderID = dsOrder.ID
				if err := s.dropshipItemRepo.CreateItem(ctx, tx, &item); err != nil {
					return fmt.Errorf("create dropship item: %w", err)
				}
				createdItems = append(createdItems, item)
			}
			dsOrder.Items = createdItems

			if err := s.auditRepo.Log(ctx, tx, model.AuditEntry{
				TenantID:   tenantID,
				UserID:     actorID,
				Action:     "dropship_order.created",
				EntityType: "dropship_order",
				EntityID:   dsOrder.ID,
				Changes:    map[string]string{"supplier": group.supplierName, "order_id": orderID.String()},
				IPAddress:  ip,
			}); err != nil {
				return err
			}

			result = append(result, *dsOrder)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	if s.webhookDispatch != nil {
		for _, ds := range result {
			go s.webhookDispatch.Dispatch(context.Background(), tenantID, "dropship_order.created", ds)
		}
	}
	return result, nil
}

// Create creates a new dropship order manually.
func (s *DropshipService) Create(ctx context.Context, tenantID uuid.UUID, req model.CreateDropshipOrderRequest, actorID uuid.UUID, ip string) (*model.DropshipOrder, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	var d *model.DropshipOrder
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		// Verify order exists
		order, err := s.orderRepo.FindByID(ctx, tx, req.OrderID)
		if err != nil {
			return err
		}
		if order == nil {
			return NewValidationError(fmt.Errorf("order not found"))
		}

		// Verify supplier exists
		supplier, err := s.supplierRepo.FindByID(ctx, tx, req.SupplierID)
		if err != nil {
			return err
		}
		if supplier == nil {
			return NewValidationError(fmt.Errorf("supplier not found"))
		}

		// Calculate total
		var totalCost float64
		for _, item := range req.Items {
			totalCost += float64(item.Quantity) * item.UnitCost
		}

		d = &model.DropshipOrder{
			ID:           uuid.New(),
			TenantID:     tenantID,
			OrderID:      req.OrderID,
			SupplierID:   req.SupplierID,
			SupplierName: supplier.Name,
			Status:       "pending",
			TotalCost:    totalCost,
			Currency:     req.Currency,
			Notes:        req.Notes,
		}

		if err := s.dropshipRepo.Create(ctx, tx, d); err != nil {
			return fmt.Errorf("create dropship order: %w", err)
		}

		var items []model.DropshipOrderItem
		for _, itemReq := range req.Items {
			item := &model.DropshipOrderItem{
				ID:              uuid.New(),
				TenantID:        tenantID,
				DropshipOrderID: d.ID,
				ProductID:       itemReq.ProductID,
				SKU:             itemReq.SKU,
				ProductName:     model.StripHTMLTags(itemReq.ProductName),
				Quantity:        itemReq.Quantity,
				UnitCost:        itemReq.UnitCost,
			}
			if err := s.dropshipItemRepo.CreateItem(ctx, tx, item); err != nil {
				return fmt.Errorf("create dropship item: %w", err)
			}
			items = append(items, *item)
		}
		d.Items = items

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "dropship_order.created",
			EntityType: "dropship_order",
			EntityID:   d.ID,
			Changes:    map[string]string{"supplier": supplier.Name, "order_id": req.OrderID.String()},
			IPAddress:  ip,
		})
	})
	if err != nil {
		return nil, err
	}

	if s.webhookDispatch != nil {
		go s.webhookDispatch.Dispatch(context.Background(), tenantID, "dropship_order.created", d)
	}
	return d, nil
}

// UpdateStatus updates the status of a dropship order with optional tracking info.
func (s *DropshipService) UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, req model.UpdateDropshipStatusRequest, actorID uuid.UUID, ip string) (*model.DropshipOrder, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	var d *model.DropshipOrder
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		existing, err := s.dropshipRepo.FindByID(ctx, tx, id)
		if err != nil {
			return err
		}
		if existing == nil {
			return ErrDropshipOrderNotFound
		}

		// Validate status transition
		if !isValidDropshipTransition(existing.Status, req.Status) {
			return NewValidationError(fmt.Errorf("cannot transition from %s to %s", existing.Status, req.Status))
		}

		if err := s.dropshipRepo.UpdateFields(ctx, tx, id, req); err != nil {
			return err
		}

		d, err = s.dropshipRepo.FindByID(ctx, tx, id)
		if err != nil {
			return err
		}
		items, err := s.dropshipItemRepo.ListByDropshipOrderID(ctx, tx, id)
		if err != nil {
			return err
		}
		if items == nil {
			items = []model.DropshipOrderItem{}
		}
		d.Items = items

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "dropship_order.status_updated",
			EntityType: "dropship_order",
			EntityID:   id,
			Changes:    map[string]string{"from": existing.Status, "to": req.Status},
			IPAddress:  ip,
		})
	})
	if err != nil {
		return nil, err
	}

	if s.webhookDispatch != nil {
		go s.webhookDispatch.Dispatch(context.Background(), tenantID, "dropship_order.status_updated", d)
	}
	return d, nil
}

// Cancel cancels a dropship order.
func (s *DropshipService) Cancel(ctx context.Context, tenantID, id uuid.UUID, actorID uuid.UUID, ip string) (*model.DropshipOrder, error) {
	var d *model.DropshipOrder
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		existing, err := s.dropshipRepo.FindByID(ctx, tx, id)
		if err != nil {
			return err
		}
		if existing == nil {
			return ErrDropshipOrderNotFound
		}
		if existing.Status == "cancelled" {
			return ErrDropshipAlreadyCancelled
		}
		if existing.Status == "delivered" {
			return NewValidationError(fmt.Errorf("cannot cancel a delivered dropship order"))
		}

		if err := s.dropshipRepo.UpdateStatus(ctx, tx, id, "cancelled"); err != nil {
			return err
		}

		d, err = s.dropshipRepo.FindByID(ctx, tx, id)
		if err != nil {
			return err
		}
		items, err := s.dropshipItemRepo.ListByDropshipOrderID(ctx, tx, id)
		if err != nil {
			return err
		}
		if items == nil {
			items = []model.DropshipOrderItem{}
		}
		d.Items = items

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "dropship_order.cancelled",
			EntityType: "dropship_order",
			EntityID:   id,
			Changes:    map[string]string{"status": "cancelled"},
			IPAddress:  ip,
		})
	})
	if err != nil {
		return nil, err
	}

	if s.webhookDispatch != nil {
		go s.webhookDispatch.Dispatch(context.Background(), tenantID, "dropship_order.cancelled", d)
	}
	return d, nil
}

// isValidDropshipTransition checks if a status transition is valid.
func isValidDropshipTransition(from, to string) bool {
	transitions := map[string][]string{
		"pending":   {"sent", "cancelled"},
		"sent":      {"confirmed", "shipped", "cancelled"},
		"confirmed": {"shipped", "cancelled"},
		"shipped":   {"delivered", "cancelled"},
		"delivered": {},
		"cancelled": {},
	}

	allowed, ok := transitions[from]
	if !ok {
		return false
	}
	return slices.Contains(allowed, to)
}
