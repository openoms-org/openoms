package crypto //nolint:revive // package name conflicts with stdlib but renaming would break imports

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTenantSettingsSecrets_EncryptDecryptRoundTrip(t *testing.T) {
	key := testSettingsSecretKey()
	raw := json.RawMessage(`{
		"email":{"smtp_host":"smtp.example.com","smtp_pass":"smtp-secret"},
		"sms":{"api_token":"sms-secret","from":"OpenOMS"},
		"ksef":{"token":"ksef-secret","nip":"1234567890"},
		"invoicing":{"provider":"fakturownia","credentials":{"api_key":"invoice-secret","nested":{"refresh":"refresh-secret"},"tokens":["array-secret"],"empty":""}},
		"webhooks":{"endpoints":[{"url":"https://example.com/hook","secret":"webhook-secret","active":true}]}
	}`)

	encrypted, changed, err := EncryptTenantSettingsSecrets(raw, key)
	require.NoError(t, err)
	require.True(t, changed)

	encryptedText := string(encrypted)
	for _, plaintext := range []string{"smtp-secret", "sms-secret", "ksef-secret", "invoice-secret", "refresh-secret", "array-secret", "webhook-secret"} {
		assert.NotContains(t, encryptedText, plaintext)
	}
	assert.Contains(t, encryptedText, settingsSecretEnvelopeVersion)
	assert.Contains(t, encryptedText, "smtp.example.com")

	decrypted, changed, err := DecryptTenantSettingsSecrets(encrypted, key)
	require.NoError(t, err)
	require.True(t, changed)

	var settings map[string]any
	require.NoError(t, json.Unmarshal(decrypted, &settings))
	assert.Equal(t, "smtp-secret", settings["email"].(map[string]any)["smtp_pass"])
	assert.Equal(t, "sms-secret", settings["sms"].(map[string]any)["api_token"])
	assert.Equal(t, "ksef-secret", settings["ksef"].(map[string]any)["token"])

	credentials := settings["invoicing"].(map[string]any)["credentials"].(map[string]any)
	assert.Equal(t, "invoice-secret", credentials["api_key"])
	assert.Equal(t, "refresh-secret", credentials["nested"].(map[string]any)["refresh"])
	assert.Equal(t, "array-secret", credentials["tokens"].([]any)[0])
	assert.Equal(t, "", credentials["empty"])

	endpoints := settings["webhooks"].(map[string]any)["endpoints"].([]any)
	assert.Equal(t, "webhook-secret", endpoints[0].(map[string]any)["secret"])
}

func TestTenantSettingsSecrets_DecryptLeavesLegacyPlaintextReadable(t *testing.T) {
	raw := json.RawMessage(`{"email":{"smtp_pass":"smtp-secret"},"sms":{"api_token":"sms-secret"}}`)

	decrypted, changed, err := DecryptTenantSettingsSecrets(raw, testSettingsSecretKey())
	require.NoError(t, err)
	assert.False(t, changed)
	assert.JSONEq(t, string(raw), string(decrypted))
}

func TestTenantSettingsSecrets_EncryptSkipsExistingEnvelopes(t *testing.T) {
	key := testSettingsSecretKey()
	raw := json.RawMessage(`{"email":{"smtp_pass":"smtp-secret"}}`)
	encrypted, changed, err := EncryptTenantSettingsSecrets(raw, key)
	require.NoError(t, err)
	require.True(t, changed)

	encryptedAgain, changed, err := EncryptTenantSettingsSecrets(encrypted, key)
	require.NoError(t, err)
	assert.False(t, changed)
	assert.JSONEq(t, string(encrypted), string(encryptedAgain))
}

func TestTenantSettingsSecrets_InvalidEnvelopeFailsClosed(t *testing.T) {
	raw := json.RawMessage(`{"email":{"smtp_pass":{"__openoms_secret":"tenant-settings/v1"}}}`)

	_, _, err := DecryptTenantSettingsSecrets(raw, testSettingsSecretKey())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid tenant settings secret envelope")
}

func testSettingsSecretKey() []byte {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 1)
	}
	return key
}
