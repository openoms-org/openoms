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
	OrderID       string          `json:"order_id"`
	ServiceType   string          `json:"service_type"`
	Receiver      CarrierReceiver `json:"receiver"`
	Parcel        CarrierParcel   `json:"parcel"`
	TargetPoint   string          `json:"target_point,omitempty"`   // locker ID for InPost
	SendingMethod string          `json:"sending_method,omitempty"` // e.g. parcel_locker, dispatch_order
	CODAmount     float64         `json:"cod_amount,omitempty"`
	CODCurrency   string          `json:"cod_currency,omitempty"`
	InsuredValue  float64         `json:"insured_value,omitempty"`
	Reference     string          `json:"reference,omitempty"`
}

// CarrierShipmentResponse is returned after a shipment is created with a carrier.
type CarrierShipmentResponse struct {
	ExternalID     string `json:"external_id"`
	TrackingNumber string `json:"tracking_number"`
	Status         string `json:"status"`
	LabelURL       string `json:"label_url,omitempty"`
}

// RateRequest contains all data needed to request shipping rates from a carrier.
type RateRequest struct {
	FromPostalCode string  `json:"from_postal_code"`
	FromCountry    string  `json:"from_country"`
	ToPostalCode   string  `json:"to_postal_code"`
	ToCountry      string  `json:"to_country"`
	Weight         float64 `json:"weight"` // kg
	Width          float64 `json:"width"`  // cm
	Height         float64 `json:"height"` // cm
	Length         float64 `json:"length"` // cm
	COD            float64 `json:"cod"`    // cash on delivery amount, 0 if none
	IsPickupPoint  bool    `json:"is_pickup_point"`
}

// Rate represents a single shipping rate option from a carrier.
type Rate struct {
	CarrierName   string  `json:"carrier_name"`
	CarrierCode   string  `json:"carrier_code"`
	ServiceName   string  `json:"service_name"`
	Price         float64 `json:"price"`
	Currency      string  `json:"currency"`
	EstimatedDays int     `json:"estimated_days"`
	PickupPoint   bool    `json:"pickup_point"`
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
	GetRates(ctx context.Context, req RateRequest) ([]Rate, error)
}

// DispatchOrderAddress is the pickup address for a dispatch order.
type DispatchOrderAddress struct {
	Street         string `json:"street"`
	BuildingNumber string `json:"building_number"`
	City           string `json:"city"`
	PostCode       string `json:"post_code"`
	CountryCode    string `json:"country_code"`
}

// DispatchOrderContact is the contact info for a dispatch order.
type DispatchOrderContact struct {
	Name    string `json:"name"`
	Phone   string `json:"phone"`
	Email   string `json:"email"`
	Comment string `json:"comment,omitempty"`
}

// DispatchOrderCreator is an optional interface that carrier providers can implement
// to support creating dispatch orders (courier pickup requests).
type DispatchOrderCreator interface {
	CreateDispatchOrder(ctx context.Context, shipmentExternalIDs []int64, address DispatchOrderAddress, contact DispatchOrderContact) (int64, error)
}
