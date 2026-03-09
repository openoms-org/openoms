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
	OrderID          string          `json:"orderId"`
	LegacyOrderID    string          `json:"legacyOrderId"`
	CreationDate     string          `json:"creationDate"`
	LastModifiedDate string          `json:"lastModifiedDate"`
	OrderFulfStatus  string          `json:"orderFulfillmentStatus"`
	OrderPaymentStat string          `json:"orderPaymentStatus"`
	PricingSummary   PricingSummary  `json:"pricingSummary"`
	Buyer            BuyerInfo       `json:"buyer"`
	FulfillmentSOs   []FulfillmentSO `json:"fulfillmentStartInstructions"`
	LineItems        []LineItem      `json:"lineItems"`
	SalesRecordRef   string          `json:"salesRecordReference,omitempty"`
	CancelStatus     *CancelStatus   `json:"cancelStatus,omitempty"`
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
	Username     string        `json:"username"`
	TaxAddress   *TaxAddress   `json:"taxAddress,omitempty"`
	BuyerRegInfo *BuyerRegInfo `json:"buyerRegistrationAddress,omitempty"`
}

// TaxAddress is a simplified address used for tax purposes.
type TaxAddress struct {
	StateOrProvince string `json:"stateOrProvince"`
	PostalCode      string `json:"postalCode"`
	CountryCode     string `json:"countryCode"`
}

// BuyerRegInfo holds the buyer's registration address.
type BuyerRegInfo struct {
	FullName       string          `json:"fullName"`
	ContactAddress *ContactAddress `json:"contactAddress,omitempty"`
	Email          string          `json:"email,omitempty"`
	PrimaryPhone   *Phone          `json:"primaryPhone,omitempty"`
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
	LineItemID   string `json:"lineItemId"`
	LegacyItemID string `json:"legacyItemId"`
	Title        string `json:"title"`
	SKU          string `json:"sku,omitempty"`
	Quantity     int    `json:"quantity"`
	LineItemCost Amount `json:"lineItemCost"`
	Total        Amount `json:"total"`
}

// CancelStatus contains cancellation information.
type CancelStatus struct {
	CancelState    string `json:"cancelState"`
	CancelRequests []any  `json:"cancelRequests,omitempty"`
}

// ShippingFulfillmentRequest is the request body for creating a shipping fulfillment.
type ShippingFulfillmentRequest struct {
	LineItems           []FulfillmentLineItem `json:"lineItems"`
	ShippedDate         string                `json:"shippedDate,omitempty"`
	ShippingCarrierCode string                `json:"shippingCarrierCode"`
	TrackingNumber      string                `json:"trackingNumber"`
}

// FulfillmentLineItem identifies a line item in a fulfillment request.
type FulfillmentLineItem struct {
	LineItemID string `json:"lineItemId"`
	Quantity   int    `json:"quantity"`
}

// ShippingFulfillment is the response from creating or getting a shipping fulfillment.
type ShippingFulfillment struct {
	FulfillmentID          string                `json:"fulfillmentId"`
	ShipmentTrackingNumber string                `json:"shipmentTrackingNumber"`
	ShippingCarrierCode    string                `json:"shippingCarrierCode"`
	ShippedDate            string                `json:"shippedDate"`
	LineItems              []FulfillmentLineItem `json:"lineItems"`
}

// ShippingFulfillmentPagedCollection is the response for listing shipping fulfillments.
type ShippingFulfillmentPagedCollection struct {
	Fulfillments []ShippingFulfillment `json:"fulfillments"`
	Total        int                   `json:"total"`
}

// InventoryItem represents an eBay inventory item.
type InventoryItem struct {
	SKU                  string           `json:"sku,omitempty"`
	Locale               string           `json:"locale,omitempty"`
	Product              InventoryProduct `json:"product"`
	Condition            string           `json:"condition,omitempty"`
	ConditionDescription string           `json:"conditionDescription,omitempty"`
	Availability         *Availability    `json:"availability,omitempty"`
}

// InventoryProduct holds the product details for an inventory item.
type InventoryProduct struct {
	Title       string              `json:"title"`
	Description string              `json:"description,omitempty"`
	Aspects     map[string][]string `json:"aspects,omitempty"`
	Brand       string              `json:"brand,omitempty"`
	MPN         string              `json:"mpn,omitempty"`
	EAN         []string            `json:"ean,omitempty"`
	ImageURLs   []string            `json:"imageUrls,omitempty"`
}

// Availability holds stock information.
type Availability struct {
	ShipToLocationAvailability *ShipToLocationAvailability `json:"shipToLocationAvailability,omitempty"`
}

// ShipToLocationAvailability holds the available quantity.
type ShipToLocationAvailability struct {
	Quantity int `json:"quantity"`
}

// InventoryItemsResponse is the response from listing inventory items.
type InventoryItemsResponse struct {
	InventoryItems []InventoryItem `json:"inventoryItems"`
	Total          int             `json:"total"`
	Size           int             `json:"size"`
	Href           string          `json:"href"`
	Next           string          `json:"next,omitempty"`
}

// Offer represents an eBay offer.
type Offer struct {
	OfferID            string           `json:"offerId,omitempty"`
	SKU                string           `json:"sku"`
	MarketplaceID      string           `json:"marketplaceId"`
	Format             string           `json:"format"`
	ListingDescription string           `json:"listingDescription,omitempty"`
	AvailableQuantity  int              `json:"availableQuantity,omitempty"`
	CategoryID         string           `json:"categoryId"`
	PricingSummary     *OfferPricing    `json:"pricingSummary,omitempty"`
	ListingPolicies    *ListingPolicies `json:"listingPolicies,omitempty"`
	Status             string           `json:"status,omitempty"`
	Listing            *OfferListing    `json:"listing,omitempty"`
}

// OfferPricing holds offer pricing.
type OfferPricing struct {
	Price Amount `json:"price"`
}

// ListingPolicies holds the business policy IDs for an offer.
type ListingPolicies struct {
	FulfillmentPolicyID string `json:"fulfillmentPolicyId"`
	ReturnPolicyID      string `json:"returnPolicyId"`
	PaymentPolicyID     string `json:"paymentPolicyId,omitempty"`
}

// OfferListing holds published listing info.
type OfferListing struct {
	ListingID string `json:"listingId,omitempty"`
}

// OffersResponse is the response from listing offers.
type OffersResponse struct {
	Offers []Offer `json:"offers"`
	Total  int     `json:"total"`
	Size   int     `json:"size"`
	Next   string  `json:"next,omitempty"`
}

// PublishResponse is the response from publishing an offer.
type PublishResponse struct {
	ListingID string `json:"listingId"`
}

// CreateOfferResponse is returned when creating an offer.
type CreateOfferResponse struct {
	OfferID string `json:"offerId"`
}

// FulfillmentPolicy represents an eBay fulfillment (shipping) policy.
type FulfillmentPolicy struct {
	FulfillmentPolicyID string `json:"fulfillmentPolicyId"`
	Name                string `json:"name"`
	MarketplaceID       string `json:"marketplaceId"`
	Description         string `json:"description,omitempty"`
}

// FulfillmentPoliciesResponse is the response from listing fulfillment policies.
type FulfillmentPoliciesResponse struct {
	FulfillmentPolicies []FulfillmentPolicy `json:"fulfillmentPolicies"`
	Total               int                 `json:"total"`
}

// ReturnPolicy represents an eBay return policy.
type ReturnPolicy struct {
	ReturnPolicyID string `json:"returnPolicyId"`
	Name           string `json:"name"`
	MarketplaceID  string `json:"marketplaceId"`
	Description    string `json:"description,omitempty"`
}

// ReturnPoliciesResponse is the response from listing return policies.
type ReturnPoliciesResponse struct {
	ReturnPolicies []ReturnPolicy `json:"returnPolicies"`
	Total          int            `json:"total"`
}

// PaymentPolicy represents an eBay payment policy.
type PaymentPolicy struct {
	PaymentPolicyID string `json:"paymentPolicyId"`
	Name            string `json:"name"`
	MarketplaceID   string `json:"marketplaceId"`
	Description     string `json:"description,omitempty"`
}

// PaymentPoliciesResponse is the response from listing payment policies.
type PaymentPoliciesResponse struct {
	PaymentPolicies []PaymentPolicy `json:"paymentPolicies"`
	Total           int             `json:"total"`
}
