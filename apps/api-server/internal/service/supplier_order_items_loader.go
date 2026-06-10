package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// Narrow repo surfaces the items loader needs (unit-testable with fakes; satisfied by the
// concrete repositories wired in main.go).

// dropshipOrdersReader lists an order's dropship orders.
type dropshipOrdersReader interface {
	FindByOrderID(ctx context.Context, tx pgx.Tx, orderID uuid.UUID) ([]model.DropshipOrder, error)
}

// dropshipItemsReader lists a dropship order's line items.
type dropshipItemsReader interface {
	ListByDropshipOrderID(ctx context.Context, tx pgx.Tx, dropshipOrderID uuid.UUID) ([]model.DropshipOrderItem, error)
}

// productReader loads a tenant product (for its canonical EAN).
type productReader interface {
	FindByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.Product, error)
}

// supplierProductMappingReader resolves the supplier_products mapping for
// (supplier, internal product) — the supplier's own catalogue identity.
type supplierProductMappingReader interface {
	FindBySupplierAndProductID(ctx context.Context, tx pgx.Tx, supplierID, productID uuid.UUID) (*model.SupplierProduct, error)
}

// NewSupplierOrderItemsLoader builds the supplier-order items loader (OPE-516): it resolves
// the dropship lines for an (order, supplier) into the request builder's input shape. The
// line's ItemID is the SUPPLIER's catalogue identity (supplier_products.external_id — e.g.
// the BTP item id that maps straight onto OrderRequestLine.ItemID), resolved from the
// supplier_products mapping for (supplier_id, product_id). The EAN comes from the tenant
// product. When no mapping exists the line is sent EAN-only (ItemID empty) — NEVER the
// tenant's internal SKU, which is meaningless (or worse, a different item) at the supplier.
// A line with neither mapping nor EAN carries no identity, so BuildSupplierOrderRequest
// routes it into the missing-identity list (supplier_order_missing_data path). Runs inside
// the supplier-order handler's tenant tx.
func NewSupplierOrderItemsLoader(
	dropshipRepo dropshipOrdersReader,
	itemRepo dropshipItemsReader,
	productRepo productReader,
	mappingRepo supplierProductMappingReader,
) func(ctx context.Context, tx pgx.Tx, orderID, supplierID uuid.UUID) ([]SupplierOrderInputLine, []string, error) {
	return func(ctx context.Context, tx pgx.Tx, orderID, supplierID uuid.UUID) ([]SupplierOrderInputLine, []string, error) {
		dsOrders, err := dropshipRepo.FindByOrderID(ctx, tx, orderID)
		if err != nil {
			return nil, nil, fmt.Errorf("find dropship orders: %w", err)
		}
		var lines []SupplierOrderInputLine
		var missing []string
		for di := range dsOrders {
			if dsOrders[di].SupplierID != supplierID {
				continue
			}
			items, ierr := itemRepo.ListByDropshipOrderID(ctx, tx, dsOrders[di].ID)
			if ierr != nil {
				return nil, nil, fmt.Errorf("list dropship items: %w", ierr)
			}
			for ii := range items {
				line := SupplierOrderInputLine{
					LineID:   items[ii].ID,
					Quantity: items[ii].Quantity,
				}
				if items[ii].ProductID != nil {
					// EAN: the tenant product's canonical barcode (best-effort — a missing
					// product never fails the load; the line may still resolve via mapping).
					if product, perr := productRepo.FindByID(ctx, tx, *items[ii].ProductID); perr == nil && product != nil {
						if product.EAN != nil && *product.EAN != "" {
							line.EAN = *product.EAN
						}
					}
					// ItemID: the supplier's catalogue identity from the supplier_products
					// mapping. A lookup failure is retryable — silently degrading the primary
					// identity could order the wrong item.
					mapping, merr := mappingRepo.FindBySupplierAndProductID(ctx, tx, supplierID, *items[ii].ProductID)
					if merr != nil {
						return nil, nil, fmt.Errorf("find supplier product mapping: %w", merr)
					}
					if mapping != nil && strings.TrimSpace(mapping.ExternalID) != "" {
						line.SupplierSKU = strings.TrimSpace(mapping.ExternalID)
					}
				}
				lines = append(lines, line)
			}
		}
		return lines, missing, nil
	}
}
