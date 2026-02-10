package amazon

// OrdersResponse is the top-level response from GET /orders/v0/orders.
type OrdersResponse struct {
	Payload OrdersPayload `json:"payload"`
}

// OrdersPayload contains the orders list and pagination token.
type OrdersPayload struct {
	Orders    []Order `json:"Orders"`
	NextToken string  `json:"NextToken,omitempty"`
}

// Order represents an Amazon SP-API order.
type Order struct {
	AmazonOrderID      string    `json:"AmazonOrderId"`
	PurchaseDate       string    `json:"PurchaseDate"`
	LastUpdateDate     string    `json:"LastUpdateDate,omitempty"`
	OrderStatus        string    `json:"OrderStatus"`
	OrderTotal         *Money    `json:"OrderTotal,omitempty"`
	ShippingAddress    *Address  `json:"ShippingAddress,omitempty"`
	BuyerInfo          *BuyerInfo `json:"BuyerInfo,omitempty"`
	PaymentMethod      string    `json:"PaymentMethod,omitempty"`
	FulfillmentChannel string    `json:"FulfillmentChannel"`
	MarketplaceID      string    `json:"MarketplaceId"`
	NumberOfItemsShipped   int   `json:"NumberOfItemsShipped"`
	NumberOfItemsUnshipped int   `json:"NumberOfItemsUnshipped"`
}

// Money represents a monetary amount with currency.
type Money struct {
	Amount       string `json:"Amount"`
	CurrencyCode string `json:"CurrencyCode"`
}

// Address represents a shipping or billing address.
type Address struct {
	Name         string `json:"Name,omitempty"`
	AddressLine1 string `json:"AddressLine1,omitempty"`
	AddressLine2 string `json:"AddressLine2,omitempty"`
	City         string `json:"City,omitempty"`
	PostalCode   string `json:"PostalCode,omitempty"`
	CountryCode  string `json:"CountryCode,omitempty"`
	Phone        string `json:"Phone,omitempty"`
}

// BuyerInfo contains buyer contact information.
type BuyerInfo struct {
	BuyerEmail string `json:"BuyerEmail,omitempty"`
}

// OrderItemsResponse is the top-level response from GET /orders/v0/orders/{orderId}/orderItems.
type OrderItemsResponse struct {
	Payload OrderItemsPayload `json:"payload"`
}

// OrderItemsPayload contains the order items and pagination token.
type OrderItemsPayload struct {
	OrderItems []OrderItem `json:"OrderItems"`
	NextToken  string      `json:"NextToken,omitempty"`
}

// OrderItem represents a single item in an Amazon order.
type OrderItem struct {
	ASIN             string `json:"ASIN"`
	SellerSKU        string `json:"SellerSKU,omitempty"`
	OrderItemID      string `json:"OrderItemId"`
	Title            string `json:"Title,omitempty"`
	QuantityOrdered  int    `json:"QuantityOrdered"`
	QuantityShipped  int    `json:"QuantityShipped"`
	ItemPrice        *Money `json:"ItemPrice,omitempty"`
	ItemTax          *Money `json:"ItemTax,omitempty"`
}

// GetOrderResponse is the top-level response from GET /orders/v0/orders/{orderId}.
type GetOrderResponse struct {
	Payload Order `json:"payload"`
}

// CatalogItemResponse is the top-level response from GET /catalog/2022-04-01/items/{asin}.
type CatalogItemResponse struct {
	ASIN       string        `json:"asin"`
	Summaries  []ItemSummary `json:"summaries,omitempty"`
}

// ItemSummary contains basic catalog item information.
type ItemSummary struct {
	MarketplaceID string `json:"marketplaceId"`
	ItemName      string `json:"itemName,omitempty"`
	BrandName     string `json:"brandName,omitempty"`
}

// TokenResponse represents the LWA OAuth2 token endpoint response.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

// APIError represents an error response from the Amazon SP-API.
type APIError struct {
	StatusCode int      `json:"-"`
	Errors     []SPError `json:"errors,omitempty"`
}

// SPError represents a single error entry in an Amazon SP-API error response.
type SPError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

func (e *APIError) Error() string {
	if len(e.Errors) > 0 {
		msg := "amazon: api error " + e.Errors[0].Code + ": " + e.Errors[0].Message
		return msg
	}
	return "amazon: api error"
}
