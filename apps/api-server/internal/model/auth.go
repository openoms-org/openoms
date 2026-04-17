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

// Validate validates the login request.
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
	Email                   string         `json:"email"`
	Password                string         `json:"password"`
	Name                    string         `json:"name"`
	Language                string         `json:"language,omitempty"`
	TenantName              string         `json:"tenant_name"`
	TenantSlug              string         `json:"tenant_slug"`
	InviteToken             string         `json:"invite_token,omitempty"`
	LicenseToken            string         `json:"license_token,omitempty"`
	CheckoutSessionID       string         `json:"checkout_session_id,omitempty"`
	Plan                    string         `json:"-"` // set internally from license/checkout, never from user input
	PlanLimits              *LicenseLimits `json:"-"` // set internally from license/checkout
	CheckoutSessionInterval string         `json:"-"` // set internally from checkout session
}

// Validate validates the register request.
func (r *RegisterRequest) Validate() error {
	if err := validateEmail(r.Email); err != nil {
		return err
	}
	if err := validatePassword(r.Password); err != nil {
		return err
	}
	if strings.TrimSpace(r.Name) == "" {
		return errors.New("name is required")
	}
	if err := validateMaxLength("name", r.Name, MaxNameLength); err != nil {
		return err
	}
	if strings.TrimSpace(r.TenantName) == "" {
		return errors.New("tenant_name is required")
	}
	if err := validateMaxLength("tenant_name", r.TenantName, MaxNameLength); err != nil {
		return err
	}
	if r.Language != "" {
		switch r.Language {
		case "pl", "en":
			// valid
		default:
			return errors.New("language must be one of: pl, en")
		}
	}
	return validateSlug(r.TenantSlug)
}

// UpdateMeRequest is the body of PATCH /v1/users/me (self-service).
type UpdateMeRequest struct {
	Language *string `json:"language,omitempty"`
}

// Validate validates the update-me request.
func (r *UpdateMeRequest) Validate() error {
	if r.Language == nil {
		return errors.New("language is required")
	}
	switch *r.Language {
	case "pl", "en":
		return nil
	default:
		return errors.New("language must be one of: pl, en")
	}
}

// TokenResponse is returned by login and register endpoints.
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	User        User   `json:"user"`
	Tenant      Tenant `json:"tenant"`
}

// LoginResponse is the unified login response that supports 2FA.
type LoginResponse struct {
	// Normal login fields (present when 2FA is not required)
	AccessToken string  `json:"access_token,omitempty"`
	ExpiresIn   int     `json:"expires_in,omitempty"`
	User        *User   `json:"user,omitempty"`
	Tenant      *Tenant `json:"tenant,omitempty"`

	// 2FA fields (present when 2FA is required)
	Requires2FA bool   `json:"requires_2fa,omitempty"`
	TempToken   string `json:"temp_token,omitempty"`
}

// TwoFALoginRequest is the body of POST /v1/auth/2fa/login.
type TwoFALoginRequest struct {
	TempToken string `json:"temp_token"`
	Code      string `json:"code"`
}

// TwoFAVerifyRequest is the body of POST /v1/auth/2fa/verify.
type TwoFAVerifyRequest struct {
	Code string `json:"code"`
}

// TwoFADisableRequest is the body of POST /v1/auth/2fa/disable.
type TwoFADisableRequest struct {
	Password string `json:"password"`
	Code     string `json:"code"`
}

// TwoFASetupRequest is the body of POST /v1/auth/2fa/setup. Requires password
// re-authentication to prevent session-hijack amplification.
type TwoFASetupRequest struct {
	Password string `json:"password"`
}

// TwoFASetupResponse is returned by POST /v1/auth/2fa/setup.
type TwoFASetupResponse struct {
	Secret string `json:"secret"`
	QRURL  string `json:"qr_url"`
}

// TwoFAStatusResponse is returned by GET /v1/auth/2fa/status.
type TwoFAStatusResponse struct {
	Enabled    bool   `json:"enabled"`
	VerifiedAt string `json:"verified_at,omitempty"`
}

// CreateUserRequest is the body of POST /v1/users.
type CreateUserRequest struct {
	Email string `json:"email"`
	Name  string `json:"name"`
	Role  string `json:"role"`

	// Transient: plan limit injected by handler, not from JSON
	MaxUsers int `json:"-"`
}

// Validate validates the create user request.
func (r *CreateUserRequest) Validate() error {
	if err := validateEmail(r.Email); err != nil {
		return err
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
	Name     *string    `json:"name,omitempty"`
	Role     *string    `json:"role,omitempty"`
	RoleID   *uuid.UUID `json:"role_id,omitempty"`
	Language *string    `json:"language,omitempty"`
}

// Validate validates the update user request.
func (r *UpdateUserRequest) Validate() error {
	if r.Name == nil && r.Role == nil && r.RoleID == nil && r.Language == nil {
		return errors.New("at least one field (name, role, role_id, or language) must be provided")
	}
	if r.Role != nil {
		switch *r.Role {
		case "owner", "admin", "member":
			// valid
		default:
			return errors.New("role must be one of: owner, admin, member")
		}
	}
	if r.Language != nil && *r.Language != "" {
		switch *r.Language {
		case "pl", "en":
			// valid
		default:
			return errors.New("language must be one of: pl, en")
		}
	}
	return nil
}
