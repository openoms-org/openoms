package btp

import (
	"errors"
	"fmt"
)

// Sentinel errors for errors.Is() checks.
var (
	ErrUnauthorized = errors.New("btp: unauthorized")
	ErrForbidden    = errors.New("btp: forbidden")
	ErrNotFound     = errors.New("btp: not found")
	ErrValidation   = errors.New("btp: validation error")
	ErrServerError  = errors.New("btp: server error")
)

// APIError represents an error response from the BTP API.
type APIError struct {
	StatusCode int        `json:"-"`
	Result     *APIResult `json:"result,omitempty"`
	RawBody    string     `json:"-"`
}

// Error returns a human-readable error message.
func (e *APIError) Error() string {
	if e.Result != nil && len(e.Result.Messages) > 0 {
		return fmt.Sprintf("btp: api error %d: %s", e.StatusCode, e.Result.Messages[0].Message)
	}
	if e.RawBody != "" {
		return fmt.Sprintf("btp: api error %d: %s", e.StatusCode, e.RawBody)
	}
	return fmt.Sprintf("btp: api error %d", e.StatusCode)
}

// Unwrap returns the appropriate sentinel error based on HTTP status code.
func (e *APIError) Unwrap() error {
	switch {
	case e.StatusCode == 401:
		return ErrUnauthorized
	case e.StatusCode == 403:
		return ErrForbidden
	case e.StatusCode == 404:
		return ErrNotFound
	case e.StatusCode == 400:
		return ErrValidation
	case e.StatusCode >= 500:
		return ErrServerError
	default:
		return nil
	}
}
