package mirakl

// OrdersResponse is the response from the Mirakl orders endpoint.
type OrdersResponse struct {
	Orders []Order `json:"orders"`
}

// Order represents a Mirakl marketplace order.
type Order struct {
	ID              string      `json:"order_id"`
	Status          string      `json:"order_state"`
	CreatedDate     string      `json:"created_date"`
	Customer        Customer    `json:"customer"`
	ShippingAddress Address     `json:"shipping_address"`
	TotalPrice      float64     `json:"total_price"`
	CurrencyCode    string      `json:"currency_iso_code"`
	PaymentType     string      `json:"payment_type"`
	OrderLines      []OrderLine `json:"order_lines"`
}

// Customer contains buyer information.
type Customer struct {
	FirstName string `json:"firstname"`
	LastName  string `json:"lastname"`
	Email     string `json:"email"`
}

// Address represents a shipping or billing address.
type Address struct {
	FirstName string `json:"firstname"`
	LastName  string `json:"lastname"`
	Street1   string `json:"street_1"`
	Street2   string `json:"street_2"`
	City      string `json:"city"`
	ZipCode   string `json:"zip_code"`
	Country   string `json:"country_iso_code"`
	Phone     string `json:"phone"`
}

// OrderLine represents a single line item in a Mirakl order.
type OrderLine struct {
	ID           string  `json:"order_line_id"`
	OfferSKU     string  `json:"offer_sku"`
	ProductTitle string  `json:"product_title"`
	Quantity     int     `json:"quantity"`
	Price        float64 `json:"price"`
}

// OfferUpdateRequest is the payload for batch offer updates.
type OfferUpdateRequest struct {
	Offers []OfferUpdate `json:"offers"`
}

// OfferUpdate represents a single offer stock/price update.
type OfferUpdate struct {
	SKU      string  `json:"shop_sku"`
	Price    float64 `json:"price,omitempty"`
	Quantity int     `json:"quantity,omitempty"`
}

// OfferResponse represents an offer retrieved from Mirakl.
type OfferResponse struct {
	Offers []OfferDetail `json:"offers"`
}

// OfferDetail contains details of a single Mirakl offer.
type OfferDetail struct {
	SKU      string  `json:"shop_sku"`
	Price    float64 `json:"price"`
	Quantity int     `json:"quantity"`
	Active   bool    `json:"active"`
}
