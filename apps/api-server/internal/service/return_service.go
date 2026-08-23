package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
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

// ReturnService handles business logic for order returns.
type ReturnService struct {
	returnRepo        repository.ReturnRepo
	orderRepo         repository.OrderRepo
	auditRepo         repository.AuditRepo
	pool              *pgxpool.Pool
	webhookDispatch   *WebhookDispatchService
	automationService *AutomationService
}

// SetAutomationService sets the automation service for rule processing.
func (s *ReturnService) SetAutomationService(automationSvc *AutomationService) {
	s.automationService = automationSvc
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
		OrderID:      req.OrderID,
		Status:       "requested",
		Reason:       req.Reason,
		Items:        items,
		RefundAmount: req.RefundAmount,
		Notes:        req.Notes,
	}

	if err := s.create(ctx, tenantID, ret, actorID, ip, map[string]string{
		"order_id": req.OrderID.String(), "reason": req.Reason,
	}); err != nil {
		return nil, err
	}
	return ret, nil
}

// CreatePublic inserts a return submitted by a customer through the public
// self-service form, and returns it with the freshly generated tracking token.
//
// The caller (the public handler) owns proving the submitter owns the order — it
// resolves the tenant from the order and matches the customer email before calling
// here, because that lookup has to bypass RLS. Everything after that ownership check
// is the same work as an internally created return: the row, the audit entry, the
// webhook and the automation event. Routing it through the service is the point: the
// public path used to insert the row on its own, so a customer-submitted return
// triggered no rule and left no trail.
func (s *ReturnService) CreatePublic(ctx context.Context, tenantID, orderID uuid.UUID, req model.PublicReturnRequest) (*model.Return, error) {
	token, err := generateReturnToken()
	if err != nil {
		return nil, err
	}

	items := req.Items
	if items == nil {
		items = json.RawMessage("[]")
	}
	var notes *string
	if req.Notes != "" {
		n := req.Notes
		notes = &n
	}
	email := req.Email

	ret := &model.Return{
		ID:      uuid.New(),
		OrderID: orderID,
		Status:  "requested",
		Reason:  req.Reason,
		Items:   items,
		// A customer form cannot set the refund amount — the tenant decides it when
		// approving the return.
		RefundAmount:  0,
		Notes:         notes,
		ReturnToken:   &token,
		CustomerEmail: &email,
		CustomerNotes: notes,
	}

	if err := s.create(ctx, tenantID, ret, uuid.Nil, "", map[string]string{
		"order_id": orderID.String(), "reason": req.Reason, "source": "public_form",
	}); err != nil {
		return nil, err
	}
	return ret, nil
}

// create runs the shared return-creation path: insert plus audit inside one
// tenant-scoped transaction, then the post-commit webhook and automation event.
func (s *ReturnService) create(ctx context.Context, tenantID uuid.UUID, ret *model.Return, actorID uuid.UUID, ip string, changes map[string]string) error {
	ret.TenantID = tenantID

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
			Changes:    changes,
			IPAddress:  ip,
		})
	})
	if err != nil {
		return err
	}

	DispatchWebhookAsync(s.webhookDispatch, tenantID, "return.created", ret)
	FireAutomationEvent(s.automationService, tenantID, "return", "return.created", ret.ID, map[string]any{
		"status": ret.Status, "reason": ret.Reason, "order_id": ret.OrderID.String(),
		"refund_amount": ret.RefundAmount,
	})
	return nil
}

// generateReturnToken produces the public tracking token for a return. It is the
// return's only credential, so it comes from crypto/rand.
func generateReturnToken() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate return token: %w", err)
	}
	return hex.EncodeToString(buf), nil
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

		// Receiving the parcel is the moment the goods are physically back in the
		// warehouse, so it is the moment stock goes up. "refunded" is the money event
		// and moves nothing; the return graph has no path back into "received"
		// (requested -> approved -> received -> refunded), so this credit cannot be
		// applied twice.
		if req.Status == "received" {
			if err := s.restockReceivedItems(ctx, tx, existing); err != nil {
				return err
			}
		}

		// Auto-update order payment_status when return is refunded
		if req.Status == "refunded" {
			refunded := "refunded"
			updateReq := model.UpdateOrderRequest{PaymentStatus: &refunded}
			if err := s.orderRepo.Update(ctx, tx, existing.OrderID, updateReq); err != nil {
				return fmt.Errorf("sync order payment status to refunded: %w", err)
			}
		}

		return nil
	})
	if err == nil && ret != nil {
		DispatchWebhookAsync(s.webhookDispatch, tenantID, "return.status_changed", map[string]any{"return_id": returnID.String(), "from": oldStatus, "to": req.Status})
		FireAutomationEvent(s.automationService, tenantID, "return", "return.status_changed", ret.ID, map[string]any{
			"status": ret.Status, "old_status": oldStatus, "new_status": req.Status,
			"order_id": ret.OrderID.String(), "refund_amount": ret.RefundAmount,
		})
	}
	return ret, err
}

// restockReceivedItems credits the returned goods back to warehouse stock, inside the
// caller's transaction so the quantity and the "received" status commit together.
//
// Each returned line is credited to one row, chosen the same way the order path debits:
// the line's own variant row first, then the product-level row, preferring the tenant's
// default warehouse. A line whose product has no warehouse rows at all is skipped —
// nothing tracks it, so there is nothing to credit, and a service or digital item must
// not block the return.
func (s *ReturnService) restockReceivedItems(ctx context.Context, tx pgx.Tx, ret *model.Return) error {
	for line, qty := range extractStockLines(ret.Items) {
		var variantID *uuid.UUID
		if line.VariantID != uuid.Nil {
			v := line.VariantID
			variantID = &v
		}

		var stockID uuid.UUID
		err := tx.QueryRow(ctx,
			`SELECT ws.id
			   FROM warehouse_stock ws
			   JOIN warehouses w ON w.id = ws.warehouse_id
			  WHERE ws.product_id = $1
			  ORDER BY (ws.variant_id IS NOT DISTINCT FROM $2::uuid) DESC,
			           (ws.variant_id IS NULL) DESC,
			           w.is_default DESC,
			           ws.updated_at DESC
			  LIMIT 1`,
			line.ProductID, variantID).Scan(&stockID)
		if errors.Is(err, pgx.ErrNoRows) {
			slog.Info("return restock skipped: product has no warehouse stock rows",
				"return_id", ret.ID, "product_id", line.ProductID)
			continue
		}
		if err != nil {
			return fmt.Errorf("find restock target for product %s: %w", line.ProductID, err)
		}

		if _, err := tx.Exec(ctx,
			`UPDATE warehouse_stock SET quantity = quantity + $1, updated_at = NOW() WHERE id = $2`,
			qty, stockID); err != nil {
			return fmt.Errorf("restock product %s: %w", line.ProductID, err)
		}
	}
	return nil
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
