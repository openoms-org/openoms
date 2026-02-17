package model

import "time"

// VATRateSet holds VAT rates for a single EU country.
type VATRateSet struct {
	Standard     float64 `json:"standard"`
	Reduced1     float64 `json:"reduced_1,omitempty"`
	Reduced2     float64 `json:"reduced_2,omitempty"`
	SuperReduced float64 `json:"super_reduced,omitempty"`
}

// VATCalculation holds the result of a VAT calculation.
type VATCalculation struct {
	NetAmount   float64 `json:"net_amount"`
	VATAmount   float64 `json:"vat_amount"`
	GrossAmount float64 `json:"gross_amount"`
	VATRate     float64 `json:"vat_rate"`
	Country     string  `json:"country"`
	RateType    string  `json:"rate_type"`
}

// OSSConfig holds tenant-level OSS configuration stored in settings JSONB.
type OSSConfig struct {
	Enabled        bool   `json:"enabled"`
	HomeCountry    string `json:"home_country"`
	DefaultVATRate string `json:"default_vat_rate"`
}

// OSSReport holds a quarterly OSS report.
type OSSReport struct {
	Quarter     int                `json:"quarter"`
	Year        int                `json:"year"`
	HomeCountry string             `json:"home_country"`
	TotalSales  float64            `json:"total_sales"`
	TotalVAT    float64            `json:"total_vat"`
	ByCountry   []OSSCountryReport `json:"by_country"`
	GeneratedAt time.Time          `json:"generated_at"`
}

// OSSCountryReport holds per-country data within an OSS report.
type OSSCountryReport struct {
	Country     string  `json:"country"`
	CountryName string  `json:"country_name"`
	OrderCount  int     `json:"order_count"`
	TotalSales  float64 `json:"total_sales"`
	VATRate     float64 `json:"vat_rate"`
	VATAmount   float64 `json:"vat_amount"`
}

// ThresholdStatus holds the annual cross-border threshold check result.
type ThresholdStatus struct {
	Year                 int     `json:"year"`
	TotalCrossBorder     float64 `json:"total_cross_border_eur"`
	Threshold            float64 `json:"threshold_eur"`
	Exceeded             bool    `json:"exceeded"`
	RemainingToThreshold float64 `json:"remaining_eur"`
}

// VATCalculateRequest is the request body for the calculate endpoint.
type VATCalculateRequest struct {
	Amount   float64 `json:"amount"`
	Country  string  `json:"country"`
	RateType string  `json:"rate_type"`
}
