//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

// seedUser inserts a minimal user for a tenant and returns its id. Cleanup
// cascades via the tenant (and from there to platform_admins by user FK).
func seedUser(t *testing.T, ctx context.Context, tenantID uuid.UUID) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := superPool.Exec(ctx,
		`INSERT INTO users (id, tenant_id, email, name, password_hash)
		 VALUES ($1, $2, $3, $4, $5)`,
		id, tenantID, "platform-"+id.String()[:8]+"@example.test", "Platform Admin", "x")
	require.NoError(t, err, "seed user")
	return id
}

// TestPlatformAdminRepository_CreateGetAudit proves, against a real database,
// that the platform_admins / platform_audit_log tables are usable by the
// least-privilege app role (grants applied), that the FK to users works, and
// that the tables are readable WITHOUT any tenant context (no RLS).
func TestPlatformAdminRepository_CreateGetAudit(t *testing.T) {
	ctx := context.Background()
	tenantID := seedTenant(t, ctx)
	userID := seedUser(t, ctx, tenantID)

	// Use the RLS-scoped app pool to prove the grants + that platform_admins is
	// reachable outside any WithTenant context (no RLS on the table).
	adminRepo := repository.NewPlatformAdminRepository(appPool)
	auditRepo := repository.NewPlatformAuditRepository(appPool)

	perms := []string{model.PermPlatformProvidersRead, model.PermPlatformProvidersSecrets}
	created, err := adminRepo.Create(ctx, userID, perms)
	require.NoError(t, err, "app role must be able to INSERT platform_admins")
	assert.Equal(t, userID, created.UserID)
	assert.ElementsMatch(t, perms, created.Permissions)

	got, err := adminRepo.GetByUserID(ctx, userID)
	require.NoError(t, err, "app role must be able to SELECT platform_admins")
	assert.Equal(t, created.ID, got.ID)
	assert.True(t, got.HasPermission(model.PermPlatformProvidersSecrets))
	assert.False(t, got.HasPermission(model.PermPlatformProvidersPublish))

	// Unknown user -> not found (fail-closed signal for the middleware).
	_, err = adminRepo.GetByUserID(ctx, uuid.New())
	require.ErrorIs(t, err, pgx.ErrNoRows)

	// Audit write works on the app pool too.
	err = auditRepo.Log(ctx, model.PlatformAuditEntry{
		ActorUserID: userID,
		Action:      "platform.session.accessed",
		Metadata:    map[string]string{"test": "ope-404"},
		IPAddress:   "203.0.113.7",
	})
	require.NoError(t, err, "app role must be able to INSERT platform_audit_log")

	var auditCount int
	require.NoError(t, superPool.QueryRow(ctx,
		"SELECT count(*) FROM platform_audit_log WHERE actor_user_id = $1", userID).Scan(&auditCount))
	assert.Equal(t, 1, auditCount)

	t.Cleanup(func() {
		_, _ = superPool.Exec(context.Background(), "DELETE FROM platform_audit_log WHERE actor_user_id = $1", userID)
	})
}
