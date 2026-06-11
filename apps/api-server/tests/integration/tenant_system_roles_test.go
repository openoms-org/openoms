//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// expectedSystemRoles maps each seeded system-role name to its canonical permission
// set, so the tests assert the seed faithfully mirrors the model definitions.
var expectedSystemRoles = map[string][]string{
	"Owner":         model.SystemRoleOwnerPermissions,
	"Administrator": model.SystemRoleAdminPermissions,
	"Employee":      model.SystemRoleMemberPermissions,
}

// assertSeededSystemRoles asserts the tenant has exactly the three system roles
// (Owner/Administrator/Employee), each is_system=true with the canonical permission
// set. Runs under the RLS-applying app pool to mirror production.
func assertSeededSystemRoles(ctx context.Context, t *testing.T, tenantID uuid.UUID) {
	t.Helper()
	roleRepo := repository.NewRoleRepository()
	err := database.WithTenant(ctx, appPool, tenantID, func(tx pgx.Tx) error {
		var total int
		if err := tx.QueryRow(ctx, "SELECT COUNT(*) FROM roles").Scan(&total); err != nil {
			return err
		}
		assert.Equal(t, len(expectedSystemRoles), total, "tenant must have exactly the seeded system roles")

		for name, wantPerms := range expectedSystemRoles {
			role, err := roleRepo.FindByName(ctx, tx, name)
			if err != nil {
				return err
			}
			require.NotNilf(t, role, "system role %q must be seeded", name)
			assert.Truef(t, role.IsSystem, "system role %q must have is_system=true", name)
			assert.Equalf(t, tenantID, role.TenantID, "system role %q must belong to the tenant", name)
			assert.ElementsMatchf(t, wantPerms, role.Permissions, "system role %q permissions must mirror the model", name)
		}
		return nil
	})
	require.NoError(t, err)
}

// TestEnsureSystemRoles_Idempotent verifies the seeding helper is safe to call more
// than once for the same tenant: the second call produces no duplicate rows and no
// unique-constraint violation, and the canonical permission sets are preserved.
func TestEnsureSystemRoles_Idempotent(t *testing.T) {
	ctx := context.Background()
	tenantID := seedTenant(t, ctx)

	roleRepo := repository.NewRoleRepository()
	auditRepo := repository.NewAuditRepository()
	roleSvc := service.NewRoleService(roleRepo, auditRepo, appPool)

	require.NoError(t, roleSvc.EnsureSystemRoles(ctx, tenantID), "first seed must succeed")
	assertSeededSystemRoles(ctx, t, tenantID)

	// Second call must be a no-op against the roles_tenant_id_name_key unique index.
	require.NoError(t, roleSvc.EnsureSystemRoles(ctx, tenantID), "re-seeding must be idempotent")
	assertSeededSystemRoles(ctx, t, tenantID)
}

// TestEnsureSystemRoles_ResyncsSystemPermissions verifies that when a system role's
// stored permissions drift from the canonical set, a re-seed re-syncs them (without
// creating duplicate rows).
func TestEnsureSystemRoles_ResyncsSystemPermissions(t *testing.T) {
	ctx := context.Background()
	tenantID := seedTenant(t, ctx)

	roleRepo := repository.NewRoleRepository()
	auditRepo := repository.NewAuditRepository()
	roleSvc := service.NewRoleService(roleRepo, auditRepo, appPool)

	require.NoError(t, roleSvc.EnsureSystemRoles(ctx, tenantID))

	// Drift the Owner role's permissions, then re-seed and assert they are restored.
	err := database.WithTenant(ctx, appPool, tenantID, func(tx pgx.Tx) error {
		owner, err := roleRepo.FindByName(ctx, tx, "Owner")
		if err != nil {
			return err
		}
		require.NotNil(t, owner)
		return roleRepo.Update(ctx, tx, owner.ID, model.UpdateRoleRequest{Permissions: []string{"orders:read"}})
	})
	require.NoError(t, err)

	require.NoError(t, roleSvc.EnsureSystemRoles(ctx, tenantID))
	assertSeededSystemRoles(ctx, t, tenantID)
}

// TestAuthService_Register_SeedsSystemRoles is the end-to-end wiring assertion: a
// registration through AuthService.Register (with the RoleService wired via
// SetRoleService, exactly as cmd/server/main.go does) seeds the tenant's three
// system roles. It also asserts the owner user keeps role='owner' with role_id NULL
// — the resolved owner decision: seeding is purely additive and permission-neutral
// (string-roles stays the canonical model).
func TestAuthService_Register_SeedsSystemRoles(t *testing.T) {
	ctx := context.Background()

	userRepo := repository.NewUserRepository(appPool)
	tenantRepo := repository.NewTenantRepository(appPool)
	auditRepo := repository.NewAuditRepository()
	roleRepo := repository.NewRoleRepository()
	passwordSvc := service.NewPasswordService()
	tokenSvc, err := service.NewTokenService("integration-test-jwt-secret-at-least-32chars")
	require.NoError(t, err)

	authSvc := service.NewAuthService(userRepo, tenantRepo, auditRepo, tokenSvc, passwordSvc, appPool)
	authSvc.SetRoleRepo(roleRepo)
	authSvc.SetRoleService(service.NewRoleService(roleRepo, auditRepo, appPool))

	slug := "it-reg-" + uuid.New().String()[:8]
	resp, refresh, err := authSvc.Register(ctx, model.RegisterRequest{
		Email:      slug + "@example.com",
		Password:   "StrongP@ss123",
		Name:       "Owner User",
		TenantName: "Integration " + slug,
		TenantSlug: slug,
	}, "127.0.0.1")
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotEmpty(t, refresh)

	tenantID := resp.Tenant.ID
	require.NotEqual(t, uuid.Nil, tenantID)
	t.Cleanup(func() {
		bg := context.Background()
		_, _ = superPool.Exec(bg, "DELETE FROM audit_log WHERE tenant_id = $1", tenantID)
		_, _ = superPool.Exec(bg, "DELETE FROM users WHERE tenant_id = $1", tenantID)
		_, _ = superPool.Exec(bg, "DELETE FROM roles WHERE tenant_id = $1", tenantID)
		_, _ = superPool.Exec(bg, "DELETE FROM tenants WHERE id = $1", tenantID)
	})

	// The seeded system roles must be present.
	assertSeededSystemRoles(ctx, t, tenantID)

	// Owner decision (resolved): the owner user is NOT linked to the seeded Owner role
	// row — role stays the string 'owner' and role_id stays NULL.
	assert.Equal(t, "owner", resp.User.Role)
	assert.Nil(t, resp.User.RoleID, "owner user role_id must remain NULL (string-role canonical)")

	err = database.WithTenant(ctx, appPool, tenantID, func(tx pgx.Tx) error {
		var role string
		var roleID *uuid.UUID
		if err := tx.QueryRow(ctx, "SELECT role, role_id FROM users WHERE tenant_id = $1", tenantID).Scan(&role, &roleID); err != nil {
			return err
		}
		assert.Equal(t, "owner", role)
		assert.Nil(t, roleID, "owner user row role_id must remain NULL")
		return nil
	})
	require.NoError(t, err)
}
