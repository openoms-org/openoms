package shopify

// ShopifyProduct represents a product from the Shopify Admin API.
type ShopifyProduct struct {
	ID          int64            `json:"id"`
	Title       string           `json:"title"`
	BodyHTML    string           `json:"body_html"`
	Vendor      string           `json:"vendor"`
	ProductType string           `json:"product_type"`
	Handle      string           `json:"handle"`
	Status      string           `json:"status"` // "active", "archived", "draft"
	Tags        string           `json:"tags"`
	Variants    []ShopifyVariant `json:"variants"`
	Images      []ShopifyImage   `json:"images"`
	CreatedAt   string           `json:"created_at"`
	UpdatedAt   string           `json:"updated_at"`
}

// ShopifyVariant represents a product variant.
type ShopifyVariant struct {
	ID                 int64   `json:"id"`
	ProductID          int64   `json:"product_id"`
	Title              string  `json:"title"`
	SKU                string  `json:"sku"`
	Barcode            string  `json:"barcode"`
	Price              string  `json:"price"`
	CompareAtPrice     *string `json:"compare_at_price"`
	InventoryItemID    int64   `json:"inventory_item_id"`
	InventoryQuantity  int     `json:"inventory_quantity"`
	InventoryPolicy    string  `json:"inventory_policy"` // "deny" or "continue"
	Weight             float64 `json:"weight"`
	WeightUnit         string  `json:"weight_unit"`
	Option1            string  `json:"option1"`
	Option2            string  `json:"option2"`
	Option3            string  `json:"option3"`
}

// ShopifyImage represents a product image.
type ShopifyImage struct {
	ID        int64  `json:"id"`
	ProductID int64  `json:"product_id"`
	Src       string `json:"src"`
	Position  int    `json:"position"`
}

// ShopifyOrder represents an order from the Shopify Admin API.
type ShopifyOrder struct {
	ID                  int64               `json:"id"`
	Name                string              `json:"name"` // e.g. "#1001"
	Email               string              `json:"email"`
	Phone               string              `json:"phone"`
	FinancialStatus     string              `json:"financial_status"`
	FulfillmentStatus   *string             `json:"fulfillment_status"`
	Currency            string              `json:"currency"`
	TotalPrice          string              `json:"total_price"`
	SubtotalPrice       string              `json:"subtotal_price"`
	TotalTax            string              `json:"total_tax"`
	TotalShippingPrice  string              `json:"total_shipping_price_set,omitempty"`
	Gateway             string              `json:"gateway"`
	LineItems           []ShopifyLineItem   `json:"line_items"`
	ShippingAddress     *ShopifyAddress     `json:"shipping_address"`
	BillingAddress      *ShopifyAddress     `json:"billing_address"`
	Customer            *ShopifyCustomer    `json:"customer"`
	Note                string              `json:"note"`
	Tags                string              `json:"tags"`
	CreatedAt           string              `json:"created_at"`
	UpdatedAt           string              `json:"updated_at"`
	ProcessedAt         string              `json:"processed_at"`
	CancelledAt         *string             `json:"cancelled_at"`
}

// ShopifyLineItem represents a single line item in a Shopify order.
type ShopifyLineItem struct {
	ID          int64   `json:"id"`
	ProductID   *int64  `json:"product_id"`
	VariantID   *int64  `json:"variant_id"`
	Title       string  `json:"title"`
	Name        string  `json:"name"`
	SKU         string  `json:"sku"`
	Quantity    int     `json:"quantity"`
	Price       string  `json:"price"`
	TotalDiscount string `json:"total_discount"`
}

// ShopifyAddress represents a Shopify address.
type ShopifyAddress struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Company   string `json:"company"`
	Address1  string `json:"address1"`
	Address2  string `json:"address2"`
	City      string `json:"city"`
	Zip       string `json:"zip"`
	Country   string `json:"country"`
	CountryCode string `json:"country_code"`
	Phone     string `json:"phone"`
}

// ShopifyCustomer represents a customer embedded in a Shopify order.
type ShopifyCustomer struct {
	ID        int64  `json:"id"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Phone     string `json:"phone"`
}

// InventoryLevel represents a Shopify inventory level at a location.
type InventoryLevel struct {
	InventoryItemID int64 `json:"inventory_item_id"`
	LocationID      int64 `json:"location_id"`
	Available       int   `json:"available"`
	UpdatedAt       string `json:"updated_at"`
}
