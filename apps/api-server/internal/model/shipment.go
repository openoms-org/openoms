package model

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Shipment struct {
	ID             uuid.UUID       `json:"id"`
	TenantID       uuid.UUID       `json:"tenant_id"`
	OrderID        uuid.UUID       `json:"order_id"`
	Provider       string          `json:"provider"`
	IntegrationID  *uuid.UUID      `json:"integration_id,omitempty"`
	TrackingNumber *string         `json:"tracking_number,omitempty"`
	Status         string          `json:"status"`
	LabelURL       *string         `json:"label_url,omitempty"`
	CarrierData    json.RawMessage `json:"carrier_data,omitempty"`
	WarehouseID    *uuid.UUID      `json:"warehouse_id,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

type CreateShipmentRequest struct {
	OrderID        uuid.UUID       `json:"order_id"`
	Provider       string          `json:"provider"`
	IntegrationID  *uuid.UUID      `json:"integration_id,omitempty"`
	TrackingNumber *string         `json:"tracking_number,omitempty"`
	LabelURL       *string         `json:"label_url,omitempty"`
	CarrierData    json.RawMessage `json:"carrier_data,omitempty"`
	WarehouseID    *uuid.UUID      `json:"warehouse_id,omitempty"`
}

func (r *CreateShipmentRequest) Validate() error {
	if r.OrderID == uuid.Nil {
		return errors.New("order_id is required")
	}
	if r.Provider == "" {
		return errors.New("provider is required")
	}
	if err := validateMaxLength("provider", r.Provider, 100); err != nil {
		return err
	}
	if err := validateMaxLengthPtr("tracking_number", r.TrackingNumber, 200); err != nil {
		return err
	}
	return nil
}

type UpdateShipmentRequest struct {
	TrackingNumber *string         `json:"tracking_number,omitempty"`
	LabelURL       *string         `json:"label_url,omitempty"`
	CarrierData    json.RawMessage `json:"carrier_data,omitempty"`
}

func (r *UpdateShipmentRequest) Validate() error {
	if r.TrackingNumber == nil && r.LabelURL == nil && r.CarrierData == nil {
		return errors.New("at least one field must be provided")
	}
	return nil
}

type ShipmentStatusTransitionRequest struct {
	Status string `json:"status"`
}

func (r *ShipmentStatusTransitionRequest) Validate() error {
	if strings.TrimSpace(r.Status) == "" {
		return errors.New("status is required")
	}
	return nil
}

type GenerateLabelRequest struct {
	ServiceType   string  `json:"service_type"`
	ParcelSize    string  `json:"parcel_size,omitempty"`
	TargetPoint   string  `json:"target_point,omitempty"`
	SendingMethod string  `json:"sending_method,omitempty"`
	LabelFormat   string  `json:"label_format"`
	WeightKg      float64 `json:"weight_kg,omitempty"`
	WidthCm       float64 `json:"width_cm,omitempty"`
	HeightCm      float64 `json:"height_cm,omitempty"`
	DepthCm       float64 `json:"depth_cm,omitempty"`
	CODAmount     float64 `json:"cod_amount,omitempty"`
	InsuredValue  float64 `json:"insured_value,omitempty"`
}

// validSendingMethods defines the allowed sending method values.
var validSendingMethods = map[string]bool{
	"parcel_locker":  true,
	"dispatch_order": true,
	"pop":            true,
	"any_point":      true,
	"pok":            true,
	"branch":         true,
}

func (r *GenerateLabelRequest) Validate() error {
	if strings.TrimSpace(r.ServiceType) == "" {
		return errors.New("service_type is required")
	}

	if r.ServiceType == "inpost_locker_standard" && strings.TrimSpace(r.TargetPoint) == "" {
		return errors.New("target_point is required for inpost_locker_standard service")
	}

	if r.SendingMethod != "" && !validSendingMethods[r.SendingMethod] {
		return errors.New("sending_method must be one of: parcel_locker, dispatch_order, pop, any_point, pok, branch")
	}

	if r.LabelFormat == "" {
		r.LabelFormat = "pdf"
	}
	switch r.LabelFormat {
	case "pdf", "zpl", "epl":
		// valid
	default:
		return errors.New("label_format must be one of: pdf, zpl, epl")
	}

	return nil
}

type ShipmentListFilter struct {
	Status   *string
	Provider *string
	OrderID  *uuid.UUID
	PaginationParams
}

// CreateDispatchOrderRequest is the payload for creating a dispatch order (courier pickup).
type CreateDispatchOrderRequest struct {
	ShipmentIDs []uuid.UUID `json:"shipment_ids"`
	Street      string      `json:"street,omitempty"`
	BuildingNo  string      `json:"building_number,omitempty"`
	City        string      `json:"city,omitempty"`
	PostCode    string      `json:"post_code,omitempty"`
	Name        string      `json:"name,omitempty"`
	Phone       string      `json:"phone,omitempty"`
	Email       string      `json:"email,omitempty"`
	Comment     string      `json:"comment,omitempty"`
}

// DispatchOrderResponse is the response returned after creating a dispatch order.
type DispatchOrderResponse struct {
	ID     int64  `json:"id"`
	Status string `json:"status"`
}

// BatchLabelsRequest is the payload for batch label download.
type BatchLabelsRequest struct {
	ShipmentIDs []uuid.UUID `json:"shipment_ids"`
}

// BatchLabelResult holds the label data for a single shipment.
type BatchLabelResult struct {
	ShipmentID string `json:"shipment_id"`
	Data       []byte `json:"-"`
	Error      string `json:"error,omitempty"`
}
