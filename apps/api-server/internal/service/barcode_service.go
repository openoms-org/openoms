package service

import (
	"context"
	"encoding/json"
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
	ErrBarcodeNotFound     = errors.New("no product found for this barcode")
	ErrPackingItemMismatch = errors.New("scanned items do not match order items")
)

type BarcodeService struct {
	productRepo repository.ProductRepo
	variantRepo repository.VariantRepo
	orderRepo   repository.OrderRepo
	auditRepo   repository.AuditRepo
	pool        *pgxpool.Pool
}

func NewBarcodeService(
	productRepo repository.ProductRepo,
	variantRepo repository.VariantRepo,
	orderRepo repository.OrderRepo,
	auditRepo repository.AuditRepo,
	pool *pgxpool.Pool,
) *BarcodeService {
	return &BarcodeService{
		productRepo: productRepo,
		variantRepo: variantRepo,
		orderRepo:   orderRepo,
		auditRepo:   auditRepo,
		pool:        pool,
	}
}

// Lookup searches for a product by SKU or EAN code.
func (s *BarcodeService) Lookup(ctx context.Context, tenantID uuid.UUID, code string) (*model.BarcodeLookupResponse, error) {
	var resp model.BarcodeLookupResponse

	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		// Try product SKU first
		product, err := s.productRepo.FindBySKU(ctx, tx, code)
		if err != nil {
			return err
		}
		if product != nil {
			resp.Product = product
			// If the product has variants, find matching variants too
			if product.HasVariants {
				variants, err := s.variantRepo.FindBySKU(ctx, tx, code)
				if err != nil {
					return err
				}
				resp.Variants = variants
			}
			return nil
		}

		// Try product EAN
		product, err = s.productRepo.FindByEAN(ctx, tx, code)
		if err != nil {
			return err
		}
		if product != nil {
			resp.Product = product
			if product.HasVariants {
				variants, err := s.variantRepo.FindByEAN(ctx, tx, code)
				if err != nil {
					return err
				}
				resp.Variants = variants
			}
			return nil
		}

		// Try variant SKU
		variants, err := s.variantRepo.FindBySKU(ctx, tx, code)
		if err != nil {
			return err
		}
		if len(variants) > 0 {
			resp.Variants = variants
			// Fetch the parent product
			product, err = s.productRepo.FindByID(ctx, tx, variants[0].ProductID)
			if err != nil {
				return err
			}
			resp.Product = product
			return nil
		}

		// Try variant EAN
		variants, err = s.variantRepo.FindByEAN(ctx, tx, code)
		if err != nil {
			return err
		}
		if len(variants) > 0 {
			resp.Variants = variants
			product, err = s.productRepo.FindByID(ctx, tx, variants[0].ProductID)
			if err != nil {
				return err
			}
			resp.Product = product
			return nil
		}

		return ErrBarcodeNotFound
	})

	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// PackOrder validates scanned items against order items and marks the order as packed.
func (s *BarcodeService) PackOrder(ctx context.Context, tenantID, orderID, actorID uuid.UUID, req model.PackOrderRequest) (*model.PackOrderResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	var resp model.PackOrderResponse
	now := time.Now()

	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		order, err := s.orderRepo.FindByID(ctx, tx, orderID)
		if err != nil {
			return err
		}
		if order == nil {
			return ErrOrderNotFound
		}

		// Parse order items to validate scanned items match
		if order.Items != nil {
			type OrderItem struct {
				SKU      string `json:"sku"`
				Quantity int    `json:"quantity"`
			}
			var orderItems []OrderItem
			if err := json.Unmarshal(order.Items, &orderItems); err == nil {
				// Build expected map
				expected := make(map[string]int)
				for _, item := range orderItems {
					if item.SKU != "" {
						expected[item.SKU] += item.Quantity
					}
				}

				// Build scanned map
				scanned := make(map[string]int)
				for _, item := range req.ScannedItems {
					scanned[item.SKU] += item.Quantity
				}

				// Validate that scanned items match expected
				for sku, qty := range expected {
					if scanned[sku] < qty {
						return fmt.Errorf("%w: brakuje produktu %s (oczekiwano %d, zeskanowano %d)", ErrPackingItemMismatch, sku, qty, scanned[sku])
					}
				}
			}
		}

		// Update order metadata with packing info
		metadata := make(map[string]interface{})
		if order.Metadata != nil {
			_ = json.Unmarshal(order.Metadata, &metadata)
		}
		metadata["packed_at"] = now.Format(time.RFC3339)
		metadata["packed_by"] = actorID.String()

		metadataBytes, err := json.Marshal(metadata)
		if err != nil {
			return fmt.Errorf("marshal metadata: %w", err)
		}
		metadataRaw := json.RawMessage(metadataBytes)

		updateReq := model.UpdateOrderRequest{
			Metadata: metadataRaw,
		}
		if err := s.orderRepo.Update(ctx, tx, orderID, updateReq); err != nil {
			return err
		}

		if err := s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "order.packed",
			EntityType: "order",
			EntityID:   orderID,
			Changes:    map[string]string{"packed_at": now.Format(time.RFC3339)},
		}); err != nil {
			return err
		}

		resp = model.PackOrderResponse{
			OrderID:  orderID,
			PackedAt: now.Format(time.RFC3339),
			PackedBy: actorID,
			Status:   order.Status,
		}

		return nil
	})

	if err != nil {
		return nil, err
	}
	return &resp, nil
}
