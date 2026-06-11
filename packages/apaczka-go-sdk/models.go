package apaczka

// Service represents a single shipping service offered by the Apaczka broker.
type Service struct {
	ServiceID    string `json:"service_id"`
	Name         string `json:"name"`
	Supplier     string `json:"supplier"`
	DeliveryTime string `json:"delivery_time"`
	Domestic     string `json:"domestic"`       // "0" or "1"
	DoorToDoor   string `json:"door_to_door"`   // "0" or "1"
	DoorToPoint  string `json:"door_to_point"`  // "0" or "1"
	PointToPoint string `json:"point_to_point"` // "0" or "1"
	PointToDoor  string `json:"point_to_door"`  // "0" or "1"
}

// ServiceStructureResponse is the decoded response from the service_structure/ route.
type ServiceStructureResponse struct {
	Services []Service `json:"services"`
}

// Address holds a sender or receiver address.
type Address struct {
	CountryCode string `json:"country_code"`
	Name        string `json:"name"`
	Line1       string `json:"line1"`
	PostalCode  string `json:"postal_code"`
	City        string `json:"city"`
	Phone       string `json:"phone,omitempty"`
	Email       string `json:"email,omitempty"`
}

// AddressPair groups a sender and receiver address.
type AddressPair struct {
	Sender   Address `json:"sender"`
	Receiver Address `json:"receiver"`
}

// ShipmentItem describes a single parcel within an order.
type ShipmentItem struct {
	// Dimensions in centimetres (integers per API spec).
	Dimension1 int `json:"dimension1"`
	Dimension2 int `json:"dimension2"`
	Dimension3 int `json:"dimension3"`

	// Weight in kilograms.
	Weight float64 `json:"weight"`

	// ShipmentTypeCode e.g. "PACZKA".
	ShipmentTypeCode string `json:"shipment_type_code"`
}

// COD holds cash-on-delivery parameters. Omit from the order payload when not needed.
type COD struct {
	// Amount in grosze (Polish subunit).
	Amount      int    `json:"amount"`
	Currency    string `json:"currency"`
	BankAccount string `json:"bankaccount"`
	Country     string `json:"country"`
}

// OrderInput is the "order" field inside a valuation or creation request.
type OrderInput struct {
	ServiceID string         `json:"service_id"`
	Address   AddressPair    `json:"address"`
	Shipment  []ShipmentItem `json:"shipment"`

	// Pickup describes how/when the carrier collects the parcel (used in order_send/).
	Pickup *Pickup `json:"pickup,omitempty"`

	// COD is optional; omit entirely when not needed.
	COD *COD `json:"cod,omitempty"`
}

// Pickup describes the collection method for an order.
type Pickup struct {
	// Type is "SELF" (self-drop) or "COURIER" (courier pickup).
	Type string `json:"type"`
	// Date in "YYYY-MM-DD" format.
	Date string `json:"date,omitempty"`
}

// ValuationRequest is the payload sent to order_valuation/.
type ValuationRequest struct {
	Order OrderInput `json:"order"`
}

// PriceRow holds gross and net prices returned by order_valuation/ for a single service.
// Values are strings in grosze (Polish sub-unit), e.g. "1299" = 12.99 PLN.
type PriceRow struct {
	Price      string `json:"price"`
	PriceGross string `json:"price_gross"`
}

// ValuationResponse is the decoded response from order_valuation/.
// PriceTable is keyed by service_id.
type ValuationResponse struct {
	PriceTable map[string]PriceRow `json:"price_table"`
}

// CreateOrderRequest is the payload sent to order_send/.
type CreateOrderRequest struct {
	Order OrderInput `json:"order"`
}

// Order is the order resource returned inside order_send/ and order/<id>/ responses.
type Order struct {
	ID            string `json:"id"`
	ServiceID     string `json:"service_id"`
	ServiceName   string `json:"service_name"`
	WaybillNumber string `json:"waybill_number"`
	TrackingURL   string `json:"tracking_url"`
	Status        string `json:"status"`
}

// createOrderResponse is the top-level "response" envelope from order_send/.
type createOrderResponse struct {
	Order Order `json:"order"`
}

// getOrderResponse is the top-level "response" envelope from order/<id>/.
type getOrderResponse struct {
	Order Order `json:"order"`
}

// TrackEvent is a single tracking event from tracking/<waybill>/.
type TrackEvent struct {
	Status         string `json:"status"`
	StatusOriginal string `json:"status_original"` // nullable — empty string if absent
	Description    string `json:"description"`
	Place          string `json:"place"`
	// UpdatedAt is an ISO-8601 UTC string; may be empty if the API returns null.
	UpdatedAt string `json:"updated_at"`
}

// TrackingResponse is the decoded response from tracking/<waybill_number>/.
type TrackingResponse struct {
	Tracking []TrackEvent `json:"tracking"`
}

// Point represents a pickup/drop-off point (e.g. InPost locker, UPS Access Point).
type Point struct {
	// ID is the point identifier; some carriers use "foreign_access_point_id".
	ID   string `json:"id"`
	Name string `json:"name"`

	// Address fields.
	Line1      string `json:"line1"`
	PostalCode string `json:"postal_code"`
	City       string `json:"city"`
	Country    string `json:"country"`

	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

// pointsResponse is the top-level "response" envelope from points/<type>/.
type pointsResponse struct {
	Points []Point `json:"points"`
}

// pointsRequest is the payload for the points/<type>/ route.
type pointsRequest struct {
	CountryCode string `json:"country_code"`
	Subtype     string `json:"subtype,omitempty"`
}

// waybillResponse is the top-level "response" envelope from waybill/<order_id>/.
type waybillResponse struct {
	// Waybill is the shipping label encoded as a base64 string.
	Waybill string `json:"waybill"`
	// Type is the document format, e.g. "pdf".
	Type string `json:"type"`
}
