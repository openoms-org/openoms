package shopify

import (
	"errors"
	"fmt"
	"strings"
)

// Sentinel errors for API responses.
var (
	ErrUnauthorized     = errors.New("shopify: unauthorized")
	ErrForbidden        = errors.New("shopify: forbidden")
	ErrNotFound         = errors.New("shopify: not found")
	ErrRateLimited      = errors.New("shopify: rate limited")
	ErrServerError      = errors.New("shopify: server error")
	ErrResponseTooLarge = errors.New("shopify: response too large")
)

// APIError represents an error response from the Shopify API.
type APIError struct {
	StatusCode int               `json:"-"`
	Errors     map[string]string `json:"errors,omitempty"`
	Message    string            `json:"-"`
}

func (e *APIError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "shopify: HTTP %d", e.StatusCode)
	if e.Message != "" {
		fmt.Fprintf(&b, ": %s", e.Message)
	}
	if len(e.Errors) > 0 {
		for k, v := range e.Errors {
			fmt.Fprintf(&b, " %s=%s", k, v)
			break // just first for brevity
		}
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
