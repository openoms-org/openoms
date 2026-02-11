package woocommerce

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrUnauthorized = errors.New("woocommerce: unauthorized")
	ErrForbidden    = errors.New("woocommerce: forbidden")
	ErrNotFound     = errors.New("woocommerce: not found")
	ErrRateLimited  = errors.New("woocommerce: rate limited")
	ErrServerError  = errors.New("woocommerce: server error")
)

// APIError represents an error response from the WooCommerce API.
type APIError struct {
	StatusCode int    `json:"-"`
	Code       string `json:"code"`
	Message    string `json:"message"`
}

func (e *APIError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "woocommerce: HTTP %d", e.StatusCode)
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
