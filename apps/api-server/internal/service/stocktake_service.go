package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

var (
	ErrStocktakeNotFound     = errors.New("stocktake not found")
	ErrStocktakeNotDraft     = errors.New("stocktake is not in draft status")
	ErrStocktakeNotActive    = errors.New("stocktake is not in progress")
	ErrStocktakeItemNotFound = errors.New("stocktake item not found")
	ErrNotAllItemsCounted    = errors.New("not all items have been counted")
)

// StocktakeService provides business logic for stocktaking.
type StocktakeService struct {
	stocktakeRepo   repository.StocktakeRepo
	itemRepo        repository.StocktakeItemRepo
	stockRepo       repository.WarehouseStockRepo
	docRepo         repository.WarehouseDocumentRepo
	docItemRepo     repository.WarehouseDocItemRepo
	auditRepo       repository.AuditRepo
	pool            *pgxpool.Pool
	webhookDispatch *WebhookDispatchService
}

// NewStocktakeService creates a new StocktakeService.
func NewStocktakeService(
	stocktakeRepo repository.StocktakeRepo,
	itemRepo repository.StocktakeItemRepo,
	stockRepo repository.WarehouseStockRepo,
	docRepo repository.WarehouseDocumentRepo,
	docItemRepo repository.WarehouseDocItemRepo,
	auditRepo repository.AuditRepo,
	pool *pgxpool.Pool,
	webhookDispatch *WebhookDispatchService,
) *StocktakeService {
	return &StocktakeService{
		stocktakeRepo:   stocktakeRepo,
		itemRepo:        itemRepo,
		stockRepo:       stockRepo,
		docRepo:         docRepo,
		docItemRepo:     docItemRepo,
		auditRepo:       auditRepo,
		pool:            pool,
		webhookDispatch: webhookDispatch,
	}
}

// CreateStocktake creates a new stocktake and snapshots current stock levels.
func (s *StocktakeService) CreateStocktake(ctx context.Context, tenantID uuid.UUID, req model.CreateStocktakeRequest, actorID uuid.UUID, ip string) (*model.Stocktake, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	var stocktake *model.Stocktake
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		stocktake = &model.Stocktake{
			ID:          uuid.New(),
			TenantID:    tenantID,
			WarehouseID: req.WarehouseID,
			Name:        req.Name,
			Status:      "draft",
			Notes:       req.Notes,
			CreatedBy:   &actorID,
		}

		if err := s.stocktakeRepo.Create(ctx, tx, stocktake); err != nil {
			return err
		}

		// Get current stock for this warehouse
		var stockItems []model.StocktakeItem

		if len(req.ProductIDs) > 0 {
			// Specific products requested
			for _, productID := range req.ProductIDs {
				// Get stock level for this product in this warehouse
				qty := s.getStockQuantity(ctx, tx, req.WarehouseID, productID)
				stockItems = append(stockItems, model.StocktakeItem{
					ID:               uuid.New(),
					TenantID:         tenantID,
					StocktakeID:      stocktake.ID,
					ProductID:        productID,
					ExpectedQuantity: qty,
				})
			}
		} else {
			// All products in warehouse
			stocks, _, err := s.stockRepo.ListByWarehouse(ctx, tx, req.WarehouseID, model.WarehouseStockListFilter{
				PaginationParams: model.PaginationParams{Limit: 10000, Offset: 0},
			})
			if err != nil {
				return fmt.Errorf("list warehouse stock: %w", err)
			}

			for _, stock := range stocks {
				stockItems = append(stockItems, model.StocktakeItem{
					ID:               uuid.New(),
					TenantID:         tenantID,
					StocktakeID:      stocktake.ID,
					ProductID:        stock.ProductID,
					ExpectedQuantity: stock.Quantity,
				})
			}
		}

		if len(stockItems) > 0 {
			if err := s.itemRepo.CreateBulk(ctx, tx, stockItems); err != nil {
				return err
			}
		}

		// Get stats
		stats, err := s.itemRepo.GetStats(ctx, tx, stocktake.ID)
		if err != nil {
			return err
		}
		stocktake.Stats = stats

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "stocktake.created",
			EntityType: "stocktake",
			EntityID:   stocktake.ID,
			Changes:    map[string]string{"name": req.Name, "items": fmt.Sprintf("%d", stats.TotalItems)},
			IPAddress:  ip,
		})
	})
	if err != nil {
		return nil, err
	}
	return stocktake, nil
}

func (s *StocktakeService) getStockQuantity(ctx context.Context, tx pgx.Tx, warehouseID, productID uuid.UUID) int {
	var qty int
	err := tx.QueryRow(ctx,
		`SELECT COALESCE(quantity, 0) FROM warehouse_stock
		 WHERE warehouse_id = $1 AND product_id = $2 AND variant_id IS NULL`,
		warehouseID, productID,
	).Scan(&qty)
	if err != nil {
		return 0
	}
	return qty
}

// StartStocktake sets the stocktake status to in_progress.
func (s *StocktakeService) StartStocktake(ctx context.Context, tenantID, stocktakeID uuid.UUID, actorID uuid.UUID, ip string) (*model.Stocktake, error) {
	var stocktake *model.Stocktake
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		existing, err := s.stocktakeRepo.FindByID(ctx, tx, stocktakeID)
		if err != nil {
			return err
		}
		if existing == nil {
			return ErrStocktakeNotFound
		}
		if existing.Status != "draft" {
			return ErrStocktakeNotDraft
		}

		if err := s.stocktakeRepo.SetStartedAt(ctx, tx, stocktakeID); err != nil {
			return err
		}

		stocktake, err = s.stocktakeRepo.FindByID(ctx, tx, stocktakeID)
		if err != nil {
			return err
		}

		stats, err := s.itemRepo.GetStats(ctx, tx, stocktakeID)
		if err != nil {
			return err
		}
		stocktake.Stats = stats

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "stocktake.started",
			EntityType: "stocktake",
			EntityID:   stocktakeID,
			Changes:    map[string]string{"name": existing.Name},
			IPAddress:  ip,
		})
	})
	if err != nil {
		return nil, err
	}
	return stocktake, nil
}

// RecordCount updates the counted quantity for a stocktake item.
func (s *StocktakeService) RecordCount(ctx context.Context, tenantID, stocktakeID, itemID uuid.UUID, req model.UpdateStocktakeItemRequest, actorID uuid.UUID, ip string) (*model.StocktakeItem, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	var item *model.StocktakeItem
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		// Validate stocktake is in_progress
		stocktake, err := s.stocktakeRepo.FindByID(ctx, tx, stocktakeID)
		if err != nil {
			return err
		}
		if stocktake == nil {
			return ErrStocktakeNotFound
		}
		if stocktake.Status != "in_progress" {
			return ErrStocktakeNotActive
		}

		// Validate item belongs to stocktake
		existing, err := s.itemRepo.FindByID(ctx, tx, itemID)
		if err != nil {
			return err
		}
		if existing == nil || existing.StocktakeID != stocktakeID {
			return ErrStocktakeItemNotFound
		}

		if err := s.itemRepo.UpdateCount(ctx, tx, itemID, req.CountedQuantity, req.Notes, actorID); err != nil {
			return err
		}

		item, err = s.itemRepo.FindByID(ctx, tx, itemID)
		if err != nil {
			return err
		}

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "stocktake.item_counted",
			EntityType: "stocktake",
			EntityID:   stocktakeID,
			Changes:    map[string]string{"item_id": itemID.String(), "counted_quantity": fmt.Sprintf("%d", req.CountedQuantity)},
			IPAddress:  ip,
		})
	})
	if err != nil {
		return nil, err
	}
	return item, nil
}

// CompleteStocktake finalizes the stocktake and creates adjustment documents.
func (s *StocktakeService) CompleteStocktake(ctx context.Context, tenantID, stocktakeID uuid.UUID, actorID uuid.UUID, ip string) (*model.Stocktake, error) {
	var stocktake *model.Stocktake
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		existing, err := s.stocktakeRepo.FindByID(ctx, tx, stocktakeID)
		if err != nil {
			return err
		}
		if existing == nil {
			return ErrStocktakeNotFound
		}
		if existing.Status != "in_progress" {
			return ErrStocktakeNotActive
		}

		// Check all items are counted
		stats, err := s.itemRepo.GetStats(ctx, tx, stocktakeID)
		if err != nil {
			return err
		}
		if stats.CountedItems < stats.TotalItems {
			return ErrNotAllItemsCounted
		}

		// Get items with discrepancies
		discrepancies, err := s.itemRepo.ListDiscrepancies(ctx, tx, stocktakeID)
		if err != nil {
			return err
		}

		// Group discrepancies: surplus (positive) and shortage (negative)
		var surplusItems []model.StocktakeItem
		var shortageItems []model.StocktakeItem
		for _, item := range discrepancies {
			if item.Difference > 0 {
				surplusItems = append(surplusItems, item)
			} else if item.Difference < 0 {
				shortageItems = append(shortageItems, item)
			}
		}

		year := time.Now().Year()

		// Create PZ document for surpluses
		if len(surplusItems) > 0 {
			seq, err := s.docRepo.NextDocumentNumber(ctx, tx, "PZ", year)
			if err != nil {
				return fmt.Errorf("next PZ number: %w", err)
			}
			docNumber := fmt.Sprintf("PZ/%d/%03d", year, seq)
			notes := fmt.Sprintf("Inwentaryzacja: %s - nadwyżki", existing.Name)

			doc := &model.WarehouseDocument{
				ID:             uuid.New(),
				TenantID:       tenantID,
				DocumentNumber: docNumber,
				DocumentType:   "PZ",
				Status:         "draft",
				WarehouseID:    existing.WarehouseID,
				Notes:          &notes,
				CreatedBy:      &actorID,
			}

			if err := s.docRepo.Create(ctx, tx, doc); err != nil {
				return fmt.Errorf("create PZ doc: %w", err)
			}

			for _, item := range surplusItems {
				docItem := &model.WarehouseDocItem{
					ID:         uuid.New(),
					TenantID:   tenantID,
					DocumentID: doc.ID,
					ProductID:  item.ProductID,
					Quantity:   item.Difference, // positive
				}
				if err := s.docItemRepo.Create(ctx, tx, docItem); err != nil {
					return fmt.Errorf("create PZ doc item: %w", err)
				}

				// Adjust stock: add surplus
				if err := s.stockRepo.AdjustQuantity(ctx, tx, existing.WarehouseID, item.ProductID, nil, item.Difference); err != nil {
					return fmt.Errorf("PZ stock adjust: %w", err)
				}
			}

			// Confirm PZ document
			if err := s.docRepo.Confirm(ctx, tx, doc.ID, actorID); err != nil {
				return fmt.Errorf("confirm PZ doc: %w", err)
			}
		}

		// Create WZ document for shortages
		if len(shortageItems) > 0 {
			seq, err := s.docRepo.NextDocumentNumber(ctx, tx, "WZ", year)
			if err != nil {
				return fmt.Errorf("next WZ number: %w", err)
			}
			docNumber := fmt.Sprintf("WZ/%d/%03d", year, seq)
			notes := fmt.Sprintf("Inwentaryzacja: %s - niedobory", existing.Name)

			doc := &model.WarehouseDocument{
				ID:             uuid.New(),
				TenantID:       tenantID,
				DocumentNumber: docNumber,
				DocumentType:   "WZ",
				Status:         "draft",
				WarehouseID:    existing.WarehouseID,
				Notes:          &notes,
				CreatedBy:      &actorID,
			}

			if err := s.docRepo.Create(ctx, tx, doc); err != nil {
				return fmt.Errorf("create WZ doc: %w", err)
			}

			for _, item := range shortageItems {
				absQty := -item.Difference // make positive
				docItem := &model.WarehouseDocItem{
					ID:         uuid.New(),
					TenantID:   tenantID,
					DocumentID: doc.ID,
					ProductID:  item.ProductID,
					Quantity:   absQty,
				}
				if err := s.docItemRepo.Create(ctx, tx, docItem); err != nil {
					return fmt.Errorf("create WZ doc item: %w", err)
				}

				// Adjust stock: subtract shortage
				if err := s.stockRepo.AdjustQuantity(ctx, tx, existing.WarehouseID, item.ProductID, nil, item.Difference); err != nil {
					return fmt.Errorf("WZ stock adjust: %w", err)
				}
			}

			// Confirm WZ document
			if err := s.docRepo.Confirm(ctx, tx, doc.ID, actorID); err != nil {
				return fmt.Errorf("confirm WZ doc: %w", err)
			}
		}

		// Mark stocktake as completed
		if err := s.stocktakeRepo.SetCompletedAt(ctx, tx, stocktakeID); err != nil {
			return err
		}

		stocktake, err = s.stocktakeRepo.FindByID(ctx, tx, stocktakeID)
		if err != nil {
			return err
		}
		stocktake.Stats = stats

		if err := s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "stocktake.completed",
			EntityType: "stocktake",
			EntityID:   stocktakeID,
			Changes: map[string]string{
				"name":          existing.Name,
				"discrepancies": fmt.Sprintf("%d", stats.Discrepancies),
				"surplus":       fmt.Sprintf("%d", stats.SurplusCount),
				"shortage":      fmt.Sprintf("%d", stats.ShortageCount),
			},
			IPAddress: ip,
		}); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// Dispatch webhook asynchronously
	if s.webhookDispatch != nil {
		go s.webhookDispatch.Dispatch(context.Background(), tenantID, "stocktake.completed", stocktake)
	}

	return stocktake, nil
}

// CancelStocktake sets the stocktake status to cancelled.
func (s *StocktakeService) CancelStocktake(ctx context.Context, tenantID, stocktakeID uuid.UUID, actorID uuid.UUID, ip string) (*model.Stocktake, error) {
	var stocktake *model.Stocktake
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		existing, err := s.stocktakeRepo.FindByID(ctx, tx, stocktakeID)
		if err != nil {
			return err
		}
		if existing == nil {
			return ErrStocktakeNotFound
		}
		if existing.Status != "draft" && existing.Status != "in_progress" {
			return fmt.Errorf("stocktake cannot be cancelled in %s status", existing.Status)
		}

		if err := s.stocktakeRepo.UpdateStatus(ctx, tx, stocktakeID, "cancelled"); err != nil {
			return err
		}

		stocktake, err = s.stocktakeRepo.FindByID(ctx, tx, stocktakeID)
		if err != nil {
			return err
		}

		stats, err := s.itemRepo.GetStats(ctx, tx, stocktakeID)
		if err != nil {
			return err
		}
		stocktake.Stats = stats

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "stocktake.cancelled",
			EntityType: "stocktake",
			EntityID:   stocktakeID,
			Changes:    map[string]string{"name": existing.Name},
			IPAddress:  ip,
		})
	})
	if err != nil {
		return nil, err
	}
	return stocktake, nil
}

// GetStocktake retrieves a stocktake by ID with stats.
func (s *StocktakeService) GetStocktake(ctx context.Context, tenantID, stocktakeID uuid.UUID) (*model.Stocktake, error) {
	var stocktake *model.Stocktake
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var err error
		stocktake, err = s.stocktakeRepo.FindByID(ctx, tx, stocktakeID)
		if err != nil {
			return err
		}
		if stocktake == nil {
			return ErrStocktakeNotFound
		}

		stats, err := s.itemRepo.GetStats(ctx, tx, stocktakeID)
		if err != nil {
			return err
		}
		stocktake.Stats = stats

		return nil
	})
	if err != nil {
		return nil, err
	}
	return stocktake, nil
}

// ListStocktakes lists stocktakes with filtering and pagination.
func (s *StocktakeService) ListStocktakes(ctx context.Context, tenantID uuid.UUID, filter model.StocktakeListFilter) (model.ListResponse[model.Stocktake], error) {
	var resp model.ListResponse[model.Stocktake]
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		stocktakes, total, err := s.stocktakeRepo.List(ctx, tx, filter)
		if err != nil {
			return err
		}
		if stocktakes == nil {
			stocktakes = []model.Stocktake{}
		}

		// Enrich each stocktake with stats
		for i := range stocktakes {
			stats, err := s.itemRepo.GetStats(ctx, tx, stocktakes[i].ID)
			if err != nil {
				return err
			}
			stocktakes[i].Stats = stats
		}

		resp = model.ListResponse[model.Stocktake]{
			Items:  stocktakes,
			Total:  total,
			Limit:  filter.Limit,
			Offset: filter.Offset,
		}
		return nil
	})
	return resp, err
}

// GetStocktakeItems lists items for a stocktake with pagination and filtering.
func (s *StocktakeService) GetStocktakeItems(ctx context.Context, tenantID, stocktakeID uuid.UUID, filter model.StocktakeItemListFilter) (model.ListResponse[model.StocktakeItem], error) {
	var resp model.ListResponse[model.StocktakeItem]
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		// Validate stocktake exists
		stocktake, err := s.stocktakeRepo.FindByID(ctx, tx, stocktakeID)
		if err != nil {
			return err
		}
		if stocktake == nil {
			return ErrStocktakeNotFound
		}

		items, total, err := s.itemRepo.List(ctx, tx, stocktakeID, filter)
		if err != nil {
			return err
		}
		if items == nil {
			items = []model.StocktakeItem{}
		}

		resp = model.ListResponse[model.StocktakeItem]{
			Items:  items,
			Total:  total,
			Limit:  filter.Limit,
			Offset: filter.Offset,
		}
		return nil
	})
	return resp, err
}

// DeleteStocktake deletes a stocktake (only if draft).
func (s *StocktakeService) DeleteStocktake(ctx context.Context, tenantID, stocktakeID uuid.UUID, actorID uuid.UUID, ip string) error {
	return database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		existing, err := s.stocktakeRepo.FindByID(ctx, tx, stocktakeID)
		if err != nil {
			return err
		}
		if existing == nil {
			return ErrStocktakeNotFound
		}
		if existing.Status != "draft" {
			return ErrStocktakeNotDraft
		}

		if err := s.stocktakeRepo.Delete(ctx, tx, stocktakeID); err != nil {
			return err
		}

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "stocktake.deleted",
			EntityType: "stocktake",
			EntityID:   stocktakeID,
			Changes:    map[string]string{"name": existing.Name},
			IPAddress:  ip,
		})
	})
}
