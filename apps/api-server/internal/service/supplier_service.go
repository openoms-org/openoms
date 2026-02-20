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
	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
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
	supplierRepo        repository.SupplierRepo
	supplierProdRepo    repository.SupplierProductRepo
	categoryMappingRepo repository.SupplierCategoryMappingRepo
	categoryRepo        repository.ProductCategoryRepo
	productRepo         repository.ProductRepo
	auditRepo           repository.AuditRepo
	pool                *pgxpool.Pool
	webhookDispatch     *WebhookDispatchService
	integrationSvc      *IntegrationService
	logger              *slog.Logger
}

func NewSupplierService(
	supplierRepo repository.SupplierRepo,
	supplierProdRepo repository.SupplierProductRepo,
	categoryMappingRepo repository.SupplierCategoryMappingRepo,
	categoryRepo repository.ProductCategoryRepo,
	productRepo repository.ProductRepo,
	auditRepo repository.AuditRepo,
	pool *pgxpool.Pool,
	webhookDispatch *WebhookDispatchService,
	integrationSvc *IntegrationService,
	logger *slog.Logger,
) *SupplierService {
	return &SupplierService{
		supplierRepo:        supplierRepo,
		supplierProdRepo:    supplierProdRepo,
		categoryMappingRepo: categoryMappingRepo,
		categoryRepo:        categoryRepo,
		productRepo:         productRepo,
		auditRepo:           auditRepo,
		pool:                pool,
		webhookDispatch:     webhookDispatch,
		integrationSvc:      integrationSvc,
		logger:              logger,
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

	syncInterval := 60
	if req.SyncIntervalMinutes != nil {
		syncInterval = *req.SyncIntervalMinutes
	}

	supplier := &model.Supplier{
		ID:                  uuid.New(),
		TenantID:            tenantID,
		Name:                req.Name,
		Code:                req.Code,
		FeedURL:             req.FeedURL,
		FeedFormat:          req.FeedFormat,
		Status:              "active",
		Settings:            settings,
		SyncIntervalMinutes: syncInterval,
		IntegrationID:       req.IntegrationID,
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
	if s.webhookDispatch != nil {
		go s.webhookDispatch.Dispatch(context.Background(), tenantID, "supplier.created", supplier)
	}
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
	if supplier != nil && s.webhookDispatch != nil {
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
	if err == nil && s.webhookDispatch != nil {
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

// ImportProducts creates OMS products from selected supplier products and links them.
func (s *SupplierService) ImportProducts(ctx context.Context, tenantID, supplierID uuid.UUID, req model.ImportSupplierProductsRequest, actorID uuid.UUID, ip string) (*model.ImportSupplierProductsResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	resp := &model.ImportSupplierProductsResponse{}

	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		// Verify supplier exists
		supplier, err := s.supplierRepo.FindByID(ctx, tx, supplierID)
		if err != nil {
			return err
		}
		if supplier == nil {
			return ErrSupplierNotFound
		}

		// Fetch all requested supplier products
		spProducts, err := s.supplierProdRepo.FindByIDs(ctx, tx, req.SupplierProductIDs)
		if err != nil {
			return err
		}

		// Build lookup map
		spMap := make(map[uuid.UUID]*model.SupplierProduct, len(spProducts))
		for i := range spProducts {
			spMap[spProducts[i].ID] = &spProducts[i]
		}

		// Load category mappings for this supplier to resolve source_category → OMS category name
		mappings, err := s.categoryMappingRepo.ListBySupplier(ctx, tx, supplierID)
		if err != nil {
			return err
		}

		// Build category ID lookup and resolve names
		categoryNameMap := make(map[string]string, len(mappings))
		for _, m := range mappings {
			if m.CategoryID != nil {
				cat, catErr := s.categoryRepo.FindByID(ctx, tx, *m.CategoryID)
				if catErr == nil && cat != nil {
					categoryNameMap[m.SourceCategory] = cat.Name
				}
			}
		}

		for _, spID := range req.SupplierProductIDs {
			sp, ok := spMap[spID]
			if !ok {
				resp.Errors = append(resp.Errors, model.ImportSupplierProductError{
					SupplierProductID: spID.String(),
					Reason:            "supplier product not found",
				})
				continue
			}

			// Skip if already linked
			if sp.ProductID != nil {
				resp.Skipped++
				continue
			}

			// Verify it belongs to this supplier
			if sp.SupplierID != supplierID {
				resp.Errors = append(resp.Errors, model.ImportSupplierProductError{
					SupplierProductID: spID.String(),
					Reason:            "supplier product does not belong to this supplier",
				})
				continue
			}

			// Resolve category from mappings
			var category *string
			if sp.SourceCategory != nil {
				if name, ok := categoryNameMap[*sp.SourceCategory]; ok {
					category = &name
				}
			}
			if category == nil {
				category = sp.SourceCategory
			}

			price := 0.0
			if sp.Price != nil {
				price = *sp.Price
			}

			metadata := sp.Metadata
			if metadata == nil {
				metadata = json.RawMessage("{}")
			}

			product := &model.Product{
				ID:            uuid.New(),
				TenantID:      tenantID,
				Name:          sp.Name,
				SKU:           sp.SKU,
				EAN:           sp.EAN,
				Price:         price,
				StockQuantity: sp.StockQuantity,
				Source:        "supplier",
				Category:      category,
				Metadata:      metadata,
				Tags:          []string{},
				Images:        json.RawMessage("[]"),
			}

			if err := s.productRepo.Create(ctx, tx, product); err != nil {
				resp.Errors = append(resp.Errors, model.ImportSupplierProductError{
					SupplierProductID: spID.String(),
					Reason:            fmt.Sprintf("failed to create product: %s", err),
				})
				continue
			}

			if err := s.supplierProdRepo.LinkToProduct(ctx, tx, spID, product.ID); err != nil {
				resp.Errors = append(resp.Errors, model.ImportSupplierProductError{
					SupplierProductID: spID.String(),
					Reason:            fmt.Sprintf("failed to link: %s", err),
				})
				continue
			}

			resp.Imported++
		}

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "supplier_products.imported",
			EntityType: "supplier",
			EntityID:   supplierID,
			Changes:    map[string]string{"imported": fmt.Sprintf("%d", resp.Imported)},
			IPAddress:  ip,
		})
	})
	if err != nil {
		return nil, err
	}

	return resp, nil
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

	// Use provider-based sync for registered supplier formats (e.g. btp)
	if integration.HasSupplierProvider(supplier.FeedFormat) {
		return s.syncViaProvider(ctx, tenantID, supplierID, supplier)
	}

	// Legacy IOF feed sync
	return s.syncViaIOF(ctx, tenantID, supplierID, supplier)
}

// syncViaProvider syncs products using a registered SupplierProvider (e.g. BTP API).
func (s *SupplierService) syncViaProvider(ctx context.Context, tenantID, supplierID uuid.UUID, supplier *model.Supplier) error {
	// Decrypt credentials from the linked integration
	if supplier.IntegrationID == nil {
		return fmt.Errorf("supplier %s has feed_format %q but no integration_id linked", supplierID, supplier.FeedFormat)
	}
	credJSON, err := s.integrationSvc.GetDecryptedCredentialsByID(ctx, tenantID, *supplier.IntegrationID)
	if err != nil {
		errMsg := fmt.Sprintf("failed to decrypt credentials: %s", err)
		if dbErr := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
			return s.supplierRepo.UpdateSyncStatus(ctx, tx, supplierID, time.Now(), &errMsg)
		}); dbErr != nil {
			s.logger.Error("failed to record supplier sync error", "supplier_id", supplierID, "error", dbErr)
		}
		return fmt.Errorf("decrypt supplier credentials: %w", err)
	}

	provider, err := integration.NewSupplierProvider(supplier.FeedFormat, credJSON, supplier.Settings)
	if err != nil {
		errMsg := err.Error()
		if dbErr := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
			return s.supplierRepo.UpdateSyncStatus(ctx, tx, supplierID, time.Now(), &errMsg)
		}); dbErr != nil {
			s.logger.Error("failed to record supplier sync error", "supplier_id", supplierID, "error", dbErr)
		}
		return fmt.Errorf("create supplier provider: %w", err)
	}

	products, err := provider.FetchInventory(ctx)
	if err != nil {
		errMsg := err.Error()
		if dbErr := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
			return s.supplierRepo.UpdateSyncStatus(ctx, tx, supplierID, time.Now(), &errMsg)
		}); dbErr != nil {
			s.logger.Error("failed to record supplier sync error", "supplier_id", supplierID, "error", dbErr)
		}
		return fmt.Errorf("fetch inventory: %w", err)
	}

	return s.upsertSupplierProducts(ctx, tenantID, supplierID, products)
}

// syncViaIOF syncs products from an IOF XML feed URL.
func (s *SupplierService) syncViaIOF(ctx context.Context, tenantID, supplierID uuid.UUID, supplier *model.Supplier) error {
	if supplier.FeedURL == nil || *supplier.FeedURL == "" {
		return ErrNoFeedURL
	}

	iofProducts, err := iof.ParseURL(ctx, *supplier.FeedURL)
	if err != nil {
		errMsg := err.Error()
		if dbErr := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
			return s.supplierRepo.UpdateSyncStatus(ctx, tx, supplierID, time.Now(), &errMsg)
		}); dbErr != nil {
			s.logger.Error("failed to record supplier sync error", "supplier_id", supplierID, "error", dbErr)
		}
		return fmt.Errorf("parse feed: %w", err)
	}

	// Convert IOF products to normalized format
	products := make([]integration.SupplierProduct, len(iofProducts))
	for i, fp := range iofProducts {
		products[i] = integration.SupplierProduct{
			ExternalID:    fp.ID,
			Name:          fp.Name,
			EAN:           fp.EAN,
			SKU:           fp.SKU,
			Price:         fp.Price,
			StockQuantity: fp.Stock,
			Category:      fp.Category,
			Attributes:    fp.Attributes,
		}
	}

	return s.upsertSupplierProducts(ctx, tenantID, supplierID, products)
}

// upsertSupplierProducts inserts or updates supplier products, auto-links by EAN,
// and resolves category mappings.
func (s *SupplierService) upsertSupplierProducts(ctx context.Context, tenantID, supplierID uuid.UUID, products []integration.SupplierProduct) error {
	syncedAt := time.Now()

	// Fetch supplier's default category for fallback
	var defaultCategoryID *uuid.UUID
	if err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		supplier, err := s.supplierRepo.FindByID(ctx, tx, supplierID)
		if err != nil {
			return err
		}
		if supplier != nil {
			defaultCategoryID = supplier.DefaultCategoryID
		}
		return nil
	}); err != nil {
		s.logger.Error("failed to fetch supplier default category", "supplier_id", supplierID, "error", err)
	}

	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
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

			var sourceCategory *string
			if fp.Category != "" {
				sourceCategory = &fp.Category
			}

			sp := &model.SupplierProduct{
				ID:             uuid.New(),
				TenantID:       tenantID,
				SupplierID:     supplierID,
				ExternalID:     fp.ExternalID,
				Name:           fp.Name,
				EAN:            ean,
				SKU:            sku,
				Price:          price,
				StockQuantity:  fp.StockQuantity,
				SourceCategory: sourceCategory,
				Metadata:       attrs,
				LastSyncedAt:   &syncedAt,
			}

			if err := s.supplierProdRepo.UpsertByExternalID(ctx, tx, sp); err != nil {
				s.logger.Error("failed to upsert supplier product",
					"supplier_id", supplierID, "external_id", fp.ExternalID, "error", err)
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

			// Resolve category and update linked product if available
			if sp.ProductID != nil && fp.Category != "" {
				categoryID := s.resolveCategoryForProduct(ctx, tx, supplierID, fp.Category, defaultCategoryID)
				if categoryID != nil {
					if _, err := tx.Exec(ctx,
						"UPDATE products SET category_id = $1 WHERE id = $2 AND category_id IS NULL",
						*categoryID, *sp.ProductID); err != nil {
						s.logger.Error("failed to set product category",
							"product_id", sp.ProductID, "category_id", categoryID, "error", err)
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

// ListCategoryMappings returns all category mappings for a supplier.
func (s *SupplierService) ListCategoryMappings(ctx context.Context, tenantID, supplierID uuid.UUID) ([]model.SupplierCategoryMapping, error) {
	var mappings []model.SupplierCategoryMapping
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		supplier, err := s.supplierRepo.FindByID(ctx, tx, supplierID)
		if err != nil {
			return err
		}
		if supplier == nil {
			return ErrSupplierNotFound
		}
		mappings, err = s.categoryMappingRepo.ListBySupplier(ctx, tx, supplierID)
		return err
	})
	if mappings == nil {
		mappings = []model.SupplierCategoryMapping{}
	}
	return mappings, err
}

// UpsertCategoryMapping creates or updates a category mapping for a supplier.
func (s *SupplierService) UpsertCategoryMapping(ctx context.Context, tenantID, supplierID uuid.UUID, req model.UpsertCategoryMappingRequest) (*model.SupplierCategoryMapping, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	m := &model.SupplierCategoryMapping{
		ID:             uuid.New(),
		TenantID:       tenantID,
		SupplierID:     supplierID,
		SourceCategory: req.SourceCategory,
		CategoryID:     req.CategoryID,
		AutoMatched:    false,
		Confirmed:      req.Confirmed,
	}

	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		supplier, err := s.supplierRepo.FindByID(ctx, tx, supplierID)
		if err != nil {
			return err
		}
		if supplier == nil {
			return ErrSupplierNotFound
		}
		return s.categoryMappingRepo.Upsert(ctx, tx, m)
	})
	if err != nil {
		return nil, err
	}
	return m, nil
}

// DeleteCategoryMapping removes a category mapping.
func (s *SupplierService) DeleteCategoryMapping(ctx context.Context, tenantID, supplierID, mappingID uuid.UUID) error {
	return database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		supplier, err := s.supplierRepo.FindByID(ctx, tx, supplierID)
		if err != nil {
			return err
		}
		if supplier == nil {
			return ErrSupplierNotFound
		}
		return s.categoryMappingRepo.Delete(ctx, tx, mappingID)
	})
}

// resolveCategoryForProduct resolves a category_id for a supplier product based on:
// 1. Confirmed mapping for the source_category
// 2. Auto-matched mapping (fuzzy match against existing categories)
// 3. Supplier's default_category_id
func (s *SupplierService) resolveCategoryForProduct(ctx context.Context, tx pgx.Tx, supplierID uuid.UUID, sourceCategory string, defaultCategoryID *uuid.UUID) *uuid.UUID {
	if sourceCategory == "" {
		return defaultCategoryID
	}

	// Check existing mapping
	mapping, err := s.categoryMappingRepo.FindBySourceCategory(ctx, tx, supplierID, sourceCategory)
	if err != nil {
		s.logger.Error("failed to find category mapping", "supplier_id", supplierID, "source_category", sourceCategory, "error", err)
		return defaultCategoryID
	}

	if mapping != nil && mapping.CategoryID != nil {
		return mapping.CategoryID
	}

	// No mapping yet — try fuzzy match
	if mapping == nil {
		matches, err := s.categoryRepo.FuzzyMatch(ctx, tx, sourceCategory)
		if err != nil {
			s.logger.Error("failed to fuzzy match category", "source_category", sourceCategory, "error", err)
			return defaultCategoryID
		}

		// Create auto-matched mapping
		autoMapping := &model.SupplierCategoryMapping{
			ID:             uuid.New(),
			TenantID:       uuid.UUID{}, // Set by RLS context
			SupplierID:     supplierID,
			SourceCategory: sourceCategory,
			AutoMatched:    true,
			Confirmed:      false,
		}
		if len(matches) > 0 {
			autoMapping.CategoryID = &matches[0].ID
		}
		// Best-effort: don't fail sync if mapping creation fails
		if err := s.categoryMappingRepo.Upsert(ctx, tx, autoMapping); err != nil {
			s.logger.Error("failed to create auto-matched mapping", "source_category", sourceCategory, "error", err)
		}

		if len(matches) > 0 {
			return &matches[0].ID
		}
	}

	return defaultCategoryID
}
