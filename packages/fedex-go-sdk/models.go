package fedex

import "time"

// ShipmentRequest is the payload sent to create a new FedEx shipment.
type ShipmentRequest struct {
	AccountNumber   AccountNumber   `json:"accountNumber"`
	RequestedShipment RequestedShipment `json:"requestedShipment"`
}

// AccountNumber wraps the FedEx account number.
type AccountNumber struct {
	Value string `json:"value"`
}

// RequestedShipment contains shipment details.
type RequestedShipment struct {
	Shipper          Party           `json:"shipper"`
	Recipients       []Party         `json:"recipients"`
	ServiceType      string          `json:"serviceType"`
	PackagingType    string          `json:"packagingType"`
	RequestedPackageLineItems []PackageLineItem `json:"requestedPackageLineItems"`
	ShipmentSpecialServices   *ShipmentSpecialServices `json:"shipmentSpecialServices,omitempty"`
	CustomerReferences        []CustomerReference      `json:"customerReferences,omitempty"`
	LabelSpecification        LabelSpecification       `json:"labelSpecification"`
}

// Party contains shipper or recipient details.
type Party struct {
	Contact Contact `json:"contact"`
	Address Address `json:"address"`
}

// Contact contains person/company contact details.
type Contact struct {
	PersonName  string `json:"personName"`
	PhoneNumber string `json:"phoneNumber,omitempty"`
	EmailAddress string `json:"emailAddress,omitempty"`
}

// Address contains FedEx-formatted address fields.
type Address struct {
	StreetLines   []string `json:"streetLines"`
	City          string   `json:"city"`
	StateOrProvince string `json:"stateOrProvinceCode,omitempty"`
	PostalCode    string   `json:"postalCode"`
	CountryCode   string   `json:"countryCode"`
}

// PackageLineItem describes a package for shipment.
type PackageLineItem struct {
	Weight     Weight     `json:"weight"`
	Dimensions *Dimensions `json:"dimensions,omitempty"`
}

// Weight describes package weight.
type Weight struct {
	Units string  `json:"units"` // "KG" or "LB"
	Value float64 `json:"value"`
}

// Dimensions describes package dimensions.
type Dimensions struct {
	Units  string  `json:"units"` // "CM" or "IN"
	Length float64 `json:"length"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// ShipmentSpecialServices contains special service options.
type ShipmentSpecialServices struct {
	SpecialServiceTypes []string `json:"specialServiceTypes,omitempty"`
	CodDetail           *CODDetail `json:"codDetail,omitempty"`
}

// CODDetail describes cash-on-delivery details.
type CODDetail struct {
	CodCollectionAmount Money `json:"codCollectionAmount"`
}

// Money represents a monetary value.
type Money struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

// CustomerReference contains a reference value for the shipment.
type CustomerReference struct {
	CustomerReferenceType string `json:"customerReferenceType"`
	Value                 string `json:"value"`
}

// LabelSpecification defines label output preferences.
type LabelSpecification struct {
	LabelFormatType string `json:"labelFormatType"` // "COMMON2D", "LABEL_DATA_ONLY"
	ImageType       string `json:"imageType"`       // "PDF", "PNG", "ZPL"
	LabelStockType  string `json:"labelStockType"`  // "PAPER_4X6"
}

// ShipmentResponse is returned after a shipment is created.
type ShipmentResponse struct {
	TransactionID string              `json:"transactionId"`
	Output        ShipmentOutput      `json:"output"`
}

// ShipmentOutput contains the shipment creation result.
type ShipmentOutput struct {
	TransactionShipments []TransactionShipment `json:"transactionShipments"`
}

// TransactionShipment contains details of a created shipment.
type TransactionShipment struct {
	MasterTrackingNumber string           `json:"masterTrackingNumber"`
	ShipmentID           string           `json:"shipmentId,omitempty"`
	PieceResponses       []PieceResponse  `json:"pieceResponses"`
}

// PieceResponse contains details for a shipped piece.
type PieceResponse struct {
	TrackingNumber   string `json:"trackingNumber"`
	PackageDocuments []PackageDocument `json:"packageDocuments"`
}

// PackageDocument contains label document info.
type PackageDocument struct {
	ContentType string   `json:"contentType"`
	EncodedLabel string  `json:"encodedLabel,omitempty"` // base64-encoded
	URL         string   `json:"url,omitempty"`
}

// TrackingInfo contains tracking information for a shipment.
type TrackingInfo struct {
	TrackingNumber string           `json:"trackingNumber"`
	Events         []TrackingEvent  `json:"scanEvents"`
	LatestStatus   string           `json:"latestStatusDetail"`
}

// TrackingEvent represents a single tracking event from FedEx.
type TrackingEvent struct {
	EventType   string    `json:"eventType"`
	Description string    `json:"eventDescription"`
	Date        time.Time `json:"date"`
	City        string    `json:"scanLocation,omitempty"`
}

// TrackingResponse is the response from the FedEx tracking API.
type TrackingResponse struct {
	Output TrackingOutput `json:"output"`
}

// TrackingOutput contains tracking results.
type TrackingOutput struct {
	CompleteTrackResults []CompleteTrackResult `json:"completeTrackResults"`
}

// CompleteTrackResult contains results for one tracking number.
type CompleteTrackResult struct {
	TrackResults []TrackResult `json:"trackResults"`
}

// TrackResult contains tracking details.
type TrackResult struct {
	TrackingNumberInfo TrackingNumberInfo `json:"trackingNumberInfo"`
	LatestStatusDetail StatusDetail       `json:"latestStatusDetail"`
	ScanEvents         []ScanEvent        `json:"scanEvents"`
}

// TrackingNumberInfo contains tracking number details.
type TrackingNumberInfo struct {
	TrackingNumber string `json:"trackingNumber"`
}

// StatusDetail contains status information.
type StatusDetail struct {
	Code        string `json:"code"`
	Description string `json:"description"`
}

// ScanEvent represents a single scan/tracking event.
type ScanEvent struct {
	Date             string      `json:"date"`
	EventType        string      `json:"eventType"`
	EventDescription string      `json:"eventDescription"`
	ScanLocation     ScanLocation `json:"scanLocation"`
}

// ScanLocation contains location details for a scan event.
type ScanLocation struct {
	City        string `json:"city"`
	CountryCode string `json:"countryCode"`
}

// CancelShipmentRequest is the payload sent to cancel a FedEx shipment.
type CancelShipmentRequest struct {
	AccountNumber AccountNumber `json:"accountNumber"`
	TrackingNumber string       `json:"trackingNumber"`
}
