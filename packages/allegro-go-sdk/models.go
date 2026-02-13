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
	ID          string      `json:"id"`
	Buyer       Buyer       `json:"buyer"`
	Payment     Payment     `json:"payment"`
	Status      string      `json:"status"`
	Fulfillment Fulfillment `json:"fulfillment"`
	Delivery    Delivery    `json:"delivery"`
	Invoice     Invoice     `json:"invoice"`
	LineItems   []LineItem  `json:"lineItems"`
	UpdatedAt   time.Time   `json:"updatedAt"`
}

// Buyer represents the buyer of an order.
type Buyer struct {
	ID    string      `json:"id"`
	Login string      `json:"login"`
	Email string      `json:"email"`
	Phone *BuyerPhone `json:"phone,omitempty"`
}

// BuyerPhone represents the buyer's phone number.
type BuyerPhone struct {
	Number string `json:"number"`
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
	Address     Address        `json:"address"`
	Method      DeliveryMethod `json:"method"`
	PickupPoint *PickupPoint   `json:"pickupPoint,omitempty"`
}

// PickupPoint represents an Allegro pickup/drop-off point.
type PickupPoint struct {
	ID      string         `json:"id"`
	Name    string         `json:"name"`
	Address *PickupAddress `json:"address,omitempty"`
}

// PickupAddress represents the address of a pickup point.
type PickupAddress struct {
	Street  string `json:"street"`
	City    string `json:"city"`
	ZipCode string `json:"zipCode"`
}

// Address represents a shipping address.
type Address struct {
	FirstName   string `json:"firstName"`
	LastName    string `json:"lastName"`
	Street      string `json:"street"`
	City        string `json:"city"`
	ZipCode     string `json:"zipCode"`
	CountryCode string `json:"countryCode"`
	Company     string `json:"companyName,omitempty"`
	Phone       string `json:"phoneNumber,omitempty"`
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
	ID            string        `json:"id"`
	Offer         LineItemOffer `json:"offer"`
	Quantity      int           `json:"quantity"`
	Price         Amount        `json:"price"`
	OriginalPrice *Amount       `json:"originalPrice,omitempty"`
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
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	SellingMode  *OfferSellingMode `json:"sellingMode,omitempty"`
	Stock        *OfferStock       `json:"stock,omitempty"`
	Publication  *OfferPublication `json:"publication,omitempty"`
	PrimaryImage *OfferImage       `json:"primaryImage,omitempty"`
}

// Offer represents a full offer/product detail.
type Offer struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Category     *OfferCategory    `json:"category,omitempty"`
	SellingMode  *OfferSellingMode `json:"sellingMode,omitempty"`
	Stock        *OfferStock       `json:"stock,omitempty"`
	Publication  *OfferPublication `json:"publication,omitempty"`
	PrimaryImage *OfferImage       `json:"primaryImage,omitempty"`
	Description  *OfferDescription `json:"description,omitempty"`
	External     *OfferExternal    `json:"external,omitempty"`
}

// OfferSellingMode represents the selling mode and price of an offer.
type OfferSellingMode struct {
	Price  Amount `json:"price"`
	Format string `json:"format"`
}

// OfferStock represents the stock information of an offer.
type OfferStock struct {
	Available int    `json:"available"`
	Unit      string `json:"unit"`
}

// OfferPublication represents the publication status of an offer.
type OfferPublication struct {
	Status string `json:"status"` // ACTIVE, INACTIVE, ENDED
}

// OfferImage represents an image associated with an offer.
type OfferImage struct {
	URL string `json:"url"`
}

// OfferCategory represents the category of an offer.
type OfferCategory struct {
	ID string `json:"id"`
}

// OfferDescription represents a structured offer description.
type OfferDescription struct {
	Sections []DescriptionSection `json:"sections"`
}

// DescriptionSection represents a section within an offer description.
type DescriptionSection struct {
	Items []DescriptionItem `json:"items"`
}

// DescriptionItem represents a single item within a description section.
type DescriptionItem struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

// OfferExternal represents external ID mapping for an offer.
type OfferExternal struct {
	ID string `json:"id"`
}

// StockUpdate represents a stock quantity change for a single offer.
type StockUpdate struct {
	OfferID  string `json:"offerId"`
	Quantity int    `json:"quantity"`
}

// PriceUpdate represents a price change for a single offer.
type PriceUpdate struct {
	OfferID  string  `json:"offerId"`
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

// User represents an Allegro user account.
type User struct {
	ID       string   `json:"id"`
	Login    string   `json:"login"`
	Email    string   `json:"email"`
	Features []string `json:"features"`
}

// SellerQuality represents the seller's quality metrics.
type SellerQuality struct {
	RecommendPercentage string `json:"recommendPercentage"`
	RecommendCount      int    `json:"recommendCount"`
}

// SmartStatus represents the seller's Smart! eligibility status.
type SmartStatus struct {
	Eligible    bool   `json:"eligible"`
	Marketplace string `json:"marketplace"`
}

// RatingsParams are the optional parameters for listing user ratings.
type RatingsParams struct {
	Limit  int
	Offset int
}

// RatingList represents a paginated list of user ratings.
type RatingList struct {
	Ratings []UserRating `json:"ratings"`
	Count   int          `json:"count"`
}

// UserRating represents a single rating from a buyer.
type UserRating struct {
	ID        string      `json:"id"`
	Rate      string      `json:"rate"`
	Comment   string      `json:"comment"`
	CreatedAt string      `json:"createdAt"`
	Buyer     RatingBuyer `json:"buyer"`
	Order     RatingOrder `json:"order"`
}

// RatingBuyer represents the buyer who left a rating.
type RatingBuyer struct {
	Login string `json:"login"`
}

// RatingOrder represents the order associated with a rating.
type RatingOrder struct {
	ID string `json:"id"`
}

// BillingParams are the optional parameters for listing billing entries.
type BillingParams struct {
	Limit     int
	Offset    int
	TypeGroup string
}

// BillingList represents a paginated list of billing entries.
type BillingList struct {
	BillingEntries []BillingEntry `json:"billingEntries"`
	Count          int            `json:"count"`
}

// BillingEntry represents a single billing entry.
type BillingEntry struct {
	ID         string      `json:"id"`
	Type       BillingType `json:"type"`
	Amount     Amount      `json:"amount"`
	OccurredAt string      `json:"occurredAt"`
}

// BillingType describes the type of a billing entry.
type BillingType struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Group string `json:"group"`
}

// FulfillmentUpdate is the request body for updating fulfillment status.
type FulfillmentUpdate struct {
	Status string `json:"status"`
}

// ShipmentInput is the request body for adding a shipment to an order.
type ShipmentInput struct {
	CarrierID string `json:"carrierId"`
	Waybill   string `json:"waybill"`
}

// OrderShipment represents a shipment associated with an order.
type OrderShipment struct {
	CarrierID string    `json:"carrierId"`
	Waybill   string    `json:"waybill"`
	CreatedAt time.Time `json:"createdAt"`
}

// ShipmentList is the response for listing order shipments.
type ShipmentList struct {
	Shipments []OrderShipment `json:"shipments"`
}

// Carrier represents a shipping carrier in Allegro.
type Carrier struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// CarrierList is the response for listing carriers.
type CarrierList struct {
	Carriers []Carrier `json:"carriers"`
}

// EventStatsResponse represents the response from GET /order/event-stats.
type EventStatsResponse struct {
	LatestEvent EventRef `json:"latestEvent"`
}

// EventRef is a reference to an event with its ID and occurred-at timestamp.
type EventRef struct {
	ID         string `json:"id"`
	OccurredAt string `json:"occurredAt"`
}

// --- Shipment Management ("Wysyłam z Allegro") models ---

// DeliveryServiceList is the response from GET /shipment-management/delivery-services.
type DeliveryServiceList struct {
	DeliveryServices []DeliveryService `json:"deliveryServices"`
}

// DeliveryService represents an available delivery service for shipment management.
type DeliveryService struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CarrierID string `json:"carrierId"`
}

// CreateShipmentCommand is the request body for creating a managed shipment.
type CreateShipmentCommand struct {
	CommandID string              `json:"commandId"`
	Input     CreateShipmentInput `json:"input"`
}

// CreateShipmentInput contains the details for a new managed shipment.
type CreateShipmentInput struct {
	DeliveryMethodID string          `json:"deliveryMethodId"`
	CredentialsID    string          `json:"credentialsId,omitempty"`
	Sender           ShipmentAddress `json:"sender"`
	Receiver         ShipmentAddress `json:"receiver"`
	Packages         []ShipmentPkg   `json:"packages"`
	LabelFormat      string          `json:"labelFormat,omitempty"`
}

// ShipmentAddress represents sender or receiver address for a managed shipment.
type ShipmentAddress struct {
	Name        string `json:"name,omitempty"`
	Company     string `json:"company,omitempty"`
	Street      string `json:"street"`
	City        string `json:"city"`
	ZipCode     string `json:"zipCode"`
	CountryCode string `json:"countryCode"`
	Phone       string `json:"phone,omitempty"`
	Email       string `json:"email,omitempty"`
}

// ShipmentPkg represents a package within a managed shipment.
type ShipmentPkg struct {
	Type   string     `json:"type,omitempty"`
	Length *Dimension `json:"length,omitempty"`
	Width  *Dimension `json:"width,omitempty"`
	Height *Dimension `json:"height,omitempty"`
	Weight *Dimension `json:"weight,omitempty"`
}

// Dimension represents a measurement value with its unit.
type Dimension struct {
	Value float64 `json:"value"`
	Unit  string  `json:"unit"`
}

// CreateShipmentResponse is returned after creating a managed shipment.
type CreateShipmentResponse struct {
	CommandID  string `json:"commandId"`
	ShipmentID string `json:"shipmentId"`
	Status     string `json:"status"`
}

// ManagedShipment represents a shipment managed through Allegro.
type ManagedShipment struct {
	ID        string `json:"id"`
	Status    string `json:"status"`
	Waybill   string `json:"waybill"`
	CarrierID string `json:"carrierId"`
	LabelURL  string `json:"labelUrl"`
}

// PickupProposalRequest is the request body for retrieving pickup proposals.
type PickupProposalRequest struct {
	DeliveryMethodID string   `json:"deliveryMethodId"`
	ShipmentIDs      []string `json:"shipmentIds"`
}

// PickupProposalList is the response from POST /shipment-management/pickup-proposals.
type PickupProposalList struct {
	Proposals []PickupProposal `json:"proposals"`
}

// PickupProposal represents a proposed pickup date and available time windows.
type PickupProposal struct {
	Date        string             `json:"date"`
	TimeWindows []PickupTimeWindow `json:"timeWindows"`
}

// PickupTimeWindow represents a time window for a pickup.
type PickupTimeWindow struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// SchedulePickupCommand is the request body for scheduling a pickup.
type SchedulePickupCommand struct {
	CommandID   string           `json:"commandId"`
	PickupDate  string           `json:"pickupDate"`
	TimeWindow  PickupTimeWindow `json:"timeWindow"`
	ShipmentIDs []string         `json:"shipmentIds"`
}

// --- Messaging models ---

// ListThreadsParams are the optional parameters for listing messaging threads.
type ListThreadsParams struct {
	Limit  int
	Offset int
}

// ThreadList represents a paginated list of messaging threads.
type ThreadList struct {
	Threads []Thread `json:"threads"`
	Count   int      `json:"count"`
}

// Thread represents an Allegro messaging thread.
type Thread struct {
	ID                  string             `json:"id"`
	Subject             string             `json:"subject"`
	Interlocutor        ThreadInterlocutor `json:"interlocutor"`
	LastMessageDateTime string             `json:"lastMessageDateTime"`
	Read                bool               `json:"read"`
	Offer               *ThreadOffer       `json:"offer,omitempty"`
}

// ThreadInterlocutor represents the other party in a messaging thread.
type ThreadInterlocutor struct {
	ID    string `json:"id"`
	Login string `json:"login"`
}

// ThreadOffer represents the offer associated with a messaging thread.
type ThreadOffer struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ListMessagesParams are the optional parameters for listing messages in a thread.
type ListMessagesParams struct {
	Limit  int
	Offset int
	Before string
}

// MessageList represents a paginated list of messages.
type MessageList struct {
	Messages []Message `json:"messages"`
	Count    int       `json:"count"`
}

// Message represents a single message in a thread.
type Message struct {
	ID                       string        `json:"id"`
	Text                     string        `json:"text"`
	Author                   MessageAuthor `json:"author"`
	CreatedAt                string        `json:"createdAt"`
	Type                     string        `json:"type"`
	HasAdditionalAttachments bool          `json:"hasAdditionalAttachments"`
}

// MessageAuthor represents the author of a message.
type MessageAuthor struct {
	Login          string `json:"login"`
	IsInterlocutor bool   `json:"isInterlocutor"`
}

// SendMessageRequest is the request body for sending a message.
type SendMessageRequest struct {
	Text string `json:"text"`
}

// --- Customer return models ---

// ListReturnsParams are the optional parameters for listing customer returns.
type ListReturnsParams struct {
	Limit  int
	Offset int
	Status string
}

// CustomerReturnList represents a paginated list of customer returns.
type CustomerReturnList struct {
	CustomerReturns []CustomerReturn `json:"customerReturns"`
	Count           int              `json:"count"`
}

// CustomerReturn represents an Allegro customer return.
type CustomerReturn struct {
	ID                string       `json:"id"`
	CreatedAt         string       `json:"createdAt"`
	ReferenceNumber   string       `json:"referenceNumber"`
	Buyer             ReturnBuyer  `json:"buyer"`
	Items             []ReturnItem `json:"items"`
	RefundAmount      *Amount      `json:"refund,omitempty"`
	Status            string       `json:"status"`
	ParcelSentByBuyer bool         `json:"parcelSentByBuyer"`
}

// ReturnBuyer represents the buyer in a customer return.
type ReturnBuyer struct {
	Login string `json:"login"`
	Email string `json:"email"`
}

// ReturnItem represents an item in a customer return.
type ReturnItem struct {
	OfferID  string `json:"offerId"`
	Name     string `json:"name"`
	Quantity int    `json:"quantity"`
}

// ReturnRejection is the request body for rejecting a customer return.
type ReturnRejection struct {
	Reason string `json:"reason"`
}

// --- Payment / Refund models ---

// PaymentOpsParams are the optional parameters for listing payment operations.
type PaymentOpsParams struct {
	Limit  int
	Offset int
	Group  string
}

// PaymentOpsList represents a paginated list of payment operations.
type PaymentOpsList struct {
	PaymentOperations []PaymentOperation `json:"paymentOperations"`
	Count             int                `json:"count"`
}

// PaymentOperation represents a single payment operation.
type PaymentOperation struct {
	Type       string `json:"type"`
	Group      string `json:"group"`
	Amount     Amount `json:"amount"`
	OccurredAt string `json:"occurredAt"`
}

// ListRefundsParams are the optional parameters for listing refunds.
type ListRefundsParams struct {
	Limit  int
	Offset int
}

// RefundList represents a paginated list of refunds.
type RefundList struct {
	Refunds []Refund `json:"refunds"`
	Count   int      `json:"count"`
}

// Refund represents an Allegro refund.
type Refund struct {
	ID         string           `json:"id"`
	Payment    RefundPayment    `json:"payment"`
	Reason     string           `json:"reason"`
	Status     string           `json:"status"`
	CreatedAt  string           `json:"createdAt"`
	TotalValue Amount           `json:"totalValue"`
	LineItems  []RefundLineItem `json:"lineItems"`
}

// RefundPayment is a reference to the payment associated with a refund.
type RefundPayment struct {
	ID string `json:"id"`
}

// RefundLineItem represents a line item in a refund.
type RefundLineItem struct {
	OfferID  string `json:"offerId"`
	Quantity int    `json:"quantity"`
	Amount   Amount `json:"amount"`
}

// CreateRefundRequest is the request body for creating a refund.
type CreateRefundRequest struct {
	Payment   RefundPayment    `json:"payment"`
	Reason    string           `json:"reason"`
	LineItems []RefundLineItem `json:"lineItems"`
}

// --- Category models ---

// CategoryList represents a list of categories.
type CategoryList struct {
	Categories []Category `json:"categories"`
}

// MatchingCategoriesResponse represents the response from category search.
type MatchingCategoriesResponse struct {
	MatchingCategories []MatchingCategory `json:"matchingCategories"`
}

// MatchingCategory is a category suggestion with nested parent chain.
type MatchingCategory struct {
	ID     string            `json:"id"`
	Name   string            `json:"name"`
	Parent *MatchingCategory `json:"parent,omitempty"`
}

// Category represents an Allegro category.
type Category struct {
	ID      string           `json:"id"`
	Name    string           `json:"name"`
	Parent  *CategoryRef     `json:"parent,omitempty"`
	Leaf    bool             `json:"leaf"`
	Options *CategoryOptions `json:"options,omitempty"`
}

// CategoryRef is a reference to a parent category.
type CategoryRef struct {
	ID string `json:"id"`
}

// CategoryOptions represents options for a category.
type CategoryOptions struct {
	Advertisement                       bool `json:"advertisement"`
	AdvertisementPriceOptional          bool `json:"advertisementPriceOptional"`
	VariantsByColorPattern              bool `json:"variantsByColorPatternAllowed"`
	OffersWithProductPublicationEnabled bool `json:"offersWithProductPublicationEnabled"`
	ProductCreationEnabled              bool `json:"productCreationEnabled"`
}

// CategoryParameterList represents a list of category parameters.
type CategoryParameterList struct {
	Parameters []CategoryParameter `json:"parameters"`
}

// CategoryParameter represents a single category parameter.
type CategoryParameter struct {
	ID                 string                 `json:"id"`
	Name               string                 `json:"name"`
	Type               string                 `json:"type"`
	Required           bool                   `json:"required"`
	RequiredForProduct bool                   `json:"requiredForProduct"`
	Unit               string                 `json:"unit,omitempty"`
	Options            *ParameterOptions      `json:"options,omitempty"`
	Restrictions       *ParameterRestrictions `json:"restrictions,omitempty"`
	Dictionary         []ParameterDictValue   `json:"dictionary,omitempty"`
}

// ParameterOptions represents options for a category parameter.
type ParameterOptions struct {
	VariantsAllowed      bool   `json:"variantsAllowed"`
	VariantsEqual        bool   `json:"variantsEqual,omitempty"`
	AmbiguousValueId     string `json:"ambiguousValueId,omitempty"`
	DependsOnParameterId string `json:"dependsOnParameterId,omitempty"`
	DescribesProduct     bool   `json:"describesProduct"`
	CustomValuesEnabled  bool   `json:"customValuesEnabled"`
}

// ParameterRestrictions represents restrictions for a category parameter.
type ParameterRestrictions struct {
	Min       *float64 `json:"min,omitempty"`
	Max       *float64 `json:"max,omitempty"`
	Range     bool     `json:"range"`
	Precision int      `json:"precision"`
	MinLength *int     `json:"minLength,omitempty"`
	MaxLength *int     `json:"maxLength,omitempty"`
}

// ParameterDictValue represents a dictionary value for a parameter.
type ParameterDictValue struct {
	ID    string `json:"id"`
	Value string `json:"value"`
}

// --- Product Catalog models ---

// SearchProductsParams are the optional parameters for searching the product catalog.
type SearchProductsParams struct {
	Phrase     string
	CategoryID string
	Limit      int
	Offset     int
}

// ProductCatalogList represents a paginated list of catalog products.
type ProductCatalogList struct {
	Products []CatalogProduct `json:"products"`
	Count    int              `json:"count"`
}

// CatalogProduct represents a product from the Allegro catalog.
type CatalogProduct struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Category    *CategoryRef        `json:"category,omitempty"`
	Images      []ProductImage      `json:"images,omitempty"`
	Parameters  []ProductParameter  `json:"parameters,omitempty"`
	Description *CatalogDescription `json:"description,omitempty"`
}

// ProductImage represents an image in the product catalog.
type ProductImage struct {
	URL string `json:"url"`
}

// ProductParameter represents a parameter value for a catalog product.
type ProductParameter struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Values    []string `json:"values,omitempty"`
	ValuesIDs []string `json:"valuesIds,omitempty"`
	Unit      string   `json:"unit,omitempty"`
}

// CatalogDescription represents a product catalog description.
type CatalogDescription struct {
	Sections []DescriptionSection `json:"sections"`
}

// --- Pricing/Fee models ---

// FeePreview represents the fee preview for an offer.
type FeePreview struct {
	Commissions []FeeCommission `json:"commissions"`
	Quotes      []FeeQuote      `json:"quotes"`
}

// FeeCommission represents a commission in a fee preview.
type FeeCommission struct {
	Type string `json:"type"`
	Rate Amount `json:"rate"`
}

// FeeQuote represents a fee quote in a fee preview.
type FeeQuote struct {
	Type string `json:"type"`
	Fee  Amount `json:"fee"`
	Name string `json:"name"`
}

// CommissionList represents a list of commission rates.
type CommissionList struct {
	Commissions []CommissionRate `json:"commissions"`
}

// CommissionRate represents commission rates for a category.
type CommissionRate struct {
	CategoryID string `json:"categoryId"`
	Rates      []Rate `json:"rates"`
}

// Rate represents a single commission rate.
type Rate struct {
	Type    string  `json:"type"`
	Value   float64 `json:"value"`
	Percent float64 `json:"percent"`
}

// --- Dispute models ---

// ListDisputesParams are the optional parameters for listing disputes.
type ListDisputesParams struct {
	Limit  int
	Offset int
	Status string // OPEN, CLOSED, etc.
}

// DisputeList represents a paginated list of disputes.
type DisputeList struct {
	Disputes []Dispute `json:"disputes"`
	Count    int       `json:"count"`
}

// Dispute represents an Allegro post-purchase dispute.
type Dispute struct {
	ID           string           `json:"id"`
	Subject      string           `json:"subject"`
	Status       string           `json:"status"`
	Buyer        DisputeBuyer     `json:"buyer"`
	CheckoutForm DisputeOrder     `json:"checkoutForm"`
	Messages     []DisputeMessage `json:"messages,omitempty"`
	CreatedAt    string           `json:"createdAt"`
	UpdatedAt    string           `json:"updatedAt"`
}

// DisputeBuyer represents the buyer involved in a dispute.
type DisputeBuyer struct {
	Login string `json:"login"`
}

// DisputeOrder represents the order associated with a dispute.
type DisputeOrder struct {
	ID string `json:"id"`
}

// DisputeMessageList represents a list of messages in a dispute.
type DisputeMessageList struct {
	Messages []DisputeMessage `json:"messages"`
	Count    int              `json:"count"`
}

// DisputeMessage represents a single message in a dispute.
type DisputeMessage struct {
	ID        string `json:"id"`
	Text      string `json:"text"`
	Author    string `json:"author"` // BUYER or SELLER
	CreatedAt string `json:"createdAt"`
	Type      string `json:"type"`
}

// DisputeMessageRequest is the request body for sending a dispute message.
type DisputeMessageRequest struct {
	Text string `json:"text"`
	Type string `json:"type,omitempty"` // MESSAGE, REFUND_OFFER, etc.
}

// --- Rating management models ---

// RatingAnswer represents the seller's answer to a rating.
type RatingAnswer struct {
	ID        string `json:"id,omitempty"`
	Text      string `json:"text"`
	CreatedAt string `json:"createdAt,omitempty"`
}

// RatingAnswerRequest is the request body for creating/updating a rating answer.
type RatingAnswerRequest struct {
	Text string `json:"text"`
}

// RatingRemovalRequest is the request body for requesting removal of a rating.
type RatingRemovalRequest struct {
	Reason string `json:"reason"`
}

// --- Promotion models ---

// ListPromotionsParams are the optional parameters for listing promotions.
type ListPromotionsParams struct {
	Limit  int
	Offset int
}

// PromotionList represents a paginated list of promotions.
type PromotionList struct {
	Promotions []Promotion `json:"promotions"`
	Count      int         `json:"count"`
}

// Promotion represents an Allegro promotion campaign.
type Promotion struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Benefits  []PromoBenefit  `json:"benefits"`
	Criteria  []PromoCriteria `json:"criteria,omitempty"`
	Status    string          `json:"status"`
	CreatedAt string          `json:"createdAt"`
	UpdatedAt string          `json:"updatedAt"`
}

// PromoBenefit represents a benefit within a promotion.
type PromoBenefit struct {
	Specification *BenefitSpec `json:"specification,omitempty"`
}

// BenefitSpec describes the type and value of a promotion benefit.
type BenefitSpec struct {
	Type  string  `json:"type"` // ORDER_FIXED_DISCOUNT, MULTI_PACK, FREE_SHIPPING, etc.
	Value *Amount `json:"value,omitempty"`
}

// PromoCriteria represents a criterion for a promotion.
type PromoCriteria struct {
	Type   string               `json:"type"` // CONTAINS_OFFERS, ORDER_VALUE_AT_LEAST, etc.
	Offers []PromoCriteriaOffer `json:"offers,omitempty"`
	Value  *Amount              `json:"value,omitempty"`
}

// PromoCriteriaOffer represents an offer reference within a promotion criterion.
type PromoCriteriaOffer struct {
	ID       string `json:"id"`
	Quantity int    `json:"quantity,omitempty"`
}

// CreatePromotionRequest is the request body for creating or updating a promotion.
type CreatePromotionRequest struct {
	Name     string          `json:"name"`
	Benefits []PromoBenefit  `json:"benefits"`
	Criteria []PromoCriteria `json:"criteria,omitempty"`
}

// BadgeList represents a list of promotion badge packages.
type BadgeList struct {
	Packages []PromoBadge `json:"packages"`
}

// PromoBadge represents a promotion badge (Bold, Highlight, etc.).
type PromoBadge struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Price       Amount `json:"price"`
}

// --- Delivery Settings models ---

// DeliverySettings represents the seller's delivery configuration.
type DeliverySettings struct {
	FreeDelivery   *FreeDeliverySettings   `json:"freeDelivery,omitempty"`
	JoinPolicy     *JoinPolicy             `json:"joinPolicy,omitempty"`
	CustomCost     *CustomCostSettings     `json:"customCost,omitempty"`
	AbroadDelivery *AbroadDeliverySettings `json:"abroadDelivery,omitempty"`
}

// FreeDeliverySettings represents free delivery threshold settings.
type FreeDeliverySettings struct {
	Amount    Amount `json:"amount"`
	Threshold Amount `json:"threshold"`
}

// JoinPolicy represents how shipping costs are combined for multiple items.
type JoinPolicy struct {
	Strategy string `json:"strategy"` // MIN, MAX, SUM
}

// CustomCostSettings represents whether custom delivery cost is allowed.
type CustomCostSettings struct {
	Allowed bool `json:"allowed"`
}

// AbroadDeliverySettings represents abroad delivery settings.
type AbroadDeliverySettings struct {
	Enabled bool `json:"enabled"`
}

// ShippingRateList represents a list of shipping rate tables.
type ShippingRateList struct {
	ShippingRates []ShippingRateSet `json:"shippingRates"`
}

// ShippingRateSet represents a single shipping rate table.
type ShippingRateSet struct {
	ID    string              `json:"id"`
	Name  string              `json:"name"`
	Rates []ShippingRateEntry `json:"rates"`
}

// ShippingRateEntry represents a single rate within a shipping rate table.
type ShippingRateEntry struct {
	DeliveryMethod        ShippingDeliveryMethod `json:"deliveryMethod"`
	MaxQuantityPerPackage int                    `json:"maxQuantityPerPackage"`
	FirstItemRate         Amount                 `json:"firstItemRate"`
	NextItemRate          Amount                 `json:"nextItemRate"`
	ShippingTime          *ShippingTime          `json:"shippingTime,omitempty"`
}

// ShippingDeliveryMethod represents a delivery method reference within a shipping rate.
type ShippingDeliveryMethod struct {
	ID string `json:"id"`
}

// MaxQuantity represents the maximum quantity per package.
type MaxQuantity struct {
	Value int `json:"value"`
}

// ShippingTime represents the shipping time range.
type ShippingTime struct {
	From string `json:"from"` // PT24H, PT48H, etc.
	To   string `json:"to"`
}

// CreateShippingRateRequest is the request body for creating or updating a shipping rate table.
type CreateShippingRateRequest struct {
	Name  string              `json:"name"`
	Rates []ShippingRateEntry `json:"rates"`
}

// DeliveryMethodList represents a list of available delivery methods.
type DeliveryMethodList struct {
	DeliveryMethods []AllegroDeliveryMethod `json:"deliveryMethods"`
}

// AllegroDeliveryMethod represents an available Allegro delivery method.
type AllegroDeliveryMethod struct {
	ID                       string               `json:"id"`
	Name                     string               `json:"name"`
	PaymentPolicy            string               `json:"paymentPolicy"`
	ShippingRatesConstraints *ShippingConstraints `json:"shippingRatesConstraints,omitempty"`
}

// ShippingConstraints represents constraints for a delivery method.
type ShippingConstraints struct {
	Allowed               bool            `json:"allowed"`
	MaxQuantityPerPackage *MaxQuantity    `json:"maxQuantityPerPackage,omitempty"`
	AllowedForFree        bool            `json:"allowedForFreeShipping"`
	FirstItemRate         *RateConstraint `json:"firstItemRate,omitempty"`
	NextItemRate          *RateConstraint `json:"nextItemRate,omitempty"`
	ShippingTime          *TimeConstraint `json:"shippingTime,omitempty"`
}

// RateConstraint represents min/max price constraints for a rate entry.
type RateConstraint struct {
	Min      string `json:"min"`
	Max      string `json:"max"`
	Currency string `json:"currency"`
}

// TimeConstraint represents shipping time constraints for a delivery method.
type TimeConstraint struct {
	Default      *ShippingTime `json:"default,omitempty"`
	Customizable bool          `json:"customizable"`
}

// --- Return Policy models ---

// ReturnPolicyList represents a list of seller return policies.
type ReturnPolicyList struct {
	ReturnPolicies []ReturnPolicy `json:"returnPolicies"`
}

// ReturnPolicy represents an Allegro return policy.
type ReturnPolicy struct {
	ID               string              `json:"id"`
	Name             string              `json:"name"`
	Availability     *ReturnAvailability `json:"availability,omitempty"`
	WithdrawalPeriod string              `json:"withdrawalPeriod,omitempty"`
	ReturnCost       *ReturnCostPolicy   `json:"returnCost,omitempty"`
	Options          *ReturnOptions      `json:"options,omitempty"`
	Address          *ReturnAddress      `json:"address,omitempty"`
	Description      string              `json:"description,omitempty"`
	Contact          *ReturnContact      `json:"contact,omitempty"`
}

// ReturnAvailability represents the return availability type.
type ReturnAvailability struct {
	Range            string `json:"range"`                      // FULL, RESTRICTED, DISABLED
	RestrictionCause string `json:"restrictionCause,omitempty"` // for RESTRICTED/DISABLED
}

// ReturnCostPolicy represents who covers the return cost.
type ReturnCostPolicy struct {
	CoveredBy string `json:"coveredBy"` // SELLER, BUYER
}

// ReturnOptions represents the return policy options.
type ReturnOptions struct {
	CashOnDeliveryNotAllowed        bool `json:"cashOnDeliveryNotAllowed"`
	FreeAccessoriesReturnRequired   bool `json:"freeAccessoriesReturnRequired"`
	RefundLoweredByReceivedDiscount bool `json:"refundLoweredByReceivedDiscount"`
	BusinessReturnAllowed           bool `json:"businessReturnAllowed"`
	CollectBySellerOnly             bool `json:"collectBySellerOnly"`
}

// ReturnAddress represents the address for returning items.
type ReturnAddress struct {
	Name        string `json:"name"`
	Street      string `json:"street"`
	City        string `json:"city"`
	PostCode    string `json:"postCode"`
	CountryCode string `json:"countryCode"`
}

// ReturnContact represents contact details for a return policy (v1: phoneNumber + email).
type ReturnContact struct {
	PhoneNumber string `json:"phoneNumber,omitempty"`
	Email       string `json:"email,omitempty"`
}

// CreateReturnPolicyRequest is the request body for creating or updating a return policy (v1).
type CreateReturnPolicyRequest struct {
	Name             string              `json:"name"`
	Availability     *ReturnAvailability `json:"availability,omitempty"`
	WithdrawalPeriod string              `json:"withdrawalPeriod,omitempty"` // ISO 8601 e.g. "P14D"
	ReturnCost       *ReturnCostPolicy   `json:"returnCost,omitempty"`
	Options          *ReturnOptions      `json:"options,omitempty"`
	Address          *ReturnAddress      `json:"address,omitempty"`
	Description      string              `json:"description,omitempty"`
	Contact          *ReturnContact      `json:"contact,omitempty"`
}

// --- Implied Warranty models ---

// WarrantyList represents a list of implied warranty policies.
type WarrantyList struct {
	ImpliedWarranties []ImpliedWarranty `json:"impliedWarranties"`
}

// ImpliedWarranty represents an Allegro implied warranty (rekojmia).
type ImpliedWarranty struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	Individual *WarrantyTerms `json:"individual,omitempty"`
	Corporate  *WarrantyTerms `json:"corporate,omitempty"`
}

// WarrantyTerms represents the warranty terms for a given buyer type.
type WarrantyTerms struct {
	Period string `json:"period"` // P1Y, P2Y, etc. (ISO 8601)
	Type   string `json:"type"`   // IMPLIED_WARRANTY, WITHOUT_WARRANTY
}

// WarrantyAddress represents the address for a warranty (uses postCode, not zipCode).
type WarrantyAddress struct {
	Name        string `json:"name"`
	Street      string `json:"street"`
	City        string `json:"city"`
	PostCode    string `json:"postCode"`
	CountryCode string `json:"countryCode"`
}

// CreateWarrantyRequest is the request body for creating or updating an implied warranty.
type CreateWarrantyRequest struct {
	Name       string           `json:"name"`
	Individual *WarrantyTerms   `json:"individual,omitempty"`
	Corporate  *WarrantyTerms   `json:"corporate,omitempty"`
	Address    *WarrantyAddress `json:"address,omitempty"`
}

// --- Size Table models ---

// SizeTableList represents a list of size tables.
type SizeTableList struct {
	SizeTables []SizeTable `json:"sizeTables"`
}

// SizeTable represents an Allegro size table.
type SizeTable struct {
	ID      string       `json:"id"`
	Name    string       `json:"name"`
	Type    string       `json:"type"` // MALE, FEMALE, KIDS, UNISEX
	Headers []SizeHeader `json:"headers"`
	Values  [][]string   `json:"values"`
}

// SizeHeader represents a column header in a size table.
type SizeHeader struct {
	Name string `json:"name"`
}

// CreateSizeTableRequest is the request body for creating or updating a size table.
type CreateSizeTableRequest struct {
	Name    string       `json:"name"`
	Type    string       `json:"type"`
	Headers []SizeHeader `json:"headers"`
	Values  [][]string   `json:"values"`
}
