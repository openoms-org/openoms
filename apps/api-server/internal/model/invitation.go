package model

import (
	"time"

	"github.com/google/uuid"
)

// Invitation represents an invitation to register under an existing tenant.
type Invitation struct {
	ID        uuid.UUID  `json:"id"`
	TenantID  uuid.UUID  `json:"tenant_id"`
	Email     string     `json:"email"`
	Token     string     `json:"-"` // never expose token in list responses
	Role      string     `json:"role"`
	ExpiresAt time.Time  `json:"expires_at"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
	CreatedBy *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// CreateInvitationRequest is the body of POST /v1/invitations.
type CreateInvitationRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

// InvitationResponse is returned when creating an invitation (includes the token).
type InvitationResponse struct {
	Invitation
	InviteToken string `json:"invite_token,omitempty"` // only returned on creation
}
