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
		if err != nil {
			return err
		}
		return s.populateAvailableStock(ctx, tx, products)
	})
	return products, total, err
}

// populateAvailableStock sets each product's AvailableStock from the canonical store
// (warehouse_stock quantity - reserved, with a products.stock_quantity fallback for products
// with no warehouse rows) for API responses. The raw stock_quantity column stays as-is.
func (s *ProductService) populateAvailableStock(ctx context.Context, tx pgx.Tx, products []model.Product) error {
	if len(products) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, len(products))
	for i := range products {
		ids[i] = products[i].ID
	}
	avail, err := s.productRepo.AvailableStockBatch(ctx, tx, ids)
	if err != nil {
		return err
	}
	for i := range products {
		products[i].AvailableStock = avail[products[i].ID]
	}
	return nil
}

// Get returns a single product by ID.
func (s *ProductService) Get(ctx context.Context, tenantID, productID uuid.UUID) (*model.Product, error) {
	var product *model.Product
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var err error
		product, err = s.productRepo.FindByID(ctx, tx, productID)
		if err != nil {
			return err
		}
		if product != nil {
			var avail map[uuid.UUID]int
			avail, err = s.productRepo.AvailableStockBatch(ctx, tx, []uuid.UUID{productID})
			if err != nil {
				return err
			}
			product.AvailableStock = avail[productID]
		}
		return nil
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

	var available int
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		if err := s.productRepo.Create(ctx, tx, product); err != nil {
			return err
		}
		var err error
		if available, err = s.availableStock(ctx, tx, product.ID); err != nil {
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
	product.AvailableStock = available
	DispatchWebhookAsync(s.webhookDispatch, tenantID, "product.created", product)
	// available_stock is the canonical figure a rule should condition on;
	// stock_quantity reports the legacy column as-is and is stale for any product the
	// warehouse tracks.
	FireAutomationEvent(s.automationService, tenantID, "product", "product.created", product.ID, map[string]any{
		"name": product.Name, "price": product.Price, "stock_quantity": product.StockQuantity,
		"available_stock": available, "source": product.Source,
	})
	// Trigger marketplace stock sync if the product is actually sellable. The
	// quantity handed to the stock owner is the canonical available stock, never the
	// legacy stock_quantity column — that column is not decremented on shipment, so
	// using it here would relist products that have nothing left in the warehouse.
	if s.stockSyncService != nil && available > 0 {
		asyncutil.SafeGo(func() {
			s.stockSyncService.OnStockChange(context.Background(), tenantID, product.ID, "product_created", 0, available)
		})
	}
	return product, nil
}

// availableStock resolves one product's canonical available stock (warehouse_stock
// quantity - reserved, with the legacy products.stock_quantity standing in only for
// products that have no warehouse rows at all).
func (s *ProductService) availableStock(ctx context.Context, tx pgx.Tx, productID uuid.UUID) (int, error) {
	avail, err := s.productRepo.AvailableStockBatch(ctx, tx, []uuid.UUID{productID})
	if err != nil {
		return 0, err
	}
	return avail[productID], nil
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
	var availableBefore, availableAfter int
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var err error
		product, err = s.productRepo.FindByID(ctx, tx, productID)
		if err != nil {
			return err
		}
		if product == nil {
			return ErrProductNotFound
		}
		if availableBefore, err = s.availableStock(ctx, tx, productID); err != nil {
			return err
		}

		if err := s.productRepo.Update(ctx, tx, productID, req); err != nil {
			return err
		}

		// Re-fetch to get updated fields
		product, err = s.productRepo.FindByID(ctx, tx, productID)
		if err != nil {
			return err
		}
		if availableAfter, err = s.availableStock(ctx, tx, productID); err != nil {
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
		product.AvailableStock = availableAfter
		DispatchWebhookAsync(s.webhookDispatch, tenantID, "product.updated", product)
		FireAutomationEvent(s.automationService, tenantID, "product", "product.updated", product.ID, map[string]any{
			"name": product.Name, "price": product.Price, "stock_quantity": product.StockQuantity,
			"available_stock": availableAfter, "source": product.Source,
		})
		// Trigger marketplace stock sync when editing the stock field actually moved
		// the canonical available stock. Comparing the legacy stock_quantity column
		// instead (as this used to) both misreported the quantities the relist /
		// deactivate decisions read and fired for warehouse-tracked products, whose
		// availability that column does not affect at all.
		if s.stockSyncService != nil && req.StockQuantity != nil && availableAfter != availableBefore {
			asyncutil.SafeGo(func() {
				s.stockSyncService.OnStockChange(context.Background(), tenantID, productID, "manual_update", availableBefore, availableAfter)
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
		DispatchWebhookAsync(s.webhookDispatch, tenantID, "product.deleted", map[string]any{"product_id": productID.String()})
	}
	return err
}
