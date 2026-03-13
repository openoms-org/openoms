package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/openoms-org/openoms/apps/api-server/internal/asyncutil"
	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

var (
	// ErrProductNotFound is returned when a product does not exist.
	ErrProductNotFound = errors.New("product not found")
	// ErrDuplicateSKU is returned when a product with the same SKU already exists.
	ErrDuplicateSKU = errors.New("product with this SKU already exists in this tenant")
)

// ProductService handles business logic for product management.
type ProductService struct {
	productRepo       repository.ProductRepo
	auditRepo         repository.AuditRepo
	pool              *pgxpool.Pool
	webhookDispatch   *WebhookDispatchService
	automationService *AutomationService
	stockSyncService  *StockSyncService
}

// SetAutomationService sets the automation service for rule processing.
func (s *ProductService) SetAutomationService(automationSvc *AutomationService) {
	s.automationService = automationSvc
}

// SetStockSyncService sets the stock sync service for propagating stock changes.
func (s *ProductService) SetStockSyncService(svc *StockSyncService) {
	s.stockSyncService = svc
}

// NewProductService creates a new ProductService.
func NewProductService(
	productRepo repository.ProductRepo,
	auditRepo repository.AuditRepo,
	pool *pgxpool.Pool,
	webhookDispatch *WebhookDispatchService,
) *ProductService {
	return &ProductService{
		productRepo:     productRepo,
		auditRepo:       auditRepo,
		pool:            pool,
		webhookDispatch: webhookDispatch,
	}
}

// List returns a paginated list of products.
func (s *ProductService) List(ctx context.Context, tenantID uuid.UUID, filter model.ProductListFilter) ([]model.Product, int, error) {
	var products []model.Product
	var total int
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var err error
		products, total, err = s.productRepo.List(ctx, tx, filter)
		return err
	})
	return products, total, err
}

// Get returns a single product by ID.
func (s *ProductService) Get(ctx context.Context, tenantID, productID uuid.UUID) (*model.Product, error) {
	var product *model.Product
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var err error
		product, err = s.productRepo.FindByID(ctx, tx, productID)
		return err
	})
	if err != nil {
		return nil, err
	}
	if product == nil {
		return nil, ErrProductNotFound
	}
	return product, nil
}

// Create inserts a new product.
func (s *ProductService) Create(ctx context.Context, tenantID uuid.UUID, req model.CreateProductRequest, actorID uuid.UUID, ip string) (*model.Product, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	// Sanitize user-facing text fields to prevent stored XSS
	req.Name = model.StripHTMLTags(req.Name)
	req.DescriptionShort = model.StripHTMLTags(req.DescriptionShort)
	req.DescriptionLong = model.StripHTMLTags(req.DescriptionLong)

	metadata := req.Metadata
	if metadata == nil {
		metadata = []byte("{}")
	}

	images := req.Images
	if images == nil {
		images = []byte("[]")
	}

	tags := req.Tags
	if tags == nil {
		tags = []string{}
	}

	product := &model.Product{
		ID:                 uuid.New(),
		TenantID:           tenantID,
		ExternalID:         req.ExternalID,
		Source:             req.Source,
		Name:               req.Name,
		SKU:                req.SKU,
		EAN:                req.EAN,
		Price:              req.Price,
		StockQuantity:      req.StockQty,
		Metadata:           metadata,
		Tags:               tags,
		DescriptionShort:   req.DescriptionShort,
		DescriptionLong:    req.DescriptionLong,
		Weight:             req.Weight,
		Width:              req.Width,
		Height:             req.Height,
		Depth:              req.Depth,
		Category:           req.Category,
		ImageURL:           req.ImageURL,
		Images:             images,
		IsDropship:         req.IsDropship != nil && *req.IsDropship,
		DropshipSupplierID: req.DropshipSupplierID,
	}

	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		if err := s.productRepo.Create(ctx, tx, product); err != nil {
			return err
		}
		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "product.created",
			EntityType: "product",
			EntityID:   product.ID,
			Changes:    map[string]string{"name": req.Name, "source": req.Source},
			IPAddress:  ip,
		})
	})
	if err != nil {
		if isDuplicateKeyError(err) {
			return nil, ErrDuplicateSKU
		}
		return nil, err
	}
	if s.webhookDispatch != nil {
		asyncutil.SafeGo(func() { s.webhookDispatch.Dispatch(context.Background(), tenantID, "product.created", product) })
	}
	FireAutomationEvent(s.automationService, tenantID, "product", "product.created", product.ID, map[string]any{
		"name": product.Name, "price": product.Price, "stock_quantity": product.StockQuantity,
		"source": product.Source,
	})
	// Trigger marketplace stock sync if product was created with stock
	if s.stockSyncService != nil && product.StockQuantity > 0 {
		asyncutil.SafeGo(func() {
			s.stockSyncService.OnStockChange(context.Background(), tenantID, product.ID, "product_created", 0, product.StockQuantity)
		})
	}
	return product, nil
}

// Update modifies an existing product.
func (s *ProductService) Update(ctx context.Context, tenantID, productID uuid.UUID, req model.UpdateProductRequest, actorID uuid.UUID, ip string) (*model.Product, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	// Sanitize user-facing text fields to prevent stored XSS
	if req.Name != nil {
		sanitized := model.StripHTMLTags(*req.Name)
		req.Name = &sanitized
	}
	if req.DescriptionShort != nil {
		sanitized := model.StripHTMLTags(*req.DescriptionShort)
		req.DescriptionShort = &sanitized
	}
	if req.DescriptionLong != nil {
		sanitized := model.StripHTMLTags(*req.DescriptionLong)
		req.DescriptionLong = &sanitized
	}

	var product *model.Product
	var oldStockQty int
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var err error
		product, err = s.productRepo.FindByID(ctx, tx, productID)
		if err != nil {
			return err
		}
		if product == nil {
			return ErrProductNotFound
		}
		oldStockQty = product.StockQuantity

		if err := s.productRepo.Update(ctx, tx, productID, req); err != nil {
			return err
		}

		// Re-fetch to get updated fields
		product, err = s.productRepo.FindByID(ctx, tx, productID)
		if err != nil {
			return err
		}

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "product.updated",
			EntityType: "product",
			EntityID:   productID,
			IPAddress:  ip,
		})
	})
	if err != nil {
		if isDuplicateKeyError(err) {
			return nil, ErrDuplicateSKU
		}
		return nil, err
	}
	if product != nil {
		if s.webhookDispatch != nil {
			asyncutil.SafeGo(func() { s.webhookDispatch.Dispatch(context.Background(), tenantID, "product.updated", product) })
		}
		FireAutomationEvent(s.automationService, tenantID, "product", "product.updated", product.ID, map[string]any{
			"name": product.Name, "price": product.Price, "stock_quantity": product.StockQuantity,
			"source": product.Source,
		})
		// Trigger marketplace stock sync if stock quantity changed
		if s.stockSyncService != nil && req.StockQuantity != nil && product.StockQuantity != oldStockQty {
			asyncutil.SafeGo(func() {
				s.stockSyncService.OnStockChange(context.Background(), tenantID, productID, "manual_update", oldStockQty, product.StockQuantity)
			})
		}
	}
	return product, err
}

// Delete removes a product by ID.
func (s *ProductService) Delete(ctx context.Context, tenantID, productID uuid.UUID, actorID uuid.UUID, ip string) error {
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		product, err := s.productRepo.FindByID(ctx, tx, productID)
		if err != nil {
			return err
		}
		if product == nil {
			return ErrProductNotFound
		}

		if err := s.productRepo.Delete(ctx, tx, productID); err != nil {
			return err
		}

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "product.deleted",
			EntityType: "product",
			EntityID:   productID,
			Changes:    map[string]string{"name": product.Name},
			IPAddress:  ip,
		})
	})
	if err == nil {
		if s.webhookDispatch != nil {
			asyncutil.SafeGo(func() {
				s.webhookDispatch.Dispatch(context.Background(), tenantID, "product.deleted", map[string]any{"product_id": productID.String()})
			})
		}
	}
	return err
}
