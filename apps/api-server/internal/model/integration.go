package model

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

type Integration struct {
	ID             uuid.UUID       `json:"id"`
	TenantID       uuid.UUID       `json:"tenant_id"`
	Provider       string          `json:"provider"`
	Status         string          `json:"status"`
	HasCredentials bool            `json:"has_credentials"`
	Settings       json.RawMessage `json:"settings"`
	LastSyncAt     *time.Time      `json:"last_sync_at,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

// IntegrationWithCreds is internal only — never returned via API.
type IntegrationWithCreds struct {
	Integration
	EncryptedCredentials string
}

type CreateIntegrationRequest struct {
	Provider    string          `json:"provider"`
	Credentials json.RawMessage `json:"credentials"`
	Settings    json.RawMessage `json:"settings,omitempty"`
}

func (r *CreateIntegrationRequest) Validate() error {
	switch r.Provider {
	case "allegro", "inpost", "dhl", "dpd", "woocommerce":
		// valid
	default:
		return errors.New("provider must be one of: allegro, inpost, dhl, dpd, woocommerce")
	}
	if len(r.Credentials) == 0 {
		return errors.New("credentials are required")
	}
	return nil
}

type UpdateIntegrationRequest struct {
	Status      *string          `json:"status,omitempty"`
	Credentials *json.RawMessage `json:"credentials,omitempty"`
	Settings    *json.RawMessage `json:"settings,omitempty"`
}

func (r *UpdateIntegrationRequest) Validate() error {
	if r.Status == nil && r.Credentials == nil && r.Settings == nil {
		return errors.New("at least one field must be provided")
	}
	if r.Status != nil {
		switch *r.Status {
		case "active", "inactive", "error":
			// valid
		default:
			return errors.New("status must be one of: active, inactive, error")
		}
	}
	return nil
}
