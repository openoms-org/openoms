package olx

import "errors"

var (
	ErrUnauthorized = errors.New("olx: unauthorized")
	ErrForbidden    = errors.New("olx: forbidden")
	ErrNotFound     = errors.New("olx: not found")
	ErrRateLimited  = errors.New("olx: rate limited")
	ErrServerError  = errors.New("olx: server error")
)
