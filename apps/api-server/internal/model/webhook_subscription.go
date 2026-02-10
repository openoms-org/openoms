package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// WebhookConfig is stored in tenants.settings["webhooks"]
type WebhookConfig struct {
	Endpoints []WebhookEndpoint `json:"endpoints"`
}

type WebhookEndpoint struct {
	ID     string   `json:"id"`
	Name   string   `json:"name"`
	URL    string   `json:"url"`
	Secret string   `json:"secret"`
	Events []string `json:"events"`
	Active bool     `json:"active"`
}

func DefaultWebhookConfig() WebhookConfig {
	return WebhookConfig{Endpoints: []WebhookEndpoint{}}
}

// WebhookDelivery is a log entry for an outgoing webhook call
type WebhookDelivery struct {
	ID           uuid.UUID       `json:"id"`
	TenantID     uuid.UUID       `json:"tenant_id"`
	URL          string          `json:"url"`
	EventType    string          `json:"event_type"`
	Payload      json.RawMessage `json:"payload"`
	Status       string          `json:"status"`
	ResponseCode *int            `json:"response_code,omitempty"`
	Error        *string         `json:"error,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
}

type WebhookDeliveryFilter struct {
	EventType *string
	Status    *string
	PaginationParams
}
