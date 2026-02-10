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
	engine "github.com/openoms-org/openoms/packages/order-engine"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

var (
	ErrOrderNotFound = errors.New("order not found")
)

type OrderService struct {
	orderRepo    *repository.OrderRepository
	auditRepo    *repository.AuditRepository
	pool         *pgxpool.Pool
	emailService *EmailService
}

func NewOrderService(
	orderRepo *repository.OrderRepository,
	auditRepo *repository.AuditRepository,
	pool *pgxpool.Pool,
	emailService *EmailService,
) *OrderService {
	return &OrderService{
		orderRepo:    orderRepo,
		auditRepo:    auditRepo,
		pool:         pool,
		emailService: emailService,
	}
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
	return order, err
}

func (s *OrderService) Delete(ctx context.Context, tenantID, orderID, actorID uuid.UUID, ip string) error {
	return database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
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

		var setShippedAt, setDeliveredAt *time.Time

		if req.Force {
			// Force mode: skip transition validation, just validate target is a known status
			if _, err := engine.ParseOrderStatus(req.Status); err != nil {
				return err
			}
		} else {
			currentStatus, err := engine.ParseOrderStatus(existing.Status)
			if err != nil {
				return err
			}

			targetStatus, err := engine.ParseOrderStatus(req.Status)
			if err != nil {
				return err
			}

			result, err := engine.TransitionOrder(currentStatus, targetStatus, time.Now())
			if err != nil {
				return err
			}
			setShippedAt = result.SetShippedAt
			setDeliveredAt = result.SetDeliveredAt
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
		targetStatus, err := engine.ParseOrderStatus(req.Status)
		if err != nil {
			return fmt.Errorf("validation: invalid target status: %w", err)
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
				currentStatus, err := engine.ParseOrderStatus(existing.Status)
				if err != nil {
					result.Error = fmt.Sprintf("invalid current status: %s", existing.Status)
					resp.Results = append(resp.Results, result)
					resp.Failed++
					continue
				}

				transitionResult, err := engine.TransitionOrder(currentStatus, targetStatus, time.Now())
				if err != nil {
					result.Error = err.Error()
					resp.Results = append(resp.Results, result)
					resp.Failed++
					continue
				}
				setShippedAt = transitionResult.SetShippedAt
				setDeliveredAt = transitionResult.SetDeliveredAt
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
