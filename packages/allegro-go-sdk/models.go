package allegro

import "time"

// TokenResponse represents the OAuth 2.0 token endpoint response.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
	JTI          string `json:"jti"`
}

// OrderList represents a paginated list of orders.
type OrderList struct {
	CheckoutForms []Order `json:"checkoutForms"`
	Count         int     `json:"count"`
	TotalCount    int     `json:"totalCount"`
}

// Order represents an Allegro checkout form (order).
type Order struct {
	ID         string      `json:"id"`
	Buyer      Buyer       `json:"buyer"`
	Payment    Payment     `json:"payment"`
	Status     string      `json:"status"`
	Fulfillment Fulfillment `json:"fulfillment"`
	Delivery   Delivery    `json:"delivery"`
	Invoice    Invoice     `json:"invoice"`
	LineItems  []LineItem  `json:"lineItems"`
	UpdatedAt  time.Time   `json:"updatedAt"`
}

// Buyer represents the buyer of an order.
type Buyer struct {
	ID    string `json:"id"`
	Login string `json:"login"`
	Email string `json:"email"`
}

// Payment represents payment information for an order.
type Payment struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	PaidAmount Amount `json:"paidAmount"`
}

// Amount represents a monetary value.
type Amount struct {
	Amount   string `json:"amount"`
	Currency string `json:"currency"`
}

// Fulfillment represents the fulfillment status of an order.
type Fulfillment struct {
	Status string `json:"status"`
}

// Delivery represents delivery details for an order.
type Delivery struct {
	Address Address        `json:"address"`
	Method  DeliveryMethod `json:"method"`
}

// Address represents a shipping address.
type Address struct {
	FirstName   string `json:"firstName"`
	LastName    string `json:"lastName"`
	Street      string `json:"street"`
	City        string `json:"city"`
	ZipCode     string `json:"zipCode"`
	CountryCode string `json:"countryCode"`
}

// DeliveryMethod represents the delivery method.
type DeliveryMethod struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Invoice represents invoice information.
type Invoice struct {
	Required bool `json:"required"`
}

// LineItem represents a single item in an order.
type LineItem struct {
	ID       string        `json:"id"`
	Offer    LineItemOffer `json:"offer"`
	Quantity int           `json:"quantity"`
	Price    Amount        `json:"price"`
}

// LineItemOffer represents a reference to the offer within a line item.
type LineItemOffer struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	External string `json:"external"`
}

// EventList represents a list of order events.
type EventList struct {
	Events []OrderEvent `json:"events"`
}

// OrderEvent represents a single order event.
type OrderEvent struct {
	ID         string        `json:"id"`
	Type       string        `json:"type"`
	OccurredAt string        `json:"occurredAt"`
	Order      OrderEventRef `json:"order"`
}

// OrderEventRef contains a reference to the order associated with an event.
type OrderEventRef struct {
	CheckoutForm OrderEventCheckoutForm `json:"checkoutForm"`
}

// OrderEventCheckoutForm is a minimal reference to a checkout form.
type OrderEventCheckoutForm struct {
	ID string `json:"id"`
}

// OfferList represents a paginated list of offers.
type OfferList struct {
	Offers     []OfferSummary `json:"offers"`
	Count      int            `json:"count"`
	TotalCount int            `json:"totalCount"`
}

// OfferSummary represents an offer in a list response.
type OfferSummary struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Offer represents a full offer/product detail.
type Offer struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
