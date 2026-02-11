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

	iof "github.com/openoms-org/openoms/packages/iof-parser"
)

var (
	ErrSupplierNotFound        = errors.New("supplier not found")
	ErrSupplierProductNotFound = errors.New("supplier product not found")
	ErrNoFeedURL               = errors.New("supplier has no feed URL configured")
)

type SupplierService struct {
	supplierRepo     repository.SupplierRepo
	supplierProdRepo repository.SupplierProductRepo
	auditRepo        repository.AuditRepo
	pool             *pgxpool.Pool
	webhookDispatch  *WebhookDispatchService
	logger           *slog.Logger
}

func NewSupplierService(
	supplierRepo repository.SupplierRepo,
	supplierProdRepo repository.SupplierProductRepo,
	auditRepo repository.AuditRepo,
	pool *pgxpool.Pool,
	webhookDispatch *WebhookDispatchService,
	logger *slog.Logger,
) *SupplierService {
	return &SupplierService{
		supplierRepo:     supplierRepo,
		supplierProdRepo: supplierProdRepo,
		auditRepo:        auditRepo,
		pool:             pool,
		webhookDispatch:  webhookDispatch,
		logger:           logger,
	}
}

func (s *SupplierService) List(ctx context.Context, tenantID uuid.UUID, filter model.SupplierListFilter) (model.ListResponse[model.Supplier], error) {
	var resp model.ListResponse[model.Supplier]
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		suppliers, total, err := s.supplierRepo.List(ctx, tx, filter)
		if err != nil {
			return err
		}
		if suppliers == nil {
			suppliers = []model.Supplier{}
		}
		resp = model.ListResponse[model.Supplier]{
			Items:  suppliers,
			Total:  total,
			Limit:  filter.Limit,
			Offset: filter.Offset,
		}
		return nil
	})
	return resp, err
}

func (s *SupplierService) Get(ctx context.Context, tenantID, supplierID uuid.UUID) (*model.Supplier, error) {
	var supplier *model.Supplier
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var err error
		supplier, err = s.supplierRepo.FindByID(ctx, tx, supplierID)
		return err
	})
	if err != nil {
		return nil, err
	}
	if supplier == nil {
		return nil, ErrSupplierNotFound
	}
	return supplier, nil
}

func (s *SupplierService) Create(ctx context.Context, tenantID uuid.UUID, req model.CreateSupplierRequest, actorID uuid.UUID, ip string) (*model.Supplier, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	// Sanitize user-facing text fields to prevent stored XSS
	req.Name = model.StripHTMLTags(req.Name)

	settings := req.Settings
	if settings == nil {
		settings = json.RawMessage("{}")
	}

	supplier := &model.Supplier{
		ID:         uuid.New(),
		TenantID:   tenantID,
		Name:       req.Name,
		Code:       req.Code,
		FeedURL:    req.FeedURL,
		FeedFormat: req.FeedFormat,
		Status:     "active",
		Settings:   settings,
	}

	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		if err := s.supplierRepo.Create(ctx, tx, supplier); err != nil {
			return err
		}
		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "supplier.created",
			EntityType: "supplier",
			EntityID:   supplier.ID,
			Changes:    map[string]string{"name": req.Name},
			IPAddress:  ip,
		})
	})
	if err != nil {
		return nil, err
	}
	go s.webhookDispatch.Dispatch(context.Background(), tenantID, "supplier.created", supplier)
	return supplier, nil
}

func (s *SupplierService) Update(ctx context.Context, tenantID, supplierID uuid.UUID, req model.UpdateSupplierRequest, actorID uuid.UUID, ip string) (*model.Supplier, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	var supplier *model.Supplier
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		existing, err := s.supplierRepo.FindByID(ctx, tx, supplierID)
		if err != nil {
			return err
		}
		if existing == nil {
			return ErrSupplierNotFound
		}

		if err := s.supplierRepo.Update(ctx, tx, supplierID, req); err != nil {
			return err
		}

		supplier, err = s.supplierRepo.FindByID(ctx, tx, supplierID)
		if err != nil {
			return err
		}

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "supplier.updated",
			EntityType: "supplier",
			EntityID:   supplierID,
			IPAddress:  ip,
		})
	})
	if err != nil {
		return nil, err
	}
	if supplier != nil {
		go s.webhookDispatch.Dispatch(context.Background(), tenantID, "supplier.updated", supplier)
	}
	return supplier, err
}

func (s *SupplierService) Delete(ctx context.Context, tenantID, supplierID uuid.UUID, actorID uuid.UUID, ip string) error {
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		supplier, err := s.supplierRepo.FindByID(ctx, tx, supplierID)
		if err != nil {
			return err
		}
		if supplier == nil {
			return ErrSupplierNotFound
		}

		if err := s.supplierRepo.Delete(ctx, tx, supplierID); err != nil {
			return err
		}

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "supplier.deleted",
			EntityType: "supplier",
			EntityID:   supplierID,
			Changes:    map[string]string{"name": supplier.Name},
			IPAddress:  ip,
		})
	})
	if err == nil {
		go s.webhookDispatch.Dispatch(context.Background(), tenantID, "supplier.deleted", map[string]any{"supplier_id": supplierID.String()})
	}
	return err
}

func (s *SupplierService) ListProducts(ctx context.Context, tenantID uuid.UUID, filter model.SupplierProductListFilter) (model.ListResponse[model.SupplierProduct], error) {
	var resp model.ListResponse[model.SupplierProduct]
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		products, total, err := s.supplierProdRepo.List(ctx, tx, filter)
		if err != nil {
			return err
		}
		if products == nil {
			products = []model.SupplierProduct{}
		}
		resp = model.ListResponse[model.SupplierProduct]{
			Items:  products,
			Total:  total,
			Limit:  filter.Limit,
			Offset: filter.Offset,
		}
		return nil
	})
	return resp, err
}

func (s *SupplierService) LinkProduct(ctx context.Context, tenantID, supplierProductID, productID uuid.UUID, actorID uuid.UUID, ip string) error {
	return database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		sp, err := s.supplierProdRepo.FindByID(ctx, tx, supplierProductID)
		if err != nil {
			return err
		}
		if sp == nil {
			return ErrSupplierProductNotFound
		}

		if err := s.supplierProdRepo.LinkToProduct(ctx, tx, supplierProductID, productID); err != nil {
			return err
		}

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "supplier_product.linked",
			EntityType: "supplier_product",
			EntityID:   supplierProductID,
			Changes:    map[string]string{"product_id": productID.String()},
			IPAddress:  ip,
		})
	})
}

func (s *SupplierService) SyncFeed(ctx context.Context, tenantID, supplierID uuid.UUID) error {
	var supplier *model.Supplier
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var err error
		supplier, err = s.supplierRepo.FindByID(ctx, tx, supplierID)
		return err
	})
	if err != nil {
		return err
	}
	if supplier == nil {
		return ErrSupplierNotFound
	}
	if supplier.FeedURL == nil || *supplier.FeedURL == "" {
		return ErrNoFeedURL
	}

	// Parse the IOF feed
	products, err := iof.ParseURL(ctx, *supplier.FeedURL)
	if err != nil {
		// Record the error on the supplier
		errMsg := err.Error()
		if dbErr := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
			return s.supplierRepo.UpdateSyncStatus(ctx, tx, supplierID, time.Now(), &errMsg)
		}); dbErr != nil {
			s.logger.Error("failed to record supplier sync error", "supplier_id", supplierID, "error", dbErr)
		}
		return fmt.Errorf("parse feed: %w", err)
	}

	// Upsert products and auto-link by EAN
	syncedAt := time.Now()
	err = database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		for _, fp := range products {
			attrs, _ := json.Marshal(fp.Attributes)

			var ean, sku *string
			if fp.EAN != "" {
				ean = &fp.EAN
			}
			if fp.SKU != "" {
				sku = &fp.SKU
			}
			var price *float64
			if fp.Price > 0 {
				price = &fp.Price
			}

			sp := &model.SupplierProduct{
				ID:            uuid.New(),
				TenantID:      tenantID,
				SupplierID:    supplierID,
				ExternalID:    fp.ID,
				Name:          fp.Name,
				EAN:           ean,
				SKU:           sku,
				Price:         price,
				StockQuantity: fp.Stock,
				Metadata:      attrs,
				LastSyncedAt:  &syncedAt,
			}

			if err := s.supplierProdRepo.UpsertByExternalID(ctx, tx, sp); err != nil {
				s.logger.Error("failed to upsert supplier product",
					"supplier_id", supplierID, "external_id", fp.ID, "error", err)
				continue
			}

			// Auto-link by EAN if not already linked and EAN is available
			if sp.ProductID == nil && ean != nil {
				var productID uuid.UUID
				err := tx.QueryRow(ctx,
					"SELECT id FROM products WHERE ean = $1 LIMIT 1", *ean,
				).Scan(&productID)
				if err == nil {
					if linkErr := s.supplierProdRepo.LinkToProduct(ctx, tx, sp.ID, productID); linkErr != nil {
						s.logger.Error("failed to auto-link supplier product by EAN",
							"supplier_product_id", sp.ID, "product_id", productID, "error", linkErr)
					}
				}
			}
		}

		return s.supplierRepo.UpdateSyncStatus(ctx, tx, supplierID, syncedAt, nil)
	})

	if err != nil {
		return fmt.Errorf("sync feed: %w", err)
	}

	s.logger.Info("supplier feed synced",
		"supplier_id", supplierID, "products_count", len(products))
	return nil
}
