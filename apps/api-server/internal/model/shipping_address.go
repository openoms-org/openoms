package model

type ShippingAddress struct {
	Name       string  `json:"name"`
	Company    *string `json:"company,omitempty"`
	Street     string  `json:"street"`
	City       string  `json:"city"`
	PostalCode string  `json:"postal_code"`
	Country    string  `json:"country"`
	Phone      string  `json:"phone,omitempty"`
	Email      string  `json:"email,omitempty"`
}
