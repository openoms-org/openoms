package poczta

import "time"

// CreateShipmentRequest is the payload sent to create a new Poczta Polska shipment.
type CreateShipmentRequest struct {
	ServiceType string   `json:"serviceType"`
	Receiver    Receiver `json:"receiver"`
	Parcel      Parcel   `json:"parcel"`
	Reference   string   `json:"reference,omitempty"`
	COD         *COD     `json:"cod,omitempty"`
	Insurance   *Money   `json:"insurance,omitempty"`
}

// Receiver contains recipient details.
type Receiver struct {
	Name       string `json:"name"`
	Email      string `json:"email,omitempty"`
	Phone      string `json:"phone,omitempty"`
	Street     string `json:"street"`
	City       string `json:"city"`
	PostalCode string `json:"postalCode"`
	Country    string `json:"country"`
}

// Parcel describes a parcel to ship.
type Parcel struct {
	Weight float64 `json:"weight"`
	Width  float64 `json:"width,omitempty"`
	Height float64 `json:"height,omitempty"`
	Length float64 `json:"length,omitempty"`
}

// COD describes cash-on-delivery parameters.
type COD struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

// Money represents a monetary value.
type Money struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

// ShipmentResponse is returned after a shipment is created.
type ShipmentResponse struct {
	ShipmentID     string `json:"shipmentId"`
	TrackingNumber string `json:"trackingNumber"`
	Status         string `json:"status"`
	LabelURL       string `json:"labelUrl,omitempty"`
}

// LabelResponse contains label data from the API.
type LabelResponse struct {
	LabelData   string `json:"labelData"` // base64-encoded PDF
	LabelFormat string `json:"labelFormat"`
}

// TrackingResponse contains tracking information.
type TrackingResponse struct {
	ShipmentID     string          `json:"shipmentId"`
	TrackingNumber string          `json:"trackingNumber"`
	Events         []TrackingEvent `json:"events"`
}

// TrackingEvent represents a single tracking event from Poczta Polska.
type TrackingEvent struct {
	Status    string    `json:"status"`
	Location  string    `json:"location,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	Details   string    `json:"details,omitempty"`
}
