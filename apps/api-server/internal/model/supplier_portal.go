package model

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// SupplierPortalToken represents a portal access token record (stores hash, not raw token).
type SupplierPortalToken struct {
	ID         uuid.UUID  `json:"id"`
	TenantID   uuid.UUID  `json:"tenant_id"`
	SupplierID uuid.UUID  `json:"supplier_id"`
	TokenHash  string     `json:"-"` // never expose
	ExpiresAt  time.Time  `json:"expires_at"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

// SupplierMessage represents a message/note on a purchase order.
type SupplierMessage struct {
	ID              uuid.UUID  `json:"id"`
	TenantID        uuid.UUID  `json:"tenant_id"`
	PurchaseOrderID uuid.UUID  `json:"purchase_order_id"`
	SupplierID      *uuid.UUID `json:"supplier_id,omitempty"`
	UserID          *uuid.UUID `json:"user_id,omitempty"`
	Message         string     `json:"message"`
	IsFromSupplier  bool       `json:"is_from_supplier"`
	CreatedAt       time.Time  `json:"created_at"`
}

// SupplierPortalPO is a limited view of a purchase order for suppliers.
type SupplierPortalPO struct {
	ID                   uuid.UUID           `json:"id"`
	PONumber             string              `json:"po_number"`
	Status               string              `json:"status"`
	Notes                string              `json:"notes,omitempty"`
	ExpectedDeliveryDate *time.Time          `json:"expected_delivery_date,omitempty"`
	TotalAmount          float64             `json:"total_amount"`
	Currency             string              `json:"currency"`
	Items                []PurchaseOrderItem `json:"items,omitempty"`
	CreatedAt            time.Time           `json:"created_at"`
	UpdatedAt            time.Time           `json:"updated_at"`
}

// SupplierConfirmRequest is the payload when a supplier confirms a PO.
type SupplierConfirmRequest struct {
	Reference string `json:"reference"` // optional supplier reference number
}

// SupplierShipRequest is the payload when a supplier marks a PO as shipped.
type SupplierShipRequest struct {
	TrackingNumber string `json:"tracking_number"`
	Carrier        string `json:"carrier"`
}

// Validate validates the supplier ship request.
func (r *SupplierShipRequest) Validate() error {
	if strings.TrimSpace(r.TrackingNumber) == "" {
		return errors.New("tracking_number is required")
	}
	if strings.TrimSpace(r.Carrier) == "" {
		return errors.New("carrier is required")
	}
	if err := validateMaxLength("tracking_number", r.TrackingNumber, 200); err != nil {
		return err
	}
	if err := validateMaxLength("carrier", r.Carrier, 200); err != nil {
		return err
	}
	return nil
}

// CreateSupplierMessageRequest is the payload for adding a message to a PO.
type CreateSupplierMessageRequest struct {
	Message string `json:"message"`
}

// Validate validates the create supplier message request.
func (r *CreateSupplierMessageRequest) Validate() error {
	if strings.TrimSpace(r.Message) == "" {
		return errors.New("message is required")
	}
	if err := validateMaxLength("message", r.Message, 5000); err != nil {
		return err
	}
	return nil
}

// SupplierPortalLinkResponse is returned when generating a portal link.
type SupplierPortalLinkResponse struct {
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expires_at"`
}
