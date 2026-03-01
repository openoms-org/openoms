package gls

import "time"

// CreateParcelRequest is the payload sent to create a new GLS shipment.
type CreateParcelRequest struct {
	Shipper     Party     `json:"Shipper,omitempty"`
	Consignee   Party     `json:"Consignee"`
	Parcels     []Parcel  `json:"Parcels"`
	ServiceType string    `json:"ServiceType,omitempty"`
	Services    []Service `json:"Services,omitempty"`
	Reference   string    `json:"Reference,omitempty"`
}

// Party contains address details for shipper or consignee.
type Party struct {
	Name        string `json:"Name"`
	Street      string `json:"Street"`
	City        string `json:"City"`
	ZipCode     string `json:"ZipCode"`
	CountryCode string `json:"CountryCode"`
	Phone       string `json:"Phone,omitempty"`
	Email       string `json:"Email,omitempty"`
}

// Parcel describes a single parcel's dimensions and weight.
type Parcel struct {
	Weight float64 `json:"Weight"`
	Width  float64 `json:"Width,omitempty"`
	Height float64 `json:"Height,omitempty"`
	Depth  float64 `json:"Depth,omitempty"`
}

// Service represents a GLS optional service (e.g. COD).
type Service struct {
	ServiceName string  `json:"serviceName"`
	Amount      float64 `json:"amount,omitempty"`
	Currency    string  `json:"currency,omitempty"`
}

// rawCreateParcelResponse is the actual GLS ShipIT API response structure.
type rawCreateParcelResponse struct {
	CreatedShipment struct {
		ShipmentReference string `json:"ShipmentReference"`
		ParcelData        []struct {
			TrackID   string `json:"TrackID"`
			PrintData string `json:"PrintData"`
		} `json:"ParcelData"`
	} `json:"CreatedShipment"`
}

// CreateParcelResponse is returned after parcels are created.
type CreateParcelResponse struct {
	ParcelIDs []string `json:"parcel_ids"`
	TrackIDs  []string `json:"track_ids"`
	PrintData []string `json:"print_data"` // base64-encoded labels from create response
}

// ParcelDetailsRequest is used to retrieve tracking info (POST /shipments/parceldetails).
type ParcelDetailsRequest struct {
	TrackIDs []string `json:"TrackIDs"`
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
