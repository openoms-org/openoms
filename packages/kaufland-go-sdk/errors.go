package kaufland

import "errors"

var (
	ErrUnauthorized = errors.New("kaufland: unauthorized")
	ErrForbidden    = errors.New("kaufland: forbidden")
	ErrNotFound     = errors.New("kaufland: not found")
	ErrRateLimited  = errors.New("kaufland: rate limited")
	ErrServerError  = errors.New("kaufland: server error")
)
