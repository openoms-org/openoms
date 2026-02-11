package inpost

// ServiceType identifies the InPost shipping service.
type ServiceType string

const (
	ServiceLockerStandard  ServiceType = "inpost_locker_standard"
	ServiceCourierStandard ServiceType = "inpost_courier_standard"
)

// ParcelTemplate identifies the standard locker parcel sizes.
type ParcelTemplate string

const (
	ParcelSmall  ParcelTemplate = "small"  // A: 8x38x64cm
	ParcelMedium ParcelTemplate = "medium" // B: 19x38x64cm
	ParcelLarge  ParcelTemplate = "large"  // C: 41x38x64cm
)

// CreateShipmentRequest is the payload sent to create a new shipment.
type CreateShipmentRequest struct {
	Receiver         Receiver          `json:"receiver"`
	Parcels          []Parcel          `json:"parcels"`
	Service          ServiceType       `json:"service"`
	Reference        string            `json:"reference,omitempty"`
	Comments         string            `json:"comments,omitempty"`
	CustomAttributes *CustomAttributes `json:"custom_attributes,omitempty"`
}

// Receiver contains recipient details.
type Receiver struct {
	Name    string   `json:"name"`
	Phone   string   `json:"phone"`
	Email   string   `json:"email"`
	Address *Address `json:"address,omitempty"`
}

// Address is a physical mailing address.
type Address struct {
	Street      string `json:"street"`
	City        string `json:"city"`
	PostCode    string `json:"post_code"`
	CountryCode string `json:"country_code"`
}

// Parcel describes a parcel to ship.
type Parcel struct {
	Template   ParcelTemplate `json:"template,omitempty"`
	Dimensions *Dimensions    `json:"dimensions,omitempty"`
	Weight     Weight         `json:"weight"`
}

// Dimensions holds parcel measurements in millimeters.
type Dimensions struct {
	Height float64 `json:"height"`
	Width  float64 `json:"width"`
	Length float64 `json:"length"`
}

// Weight holds parcel weight.
type Weight struct {
	Amount float64 `json:"amount"`
	Unit   string  `json:"unit"`
}

// CustomAttributes holds service-specific attributes.
type CustomAttributes struct {
	TargetPoint   string `json:"target_point,omitempty"`
	SendingMethod string `json:"sending_method,omitempty"`
}

// Shipment is the API response for a shipment resource.
type Shipment struct {
	ID               int64             `json:"id"`
	Status           string            `json:"status"`
	TrackingNumber   string            `json:"tracking_number"`
	Service          string            `json:"service"`
	Receiver         Receiver          `json:"receiver"`
	Parcels          []ParcelResponse  `json:"parcels"`
	CustomAttributes *CustomAttributes `json:"custom_attributes"`
	CreatedAt        string            `json:"created_at"`
	UpdatedAt        string            `json:"updated_at"`
}

// ParcelResponse is a parcel as returned by the API.
type ParcelResponse struct {
	ID         int64      `json:"id"`
	Template   string     `json:"template"`
	Dimensions Dimensions `json:"dimensions"`
	Weight     Weight     `json:"weight"`
}

// PointType identifies the type of InPost point.
type PointType string

const (
	PointTypeParcelLocker PointType = "parcel_locker"
	PointTypePOP          PointType = "pop"
)

// PointAddress is the address of an InPost point.
type PointAddress struct {
	Line1 string `json:"line1"`
	Line2 string `json:"line2"`
}

// PointLocation holds GPS coordinates.
type PointLocation struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// PointAddressDetails contains structured address fields.
type PointAddressDetails struct {
	City           string `json:"city"`
	Province       string `json:"province"`
	PostCode       string `json:"post_code"`
	Street         string `json:"street"`
	BuildingNumber string `json:"building_number"`
}

// Point represents an InPost pickup/delivery point (paczkomat).
type Point struct {
	Name                string               `json:"name"`
	Type                []string             `json:"type"`
	Address             PointAddress         `json:"address"`
	AddressDetails      *PointAddressDetails `json:"address_details,omitempty"`
	Location            PointLocation        `json:"location"`
	LocationDescription string               `json:"location_description"`
	OpeningHours        string               `json:"opening_hours"`
	Status              string               `json:"status"`
}

// PointSearchResponse is the paginated response from the points search endpoint.
type PointSearchResponse struct {
	Items      []Point `json:"items"`
	Count      int     `json:"count"`
	Page       int     `json:"page"`
	PerPage    int     `json:"per_page"`
	TotalPages int     `json:"total_pages"`
}

// TrackingResponse is the response from the tracking endpoint.
type TrackingResponse struct {
	TrackingNumber  string           `json:"tracking_number"`
	Service         string           `json:"service"`
	TrackingDetails []TrackingDetail `json:"tracking_details"`
}

// TrackingDetail represents a single tracking event from InPost.
type TrackingDetail struct {
	Status       string `json:"status"`
	OriginStatus string `json:"origin_status"`
	Datetime     string `json:"datetime"`
	Agency       string `json:"agency"`
}
