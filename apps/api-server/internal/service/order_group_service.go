package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

var (
	ErrOrderGroupNotFound = errors.New("order group not found")
)

type OrderGroupService struct {
	orderGroupRepo *repository.OrderGroupRepository
	orderRepo      repository.OrderRepo
	auditRepo      repository.AuditRepo
	pool           *pgxpool.Pool
}

func NewOrderGroupService(
	orderGroupRepo *repository.OrderGroupRepository,
	orderRepo repository.OrderRepo,
	auditRepo repository.AuditRepo,
	pool *pgxpool.Pool,
) *OrderGroupService {
	return &OrderGroupService{
		orderGroupRepo: orderGroupRepo,
		orderRepo:      orderRepo,
		auditRepo:      auditRepo,
		pool:           pool,
	}
}

func (s *OrderGroupService) MergeOrders(ctx context.Context, tenantID, userID uuid.UUID, req model.MergeOrdersRequest) (*model.OrderGroup, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	var group *model.OrderGroup
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		// Fetch all source orders
		var orders []model.Order
		var totalAmount float64
		var allItems []json.RawMessage
		var currency string

		for _, orderID := range req.OrderIDs {
			order, err := s.orderRepo.FindByID(ctx, tx, orderID)
			if err != nil {
				return err
			}
			if order == nil {
				return NewValidationError(fmt.Errorf("order %s not found", orderID))
			}
			if order.Status == "merged" || order.Status == "split" || order.Status == "cancelled" {
				return NewValidationError(fmt.Errorf("order %s has status %s and cannot be merged", orderID, order.Status))
			}
			orders = append(orders, *order)
			totalAmount += order.TotalAmount
			if currency == "" {
				currency = order.Currency
			}
			if order.Items != nil && string(order.Items) != "null" && string(order.Items) != "[]" {
				allItems = append(allItems, order.Items)
			}
		}

		for i := 1; i < len(orders); i++ {
			e0 := orders[0].CustomerEmail
			ei := orders[i].CustomerEmail
			if (e0 == nil) != (ei == nil) || (e0 != nil && *e0 != *ei) {
				return NewValidationError(fmt.Errorf("wszystkie zamówienia muszą być od tego samego klienta"))
			}
		}

		// Combine items from all orders
		var combinedItems []json.RawMessage
		for _, items := range allItems {
			var parsed []json.RawMessage
			if err := json.Unmarshal(items, &parsed); err != nil {
				// If it's not an array, wrap it as a single item
				combinedItems = append(combinedItems, items)
				continue
			}
			combinedItems = append(combinedItems, parsed...)
		}
		combinedItemsJSON, err := json.Marshal(combinedItems)
		if err != nil {
			return fmt.Errorf("marshal combined items: %w", err)
		}

		// Create the new merged order using data from the first order
		firstOrder := orders[0]
		newOrder := &model.Order{
			ID:              uuid.New(),
			TenantID:        tenantID,
			Source:          "manual",
			Status:          "new",
			CustomerName:    firstOrder.CustomerName,
			CustomerEmail:   firstOrder.CustomerEmail,
			CustomerPhone:   firstOrder.CustomerPhone,
			ShippingAddress: firstOrder.ShippingAddress,
			BillingAddress:  firstOrder.BillingAddress,
			Items:           combinedItemsJSON,
			TotalAmount:     totalAmount,
			Currency:        currency,
			PaymentStatus:   "pending",
			Tags:            []string{"merged"},
		}
		if req.Notes != nil {
			newOrder.Notes = req.Notes
		}
		newOrder.Metadata = json.RawMessage(`{}`)
		if newOrder.OrderedAt == nil {
			now := time.Now()
			newOrder.OrderedAt = &now
		}

		if err := s.orderRepo.Create(ctx, tx, newOrder); err != nil {
			return fmt.Errorf("create merged order: %w", err)
		}

		// Mark source orders as merged
		for _, order := range orders {
			if err := s.orderRepo.UpdateStatus(ctx, tx, order.ID, "merged", nil, nil); err != nil {
				return fmt.Errorf("mark order %s as merged: %w", order.ID, err)
			}
			// Set merged_into reference
			if _, err := tx.Exec(ctx,
				"UPDATE orders SET merged_into = $1, updated_at = NOW() WHERE id = $2",
				newOrder.ID, order.ID,
			); err != nil {
				return fmt.Errorf("set merged_into for order %s: %w", order.ID, err)
			}
		}

		// Create the order group record
		group = &model.OrderGroup{
			ID:             uuid.New(),
			TenantID:       tenantID,
			GroupType:      "merged",
			SourceOrderIDs: req.OrderIDs,
			TargetOrderIDs: []uuid.UUID{newOrder.ID},
			Notes:          req.Notes,
			CreatedBy:      &userID,
		}

		if err := s.orderGroupRepo.Create(ctx, tx, group); err != nil {
			return fmt.Errorf("create order group: %w", err)
		}

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     userID,
			Action:     "orders.merged",
			EntityType: "order_group",
			EntityID:   group.ID,
			Changes:    map[string]string{"source_count": fmt.Sprintf("%d", len(req.OrderIDs)), "target_order": newOrder.ID.String()},
		})
	})
	if err != nil {
		return nil, err
	}
	return group, nil
}

func (s *OrderGroupService) SplitOrder(ctx context.Context, tenantID, userID uuid.UUID, orderID uuid.UUID, req model.SplitOrderRequest) (*model.OrderGroup, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	var group *model.OrderGroup
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		sourceOrder, err := s.orderRepo.FindByID(ctx, tx, orderID)
		if err != nil {
			return err
		}
		if sourceOrder == nil {
			return NewValidationError(errors.New("order not found"))
		}
		if sourceOrder.Status == "merged" || sourceOrder.Status == "split" || sourceOrder.Status == "cancelled" {
			return NewValidationError(fmt.Errorf("order has status %s and cannot be split", sourceOrder.Status))
		}

		var targetIDs []uuid.UUID

		for _, split := range req.Splits {
			// Calculate total for this split
			var items []struct {
				Price    float64 `json:"price"`
				Quantity int     `json:"quantity"`
			}
			if err := json.Unmarshal(split.Items, &items); err != nil {
				// If can't parse items for total calculation, set 0
				items = nil
			}
			var splitTotal float64
			for _, item := range items {
				splitTotal += item.Price * float64(item.Quantity)
			}

			customerName := sourceOrder.CustomerName
			if split.CustomerName != "" {
				customerName = split.CustomerName
			}
			shippingAddr := sourceOrder.ShippingAddress
			if split.ShippingAddress != nil {
				shippingAddr = split.ShippingAddress
			}

			newOrder := &model.Order{
				ID:              uuid.New(),
				TenantID:        tenantID,
				Source:          sourceOrder.Source,
				Status:          "new",
				CustomerName:    customerName,
				CustomerEmail:   sourceOrder.CustomerEmail,
				CustomerPhone:   sourceOrder.CustomerPhone,
				ShippingAddress: shippingAddr,
				BillingAddress:  sourceOrder.BillingAddress,
				Items:           split.Items,
				TotalAmount:     splitTotal,
				Currency:        sourceOrder.Currency,
				PaymentStatus:   "pending",
				Tags:            []string{"split"},
			}
			if req.Notes != nil {
				newOrder.Notes = req.Notes
			}
			newOrder.Metadata = json.RawMessage(`{}`)
			if newOrder.OrderedAt == nil {
				now := time.Now()
				newOrder.OrderedAt = &now
			}

			if err := s.orderRepo.Create(ctx, tx, newOrder); err != nil {
				return fmt.Errorf("create split order: %w", err)
			}

			// Set split_from reference
			if _, err := tx.Exec(ctx,
				"UPDATE orders SET split_from = $1, updated_at = NOW() WHERE id = $2",
				orderID, newOrder.ID,
			); err != nil {
				return fmt.Errorf("set split_from for new order: %w", err)
			}

			targetIDs = append(targetIDs, newOrder.ID)
		}

		// Mark source order as split
		if err := s.orderRepo.UpdateStatus(ctx, tx, orderID, "split", nil, nil); err != nil {
			return fmt.Errorf("mark order as split: %w", err)
		}

		// Create the order group record
		group = &model.OrderGroup{
			ID:             uuid.New(),
			TenantID:       tenantID,
			GroupType:      "split",
			SourceOrderIDs: []uuid.UUID{orderID},
			TargetOrderIDs: targetIDs,
			Notes:          req.Notes,
			CreatedBy:      &userID,
		}

		if err := s.orderGroupRepo.Create(ctx, tx, group); err != nil {
			return fmt.Errorf("create order group: %w", err)
		}

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     userID,
			Action:     "order.split",
			EntityType: "order_group",
			EntityID:   group.ID,
			Changes:    map[string]string{"source_order": orderID.String(), "target_count": fmt.Sprintf("%d", len(targetIDs))},
		})
	})
	if err != nil {
		return nil, err
	}
	return group, nil
}

func (s *OrderGroupService) ListByOrderID(ctx context.Context, tenantID, orderID uuid.UUID) ([]model.OrderGroup, error) {
	var groups []model.OrderGroup
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var err error
		groups, err = s.orderGroupRepo.ListByOrderID(ctx, tx, orderID)
		return err
	})
	if groups == nil {
		groups = []model.OrderGroup{}
	}
	return groups, err
}
