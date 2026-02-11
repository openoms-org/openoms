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
