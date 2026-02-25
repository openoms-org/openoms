package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

var (
	ErrCategoryNotFound   = errors.New("category not found")
	ErrCategoryDepthLimit = errors.New("maximum category depth exceeded")
	ErrCircularReference  = errors.New("circular parent reference detected")
	ErrDuplicateSlug      = errors.New("duplicate category slug")
)

type ProductCategoryService struct {
	categoryRepo           repository.ProductCategoryRepo
	marketplaceMappingRepo repository.MarketplaceCategoryMappingRepo
	auditRepo              repository.AuditRepo
	pool                   *pgxpool.Pool
	logger                 *slog.Logger
}

func NewProductCategoryService(
	categoryRepo repository.ProductCategoryRepo,
	marketplaceMappingRepo repository.MarketplaceCategoryMappingRepo,
	auditRepo repository.AuditRepo,
	pool *pgxpool.Pool,
) *ProductCategoryService {
	return &ProductCategoryService{
		categoryRepo:           categoryRepo,
		marketplaceMappingRepo: marketplaceMappingRepo,
		auditRepo:              auditRepo,
		pool:                   pool,
		logger:                 slog.Default().With("component", "product_category_service"),
	}
}

func (s *ProductCategoryService) List(ctx context.Context, tenantID uuid.UUID, filter model.CategoryListFilter) ([]model.ProductCategory, error) {
	var categories []model.ProductCategory
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var err error
		categories, err = s.categoryRepo.List(ctx, tx, filter)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}

	if filter.IncludeTree {
		categories = model.BuildCategoryTree(categories)
	}

	return categories, nil
}

func (s *ProductCategoryService) Get(ctx context.Context, tenantID, id uuid.UUID) (*model.ProductCategory, error) {
	var category *model.ProductCategory
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var err error
		category, err = s.categoryRepo.FindByID(ctx, tx, id)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("get category: %w", err)
	}
	if category == nil {
		return nil, ErrCategoryNotFound
	}
	return category, nil
}

func (s *ProductCategoryService) Create(ctx context.Context, tenantID, actorID uuid.UUID, req model.CreateCategoryRequest) (*model.ProductCategory, error) {
	category := &model.ProductCategory{
		ID:       uuid.New(),
		TenantID: tenantID,
		ParentID: req.ParentID,
		Name:     req.Name,
		Color:    req.Color,
		Icon:     req.Icon,
	}

	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		// Resolve parent depth
		if req.ParentID != nil {
			parent, err := s.categoryRepo.FindByID(ctx, tx, *req.ParentID)
			if err != nil {
				return err
			}
			if parent == nil {
				return ErrCategoryNotFound
			}
			category.Depth = parent.Depth + 1
			if category.Depth > model.MaxCategoryDepth {
				return ErrCategoryDepthLimit
			}
		}

		// Generate unique slug
		slug := model.GenerateSlug(req.Name)
		if slug == "" {
			slug = category.ID.String()[:8]
		}
		count, err := s.categoryRepo.CountBySlug(ctx, tx, slug)
		if err != nil {
			return err
		}
		if count > 0 {
			slug = fmt.Sprintf("%s-%d", slug, count+1)
		}
		category.Slug = slug

		if err := s.categoryRepo.Create(ctx, tx, category); err != nil {
			return err
		}

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "category.created",
			EntityType: "product_category",
			EntityID:   category.ID,
		})
	})
	if err != nil {
		return nil, fmt.Errorf("create category: %w", err)
	}
	return category, nil
}

func (s *ProductCategoryService) Update(ctx context.Context, tenantID, actorID, id uuid.UUID, req model.UpdateCategoryRequest) (*model.ProductCategory, error) {
	var category *model.ProductCategory
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var err error
		category, err = s.categoryRepo.FindByID(ctx, tx, id)
		if err != nil {
			return err
		}
		if category == nil {
			return ErrCategoryNotFound
		}

		if req.Name != nil {
			category.Name = *req.Name
			slug := model.GenerateSlug(*req.Name)
			if slug == "" {
				slug = category.ID.String()[:8]
			}
			// Check if slug changed and is unique
			if slug != category.Slug {
				existing, err := s.categoryRepo.FindBySlug(ctx, tx, slug)
				if err != nil {
					return err
				}
				if existing != nil && existing.ID != category.ID {
					count, err := s.categoryRepo.CountBySlug(ctx, tx, slug)
					if err != nil {
						return err
					}
					slug = fmt.Sprintf("%s-%d", slug, count+1)
				}
				category.Slug = slug
			}
		}

		if req.ParentID != nil {
			newParentID := *req.ParentID
			if newParentID == uuid.Nil {
				// Moving to root
				category.ParentID = nil
				category.Depth = 0
			} else {
				// Check for circular reference
				if newParentID == category.ID {
					return ErrCircularReference
				}
				descendantIDs, err := s.categoryRepo.GetDescendantIDs(ctx, tx, category.ID)
				if err != nil {
					return err
				}
				if slices.Contains(descendantIDs, newParentID) {
					return ErrCircularReference
				}

				parent, err := s.categoryRepo.FindByID(ctx, tx, newParentID)
				if err != nil {
					return err
				}
				if parent == nil {
					return ErrCategoryNotFound
				}
				newDepth := parent.Depth + 1
				if newDepth > model.MaxCategoryDepth {
					return ErrCategoryDepthLimit
				}
				category.ParentID = &newParentID
				category.Depth = newDepth
			}
		}

		if req.Color != nil {
			category.Color = *req.Color
		}
		if req.Icon != nil {
			category.Icon = req.Icon
		}
		if req.Position != nil {
			category.Position = *req.Position
		}

		if err := s.categoryRepo.Update(ctx, tx, category); err != nil {
			return err
		}

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "category.updated",
			EntityType: "product_category",
			EntityID:   category.ID,
		})
	})
	if err != nil {
		return nil, fmt.Errorf("update category: %w", err)
	}
	return category, nil
}

func (s *ProductCategoryService) Delete(ctx context.Context, tenantID, actorID, id uuid.UUID) error {
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		category, err := s.categoryRepo.FindByID(ctx, tx, id)
		if err != nil {
			return err
		}
		if category == nil {
			return ErrCategoryNotFound
		}

		if err := s.categoryRepo.Delete(ctx, tx, id); err != nil {
			return err
		}

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "category.deleted",
			EntityType: "product_category",
			EntityID:   id,
		})
	})
	if err != nil {
		return fmt.Errorf("delete category: %w", err)
	}
	return nil
}

func (s *ProductCategoryService) GetDescendantIDs(ctx context.Context, tenantID, id uuid.UUID) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var err error
		ids, err = s.categoryRepo.GetDescendantIDs(ctx, tx, id)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("get descendant IDs: %w", err)
	}
	return ids, nil
}

// ResolveMarketplaceCategory resolves an external marketplace category to an internal category.
// Priority: existing mapping → fuzzy match + auto-create mapping → auto-create category → defaultCategoryID → nil.
// This is called within an existing transaction (tx already has tenant context set).
func (s *ProductCategoryService) ResolveMarketplaceCategory(
	ctx context.Context,
	tx pgx.Tx,
	tenantID uuid.UUID,
	integrationID uuid.UUID,
	externalCategoryID string,
	externalCategoryName string,
	autoCreate bool,
	defaultCategoryID *uuid.UUID,
) (*uuid.UUID, error) {
	if externalCategoryID == "" {
		return defaultCategoryID, nil
	}

	// 1. Check existing mapping
	mapping, err := s.marketplaceMappingRepo.FindByExternalID(ctx, tx, integrationID, externalCategoryID)
	if err != nil {
		s.logger.Error("failed to find marketplace category mapping",
			"integration_id", integrationID,
			"external_category_id", externalCategoryID,
			"error", err)
		return defaultCategoryID, nil
	}

	if mapping != nil && mapping.CategoryID != nil {
		return mapping.CategoryID, nil
	}

	// 2. No mapping or mapping without category_id — try fuzzy match by name
	if externalCategoryName != "" && mapping == nil {
		matches, err := s.categoryRepo.FuzzyMatch(ctx, tx, externalCategoryName)
		if err != nil {
			s.logger.Error("failed to fuzzy match category",
				"external_category_name", externalCategoryName, "error", err)
			return defaultCategoryID, nil
		}

		newMapping := &model.MarketplaceCategoryMapping{
			ID:                   uuid.New(),
			TenantID:             tenantID,
			IntegrationID:        integrationID,
			ExternalCategoryID:   externalCategoryID,
			ExternalCategoryName: externalCategoryName,
			AutoCreated:          true,
			Confirmed:            false,
		}

		if len(matches) > 0 {
			newMapping.CategoryID = &matches[0].ID
			if err := s.marketplaceMappingRepo.Upsert(ctx, tx, newMapping); err != nil {
				s.logger.Error("failed to create auto-matched mapping",
					"external_category_id", externalCategoryID, "error", err)
			}
			return &matches[0].ID, nil
		}

		// 3. No fuzzy match — optionally auto-create internal category
		if autoCreate {
			newCat := &model.ProductCategory{
				ID:       uuid.New(),
				TenantID: tenantID,
				Name:     externalCategoryName,
				Slug:     model.GenerateSlug(externalCategoryName),
				Color:    "#6b7280", // default gray
			}
			// Guard against empty slug from non-alphanumeric names
			if newCat.Slug == "" {
				newCat.Slug = newCat.ID.String()[:8]
			}
			// Ensure slug uniqueness
			count, err := s.categoryRepo.CountBySlug(ctx, tx, newCat.Slug)
			if err != nil {
				s.logger.Error("failed to count slugs during auto-create",
					"slug", newCat.Slug, "error", err)
				// continue — DB unique constraint will catch true duplicates
			} else if count > 0 {
				newCat.Slug = fmt.Sprintf("%s-%d", newCat.Slug, count+1)
			}
			if err := s.categoryRepo.Create(ctx, tx, newCat); err != nil {
				s.logger.Error("failed to auto-create category",
					"name", externalCategoryName, "error", err)
				// Still create mapping without category
				if upsertErr := s.marketplaceMappingRepo.Upsert(ctx, tx, newMapping); upsertErr != nil {
					s.logger.Error("failed to create unmapped mapping",
						"external_category_id", externalCategoryID, "error", upsertErr)
				}
				return defaultCategoryID, nil
			}
			newMapping.CategoryID = &newCat.ID
			if err := s.marketplaceMappingRepo.Upsert(ctx, tx, newMapping); err != nil {
				s.logger.Error("failed to create auto-created mapping",
					"external_category_id", externalCategoryID, "error", err)
			}
			return &newCat.ID, nil
		}

		// No autoCreate — create mapping without category
		if err := s.marketplaceMappingRepo.Upsert(ctx, tx, newMapping); err != nil {
			s.logger.Error("failed to create unmapped mapping",
				"external_category_id", externalCategoryID, "error", err)
		}
	}

	// 4. Mapping exists but no category_id, or no name to match — create/update mapping record
	if mapping == nil && externalCategoryName == "" {
		newMapping := &model.MarketplaceCategoryMapping{
			ID:                 uuid.New(),
			TenantID:           tenantID,
			IntegrationID:      integrationID,
			ExternalCategoryID: externalCategoryID,
			AutoCreated:        true,
			Confirmed:          false,
		}
		if err := s.marketplaceMappingRepo.Upsert(ctx, tx, newMapping); err != nil {
			s.logger.Error("failed to create placeholder mapping",
				"external_category_id", externalCategoryID, "error", err)
		}
	}

	return defaultCategoryID, nil
}
