package ups

import "time"

// ShipmentRequest is the payload sent to create a new UPS shipment.
type ShipmentRequest struct {
	Shipper     Party        `json:"Shipper"`
	ShipTo      Party        `json:"ShipTo"`
	Service     ServiceCode  `json:"Service"`
	Package     []PackageSpec `json:"Package"`
	Description string       `json:"Description,omitempty"`
	Reference   *Reference   `json:"ReferenceNumber,omitempty"`
}

// Party contains shipper or recipient details.
type Party struct {
	Name    string     `json:"Name"`
	Address UPSAddress `json:"Address"`
	Phone   *Phone     `json:"Phone,omitempty"`
}

// UPSAddress contains UPS-formatted address fields.
type UPSAddress struct {
	AddressLine       []string `json:"AddressLine"`
	City              string   `json:"City"`
	StateProvinceCode string   `json:"StateProvinceCode,omitempty"`
	PostalCode        string   `json:"PostalCode"`
	CountryCode       string   `json:"CountryCode"`
}

// Phone contains a phone number.
type Phone struct {
	Number string `json:"Number"`
}

// ServiceCode identifies a UPS service type.
type ServiceCode struct {
	Code        string `json:"Code"`
	Description string `json:"Description,omitempty"`
}

// PackageSpec describes a package for shipment.
type PackageSpec struct {
	PackagingType Code      `json:"PackagingType"`
	Dimensions    Dims      `json:"Dimensions,omitempty"`
	PackageWeight PkgWeight `json:"PackageWeight"`
}

// Code is a generic code container used by UPS API.
type Code struct {
	Code string `json:"Code"`
}

// Dims describes package dimensions.
type Dims struct {
	UnitOfMeasurement Code   `json:"UnitOfMeasurement"`
	Length            string `json:"Length"`
	Width             string `json:"Width"`
	Height            string `json:"Height"`
}

// PkgWeight describes package weight.
type PkgWeight struct {
	UnitOfMeasurement Code   `json:"UnitOfMeasurement"`
	Weight            string `json:"Weight"`
}

// Reference contains a reference value for the shipment.
type Reference struct {
	Value string `json:"Value"`
}

// ShipmentResponse is returned after a shipment is created.
type ShipmentResponse struct {
	TrackingNumber string `json:"trackingNumber"`
	ShipmentID     string `json:"shipmentId"`
	LabelImage     string `json:"labelImage"` // base64-encoded
}

// TrackingResponse contains tracking information for a shipment.
type TrackingResponse struct {
	Events []TrackingEvent `json:"events"`
}

// TrackingEvent represents a single tracking event from UPS.
type TrackingEvent struct {
	Status      string    `json:"status"`
	Location    string    `json:"location,omitempty"`
	Description string    `json:"description,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
}
