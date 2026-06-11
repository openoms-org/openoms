package model

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNextOutboxBackoff(t *testing.T) {
	assert.Equal(t, 30*time.Second, NextOutboxBackoff(0))
	assert.Equal(t, 60*time.Second, NextOutboxBackoff(1))
	assert.Equal(t, 120*time.Second, NextOutboxBackoff(2))
	// Monotonic non-decreasing and capped at 1h.
	prev := time.Duration(0)
	for a := 0; a <= 20; a++ {
		d := NextOutboxBackoff(a)
		assert.GreaterOrEqual(t, d, prev)
		assert.LessOrEqual(t, d, time.Hour)
		prev = d
	}
	assert.Equal(t, time.Hour, NextOutboxBackoff(20))
}

func TestPermanentError(t *testing.T) {
	base := errors.New("boom")
	perr := Permanent(base)
	assert.True(t, IsPermanent(perr))
	assert.True(t, errors.Is(perr, base), "PermanentError unwraps to the cause")
	assert.False(t, IsPermanent(base))
	assert.False(t, IsPermanent(nil))
	// Wrapped permanent error is still detected.
	assert.True(t, IsPermanent(errors.Join(errors.New("ctx"), perr)))
}
