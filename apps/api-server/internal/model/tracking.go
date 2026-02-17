package model

import "time"

// TrackingResponse is the public-facing order tracking response.
// It exposes only customer-safe data (no internal IDs, no admin notes).
type TrackingResponse struct {
	OrderNumber  string             `json:"order_number"`
	Status       string             `json:"status"`
	StatusLabel  string             `json:"status_label"`
	CustomerName string             `json:"customer_name"`
	CreatedAt    time.Time          `json:"created_at"`
	UpdatedAt    time.Time          `json:"updated_at"`
	TotalAmount  float64            `json:"total_amount"`
	Currency     string             `json:"currency"`
	Items        []TrackingItem     `json:"items"`
	Shipments    []TrackingShipment `json:"shipments"`
	Timeline     []TrackingEvent    `json:"timeline"`
	CompanyName  string             `json:"company_name,omitempty"`
	CompanyLogo  string             `json:"company_logo,omitempty"`
}

// TrackingItem is a public-safe order line item.
type TrackingItem struct {
	Name     string  `json:"name"`
	Quantity int     `json:"quantity"`
	Price    float64 `json:"price"`
}

// TrackingShipment is a public-safe shipment summary.
type TrackingShipment struct {
	TrackingNumber string `json:"tracking_number"`
	Carrier        string `json:"carrier"`
	Status         string `json:"status"`
}

// TrackingEvent is a status change event in the order timeline.
type TrackingEvent struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}
