package gls

import "time"

// CreateParcelRequest is the payload sent to POST /shipments per GLS ShipIT REST API.
// See: https://shipit.gls-group.eu/webservices/3_2_9/doxygen/WS-REST-API/rest_shipment_processing.html
type CreateParcelRequest struct {
	Consignee       Consignee        `json:"Consignee"`
	Shipper         *Shipper         `json:"Shipper,omitempty"`
	ShippingUnit    []ShipmentUnit   `json:"ShippingUnit"`
	Product         string           `json:"Product"` // mandatory: "PARCEL", "EXPRESS", etc.
	Service         *ServiceSection  `json:"Service,omitempty"`
	References      []string         `json:"References,omitempty"`
	Middleware      *Middleware      `json:"Middleware,omitempty"`
	PrintingOptions *PrintingOptions `json:"PrintingOptions,omitempty"`
}

// Consignee wraps the recipient address.
type Consignee struct {
	Address ConsigneeAddress `json:"Address"`
}

// ConsigneeAddress contains recipient address details per GLS ShipIT API field names.
type ConsigneeAddress struct {
	Name1       string `json:"Name1"`
	Name2       string `json:"Name2,omitempty"`
	Street      string `json:"Street"`
	City        string `json:"City"`
	ZIPCode     string `json:"ZIPCode"`
	CountryCode string `json:"CountryCode"`
	Phone       string `json:"Phone,omitempty"`
	EMail       string `json:"eMail,omitempty"`
}

// Shipper references the registered shipper by their GLS ContactID.
// Full shipper address is registered at GLS and referenced by ContactID.
type Shipper struct {
	ContactID string `json:"ContactID"`
}

// ShipmentUnit describes a single parcel's weight and optional dimensions.
type ShipmentUnit struct {
	Weight float64 `json:"Weight"`
	Width  float64 `json:"Width,omitempty"`
	Height float64 `json:"Height,omitempty"`
	Depth  float64 `json:"Depth,omitempty"`
}

// ServiceSection wraps the list of optional services.
type ServiceSection struct {
	Service []Service `json:"Service"`
}

// Service represents a GLS optional service.
// Use ServiceName "service_cash" for COD, "service_addonliability" for insurance.
type Service struct {
	ServiceName    string                 `json:"ServiceName"`
	Cash           *CashService           `json:"Cash,omitempty"`
	AddOnLiability *AddOnLiabilityService `json:"AddOnLiability,omitempty"`
}

// CashService contains COD (cash-on-delivery) payment details.
type CashService struct {
	Amount   float64 `json:"Amount"`
	Currency string  `json:"Currency"`
	Reason   string  `json:"Reason"`
}

// AddOnLiabilityService contains additional liability (insurance) details.
type AddOnLiabilityService struct {
	ParcelContent string  `json:"ParcelContent"`
	Currency      string  `json:"Currency,omitempty"`
	Amount        float64 `json:"Amount,omitempty"`
}

// Middleware provides GLS system integration metadata (required by GLS API).
type Middleware struct {
	SendingDepot string `json:"SendingDepot,omitempty"`
	Software     string `json:"Software"`
	SoftVersion  string `json:"SoftVersion"`
}

// PrintingOptions controls the label format returned in the create response.
type PrintingOptions struct {
	ReturnLabels LabelOptions `json:"ReturnLabels"`
}

// LabelOptions specifies label template and format.
type LabelOptions struct {
	TemplateSet string `json:"TemplateSet"` // e.g. "NONE", "A4"
	LabelFormat string `json:"LabelFormat"` // e.g. "PDF", "ZPL"
}

// rawCreateParcelResponse is the actual GLS ShipIT API response structure.
// PrintData is at CreatedShipment level (sibling of ParcelData, not nested inside it).
type rawCreateParcelResponse struct {
	CreatedShipment struct {
		ShipmentReference string `json:"ShipmentReference"`
		ParcelData        []struct {
			TrackID      string `json:"TrackID"`
			ParcelNumber string `json:"ParcelNumber,omitempty"`
		} `json:"ParcelData"`
		PrintData []struct {
			Data     string `json:"Data"` // base64-encoded label
			Sequence int    `json:"Sequence,omitempty"`
		} `json:"PrintData"`
	} `json:"CreatedShipment"`
}

// CreateParcelResponse is returned after parcels are created.
type CreateParcelResponse struct {
	ParcelIDs []string `json:"parcel_ids"`
	TrackIDs  []string `json:"track_ids"`
	PrintData []string `json:"print_data"` // base64-encoded labels from create response
}

// ParcelDetailsRequest is used to retrieve tracking info (POST /shipments/parceldetails).
// GLS API expects a single TrackID string, not an array.
type ParcelDetailsRequest struct {
	TrackID string `json:"TrackID"`
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
