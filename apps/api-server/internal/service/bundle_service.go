package service

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

var (
	// ErrBundleComponentNotFound is returned when a bundle component does not exist.
	ErrBundleComponentNotFound = errors.New("bundle component not found")
	// ErrProductNotBundle is returned when a product is not configured as a bundle.
	ErrProductNotBundle = errors.New("product is not a bundle")
)

// BundleService handles business logic for product bundles.
type BundleService struct {
	bundleRepo  *repository.BundleRepository
	productRepo repository.ProductRepo
	auditRepo   repository.AuditRepo
	pool        *pgxpool.Pool
}

// NewBundleService creates a new BundleService.
func NewBundleService(
	bundleRepo *repository.BundleRepository,
	productRepo repository.ProductRepo,
	auditRepo repository.AuditRepo,
	pool *pgxpool.Pool,
) *BundleService {
	return &BundleService{
		bundleRepo:  bundleRepo,
		productRepo: productRepo,
		auditRepo:   auditRepo,
		pool:        pool,
	}
}

// ListComponents returns all components of a bundle product.
func (s *BundleService) ListComponents(ctx context.Context, tenantID, bundleProductID uuid.UUID) ([]model.ProductBundle, error) {
	var components []model.ProductBundle
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var err error
		components, err = s.bundleRepo.ListByBundleProduct(ctx, tx, bundleProductID)
		if err != nil {
			return err
		}
		return s.populateComponentStock(ctx, tx, components)
	})
	if components == nil {
		components = []model.ProductBundle{}
	}
	return components, err
}

// AddComponent adds a product as a component of a bundle.
func (s *BundleService) AddComponent(ctx context.Context, tenantID uuid.UUID, bundleProductID uuid.UUID, req model.CreateBundleComponentRequest, actorID uuid.UUID, ip string) (*model.ProductBundle, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	bundle := &model.ProductBundle{
		ID:                 uuid.New(),
		TenantID:           tenantID,
		BundleProductID:    bundleProductID,
		ComponentProductID: req.ComponentProductID,
		ComponentVariantID: req.ComponentVariantID,
		Quantity:           req.Quantity,
		Position:           req.Position,
	}

	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		// Verify bundle product exists and is a bundle
		product, err := s.productRepo.FindByID(ctx, tx, bundleProductID)
		if err != nil {
			return err
		}
		if product == nil {
			return NewValidationError(errors.New("bundle product not found"))
		}
		if !product.IsBundle {
			return NewValidationError(ErrProductNotBundle)
		}

		// Verify component product exists
		component, err := s.productRepo.FindByID(ctx, tx, req.ComponentProductID)
		if err != nil {
			return err
		}
		if component == nil {
			return NewValidationError(errors.New("component product not found"))
		}

		// Prevent self-reference
		if bundleProductID == req.ComponentProductID {
			return NewValidationError(errors.New("a product cannot be a component of itself"))
		}

		if err := s.bundleRepo.Create(ctx, tx, bundle); err != nil {
			if isDuplicateKeyError(err) {
				return NewValidationError(errors.New("this component is already in the bundle"))
			}
			return err
		}

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "bundle.component_added",
			EntityType: "product_bundle",
			EntityID:   bundle.ID,
			Changes:    map[string]string{"bundle_product": bundleProductID.String(), "component_product": req.ComponentProductID.String()},
			IPAddress:  ip,
		})
	})
	if err != nil {
		return nil, err
	}
	return bundle, nil
}

// UpdateComponent modifies the quantity or details of a bundle component.
func (s *BundleService) UpdateComponent(ctx context.Context, tenantID uuid.UUID, componentID uuid.UUID, req model.UpdateBundleComponentRequest, actorID uuid.UUID, ip string) (*model.ProductBundle, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	var result *model.ProductBundle
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		existing, err := s.bundleRepo.FindByID(ctx, tx, componentID)
		if err != nil {
			return err
		}
		if existing == nil {
			return ErrBundleComponentNotFound
		}

		if err := s.bundleRepo.Update(ctx, tx, componentID, req); err != nil {
			return err
		}

		result, err = s.bundleRepo.FindByID(ctx, tx, componentID)
		if err != nil {
			return err
		}
		if result != nil {
			hydrated := []model.ProductBundle{*result}
			if err := s.populateComponentStock(ctx, tx, hydrated); err != nil {
				return err
			}
			*result = hydrated[0]
		}

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "bundle.component_updated",
			EntityType: "product_bundle",
			EntityID:   componentID,
			IPAddress:  ip,
		})
	})
	return result, err
}

// RemoveComponent removes a component from a bundle.
func (s *BundleService) RemoveComponent(ctx context.Context, tenantID uuid.UUID, componentID uuid.UUID, actorID uuid.UUID, ip string) error {
	return database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		existing, err := s.bundleRepo.FindByID(ctx, tx, componentID)
		if err != nil {
			return err
		}
		if existing == nil {
			return ErrBundleComponentNotFound
		}

		if err := s.bundleRepo.Delete(ctx, tx, componentID); err != nil {
			return err
		}

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "bundle.component_removed",
			EntityType: "product_bundle",
			EntityID:   componentID,
			IPAddress:  ip,
		})
	})
}

// CalculateBundleStock returns the maximum number of bundles that can be assembled
// based on component stock: min(component_stock / component_qty) for all components.
func (s *BundleService) CalculateBundleStock(ctx context.Context, tenantID, bundleProductID uuid.UUID) (int, error) {
	var stock int
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		product, err := s.productRepo.FindByID(ctx, tx, bundleProductID)
		if err != nil {
			return err
		}
		if product == nil {
			return ErrProductNotBundle
		}
		if !product.IsBundle {
			return ErrProductNotBundle
		}

		components, err := s.bundleRepo.ListByBundleProduct(ctx, tx, bundleProductID)
		if err != nil {
			return err
		}
		if err := s.populateComponentStock(ctx, tx, components); err != nil {
			return err
		}
		stock = assembleableBundles(components)
		return nil
	})
	return stock, err
}

func (s *BundleService) populateComponentStock(ctx context.Context, tx pgx.Tx, components []model.ProductBundle) error {
	if len(components) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, len(components))
	for i := range components {
		ids[i] = components[i].ComponentProductID
	}
	avail, err := s.productRepo.AvailableStockBatch(ctx, tx, ids)
	if err != nil {
		return err
	}
	for i := range components {
		components[i].ComponentStock = avail[components[i].ComponentProductID]
	}
	return nil
}

func assembleableBundles(components []model.ProductBundle) int {
	if len(components) == 0 {
		return 0
	}
	minStock := math.MaxInt32
	for _, c := range components {
		if c.Quantity <= 0 {
			continue
		}
		available := c.ComponentStock / c.Quantity
		if available < minStock {
			minStock = available
		}
	}
	if minStock == math.MaxInt32 {
		return 0
	}
	return minStock
}

// ExpandBundleComponents resolves bundle product IDs in productQtys into their component
// product quantities. It returns the accumulated component quantities (componentProductID
// -> Σ component_qty * order_qty) and the set of input product IDs that are bundles, so
// callers can exclude those virtual bundle IDs from direct stock adjustment (a bundle holds
// no warehouse_stock row of its own — only its components draw down stock). Runs inside the
// caller's tenant-scoped transaction so it shares the RLS context.
func (s *BundleService) ExpandBundleComponents(ctx context.Context, tx pgx.Tx, productQtys map[uuid.UUID]int) (map[uuid.UUID]int, map[uuid.UUID]bool, error) {
	componentQtys := make(map[uuid.UUID]int)
	bundleIDs := make(map[uuid.UUID]bool)
	for productID, orderQty := range productQtys {
		product, err := s.productRepo.FindByID(ctx, tx, productID)
		if err != nil {
			return nil, nil, fmt.Errorf("load product %s: %w", productID, err)
		}
		if product == nil || !product.IsBundle {
			continue
		}
		bundleIDs[productID] = true
		components, err := s.bundleRepo.ListByBundleProduct(ctx, tx, productID)
		if err != nil {
			return nil, nil, fmt.Errorf("list bundle components for %s: %w", productID, err)
		}
		for _, c := range components {
			componentQtys[c.ComponentProductID] += c.Quantity * orderQty
		}
	}
	return componentQtys, bundleIDs, nil
}
