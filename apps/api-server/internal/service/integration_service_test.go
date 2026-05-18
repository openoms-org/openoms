package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

func TestIntegrationService_Create_ValidationError_MissingProvider(t *testing.T) {
	svc := NewIntegrationService(nil, nil, nil, make([]byte, 32))

	_, err := svc.Create(context.Background(), uuid.New(), model.CreateIntegrationRequest{
		Credentials: json.RawMessage(`{"key":"val"}`),
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "provider")
}

func TestIntegrationService_Create_ValidationError_MissingCredentials(t *testing.T) {
	svc := NewIntegrationService(nil, nil, nil, make([]byte, 32))

	_, err := svc.Create(context.Background(), uuid.New(), model.CreateIntegrationRequest{
		Provider: "allegro",
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
	assert.Contains(t, err.Error(), "credentials")
}

func TestIntegrationService_Update_ValidationError_NoFields(t *testing.T) {
	svc := NewIntegrationService(nil, nil, nil, make([]byte, 32))

	_, err := svc.Update(context.Background(), uuid.New(), uuid.New(), model.UpdateIntegrationRequest{}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
}

func TestIntegrationService_Update_ValidationError_InvalidStatus(t *testing.T) {
	svc := NewIntegrationService(nil, nil, nil, make([]byte, 32))
	badStatus := "bogus"

	_, err := svc.Update(context.Background(), uuid.New(), uuid.New(), model.UpdateIntegrationRequest{
		Status: &badStatus,
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	var ve *ValidationError
	assert.True(t, errors.As(err, &ve))
}

func TestIntegrationService_Create_BadEncryptionKey(t *testing.T) {
	// Too short key should cause an encryption error before hitting DB
	svc := NewIntegrationService(nil, nil, nil, []byte("short"))

	_, err := svc.Create(context.Background(), uuid.New(), model.CreateIntegrationRequest{
		Provider:    "allegro",
		Credentials: json.RawMessage(`{"token":"abc123"}`),
	}, uuid.New(), "127.0.0.1")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "encrypt")
}

func TestMergeCredentialUpdate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		existing []byte
		update   []byte
		want     map[string]any
	}{
		{
			name:     "partial update preserves existing credentials",
			existing: []byte(`{"client_id":"old-id","client_secret":"old-secret","refresh_token":"old-refresh"}`),
			update:   []byte(`{"webhook_secret":"new-webhook"}`),
			want: map[string]any{
				"client_id":      "old-id",
				"client_secret":  "old-secret",
				"refresh_token":  "old-refresh",
				"webhook_secret": "new-webhook",
			},
		},
		{
			name:     "blank webhook secret keeps existing secret",
			existing: []byte(`{"client_id":"old-id","webhook_secret":"old-webhook"}`),
			update:   []byte(`{"webhook_secret":"   "}`),
			want: map[string]any{
				"client_id":      "old-id",
				"webhook_secret": "old-webhook",
			},
		},
		{
			name:     "non empty webhook secret rotates only that field",
			existing: []byte(`{"client_id":"old-id","webhook_secret":"old-webhook"}`),
			update:   []byte(`{"webhook_secret":"new-webhook"}`),
			want: map[string]any{
				"client_id":      "old-id",
				"webhook_secret": "new-webhook",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			merged, err := mergeCredentialUpdate(tt.existing, tt.update)

			require.NoError(t, err)
			var got map[string]any
			require.NoError(t, json.Unmarshal(merged, &got))
			assert.Equal(t, tt.want, got)
		})
	}
}
