package model

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

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

type CreateExchangeRateRequest struct {
	BaseCurrency   string  `json:"base_currency"`
	TargetCurrency string  `json:"target_currency"`
	Rate           float64 `json:"rate"`
	Source         string  `json:"source,omitempty"`
}

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

type UpdateExchangeRateRequest struct {
	Rate   *float64 `json:"rate,omitempty"`
	Source *string  `json:"source,omitempty"`
}

func (r UpdateExchangeRateRequest) Validate() error {
	if r.Rate == nil && r.Source == nil {
		return errors.New("at least one field must be provided")
	}
	if r.Rate != nil && *r.Rate <= 0 {
		return errors.New("rate must be positive")
	}
	return nil
}

type ConvertAmountRequest struct {
	Amount float64 `json:"amount"`
	From   string  `json:"from"`
	To     string  `json:"to"`
}

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

type ConvertAmountResponse struct {
	OriginalAmount  float64 `json:"original_amount"`
	ConvertedAmount float64 `json:"converted_amount"`
	From            string  `json:"from"`
	To              string  `json:"to"`
	Rate            float64 `json:"rate"`
}

type ExchangeRateListFilter struct {
	BaseCurrency   *string
	TargetCurrency *string
	PaginationParams
}
