package fakturownia

import "time"

// InvoicePosition represents a single line item on an invoice.
type InvoicePosition struct {
	Name      string  `json:"name"`
	Quantity  int     `json:"quantity"`
	NetPrice  float64 `json:"total_price_net,string"`
	Tax       int     `json:"tax"`
	Unit      string  `json:"quantity_unit,omitempty"`
}

// CreateInvoiceRequest is the request body for creating an invoice in Fakturownia.
type CreateInvoiceRequest struct {
	APIToken  string              `json:"api_token"`
	Invoice   InvoiceRequestData  `json:"invoice"`
}

// InvoiceRequestData holds the invoice fields sent to Fakturownia.
type InvoiceRequestData struct {
	Kind              string            `json:"kind"`
	Number            *string           `json:"number,omitempty"`
	SellDate          string            `json:"sell_date"`
	IssueDate         string            `json:"issue_date"`
	PaymentTo         string            `json:"payment_to"`
	SellerName        string            `json:"seller_name,omitempty"`
	SellerTaxNo       string            `json:"seller_tax_no,omitempty"`
	BuyerName         string            `json:"buyer_name"`
	BuyerEmail        string            `json:"buyer_email,omitempty"`
	BuyerTaxNo        string            `json:"buyer_tax_no,omitempty"`
	PaymentType       string            `json:"payment_type,omitempty"`
	Currency          string            `json:"currency,omitempty"`
	Description       string            `json:"description,omitempty"`
	Positions         []InvoicePosition `json:"positions"`
}

// InvoiceResponse is the response from Fakturownia after creating/getting an invoice.
type InvoiceResponse struct {
	ID              int     `json:"id"`
	Number          string  `json:"number"`
	Kind            string  `json:"kind"`
	Status          string  `json:"status"`
	PriceNet        string  `json:"price_net"`
	PriceGross      string  `json:"price_gross"`
	Currency        string  `json:"currency"`
	IssueDate       string  `json:"issue_date"`
	PaymentTo       string  `json:"payment_to"`
	BuyerName       string  `json:"buyer_name"`
	BuyerEmail      string  `json:"buyer_email"`
	BuyerTaxNo      string  `json:"buyer_tax_no"`
	SellerName      string  `json:"seller_name"`
	SellerTaxNo     string  `json:"seller_tax_no"`
	ViewURL         string  `json:"view_url"`
	Token           string  `json:"token"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// APIError represents an error response from the Fakturownia API.
type APIError struct {
	StatusCode int
	Code       string `json:"code"`
	Message    string `json:"message"`
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return "fakturownia: " + e.Message
	}
	return "fakturownia: unexpected status " + httpStatusText(e.StatusCode)
}

func httpStatusText(code int) string {
	switch code {
	case 400:
		return "400 Bad Request"
	case 401:
		return "401 Unauthorized"
	case 403:
		return "403 Forbidden"
	case 404:
		return "404 Not Found"
	case 422:
		return "422 Unprocessable Entity"
	case 500:
		return "500 Internal Server Error"
	default:
		return "unknown error"
	}
}
