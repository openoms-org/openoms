package service

import (
	"errors"

	"golang.org/x/crypto/bcrypt"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
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

// ValidateStrength enforces the shared password policy (model.ValidatePassword:
// length, uppercase, lowercase, digit) plus the bcrypt 72-byte input limit that
// is specific to hashing. Previously this used a weaker, divergent rule
// (letter+digit) than request validation; it now shares the single validator.
func (s *PasswordService) ValidateStrength(password string) error {
	if err := model.ValidatePassword(password); err != nil {
		return err
	}
	// bcrypt silently truncates / errors past 72 bytes; reject early.
	if len(password) > 72 {
		return errors.New("password must not exceed 72 characters (bcrypt limit)")
	}
	return nil
}
