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
	ErrWarehouseDocumentNotFound = errors.New("warehouse document not found")
	ErrDocumentNotDraft          = errors.New("document is not in draft status")
)

// WarehouseDocumentService provides business logic for warehouse documents.
type WarehouseDocumentService struct {
	docRepo   repository.WarehouseDocumentRepo
	itemRepo  repository.WarehouseDocItemRepo
	stockRepo repository.WarehouseStockRepo
	auditRepo repository.AuditRepo
	pool      *pgxpool.Pool
}

// NewWarehouseDocumentService creates a new WarehouseDocumentService.
func NewWarehouseDocumentService(
	docRepo repository.WarehouseDocumentRepo,
	itemRepo repository.WarehouseDocItemRepo,
	stockRepo repository.WarehouseStockRepo,
	auditRepo repository.AuditRepo,
	pool *pgxpool.Pool,
) *WarehouseDocumentService {
	return &WarehouseDocumentService{
		docRepo:   docRepo,
		itemRepo:  itemRepo,
		stockRepo: stockRepo,
		auditRepo: auditRepo,
		pool:      pool,
	}
}

// List lists warehouse documents for a tenant.
func (s *WarehouseDocumentService) List(ctx context.Context, tenantID uuid.UUID, filter model.WarehouseDocumentListFilter) (model.ListResponse[model.WarehouseDocument], error) {
	var resp model.ListResponse[model.WarehouseDocument]
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		docs, total, err := s.docRepo.List(ctx, tx, filter)
		if err != nil {
			return err
		}
		if docs == nil {
			docs = []model.WarehouseDocument{}
		}
		resp = model.ListResponse[model.WarehouseDocument]{
			Items:  docs,
			Total:  total,
			Limit:  filter.Limit,
			Offset: filter.Offset,
		}
		return nil
	})
	return resp, err
}

// Get retrieves a single warehouse document by ID, including its items.
func (s *WarehouseDocumentService) Get(ctx context.Context, tenantID, docID uuid.UUID) (*model.WarehouseDocument, error) {
	var doc *model.WarehouseDocument
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var err error
		doc, err = s.docRepo.FindByID(ctx, tx, docID)
		if err != nil {
			return err
		}
		if doc == nil {
			return ErrWarehouseDocumentNotFound
		}
		items, err := s.itemRepo.ListByDocumentID(ctx, tx, docID)
		if err != nil {
			return err
		}
		if items == nil {
			items = []model.WarehouseDocItem{}
		}
		doc.Items = items
		return nil
	})
	if err != nil {
		return nil, err
	}
	return doc, nil
}

// Create creates a new warehouse document with auto-generated document number.
func (s *WarehouseDocumentService) Create(ctx context.Context, tenantID uuid.UUID, req model.CreateWarehouseDocumentRequest, actorID uuid.UUID, ip string) (*model.WarehouseDocument, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	var doc *model.WarehouseDocument
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		// Generate document number: TYPE/YEAR/SEQ
		year := time.Now().Year()
		seq, err := s.docRepo.NextDocumentNumber(ctx, tx, req.DocumentType, year)
		if err != nil {
			return err
		}
		docNumber := fmt.Sprintf("%s/%d/%03d", req.DocumentType, year, seq)

		doc = &model.WarehouseDocument{
			ID:                uuid.New(),
			TenantID:          tenantID,
			DocumentNumber:    docNumber,
			DocumentType:      req.DocumentType,
			Status:            "draft",
			WarehouseID:       req.WarehouseID,
			TargetWarehouseID: req.TargetWarehouseID,
			SupplierID:        req.SupplierID,
			OrderID:           req.OrderID,
			Notes:             req.Notes,
			CreatedBy:         &actorID,
		}

		if err := s.docRepo.Create(ctx, tx, doc); err != nil {
			return err
		}

		// Create items
		var items []model.WarehouseDocItem
		for _, itemReq := range req.Items {
			item := &model.WarehouseDocItem{
				ID:         uuid.New(),
				TenantID:   tenantID,
				DocumentID: doc.ID,
				ProductID:  itemReq.ProductID,
				VariantID:  itemReq.VariantID,
				Quantity:   itemReq.Quantity,
				UnitPrice:  itemReq.UnitPrice,
				Notes:      itemReq.Notes,
			}
			if err := s.itemRepo.Create(ctx, tx, item); err != nil {
				return err
			}
			items = append(items, *item)
		}
		doc.Items = items

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "warehouse_document.created",
			EntityType: "warehouse_document",
			EntityID:   doc.ID,
			Changes:    map[string]string{"document_number": docNumber, "type": req.DocumentType},
			IPAddress:  ip,
		})
	})
	if err != nil {
		return nil, err
	}
	return doc, nil
}

// Update updates a warehouse document (only draft documents).
func (s *WarehouseDocumentService) Update(ctx context.Context, tenantID, docID uuid.UUID, req model.UpdateWarehouseDocumentRequest, actorID uuid.UUID, ip string) (*model.WarehouseDocument, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	var doc *model.WarehouseDocument
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		existing, err := s.docRepo.FindByID(ctx, tx, docID)
		if err != nil {
			return err
		}
		if existing == nil {
			return ErrWarehouseDocumentNotFound
		}
		if existing.Status != "draft" {
			return ErrDocumentNotDraft
		}
		if err := s.docRepo.Update(ctx, tx, docID, req); err != nil {
			return err
		}
		doc, err = s.docRepo.FindByID(ctx, tx, docID)
		if err != nil {
			return err
		}
		items, err := s.itemRepo.ListByDocumentID(ctx, tx, docID)
		if err != nil {
			return err
		}
		if items == nil {
			items = []model.WarehouseDocItem{}
		}
		doc.Items = items

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "warehouse_document.updated",
			EntityType: "warehouse_document",
			EntityID:   docID,
			IPAddress:  ip,
		})
	})
	if err != nil {
		return nil, err
	}
	return doc, nil
}

// Delete deletes a warehouse document (only draft documents).
func (s *WarehouseDocumentService) Delete(ctx context.Context, tenantID, docID uuid.UUID, actorID uuid.UUID, ip string) error {
	return database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		existing, err := s.docRepo.FindByID(ctx, tx, docID)
		if err != nil {
			return err
		}
		if existing == nil {
			return ErrWarehouseDocumentNotFound
		}
		if existing.Status != "draft" {
			return ErrDocumentNotDraft
		}
		if err := s.docRepo.Delete(ctx, tx, docID); err != nil {
			return err
		}
		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "warehouse_document.deleted",
			EntityType: "warehouse_document",
			EntityID:   docID,
			Changes:    map[string]string{"document_number": existing.DocumentNumber},
			IPAddress:  ip,
		})
	})
}

// Confirm confirms a warehouse document and updates stock levels.
// PZ: adds stock to warehouse
// WZ: subtracts stock from warehouse
// MM: subtracts from source, adds to target
func (s *WarehouseDocumentService) Confirm(ctx context.Context, tenantID, docID uuid.UUID, actorID uuid.UUID, ip string) (*model.WarehouseDocument, error) {
	var doc *model.WarehouseDocument
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		existing, err := s.docRepo.FindByID(ctx, tx, docID)
		if err != nil {
			return err
		}
		if existing == nil {
			return ErrWarehouseDocumentNotFound
		}
		if existing.Status != "draft" {
			return ErrDocumentNotDraft
		}

		// Get items
		items, err := s.itemRepo.ListByDocumentID(ctx, tx, docID)
		if err != nil {
			return err
		}

		// Update stock based on document type
		for _, item := range items {
			switch existing.DocumentType {
			case "PZ":
				// Add stock to warehouse
				if err := s.stockRepo.AdjustQuantity(ctx, tx, existing.WarehouseID, item.ProductID, item.VariantID, item.Quantity); err != nil {
					return fmt.Errorf("PZ stock adjust: %w", err)
				}
			case "WZ":
				// Subtract stock from warehouse
				if err := s.stockRepo.AdjustQuantity(ctx, tx, existing.WarehouseID, item.ProductID, item.VariantID, -item.Quantity); err != nil {
					return fmt.Errorf("WZ stock adjust: %w", err)
				}
			case "MM":
				// Subtract from source warehouse
				if err := s.stockRepo.AdjustQuantity(ctx, tx, existing.WarehouseID, item.ProductID, item.VariantID, -item.Quantity); err != nil {
					return fmt.Errorf("MM source stock adjust: %w", err)
				}
				// Add to target warehouse
				if existing.TargetWarehouseID != nil {
					if err := s.stockRepo.AdjustQuantity(ctx, tx, *existing.TargetWarehouseID, item.ProductID, item.VariantID, item.Quantity); err != nil {
						return fmt.Errorf("MM target stock adjust: %w", err)
					}
				}
			}
		}

		// Mark as confirmed
		if err := s.docRepo.Confirm(ctx, tx, docID, actorID); err != nil {
			return err
		}

		doc, err = s.docRepo.FindByID(ctx, tx, docID)
		if err != nil {
			return err
		}
		doc.Items = items

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "warehouse_document.confirmed",
			EntityType: "warehouse_document",
			EntityID:   docID,
			Changes:    map[string]string{"document_number": existing.DocumentNumber, "type": existing.DocumentType},
			IPAddress:  ip,
		})
	})
	if err != nil {
		return nil, err
	}
	return doc, nil
}

// Cancel cancels a warehouse document (only draft documents).
func (s *WarehouseDocumentService) Cancel(ctx context.Context, tenantID, docID uuid.UUID, actorID uuid.UUID, ip string) (*model.WarehouseDocument, error) {
	var doc *model.WarehouseDocument
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		existing, err := s.docRepo.FindByID(ctx, tx, docID)
		if err != nil {
			return err
		}
		if existing == nil {
			return ErrWarehouseDocumentNotFound
		}
		if existing.Status != "draft" {
			return ErrDocumentNotDraft
		}

		if err := s.docRepo.Cancel(ctx, tx, docID); err != nil {
			return err
		}

		doc, err = s.docRepo.FindByID(ctx, tx, docID)
		if err != nil {
			return err
		}
		items, err := s.itemRepo.ListByDocumentID(ctx, tx, docID)
		if err != nil {
			return err
		}
		if items == nil {
			items = []model.WarehouseDocItem{}
		}
		doc.Items = items

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "warehouse_document.cancelled",
			EntityType: "warehouse_document",
			EntityID:   docID,
			Changes:    map[string]string{"document_number": existing.DocumentNumber},
			IPAddress:  ip,
		})
	})
	if err != nil {
		return nil, err
	}
	return doc, nil
}
