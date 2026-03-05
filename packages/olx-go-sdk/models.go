package olx

// AdvertListResponse is the response from the OLX adverts endpoint.
type AdvertListResponse struct {
	Data   []Advert   `json:"data"`
	Links  PageLinks  `json:"links"`
}

// PageLinks contains pagination links.
type PageLinks struct {
	Self string `json:"self"`
	Next string `json:"next,omitempty"`
	Prev string `json:"prev,omitempty"`
}

// Advert represents an OLX classified advertisement.
type Advert struct {
	ID          int64           `json:"id"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Status      string          `json:"status"`
	URL         string          `json:"url"`
	CreatedAt   string          `json:"created_time"`
	ValidTo     string          `json:"valid_to"`
	Price       *AdvertPrice    `json:"price,omitempty"`
	Category    *AdvertCategory `json:"category,omitempty"`
	Contact     *Contact        `json:"contact,omitempty"`
	Params      []Param         `json:"params,omitempty"`
	ExternalID  string          `json:"external_id,omitempty"`
}

// AdvertPrice represents the pricing of an advert.
type AdvertPrice struct {
	Value        float64 `json:"value"`
	Currency     string  `json:"currency"`
	Negotiable   bool    `json:"negotiable"`
	DisplayValue string  `json:"displayValue,omitempty"`
}

// AdvertCategory represents the category of an advert.
type AdvertCategory struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// Contact holds seller contact details.
type Contact struct {
	Name  string `json:"name"`
	Phone string `json:"phone,omitempty"`
	Email string `json:"email,omitempty"`
}

// Param represents a custom attribute of an advert.
type Param struct {
	Key   string `json:"key"`
	Name  string `json:"name"`
	Value any    `json:"value"`
}

// TransactionListResponse is the response from the OLX transactions endpoint.
type TransactionListResponse struct {
	Data  []Transaction `json:"data"`
	Links PageLinks     `json:"links"`
}

// Transaction represents a completed transaction (sale) on OLX.
type Transaction struct {
	ID            string          `json:"id"`
	AdvertID      int64           `json:"advert_id"`
	Status        string          `json:"status"`
	Amount        float64         `json:"amount"`
	Currency      string          `json:"currency"`
	CreatedAt     string          `json:"created_at"`
	BuyerName     string          `json:"buyer_name"`
	BuyerEmail    string          `json:"buyer_email"`
	BuyerPhone    string          `json:"buyer_phone,omitempty"`
	ShippingAddr  *ShippingAddr   `json:"shipping_address,omitempty"`
	AdvertTitle   string          `json:"advert_title,omitempty"`
	Quantity      int             `json:"quantity"`
}

// ShippingAddr is a delivery address for an OLX transaction.
type ShippingAddr struct {
	Name       string `json:"name"`
	Street     string `json:"street"`
	City       string `json:"city"`
	PostalCode string `json:"postal_code"`
	Country    string `json:"country"`
	Phone      string `json:"phone,omitempty"`
}

// User represents an OLX user profile.
type User struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Email   string `json:"email"`
	Phone   string `json:"phone,omitempty"`
	Created string `json:"created"`
}

// CreateAdvertRequest is the payload for POST /adverts.
type CreateAdvertRequest struct {
	Title          string            `json:"title"`
	Description    string            `json:"description"`
	CategoryID     int               `json:"category_id"`
	AdvertiserType string            `json:"advertiser_type"` // "private" or "business"
	ExternalID     string            `json:"external_id,omitempty"`
	ExternalURL    string            `json:"external_url,omitempty"`
	Contact        *Contact          `json:"contact"`
	Location       *Location         `json:"location"`
	Images         []AdvertImage     `json:"images,omitempty"`
	Price          *AdvertPrice      `json:"price,omitempty"`
	Attributes     []AdvertAttribute `json:"attributes,omitempty"`
	Delivery       *AdvertDelivery   `json:"ad_delivery,omitempty"`
	Courier        bool              `json:"courier,omitempty"`
	AutoExtend     bool              `json:"auto_extend_enabled,omitempty"`
}

// Location represents the geographic location for an advert.
type Location struct {
	CityID     int     `json:"city_id"`
	DistrictID int     `json:"district_id,omitempty"`
	Latitude   float64 `json:"latitude,omitempty"`
	Longitude  float64 `json:"longitude,omitempty"`
}

// AdvertImage is a URL-based image reference.
type AdvertImage struct {
	URL string `json:"url"`
}

// AdvertAttribute is a category-specific attribute value.
type AdvertAttribute struct {
	Code   string `json:"code"`
	Value  any    `json:"value,omitempty"`
	Values []any  `json:"values,omitempty"`
}

// AdvertDelivery configures OLX Delivery shipping options.
type AdvertDelivery struct {
	DeliveryPackageIDs []string `json:"delivery_package_ids"`
}

// AdvertCommandRequest is the payload for POST /adverts/{id}/commands.
type AdvertCommandRequest struct {
	Command   string `json:"command"`              // "activate", "deactivate", "finish", "extend"
	IsSuccess bool   `json:"is_success,omitempty"` // required for "deactivate"
}

// Category represents an OLX category.
type Category struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	ParentID    int    `json:"parent_id,omitempty"`
	PhotosLimit int    `json:"photos_limit,omitempty"`
	IsLeaf      bool   `json:"is_leaf"`
}

// CategoryListResponse is the response from GET /categories.
type CategoryListResponse struct {
	Data []Category `json:"data"`
}

// CategoryAttribute describes an attribute definition for a category.
type CategoryAttribute struct {
	Code       string                 `json:"code"`
	Label      string                 `json:"label"`
	Unit       string                 `json:"unit,omitempty"`
	Validation CategoryAttrValidation `json:"validation"`
	Values     []CategoryAttrValue    `json:"values,omitempty"`
}

// CategoryAttrValidation describes validation rules for a category attribute.
type CategoryAttrValidation struct {
	Type                string `json:"type"`
	Required            bool   `json:"required"`
	Numeric             bool   `json:"numeric"`
	Min                 any    `json:"min,omitempty"`
	Max                 any    `json:"max,omitempty"`
	AllowMultipleValues bool   `json:"allow_multiple_values"`
}

// CategoryAttrValue is a possible value for a category attribute.
type CategoryAttrValue struct {
	Code  string `json:"code"`
	Label string `json:"label"`
}

// CategoryAttributeListResponse is the response from GET /categories/{id}/attributes.
type CategoryAttributeListResponse struct {
	Data []CategoryAttribute `json:"data"`
}

// CategorySuggestion is returned by GET /categories/suggestion.
type CategorySuggestion struct {
	ID   int      `json:"id"`
	Name string   `json:"name"`
	Path []string `json:"path"`
}

// CategorySuggestionResponse is the response from GET /categories/suggestion.
type CategorySuggestionResponse struct {
	Data []CategorySuggestion `json:"data"`
}
