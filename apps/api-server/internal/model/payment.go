package model

import (
	"time"

	"github.com/google/uuid"
)

// PaymentSettlement represents an imported gateway payout/settlement.
type PaymentSettlement struct {
	ID             uuid.UUID  `json:"id"`
	TenantID       uuid.UUID  `json:"tenant_id"`
	Provider       string     `json:"provider"`
	SettlementID   *string    `json:"settlement_id,omitempty"`
	SettlementDate string     `json:"settlement_date"` // DATE as YYYY-MM-DD
	TotalAmount    float64    `json:"total_amount"`
	FeeAmount      float64    `json:"fee_amount"`
	NetAmount      float64    `json:"net_amount"`
	Currency       string     `json:"currency"`
	Status         string     `json:"status"`
	Notes          *string    `json:"notes,omitempty"`
	ImportedAt     time.Time  `json:"imported_at"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// PaymentSettlementWithTransactions combines a settlement with its transactions.
type PaymentSettlementWithTransactions struct {
	PaymentSettlement
	Transactions []PaymentTransaction `json:"transactions"`
}

// PaymentTransaction represents an individual payment transaction within a settlement.
type PaymentTransaction struct {
	ID                    uuid.UUID  `json:"id"`
	TenantID              uuid.UUID  `json:"tenant_id"`
	SettlementID          *uuid.UUID `json:"settlement_id,omitempty"`
	OrderID               *uuid.UUID `json:"order_id,omitempty"`
	Provider              string     `json:"provider"`
	ExternalTransactionID *string    `json:"external_transaction_id,omitempty"`
	Amount                float64    `json:"amount"`
	Fee                   float64    `json:"fee"`
	NetAmount             float64    `json:"net_amount"`
	Currency              string     `json:"currency"`
	TransactionType       string     `json:"transaction_type"`
	TransactionDate       time.Time  `json:"transaction_date"`
	MatchStatus           string     `json:"match_status"`
	MatchNotes            *string    `json:"match_notes,omitempty"`
	CreatedAt             time.Time  `json:"created_at"`
}

// CreateSettlementRequest is the request body for creating a settlement with transactions.
type CreateSettlementRequest struct {
	Provider       string                          `json:"provider"`
	SettlementID   *string                         `json:"settlement_id,omitempty"`
	SettlementDate string                          `json:"settlement_date"`
	TotalAmount    float64                         `json:"total_amount"`
	FeeAmount      float64                         `json:"fee_amount"`
	NetAmount      float64                         `json:"net_amount"`
	Currency       string                          `json:"currency,omitempty"`
	Notes          *string                         `json:"notes,omitempty"`
	Transactions   []CreateTransactionRequest      `json:"transactions,omitempty"`
}

// CreateTransactionRequest is the request body for a single transaction within an import.
type CreateTransactionRequest struct {
	ExternalTransactionID *string  `json:"external_transaction_id,omitempty"`
	Amount                float64  `json:"amount"`
	Fee                   float64  `json:"fee"`
	NetAmount             float64  `json:"net_amount"`
	Currency              string   `json:"currency,omitempty"`
	TransactionType       string   `json:"transaction_type,omitempty"`
	TransactionDate       string   `json:"transaction_date"`
}

// ManualMatchRequest is the request body for manually matching a transaction to an order.
type ManualMatchRequest struct {
	OrderID uuid.UUID `json:"order_id"`
	Notes   *string   `json:"notes,omitempty"`
}

// ReconciliationSummary contains aggregate reconciliation statistics.
type ReconciliationSummary struct {
	TotalTransactions int     `json:"total_transactions"`
	MatchedCount      int     `json:"matched_count"`
	UnmatchedCount    int     `json:"unmatched_count"`
	DiscrepancyCount  int     `json:"discrepancy_count"`
	ManualMatchCount  int     `json:"manual_match_count"`
	TotalAmount       float64 `json:"total_amount"`
	TotalFees         float64 `json:"total_fees"`
	TotalNet          float64 `json:"total_net"`
	MatchedAmount     float64 `json:"matched_amount"`
	UnmatchedAmount   float64 `json:"unmatched_amount"`
}

// MatchResult represents the result of an auto-match attempt for a single transaction.
type MatchResult struct {
	TransactionID uuid.UUID  `json:"transaction_id"`
	OrderID       *uuid.UUID `json:"order_id,omitempty"`
	Status        string     `json:"status"` // matched, unmatched, discrepancy
	Notes         string     `json:"notes,omitempty"`
}

// AutoMatchResponse is returned after running auto-match on a settlement.
type AutoMatchResponse struct {
	SettlementID uuid.UUID     `json:"settlement_id"`
	Results      []MatchResult `json:"results"`
	Matched      int           `json:"matched"`
	Unmatched    int           `json:"unmatched"`
	Discrepancy  int           `json:"discrepancy"`
}

// SettlementListFilter defines filters for listing settlements.
type SettlementListFilter struct {
	PaginationParams
	Provider *string `json:"provider,omitempty"`
	Status   *string `json:"status,omitempty"`
	DateFrom *string `json:"date_from,omitempty"`
	DateTo   *string `json:"date_to,omitempty"`
}

// TransactionListFilter defines filters for listing transactions.
type TransactionListFilter struct {
	PaginationParams
	SettlementID    *uuid.UUID `json:"settlement_id,omitempty"`
	MatchStatus     *string    `json:"match_status,omitempty"`
	Provider        *string    `json:"provider,omitempty"`
	TransactionType *string    `json:"transaction_type,omitempty"`
	DateFrom        *string    `json:"date_from,omitempty"`
	DateTo          *string    `json:"date_to,omitempty"`
}
