package config

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_ParseLicensePublicKey_Valid(t *testing.T) {
	pub, _, _ := ed25519.GenerateKey(rand.Reader)
	encoded := base64.StdEncoding.EncodeToString(pub)

	cfg := &Config{LicensePublicKey: encoded}
	key, err := cfg.ParseLicensePublicKey()
	require.NoError(t, err)
	assert.Equal(t, pub, key)
}

func TestConfig_ParseLicensePublicKey_Empty(t *testing.T) {
	cfg := &Config{LicensePublicKey: ""}
	key, err := cfg.ParseLicensePublicKey()
	assert.NoError(t, err)
	assert.Nil(t, key)
}

func TestConfig_ParseLicensePublicKey_Invalid(t *testing.T) {
	cfg := &Config{LicensePublicKey: "not-valid-base64!!!"}
	_, err := cfg.ParseLicensePublicKey()
	assert.Error(t, err)
}

func TestConfig_ParseLicensePublicKey_WrongLength(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte("too-short"))
	cfg := &Config{LicensePublicKey: encoded}
	_, err := cfg.ParseLicensePublicKey()
	assert.Error(t, err)
}
