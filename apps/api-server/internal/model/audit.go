package model

import "time"

// AuditLogEntry is the read model for audit timeline display.
type AuditLogEntry struct {
	ID        int64             `json:"id"`
	UserName  *string           `json:"user_name,omitempty"`
	Action    string            `json:"action"`
	Changes   map[string]string `json:"changes"`
	IPAddress *string           `json:"ip_address,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
}
