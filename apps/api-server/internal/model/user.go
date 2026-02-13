package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// User represents a user in the system (JSON-safe, no password_hash).
type User struct {
	ID           uuid.UUID  `json:"id"`
	TenantID     uuid.UUID  `json:"tenant_id"`
	Email        string     `json:"email"`
	Name         string     `json:"name"`
	Role         string     `json:"role"`
	RoleID       *uuid.UUID `json:"role_id,omitempty"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty"`
	LastLogoutAt *time.Time `json:"last_logout_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// Tenant represents a tenant organization.
type Tenant struct {
	ID        uuid.UUID       `json:"id"`
	Name      string          `json:"name"`
	Slug      string          `json:"slug"`
	Plan      string          `json:"plan"`
	Settings  json.RawMessage `json:"settings"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

type EmailSettings struct {
	Enabled   bool     `json:"enabled"`
	SMTPHost  string   `json:"smtp_host"`
	SMTPPort  int      `json:"smtp_port"`
	SMTPUser  string   `json:"smtp_user"`
	SMTPPass  string   `json:"smtp_pass"`
	FromEmail string   `json:"from_email"`
	FromName  string   `json:"from_name"`
	NotifyOn  []string `json:"notify_on"`
}

type CompanySettings struct {
	CompanyName string `json:"company_name"`
	LogoURL     string `json:"logo_url"`
	Address     string `json:"address"`
	City        string `json:"city"`
	PostCode    string `json:"post_code"`
	NIP         string `json:"nip"`
	Phone       string `json:"phone"`
	Email       string `json:"email"`
	Website     string `json:"website"`
}

type SMSSettings struct {
	Enabled   bool              `json:"enabled"`
	APIToken  string            `json:"api_token"`
	From      string            `json:"from"`
	NotifyOn  []string          `json:"notify_on"`
	Templates map[string]string `json:"templates"`
}

// InventorySettings controls warehouse inventory behaviour for a tenant.
type InventorySettings struct {
	StrictMode bool `json:"strict_mode"`
}

// AuditEntry represents an audit log record.
type AuditEntry struct {
	TenantID   uuid.UUID `json:"tenant_id"`
	UserID     uuid.UUID `json:"user_id"`
	Action     string    `json:"action"`
	EntityType string    `json:"entity_type"`
	EntityID   uuid.UUID `json:"entity_id"`
	Changes    any       `json:"changes,omitempty"`
	IPAddress  string    `json:"ip_address,omitempty"`
}
