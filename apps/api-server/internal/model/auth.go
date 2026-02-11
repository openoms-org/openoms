package model

import (
	"errors"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// AuthClaims are the JWT token claims. Lives in model package
// to avoid circular dependencies between service and middleware.
type AuthClaims struct {
	jwt.RegisteredClaims
	TenantID uuid.UUID `json:"tid"`
	Email    string    `json:"email"`
	Role     string    `json:"role"`
	RoleID   uuid.UUID `json:"role_id,omitempty"`
	Type     string    `json:"type,omitempty"`
}

// LoginRequest is the body of POST /v1/auth/login.
type LoginRequest struct {
	Email      string `json:"email"`
	Password   string `json:"password"`
	TenantSlug string `json:"tenant_slug"`
}

func (r *LoginRequest) Validate() error {
	if strings.TrimSpace(r.Email) == "" {
		return errors.New("email is required")
	}
	if r.Password == "" {
		return errors.New("password is required")
	}
	if strings.TrimSpace(r.TenantSlug) == "" {
		return errors.New("tenant_slug is required")
	}
	return nil
}

// RegisterRequest is the body of POST /v1/auth/register.
type RegisterRequest struct {
	Email      string `json:"email"`
	Password   string `json:"password"`
	Name       string `json:"name"`
	TenantName string `json:"tenant_name"`
	TenantSlug string `json:"tenant_slug"`
}

func (r *RegisterRequest) Validate() error {
	if strings.TrimSpace(r.Email) == "" {
		return errors.New("email is required")
	}
	if len(r.Password) < 8 {
		return errors.New("password must be at least 8 characters")
	}
	if strings.TrimSpace(r.Name) == "" {
		return errors.New("name is required")
	}
	if strings.TrimSpace(r.TenantName) == "" {
		return errors.New("tenant_name is required")
	}
	if strings.TrimSpace(r.TenantSlug) == "" {
		return errors.New("tenant_slug is required")
	}
	return nil
}

// TokenResponse is returned by login and register endpoints.
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	User        User   `json:"user"`
	Tenant      Tenant `json:"tenant"`
}

// CreateUserRequest is the body of POST /v1/users.
type CreateUserRequest struct {
	Email string `json:"email"`
	Name  string `json:"name"`
	Role  string `json:"role"`
}

func (r *CreateUserRequest) Validate() error {
	if strings.TrimSpace(r.Email) == "" {
		return errors.New("email is required")
	}
	if strings.TrimSpace(r.Name) == "" {
		return errors.New("name is required")
	}
	switch r.Role {
	case "owner", "admin", "member":
		// valid
	default:
		return errors.New("role must be one of: owner, admin, member")
	}
	return nil
}

// UpdateUserRequest is the body of PATCH /v1/users/{id}.
type UpdateUserRequest struct {
	Name   *string    `json:"name,omitempty"`
	Role   *string    `json:"role,omitempty"`
	RoleID *uuid.UUID `json:"role_id,omitempty"`
}

func (r *UpdateUserRequest) Validate() error {
	if r.Name == nil && r.Role == nil && r.RoleID == nil {
		return errors.New("at least one field (name, role, or role_id) must be provided")
	}
	if r.Role != nil {
		switch *r.Role {
		case "owner", "admin", "member":
			// valid
		default:
			return errors.New("role must be one of: owner, admin, member")
		}
	}
	return nil
}
