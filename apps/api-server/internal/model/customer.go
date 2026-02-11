package model

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Customer struct {
	ID                     uuid.UUID       `json:"id"`
	TenantID               uuid.UUID       `json:"tenant_id"`
	Email                  *string         `json:"email,omitempty"`
	Phone                  *string         `json:"phone,omitempty"`
	Name                   string          `json:"name"`
	CompanyName            *string         `json:"company_name,omitempty"`
	NIP                    *string         `json:"nip,omitempty"`
	DefaultShippingAddress json.RawMessage `json:"default_shipping_address,omitempty"`
	DefaultBillingAddress  json.RawMessage `json:"default_billing_address,omitempty"`
	Tags                   []string        `json:"tags"`
	Notes                  *string         `json:"notes,omitempty"`
	TotalOrders            int             `json:"total_orders"`
	TotalSpent             float64         `json:"total_spent"`
	PriceListID            *uuid.UUID      `json:"price_list_id,omitempty"`
	CreatedAt              time.Time       `json:"created_at"`
	UpdatedAt              time.Time       `json:"updated_at"`
}

type CreateCustomerRequest struct {
	Email                  *string         `json:"email,omitempty"`
	Phone                  *string         `json:"phone,omitempty"`
	Name                   string          `json:"name"`
	CompanyName            *string         `json:"company_name,omitempty"`
	NIP                    *string         `json:"nip,omitempty"`
	DefaultShippingAddress json.RawMessage `json:"default_shipping_address,omitempty"`
	DefaultBillingAddress  json.RawMessage `json:"default_billing_address,omitempty"`
	Tags                   []string        `json:"tags,omitempty"`
	Notes                  *string         `json:"notes,omitempty"`
}

func (r *CreateCustomerRequest) Validate() error {
	if strings.TrimSpace(r.Name) == "" {
		return errors.New("name is required")
	}
	if err := validateMaxLength("name", r.Name, 500); err != nil {
		return err
	}
	if err := validateMaxLengthPtr("email", r.Email, 255); err != nil {
		return err
	}
	if err := validateMaxLengthPtr("phone", r.Phone, 50); err != nil {
		return err
	}
	if err := validateMaxLengthPtr("company_name", r.CompanyName, 500); err != nil {
		return err
	}
	if err := validateMaxLengthPtr("nip", r.NIP, 20); err != nil {
		return err
	}
	if err := validateMaxLengthPtr("notes", r.Notes, 5000); err != nil {
		return err
	}
	return nil
}

type UpdateCustomerRequest struct {
	Email                  *string          `json:"email,omitempty"`
	Phone                  *string          `json:"phone,omitempty"`
	Name                   *string          `json:"name,omitempty"`
	CompanyName            *string          `json:"company_name,omitempty"`
	NIP                    *string          `json:"nip,omitempty"`
	DefaultShippingAddress json.RawMessage  `json:"default_shipping_address,omitempty"`
	DefaultBillingAddress  json.RawMessage  `json:"default_billing_address,omitempty"`
	Tags                   *[]string        `json:"tags,omitempty"`
	Notes                  *string          `json:"notes,omitempty"`
	PriceListID            *uuid.UUID       `json:"price_list_id,omitempty"`
}

func (r *UpdateCustomerRequest) Validate() error {
	if r.Email == nil && r.Phone == nil && r.Name == nil &&
		r.CompanyName == nil && r.NIP == nil &&
		r.DefaultShippingAddress == nil && r.DefaultBillingAddress == nil &&
		r.Tags == nil && r.Notes == nil && r.PriceListID == nil {
		return errors.New("at least one field must be provided")
	}
	if r.Name != nil {
		if strings.TrimSpace(*r.Name) == "" {
			return errors.New("name cannot be empty")
		}
		if err := validateMaxLength("name", *r.Name, 500); err != nil {
			return err
		}
	}
	if err := validateMaxLengthPtr("email", r.Email, 255); err != nil {
		return err
	}
	if err := validateMaxLengthPtr("phone", r.Phone, 50); err != nil {
		return err
	}
	if err := validateMaxLengthPtr("company_name", r.CompanyName, 500); err != nil {
		return err
	}
	if err := validateMaxLengthPtr("nip", r.NIP, 20); err != nil {
		return err
	}
	if err := validateMaxLengthPtr("notes", r.Notes, 5000); err != nil {
		return err
	}
	return nil
}

type CustomerListFilter struct {
	Search *string
	Tags   *string
	PaginationParams
}
