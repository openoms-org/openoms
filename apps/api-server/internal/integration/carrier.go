package integration

import (
	"context"
	"time"
)

// CarrierReceiver represents the recipient of a shipment.
type CarrierReceiver struct {
	Name       string `json:"name"`
	Email      string `json:"email,omitempty"`
	Phone      string `json:"phone"`
	Street     string `json:"street"`
	City       string `json:"city"`
	PostalCode string `json:"postal_code"`
	Country    string `json:"country"`
}

// CarrierParcel describes the physical parcel to be shipped.
type CarrierParcel struct {
	SizeCode string  `json:"size_code,omitempty"` // e.g. "small", "medium", "large" for InPost
	WeightKg float64 `json:"weight_kg,omitempty"`
	WidthCm  float64 `json:"width_cm,omitempty"`
	HeightCm float64 `json:"height_cm,omitempty"`
	DepthCm  float64 `json:"depth_cm,omitempty"`
}

// TrackingEvent represents a single tracking event from a carrier.
type TrackingEvent struct {
	Status    string    `json:"status"`
	Location  string    `json:"location,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	Details   string    `json:"details,omitempty"`
}

// PickupPoint represents a carrier pickup/drop-off point (e.g. InPost locker).
type PickupPoint struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	Street     string  `json:"street"`
	City       string  `json:"city"`
	PostalCode string  `json:"postal_code"`
	Latitude   float64 `json:"latitude,omitempty"`
	Longitude  float64 `json:"longitude,omitempty"`
	Type       string  `json:"type,omitempty"` // e.g. "parcel_locker", "pop"
}

// CarrierShipmentRequest contains all data needed to create a shipment with a carrier.
type CarrierShipmentRequest struct {
	OrderID      string          `json:"order_id"`
	ServiceType  string          `json:"service_type"`
	Receiver     CarrierReceiver `json:"receiver"`
	Parcel       CarrierParcel   `json:"parcel"`
	TargetPoint  string          `json:"target_point,omitempty"` // locker ID for InPost
	CODAmount    float64         `json:"cod_amount,omitempty"`
	CODCurrency  string          `json:"cod_currency,omitempty"`
	InsuredValue float64         `json:"insured_value,omitempty"`
	Reference    string          `json:"reference,omitempty"`
}

// CarrierShipmentResponse is returned after a shipment is created with a carrier.
type CarrierShipmentResponse struct {
	ExternalID     string `json:"external_id"`
	TrackingNumber string `json:"tracking_number"`
	Status         string `json:"status"`
	LabelURL       string `json:"label_url,omitempty"`
}

// CarrierProvider defines the interface for carrier/shipping integrations
// (e.g. InPost, DHL, DPD).
type CarrierProvider interface {
	ProviderName() string
	CreateShipment(ctx context.Context, req CarrierShipmentRequest) (*CarrierShipmentResponse, error)
	GetLabel(ctx context.Context, externalID string, format string) ([]byte, error)
	GetTracking(ctx context.Context, trackingNumber string) ([]TrackingEvent, error)
	CancelShipment(ctx context.Context, externalID string) error
	MapStatus(carrierStatus string) (omsStatus string, ok bool)
	SupportsPickupPoints() bool
	SearchPickupPoints(ctx context.Context, query string) ([]PickupPoint, error)
}
