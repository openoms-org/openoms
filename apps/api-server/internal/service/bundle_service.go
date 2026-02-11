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
	ErrBundleComponentNotFound = errors.New("bundle component not found")
	ErrProductNotBundle        = errors.New("product is not a bundle")
)

type BundleService struct {
	bundleRepo  *repository.BundleRepository
	productRepo repository.ProductRepo
	auditRepo   repository.AuditRepo
	pool        *pgxpool.Pool
}

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

func (s *BundleService) ListComponents(ctx context.Context, tenantID, bundleProductID uuid.UUID) ([]model.ProductBundle, error) {
	var components []model.ProductBundle
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var err error
		components, err = s.bundleRepo.ListByBundleProduct(ctx, tx, bundleProductID)
		return err
	})
	if components == nil {
		components = []model.ProductBundle{}
	}
	return components, err
}

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
		components, err := s.bundleRepo.ListByBundleProduct(ctx, tx, bundleProductID)
		if err != nil {
			return err
		}
		if len(components) == 0 {
			stock = 0
			return nil
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
			minStock = 0
		}
		stock = minStock
		return nil
	})
	return stock, err
}

// DecrementComponentStock decrements stock for all components of a bundle.
// Called when an order with bundle products is confirmed.
func (s *BundleService) DecrementComponentStock(ctx context.Context, tx pgx.Tx, bundleProductID uuid.UUID, quantity int) error {
	components, err := s.bundleRepo.ListByBundleProduct(ctx, tx, bundleProductID)
	if err != nil {
		return fmt.Errorf("list bundle components: %w", err)
	}

	for _, c := range components {
		decrementQty := c.Quantity * quantity
		_, err := tx.Exec(ctx,
			"UPDATE products SET stock_quantity = stock_quantity - $1, updated_at = NOW() WHERE id = $2 AND stock_quantity >= $1",
			decrementQty, c.ComponentProductID,
		)
		if err != nil {
			return fmt.Errorf("decrement stock for component %s: %w", c.ComponentProductID, err)
		}
	}
	return nil
}
