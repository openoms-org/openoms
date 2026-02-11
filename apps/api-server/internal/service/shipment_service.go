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
	ErrShipmentNotFound         = errors.New("shipment not found")
	ErrOrderNotFoundForShipment = errors.New("order not found for shipment")
)

type ShipmentService struct {
	shipmentRepo    repository.ShipmentRepo
	orderRepo       repository.OrderRepo
	auditRepo       repository.AuditRepo
	pool            *pgxpool.Pool
	webhookDispatch *WebhookDispatchService
}

func NewShipmentService(
	shipmentRepo repository.ShipmentRepo,
	orderRepo repository.OrderRepo,
	auditRepo repository.AuditRepo,
	pool *pgxpool.Pool,
	webhookDispatch *WebhookDispatchService,
) *ShipmentService {
	return &ShipmentService{
		shipmentRepo:    shipmentRepo,
		orderRepo:       orderRepo,
		auditRepo:       auditRepo,
		pool:            pool,
		webhookDispatch: webhookDispatch,
	}
}

func (s *ShipmentService) List(ctx context.Context, tenantID uuid.UUID, filter model.ShipmentListFilter) (model.ListResponse[model.Shipment], error) {
	var resp model.ListResponse[model.Shipment]
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		shipments, total, err := s.shipmentRepo.List(ctx, tx, filter)
		if err != nil {
			return err
		}
		if shipments == nil {
			shipments = []model.Shipment{}
		}
		resp = model.ListResponse[model.Shipment]{
			Items:  shipments,
			Total:  total,
			Limit:  filter.Limit,
			Offset: filter.Offset,
		}
		return nil
	})
	return resp, err
}

func (s *ShipmentService) Get(ctx context.Context, tenantID, shipmentID uuid.UUID) (*model.Shipment, error) {
	var shipment *model.Shipment
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var err error
		shipment, err = s.shipmentRepo.FindByID(ctx, tx, shipmentID)
		return err
	})
	if err != nil {
		return nil, err
	}
	if shipment == nil {
		return nil, ErrShipmentNotFound
	}
	return shipment, nil
}

func (s *ShipmentService) Create(ctx context.Context, tenantID uuid.UUID, req model.CreateShipmentRequest, actorID uuid.UUID, ip string) (*model.Shipment, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	carrierData := req.CarrierData
	if carrierData == nil {
		carrierData = json.RawMessage("{}")
	}

	shipment := &model.Shipment{
		ID:             uuid.New(),
		TenantID:       tenantID,
		OrderID:        req.OrderID,
		Provider:       req.Provider,
		IntegrationID:  req.IntegrationID,
		TrackingNumber: req.TrackingNumber,
		Status:         "created",
		LabelURL:       req.LabelURL,
		CarrierData:    carrierData,
	}

	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		order, err := s.orderRepo.FindByID(ctx, tx, req.OrderID)
		if err != nil {
			return err
		}
		if order == nil {
			return ErrOrderNotFoundForShipment
		}

		if err := s.shipmentRepo.Create(ctx, tx, shipment); err != nil {
			return err
		}
		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "shipment.created",
			EntityType: "shipment",
			EntityID:   shipment.ID,
			Changes:    map[string]string{"order_id": req.OrderID.String(), "provider": req.Provider},
			IPAddress:  ip,
		})
	})
	if err != nil {
		return nil, err
	}
	go s.webhookDispatch.Dispatch(context.Background(), tenantID, "shipment.created", shipment)
	return shipment, nil
}

func (s *ShipmentService) Update(ctx context.Context, tenantID, shipmentID uuid.UUID, req model.UpdateShipmentRequest, actorID uuid.UUID, ip string) (*model.Shipment, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	var shipment *model.Shipment
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		existing, err := s.shipmentRepo.FindByID(ctx, tx, shipmentID)
		if err != nil {
			return err
		}
		if existing == nil {
			return ErrShipmentNotFound
		}

		if err := s.shipmentRepo.Update(ctx, tx, shipmentID, req); err != nil {
			return err
		}

		shipment, err = s.shipmentRepo.FindByID(ctx, tx, shipmentID)
		if err != nil {
			return err
		}

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "shipment.updated",
			EntityType: "shipment",
			EntityID:   shipmentID,
			IPAddress:  ip,
		})
	})
	if err == nil && shipment != nil {
		go s.webhookDispatch.Dispatch(context.Background(), tenantID, "shipment.updated", shipment)
	}
	return shipment, err
}

func (s *ShipmentService) Delete(ctx context.Context, tenantID, shipmentID, actorID uuid.UUID, ip string) error {
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		shipment, err := s.shipmentRepo.FindByID(ctx, tx, shipmentID)
		if err != nil {
			return err
		}
		if shipment == nil {
			return ErrShipmentNotFound
		}

		if err := s.shipmentRepo.Delete(ctx, tx, shipmentID); err != nil {
			return err
		}

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "shipment.deleted",
			EntityType: "shipment",
			EntityID:   shipmentID,
			Changes:    map[string]string{"order_id": shipment.OrderID.String()},
			IPAddress:  ip,
		})
	})
	if err == nil {
		go s.webhookDispatch.Dispatch(context.Background(), tenantID, "shipment.deleted", map[string]any{"shipment_id": shipmentID.String()})
	}
	return err
}

func (s *ShipmentService) TransitionStatus(ctx context.Context, tenantID, shipmentID uuid.UUID, req model.ShipmentStatusTransitionRequest, actorID uuid.UUID, ip string) (*model.Shipment, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	var shipment *model.Shipment
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		existing, err := s.shipmentRepo.FindByID(ctx, tx, shipmentID)
		if err != nil {
			return err
		}
		if existing == nil {
			return ErrShipmentNotFound
		}

		currentStatus, err := engine.ParseShipmentStatus(existing.Status)
		if err != nil {
			return err
		}

		targetStatus, err := engine.ParseShipmentStatus(req.Status)
		if err != nil {
			return err
		}

		if _, err := engine.TransitionShipment(currentStatus, targetStatus, time.Now()); err != nil {
			return err
		}

		if err := s.shipmentRepo.UpdateStatus(ctx, tx, shipmentID, req.Status); err != nil {
			return err
		}

		shipment, err = s.shipmentRepo.FindByID(ctx, tx, shipmentID)
		if err != nil {
			return err
		}

		if err := s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "shipment.status_changed",
			EntityType: "shipment",
			EntityID:   shipmentID,
			Changes:    map[string]string{"from": existing.Status, "to": req.Status},
			IPAddress:  ip,
		}); err != nil {
			return err
		}

		// Order-shipment status sync
		if req.Status == "delivered" {
			if err := s.orderRepo.UpdateStatus(ctx, tx, existing.OrderID, "delivered", nil, func() *time.Time { t := time.Now(); return &t }()); err != nil {
				return fmt.Errorf("sync order status to delivered: %w", err)
			}
		} else if req.Status == "picked_up" || req.Status == "in_transit" {
			order, err := s.orderRepo.FindByID(ctx, tx, existing.OrderID)
			if err == nil && order != nil && order.Status != "shipped" && order.Status != "delivered" {
				now := time.Now()
				if err := s.orderRepo.UpdateStatus(ctx, tx, existing.OrderID, "shipped", &now, nil); err != nil {
					return fmt.Errorf("sync order status to shipped: %w", err)
				}
			}
		}

		return nil
	})
	if err == nil && shipment != nil {
		go s.webhookDispatch.Dispatch(context.Background(), tenantID, "shipment.status_changed", shipment)
	}
	return shipment, err
}
