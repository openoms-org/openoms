package service

import (
	"context"
	"encoding/json"
	"errors"
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
	ErrRecurringOrderNotFound = errors.New("recurring order not found")
	ErrRecurringOrderNotActive = errors.New("recurring order is not active")
	ErrRecurringOrderHasOrders = errors.New("cannot delete recurring order that has created orders")
)

type RecurringOrderService struct {
	recurringRepo   repository.RecurringOrderRepo
	orderRepo       repository.OrderRepo
	auditRepo       repository.AuditRepo
	pool            *pgxpool.Pool
	webhookDispatch *WebhookDispatchService
	logger          *slog.Logger
}

func NewRecurringOrderService(
	recurringRepo repository.RecurringOrderRepo,
	orderRepo repository.OrderRepo,
	auditRepo repository.AuditRepo,
	pool *pgxpool.Pool,
	webhookDispatch *WebhookDispatchService,
	logger *slog.Logger,
) *RecurringOrderService {
	return &RecurringOrderService{
		recurringRepo:   recurringRepo,
		orderRepo:       orderRepo,
		auditRepo:       auditRepo,
		pool:            pool,
		webhookDispatch: webhookDispatch,
		logger:          logger,
	}
}

func (s *RecurringOrderService) List(ctx context.Context, tenantID uuid.UUID, filter model.RecurringOrderListFilter) (model.ListResponse[model.RecurringOrder], error) {
	var resp model.ListResponse[model.RecurringOrder]
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		orders, total, err := s.recurringRepo.List(ctx, tx, filter)
		if err != nil {
			return err
		}
		if orders == nil {
			orders = []model.RecurringOrder{}
		}
		resp = model.ListResponse[model.RecurringOrder]{
			Items:  orders,
			Total:  total,
			Limit:  filter.Limit,
			Offset: filter.Offset,
		}
		return nil
	})
	return resp, err
}

func (s *RecurringOrderService) Get(ctx context.Context, tenantID, id uuid.UUID) (*model.RecurringOrder, error) {
	var ro *model.RecurringOrder
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var err error
		ro, err = s.recurringRepo.FindByID(ctx, tx, id)
		if err != nil {
			return err
		}
		if ro == nil {
			return ErrRecurringOrderNotFound
		}
		// Load items
		items, err := s.recurringRepo.ListItems(ctx, tx, id)
		if err != nil {
			return err
		}
		if items == nil {
			items = []model.RecurringOrderItem{}
		}
		ro.Items = items
		return nil
	})
	if err != nil {
		return nil, err
	}
	return ro, nil
}

func (s *RecurringOrderService) Create(ctx context.Context, tenantID uuid.UUID, req model.CreateRecurringOrderRequest, actorID uuid.UUID, ip string) (*model.RecurringOrder, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	req.CustomerName = model.StripHTMLTags(req.CustomerName)

	nextDate, _ := time.Parse("2006-01-02", req.NextOrderDate)
	var endDate *time.Time
	if req.EndDate != nil {
		ed, _ := time.Parse("2006-01-02", *req.EndDate)
		endDate = &ed
	}

	shippingAddr := req.ShippingAddress
	if shippingAddr == nil {
		shippingAddr = json.RawMessage("{}")
	}

	ro := &model.RecurringOrder{
		ID:              uuid.New(),
		TenantID:        tenantID,
		CustomerID:      req.CustomerID,
		CustomerName:    req.CustomerName,
		CustomerEmail:   req.CustomerEmail,
		Status:          "active",
		Frequency:       req.Frequency,
		IntervalDays:    model.FrequencyToIntervalDays(req.Frequency),
		NextOrderDate:   nextDate,
		EndDate:         endDate,
		MaxOrders:       req.MaxOrders,
		ShippingAddress: shippingAddr,
		Notes:           req.Notes,
		CreatedBy:       &actorID,
	}

	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		if err := s.recurringRepo.Create(ctx, tx, ro); err != nil {
			return err
		}

		// Create items
		for _, itemReq := range req.Items {
			item := &model.RecurringOrderItem{
				ID:               uuid.New(),
				TenantID:         tenantID,
				RecurringOrderID: ro.ID,
				ProductID:        itemReq.ProductID,
				SKU:              itemReq.SKU,
				ProductName:      itemReq.ProductName,
				Quantity:         itemReq.Quantity,
				UnitPrice:        itemReq.UnitPrice,
			}
			if err := s.recurringRepo.CreateItem(ctx, tx, item); err != nil {
				return err
			}
			ro.Items = append(ro.Items, *item)
		}

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "recurring_order.created",
			EntityType: "recurring_order",
			EntityID:   ro.ID,
			Changes:    map[string]string{"customer_name": req.CustomerName, "frequency": req.Frequency},
			IPAddress:  ip,
		})
	})
	if err != nil {
		return nil, err
	}
	go s.webhookDispatch.Dispatch(context.Background(), tenantID, "recurring_order.created", ro)
	return ro, nil
}

func (s *RecurringOrderService) Update(ctx context.Context, tenantID, id uuid.UUID, req model.UpdateRecurringOrderRequest, actorID uuid.UUID, ip string) (*model.RecurringOrder, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	var ro *model.RecurringOrder
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		existing, err := s.recurringRepo.FindByID(ctx, tx, id)
		if err != nil {
			return err
		}
		if existing == nil {
			return ErrRecurringOrderNotFound
		}

		if err := s.recurringRepo.Update(ctx, tx, id, req); err != nil {
			return err
		}

		// Replace items if provided
		if req.Items != nil {
			if err := s.recurringRepo.DeleteItemsByRecurringOrderID(ctx, tx, id); err != nil {
				return err
			}
			for _, itemReq := range *req.Items {
				item := &model.RecurringOrderItem{
					ID:               uuid.New(),
					TenantID:         tenantID,
					RecurringOrderID: id,
					ProductID:        itemReq.ProductID,
					SKU:              itemReq.SKU,
					ProductName:      itemReq.ProductName,
					Quantity:         itemReq.Quantity,
					UnitPrice:        itemReq.UnitPrice,
				}
				if err := s.recurringRepo.CreateItem(ctx, tx, item); err != nil {
					return err
				}
			}
		}

		ro, err = s.recurringRepo.FindByID(ctx, tx, id)
		if err != nil {
			return err
		}
		items, err := s.recurringRepo.ListItems(ctx, tx, id)
		if err != nil {
			return err
		}
		if items == nil {
			items = []model.RecurringOrderItem{}
		}
		ro.Items = items

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "recurring_order.updated",
			EntityType: "recurring_order",
			EntityID:   id,
			IPAddress:  ip,
		})
	})
	if err != nil {
		return nil, err
	}
	if ro != nil {
		go s.webhookDispatch.Dispatch(context.Background(), tenantID, "recurring_order.updated", ro)
	}
	return ro, nil
}

func (s *RecurringOrderService) Delete(ctx context.Context, tenantID, id uuid.UUID, actorID uuid.UUID, ip string) error {
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		existing, err := s.recurringRepo.FindByID(ctx, tx, id)
		if err != nil {
			return err
		}
		if existing == nil {
			return ErrRecurringOrderNotFound
		}
		if existing.TotalOrdersCreated > 0 {
			return ErrRecurringOrderHasOrders
		}

		if err := s.recurringRepo.Delete(ctx, tx, id); err != nil {
			return err
		}

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "recurring_order.deleted",
			EntityType: "recurring_order",
			EntityID:   id,
			Changes:    map[string]string{"customer_name": existing.CustomerName},
			IPAddress:  ip,
		})
	})
	if err == nil {
		go s.webhookDispatch.Dispatch(context.Background(), tenantID, "recurring_order.deleted", map[string]any{"recurring_order_id": id.String()})
	}
	return err
}

func (s *RecurringOrderService) Pause(ctx context.Context, tenantID, id uuid.UUID, actorID uuid.UUID, ip string) error {
	return database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		existing, err := s.recurringRepo.FindByID(ctx, tx, id)
		if err != nil {
			return err
		}
		if existing == nil {
			return ErrRecurringOrderNotFound
		}
		if existing.Status != "active" {
			return ErrRecurringOrderNotActive
		}
		if err := s.recurringRepo.UpdateStatus(ctx, tx, id, "paused"); err != nil {
			return err
		}
		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "recurring_order.paused",
			EntityType: "recurring_order",
			EntityID:   id,
			IPAddress:  ip,
		})
	})
}

func (s *RecurringOrderService) Resume(ctx context.Context, tenantID, id uuid.UUID, actorID uuid.UUID, ip string) error {
	return database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		existing, err := s.recurringRepo.FindByID(ctx, tx, id)
		if err != nil {
			return err
		}
		if existing == nil {
			return ErrRecurringOrderNotFound
		}
		if existing.Status != "paused" {
			return NewValidationError(errors.New("recurring order is not paused"))
		}
		if err := s.recurringRepo.UpdateStatus(ctx, tx, id, "active"); err != nil {
			return err
		}
		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "recurring_order.resumed",
			EntityType: "recurring_order",
			EntityID:   id,
			IPAddress:  ip,
		})
	})
}

func (s *RecurringOrderService) Cancel(ctx context.Context, tenantID, id uuid.UUID, actorID uuid.UUID, ip string) error {
	return database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		existing, err := s.recurringRepo.FindByID(ctx, tx, id)
		if err != nil {
			return err
		}
		if existing == nil {
			return ErrRecurringOrderNotFound
		}
		if existing.Status == "cancelled" || existing.Status == "completed" {
			return NewValidationError(errors.New("recurring order is already " + existing.Status))
		}
		if err := s.recurringRepo.UpdateStatus(ctx, tx, id, "cancelled"); err != nil {
			return err
		}
		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "recurring_order.cancelled",
			EntityType: "recurring_order",
			EntityID:   id,
			IPAddress:  ip,
		})
	})
}

// ProcessDueOrders finds all due recurring orders for a tenant and creates orders from them.
// Returns the count of orders created.
func (s *RecurringOrderService) ProcessDueOrders(ctx context.Context, tenantID uuid.UUID) (int, error) {
	var created int
	today := time.Now().Truncate(24 * time.Hour)

	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		dueOrders, err := s.recurringRepo.FindDue(ctx, tx, today)
		if err != nil {
			return err
		}

		for _, ro := range dueOrders {
			// Load items for this recurring order
			items, err := s.recurringRepo.ListItems(ctx, tx, ro.ID)
			if err != nil {
				s.logger.Error("recurring order worker: failed to load items",
					"recurring_order_id", ro.ID,
					"error", err,
				)
				continue
			}

			// Build order items JSON
			orderItems := buildOrderItemsJSON(items)

			// Calculate total amount
			var totalAmount float64
			for _, item := range items {
				totalAmount += float64(item.Quantity) * item.UnitPrice
			}

			// Create order via repository (we're already in a tenant tx)
			source := "recurring"
			orderReq := model.CreateOrderRequest{
				Source:          source,
				CustomerName:    ro.CustomerName,
				CustomerEmail:   ro.CustomerEmail,
				ShippingAddress: ro.ShippingAddress,
				Items:           orderItems,
				TotalAmount:     totalAmount,
				Currency:        "PLN",
				Notes:           ro.Notes,
			}
			if err := orderReq.Validate(); err != nil {
				s.logger.Error("recurring order worker: invalid order request",
					"recurring_order_id", ro.ID,
					"error", err,
				)
				continue
			}

			// Build order model
			now := time.Now()
			shippingAddr := ro.ShippingAddress
			if shippingAddr == nil {
				shippingAddr = json.RawMessage("{}")
			}
			order := &model.Order{
				ID:              uuid.New(),
				TenantID:        tenantID,
				Source:          source,
				Status:          "new",
				CustomerName:    ro.CustomerName,
				CustomerEmail:   ro.CustomerEmail,
				CustomerID:      ro.CustomerID,
				ShippingAddress: shippingAddr,
				BillingAddress:  json.RawMessage("{}"),
				Items:           orderItems,
				TotalAmount:     totalAmount,
				Currency:        "PLN",
				Notes:           ro.Notes,
				Metadata:        json.RawMessage("{}"),
				Tags:            []string{},
				PaymentStatus:   "unpaid",
				OrderedAt:       &now,
				InternalNotes:   "",
				Priority:        "normal",
			}
			if err := s.orderRepo.Create(ctx, tx, order); err != nil {
				s.logger.Error("recurring order worker: failed to create order",
					"recurring_order_id", ro.ID,
					"error", err,
				)
				continue
			}

			// Calculate next date
			newCount := ro.TotalOrdersCreated + 1
			nextDate := calculateNextDate(ro.NextOrderDate, ro.Frequency)

			// Check if we should complete this recurring order
			shouldComplete := false
			if ro.MaxOrders != nil && newCount >= *ro.MaxOrders {
				shouldComplete = true
			}
			if ro.EndDate != nil && nextDate.After(*ro.EndDate) {
				shouldComplete = true
			}

			if shouldComplete {
				if err := s.recurringRepo.UpdateStatus(ctx, tx, ro.ID, "completed"); err != nil {
					s.logger.Error("recurring order worker: failed to complete recurring order",
						"recurring_order_id", ro.ID,
						"error", err,
					)
				}
			}

			if err := s.recurringRepo.UpdateAfterCreation(ctx, tx, ro.ID, nextDate, newCount); err != nil {
				s.logger.Error("recurring order worker: failed to update recurring order",
					"recurring_order_id", ro.ID,
					"error", err,
				)
				continue
			}

			created++
		}
		return nil
	})
	return created, err
}

// calculateNextDate computes the next order date based on frequency.
func calculateNextDate(from time.Time, frequency string) time.Time {
	days := model.FrequencyToIntervalDays(frequency)
	return from.AddDate(0, 0, days)
}

// buildOrderItemsJSON converts recurring order items to the JSON format expected by orders.items.
func buildOrderItemsJSON(items []model.RecurringOrderItem) json.RawMessage {
	type orderItem struct {
		Name     string  `json:"name"`
		SKU      string  `json:"sku"`
		Quantity int     `json:"quantity"`
		Price    float64 `json:"price"`
	}

	var orderItems []orderItem
	for _, item := range items {
		orderItems = append(orderItems, orderItem{
			Name:     item.ProductName,
			SKU:      item.SKU,
			Quantity: item.Quantity,
			Price:    item.UnitPrice,
		})
	}

	data, _ := json.Marshal(orderItems)
	return data
}
