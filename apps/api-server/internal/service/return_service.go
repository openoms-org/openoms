package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

var (
	// ErrReturnNotFound is returned when a return does not exist.
	ErrReturnNotFound = errors.New("return not found")
	// ErrInvalidReturnTransition is returned for an invalid return status transition.
	ErrInvalidReturnTransition = errors.New("invalid return status transition")
)

// StockRestocker puts returned line items back into warehouse stock.
// Implemented by *OrderService.
type StockRestocker interface {
	RestockItems(ctx context.Context, tenantID uuid.UUID, items json.RawMessage) error
}

// ReturnService handles business logic for order returns.
type ReturnService struct {
	returnRepo        repository.ReturnRepo
	orderRepo         repository.OrderRepo
	auditRepo         repository.AuditRepo
	tenantRepo        repository.TenantRepo
	pool              *pgxpool.Pool
	webhookDispatch   *WebhookDispatchService
	automationService *AutomationService
	stockRestocker    StockRestocker
}

// SetAutomationService sets the automation service for rule processing.
func (s *ReturnService) SetAutomationService(automationSvc *AutomationService) {
	s.automationService = automationSvc
}

// SetRestockPolicy wires what a return needs to put goods back into stock: the tenant
// repo the restock policy is read from and the restocker that performs it. Called after
// construction to avoid a circular dependency with OrderService. Leaving either unwired
// disables restocking entirely.
func (s *ReturnService) SetRestockPolicy(tenantRepo repository.TenantRepo, restocker StockRestocker) {
	s.tenantRepo = tenantRepo
	s.stockRestocker = restocker
}

// NewReturnService creates a new ReturnService.
func NewReturnService(
	returnRepo repository.ReturnRepo,
	orderRepo repository.OrderRepo,
	auditRepo repository.AuditRepo,
	pool *pgxpool.Pool,
	webhookDispatch *WebhookDispatchService,
) *ReturnService {
	return &ReturnService{
		returnRepo:      returnRepo,
		orderRepo:       orderRepo,
		auditRepo:       auditRepo,
		pool:            pool,
		webhookDispatch: webhookDispatch,
	}
}

// List returns a paginated list of returns for a tenant.
func (s *ReturnService) List(ctx context.Context, tenantID uuid.UUID, filter model.ReturnListFilter) (model.ListResponse[model.Return], error) {
	var resp model.ListResponse[model.Return]
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		returns, total, err := s.returnRepo.List(ctx, tx, filter)
		if err != nil {
			return err
		}
		resp = model.NewListResponse(returns, total, filter.Limit, filter.Offset)
		return nil
	})
	return resp, err
}

// Get returns a single return by ID.
func (s *ReturnService) Get(ctx context.Context, tenantID, returnID uuid.UUID) (*model.Return, error) {
	var ret *model.Return
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var err error
		ret, err = s.returnRepo.FindByID(ctx, tx, returnID)
		return err
	})
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return nil, ErrReturnNotFound
	}
	return ret, nil
}

// Create inserts a new return.
func (s *ReturnService) Create(ctx context.Context, tenantID uuid.UUID, req model.CreateReturnRequest, actorID uuid.UUID, ip string) (*model.Return, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	// Sanitize user-facing text fields to prevent stored XSS
	req.Reason = model.StripHTMLTags(req.Reason)
	if req.Notes != nil {
		sanitized := model.StripHTMLTags(*req.Notes)
		req.Notes = &sanitized
	}

	items := req.Items
	if items == nil {
		items = json.RawMessage("[]")
	}

	ret := &model.Return{
		ID:           uuid.New(),
		TenantID:     tenantID,
		OrderID:      req.OrderID,
		Status:       "requested",
		Reason:       req.Reason,
		Items:        items,
		RefundAmount: req.RefundAmount,
		Notes:        req.Notes,
	}
	return s.persistNewReturn(ctx, tenantID, ret, actorID, ip, map[string]string{
		"order_id": req.OrderID.String(), "reason": req.Reason,
	})
}

// PublicReturnInput is a return submitted through the unauthenticated self-service
// form. The tenant and customer email have already been resolved and verified against
// the order by the caller; ReturnToken is the customer's handle on the request.
type PublicReturnInput struct {
	OrderID       uuid.UUID
	Reason        string
	Items         json.RawMessage
	Notes         *string
	CustomerEmail string
	ReturnToken   string
	IP            string
}

// CreatePublic records a return submitted through the public self-service form. It goes
// through the same path as an operator-created return, so a customer-submitted return
// gets its audit entry, return.created webhook and automation event — the public
// endpoint used to insert the row itself and produce none of them, leaving returns that
// no rule or integration ever saw.
func (s *ReturnService) CreatePublic(ctx context.Context, tenantID uuid.UUID, in PublicReturnInput) (*model.Return, error) {
	items := in.Items
	if items == nil {
		items = json.RawMessage("[]")
	}
	reason := model.StripHTMLTags(in.Reason)
	notes := in.Notes
	if notes != nil {
		sanitized := model.StripHTMLTags(*notes)
		notes = &sanitized
	}

	ret := &model.Return{
		ID:            uuid.New(),
		TenantID:      tenantID,
		OrderID:       in.OrderID,
		Status:        "requested",
		Reason:        reason,
		Items:         items,
		RefundAmount:  0,
		Notes:         notes,
		ReturnToken:   &in.ReturnToken,
		CustomerEmail: &in.CustomerEmail,
		CustomerNotes: notes,
	}
	// The actor is uuid.Nil: the submitter is the customer, not a tenant user.
	return s.persistNewReturn(ctx, tenantID, ret, uuid.Nil, in.IP, map[string]string{
		"order_id": in.OrderID.String(), "reason": reason, "source": "public_form",
	})
}

// persistNewReturn checks the order exists, inserts the return and its audit entry in
// one tenant transaction, then dispatches the return.created webhook and automation
// event. Single owner of "a return came into existence" for both the operator and the
// public self-service paths.
func (s *ReturnService) persistNewReturn(ctx context.Context, tenantID uuid.UUID, ret *model.Return, actorID uuid.UUID, ip string, auditChanges map[string]string) (*model.Return, error) {
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		order, err := s.orderRepo.FindByID(ctx, tx, ret.OrderID)
		if err != nil {
			return err
		}
		if order == nil {
			return NewValidationError(errors.New("order not found"))
		}

		if err := s.returnRepo.Create(ctx, tx, ret); err != nil {
			return err
		}
		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "return.created",
			EntityType: "return",
			EntityID:   ret.ID,
			Changes:    auditChanges,
			IPAddress:  ip,
		})
	})
	if err != nil {
		return nil, err
	}
	DispatchWebhookAsync(s.webhookDispatch, tenantID, "return.created", ret)
	FireAutomationEvent(s.automationService, tenantID, "return", "return.created", ret.ID, map[string]any{
		"status": ret.Status, "reason": ret.Reason, "order_id": ret.OrderID.String(),
		"refund_amount": ret.RefundAmount,
	})
	return ret, nil
}

// Update modifies an existing return.
func (s *ReturnService) Update(ctx context.Context, tenantID, returnID uuid.UUID, req model.UpdateReturnRequest, actorID uuid.UUID, ip string) (*model.Return, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	var ret *model.Return
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		existing, err := s.returnRepo.FindByID(ctx, tx, returnID)
		if err != nil {
			return err
		}
		if existing == nil {
			return ErrReturnNotFound
		}

		if err := s.returnRepo.Update(ctx, tx, returnID, req); err != nil {
			return err
		}

		ret, err = s.returnRepo.FindByID(ctx, tx, returnID)
		if err != nil {
			return err
		}

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "return.updated",
			EntityType: "return",
			EntityID:   returnID,
			IPAddress:  ip,
		})
	})
	if err == nil && ret != nil {
		DispatchWebhookAsync(s.webhookDispatch, tenantID, "return.updated", ret)
	}
	return ret, err
}

// TransitionStatus moves a return to a new status.
func (s *ReturnService) TransitionStatus(ctx context.Context, tenantID, returnID uuid.UUID, req model.ReturnStatusRequest, actorID uuid.UUID, ip string) (*model.Return, error) {
	if req.Status == "" {
		return nil, NewValidationError(errors.New("status is required"))
	}

	var ret *model.Return
	var oldStatus string
	var restock bool
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		existing, err := s.returnRepo.FindByID(ctx, tx, returnID)
		if err != nil {
			return err
		}
		if existing == nil {
			return ErrReturnNotFound
		}
		oldStatus = existing.Status

		if !model.IsValidReturnTransition(existing.Status, req.Status) {
			return fmt.Errorf("%w: %s -> %s", ErrInvalidReturnTransition, existing.Status, req.Status)
		}

		if err := s.returnRepo.UpdateStatus(ctx, tx, returnID, req.Status); err != nil {
			return err
		}

		ret, err = s.returnRepo.FindByID(ctx, tx, returnID)
		if err != nil {
			return err
		}

		if err := s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "return.status_changed",
			EntityType: "return",
			EntityID:   returnID,
			Changes:    map[string]string{"from": existing.Status, "to": req.Status},
			IPAddress:  ip,
		}); err != nil {
			return err
		}

		// Auto-update order payment_status when return is refunded
		if req.Status == "refunded" {
			refunded := "refunded"
			updateReq := model.UpdateOrderRequest{PaymentStatus: &refunded}
			if err := s.orderRepo.Update(ctx, tx, existing.OrderID, updateReq); err != nil {
				return fmt.Errorf("sync order payment status to refunded: %w", err)
			}
		}

		// Read the restock policy while the tenant tx is open; apply it after commit,
		// so a warehouse write can never roll the status change back.
		restockStatus, err := s.restockStatus(ctx, tx, tenantID)
		if err != nil {
			return err
		}
		restock = restockStatus != "" && restockStatus == req.Status

		return nil
	})
	if err == nil && ret != nil {
		if restock {
			if rerr := s.stockRestocker.RestockItems(ctx, tenantID, ret.Items); rerr != nil {
				slog.Error("return restock failed", "error", rerr, "return_id", ret.ID, "tenant_id", tenantID)
			}
		}
		DispatchWebhookAsync(s.webhookDispatch, tenantID, "return.status_changed", map[string]any{"return_id": returnID.String(), "from": oldStatus, "to": req.Status})
		FireAutomationEvent(s.automationService, tenantID, "return", "return.status_changed", ret.ID, map[string]any{
			"status": ret.Status, "old_status": oldStatus, "new_status": req.Status,
			"order_id": ret.OrderID.String(), "refund_amount": ret.RefundAmount,
		})
	}
	return ret, err
}

// restockStatus returns the return status whose entry puts the returned items back into
// warehouse stock for this tenant, or "" when returns do not restock (the policy is off,
// or the restock dependencies are unwired). Only the return's own line items are
// restocked, so a return that lists nothing restocks nothing — the goods that came back
// are not knowable from the order alone.
func (s *ReturnService) restockStatus(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID) (string, error) {
	if s.tenantRepo == nil || s.stockRestocker == nil {
		return "", nil
	}
	settings, err := s.tenantRepo.GetSettings(ctx, tx, tenantID)
	if err != nil {
		return "", fmt.Errorf("load returns settings: %w", err)
	}
	cfg := model.ReturnSettings{}
	if len(settings) > 0 {
		var allSettings map[string]json.RawMessage
		if err := json.Unmarshal(settings, &allSettings); err != nil {
			return "", fmt.Errorf("parse tenant settings: %w", err)
		}
		if raw, ok := allSettings["returns"]; ok {
			if err := json.Unmarshal(raw, &cfg); err != nil {
				slog.Warn("failed to unmarshal returns settings, using the default restock policy",
					"error", err, "tenant_id", tenantID)
				cfg = model.ReturnSettings{}
			}
		}
	}
	return cfg.RestockStatus(), nil
}

// Delete removes a return by ID.
func (s *ReturnService) Delete(ctx context.Context, tenantID, returnID, actorID uuid.UUID, ip string) error {
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		ret, err := s.returnRepo.FindByID(ctx, tx, returnID)
		if err != nil {
			return err
		}
		if ret == nil {
			return ErrReturnNotFound
		}

		if err := s.returnRepo.Delete(ctx, tx, returnID); err != nil {
			return err
		}

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "return.deleted",
			EntityType: "return",
			EntityID:   returnID,
			Changes:    map[string]string{"order_id": ret.OrderID.String()},
			IPAddress:  ip,
		})
	})
	if err == nil {
		DispatchWebhookAsync(s.webhookDispatch, tenantID, "return.deleted", map[string]any{"return_id": returnID.String()})
	}
	return err
}
