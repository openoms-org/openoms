package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractProviderWebhookSecret(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		credentials []byte
		wantSecret  string
		wantOK      bool
	}{
		{
			name:        "webhook secret present",
			credentials: []byte(`{"client_id":"abc","webhook_secret":"tenant-secret"}`),
			wantSecret:  "tenant-secret",
			wantOK:      true,
		},
		{
			name:        "trims whitespace",
			credentials: []byte(`{"webhook_secret":"  tenant-secret  "}`),
			wantSecret:  "tenant-secret",
			wantOK:      true,
		},
		{
			name:        "missing secret",
			credentials: []byte(`{"client_id":"abc"}`),
			wantOK:      false,
		},
		{
			name:        "empty secret",
			credentials: []byte(`{"webhook_secret":"   "}`),
			wantOK:      false,
		},
		{
			name:        "non string secret",
			credentials: []byte(`{"webhook_secret":true}`),
			wantOK:      false,
		},
		{
			name:        "invalid json",
			credentials: []byte(`not-json`),
			wantOK:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			secret, ok := ExtractProviderWebhookSecret(tt.credentials)

			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.wantSecret, secret)
		})
	}
}
