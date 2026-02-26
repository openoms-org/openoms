package model

import (
	"errors"
	"slices"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// ValidPlans lists the accepted plan tiers for license tokens.
var ValidPlans = []string{"standard", "plus", "pro"}

// LicenseIssuer is the expected issuer claim in license JWTs.
const LicenseIssuer = "openoms-cloud"

// LicenseSubject is the expected subject claim in license JWTs.
const LicenseSubject = "license"

// LicenseLimits defines plan-specific resource limits encoded in the license token.
type LicenseLimits struct {
	MaxUsers         int `json:"max_users,omitempty"`
	MaxOrdersMonthly int `json:"max_orders_monthly,omitempty"`
	MaxIntegrations  int `json:"max_integrations,omitempty"`
}

// LicenseClaims represents the claims in a license JWT token.
type LicenseClaims struct {
	jwt.RegisteredClaims
	Email  string        `json:"email"`
	Plan   string        `json:"plan"`
	Limits LicenseLimits `json:"limits"`
	JTI    uuid.UUID     `json:"jti"`
}

// Validate checks that required fields are present and plan is valid.
func (c *LicenseClaims) Validate() error {
	if strings.TrimSpace(c.Email) == "" {
		return errors.New("license token: email is required")
	}
	if !isValidPlan(c.Plan) {
		return errors.New("license token: plan must be one of: standard, plus, pro")
	}
	if c.JTI == uuid.Nil {
		return errors.New("license token: jti is required")
	}
	return nil
}

func isValidPlan(plan string) bool {
	return slices.Contains(ValidPlans, plan)
}
