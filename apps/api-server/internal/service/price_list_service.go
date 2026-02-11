package service

import (
	"context"
	"errors"
	"math"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

var (
	ErrPriceListNotFound     = errors.New("price list not found")
	ErrPriceListItemNotFound = errors.New("price list item not found")
)

type PriceListService struct {
	priceListRepo repository.PriceListRepo
	productRepo   repository.ProductRepo
	auditRepo     repository.AuditRepo
	pool          *pgxpool.Pool
}

func NewPriceListService(
	priceListRepo repository.PriceListRepo,
	productRepo repository.ProductRepo,
	auditRepo repository.AuditRepo,
	pool *pgxpool.Pool,
) *PriceListService {
	return &PriceListService{
		priceListRepo: priceListRepo,
		productRepo:   productRepo,
		auditRepo:     auditRepo,
		pool:          pool,
	}
}

func (s *PriceListService) List(ctx context.Context, tenantID uuid.UUID, filter model.PriceListListFilter) ([]model.PriceList, int, error) {
	var priceLists []model.PriceList
	var total int
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var err error
		priceLists, total, err = s.priceListRepo.List(ctx, tx, filter)
		return err
	})
	return priceLists, total, err
}

func (s *PriceListService) Get(ctx context.Context, tenantID, id uuid.UUID) (*model.PriceList, error) {
	var pl *model.PriceList
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var err error
		pl, err = s.priceListRepo.FindByID(ctx, tx, id)
		return err
	})
	if err != nil {
		return nil, err
	}
	if pl == nil {
		return nil, ErrPriceListNotFound
	}
	return pl, nil
}

func (s *PriceListService) Create(ctx context.Context, tenantID uuid.UUID, req model.CreatePriceListRequest, actorID uuid.UUID, ip string) (*model.PriceList, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	active := true
	if req.Active != nil {
		active = *req.Active
	}

	pl := &model.PriceList{
		ID:           uuid.New(),
		TenantID:     tenantID,
		Name:         req.Name,
		Description:  req.Description,
		Currency:     req.Currency,
		IsDefault:    req.IsDefault,
		DiscountType: req.DiscountType,
		Active:       active,
		ValidFrom:    req.ValidFrom,
		ValidTo:      req.ValidTo,
	}

	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		if err := s.priceListRepo.Create(ctx, tx, pl); err != nil {
			return err
		}
		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "price_list.created",
			EntityType: "price_list",
			EntityID:   pl.ID,
			Changes:    map[string]string{"name": req.Name},
			IPAddress:  ip,
		})
	})
	if err != nil {
		return nil, err
	}
	return pl, nil
}

func (s *PriceListService) Update(ctx context.Context, tenantID, id uuid.UUID, req model.UpdatePriceListRequest, actorID uuid.UUID, ip string) (*model.PriceList, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	var pl *model.PriceList
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var err error
		pl, err = s.priceListRepo.FindByID(ctx, tx, id)
		if err != nil {
			return err
		}
		if pl == nil {
			return ErrPriceListNotFound
		}

		if err := s.priceListRepo.Update(ctx, tx, id, req); err != nil {
			return err
		}

		pl, err = s.priceListRepo.FindByID(ctx, tx, id)
		if err != nil {
			return err
		}

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "price_list.updated",
			EntityType: "price_list",
			EntityID:   id,
			IPAddress:  ip,
		})
	})
	if err != nil {
		return nil, err
	}
	return pl, nil
}

func (s *PriceListService) Delete(ctx context.Context, tenantID, id uuid.UUID, actorID uuid.UUID, ip string) error {
	return database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		pl, err := s.priceListRepo.FindByID(ctx, tx, id)
		if err != nil {
			return err
		}
		if pl == nil {
			return ErrPriceListNotFound
		}

		if err := s.priceListRepo.Delete(ctx, tx, id); err != nil {
			return err
		}

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "price_list.deleted",
			EntityType: "price_list",
			EntityID:   id,
			Changes:    map[string]string{"name": pl.Name},
			IPAddress:  ip,
		})
	})
}

// --- Price List Items ---

func (s *PriceListService) ListItems(ctx context.Context, tenantID, priceListID uuid.UUID, limit, offset int) ([]model.PriceListItem, int, error) {
	var items []model.PriceListItem
	var total int
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		pl, err := s.priceListRepo.FindByID(ctx, tx, priceListID)
		if err != nil {
			return err
		}
		if pl == nil {
			return ErrPriceListNotFound
		}
		items, total, err = s.priceListRepo.ListItems(ctx, tx, priceListID, limit, offset)
		return err
	})
	return items, total, err
}

func (s *PriceListService) CreateItem(ctx context.Context, tenantID, priceListID uuid.UUID, req model.CreatePriceListItemRequest, actorID uuid.UUID, ip string) (*model.PriceListItem, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	item := &model.PriceListItem{
		ID:          uuid.New(),
		TenantID:    tenantID,
		PriceListID: priceListID,
		ProductID:   req.ProductID,
		VariantID:   req.VariantID,
		Price:       req.Price,
		Discount:    req.Discount,
		MinQuantity: req.MinQuantity,
	}

	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		pl, err := s.priceListRepo.FindByID(ctx, tx, priceListID)
		if err != nil {
			return err
		}
		if pl == nil {
			return ErrPriceListNotFound
		}

		if err := s.priceListRepo.CreateItem(ctx, tx, item); err != nil {
			return err
		}

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "price_list_item.created",
			EntityType: "price_list_item",
			EntityID:   item.ID,
			Changes:    map[string]string{"price_list_id": priceListID.String(), "product_id": req.ProductID.String()},
			IPAddress:  ip,
		})
	})
	if err != nil {
		if isDuplicateKeyError(err) {
			return nil, NewValidationError(errors.New("this product/variant/quantity combination already exists in this price list"))
		}
		return nil, err
	}
	return item, nil
}

func (s *PriceListService) DeleteItem(ctx context.Context, tenantID, itemID uuid.UUID, actorID uuid.UUID, ip string) error {
	return database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		if err := s.priceListRepo.DeleteItem(ctx, tx, itemID); err != nil {
			return err
		}
		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "price_list_item.deleted",
			EntityType: "price_list_item",
			EntityID:   itemID,
			IPAddress:  ip,
		})
	})
}

// CalculatePrice calculates the effective price for a product/variant given a price list and quantity.
func (s *PriceListService) CalculatePrice(ctx context.Context, tenantID uuid.UUID, productID uuid.UUID, variantID *uuid.UUID, quantity int, priceListID uuid.UUID) (*model.CalculatePriceResponse, error) {
	var resp model.CalculatePriceResponse

	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		// Get the product's base price
		product, err := s.productRepo.FindByID(ctx, tx, productID)
		if err != nil {
			return err
		}
		if product == nil {
			return ErrProductNotFound
		}
		resp.OriginalPrice = product.Price

		// Get the price list
		pl, err := s.priceListRepo.FindByID(ctx, tx, priceListID)
		if err != nil {
			return err
		}
		if pl == nil {
			return ErrPriceListNotFound
		}

		// Find applicable price list items
		items, err := s.priceListRepo.FindItemsByProduct(ctx, tx, priceListID, productID, variantID, quantity)
		if err != nil {
			return err
		}

		if len(items) == 0 {
			// No price list item found — return original price
			resp.EffectivePrice = resp.OriginalPrice
			resp.DiscountType = "none"
			resp.DiscountValue = 0
			return nil
		}

		// Use the first match (highest min_quantity that is <= requested quantity)
		item := items[0]
		resp.DiscountType = pl.DiscountType

		switch pl.DiscountType {
		case "override":
			if item.Price != nil {
				resp.EffectivePrice = *item.Price
				resp.DiscountValue = resp.OriginalPrice - *item.Price
			} else {
				resp.EffectivePrice = resp.OriginalPrice
			}
		case "percentage":
			if item.Discount != nil {
				discount := resp.OriginalPrice * (*item.Discount / 100)
				resp.EffectivePrice = math.Round((resp.OriginalPrice-discount)*100) / 100
				resp.DiscountValue = *item.Discount
			} else {
				resp.EffectivePrice = resp.OriginalPrice
			}
		case "fixed":
			if item.Discount != nil {
				resp.EffectivePrice = math.Round((resp.OriginalPrice-*item.Discount)*100) / 100
				resp.DiscountValue = *item.Discount
			} else {
				resp.EffectivePrice = resp.OriginalPrice
			}
		default:
			resp.EffectivePrice = resp.OriginalPrice
		}

		if resp.EffectivePrice < 0 {
			resp.EffectivePrice = 0
		}

		return nil
	})

	if err != nil {
		return nil, err
	}
	return &resp, nil
}
