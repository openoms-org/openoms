package service

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
)

// ValidationError wraps a validation error from the service layer.
// Handlers can use errors.As to detect it.
type ValidationError struct {
	Err error
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation: %s", e.Err.Error())
}

func (e *ValidationError) Unwrap() error {
	return e.Err
}

// NewValidationError wraps err as a ValidationError.
func NewValidationError(err error) error {
	return &ValidationError{Err: err}
}

// isDuplicateKeyError checks for PostgreSQL unique constraint violation (code 23505).
func isDuplicateKeyError(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}

// IsForeignKeyError checks for PostgreSQL foreign key constraint violation (code 23503).
func IsForeignKeyError(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23503"
	}
	return false
}

// ErrFreshdeskNotConfigured is returned when Freshdesk is not enabled or configured for the tenant.
var ErrFreshdeskNotConfigured = errors.New("freshdesk is not configured")
