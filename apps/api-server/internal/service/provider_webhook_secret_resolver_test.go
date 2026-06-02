package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// OPE-490: Resolve must fail closed (never panic, never return an empty secret as valid)
// when it is not wired to a database — a webhook with no resolvable secret must be rejected.
func TestProviderWebhookSecretResolverResolveFailsClosedWithoutPool(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name     string
		resolver *ProviderWebhookSecretResolver
	}{
		{name: "nil receiver", resolver: nil},
		{name: "nil pool", resolver: &ProviderWebhookSecretResolver{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			secret, scope, err := tc.resolver.Resolve(context.Background(), "inpost", uuid.New())

			require.ErrorIs(t, err, ErrWebhookSecretNotConfigured)
			assert.Empty(t, secret)
			assert.Nil(t, scope)
		})
	}
}
