package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/asyncutil"
	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
	"github.com/openoms-org/openoms/apps/api-server/internal/integration/btp"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/netutil"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
	btpsdk "github.com/openoms-org/openoms/packages/btp-go-sdk"
	iof "github.com/openoms-org/openoms/packages/iof-parser"
)

var (
	// ErrSupplierNotFound is returned when a supplier does not exist.
	ErrSupplierNotFound = errors.New("supplier not found")
	// ErrSupplierProductNotFound is returned when a supplier product does not exist.
	ErrSupplierProductNotFound = errors.New("supplier product not found")
	// ErrNoFeedURL is returned when a supplier sync is attempted without a configured feed URL.
	ErrNoFeedURL = errors.New("supplier has no feed URL configured")
)

// SupplierService handles business logic for supplier management.
type SupplierService struct {
	supplierRepo        repository.SupplierRepo
	supplierProdRepo    repository.SupplierProductRepo
	categoryMappingRepo repository.SupplierCategoryMappingRepo
	allegroMappingRepo  repository.AllegroParameterMappingRepo
	categoryRepo        repository.ProductCategoryRepo
	productRepo         repository.ProductRepo
	auditRepo           repository.AuditRepo
	pool                *pgxpool.Pool
	webhookDispatch     *WebhookDispatchService
	integrationSvc      *IntegrationService
	stockSyncService    *StockSyncService
	availability        *SupplierAvailabilityService
	logger              *slog.Logger
}

// SetStockSyncService sets the stock sync service for propagating stock changes after supplier sync.
func (s *SupplierService) SetStockSyncService(svc *StockSyncService) {
	s.stockSyncService = svc
}

// WithAvailability wires the gated supplier-availability service (OPE-418) so the catalog
// sync additionally writes per-product availability snapshots. Nil-safe and a complete
// no-op until SUPPLIER_AVAILABILITY_ENABLED is set.
func (s *SupplierService) WithAvailability(svc *SupplierAvailabilityService) *SupplierService {
	s.availability = svc
	return s
}

// NewSupplierService creates a new SupplierService.
func NewSupplierService(
	supplierRepo repository.SupplierRepo,
	supplierProdRepo repository.SupplierProductRepo,
	categoryMappingRepo repository.SupplierCategoryMappingRepo,
	allegroMappingRepo repository.AllegroParameterMappingRepo,
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
		allegroMappingRepo:  allegroMappingRepo,
		categoryRepo:        categoryRepo,
		productRepo:         productRepo,
		auditRepo:           auditRepo,
		pool:                pool,
		webhookDispatch:     webhookDispatch,
		integrationSvc:      integrationSvc,
		logger:              logger,
	}
}

// List returns a paginated list of suppliers for a tenant.
func (s *SupplierService) List(ctx context.Context, tenantID uuid.UUID, filter model.SupplierListFilter) (model.ListResponse[model.Supplier], error) {
	var resp model.ListResponse[model.Supplier]
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		suppliers, total, err := s.supplierRepo.List(ctx, tx, filter)
		if err != nil {
			return err
		}
		resp = model.NewListResponse(suppliers, total, filter.Limit, filter.Offset)
		return nil
	})
	return resp, err
}

// Get returns a single supplier by ID.
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

// Create inserts a new supplier.
func (s *SupplierService) Create(ctx context.Context, tenantID uuid.UUID, req model.CreateSupplierRequest, actorID uuid.UUID, ip string) (*model.Supplier, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	// Sanitize user-facing text fields to prevent stored XSS
	req.Name = model.StripHTMLTags(req.Name)

	// Block private/internal URLs to prevent SSRF when feed is fetched
	if req.FeedURL != nil && *req.FeedURL != "" {
		if netutil.IsPrivateURL(*req.FeedURL) {
			return nil, NewValidationError(fmt.Errorf("feed_url must not point to a private/internal address"))
		}
	}

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
		asyncutil.SafeGo(func() { s.webhookDispatch.Dispatch(context.Background(), tenantID, "supplier.created", supplier) })
	}
	return supplier, nil
}

// Update modifies an existing supplier.
func (s *SupplierService) Update(ctx context.Context, tenantID, supplierID uuid.UUID, req model.UpdateSupplierRequest, actorID uuid.UUID, ip string) (*model.Supplier, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	// SSRF prevention: validate feed URL is not a private/internal address
	if req.FeedURL != nil && *req.FeedURL != "" {
		if netutil.IsPrivateURL(*req.FeedURL) {
			return nil, NewValidationError(fmt.Errorf("feed URL must not point to a private network address"))
		}
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
		asyncutil.SafeGo(func() { s.webhookDispatch.Dispatch(context.Background(), tenantID, "supplier.updated", supplier) })
	}
	return supplier, err
}

// Delete removes a supplier by ID.
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
		asyncutil.SafeGo(func() {
			s.webhookDispatch.Dispatch(context.Background(), tenantID, "supplier.deleted", map[string]any{"supplier_id": supplierID.String()})
		})
	}
	return err
}

// ListProducts returns a paginated list of products linked to a supplier.
func (s *SupplierService) ListProducts(ctx context.Context, tenantID uuid.UUID, filter model.SupplierProductListFilter) (model.ListResponse[model.SupplierProduct], error) {
	var resp model.ListResponse[model.SupplierProduct]
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		products, total, err := s.supplierProdRepo.List(ctx, tx, filter)
		if err != nil {
			return err
		}
		resp = model.NewListResponse(products, total, filter.Limit, filter.Offset)
		return nil
	})
	return resp, err
}

// ListAllProducts returns supplier products across all suppliers with supplier names included.
func (s *SupplierService) ListAllProducts(ctx context.Context, tenantID uuid.UUID, params model.SupplierProductListAllParams) (model.ListResponse[model.SupplierProductWithSupplier], error) {
	var resp model.ListResponse[model.SupplierProductWithSupplier]
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		products, total, err := s.supplierProdRepo.ListAll(ctx, tx, params)
		if err != nil {
			return err
		}
		resp = model.NewListResponse(products, total, params.Limit, params.Offset)
		return nil
	})
	return resp, err
}

// LinkProduct associates a supplier product with an internal product.
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

// categoryInfo holds resolved OMS category name and ID.
type categoryInfo struct {
	Name string
	ID   *uuid.UUID
}

// buildCategoryMap loads category mappings for a supplier and resolves them to OMS category names.
func (s *SupplierService) buildCategoryMap(ctx context.Context, tx pgx.Tx, supplierID uuid.UUID) (map[string]categoryInfo, error) {
	mappings, err := s.categoryMappingRepo.ListBySupplier(ctx, tx, supplierID)
	if err != nil {
		return nil, err
	}
	catMap := make(map[string]categoryInfo, len(mappings))
	for _, m := range mappings {
		if m.CategoryID != nil {
			cat, catErr := s.categoryRepo.FindByID(ctx, tx, *m.CategoryID)
			if catErr == nil && cat != nil {
				catMap[m.SourceCategory] = categoryInfo{Name: cat.Name, ID: m.CategoryID}
			}
		}
	}
	return catMap, nil
}

// mapSupplierToProduct converts a SupplierProduct to an OMS Product using category mappings.
func (s *SupplierService) mapSupplierToProduct(tenantID uuid.UUID, sp *model.SupplierProduct, catMap map[string]categoryInfo, defaultCategoryID *uuid.UUID) (product *model.Product, categoryID *uuid.UUID) {
	var category *string
	if sp.SourceCategory != nil {
		if info, ok := catMap[*sp.SourceCategory]; ok {
			category = &info.Name
			categoryID = info.ID
		}
	}
	if category == nil {
		category = sp.SourceCategory
	}
	if categoryID == nil {
		categoryID = defaultCategoryID
	}

	price := 0.0
	if sp.Price != nil {
		price = *sp.Price
	}

	metadata := sp.Metadata
	if metadata == nil {
		metadata = json.RawMessage("{}")
	}

	var metaMap map[string]any
	_ = json.Unmarshal(metadata, &metaMap)

	var descLong string
	if v, ok := metaMap["description"].(string); ok {
		descLong = v
	}

	var imageURL *string
	if v, ok := metaMap["image_url"].(string); ok && v != "" {
		imageURL = &v
	}

	var images json.RawMessage
	if rawImages, ok := metaMap["images"]; ok {
		if arr, ok := rawImages.([]any); ok && len(arr) > 0 {
			imagesArr, _ := json.Marshal(arr)
			images = imagesArr
		}
	}
	if images == nil && imageURL != nil {
		imagesArr, _ := json.Marshal([]string{*imageURL})
		images = imagesArr
	}
	if images == nil {
		images = json.RawMessage("[]")
	}

	var weight *float64
	if v, ok := metaMap["weight"].(float64); ok && v > 0 {
		weight = &v
	}

	var tags []string
	if v, ok := metaMap["brand"].(string); ok && v != "" {
		tags = append(tags, v)
	}
	if tags == nil {
		tags = []string{}
	}

	product = &model.Product{
		ID:              uuid.New(),
		TenantID:        tenantID,
		Name:            sp.Name,
		SKU:             sp.SKU,
		EAN:             sp.EAN,
		Price:           price,
		StockQuantity:   sp.StockQuantity,
		Source:          "supplier",
		Category:        category,
		DescriptionLong: descLong,
		ImageURL:        imageURL,
		Images:          images,
		Weight:          weight,
		Tags:            tags,
		Metadata:        metadata,
	}
	return product, categoryID
}

// ImportSingleProduct imports a single supplier product into the OMS catalog and returns the created product.
// If the supplier product is already linked, it returns the existing product.
func (s *SupplierService) ImportSingleProduct(ctx context.Context, tenantID, supplierID, supplierProductID uuid.UUID, actorID uuid.UUID, ip string) (*model.Product, error) {
	var result *model.Product

	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		supplier, err := s.supplierRepo.FindByID(ctx, tx, supplierID)
		if err != nil {
			return err
		}
		if supplier == nil {
			return ErrSupplierNotFound
		}

		sp, err := s.supplierProdRepo.FindByID(ctx, tx, supplierProductID)
		if err != nil {
			return err
		}
		if sp == nil {
			return ErrSupplierProductNotFound
		}
		if sp.SupplierID != supplierID {
			return ErrSupplierProductNotFound
		}

		// Already linked — return existing product
		if sp.ProductID != nil {
			result, err = s.productRepo.FindByID(ctx, tx, *sp.ProductID)
			return err
		}

		catMap, err := s.buildCategoryMap(ctx, tx, supplierID)
		if err != nil {
			return err
		}

		product, catID := s.mapSupplierToProduct(tenantID, sp, catMap, supplier.DefaultCategoryID)

		if err := s.productRepo.Create(ctx, tx, product); err != nil {
			return fmt.Errorf("create product: %w", err)
		}

		if catID != nil {
			if _, err := tx.Exec(ctx,
				"UPDATE products SET category_id = $1 WHERE id = $2",
				*catID, product.ID); err != nil {
				s.logger.Error("failed to set product category_id",
					"product_id", product.ID, "category_id", catID, "error", err)
			}
		}

		if err := s.supplierProdRepo.LinkToProduct(ctx, tx, supplierProductID, product.ID); err != nil {
			return fmt.Errorf("link product: %w", err)
		}

		result = product

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "supplier_product.imported",
			EntityType: "supplier_product",
			EntityID:   supplierProductID,
			Changes:    map[string]string{"product_id": product.ID.String()},
			IPAddress:  ip,
		})
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// ImportProducts imports supplier products from a CSV or feed payload.
func (s *SupplierService) ImportProducts(ctx context.Context, tenantID, supplierID uuid.UUID, req model.ImportSupplierProductsRequest, actorID uuid.UUID, ip string) (*model.ImportSupplierProductsResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	resp := &model.ImportSupplierProductsResponse{}

	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		supplier, err := s.supplierRepo.FindByID(ctx, tx, supplierID)
		if err != nil {
			return err
		}
		if supplier == nil {
			return ErrSupplierNotFound
		}

		spProducts, err := s.supplierProdRepo.FindByIDs(ctx, tx, req.SupplierProductIDs)
		if err != nil {
			return err
		}

		spMap := make(map[uuid.UUID]*model.SupplierProduct, len(spProducts))
		for i := range spProducts {
			spMap[spProducts[i].ID] = &spProducts[i]
		}

		catMap, err := s.buildCategoryMap(ctx, tx, supplierID)
		if err != nil {
			return err
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

			if sp.ProductID != nil {
				resp.Skipped++
				continue
			}

			if sp.SupplierID != supplierID {
				resp.Errors = append(resp.Errors, model.ImportSupplierProductError{
					SupplierProductID: spID.String(),
					Reason:            "supplier product does not belong to this supplier",
				})
				continue
			}

			product, catID := s.mapSupplierToProduct(tenantID, sp, catMap, supplier.DefaultCategoryID)

			if err := s.productRepo.Create(ctx, tx, product); err != nil {
				resp.Errors = append(resp.Errors, model.ImportSupplierProductError{
					SupplierProductID: spID.String(),
					Reason:            fmt.Sprintf("failed to create product: %s", err),
				})
				continue
			}

			if catID != nil {
				if _, err := tx.Exec(ctx,
					"UPDATE products SET category_id = $1 WHERE id = $2",
					*catID, product.ID); err != nil {
					s.logger.Error("failed to set product category_id",
						"product_id", product.ID, "category_id", catID, "error", err)
				}
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

// ValidateSyncable checks that a supplier exists and has a feed URL configured.
// Used by the handler to validate before starting an async sync.
func (s *SupplierService) ValidateSyncable(ctx context.Context, tenantID, supplierID uuid.UUID) error {
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
	if integration.HasSupplierProvider(supplier.FeedFormat) {
		if supplier.IntegrationID == nil {
			return fmt.Errorf("supplier uses provider %q but has no integration configured: %w", supplier.FeedFormat, ErrNoFeedURL)
		}
		return nil
	}
	if supplier.FeedURL == nil || *supplier.FeedURL == "" {
		return ErrNoFeedURL
	}
	return nil
}

// SyncFeed fetches the supplier's product feed URL and reconciles products.
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

	// BTP XML catalogue feed (no integration/credentials needed — direct URL)
	if supplier.FeedFormat == "xml" {
		return s.syncViaXML(ctx, tenantID, supplierID, supplier)
	}

	// Legacy IOF feed sync
	return s.syncViaIOF(ctx, tenantID, supplierID, supplier)
}

// syncViaProvider syncs products using a registered SupplierProvider (e.g. BTP API).
// Uses full sync (FetchProducts with descriptions/images) when last full sync is >24h old,
// otherwise uses inventory-only sync (FetchInventory for stock/prices).
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

	fullSync := s.shouldFullSync(supplier)

	var products []integration.SupplierProduct
	if fullSync {
		products, err = provider.FetchProducts(ctx)
	} else {
		products, err = provider.FetchInventory(ctx)
	}
	if err != nil {
		errMsg := err.Error()
		if dbErr := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
			return s.supplierRepo.UpdateSyncStatus(ctx, tx, supplierID, time.Now(), &errMsg)
		}); dbErr != nil {
			s.logger.Error("failed to record supplier sync error", "supplier_id", supplierID, "error", dbErr)
		}
		kind := "inventory"
		if fullSync {
			kind = "products"
		}
		return fmt.Errorf("fetch %s: %w", kind, err)
	}

	if err := s.upsertSupplierProducts(ctx, tenantID, supplierID, products, "api"); err != nil {
		return err
	}

	// Record last full sync timestamp
	if fullSync {
		if dbErr := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
			return s.supplierRepo.UpdateLastFullSync(ctx, tx, supplierID, time.Now())
		}); dbErr != nil {
			s.logger.Error("failed to update last full sync", "supplier_id", supplierID, "error", dbErr)
		}
	}

	return nil
}

// shouldFullSync checks if a full product sync (with descriptions/images) is needed.
// Returns true if last_full_sync_at is missing or older than 24 hours.
func (s *SupplierService) shouldFullSync(supplier *model.Supplier) bool {
	if supplier.Settings == nil {
		return true
	}

	var settings map[string]any
	if err := json.Unmarshal(supplier.Settings, &settings); err != nil {
		return true
	}

	lastFullSyncStr, ok := settings["last_full_sync_at"].(string)
	if !ok || lastFullSyncStr == "" {
		return true
	}

	lastFullSync, err := time.Parse(time.RFC3339, lastFullSyncStr)
	if err != nil {
		return true
	}

	// Read configurable interval from settings (default 24h)
	intervalHours := 24.0
	if v, ok := settings["xml_sync_interval_hours"].(float64); ok && v > 0 {
		intervalHours = v
	}

	return time.Since(lastFullSync) > time.Duration(intervalHours*float64(time.Hour))
}

// syncViaXML syncs products from a BTP-format XML catalogue URL.
// Hybrid mode: full catalogue from XML every 24h, inventory updates from BTP API in between
// (when integration_id is linked). Orders/fulfillment also use the API.
func (s *SupplierService) syncViaXML(ctx context.Context, tenantID, supplierID uuid.UUID, supplier *model.Supplier) error {
	fullSync := s.shouldFullSync(supplier)

	// Inventory-only sync via BTP API (between full XML syncs)
	if !fullSync && supplier.IntegrationID != nil {
		return s.syncXMLInventory(ctx, tenantID, supplierID, supplier)
	}

	// Full catalogue sync from XML
	if supplier.FeedURL == nil || *supplier.FeedURL == "" {
		return ErrNoFeedURL
	}

	items, err := btpsdk.ParseCatalogueURL(ctx, *supplier.FeedURL, netutil.SafeHTTPClient(60*time.Second))
	if err != nil {
		errMsg := err.Error()
		if dbErr := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
			return s.supplierRepo.UpdateSyncStatus(ctx, tx, supplierID, time.Now(), &errMsg)
		}); dbErr != nil {
			s.logger.Error("failed to record supplier sync error", "supplier_id", supplierID, "error", dbErr)
		}
		return fmt.Errorf("parse XML catalogue: %w", err)
	}

	products := btp.MapCatalogueProducts(items)

	// XML source: skip stock/price on linked products when API integration exists
	source := "full"
	if supplier.IntegrationID != nil {
		source = "xml"
	}

	if err := s.upsertSupplierProducts(ctx, tenantID, supplierID, products, source); err != nil {
		return err
	}

	// Remove products that disappeared from XML feed
	if supplier.IntegrationID != nil {
		currentIDs := make([]string, len(products))
		for i, p := range products {
			currentIDs[i] = p.ExternalID
		}
		s.removeStaleProducts(ctx, tenantID, supplierID, currentIDs)
	}

	// Record last full sync timestamp
	if dbErr := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		return s.supplierRepo.UpdateLastFullSync(ctx, tx, supplierID, time.Now())
	}); dbErr != nil {
		s.logger.Error("failed to update last full sync", "supplier_id", supplierID, "error", dbErr)
	}

	return nil
}

// syncXMLInventory performs an inventory-only sync for XML-format suppliers using BTP API.
func (s *SupplierService) syncXMLInventory(ctx context.Context, tenantID, supplierID uuid.UUID, supplier *model.Supplier) error {
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

	provider, err := integration.NewSupplierProvider("btp", credJSON, supplier.Settings)
	if err != nil {
		errMsg := err.Error()
		if dbErr := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
			return s.supplierRepo.UpdateSyncStatus(ctx, tx, supplierID, time.Now(), &errMsg)
		}); dbErr != nil {
			s.logger.Error("failed to record supplier sync error", "supplier_id", supplierID, "error", dbErr)
		}
		return fmt.Errorf("create btp provider for inventory sync: %w", err)
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

	return s.upsertSupplierProducts(ctx, tenantID, supplierID, products, "api")
}

// syncViaIOF syncs products from an IOF XML feed URL.
func (s *SupplierService) syncViaIOF(ctx context.Context, tenantID, supplierID uuid.UUID, supplier *model.Supplier) error {
	if supplier.FeedURL == nil || *supplier.FeedURL == "" {
		return ErrNoFeedURL
	}

	iofProducts, err := iof.ParseURL(ctx, *supplier.FeedURL, netutil.SafeHTTPClient(60*time.Second))
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
		brand := fp.Attributes["producer"]
		products[i] = integration.SupplierProduct{
			ExternalID:    fp.ID,
			Name:          fp.Name,
			Description:   fp.Description,
			EAN:           fp.EAN,
			SKU:           fp.SKU,
			Price:         fp.Price,
			StockQuantity: fp.Stock,
			Category:      fp.Category,
			Brand:         brand,
			ImageURL:      fp.ImageURL,
			Weight:        fp.Weight,
			Attributes:    fp.Attributes,
		}
	}

	return s.upsertSupplierProducts(ctx, tenantID, supplierID, products, "full")
}

// upsertSupplierProducts inserts or updates supplier products, auto-links by EAN,
// and resolves category mappings.
// syncSource controls stock/price updates on linked OMS products:
//   - "xml": skip stock/price (XML stock is always 0 for BTP suppliers with API)
//   - "api": full sync including stock/price
//   - "full": full sync (default, for IOF/CSV)
func (s *SupplierService) upsertSupplierProducts(ctx context.Context, tenantID, supplierID uuid.UUID, products []integration.SupplierProduct, syncSource string) error {
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
			// Build rich metadata including description, images, weight, brand
			meta := map[string]any{}
			if fp.Description != "" {
				meta["description"] = fp.Description
			}
			if fp.Brand != "" {
				meta["brand"] = fp.Brand
			}
			if fp.ImageURL != "" {
				meta["image_url"] = fp.ImageURL
			}
			if fp.Weight > 0 {
				meta["weight"] = fp.Weight
			}
			if fp.RetailPrice > 0 {
				meta["retail_price"] = fp.RetailPrice
			}
			if len(fp.Images) > 0 {
				meta["images"] = fp.Images
			}
			if len(fp.Attributes) > 0 {
				meta["attributes"] = fp.Attributes
			}
			metaJSON, _ := json.Marshal(meta)

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
				Metadata:       metaJSON,
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
					} else {
						sp.ProductID = &productID
					}
				}
			}

			// OPE-418: write the supplier-availability snapshot (gated, best-effort).
			// Off by default — the legacy supplier_products.stock_quantity path is unchanged.
			// A snapshot-write failure must never fail a catalog sync, so the error is logged
			// and the loop continues. Single-warehouse feeds use the '' warehouse id;
			// multi-warehouse enrichment is a later per-format task.
			if s.availability.Enabled() {
				if _, snapErr := s.availability.Repo().UpsertSnapshot(ctx, tx, model.SupplierAvailability{
					TenantID:             tenantID,
					SupplierID:           supplierID,
					SupplierProductID:    sp.ID,
					ProductID:            sp.ProductID, // *uuid.UUID, nil until mapped
					WarehouseExternalID:  "",
					SourceQuantity:       sp.StockQuantity,
					AvailabilityType:     model.AvailabilityExactQuantity,
					ReservationSupported: false,
					FreshnessObservedAt:  syncedAt,
				}); snapErr != nil {
					s.logger.Error("failed to upsert supplier availability snapshot",
						"supplier_id", supplierID, "supplier_product_id", sp.ID, "error", snapErr)
				}
			}

			// Sync data to linked OMS product.
			if sp.ProductID != nil {
				imagesJSON := json.RawMessage("[]")
				if len(fp.Images) > 0 {
					imagesJSON, _ = json.Marshal(fp.Images)
				}

				if syncSource == "xml" {
					// XML source: only update enrichment data (catalog), skip stock/price.
					// BTP XML always reports stock=0, so we must not overwrite API stock.
					if _, err := tx.Exec(ctx,
						`UPDATE products SET
							weight = COALESCE(weight, NULLIF($1::float8, 0)),
							description_long = CASE WHEN COALESCE(description_long, '') = '' THEN COALESCE(NULLIF($2, ''), description_long) ELSE description_long END,
							image_url = COALESCE(image_url, NULLIF($3, '')),
							images = CASE WHEN images IS NULL OR images = '[]'::jsonb THEN COALESCE(NULLIF($4::jsonb, '[]'::jsonb), images) ELSE images END,
							metadata = COALESCE(metadata, '{}'::jsonb) || $5::jsonb,
							updated_at = NOW()
						 WHERE id = $6`,
						fp.Weight, fp.Description, fp.ImageURL, string(imagesJSON), string(metaJSON), *sp.ProductID); err != nil {
						s.logger.Error("failed to sync enrichment to linked product",
							"product_id", sp.ProductID, "error", err)
					}
				} else {
					// API/full source: sync everything including stock and price.
					if _, err := tx.Exec(ctx,
						`UPDATE products SET
							stock_quantity = $1,
							price = COALESCE($2, price),
							weight = COALESCE(weight, NULLIF($3::float8, 0)),
							description_long = CASE WHEN COALESCE(description_long, '') = '' THEN COALESCE(NULLIF($4, ''), description_long) ELSE description_long END,
							image_url = COALESCE(image_url, NULLIF($5, '')),
							images = CASE WHEN images IS NULL OR images = '[]'::jsonb THEN COALESCE(NULLIF($6::jsonb, '[]'::jsonb), images) ELSE images END,
							metadata = COALESCE(metadata, '{}'::jsonb) || $7::jsonb,
							updated_at = NOW()
						 WHERE id = $8`,
						fp.StockQuantity, price, fp.Weight, fp.Description, fp.ImageURL, string(imagesJSON), string(metaJSON), *sp.ProductID); err != nil {
						s.logger.Error("failed to sync to linked product",
							"product_id", sp.ProductID, "error", err)
					}
				}
			}

			// Resolve category and update linked product if available
			if sp.ProductID != nil && fp.Category != "" {
				categoryID := s.resolveCategoryForProduct(ctx, tx, tenantID, supplierID, fp.Category, defaultCategoryID)
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
func (s *SupplierService) resolveCategoryForProduct(ctx context.Context, tx pgx.Tx, tenantID, supplierID uuid.UUID, sourceCategory string, defaultCategoryID *uuid.UUID) *uuid.UUID {
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
			TenantID:       tenantID,
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

// UnlinkProduct removes the link between a supplier product and an OMS product.
func (s *SupplierService) UnlinkProduct(ctx context.Context, tenantID, supplierProductID uuid.UUID, actorID uuid.UUID, ip string) error {
	return database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		sp, err := s.supplierProdRepo.FindByID(ctx, tx, supplierProductID)
		if err != nil {
			return err
		}
		if sp == nil {
			return ErrSupplierProductNotFound
		}

		if err := s.supplierProdRepo.UnlinkProduct(ctx, tx, supplierProductID); err != nil {
			return err
		}

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "supplier_product.unlinked",
			EntityType: "supplier_product",
			EntityID:   supplierProductID,
			IPAddress:  ip,
		})
	})
}

// DeleteSupplierProduct deletes a single supplier product.
func (s *SupplierService) DeleteSupplierProduct(ctx context.Context, tenantID, supplierID, supplierProductID uuid.UUID, actorID uuid.UUID, ip string) error {
	return database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		sp, err := s.supplierProdRepo.FindByID(ctx, tx, supplierProductID)
		if err != nil {
			return err
		}
		if sp == nil {
			return ErrSupplierProductNotFound
		}
		if sp.SupplierID != supplierID {
			return ErrSupplierProductNotFound
		}

		if err := s.supplierProdRepo.Delete(ctx, tx, supplierProductID); err != nil {
			return err
		}

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "supplier_product.deleted",
			EntityType: "supplier_product",
			EntityID:   supplierProductID,
			IPAddress:  ip,
		})
	})
}

// BulkDeleteSupplierProducts deletes multiple supplier products.
func (s *SupplierService) BulkDeleteSupplierProducts(ctx context.Context, tenantID, supplierID uuid.UUID, req model.BulkDeleteSupplierProductsRequest, actorID uuid.UUID, ip string) (int, error) {
	if err := req.Validate(); err != nil {
		return 0, NewValidationError(err)
	}

	var deleted int
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		// Verify supplier exists
		supplier, err := s.supplierRepo.FindByID(ctx, tx, supplierID)
		if err != nil {
			return err
		}
		if supplier == nil {
			return ErrSupplierNotFound
		}

		var count int
		count, err = s.supplierProdRepo.BulkDelete(ctx, tx, req.SupplierProductIDs)
		if err != nil {
			return err
		}
		deleted = count

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "supplier_products.bulk_deleted",
			EntityType: "supplier",
			EntityID:   supplierID,
			Changes:    map[string]string{"deleted": fmt.Sprintf("%d", deleted)},
			IPAddress:  ip,
		})
	})
	return deleted, err
}

// ListSourceCategories returns distinct source categories for a supplier's products.
func (s *SupplierService) ListSourceCategories(ctx context.Context, tenantID, supplierID uuid.UUID) ([]string, error) {
	var categories []string
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var err error
		categories, err = s.supplierProdRepo.ListSourceCategories(ctx, tx, supplierID)
		return err
	})
	if categories == nil {
		categories = []string{}
	}
	return categories, err
}

// ListAllegroMappings returns Allegro parameter mappings for a supplier and category.
func (s *SupplierService) ListAllegroMappings(ctx context.Context, tenantID, supplierID uuid.UUID, allegroCategoryID string) ([]model.AllegroParameterMapping, error) {
	var mappings []model.AllegroParameterMapping
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		supplier, err := s.supplierRepo.FindByID(ctx, tx, supplierID)
		if err != nil {
			return err
		}
		if supplier == nil {
			return ErrSupplierNotFound
		}
		mappings, err = s.allegroMappingRepo.ListBySupplierAndCategory(ctx, tx, supplierID, allegroCategoryID)
		return err
	})
	if mappings == nil {
		mappings = []model.AllegroParameterMapping{}
	}
	return mappings, err
}

// BulkUpsertAllegroMappings creates or updates Allegro parameter mappings for a supplier.
func (s *SupplierService) BulkUpsertAllegroMappings(ctx context.Context, tenantID, supplierID uuid.UUID, req model.BulkUpsertAllegroMappingsRequest) ([]model.AllegroParameterMapping, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	var result []model.AllegroParameterMapping
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		supplier, err := s.supplierRepo.FindByID(ctx, tx, supplierID)
		if err != nil {
			return err
		}
		if supplier == nil {
			return ErrSupplierNotFound
		}

		models := make([]*model.AllegroParameterMapping, len(req.Mappings))
		for i, m := range req.Mappings {
			vm := m.ValueMapping
			if vm == nil {
				vm = json.RawMessage("{}")
			}
			models[i] = &model.AllegroParameterMapping{
				ID:                uuid.New(),
				TenantID:          tenantID,
				SupplierID:        supplierID,
				AllegroCategoryID: req.AllegroCategoryID,
				AllegroParamID:    m.AllegroParamID,
				AllegroParamName:  m.AllegroParamName,
				SourceType:        m.SourceType,
				SourceKey:         m.SourceKey,
				ValueMapping:      vm,
			}
		}

		if err := s.allegroMappingRepo.BulkUpsert(ctx, tx, models); err != nil {
			return err
		}
		result = make([]model.AllegroParameterMapping, len(models))
		for i, m := range models {
			result[i] = *m
		}
		return nil
	})
	return result, err
}

// DeleteAllegroMapping removes a single Allegro parameter mapping.
func (s *SupplierService) DeleteAllegroMapping(ctx context.Context, tenantID, supplierID, mappingID uuid.UUID) error {
	return database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		supplier, err := s.supplierRepo.FindByID(ctx, tx, supplierID)
		if err != nil {
			return err
		}
		if supplier == nil {
			return ErrSupplierNotFound
		}
		return s.allegroMappingRepo.Delete(ctx, tx, mappingID)
	})
}

// ListSupplierAttributes returns unique attribute names from supplier products' metadata.
func (s *SupplierService) ListSupplierAttributes(ctx context.Context, tenantID, supplierID uuid.UUID) ([]string, error) {
	var attrs []string
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		supplier, err := s.supplierRepo.FindByID(ctx, tx, supplierID)
		if err != nil {
			return err
		}
		if supplier == nil {
			return ErrSupplierNotFound
		}
		attrs, err = s.supplierProdRepo.ListAttributes(ctx, tx, supplierID)
		return err
	})
	if attrs == nil {
		attrs = []string{}
	}
	return attrs, err
}

// ListAllegroMappingCategories returns distinct Allegro category IDs with saved mappings.
func (s *SupplierService) ListAllegroMappingCategories(ctx context.Context, tenantID, supplierID uuid.UUID) ([]string, error) {
	var categories []string
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		supplier, err := s.supplierRepo.FindByID(ctx, tx, supplierID)
		if err != nil {
			return err
		}
		if supplier == nil {
			return ErrSupplierNotFound
		}
		categories, err = s.allegroMappingRepo.ListCategoriesForSupplier(ctx, tx, supplierID)
		return err
	})
	if categories == nil {
		categories = []string{}
	}
	return categories, err
}

// FindSupplierForProduct returns the supplier_id linked to a product via supplier_products.
func (s *SupplierService) FindSupplierForProduct(ctx context.Context, tenantID, productID uuid.UUID) (*uuid.UUID, error) {
	var supplierID *uuid.UUID
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var err error
		supplierID, err = s.supplierProdRepo.FindSupplierIDByProductID(ctx, tx, productID)
		return err
	})
	return supplierID, err
}

// ---------------------------------------------------------------------------
// BTP Wizard
// ---------------------------------------------------------------------------

// BTPWizardStartImport creates a new BTP supplier and starts an async XML import (step 1).
func (s *SupplierService) BTPWizardStartImport(ctx context.Context, tenantID uuid.UUID, req model.BTPWizardStartImportRequest, actorID uuid.UUID, ip string) (*model.Supplier, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	name := model.StripHTMLTags(req.Name)
	feedURL := req.FeedURL

	settings, _ := json.Marshal(map[string]any{
		"setup_step":       "xml_import",
		"import_status":    "pending",
		"import_total":     0,
		"import_processed": 0,
	})

	supplier := &model.Supplier{
		ID:                  uuid.New(),
		TenantID:            tenantID,
		Name:                name,
		FeedURL:             &feedURL,
		FeedFormat:          "xml",
		Status:              "inactive",
		Settings:            settings,
		SyncIntervalMinutes: 60,
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
			IPAddress:  ip,
		})
	})
	if err != nil {
		return nil, fmt.Errorf("create btp supplier: %w", err)
	}

	// Start async XML import
	asyncutil.SafeGo(func() { s.runBTPXMLImport(tenantID, supplier.ID, feedURL) })

	return supplier, nil
}

// runBTPXMLImport runs the BTP XML catalogue import in a background goroutine.
func (s *SupplierService) runBTPXMLImport(tenantID, supplierID uuid.UUID, feedURL string) {
	ctx := context.Background()

	// Update status to running
	if err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		return s.supplierRepo.UpdateSettingsKeys(ctx, tx, supplierID, map[string]any{
			"import_status": "running",
		})
	}); err != nil {
		s.logger.Error("btp wizard: failed to set import running", "error", err)
		return
	}

	// Parse XML catalogue
	items, err := btpsdk.ParseCatalogueURL(ctx, feedURL, netutil.SafeHTTPClient(120*time.Second))
	if err != nil {
		s.logger.Error("btp wizard: XML parse failed", "supplier_id", supplierID, "error", err)
		_ = database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
			return s.supplierRepo.UpdateSettingsKeys(ctx, tx, supplierID, map[string]any{
				"import_status": "failed",
				"import_error":  err.Error(),
			})
		})
		return
	}

	products := btp.MapCatalogueProducts(items)
	total := len(products)

	// Update total
	_ = database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		return s.supplierRepo.UpdateSettingsKeys(ctx, tx, supplierID, map[string]any{
			"import_total": total,
		})
	})

	// Upsert in batches of 100, updating progress
	batchSize := 100
	for i := 0; i < total; i += batchSize {
		end := min(i+batchSize, total)
		batch := products[i:end]

		if err := s.upsertSupplierProducts(ctx, tenantID, supplierID, batch, "xml"); err != nil {
			s.logger.Error("btp wizard: batch upsert failed", "supplier_id", supplierID, "batch", i, "error", err)
			_ = database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
				return s.supplierRepo.UpdateSettingsKeys(ctx, tx, supplierID, map[string]any{
					"import_status": "failed",
					"import_error":  err.Error(),
				})
			})
			return
		}

		// Update progress
		_ = database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
			return s.supplierRepo.UpdateSettingsKeys(ctx, tx, supplierID, map[string]any{
				"import_processed": end,
			})
		})
	}

	// Mark complete
	_ = database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		return s.supplierRepo.UpdateSettingsKeys(ctx, tx, supplierID, map[string]any{
			"import_status":    "completed",
			"import_processed": total,
			"setup_step":       "api_keys",
		})
	})
	_ = database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		return s.supplierRepo.UpdateLastFullSync(ctx, tx, supplierID, time.Now())
	})

	s.logger.Info("btp wizard: XML import completed", "supplier_id", supplierID, "products", total)
}

// BTPWizardGetImportProgress returns the current progress of a BTP XML import (step 1 polling).
func (s *SupplierService) BTPWizardGetImportProgress(ctx context.Context, tenantID, supplierID uuid.UUID) (*model.BTPImportProgressResponse, error) {
	var resp model.BTPImportProgressResponse
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		supplier, err := s.supplierRepo.FindByID(ctx, tx, supplierID)
		if err != nil {
			return err
		}
		if supplier == nil {
			return ErrSupplierNotFound
		}

		var settings map[string]any
		if supplier.Settings != nil {
			_ = json.Unmarshal(supplier.Settings, &settings)
		}

		if status, ok := settings["import_status"].(string); ok {
			resp.Status = status
		} else {
			resp.Status = "pending"
		}
		if total, ok := settings["import_total"].(float64); ok {
			resp.Total = int(total)
		}
		if processed, ok := settings["import_processed"].(float64); ok {
			resp.Processed = int(processed)
		}
		if errStr, ok := settings["import_error"].(string); ok {
			resp.Error = errStr
		}
		return nil
	})
	return &resp, err
}

// validateBTPBaseURL performs an early, defense-in-depth check on a user-supplied BTP
// base URL before it is handed to the SDK client. It is intentionally a pure, DNS-free
// check (literal parse only) so it never blocks on network I/O; the authoritative SSRF
// guard remains netutil.SafeHTTPClient, whose dialer re-resolves the host at connect time
// and rejects private/loopback/link-local/metadata IPs (including DNS-rebinding cases).
// An empty base_url is allowed — the SDK falls back to its public default endpoint.
func validateBTPBaseURL(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("base_url is not a valid URL: %w", err)
	}
	if u.Scheme != "https" {
		return errors.New("base_url must use the https scheme")
	}
	host := u.Hostname()
	if host == "" {
		return errors.New("base_url must include a host")
	}
	if strings.EqualFold(host, "localhost") || strings.EqualFold(host, "localhost.") {
		return errors.New("base_url host is not allowed")
	}
	// Reject literal private/loopback/link-local/metadata IP addresses early.
	if ip := net.ParseIP(host); ip != nil && netutil.IsPrivateIP(ip) {
		return errors.New("base_url host is not allowed")
	}
	return nil
}

// BTPWizardSetAPIKeys validates and saves BTP API credentials (step 2).
func (s *SupplierService) BTPWizardSetAPIKeys(ctx context.Context, tenantID, supplierID uuid.UUID, req model.BTPWizardSetAPIKeysRequest, actorID uuid.UUID, ip string) (*model.Supplier, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	// Defense-in-depth SSRF check on the user-supplied base URL before it reaches the
	// SDK client. The authoritative guard is netutil.SafeHTTPClient below (re-resolves
	// DNS at connect time and blocks private/loopback/link-local/metadata targets,
	// defeating DNS rebinding); this rejects the obvious cases early with a clear error.
	if err := validateBTPBaseURL(req.BaseURL); err != nil {
		return nil, NewValidationError(err)
	}

	// Verify supplier is on the right step
	var supplier *model.Supplier
	if err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var err error
		supplier, err = s.supplierRepo.FindByID(ctx, tx, supplierID)
		return err
	}); err != nil {
		return nil, err
	}
	if supplier == nil {
		return nil, ErrSupplierNotFound
	}

	var settings map[string]any
	if supplier.Settings != nil {
		_ = json.Unmarshal(supplier.Settings, &settings)
	}
	if step, _ := settings["setup_step"].(string); step != "api_keys" {
		return nil, NewValidationError(fmt.Errorf("cannot set API keys: current step is %q, expected api_keys", step))
	}

	// Validate credentials via BTP health check. The SDK client MUST use the
	// SSRF-guarded transport (netutil.SafeHTTPClient) so a user-controlled base_url
	// cannot make the server reach internal/metadata endpoints — matching every other
	// btpsdk.NewClient call site in the codebase.
	opts := []btpsdk.Option{btpsdk.WithHTTPClient(netutil.SafeHTTPClient(30 * time.Second))}
	if req.BaseURL != "" {
		opts = append(opts, btpsdk.WithBaseURL(req.BaseURL))
	}
	client := btpsdk.NewClient(req.PublicKey, req.PrivateKey, opts...)
	if err := client.HealthCheck(ctx); err != nil {
		return nil, NewValidationError(fmt.Errorf("BTP API authentication failed: %w", err))
	}

	// Create or update BTP integration with encrypted credentials
	credJSON, _ := json.Marshal(map[string]string{
		"username": req.PublicKey,
		"password": req.PrivateKey,
		"base_url": req.BaseURL,
	})

	var integ *model.Integration
	credRaw := json.RawMessage(credJSON)

	if supplier.IntegrationID != nil {
		// Supplier already linked to integration — update credentials
		var integErr error
		integ, integErr = s.integrationSvc.Update(ctx, tenantID, *supplier.IntegrationID, model.UpdateIntegrationRequest{
			Credentials: &credRaw,
		}, actorID, ip)
		if integErr != nil {
			return nil, fmt.Errorf("update btp integration: %w", integErr)
		}
	} else {
		// Try to find existing orphaned BTP integration (e.g. from a previously deleted supplier)
		_, existing, _ := s.integrationSvc.GetDecryptedCredentialsByProvider(ctx, tenantID, "btp")
		if existing != nil {
			// Reuse existing BTP integration — update its credentials
			var integErr error
			integ, integErr = s.integrationSvc.Update(ctx, tenantID, existing.ID, model.UpdateIntegrationRequest{
				Credentials: &credRaw,
			}, actorID, ip)
			if integErr != nil {
				return nil, fmt.Errorf("update orphaned btp integration: %w", integErr)
			}
		} else {
			// No existing BTP integration — create new
			var integErr error
			integ, integErr = s.integrationSvc.Create(ctx, tenantID, model.CreateIntegrationRequest{
				Provider:    "btp",
				Credentials: credJSON,
			}, actorID, ip)
			if integErr != nil {
				return nil, fmt.Errorf("create btp integration: %w", integErr)
			}
		}
	}

	// Link integration to supplier and advance step
	if err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		if err := s.supplierRepo.Update(ctx, tx, supplierID, model.UpdateSupplierRequest{
			IntegrationID: &integ.ID,
		}); err != nil {
			return err
		}
		if err := s.supplierRepo.UpdateSettingsKeys(ctx, tx, supplierID, map[string]any{
			"setup_step": "sync_settings",
		}); err != nil {
			return err
		}
		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "supplier.api_keys_set",
			EntityType: "supplier",
			EntityID:   supplierID,
			IPAddress:  ip,
		})
	}); err != nil {
		return nil, err
	}

	// Return updated supplier
	return s.Get(ctx, tenantID, supplierID)
}

// BTPWizardCompleteSyncSettings finalizes the BTP wizard by setting sync intervals (step 3).
func (s *SupplierService) BTPWizardCompleteSyncSettings(ctx context.Context, tenantID, supplierID uuid.UUID, req model.BTPWizardSyncSettingsRequest, actorID uuid.UUID, ip string) (*model.Supplier, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	// Verify supplier is on the right step
	var supplier *model.Supplier
	if err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var err error
		supplier, err = s.supplierRepo.FindByID(ctx, tx, supplierID)
		return err
	}); err != nil {
		return nil, err
	}
	if supplier == nil {
		return nil, ErrSupplierNotFound
	}

	var settings map[string]any
	if supplier.Settings != nil {
		_ = json.Unmarshal(supplier.Settings, &settings)
	}
	if step, _ := settings["setup_step"].(string); step != "sync_settings" {
		return nil, NewValidationError(fmt.Errorf("cannot set sync settings: current step is %q, expected sync_settings", step))
	}

	// Activate supplier with configured intervals
	active := "active"
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		if err := s.supplierRepo.Update(ctx, tx, supplierID, model.UpdateSupplierRequest{
			Status:              &active,
			SyncIntervalMinutes: &req.APISyncIntervalMin,
		}); err != nil {
			return err
		}
		if err := s.supplierRepo.UpdateSettingsKeys(ctx, tx, supplierID, map[string]any{
			"setup_step":              "complete",
			"xml_sync_interval_hours": req.XMLSyncIntervalHours,
		}); err != nil {
			return err
		}
		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "supplier.setup_completed",
			EntityType: "supplier",
			EntityID:   supplierID,
			IPAddress:  ip,
		})
	})
	if err != nil {
		return nil, err
	}

	// Trigger immediate API inventory sync to get real stock data
	asyncutil.SafeGo(func() {
		updated, _ := s.Get(context.Background(), tenantID, supplierID)
		if updated != nil {
			_ = s.syncXMLInventory(context.Background(), tenantID, supplierID, updated)
		}
	})

	return s.Get(ctx, tenantID, supplierID)
}

// removeStaleProducts deletes supplier products not present in the current XML feed
// and sets stock to 0 on their linked OMS products.
func (s *SupplierService) removeStaleProducts(ctx context.Context, tenantID, supplierID uuid.UUID, currentExternalIDs []string) {
	if len(currentExternalIDs) == 0 {
		return
	}

	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		linkedProductIDs, err := s.supplierProdRepo.DeleteStaleByExternalIDs(ctx, tx, supplierID, currentExternalIDs)
		if err != nil {
			return err
		}

		// Set stock to 0 on linked OMS products that lost their supplier product
		for _, pid := range linkedProductIDs {
			if _, err := tx.Exec(ctx,
				"UPDATE products SET stock_quantity = 0, updated_at = NOW() WHERE id = $1",
				pid,
			); err != nil {
				s.logger.Error("failed to zero stock on delinked product",
					"product_id", pid, "error", err)
			}
		}

		if len(linkedProductIDs) > 0 {
			s.logger.Info("removed stale supplier products",
				"supplier_id", supplierID, "removed_linked", len(linkedProductIDs))
		}
		return nil
	})
	if err != nil {
		s.logger.Error("failed to remove stale products",
			"supplier_id", supplierID, "error", err)
		return
	}

}
