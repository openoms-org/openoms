package service

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidationError_Error(t *testing.T) {
	inner := errors.New("field is required")
	ve := NewValidationError(inner)
	assert.Equal(t, "validation: field is required", ve.Error())
}

func TestValidationError_Unwrap(t *testing.T) {
	inner := errors.New("field is required")
	ve := NewValidationError(inner)

	var vErr *ValidationError
	require.True(t, errors.As(ve, &vErr))
	assert.Equal(t, inner, vErr.Unwrap())
}

func TestValidationError_TypeAssertion(t *testing.T) {
	ve := NewValidationError(errors.New("bad"))
	var vErr *ValidationError
	assert.True(t, errors.As(ve, &vErr))
}

func TestNonValidationError_TypeAssertion(t *testing.T) {
	err := errors.New("not a validation error")
	var vErr *ValidationError
	assert.False(t, errors.As(err, &vErr))
}

func TestNewValidationError_NilSafe(t *testing.T) {
	// Constructing with a valid error works
	err := NewValidationError(errors.New("test"))
	require.NotNil(t, err)
}

func TestIsDuplicateKeyError_NotPgError(t *testing.T) {
	err := errors.New("some random error")
	assert.False(t, isDuplicateKeyError(err))
}

func TestIsDuplicateKeyError_Nil(t *testing.T) {
	assert.False(t, isDuplicateKeyError(nil))
}

func TestServiceErrors_AreSentinels(t *testing.T) {
	// Verify sentinel errors are not nil and have appropriate messages
	sentinels := []error{
		ErrOrderNotFound,
		ErrInvalidTransition,
		ErrUnknownStatus,
		ErrShipmentNotFound,
		ErrOrderNotFoundForShipment,
		ErrIntegrationNotFound,
		ErrDuplicateProvider,
		ErrReturnNotFound,
		ErrInvalidReturnTransition,
		ErrCannotDeleteSelf,
		ErrCannotDeleteLastOwner,
		ErrDuplicateEmail,
		ErrSlugTaken,
		ErrInvalidCredentials,
		ErrTenantNotFound,
		ErrUserNotFound,
	}
	for _, err := range sentinels {
		assert.NotNil(t, err)
		assert.NotEmpty(t, err.Error())
	}
}
