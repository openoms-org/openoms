package crypto

import (
	"encoding/json"
	"fmt"
)

const (
	settingsSecretEnvelopeKey     = "__openoms_secret" // #nosec G101 -- JSON marker, not credential material.
	settingsSecretEnvelopeVersion = "tenant-settings/v1"
	settingsSecretCiphertextKey   = "ciphertext"
)

// EncryptTenantSettingsSecrets encrypts secret-bearing fields inside tenants.settings.
// It is intentionally narrow: non-secret settings remain queryable/debuggable as JSON.
func EncryptTenantSettingsSecrets(raw json.RawMessage, key []byte) (json.RawMessage, bool, error) {
	if len(raw) == 0 || len(key) == 0 {
		return raw, false, nil
	}

	var root any
	if err := json.Unmarshal(raw, &root); err != nil {
		return nil, false, fmt.Errorf("decode tenant settings: %w", err)
	}

	encrypted, changed, err := encryptSettingsNode(root, key)
	if err != nil {
		return nil, false, err
	}
	if !changed {
		return raw, false, nil
	}

	result, err := json.Marshal(encrypted)
	if err != nil {
		return nil, false, fmt.Errorf("encode encrypted tenant settings: %w", err)
	}
	return result, true, nil
}

// DecryptTenantSettingsSecrets returns application-readable settings with secret
// fields decrypted. Plaintext legacy values are left as-is so old rows remain readable
// until the startup backfill encrypts them.
func DecryptTenantSettingsSecrets(raw json.RawMessage, key []byte) (json.RawMessage, bool, error) {
	if len(raw) == 0 || len(key) == 0 {
		return raw, false, nil
	}

	var root any
	if err := json.Unmarshal(raw, &root); err != nil {
		return nil, false, fmt.Errorf("decode tenant settings: %w", err)
	}

	decrypted, changed, err := decryptSettingsNode(root, key)
	if err != nil {
		return nil, false, err
	}
	if !changed {
		return raw, false, nil
	}

	result, err := json.Marshal(decrypted)
	if err != nil {
		return nil, false, fmt.Errorf("encode decrypted tenant settings: %w", err)
	}
	return result, true, nil
}

func encryptSettingsNode(root any, key []byte) (any, bool, error) {
	settings, ok := root.(map[string]any)
	if !ok {
		return root, false, nil
	}

	changed := false
	var err error

	if section, ok := settings["email"].(map[string]any); ok {
		changed, err = encryptStringField(section, "smtp_pass", key, changed)
		if err != nil {
			return nil, false, err
		}
	}

	if section, ok := settings["sms"].(map[string]any); ok {
		changed, err = encryptStringField(section, "api_token", key, changed)
		if err != nil {
			return nil, false, err
		}
	}

	if section, ok := settings["ksef"].(map[string]any); ok {
		changed, err = encryptStringField(section, "token", key, changed)
		if err != nil {
			return nil, false, err
		}
	}

	if section, ok := settings["invoicing"].(map[string]any); ok {
		if credentials, hasCredentials := section["credentials"]; hasCredentials {
			section["credentials"], changed, err = encryptSecretTree(credentials, key, changed)
			if err != nil {
				return nil, false, err
			}
		}
	}

	if section, ok := settings["webhooks"].(map[string]any); ok {
		changed, err = encryptWebhookSecrets(section, key, changed)
		if err != nil {
			return nil, false, err
		}
	}

	return settings, changed, nil
}

func decryptSettingsNode(root any, key []byte) (any, bool, error) {
	settings, ok := root.(map[string]any)
	if !ok {
		return root, false, nil
	}

	changed := false
	var err error

	if section, ok := settings["email"].(map[string]any); ok {
		changed, err = decryptStringField(section, "smtp_pass", key, changed)
		if err != nil {
			return nil, false, err
		}
	}

	if section, ok := settings["sms"].(map[string]any); ok {
		changed, err = decryptStringField(section, "api_token", key, changed)
		if err != nil {
			return nil, false, err
		}
	}

	if section, ok := settings["ksef"].(map[string]any); ok {
		changed, err = decryptStringField(section, "token", key, changed)
		if err != nil {
			return nil, false, err
		}
	}

	if section, ok := settings["invoicing"].(map[string]any); ok {
		if credentials, hasCredentials := section["credentials"]; hasCredentials {
			section["credentials"], changed, err = decryptSecretTree(credentials, key, changed)
			if err != nil {
				return nil, false, err
			}
		}
	}

	if section, ok := settings["webhooks"].(map[string]any); ok {
		changed, err = decryptWebhookSecrets(section, key, changed)
		if err != nil {
			return nil, false, err
		}
	}

	return settings, changed, nil
}

func encryptStringField(section map[string]any, field string, key []byte, changed bool) (bool, error) {
	value, ok := section[field]
	if !ok {
		return changed, nil
	}
	encrypted, encryptedChanged, err := encryptSecretTree(value, key, changed)
	if err != nil {
		return false, fmt.Errorf("encrypt %s: %w", field, err)
	}
	section[field] = encrypted
	return encryptedChanged, nil
}

func decryptStringField(section map[string]any, field string, key []byte, changed bool) (bool, error) {
	value, ok := section[field]
	if !ok {
		return changed, nil
	}
	decrypted, decryptedChanged, err := decryptSecretTree(value, key, changed)
	if err != nil {
		return false, fmt.Errorf("decrypt %s: %w", field, err)
	}
	section[field] = decrypted
	return decryptedChanged, nil
}

func encryptSecretTree(value any, key []byte, changed bool) (any, bool, error) {
	switch v := value.(type) {
	case string:
		if v == "" {
			return v, changed, nil
		}
		ciphertext, err := Encrypt([]byte(v), key)
		if err != nil {
			return nil, false, err
		}
		return map[string]any{
			settingsSecretEnvelopeKey:   settingsSecretEnvelopeVersion,
			settingsSecretCiphertextKey: ciphertext,
		}, true, nil
	case map[string]any:
		if _, ok := secretEnvelopeCiphertext(v); ok {
			return v, changed, nil
		}
		if _, looksEncrypted := v[settingsSecretEnvelopeKey]; looksEncrypted {
			return nil, false, fmt.Errorf("invalid tenant settings secret envelope")
		}
		for field, child := range v {
			var childChanged bool
			var err error
			v[field], childChanged, err = encryptSecretTree(child, key, changed)
			if err != nil {
				return nil, false, err
			}
			changed = childChanged
		}
		return v, changed, nil
	case []any:
		for i, child := range v {
			var childChanged bool
			var err error
			v[i], childChanged, err = encryptSecretTree(child, key, changed)
			if err != nil {
				return nil, false, err
			}
			changed = childChanged
		}
		return v, changed, nil
	default:
		return v, changed, nil
	}
}

func decryptSecretTree(value any, key []byte, changed bool) (any, bool, error) {
	switch v := value.(type) {
	case map[string]any:
		if ciphertext, ok := secretEnvelopeCiphertext(v); ok {
			plaintext, err := Decrypt(ciphertext, key)
			if err != nil {
				return nil, false, err
			}
			return string(plaintext), true, nil
		}
		if _, looksEncrypted := v[settingsSecretEnvelopeKey]; looksEncrypted {
			return nil, false, fmt.Errorf("invalid tenant settings secret envelope")
		}
		for field, child := range v {
			var childChanged bool
			var err error
			v[field], childChanged, err = decryptSecretTree(child, key, changed)
			if err != nil {
				return nil, false, err
			}
			changed = childChanged
		}
		return v, changed, nil
	case []any:
		for i, child := range v {
			var childChanged bool
			var err error
			v[i], childChanged, err = decryptSecretTree(child, key, changed)
			if err != nil {
				return nil, false, err
			}
			changed = childChanged
		}
		return v, changed, nil
	default:
		return v, changed, nil
	}
}

func encryptWebhookSecrets(section map[string]any, key []byte, changed bool) (bool, error) {
	endpoints, ok := section["endpoints"].([]any)
	if !ok {
		return changed, nil
	}

	for _, endpoint := range endpoints {
		endpointMap, ok := endpoint.(map[string]any)
		if !ok {
			continue
		}
		var err error
		changed, err = encryptStringField(endpointMap, "secret", key, changed)
		if err != nil {
			return false, err
		}
	}
	return changed, nil
}

func decryptWebhookSecrets(section map[string]any, key []byte, changed bool) (bool, error) {
	endpoints, ok := section["endpoints"].([]any)
	if !ok {
		return changed, nil
	}

	for _, endpoint := range endpoints {
		endpointMap, ok := endpoint.(map[string]any)
		if !ok {
			continue
		}
		var err error
		changed, err = decryptStringField(endpointMap, "secret", key, changed)
		if err != nil {
			return false, err
		}
	}
	return changed, nil
}

func secretEnvelopeCiphertext(value map[string]any) (string, bool) {
	version, ok := value[settingsSecretEnvelopeKey].(string)
	if !ok || version != settingsSecretEnvelopeVersion {
		return "", false
	}
	ciphertext, ok := value[settingsSecretCiphertextKey].(string)
	return ciphertext, ok && ciphertext != ""
}
