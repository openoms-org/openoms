package service

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

var (
	ErrWarehouseNotFound = errors.New("warehouse not found")
	ErrStockEntryNotFound = errors.New("stock entry not found")
)

// WarehouseService provides business logic for warehouses and warehouse stock.
type WarehouseService struct {
	warehouseRepo      repository.WarehouseRepo
	warehouseStockRepo repository.WarehouseStockRepo
	auditRepo          repository.AuditRepo
	pool               *pgxpool.Pool
}

// NewWarehouseService creates a new WarehouseService.
func NewWarehouseService(
	warehouseRepo repository.WarehouseRepo,
	warehouseStockRepo repository.WarehouseStockRepo,
	auditRepo repository.AuditRepo,
	pool *pgxpool.Pool,
) *WarehouseService {
	return &WarehouseService{
		warehouseRepo:      warehouseRepo,
		warehouseStockRepo: warehouseStockRepo,
		auditRepo:          auditRepo,
		pool:               pool,
	}
}

// List lists all warehouses for a tenant.
func (s *WarehouseService) List(ctx context.Context, tenantID uuid.UUID, filter model.WarehouseListFilter) (model.ListResponse[model.Warehouse], error) {
	var resp model.ListResponse[model.Warehouse]
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		warehouses, total, err := s.warehouseRepo.List(ctx, tx, filter)
		if err != nil {
			return err
		}
		if warehouses == nil {
			warehouses = []model.Warehouse{}
		}
		resp = model.ListResponse[model.Warehouse]{
			Items:  warehouses,
			Total:  total,
			Limit:  filter.Limit,
			Offset: filter.Offset,
		}
		return nil
	})
	return resp, err
}

// Get retrieves a single warehouse by ID.
func (s *WarehouseService) Get(ctx context.Context, tenantID, warehouseID uuid.UUID) (*model.Warehouse, error) {
	var warehouse *model.Warehouse
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var err error
		warehouse, err = s.warehouseRepo.FindByID(ctx, tx, warehouseID)
		return err
	})
	if err != nil {
		return nil, err
	}
	if warehouse == nil {
		return nil, ErrWarehouseNotFound
	}
	return warehouse, nil
}

// Create creates a new warehouse.
func (s *WarehouseService) Create(ctx context.Context, tenantID uuid.UUID, req model.CreateWarehouseRequest, actorID uuid.UUID, ip string) (*model.Warehouse, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	// Sanitize user-facing text fields
	req.Name = model.StripHTMLTags(req.Name)

	address := req.Address
	if address == nil {
		address = json.RawMessage("{}")
	}

	isDefault := false
	if req.IsDefault != nil {
		isDefault = *req.IsDefault
	}

	active := true
	if req.Active != nil {
		active = *req.Active
	}

	warehouse := &model.Warehouse{
		ID:        uuid.New(),
		TenantID:  tenantID,
		Name:      req.Name,
		Code:      req.Code,
		Address:   address,
		IsDefault: isDefault,
		Active:    active,
	}

	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		if err := s.warehouseRepo.Create(ctx, tx, warehouse); err != nil {
			return err
		}
		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "warehouse.created",
			EntityType: "warehouse",
			EntityID:   warehouse.ID,
			Changes:    map[string]string{"name": req.Name},
			IPAddress:  ip,
		})
	})
	if err != nil {
		return nil, err
	}
	return warehouse, nil
}

// Update updates an existing warehouse.
func (s *WarehouseService) Update(ctx context.Context, tenantID, warehouseID uuid.UUID, req model.UpdateWarehouseRequest, actorID uuid.UUID, ip string) (*model.Warehouse, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	var warehouse *model.Warehouse
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		existing, err := s.warehouseRepo.FindByID(ctx, tx, warehouseID)
		if err != nil {
			return err
		}
		if existing == nil {
			return ErrWarehouseNotFound
		}

		if err := s.warehouseRepo.Update(ctx, tx, warehouseID, req); err != nil {
			return err
		}

		warehouse, err = s.warehouseRepo.FindByID(ctx, tx, warehouseID)
		if err != nil {
			return err
		}

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "warehouse.updated",
			EntityType: "warehouse",
			EntityID:   warehouseID,
			IPAddress:  ip,
		})
	})
	if err != nil {
		return nil, err
	}
	return warehouse, err
}

// Delete removes a warehouse.
func (s *WarehouseService) Delete(ctx context.Context, tenantID, warehouseID uuid.UUID, actorID uuid.UUID, ip string) error {
	return database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		warehouse, err := s.warehouseRepo.FindByID(ctx, tx, warehouseID)
		if err != nil {
			return err
		}
		if warehouse == nil {
			return ErrWarehouseNotFound
		}

		if err := s.warehouseRepo.Delete(ctx, tx, warehouseID); err != nil {
			return err
		}

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "warehouse.deleted",
			EntityType: "warehouse",
			EntityID:   warehouseID,
			Changes:    map[string]string{"name": warehouse.Name},
			IPAddress:  ip,
		})
	})
}

// ListStock lists stock entries for a warehouse.
func (s *WarehouseService) ListStock(ctx context.Context, tenantID, warehouseID uuid.UUID, filter model.WarehouseStockListFilter) (model.ListResponse[model.WarehouseStock], error) {
	var resp model.ListResponse[model.WarehouseStock]
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		// Verify warehouse exists
		wh, err := s.warehouseRepo.FindByID(ctx, tx, warehouseID)
		if err != nil {
			return err
		}
		if wh == nil {
			return ErrWarehouseNotFound
		}

		stocks, total, err := s.warehouseStockRepo.ListByWarehouse(ctx, tx, warehouseID, filter)
		if err != nil {
			return err
		}
		if stocks == nil {
			stocks = []model.WarehouseStock{}
		}
		resp = model.ListResponse[model.WarehouseStock]{
			Items:  stocks,
			Total:  total,
			Limit:  filter.Limit,
			Offset: filter.Offset,
		}
		return nil
	})
	return resp, err
}

// UpsertStock creates or updates a stock entry.
func (s *WarehouseService) UpsertStock(ctx context.Context, tenantID, warehouseID uuid.UUID, req model.UpsertWarehouseStockRequest, actorID uuid.UUID, ip string) (*model.WarehouseStock, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	stock := &model.WarehouseStock{
		ID:          uuid.New(),
		TenantID:    tenantID,
		WarehouseID: warehouseID,
		ProductID:   req.ProductID,
		VariantID:   req.VariantID,
		Quantity:    req.Quantity,
		Reserved:    req.Reserved,
		MinStock:    req.MinStock,
	}

	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		// Verify warehouse exists
		wh, err := s.warehouseRepo.FindByID(ctx, tx, warehouseID)
		if err != nil {
			return err
		}
		if wh == nil {
			return ErrWarehouseNotFound
		}

		if err := s.warehouseStockRepo.Upsert(ctx, tx, stock); err != nil {
			return err
		}

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "warehouse_stock.upserted",
			EntityType: "warehouse_stock",
			EntityID:   stock.ID,
			Changes: map[string]string{
				"warehouse_id": warehouseID.String(),
				"product_id":   req.ProductID.String(),
			},
			IPAddress: ip,
		})
	})
	if err != nil {
		return nil, err
	}
	return stock, nil
}

// ListProductStock lists stock across all warehouses for a product.
func (s *WarehouseService) ListProductStock(ctx context.Context, tenantID, productID uuid.UUID) ([]model.WarehouseStock, error) {
	var stocks []model.WarehouseStock
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var err error
		stocks, err = s.warehouseStockRepo.ListByProduct(ctx, tx, productID)
		return err
	})
	if err != nil {
		return nil, err
	}
	if stocks == nil {
		stocks = []model.WarehouseStock{}
	}
	return stocks, nil
}
