package model

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// PurchaseOrder represents a purchase order to a supplier.
type PurchaseOrder struct {
	ID                   uuid.UUID           `json:"id"`
	TenantID             uuid.UUID           `json:"tenant_id"`
	PONumber             string              `json:"po_number"`
	SupplierID           *uuid.UUID          `json:"supplier_id,omitempty"`
	SupplierName         string              `json:"supplier_name"`
	WarehouseID          *uuid.UUID          `json:"warehouse_id,omitempty"`
	Status               string              `json:"status"`
	Notes                string              `json:"notes,omitempty"`
	ExpectedDeliveryDate *time.Time          `json:"expected_delivery_date,omitempty"`
	TotalAmount          float64             `json:"total_amount"`
	Currency             string              `json:"currency"`
	CreatedBy            *uuid.UUID          `json:"created_by,omitempty"`
	Items                []PurchaseOrderItem `json:"items,omitempty"`
	CreatedAt            time.Time           `json:"created_at"`
	UpdatedAt            time.Time           `json:"updated_at"`
}

// PurchaseOrderItem represents a line item in a purchase order.
type PurchaseOrderItem struct {
	ID               uuid.UUID  `json:"id"`
	TenantID         uuid.UUID  `json:"tenant_id"`
	PurchaseOrderID  uuid.UUID  `json:"purchase_order_id"`
	ProductID        *uuid.UUID `json:"product_id,omitempty"`
	SKU              string     `json:"sku"`
	ProductName      string     `json:"product_name"`
	QuantityOrdered  int        `json:"quantity_ordered"`
	QuantityReceived int        `json:"quantity_received"`
	UnitCost         float64    `json:"unit_cost"`
	TotalCost        float64    `json:"total_cost"`
	Notes            string     `json:"notes,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
}

// CreatePurchaseOrderRequest is the payload for creating a purchase order.
type CreatePurchaseOrderRequest struct {
	SupplierID           *uuid.UUID                   `json:"supplier_id"`
	SupplierName         string                       `json:"supplier_name"`
	WarehouseID          *uuid.UUID                   `json:"warehouse_id"`
	Notes                string                       `json:"notes"`
	ExpectedDeliveryDate *string                      `json:"expected_delivery_date"`
	Currency             string                       `json:"currency"`
	Items                []CreatePurchaseOrderItemReq  `json:"items"`
}

// Validate validates the create purchase order request.
func (r *CreatePurchaseOrderRequest) Validate() error {
	if strings.TrimSpace(r.SupplierName) == "" {
		return errors.New("supplier_name is required")
	}
	if err := validateMaxLength("supplier_name", r.SupplierName, 500); err != nil {
		return err
	}
	if r.Currency == "" {
		r.Currency = "PLN"
	}
	if len(r.Items) == 0 {
		return errors.New("at least one item is required")
	}
	for _, item := range r.Items {
		if err := item.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// CreatePurchaseOrderItemReq is a single item in the create PO request.
type CreatePurchaseOrderItemReq struct {
	ProductID   *uuid.UUID `json:"product_id"`
	SKU         string     `json:"sku"`
	ProductName string     `json:"product_name"`
	Quantity    int        `json:"quantity"`
	UnitCost    float64    `json:"unit_cost"`
	Notes       string     `json:"notes"`
}

// Validate validates a single PO item request.
func (r *CreatePurchaseOrderItemReq) Validate() error {
	if strings.TrimSpace(r.ProductName) == "" {
		return errors.New("product_name is required")
	}
	if r.Quantity <= 0 {
		return errors.New("quantity must be greater than 0")
	}
	if r.UnitCost < 0 {
		return errors.New("unit_cost must not be negative")
	}
	return nil
}

// UpdatePurchaseOrderRequest is the payload for updating a purchase order.
type UpdatePurchaseOrderRequest struct {
	SupplierID           *uuid.UUID                    `json:"supplier_id,omitempty"`
	SupplierName         *string                       `json:"supplier_name,omitempty"`
	WarehouseID          *uuid.UUID                    `json:"warehouse_id,omitempty"`
	Notes                *string                       `json:"notes,omitempty"`
	ExpectedDeliveryDate *string                       `json:"expected_delivery_date,omitempty"`
	Currency             *string                       `json:"currency,omitempty"`
	Items                []CreatePurchaseOrderItemReq   `json:"items,omitempty"`
}

// Validate validates the update purchase order request.
func (r *UpdatePurchaseOrderRequest) Validate() error {
	if r.SupplierName != nil && strings.TrimSpace(*r.SupplierName) == "" {
		return errors.New("supplier_name must not be empty")
	}
	if r.Items != nil {
		for _, item := range r.Items {
			if err := item.Validate(); err != nil {
				return err
			}
		}
	}
	return nil
}

// ReceiveItemsRequest is the payload for receiving items against a PO.
type ReceiveItemsRequest struct {
	Items []ReceiveItemEntry `json:"items"`
}

// ReceiveItemEntry describes the quantity received for a single PO item.
type ReceiveItemEntry struct {
	ItemID           uuid.UUID `json:"item_id"`
	QuantityReceived int       `json:"quantity_received"`
}

// Validate validates the receive items request.
func (r *ReceiveItemsRequest) Validate() error {
	if len(r.Items) == 0 {
		return errors.New("at least one item is required")
	}
	for _, entry := range r.Items {
		if entry.ItemID == uuid.Nil {
			return errors.New("item_id is required")
		}
		if entry.QuantityReceived <= 0 {
			return errors.New("quantity_received must be greater than 0")
		}
	}
	return nil
}

// PurchaseOrderListFilter holds the filtering/pagination for listing purchase orders.
type PurchaseOrderListFilter struct {
	Status     *string
	SupplierID *uuid.UUID
	PaginationParams
}
