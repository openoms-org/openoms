package crypto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncryptDecrypt_Roundtrip(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	plaintext := []byte(`{"token":"secret123","refresh":"refresh456"}`)

	encrypted, err := Encrypt(plaintext, key)
	require.NoError(t, err)
	assert.NotEmpty(t, encrypted)
	assert.NotEqual(t, string(plaintext), encrypted)

	decrypted, err := Decrypt(encrypted, key)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestEncrypt_InvalidKeyLength(t *testing.T) {
	tests := []struct {
		name    string
		keyLen  int
	}{
		{"too short", 16},
		{"too long", 64},
		{"empty", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := make([]byte, tt.keyLen)
			_, err := Encrypt([]byte("data"), key)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "32 bytes")
		})
	}
}

func TestDecrypt_InvalidKeyLength(t *testing.T) {
	_, err := Decrypt("dGVzdA==", make([]byte, 16))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "32 bytes")
}

func TestDecrypt_InvalidBase64(t *testing.T) {
	key := make([]byte, 32)
	_, err := Decrypt("not-base64!!!", key)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "base64")
}

func TestDecrypt_CiphertextTooShort(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	// Very short valid base64 that is shorter than nonce size
	_, err := Decrypt("dGVzdA==", key)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "too short")
}

func TestEncrypt_DifferentCiphertexts(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	plaintext := []byte("same data")
	enc1, err := Encrypt(plaintext, key)
	require.NoError(t, err)

	enc2, err := Encrypt(plaintext, key)
	require.NoError(t, err)

	// Due to random nonce, encrypting the same plaintext twice should yield different ciphertexts
	assert.NotEqual(t, enc1, enc2)
}

func TestDecrypt_WrongKey(t *testing.T) {
	key1 := make([]byte, 32)
	key2 := make([]byte, 32)
	for i := range key1 {
		key1[i] = byte(i)
		key2[i] = byte(i + 1)
	}

	encrypted, err := Encrypt([]byte("secret"), key1)
	require.NoError(t, err)

	_, err = Decrypt(encrypted, key2)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decrypting")
}

func TestEncrypt_EmptyPlaintext(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	encrypted, err := Encrypt([]byte{}, key)
	require.NoError(t, err)
	assert.NotEmpty(t, encrypted)

	decrypted, err := Decrypt(encrypted, key)
	require.NoError(t, err)
	assert.Empty(t, decrypted)
}
