package erli

// OrdersResponse is the response from the Erli orders endpoint.
type OrdersResponse struct {
	Data []Order `json:"data"`
	Meta struct {
		NextCursor string `json:"next_cursor"`
	} `json:"meta"`
}

// Order represents an Erli marketplace order.
type Order struct {
	ID            string      `json:"id"`
	Status        string      `json:"status"`
	CreatedAt     string      `json:"created_at"`
	BuyerName     string      `json:"buyer_name"`
	BuyerEmail    string      `json:"buyer_email"`
	BuyerPhone    string      `json:"buyer_phone"`
	Address       Address     `json:"delivery_address"`
	TotalAmount   float64     `json:"total_amount"`
	Currency      string      `json:"currency"`
	PaymentStatus string      `json:"payment_status"`
	Items         []OrderItem `json:"items"`
	// Cursor is the per-order pagination cursor used as the next pagination.after value.
	// Per Erli docs: use the cursor from the last order in the response to paginate.
	Cursor string `json:"cursor"`
}

// Address represents a delivery address.
type Address struct {
	Name     string `json:"name"`
	Street   string `json:"street"`
	City     string `json:"city"`
	PostCode string `json:"post_code"`
	Country  string `json:"country"`
}

// OrderItem represents a single item in an Erli order.
type OrderItem struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	SKU      string  `json:"sku"`
	Quantity int     `json:"quantity"`
	Price    float64 `json:"unit_price"`
}

// CreateOfferRequest is the request body for creating a new Erli offer.
type CreateOfferRequest struct {
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Price       float64  `json:"price"`
	Stock       int      `json:"stock"`
	SKU         string   `json:"sku,omitempty"`
	EAN         string   `json:"ean,omitempty"`
	CategoryID  string   `json:"category_id,omitempty"`
	Images      []string `json:"images,omitempty"`
}

// CreateOfferResponse is the response body after creating an Erli offer.
type CreateOfferResponse struct {
	ID string `json:"id"`
}
