package inpost

import (
	"errors"
	"fmt"
)

var (
	ErrUnauthorized = errors.New("inpost: unauthorized")
	ErrNotFound     = errors.New("inpost: not found")
	ErrServerError  = errors.New("inpost: server error")
	ErrValidation   = errors.New("inpost: validation error")
)

// APIError represents an error response from the InPost API.
type APIError struct {
	StatusCode int               `json:"-"`
	Message    string            `json:"message"`
	Details    map[string]string `json:"details"`
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("inpost: api error %d: %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("inpost: api error %d", e.StatusCode)
}

func (e *APIError) Unwrap() error {
	switch {
	case e.StatusCode == 401:
		return ErrUnauthorized
	case e.StatusCode == 404:
		return ErrNotFound
	case e.StatusCode == 422:
		return ErrValidation
	case e.StatusCode >= 500:
		return ErrServerError
	default:
		return nil
	}
}
