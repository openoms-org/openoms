package dpd

import "time"

// CreateParcelRequest is the payload sent to POST /public/shipment/v1/generatePackagesNumbers.
type CreateParcelRequest struct {
	Sender      *Address     `json:"sender,omitempty"`
	Receiver    Address      `json:"receiver"`
	Parcels     []ParcelSpec `json:"parcels"`
	Services    *Services    `json:"services,omitempty"`
	Reference   string       `json:"reference,omitempty"`
	ServiceType string       `json:"serviceType,omitempty"`
	TargetPoint string       `json:"targetPoint,omitempty"`
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

// createParcelRawResponse is the nested response from /public/shipment/v1/generatePackagesNumbers.
type createParcelRawResponse struct {
	Status    string            `json:"status"`
	SessionID int64             `json:"sessionId"`
	Packages  []responsePackage `json:"packages"`
}

type responsePackage struct {
	StatusInfo packageStatus   `json:"statusInfo"`
	Parcels    []responseParcel `json:"parcels"`
}

type packageStatus struct {
	Status string `json:"status"`
}

type responseParcel struct {
	Status    string `json:"status"`
	Reference string `json:"reference,omitempty"`
	Waybill   string `json:"waybill"`
}

// CreateParcelResponse is the parsed response after shipment creation.
type CreateParcelResponse struct {
	SessionID int64  `json:"sessionId"`
	ParcelID  string `json:"parcelId"`
	Waybill   string `json:"waybill"`
	Status    string `json:"status"`
}

// generateLabelRequest is the payload for POST /public/shipment/v1/generateSpedLabels.
type generateLabelRequest struct {
	DPDServicesParcelsPPLOD []labelParcel `json:"DPDServicesParcelsPPLOD"`
	OutputDocFormat         string        `json:"OutputDocFormat"`
	OutputDocPage           string        `json:"OutputDocPage"`
}

type labelParcel struct {
	WaybillRef string `json:"parcel_waybill"`
}

// generateLabelResponse contains label data from POST /public/shipment/v1/generateSpedLabels.
type generateLabelResponse struct {
	Status       string `json:"status"`
	DocumentData string `json:"documentData"`
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
