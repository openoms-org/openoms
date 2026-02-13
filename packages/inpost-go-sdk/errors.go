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
	Details    map[string]any    `json:"details"`
	Keys       map[string]string `json:"keys"`
}

func (e *APIError) Error() string {
	msg := fmt.Sprintf("inpost: api error %d", e.StatusCode)
	if e.Message != "" {
		msg = fmt.Sprintf("inpost: api error %d: %s", e.StatusCode, e.Message)
	}
	if len(e.Details) > 0 {
		msg += " ["
		first := true
		for k, v := range e.Details {
			if !first {
				msg += ", "
			}
			msg += fmt.Sprintf("%s: %v", k, v)
			first = false
		}
		msg += "]"
	}
	return msg
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
