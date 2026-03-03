package model

import (
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"unicode"
)

// Validation limits for user-facing input fields.
const (
	MaxEmailLength = 254
	MaxSlugLength  = 63
	MaxNameLength  = 200
	MinPasswordLen = 8
	MaxPasswordLen = 128
)

var slugPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,61}[a-z0-9]$`)

func validateMaxLength(field, value string, max int) error {
	if len(value) > max {
		return fmt.Errorf("%s exceeds maximum length of %d characters", field, max)
	}
	return nil
}

func validateMaxLengthPtr(field string, value *string, max int) error {
	if value != nil && len(*value) > max {
		return fmt.Errorf("%s exceeds maximum length of %d characters", field, max)
	}
	return nil
}

// validateEmail checks email format using net/mail (RFC 5322) and length.
func validateEmail(email string) error {
	email = strings.TrimSpace(email)
	if email == "" {
		return fmt.Errorf("email is required")
	}
	if len(email) > MaxEmailLength {
		return fmt.Errorf("email must not exceed %d characters", MaxEmailLength)
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return fmt.Errorf("invalid email format")
	}
	return nil
}

// validateSlug checks that slug is 3-63 chars, lowercase alphanumeric + hyphens.
func validateSlug(slug string) error {
	if !slugPattern.MatchString(slug) {
		return fmt.Errorf("slug must be 3-63 characters, lowercase alphanumeric and hyphens")
	}
	return nil
}

// validatePassword enforces minimum security requirements.
func validatePassword(password string) error {
	if len(password) < MinPasswordLen {
		return fmt.Errorf("password must be at least %d characters", MinPasswordLen)
	}
	if len(password) > MaxPasswordLen {
		return fmt.Errorf("password must not exceed %d characters", MaxPasswordLen)
	}
	var hasUpper, hasLower, hasDigit bool
	for _, c := range password {
		switch {
		case unicode.IsUpper(c):
			hasUpper = true
		case unicode.IsLower(c):
			hasLower = true
		case unicode.IsDigit(c):
			hasDigit = true
		}
	}
	if !hasUpper || !hasLower || !hasDigit {
		return fmt.Errorf("password must contain uppercase, lowercase, and digit")
	}
	return nil
}
