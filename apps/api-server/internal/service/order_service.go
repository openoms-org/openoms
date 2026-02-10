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
	ErrOrderNotFound     = errors.New("order not found")
	ErrInvalidTransition = errors.New("invalid status transition")
	ErrUnknownStatus     = errors.New("unknown status")
)

type OrderService struct {
	orderRepo       repository.OrderRepo
	auditRepo       repository.AuditRepo
	tenantRepo      repository.TenantRepo
	pool            *pgxpool.Pool
	emailService    *EmailService
	webhookDispatch *WebhookDispatchService
}

func NewOrderService(
	orderRepo repository.OrderRepo,
	auditRepo repository.AuditRepo,
	tenantRepo repository.TenantRepo,
	pool *pgxpool.Pool,
	emailService *EmailService,
	webhookDispatch *WebhookDispatchService,
) *OrderService {
	return &OrderService{
		orderRepo:       orderRepo,
		auditRepo:       auditRepo,
		tenantRepo:      tenantRepo,
		pool:            pool,
		emailService:    emailService,
		webhookDispatch: webhookDispatch,
	}
}

func (s *OrderService) loadStatusConfig(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID) (*model.OrderStatusConfig, error) {
	settings, err := s.tenantRepo.GetSettings(ctx, tx, tenantID)
	if err != nil {
		return nil, err
	}

	if settings != nil {
		var allSettings map[string]json.RawMessage
		if err := json.Unmarshal(settings, &allSettings); err == nil {
			if raw, ok := allSettings["order_statuses"]; ok {
				var config model.OrderStatusConfig
				if err := json.Unmarshal(raw, &config); err == nil && len(config.Statuses) > 0 {
					return &config, nil
				}
			}
		}
	}

	cfg := model.DefaultOrderStatusConfig()
	return &cfg, nil
}

func (s *OrderService) List(ctx context.Context, tenantID uuid.UUID, filter model.OrderListFilter) (model.ListResponse[model.Order], error) {
	var resp model.ListResponse[model.Order]
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		orders, total, err := s.orderRepo.List(ctx, tx, filter)
		if err != nil {
			return err
		}
		if orders == nil {
			orders = []model.Order{}
		}
		resp = model.ListResponse[model.Order]{
			Items:  orders,
			Total:  total,
			Limit:  filter.Limit,
			Offset: filter.Offset,
		}
		return nil
	})
	return resp, err
}

func (s *OrderService) Get(ctx context.Context, tenantID, orderID uuid.UUID) (*model.Order, error) {
	var order *model.Order
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var err error
		order, err = s.orderRepo.FindByID(ctx, tx, orderID)
		return err
	})
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, ErrOrderNotFound
	}
	return order, nil
}

func (s *OrderService) Create(ctx context.Context, tenantID uuid.UUID, req model.CreateOrderRequest, actorID uuid.UUID, ip string) (*model.Order, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation: %w", err)
	}

	// Default NOT NULL jsonb fields to avoid inserting NULL
	shippingAddr := req.ShippingAddress
	if shippingAddr == nil {
		shippingAddr = json.RawMessage("{}")
	}
	items := req.Items
	if items == nil {
		items = json.RawMessage("[]")
	}
	metadata := req.Metadata
	if metadata == nil {
		metadata = json.RawMessage("{}")
	}
	orderedAt := req.OrderedAt
	if orderedAt == nil {
		now := time.Now()
		orderedAt = &now
	}

	tags := req.Tags
	if tags == nil {
		tags = []string{}
	}

	order := &model.Order{
		ID:              uuid.New(),
		TenantID:        tenantID,
		ExternalID:      req.ExternalID,
		Source:          req.Source,
		IntegrationID:   req.IntegrationID,
		Status:          "new",
		CustomerName:    req.CustomerName,
		CustomerEmail:   req.CustomerEmail,
		CustomerPhone:   req.CustomerPhone,
		ShippingAddress: shippingAddr,
		BillingAddress:  req.BillingAddress,
		Items:           items,
		TotalAmount:     req.TotalAmount,
		Currency:        req.Currency,
		Notes:           req.Notes,
		Metadata:        metadata,
		Tags:            tags,
		DeliveryMethod:  req.DeliveryMethod,
		PickupPointID:   req.PickupPointID,
		OrderedAt:       orderedAt,
	}

	if req.PaymentStatus != nil {
		order.PaymentStatus = *req.PaymentStatus
	} else {
		order.PaymentStatus = "pending"
	}
	if req.PaymentMethod != nil {
		order.PaymentMethod = req.PaymentMethod
	}

	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		if err := s.orderRepo.Create(ctx, tx, order); err != nil {
			return err
		}
		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "order.created",
			EntityType: "order",
			EntityID:   order.ID,
			Changes:    map[string]string{"source": req.Source, "customer_name": req.CustomerName},
			IPAddress:  ip,
		})
	})
	if err != nil {
		return nil, err
	}
	go s.webhookDispatch.Dispatch(context.Background(), tenantID, "order.created", order)
	return order, nil
}

func (s *OrderService) Update(ctx context.Context, tenantID, orderID uuid.UUID, req model.UpdateOrderRequest, actorID uuid.UUID, ip string) (*model.Order, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation: %w", err)
	}

	var order *model.Order
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		existing, err := s.orderRepo.FindByID(ctx, tx, orderID)
		if err != nil {
			return err
		}
		if existing == nil {
			return ErrOrderNotFound
		}

		if err := s.orderRepo.Update(ctx, tx, orderID, req); err != nil {
			return err
		}

		order, err = s.orderRepo.FindByID(ctx, tx, orderID)
		if err != nil {
			return err
		}

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "order.updated",
			EntityType: "order",
			EntityID:   orderID,
			IPAddress:  ip,
		})
	})
	if err == nil && order != nil {
		go s.webhookDispatch.Dispatch(context.Background(), tenantID, "order.updated", order)
	}
	return order, err
}

func (s *OrderService) Delete(ctx context.Context, tenantID, orderID, actorID uuid.UUID, ip string) error {
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		order, err := s.orderRepo.FindByID(ctx, tx, orderID)
		if err != nil {
			return err
		}
		if order == nil {
			return ErrOrderNotFound
		}

		if err := s.orderRepo.Delete(ctx, tx, orderID); err != nil {
			return err
		}

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "order.deleted",
			EntityType: "order",
			EntityID:   orderID,
			Changes:    map[string]string{"external_id": stringOrEmpty(order.ExternalID)},
			IPAddress:  ip,
		})
	})
	if err == nil {
		go s.webhookDispatch.Dispatch(context.Background(), tenantID, "order.deleted", map[string]any{"order_id": orderID.String()})
	}
	return err
}

func (s *OrderService) TransitionStatus(ctx context.Context, tenantID, orderID uuid.UUID, req model.StatusTransitionRequest, actorID uuid.UUID, ip string) (*model.Order, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation: %w", err)
	}

	var order *model.Order
	var oldStatus string
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		existing, err := s.orderRepo.FindByID(ctx, tx, orderID)
		if err != nil {
			return err
		}
		if existing == nil {
			return ErrOrderNotFound
		}
		oldStatus = existing.Status

		config, err := s.loadStatusConfig(ctx, tx, tenantID)
		if err != nil {
			return fmt.Errorf("load status config: %w", err)
		}

		if !config.IsValidStatus(req.Status) {
			return fmt.Errorf("%w: %q", ErrUnknownStatus, req.Status)
		}

		var setShippedAt, setDeliveredAt *time.Time

		if req.Force {
			// Force mode: skip transition validation
		} else {
			if !config.IsValidStatus(existing.Status) {
				return fmt.Errorf("%w: current %q", ErrUnknownStatus, existing.Status)
			}
			if !config.CanTransition(existing.Status, req.Status) {
				return fmt.Errorf("%w: %s -> %s", ErrInvalidTransition, existing.Status, req.Status)
			}
		}

		// Set timestamps for special statuses
		if req.Status == "shipped" {
			now := time.Now()
			setShippedAt = &now
		}
		if req.Status == "delivered" {
			now := time.Now()
			setDeliveredAt = &now
		}

		if err := s.orderRepo.UpdateStatus(ctx, tx, orderID, req.Status, setShippedAt, setDeliveredAt); err != nil {
			return err
		}

		order, err = s.orderRepo.FindByID(ctx, tx, orderID)
		if err != nil {
			return err
		}

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "order.status_changed",
			EntityType: "order",
			EntityID:   orderID,
			Changes:    map[string]string{"from": existing.Status, "to": req.Status},
			IPAddress:  ip,
		})
	})
	if err == nil && order != nil {
		go s.emailService.SendOrderStatusEmail(context.Background(), tenantID, order, oldStatus, req.Status)
		go s.webhookDispatch.Dispatch(context.Background(), tenantID, "order.status_changed", map[string]any{"order_id": orderID.String(), "from": oldStatus, "to": req.Status})
	}
	return order, err
}

func (s *OrderService) BulkTransitionStatus(ctx context.Context, tenantID uuid.UUID, req model.BulkStatusTransitionRequest, actorID uuid.UUID, ip string) (*model.BulkStatusTransitionResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation: %w", err)
	}

	resp := &model.BulkStatusTransitionResponse{
		Results: make([]model.BulkStatusResult, 0, len(req.OrderIDs)),
	}

	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		config, err := s.loadStatusConfig(ctx, tx, tenantID)
		if err != nil {
			return fmt.Errorf("load status config: %w", err)
		}

		if !config.IsValidStatus(req.Status) {
			return fmt.Errorf("%w: %q", ErrUnknownStatus, req.Status)
		}

		for _, orderID := range req.OrderIDs {
			result := model.BulkStatusResult{OrderID: orderID}

			existing, err := s.orderRepo.FindByID(ctx, tx, orderID)
			if err != nil || existing == nil {
				result.Error = "order not found"
				resp.Results = append(resp.Results, result)
				resp.Failed++
				continue
			}

			var setShippedAt, setDeliveredAt *time.Time

			if req.Force {
				// Force mode: skip transition validation
			} else {
				if !config.CanTransition(existing.Status, req.Status) {
					result.Error = fmt.Sprintf("invalid transition: %s -> %s", existing.Status, req.Status)
					resp.Results = append(resp.Results, result)
					resp.Failed++
					continue
				}
			}

			if req.Status == "shipped" {
				now := time.Now()
				setShippedAt = &now
			}
			if req.Status == "delivered" {
				now := time.Now()
				setDeliveredAt = &now
			}

			oldStatus := existing.Status

			if err := s.orderRepo.UpdateStatus(ctx, tx, orderID, req.Status, setShippedAt, setDeliveredAt); err != nil {
				result.Error = "failed to update status"
				resp.Results = append(resp.Results, result)
				resp.Failed++
				continue
			}

			_ = s.auditRepo.Log(ctx, tx, model.AuditEntry{
				TenantID:   tenantID,
				UserID:     actorID,
				Action:     "order.status_changed",
				EntityType: "order",
				EntityID:   orderID,
				Changes:    map[string]string{"from": existing.Status, "to": req.Status},
				IPAddress:  ip,
			})

			updated, err := s.orderRepo.FindByID(ctx, tx, orderID)
			if err == nil && updated != nil {
				go s.emailService.SendOrderStatusEmail(context.Background(), tenantID, updated, oldStatus, req.Status)
			}

			result.Success = true
			resp.Results = append(resp.Results, result)
			resp.Succeeded++

			go s.webhookDispatch.Dispatch(context.Background(), tenantID, "order.status_changed", map[string]any{"order_id": orderID.String(), "from": oldStatus, "to": req.Status})
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (s *OrderService) GetAudit(ctx context.Context, tenantID, orderID uuid.UUID) ([]model.AuditLogEntry, error) {
	var entries []model.AuditLogEntry
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var err error
		entries, err = s.auditRepo.ListByEntity(ctx, tx, "order", orderID)
		return err
	})
	return entries, err
}

func stringOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
