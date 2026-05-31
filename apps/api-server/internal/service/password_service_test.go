package service

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPasswordService_Hash_And_Compare_RoundTrip(t *testing.T) {
	svc := NewPasswordService()

	hash, err := svc.Hash("StrongP@ss123")
	require.NoError(t, err)
	require.NotEmpty(t, hash)

	err = svc.Compare(hash, "StrongP@ss123")
	assert.NoError(t, err)
}

func TestPasswordService_Compare_WrongPassword(t *testing.T) {
	svc := NewPasswordService()

	hash, err := svc.Hash("password1")
	require.NoError(t, err)

	err = svc.Compare(hash, "password2")
	assert.Error(t, err)
}

func TestPasswordService_ValidateStrength_TooShort(t *testing.T) {
	svc := NewPasswordService()

	err := svc.ValidateStrength("Abc1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least 8 characters")
}

func TestPasswordService_ValidateStrength_TooLong(t *testing.T) {
	svc := NewPasswordService()

	// 73 characters: exceeds bcrypt's 72-byte limit
	longPassword := "Aa1" + strings.Repeat("a", 70)

	err := svc.ValidateStrength(longPassword)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "72")
}

func TestPasswordService_ValidateStrength_NoLetter(t *testing.T) {
	svc := NewPasswordService()

	err := svc.ValidateStrength("12345678")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "uppercase, lowercase, and digit")
}

// OPE-488: ValidateStrength now shares the stronger request validator, so a
// password lacking an uppercase letter (previously accepted by the weaker
// service-level rule) is rejected — matching registration/create/change.
func TestPasswordService_ValidateStrength_NoUppercaseRejected(t *testing.T) {
	svc := NewPasswordService()

	err := svc.ValidateStrength("password123")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "uppercase, lowercase, and digit")
}

func TestPasswordService_ValidateStrength_NoLowercaseRejected(t *testing.T) {
	svc := NewPasswordService()

	err := svc.ValidateStrength("PASSWORD123")
	require.Error(t, err)
}

func TestPasswordService_ValidateStrength_NoDigit(t *testing.T) {
	svc := NewPasswordService()

	err := svc.ValidateStrength("abcdefgh")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "digit")
}

func TestPasswordService_ValidateStrength_Valid(t *testing.T) {
	svc := NewPasswordService()

	err := svc.ValidateStrength("StrongP@ss123")
	assert.NoError(t, err)
}
