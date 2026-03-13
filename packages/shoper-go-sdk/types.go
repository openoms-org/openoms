package shoper

// Product represents a product from the Shoper API.
type Product struct {
	ID           int                           `json:"product_id"`
	Code         string                        `json:"code"` // SKU
	EAN          string                        `json:"ean"`
	Name         string                        `json:"translations,omitempty"` // raw; see ProductTranslation
	Stock        Stock                         `json:"stock"`
	CategoryID   int                           `json:"category_id"`
	Price        float64                       `json:"price"`
	Translations map[string]ProductTranslation `json:"all_translations,omitempty"`
}

// ProductTranslation holds localized product fields.
type ProductTranslation struct {
	Name             string `json:"name"`
	ShortDescription string `json:"short_description"`
	Description      string `json:"description"`
}

// Stock represents stock information for a product.
type Stock struct {
	Stock    float64 `json:"stock"`
	Price    float64 `json:"price"`
	PriceNet float64 `json:"comp_priceGross,omitempty"`
}

// Order represents an order from the Shoper API.
type Order struct {
	ID              int         `json:"order_id"`
	OrderSerial     string      `json:"order_serial_number"`
	StatusID        int         `json:"status_id"`
	Email           string      `json:"email"`
	Phone           string      `json:"phone"`
	Currency        string      `json:"currency_id"`
	Sum             float64     `json:"sum"`
	PaymentID       int         `json:"payment_id"`
	ShippingID      int         `json:"shipping_id"`
	DateAdd         string      `json:"date_add"`
	DateModify      string      `json:"date_modify"`
	DeliveryAddress Address     `json:"delivery_address"`
	BillingAddress  Address     `json:"billing_address"`
	Products        []OrderItem `json:"products"`
	Notes           string      `json:"notes"`
	Paid            float64     `json:"paid"`
}

// Address represents an address in a Shoper order.
type Address struct {
	FirstName string `json:"firstname"`
	LastName  string `json:"lastname"`
	Company   string `json:"company"`
	Street1   string `json:"street1"`
	Street2   string `json:"street2"`
	City      string `json:"city"`
	PostCode  string `json:"postcode"`
	Country   string `json:"country"`
	Phone     string `json:"phone"`
}

// OrderItem represents a line item in a Shoper order.
type OrderItem struct {
	ProductID int     `json:"product_id"`
	Code      string  `json:"code"` // SKU
	Name      string  `json:"name"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"`
	Tax       float64 `json:"tax"`
}

// Category represents a category from the Shoper API.
type Category struct {
	ID       int    `json:"category_id"`
	ParentID int    `json:"parent_id"`
	Name     string `json:"name"`
}

// ListResponse represents a paginated response from the Shoper API.
type ListResponse[T any] struct {
	Count int `json:"count"`
	Pages int `json:"pages"`
	Page  int `json:"page"`
	List  []T `json:"list"`
}
