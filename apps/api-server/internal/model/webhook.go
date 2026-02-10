package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type WebhookEvent struct {
	ID        uuid.UUID       `json:"id"`
	TenantID  uuid.UUID       `json:"tenant_id"`
	Provider  string          `json:"provider"`
	EventType string          `json:"event_type"`
	Payload   json.RawMessage `json:"payload"`
	Status    string          `json:"status"`
	CreatedAt time.Time       `json:"created_at"`
}
