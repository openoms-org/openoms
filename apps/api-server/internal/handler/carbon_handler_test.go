package handler

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCarbonHandler(t *testing.T) {
	h := NewCarbonHandler(nil)
	assert.NotNil(t, h)
}
