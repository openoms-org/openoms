package service

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func generateTestKeypair(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	return pub, priv
}

func signLicenseToken(t *testing.T, privKey ed25519.PrivateKey, claims model.LicenseClaims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	signed, err := token.SignedString(privKey)
	require.NoError(t, err)
	return signed
}

func TestLicenseService_VerifyToken_Valid(t *testing.T) {
	pub, priv := generateTestKeypair(t)
	svc := NewLicenseService(pub, nil)

	jti := uuid.New()
	claims := model.LicenseClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    model.LicenseIssuer,
			Subject:   model.LicenseSubject,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		Email: "jan@firma.pl",
		Plan:  "plus",
		JTI:   jti,
		Limits: model.LicenseLimits{
			MaxUsers:         10,
			MaxOrdersMonthly: 5000,
		},
	}
	tokenStr := signLicenseToken(t, priv, claims)

	result, err := svc.ParseAndVerify(tokenStr)
	require.NoError(t, err)
	assert.Equal(t, "jan@firma.pl", result.Email)
	assert.Equal(t, "plus", result.Plan)
	assert.Equal(t, jti, result.JTI)
	assert.Equal(t, 10, result.Limits.MaxUsers)
}

func TestLicenseService_VerifyToken_ExpiredToken(t *testing.T) {
	pub, priv := generateTestKeypair(t)
	svc := NewLicenseService(pub, nil)

	claims := model.LicenseClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    model.LicenseIssuer,
			Subject:   model.LicenseSubject,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
		Email: "jan@firma.pl",
		Plan:  "standard",
		JTI:   uuid.New(),
	}
	tokenStr := signLicenseToken(t, priv, claims)

	_, err := svc.ParseAndVerify(tokenStr)
	assert.ErrorIs(t, err, ErrLicenseTokenExpired)
}

func TestLicenseService_VerifyToken_WrongIssuer(t *testing.T) {
	pub, priv := generateTestKeypair(t)
	svc := NewLicenseService(pub, nil)

	claims := model.LicenseClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "evil-issuer",
			Subject:   model.LicenseSubject,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		},
		Email: "jan@firma.pl",
		Plan:  "plus",
		JTI:   uuid.New(),
	}
	tokenStr := signLicenseToken(t, priv, claims)

	_, err := svc.ParseAndVerify(tokenStr)
	assert.ErrorIs(t, err, ErrLicenseTokenInvalid)
}

func TestLicenseService_VerifyToken_WrongSubject(t *testing.T) {
	pub, priv := generateTestKeypair(t)
	svc := NewLicenseService(pub, nil)

	claims := model.LicenseClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    model.LicenseIssuer,
			Subject:   "not-a-license",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		},
		Email: "jan@firma.pl",
		Plan:  "plus",
		JTI:   uuid.New(),
	}
	tokenStr := signLicenseToken(t, priv, claims)

	_, err := svc.ParseAndVerify(tokenStr)
	assert.ErrorIs(t, err, ErrLicenseTokenInvalid)
}

func TestLicenseService_VerifyToken_WrongKey(t *testing.T) {
	pub, _ := generateTestKeypair(t)
	_, wrongPriv := generateTestKeypair(t)
	svc := NewLicenseService(pub, nil)

	claims := model.LicenseClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    model.LicenseIssuer,
			Subject:   model.LicenseSubject,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		},
		Email: "jan@firma.pl",
		Plan:  "plus",
		JTI:   uuid.New(),
	}
	tokenStr := signLicenseToken(t, wrongPriv, claims)

	_, err := svc.ParseAndVerify(tokenStr)
	assert.Error(t, err)
}

func TestLicenseService_VerifyToken_InvalidPlan(t *testing.T) {
	pub, priv := generateTestKeypair(t)
	svc := NewLicenseService(pub, nil)

	claims := model.LicenseClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    model.LicenseIssuer,
			Subject:   model.LicenseSubject,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		},
		Email: "jan@firma.pl",
		Plan:  "mega-plan",
		JTI:   uuid.New(),
	}
	tokenStr := signLicenseToken(t, priv, claims)

	_, err := svc.ParseAndVerify(tokenStr)
	assert.ErrorIs(t, err, ErrLicenseTokenInvalid)
}

func TestLicenseService_Nil_PublicKey(t *testing.T) {
	svc := NewLicenseService(nil, nil)
	assert.True(t, svc.IsDisabled())
}
