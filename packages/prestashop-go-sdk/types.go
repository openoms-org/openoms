package prestashop

// PSProduct represents a product from the PrestaShop API.
type PSProduct struct {
	ID               int          `json:"id"`
	Reference        string       `json:"reference"` // SKU
	EAN13            string       `json:"ean13"`
	Price            string       `json:"price"`
	WholesalePrice   string       `json:"wholesale_price"`
	Quantity         int          `json:"quantity"`
	Active           string       `json:"active"` // "1" or "0"
	IDCategoryDefault int         `json:"id_category_default"`
	Name             []PSLangVal  `json:"name"`
	Description      []PSLangVal  `json:"description"`
	DescriptionShort []PSLangVal  `json:"description_short"`
	DateUpd          string       `json:"date_upd"`
}

// PSLangVal is a language-specific value from the PrestaShop API.
type PSLangVal struct {
	ID    int    `json:"id"`
	Value string `json:"value"`
}

// PSOrder represents an order from the PrestaShop API.
type PSOrder struct {
	ID               int    `json:"id"`
	Reference        string `json:"reference"`
	CurrentState     int    `json:"current_state"`
	IDCurrency       int    `json:"id_currency"`
	TotalPaid        string `json:"total_paid"`
	TotalPaidTaxIncl string `json:"total_paid_tax_incl"`
	TotalProducts    string `json:"total_products"`
	TotalShipping    string `json:"total_shipping"`
	Payment          string `json:"payment"`
	Module           string `json:"module"` // payment module
	DateAdd          string `json:"date_add"`
	DateUpd          string `json:"date_upd"`
	IDCustomer       int    `json:"id_customer"`
	IDAddressDelivery int   `json:"id_address_delivery"`
	IDAddressInvoice  int   `json:"id_address_invoice"`
}

// PSOrderDetail represents a line item in a PrestaShop order.
type PSOrderDetail struct {
	ID            int    `json:"id"`
	IDOrder       int    `json:"id_order"`
	ProductID     int    `json:"product_id"`
	ProductName   string `json:"product_name"`
	ProductRef    string `json:"product_reference"`
	ProductEAN    string `json:"product_ean13"`
	Quantity      int    `json:"product_quantity"`
	UnitPrice     string `json:"unit_price_tax_incl"`
	TotalPrice    string `json:"total_price_tax_incl"`
}

// PSAddress represents an address from the PrestaShop API.
type PSAddress struct {
	ID        int    `json:"id"`
	FirstName string `json:"firstname"`
	LastName  string `json:"lastname"`
	Company   string `json:"company"`
	Address1  string `json:"address1"`
	Address2  string `json:"address2"`
	City      string `json:"city"`
	PostCode  string `json:"postcode"`
	IDCountry int    `json:"id_country"`
	Phone     string `json:"phone"`
	PhoneMobile string `json:"phone_mobile"`
}

// PSCustomer represents a customer from the PrestaShop API.
type PSCustomer struct {
	ID        int    `json:"id"`
	FirstName string `json:"firstname"`
	LastName  string `json:"lastname"`
	Email     string `json:"email"`
}

// PSStockAvailable represents stock for a product in PrestaShop.
type PSStockAvailable struct {
	ID        int `json:"id"`
	IDProduct int `json:"id_product"`
	Quantity  int `json:"quantity"`
}

// ListResponse wraps a PrestaShop JSON list response.
type ListResponse[T any] struct {
	Items []T `json:"items"`
}
