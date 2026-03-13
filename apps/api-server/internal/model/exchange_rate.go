package model

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// ExchangeRate stores a currency conversion rate between two currencies.
type ExchangeRate struct {
	ID             uuid.UUID `json:"id"`
	TenantID       uuid.UUID `json:"tenant_id"`
	BaseCurrency   string    `json:"base_currency"`
	TargetCurrency string    `json:"target_currency"`
	Rate           float64   `json:"rate"`
	Source         string    `json:"source"`
	FetchedAt      time.Time `json:"fetched_at"`
	CreatedAt      time.Time `json:"created_at"`
}

// CreateExchangeRateRequest is the payload for creating a new exchange rate.
type CreateExchangeRateRequest struct {
	BaseCurrency   string  `json:"base_currency"`
	TargetCurrency string  `json:"target_currency"`
	Rate           float64 `json:"rate"`
	Source         string  `json:"source,omitempty"`
}

// Validate validates the create exchange rate request.
func (r CreateExchangeRateRequest) Validate() error {
	if r.BaseCurrency == "" {
		return errors.New("base_currency is required")
	}
	if r.TargetCurrency == "" {
		return errors.New("target_currency is required")
	}
	if r.Rate <= 0 {
		return errors.New("rate must be positive")
	}
	if r.BaseCurrency == r.TargetCurrency {
		return errors.New("base_currency and target_currency must be different")
	}
	return nil
}

// UpdateExchangeRateRequest is the payload for updating an existing exchange rate.
type UpdateExchangeRateRequest struct {
	Rate   *float64 `json:"rate,omitempty"`
	Source *string  `json:"source,omitempty"`
}

// Validate validates the update exchange rate request.
func (r UpdateExchangeRateRequest) Validate() error {
	if r.Rate == nil && r.Source == nil {
		return errors.New("at least one field must be provided")
	}
	if r.Rate != nil && *r.Rate <= 0 {
		return errors.New("rate must be positive")
	}
	return nil
}

// ConvertAmountRequest is the payload for converting an amount between two currencies.
type ConvertAmountRequest struct {
	Amount float64 `json:"amount"`
	From   string  `json:"from"`
	To     string  `json:"to"`
}

// Validate validates the convert amount request.
func (r ConvertAmountRequest) Validate() error {
	if r.Amount < 0 {
		return errors.New("amount must be non-negative")
	}
	if r.From == "" {
		return errors.New("from currency is required")
	}
	if r.To == "" {
		return errors.New("to currency is required")
	}
	return nil
}

// ConvertAmountResponse holds the result of a currency conversion.
type ConvertAmountResponse struct {
	OriginalAmount  float64 `json:"original_amount"`
	ConvertedAmount float64 `json:"converted_amount"`
	From            string  `json:"from"`
	To              string  `json:"to"`
	Rate            float64 `json:"rate"`
}

// ExchangeRateListFilter holds query parameters for listing exchange rates.
type ExchangeRateListFilter struct {
	BaseCurrency   *string
	TargetCurrency *string
	PaginationParams
}
