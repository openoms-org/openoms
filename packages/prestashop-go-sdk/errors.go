package prestashop

import (
	"errors"
	"fmt"
	"strings"
)

// Sentinel errors for API responses.
var (
	ErrUnauthorized = errors.New("prestashop: unauthorized")
	ErrForbidden    = errors.New("prestashop: forbidden")
	ErrNotFound     = errors.New("prestashop: not found")
	ErrRateLimited  = errors.New("prestashop: rate limited")
	ErrServerError  = errors.New("prestashop: server error")
)

// APIError represents an error response from the PrestaShop API.
type APIError struct {
	StatusCode int    `json:"-"`
	Code       int    `json:"code"`
	Message    string `json:"message"`
}

func (e *APIError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "prestashop: HTTP %d", e.StatusCode)
	if e.Code != 0 {
		fmt.Fprintf(&b, " [%d]", e.Code)
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
