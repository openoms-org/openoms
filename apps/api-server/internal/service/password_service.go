package service

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

// PasswordService handles bcrypt password hashing and validation.
type PasswordService struct {
	cost int
}

// NewPasswordService creates a PasswordService with bcrypt cost 12.
func NewPasswordService() *PasswordService {
	return &PasswordService{cost: 12}
}

// Hash generates a bcrypt hash of the password.
func (s *PasswordService) Hash(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), s.cost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// Compare checks a password against a bcrypt hash.
func (s *PasswordService) Compare(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

// ValidateStrength enforces minimum password requirements.
func (s *PasswordService) ValidateStrength(password string) error {
	if len(password) < 8 {
		return errors.New("password must be at least 8 characters")
	}
	if len(password) > 72 {
		return errors.New("password must not exceed 72 characters (bcrypt limit)")
	}

	hasLetter := false
	hasDigit := false
	for _, ch := range password {
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') {
			hasLetter = true
		}
		if ch >= '0' && ch <= '9' {
			hasDigit = true
		}
		if hasLetter && hasDigit {
			break
		}
	}
	if !hasLetter {
		return errors.New("password must contain at least one letter")
	}
	if !hasDigit {
		return errors.New("password must contain at least one digit")
	}

	return nil
}
