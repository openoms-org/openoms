package ebay

import "errors"

var (
	ErrUnauthorized = errors.New("ebay: unauthorized")
	ErrForbidden    = errors.New("ebay: forbidden")
	ErrNotFound     = errors.New("ebay: not found")
	ErrRateLimited  = errors.New("ebay: rate limited")
	ErrServerError  = errors.New("ebay: server error")
)
