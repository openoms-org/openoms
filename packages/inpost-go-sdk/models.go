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
