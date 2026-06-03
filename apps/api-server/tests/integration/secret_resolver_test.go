//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/crypto"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// resolverTestKey is a fixed 32-byte AES-256 key for the resolver tests.
var resolverTestKey = bytes.Repeat([]byte{0x42}, 32)

// seedActiveIntegration inserts an active integration whose credentials JSONB holds
// the AES-256-GCM-encrypted ciphertext (a JSON-encoded base64 string), with the given
// webhook secret embedded. Returns the integration id. Cleaned up via tenant cascade,
// plus an explicit delete for safety.
func seedActiveIntegration(t *testing.T, ctx context.Context, tenantID uuid.UUID, provider, webhookSecret string) uuid.UUID {
	t.Helper()

	plaintext := []byte(fmt.Sprintf(`{"client_id":"cid-%s","webhook_secret":%q}`, provider, webhookSecret))
	encB64, err := crypto.Encrypt(plaintext, resolverTestKey)
	require.NoError(t, err, "encrypt integration credentials")

	// The resolver decodes credentials as a JSON-encoded string of the ciphertext.
	credentials, err := json.Marshal(encB64)
	require.NoError(t, err, "wrap ciphertext as JSON string")

	id := uuid.New()
	_, err = superPool.Exec(ctx,
		`INSERT INTO integrations (id, tenant_id, provider, status, credentials)
		 VALUES ($1, $2, $3, 'active', $4)`,
		id, tenantID, provider, json.RawMessage(credentials))
	require.NoError(t, err, "seed active integration")

	t.Cleanup(func() {
		_, _ = superPool.Exec(context.Background(), "DELETE FROM integrations WHERE id = $1", id)
	})
	return id
}

// TestProviderWebhookSecretResolver_PerIntegrationScopeAndSecret proves that the
// resolver maps each integration id to its OWN tenant scope and decrypts its OWN
// webhook secret — there is no cross-integration leakage of secrets or tenant scope,
// and an unknown integration fails closed.
//
// The resolver runs on the privileged worker pool (bypasses RLS by design, because
// webhook tenant-resolution happens BEFORE any tenant context exists). Isolation here
// is enforced by the integration-id -> tenant scope mapping, not by RLS.
func TestProviderWebhookSecretResolver_PerIntegrationScopeAndSecret(t *testing.T) {
	ctx := context.Background()
	tenantA := seedTenant(t, ctx)
	tenantB := seedTenant(t, ctx)

	intA := seedActiveIntegration(t, ctx, tenantA, "allegro", "secret-A")
	intB := seedActiveIntegration(t, ctx, tenantB, "allegro", "secret-B")

	resolver := service.NewProviderWebhookSecretResolver(superPool, resolverTestKey)

	// Integration A -> tenant A scope + secret A.
	secretA, scopeA, err := resolver.Resolve(ctx, "allegro", intA)
	require.NoError(t, err, "resolve integration A")
	assert.Equal(t, "secret-A", secretA, "must return integration A's own secret")
	require.NotNil(t, scopeA)
	assert.Equal(t, tenantA, scopeA.TenantID, "scope must map to tenant A")
	assert.Equal(t, intA, scopeA.IntegrationID, "scope must carry integration A's id")
	assert.Equal(t, "allegro", scopeA.Provider)

	// Integration B -> tenant B scope + secret B (no cross-contamination).
	secretB, scopeB, err := resolver.Resolve(ctx, "allegro", intB)
	require.NoError(t, err, "resolve integration B")
	assert.Equal(t, "secret-B", secretB, "must return integration B's own secret")
	require.NotNil(t, scopeB)
	assert.Equal(t, tenantB, scopeB.TenantID, "scope must map to tenant B")
	assert.NotEqual(t, secretA, secretB, "the two integrations must not share a secret")

	// Fail-closed: unknown integration id resolves to the not-found sentinel.
	_, _, err = resolver.Resolve(ctx, "allegro", uuid.New())
	require.Error(t, err, "unknown integration must fail closed")
	assert.ErrorIs(t, err, service.ErrProviderWebhookIntegrationNotFound)

	// Fail-closed: provider mismatch (right id, wrong provider) must not resolve.
	_, _, err = resolver.Resolve(ctx, "inpost", intA)
	require.Error(t, err, "provider mismatch must fail closed")
	assert.ErrorIs(t, err, service.ErrProviderWebhookIntegrationNotFound)
}
