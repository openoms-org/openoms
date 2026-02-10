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
}

func (r *CreateShipmentRequest) Validate() error {
	if r.OrderID == uuid.Nil {
		return errors.New("order_id is required")
	}
	switch r.Provider {
	case "inpost", "dhl", "dpd", "manual":
		// valid
	default:
		return errors.New("provider must be one of: inpost, dhl, dpd, manual")
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
	ServiceType string `json:"service_type"`
	ParcelSize  string `json:"parcel_size"`
	TargetPoint string `json:"target_point"`
	LabelFormat string `json:"label_format"`
}

func (r *GenerateLabelRequest) Validate() error {
	switch r.ServiceType {
	case "inpost_locker_standard", "inpost_courier_standard":
		// valid
	default:
		return errors.New("service_type must be one of: inpost_locker_standard, inpost_courier_standard")
	}

	switch r.ParcelSize {
	case "small", "medium", "large":
		// valid
	default:
		return errors.New("parcel_size must be one of: small, medium, large")
	}

	if r.ServiceType == "inpost_locker_standard" && strings.TrimSpace(r.TargetPoint) == "" {
		return errors.New("target_point is required for inpost_locker_standard service")
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
