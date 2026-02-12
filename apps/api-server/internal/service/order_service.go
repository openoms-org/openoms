package service

import (
	"context"
	"encoding/json"
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
	ErrOrderNotFound     = errors.New("order not found")
	ErrInvalidTransition = errors.New("invalid status transition")
	ErrUnknownStatus     = errors.New("unknown status")
)

type OrderService struct {
	orderRepo         repository.OrderRepo
	auditRepo         repository.AuditRepo
	tenantRepo        repository.TenantRepo
	pool              *pgxpool.Pool
	emailService      *EmailService
	webhookDispatch   *WebhookDispatchService
	invoiceService    *InvoiceService
	smsService        *SMSService
	automationService *AutomationService
	shipmentService   *ShipmentService
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

// OrderRepo returns the underlying order repository for direct access.
func (s *OrderService) OrderRepo() repository.OrderRepo {
	return s.orderRepo
}

// AuditRepo returns the underlying audit repository for direct access.
func (s *OrderService) AuditRepo() repository.AuditRepo {
	return s.auditRepo
}

// SetInvoiceService sets the invoice service for auto-invoicing on status change.
// Called after both services are constructed to avoid circular dependency.
func (s *OrderService) SetInvoiceService(invoiceSvc *InvoiceService) {
	s.invoiceService = invoiceSvc
}

// SetAutomationService sets the automation service for rule processing.
// Called after construction to avoid circular dependency.
func (s *OrderService) SetAutomationService(automationSvc *AutomationService) {
	s.automationService = automationSvc
}

// SetSMSService sets the SMS service for sending SMS notifications on status change.
func (s *OrderService) SetSMSService(smsSvc *SMSService) {
	s.smsService = smsSvc
}

// SetShipmentService sets the shipment service for auto-creating shipments with orders.
func (s *OrderService) SetShipmentService(shipmentSvc *ShipmentService) {
	s.shipmentService = shipmentSvc
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
		return nil, NewValidationError(err)
	}

	// Sanitize user-facing text fields to prevent stored XSS
	req.CustomerName = model.StripHTMLTags(req.CustomerName)
	if req.Notes != nil {
		sanitized := model.StripHTMLTags(*req.Notes)
		req.Notes = &sanitized
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
		InternalNotes:   req.InternalNotes,
		Priority:        req.Priority,
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
	FireAutomationEvent(s.automationService, tenantID, "order", "order.created", order.ID, map[string]any{
		"status": order.Status, "source": order.Source,
		"customer_name": order.CustomerName, "total_amount": order.TotalAmount,
		"currency": order.Currency, "payment_status": order.PaymentStatus,
	})

	// Auto-create shipment if requested (best effort — never fails order creation)
	if req.AutoCreateShipment && req.ShipmentProvider != nil && *req.ShipmentProvider != "" && s.shipmentService != nil {
		go func() {
			shipReq := model.CreateShipmentRequest{
				OrderID:  order.ID,
				Provider: *req.ShipmentProvider,
			}
			// Include pickup_point_id as carrier_data.target_point if present
			if req.PickupPointID != nil && *req.PickupPointID != "" {
				cd, _ := json.Marshal(map[string]string{"target_point": *req.PickupPointID})
				shipReq.CarrierData = cd
			}
			if _, err := s.shipmentService.Create(context.Background(), tenantID, shipReq, actorID, ip); err != nil {
				slog.Error("auto-create shipment failed", "order_id", order.ID, "provider", *req.ShipmentProvider, "error", err)
			}
		}()
	}

	return order, nil
}

func (s *OrderService) Update(ctx context.Context, tenantID, orderID uuid.UUID, req model.UpdateOrderRequest, actorID uuid.UUID, ip string) (*model.Order, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
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
		FireAutomationEvent(s.automationService, tenantID, "order", "order.updated", order.ID, map[string]any{
			"status": order.Status, "source": order.Source,
			"customer_name": order.CustomerName, "total_amount": order.TotalAmount,
			"currency": order.Currency, "payment_status": order.PaymentStatus,
		})
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
		return nil, NewValidationError(err)
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
		if s.invoiceService != nil {
			go s.invoiceService.HandleOrderStatusChange(context.Background(), tenantID, order)
		}
		if s.smsService != nil {
			go s.smsService.SendOrderStatusSMS(context.Background(), tenantID, order, oldStatus, req.Status)
		}
		FireAutomationEvent(s.automationService, tenantID, "order", "order.status_changed", order.ID, map[string]any{
			"status": order.Status, "old_status": oldStatus, "new_status": req.Status,
			"source": order.Source, "customer_name": order.CustomerName,
			"total_amount": order.TotalAmount, "currency": order.Currency,
			"payment_status": order.PaymentStatus,
		})
	}
	return order, err
}

func (s *OrderService) BulkTransitionStatus(ctx context.Context, tenantID uuid.UUID, req model.BulkStatusTransitionRequest, actorID uuid.UUID, ip string) (*model.BulkStatusTransitionResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	resp := &model.BulkStatusTransitionResponse{
		Results: make([]model.BulkStatusResult, 0, len(req.OrderIDs)),
	}

	// Collect notifications to dispatch after the transaction commits
	type emailNotification struct {
		order     *model.Order
		oldStatus string
		newStatus string
	}
	type webhookNotification struct {
		orderID   uuid.UUID
		oldStatus string
		newStatus string
	}
	var pendingEmails []emailNotification
	var pendingWebhooks []webhookNotification

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

			if err := s.auditRepo.Log(ctx, tx, model.AuditEntry{
				TenantID:   tenantID,
				UserID:     actorID,
				Action:     "order.status_changed",
				EntityType: "order",
				EntityID:   orderID,
				Changes:    map[string]string{"from": existing.Status, "to": req.Status},
				IPAddress:  ip,
			}); err != nil {
				slog.Error("bulk status transition: failed to log audit", "order_id", orderID, "error", err)
			}

			updated, err := s.orderRepo.FindByID(ctx, tx, orderID)
			if err == nil && updated != nil {
				pendingEmails = append(pendingEmails, emailNotification{
					order: updated, oldStatus: oldStatus, newStatus: req.Status,
				})
			}

			result.Success = true
			resp.Results = append(resp.Results, result)
			resp.Succeeded++

			pendingWebhooks = append(pendingWebhooks, webhookNotification{
				orderID: orderID, oldStatus: oldStatus, newStatus: req.Status,
			})
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Dispatch notifications outside the transaction
	for _, n := range pendingEmails {
		n := n
		go s.emailService.SendOrderStatusEmail(context.Background(), tenantID, n.order, n.oldStatus, n.newStatus)
		if s.smsService != nil {
			go s.smsService.SendOrderStatusSMS(context.Background(), tenantID, n.order, n.oldStatus, n.newStatus)
		}
	}
	for _, n := range pendingWebhooks {
		n := n
		go s.webhookDispatch.Dispatch(context.Background(), tenantID, "order.status_changed", map[string]any{"order_id": n.orderID.String(), "from": n.oldStatus, "to": n.newStatus})
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
