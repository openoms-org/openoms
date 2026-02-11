package ebay

// OrderSearchResponse is the response from the Fulfillment API order search endpoint.
type OrderSearchResponse struct {
	Href   string  `json:"href"`
	Total  int     `json:"total"`
	Limit  int     `json:"limit"`
	Offset int     `json:"offset"`
	Orders []Order `json:"orders"`
	Next   string  `json:"next,omitempty"`
}

// Order represents an eBay order from the Fulfillment API v1.
type Order struct {
	OrderID          string           `json:"orderId"`
	LegacyOrderID    string           `json:"legacyOrderId"`
	CreationDate     string           `json:"creationDate"`
	LastModifiedDate string           `json:"lastModifiedDate"`
	OrderFulfStatus  string           `json:"orderFulfillmentStatus"`
	OrderPaymentStat string           `json:"orderPaymentStatus"`
	PricingSummary   PricingSummary   `json:"pricingSummary"`
	Buyer            BuyerInfo        `json:"buyer"`
	FulfillmentSOs   []FulfillmentSO  `json:"fulfillmentStartInstructions"`
	LineItems        []LineItem       `json:"lineItems"`
	SalesRecordRef   string           `json:"salesRecordReference,omitempty"`
	CancelStatus     *CancelStatus    `json:"cancelStatus,omitempty"`
}

// PricingSummary contains order-level price totals.
type PricingSummary struct {
	Total            Amount `json:"total"`
	Subtotal         Amount `json:"priceSubtotal"`
	DeliveryCost     Amount `json:"deliveryCost"`
	Tax              Amount `json:"tax"`
	PriceDiscount    Amount `json:"priceDiscount"`
	DeliveryDiscount Amount `json:"deliveryDiscount"`
}

// Amount represents a monetary value with currency.
type Amount struct {
	Value        string `json:"value"`
	Currency     string `json:"currency"`
	ConvertedVal string `json:"convertedFromValue,omitempty"`
	ConvertedCur string `json:"convertedFromCurrency,omitempty"`
}

// BuyerInfo holds the buyer details.
type BuyerInfo struct {
	Username      string         `json:"username"`
	TaxAddress    *TaxAddress    `json:"taxAddress,omitempty"`
	BuyerRegInfo  *BuyerRegInfo  `json:"buyerRegistrationAddress,omitempty"`
}

// TaxAddress is a simplified address used for tax purposes.
type TaxAddress struct {
	StateOrProvince string `json:"stateOrProvince"`
	PostalCode      string `json:"postalCode"`
	CountryCode     string `json:"countryCode"`
}

// BuyerRegInfo holds the buyer's registration address.
type BuyerRegInfo struct {
	FullName        string          `json:"fullName"`
	ContactAddress  *ContactAddress `json:"contactAddress,omitempty"`
	Email           string          `json:"email,omitempty"`
	PrimaryPhone    *Phone          `json:"primaryPhone,omitempty"`
}

// ContactAddress holds structured address fields.
type ContactAddress struct {
	AddressLine1    string `json:"addressLine1"`
	AddressLine2    string `json:"addressLine2,omitempty"`
	City            string `json:"city"`
	StateOrProvince string `json:"stateOrProvince"`
	PostalCode      string `json:"postalCode"`
	CountryCode     string `json:"countryCode"`
}

// Phone represents a phone number.
type Phone struct {
	PhoneNumber string `json:"phoneNumber"`
}

// FulfillmentSO contains fulfillment start instructions including the shipping address.
type FulfillmentSO struct {
	ShippingStep ShippingStep `json:"shippingStep"`
}

// ShippingStep contains the ship-to address.
type ShippingStep struct {
	ShipTo ShipTo `json:"shipTo"`
}

// ShipTo is the shipping destination.
type ShipTo struct {
	FullName       string          `json:"fullName"`
	ContactAddress *ContactAddress `json:"contactAddress,omitempty"`
	Email          string          `json:"email,omitempty"`
	PrimaryPhone   *Phone          `json:"primaryPhone,omitempty"`
}

// LineItem represents a single line item in an eBay order.
type LineItem struct {
	LineItemID    string `json:"lineItemId"`
	LegacyItemID string `json:"legacyItemId"`
	Title         string `json:"title"`
	SKU           string `json:"sku,omitempty"`
	Quantity      int    `json:"quantity"`
	LineItemCost  Amount `json:"lineItemCost"`
	Total         Amount `json:"total"`
}

// CancelStatus contains cancellation information.
type CancelStatus struct {
	CancelState     string `json:"cancelState"`
	CancelRequests  []any  `json:"cancelRequests,omitempty"`
}
