package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewLicenseRepository(t *testing.T) {
	repo := NewLicenseRepository()
	assert.NotNil(t, repo)

	// Verify it implements the interface
	var _ LicenseRepo = repo
}
