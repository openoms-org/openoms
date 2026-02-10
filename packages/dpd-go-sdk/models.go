package dpd

import "time"

// CreateParcelRequest is the payload sent to create a new DPD shipment.
type CreateParcelRequest struct {
	Sender    Address      `json:"sender"`
	Receiver  Address      `json:"receiver"`
	Parcels   []ParcelSpec `json:"parcels"`
	Services  *Services    `json:"services,omitempty"`
	Reference string       `json:"reference,omitempty"`
}

// Address contains address details for sender or receiver.
type Address struct {
	Name        string `json:"name"`
	Company     string `json:"company,omitempty"`
	Street      string `json:"street"`
	City        string `json:"city"`
	PostalCode  string `json:"postalCode"`
	CountryCode string `json:"countryCode"`
	Phone       string `json:"phone,omitempty"`
	Email       string `json:"email,omitempty"`
}

// ParcelSpec describes a single parcel's dimensions and weight.
type ParcelSpec struct {
	Weight float64 `json:"weight"`
	SizeX  float64 `json:"sizeX,omitempty"`
	SizeY  float64 `json:"sizeY,omitempty"`
	SizeZ  float64 `json:"sizeZ,omitempty"`
}

// Services describes optional services for the shipment.
type Services struct {
	COD           *COD   `json:"cod,omitempty"`
	DeclaredValue *Money `json:"declaredValue,omitempty"`
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

// CreateParcelResponse is returned after a shipment is created.
type CreateParcelResponse struct {
	ParcelID string `json:"parcelId"`
	Waybill  string `json:"waybill"`
	Status   string `json:"status"`
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

// TrackingEvent represents a single tracking event from DPD.
type TrackingEvent struct {
	Status      string    `json:"status"`
	Description string    `json:"description,omitempty"`
	Location    string    `json:"location,omitempty"`
	DateTime    time.Time `json:"dateTime"`
}
