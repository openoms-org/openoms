package kaufland

// OrderUnitListResponse is the response from the Kaufland order units endpoint.
type OrderUnitListResponse struct {
	Data       []OrderUnit `json:"data"`
	Pagination Pagination  `json:"pagination"`
}

// Pagination contains pagination metadata from the Kaufland API.
type Pagination struct {
	Offset     int `json:"offset"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
}

// OrderUnit represents an order unit from the Kaufland Seller API v2.
// An order unit is the smallest unit of an order, corresponding to a single item.
type OrderUnit struct {
	IDOrderUnit  int64        `json:"id_order_unit"`
	IDOrder      int64        `json:"id_order"`
	Status       string       `json:"status"`
	CreatedAt    string       `json:"ts_created_iso"`
	UpdatedAt    string       `json:"ts_updated_iso"`
	Item         Item         `json:"item"`
	Buyer        Buyer        `json:"buyer"`
	ShippingAddr Address      `json:"shipping_address"`
	BillingAddr  Address      `json:"billing_address"`
	Price        float64      `json:"price"`
	Revenue      float64      `json:"revenue_gross"`
	Quantity     int          `json:"quantity"`
	Currency     string       `json:"currency"`
}

// Item contains details about the product in an order unit.
type Item struct {
	IDItem     int64  `json:"id_item"`
	Title      string `json:"title"`
	EAN        string `json:"ean"`
	IDOffer    string `json:"id_offer"`
}

// Buyer holds buyer details.
type Buyer struct {
	Email string `json:"email"`
}

// Address represents a postal address in the Kaufland API.
type Address struct {
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	Street       string `json:"street"`
	HouseNumber  string `json:"house_number"`
	City         string `json:"city"`
	PostalCode   string `json:"postcode"`
	Country      string `json:"country"`
	Phone        string `json:"phone,omitempty"`
	Company      string `json:"company,omitempty"`
}
