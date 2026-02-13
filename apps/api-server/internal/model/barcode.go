package model

import (
	"errors"

	"github.com/google/uuid"
)

// BarcodeLookupResponse is the response for a barcode/SKU/EAN lookup.
type BarcodeLookupResponse struct {
	Product  *Product         `json:"product,omitempty"`
	Variants []ProductVariant `json:"variants,omitempty"`
}

// ScannedItem represents a single scanned item in the packing request.
type ScannedItem struct {
	SKU      string `json:"sku"`
	Quantity int    `json:"quantity"`
}

// PackOrderRequest is the request body for POST /v1/orders/{id}/pack.
type PackOrderRequest struct {
	ScannedItems []ScannedItem `json:"scanned_items"`
}

func (r *PackOrderRequest) Validate() error {
	if len(r.ScannedItems) == 0 {
		return errors.New("scanned_items is required and must not be empty")
	}
	for _, item := range r.ScannedItems {
		if item.SKU == "" {
			return errors.New("each scanned_item must have a sku")
		}
		if item.Quantity <= 0 {
			return errors.New("each scanned_item must have a positive quantity")
		}
	}
	return nil
}

// PackOrderResponse is the response for POST /v1/orders/{id}/pack.
type PackOrderResponse struct {
	OrderID  uuid.UUID `json:"order_id"`
	PackedAt string    `json:"packed_at"`
	PackedBy uuid.UUID `json:"packed_by"`
	Status   string    `json:"status"`
}
