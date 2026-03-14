package shoper

import (
	"errors"
	"fmt"
	"strings"
)

// Sentinel errors for API responses.
var (
	ErrUnauthorized = errors.New("shoper: unauthorized")
	ErrForbidden    = errors.New("shoper: forbidden")
	ErrNotFound     = errors.New("shoper: not found")
	ErrRateLimited  = errors.New("shoper: rate limited")
	ErrServerError  = errors.New("shoper: server error")
)

// APIError represents an error response from the Shoper API.
type APIError struct {
	StatusCode int    `json:"-"`
	Code       string `json:"error"`
	Message    string `json:"error_description"`
}

func (e *APIError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "shoper: HTTP %d", e.StatusCode)
	if e.Code != "" {
		fmt.Fprintf(&b, " [%s]", e.Code)
	}
	if e.Message != "" {
		fmt.Fprintf(&b, ": %s", e.Message)
	}
	return b.String()
}

func (e *APIError) Unwrap() error {
	switch {
	case e.StatusCode == 401:
		return ErrUnauthorized
	case e.StatusCode == 403:
		return ErrForbidden
	case e.StatusCode == 404:
		return ErrNotFound
	case e.StatusCode == 429:
		return ErrRateLimited
	case e.StatusCode >= 500:
		return ErrServerError
	default:
		return nil
	}
}
