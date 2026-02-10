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
	ErrReturnNotFound          = errors.New("return not found")
	ErrInvalidReturnTransition = errors.New("invalid return status transition")
)

type ReturnService struct {
	returnRepo      *repository.ReturnRepository
	orderRepo       *repository.OrderRepository
	auditRepo       *repository.AuditRepository
	pool            *pgxpool.Pool
	webhookDispatch *WebhookDispatchService
}

func NewReturnService(
	returnRepo *repository.ReturnRepository,
	orderRepo *repository.OrderRepository,
	auditRepo *repository.AuditRepository,
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

func (s *ReturnService) List(ctx context.Context, tenantID uuid.UUID, filter model.ReturnListFilter) (model.ListResponse[model.Return], error) {
	var resp model.ListResponse[model.Return]
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		returns, total, err := s.returnRepo.List(ctx, tx, filter)
		if err != nil {
			return err
		}
		if returns == nil {
			returns = []model.Return{}
		}
		resp = model.ListResponse[model.Return]{
			Items:  returns,
			Total:  total,
			Limit:  filter.Limit,
			Offset: filter.Offset,
		}
		return nil
	})
	return resp, err
}

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

func (s *ReturnService) Create(ctx context.Context, tenantID uuid.UUID, req model.CreateReturnRequest, actorID uuid.UUID, ip string) (*model.Return, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation: %w", err)
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

	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		order, err := s.orderRepo.FindByID(ctx, tx, req.OrderID)
		if err != nil {
			return err
		}
		if order == nil {
			return fmt.Errorf("validation: %w", errors.New("order not found"))
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
			Changes:    map[string]string{"order_id": req.OrderID.String(), "reason": req.Reason},
			IPAddress:  ip,
		})
	})
	if err != nil {
		return nil, err
	}
	go s.webhookDispatch.Dispatch(context.Background(), tenantID, "return.created", ret)
	return ret, nil
}

func (s *ReturnService) Update(ctx context.Context, tenantID, returnID uuid.UUID, req model.UpdateReturnRequest, actorID uuid.UUID, ip string) (*model.Return, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation: %w", err)
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
		go s.webhookDispatch.Dispatch(context.Background(), tenantID, "return.updated", ret)
	}
	return ret, err
}

func (s *ReturnService) TransitionStatus(ctx context.Context, tenantID, returnID uuid.UUID, req model.ReturnStatusRequest, actorID uuid.UUID, ip string) (*model.Return, error) {
	if req.Status == "" {
		return nil, fmt.Errorf("validation: %w", errors.New("status is required"))
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

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "return.status_changed",
			EntityType: "return",
			EntityID:   returnID,
			Changes:    map[string]string{"from": existing.Status, "to": req.Status},
			IPAddress:  ip,
		})
	})
	if err == nil && ret != nil {
		go s.webhookDispatch.Dispatch(context.Background(), tenantID, "return.status_changed", map[string]any{"return_id": returnID.String(), "from": oldStatus, "to": req.Status})
	}
	return ret, err
}

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
		go s.webhookDispatch.Dispatch(context.Background(), tenantID, "return.deleted", map[string]any{"return_id": returnID.String()})
	}
	return err
}
