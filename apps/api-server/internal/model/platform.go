package model

import (
	"slices"
	"time"

	"github.com/google/uuid"
)

// Platform-admin permissions. Colon-style to visibly differ from tenant
// dot-style permissions (e.g. "orders.view"). No tenant role grants these.
const (
	PermPlatformProvidersRead     = "providers:read"
	PermPlatformProvidersWrite    = "providers:write"
	PermPlatformProvidersValidate = "providers:validate"
	PermPlatformProvidersPublish  = "providers:publish"
	PermPlatformProvidersSecrets  = "providers:secrets"
)

// PlatformAdmin is an OpenOMS operator with platform-level (cross-tenant)
// provider-administration powers. Identity is separate from tenant RBAC: the
// user authenticates normally, but these permissions live only in the
// platform_admins table and are never granted through tenant roles.
type PlatformAdmin struct {
	ID          uuid.UUID `json:"id"`
	UserID      uuid.UUID `json:"user_id"`
	Permissions []string  `json:"permissions"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// HasPermission reports whether the admin holds the given platform permission.
func (a *PlatformAdmin) HasPermission(perm string) bool {
	return slices.Contains(a.Permissions, perm)
}

// PlatformAuditEntry is a single platform-admin audit record. It is not
// tenant-scoped; ActorUserID identifies the acting platform admin's user.
type PlatformAuditEntry struct {
	ActorUserID uuid.UUID
	Action      string
	TargetType  string
	TargetID    string
	Metadata    any
	IPAddress   string
}
