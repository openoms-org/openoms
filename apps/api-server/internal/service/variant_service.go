package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

var (
	ErrVariantNotFound = errors.New("variant not found")
)

// VariantService provides business logic for product variants.
type VariantService struct {
	variantRepo repository.VariantRepo
	productRepo repository.ProductRepo
	auditRepo   repository.AuditRepo
	pool        *pgxpool.Pool
}

// NewVariantService creates a new VariantService.
func NewVariantService(
	variantRepo repository.VariantRepo,
	productRepo repository.ProductRepo,
	auditRepo repository.AuditRepo,
	pool *pgxpool.Pool,
) *VariantService {
	return &VariantService{
		variantRepo: variantRepo,
		productRepo: productRepo,
		auditRepo:   auditRepo,
		pool:        pool,
	}
}

// List lists all variants for a product within a tenant.
func (s *VariantService) List(ctx context.Context, tenantID uuid.UUID, filter model.VariantListFilter) ([]model.ProductVariant, int, error) {
	var variants []model.ProductVariant
	var total int
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var err error
		variants, total, err = s.variantRepo.List(ctx, tx, filter)
		return err
	})
	return variants, total, err
}

// Get retrieves a single variant by ID within a tenant.
func (s *VariantService) Get(ctx context.Context, tenantID, variantID uuid.UUID) (*model.ProductVariant, error) {
	var variant *model.ProductVariant
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var err error
		variant, err = s.variantRepo.FindByID(ctx, tx, variantID)
		return err
	})
	if err != nil {
		return nil, err
	}
	if variant == nil {
		return nil, ErrVariantNotFound
	}
	return variant, nil
}

// Create creates a new variant for a product within a tenant.
func (s *VariantService) Create(ctx context.Context, tenantID, productID uuid.UUID, req model.CreateVariantRequest, actorID uuid.UUID, ip string) (*model.ProductVariant, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	// Sanitize user-facing text
	req.Name = model.StripHTMLTags(req.Name)

	attributes := req.Attributes
	if attributes == nil {
		attributes = []byte("{}")
	}

	active := true
	if req.Active != nil {
		active = *req.Active
	}

	position := 0
	if req.Position != nil {
		position = *req.Position
	}

	variant := &model.ProductVariant{
		ID:            uuid.New(),
		TenantID:      tenantID,
		ProductID:     productID,
		SKU:           req.SKU,
		EAN:           req.EAN,
		Name:          req.Name,
		Attributes:    attributes,
		PriceOverride: req.PriceOverride,
		StockQuantity: req.StockQuantity,
		Weight:        req.Weight,
		ImageURL:      req.ImageURL,
		Position:      position,
		Active:        active,
	}

	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		// Verify product exists
		product, err := s.productRepo.FindByID(ctx, tx, productID)
		if err != nil {
			return err
		}
		if product == nil {
			return ErrProductNotFound
		}

		if err := s.variantRepo.Create(ctx, tx, variant); err != nil {
			return err
		}

		// Set has_variants = true on the parent product
		if _, err := tx.Exec(ctx, "UPDATE products SET has_variants = true WHERE id = $1", productID); err != nil {
			return err
		}

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "variant.created",
			EntityType: "product_variant",
			EntityID:   variant.ID,
			Changes:    map[string]string{"name": req.Name, "product_id": productID.String()},
			IPAddress:  ip,
		})
	})
	if err != nil {
		return nil, err
	}
	return variant, nil
}

// Update updates an existing variant within a tenant.
func (s *VariantService) Update(ctx context.Context, tenantID, variantID uuid.UUID, req model.UpdateVariantRequest, actorID uuid.UUID, ip string) (*model.ProductVariant, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	var variant *model.ProductVariant
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var err error
		variant, err = s.variantRepo.FindByID(ctx, tx, variantID)
		if err != nil {
			return err
		}
		if variant == nil {
			return ErrVariantNotFound
		}

		if err := s.variantRepo.Update(ctx, tx, variantID, req); err != nil {
			return err
		}

		// Re-fetch to get updated fields
		variant, err = s.variantRepo.FindByID(ctx, tx, variantID)
		if err != nil {
			return err
		}

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "variant.updated",
			EntityType: "product_variant",
			EntityID:   variantID,
			IPAddress:  ip,
		})
	})
	if err != nil {
		return nil, err
	}
	return variant, nil
}

// Delete removes a variant and updates the parent product's has_variants flag if needed.
func (s *VariantService) Delete(ctx context.Context, tenantID, variantID uuid.UUID, actorID uuid.UUID, ip string) error {
	return database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		variant, err := s.variantRepo.FindByID(ctx, tx, variantID)
		if err != nil {
			return err
		}
		if variant == nil {
			return ErrVariantNotFound
		}

		if err := s.variantRepo.Delete(ctx, tx, variantID); err != nil {
			return err
		}

		// Check remaining variant count for the product
		remaining, err := s.variantRepo.CountByProductID(ctx, tx, variant.ProductID)
		if err != nil {
			return err
		}
		if remaining == 0 {
			if _, err := tx.Exec(ctx, "UPDATE products SET has_variants = false WHERE id = $1", variant.ProductID); err != nil {
				return err
			}
		}

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "variant.deleted",
			EntityType: "product_variant",
			EntityID:   variantID,
			Changes:    map[string]string{"name": variant.Name, "product_id": variant.ProductID.String()},
			IPAddress:  ip,
		})
	})
}
