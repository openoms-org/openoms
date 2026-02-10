package gls

import "time"

// CreateParcelRequest is the payload sent to create a new GLS parcel.
type CreateParcelRequest struct {
	Shipper   Party    `json:"shipper"`
	Consignee Party    `json:"consignee"`
	Parcels   []Parcel `json:"parcels"`
	Services  []string `json:"services,omitempty"`
	Reference string   `json:"reference,omitempty"`
}

// Party contains address details for shipper or consignee.
type Party struct {
	Name        string `json:"name"`
	Street      string `json:"street"`
	City        string `json:"city"`
	ZipCode     string `json:"zipCode"`
	CountryCode string `json:"countryCode"`
	Phone       string `json:"phone,omitempty"`
	Email       string `json:"email,omitempty"`
}

// Parcel describes a single parcel's dimensions and weight.
type Parcel struct {
	Weight float64 `json:"weight"`
	Width  float64 `json:"width,omitempty"`
	Height float64 `json:"height,omitempty"`
	Length float64 `json:"length,omitempty"`
}

// CreateParcelResponse is returned after parcels are created.
type CreateParcelResponse struct {
	ParcelIDs []string `json:"parcel_ids"`
	TrackIDs  []string `json:"track_ids"`
}

// LabelResponse contains label data from the API.
type LabelResponse struct {
	LabelData   string `json:"labelData"`
	LabelFormat string `json:"labelFormat"`
}

// TrackingResponse contains tracking information for a parcel.
type TrackingResponse struct {
	Events []TrackingEvent `json:"events"`
}

// TrackingEvent represents a single tracking event from GLS.
type TrackingEvent struct {
	Status    string    `json:"status"`
	Location  string    `json:"location,omitempty"`
	Details   string    `json:"details,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}
