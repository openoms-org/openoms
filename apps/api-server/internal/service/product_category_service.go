package service

import (
	"context"
	"errors"
	"fmt"
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
	categoryRepo repository.ProductCategoryRepo
	auditRepo    repository.AuditRepo
	pool         *pgxpool.Pool
}

func NewProductCategoryService(
	categoryRepo repository.ProductCategoryRepo,
	auditRepo repository.AuditRepo,
	pool *pgxpool.Pool,
) *ProductCategoryService {
	return &ProductCategoryService{
		categoryRepo: categoryRepo,
		auditRepo:    auditRepo,
		pool:         pool,
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
