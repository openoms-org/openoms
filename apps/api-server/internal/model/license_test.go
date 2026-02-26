package model

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestLicenseClaims_Validate_Valid(t *testing.T) {
	claims := &LicenseClaims{
		Email: "jan@firma.pl",
		Plan:  "plus",
		JTI:   uuid.New(),
	}
	assert.NoError(t, claims.Validate())
}

func TestLicenseClaims_Validate_MissingEmail(t *testing.T) {
	claims := &LicenseClaims{
		Plan: "plus",
		JTI:  uuid.New(),
	}
	assert.ErrorContains(t, claims.Validate(), "email")
}

func TestLicenseClaims_Validate_MissingPlan(t *testing.T) {
	claims := &LicenseClaims{
		Email: "jan@firma.pl",
		JTI:   uuid.New(),
	}
	assert.ErrorContains(t, claims.Validate(), "plan")
}

func TestLicenseClaims_Validate_InvalidPlan(t *testing.T) {
	claims := &LicenseClaims{
		Email: "jan@firma.pl",
		Plan:  "mega-ultra",
		JTI:   uuid.New(),
	}
	assert.ErrorContains(t, claims.Validate(), "plan")
}

func TestLicenseClaims_Validate_ZeroJTI(t *testing.T) {
	claims := &LicenseClaims{
		Email: "jan@firma.pl",
		Plan:  "standard",
	}
	assert.ErrorContains(t, claims.Validate(), "jti")
}
