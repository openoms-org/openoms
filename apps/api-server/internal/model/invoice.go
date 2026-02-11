package model

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Invoice represents an invoice record in the system.
type Invoice struct {
	ID             uuid.UUID       `json:"id"`
	TenantID       uuid.UUID       `json:"tenant_id"`
	OrderID        uuid.UUID       `json:"order_id"`
	Provider       string          `json:"provider"`
	ExternalID     *string         `json:"external_id,omitempty"`
	ExternalNumber *string         `json:"external_number,omitempty"`
	Status         string          `json:"status"`
	InvoiceType    string          `json:"invoice_type"`
	TotalNet       *float64        `json:"total_net,omitempty"`
	TotalGross     *float64        `json:"total_gross,omitempty"`
	Currency       string          `json:"currency"`
	IssueDate      *time.Time      `json:"issue_date,omitempty"`
	DueDate        *time.Time      `json:"due_date,omitempty"`
	PDFURL         *string         `json:"pdf_url,omitempty"`
	Metadata       json.RawMessage `json:"metadata"`
	ErrorMessage   *string         `json:"error_message,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

// CreateInvoiceRequest is the payload for creating an invoice.
type CreateInvoiceRequest struct {
	OrderID       uuid.UUID `json:"order_id"`
	Provider      string    `json:"provider"`
	InvoiceType   string    `json:"invoice_type,omitempty"`
	CustomerName  string    `json:"customer_name"`
	CustomerEmail string    `json:"customer_email,omitempty"`
	NIP           string    `json:"nip,omitempty"`
	PaymentMethod string    `json:"payment_method,omitempty"`
	Notes         string    `json:"notes,omitempty"`
}

func (r *CreateInvoiceRequest) Validate() error {
	if r.OrderID == uuid.Nil {
		return errors.New("order_id is required")
	}
	if strings.TrimSpace(r.Provider) == "" {
		return errors.New("provider is required")
	}
	if r.InvoiceType == "" {
		r.InvoiceType = "vat"
	}
	if err := validateMaxLength("provider", r.Provider, 100); err != nil {
		return err
	}
	if err := validateMaxLength("customer_name", r.CustomerName, 500); err != nil {
		return err
	}
	if err := validateMaxLength("nip", r.NIP, 20); err != nil {
		return err
	}
	if err := validateMaxLength("notes", r.Notes, 5000); err != nil {
		return err
	}
	return nil
}

// InvoiceListFilter defines the filter parameters for listing invoices.
type InvoiceListFilter struct {
	Status   *string
	Provider *string
	OrderID  *uuid.UUID
	PaginationParams
}
