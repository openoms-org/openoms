package allegro

import (
	"errors"
	"fmt"
	"strings"
)

// ErrUnauthorized is returned when the API responds with 401 Unauthorized.
var ErrUnauthorized = errors.New("allegro: unauthorized")

// ErrForbidden is returned when the API responds with 403 Forbidden.
var ErrForbidden = errors.New("allegro: forbidden")

// ErrNotFound is returned when the API responds with 404 Not Found.
var ErrNotFound = errors.New("allegro: not found")

// ErrRateLimited is returned when the API responds with 429 Too Many Requests.
var ErrRateLimited = errors.New("allegro: rate limited")

// ErrServerError is returned when the API responds with a 5xx status code.
var ErrServerError = errors.New("allegro: server error")

// APIError represents an error response from the Allegro API.
type APIError struct {
	StatusCode int           `json:"-"`
	Code       string        `json:"code"`
	Message    string        `json:"message"`
	Details    []ErrorDetail `json:"errors"`
}

// ErrorDetail represents a single validation or field-level error.
type ErrorDetail struct {
	Code        string `json:"code"`
	Message     string `json:"message"`
	UserMessage string `json:"userMessage"`
	Details     any    `json:"details"`
	Path        string `json:"path"`
}

func (e *APIError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "allegro: HTTP %d", e.StatusCode)
	if e.Code != "" {
		fmt.Fprintf(&b, " [%s]", e.Code)
	}
	if e.Message != "" {
		fmt.Fprintf(&b, ": %s", e.Message)
	}
	for _, d := range e.Details {
		msg := d.Message
		if d.UserMessage != "" && d.UserMessage != msg {
			msg = d.UserMessage
		}
		if d.Details != nil {
			msg += fmt.Sprintf(" | %v", d.Details)
		}
		fmt.Fprintf(&b, "\n  - %s: %s (path: %s)", d.Code, msg, d.Path)
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
