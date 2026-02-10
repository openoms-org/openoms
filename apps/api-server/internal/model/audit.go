package model

import (
	"time"

	"github.com/google/uuid"
)

// AuditLogEntry is the read model for audit timeline display.
type AuditLogEntry struct {
	ID         int64             `json:"id"`
	UserName   *string           `json:"user_name,omitempty"`
	Action     string            `json:"action"`
	EntityType string            `json:"entity_type"`
	EntityID   string            `json:"entity_id"`
	Changes    map[string]string `json:"changes"`
	IPAddress  *string           `json:"ip_address,omitempty"`
	CreatedAt  time.Time         `json:"created_at"`
}

// AuditListFilter is used for the global admin audit log listing.
type AuditListFilter struct {
	EntityType *string
	Action     *string
	UserID     *uuid.UUID
	PaginationParams
}
