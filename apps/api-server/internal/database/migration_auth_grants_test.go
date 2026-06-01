package database

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAuthFunctionMigrationsGrantOpenOMSRole(t *testing.T) {
	findUserSignature := "public.find_user_for_auth(text, uuid)"

	for _, file := range []string{
		"000015_add_user_language.up.sql",
		"000015_add_user_language.down.sql",
	} {
		t.Run(file, func(t *testing.T) {
			sql := readMigrationSQL(t, file)

			require.Contains(t, normalizedSQL(sql), "rolname = 'openoms'")
			require.Contains(t, normalizedSQL(sql), "grant execute on function "+findUserSignature+" to openoms")
		})
	}

	t.Run("repair migration grants auth functions to openoms", func(t *testing.T) {
		sql := normalizedSQL(readMigrationSQL(t, "000025_auth_function_openoms_grants.up.sql"))

		require.Contains(t, sql, "rolname = 'openoms'")

		for _, signature := range []string{
			"public.find_tenant_by_slug(text)",
			findUserSignature,
			"public.find_invitation_by_token(text)",
			"public.find_return_by_token(text)",
			"public.find_order_tenant_id(uuid)",
			"public.use_invitation(text)",
		} {
			require.Contains(t, sql, "grant execute on function "+signature+" to openoms")
		}
	})

	t.Run("repair migration revokes explicit openoms grants on rollback", func(t *testing.T) {
		sql := normalizedSQL(readMigrationSQL(t, "000025_auth_function_openoms_grants.down.sql"))

		require.Contains(t, sql, "rolname = 'openoms'")

		for _, signature := range []string{
			"public.find_tenant_by_slug(text)",
			findUserSignature,
			"public.find_invitation_by_token(text)",
			"public.find_return_by_token(text)",
			"public.find_order_tenant_id(uuid)",
			"public.use_invitation(text)",
		} {
			require.Contains(t, sql, "revoke execute on function "+signature+" from openoms")
		}
	})
}

func TestRollbackMigrationsGuardOptionalRoleRevokes(t *testing.T) {
	for _, file := range []string{
		"000009_used_license_tokens.down.sql",
		"000010_tenant_plan_guard.down.sql",
	} {
		t.Run(file, func(t *testing.T) {
			sql := readMigrationSQL(t, file)
			normalized := normalizedSQL(sql)

			require.Contains(t, normalized, "select 1 from pg_roles where rolname =")
			require.Empty(t, topLevelOptionalRoleRevokes(sql))
		})
	}
}

func TestFindTenantBySlugMigrationRedactsSettings(t *testing.T) {
	up := normalizedSQL(readMigrationSQL(t, "000027_redact_find_tenant_by_slug_settings.up.sql"))
	down := normalizedSQL(readMigrationSQL(t, "000027_redact_find_tenant_by_slug_settings.down.sql"))

	require.Contains(t, up, "create or replace function public.find_tenant_by_slug(p_slug text)")
	require.NotContains(t, up, "drop function")
	require.Contains(t, up, "security definer")
	require.Contains(t, up, "set search_path to 'public'")
	require.Contains(t, up, "'{}'::jsonb as settings")
	require.NotContains(t, up, "t.settings, t.created_at")
	require.Contains(t, up, "revoke execute on function public.find_tenant_by_slug(text) from public")
	require.Contains(t, up, "grant execute on function public.find_tenant_by_slug(text) to openoms_app")
	require.Contains(t, up, "grant execute on function public.find_tenant_by_slug(text) to openoms")

	require.Contains(t, down, "t.settings, t.created_at")
	require.NotContains(t, down, "drop function")
	require.Contains(t, down, "revoke execute on function public.find_tenant_by_slug(text) from public")
}

func TestBillingSyncTenantPlanMigrationTargetsSubscriptionStatus(t *testing.T) {
	up := normalizedSQL(readMigrationSQL(t, "000030_billing_sync_tenant_plan_targeted_key.up.sql"))
	down := normalizedSQL(readMigrationSQL(t, "000030_billing_sync_tenant_plan_targeted_key.down.sql"))

	require.Contains(t, up, "create or replace function public.billing_sync_tenant_plan(")
	require.NotContains(t, up, "drop function")
	require.Contains(t, up, "security definer")
	// The function must touch ONLY the subscription_status key, never a blanket merge of
	// the caller's blob into tenants.settings.
	require.Contains(t, up, "jsonb_set(")
	require.Contains(t, up, "'{subscription_status}'")
	require.NotContains(t, up, "|| p_settings")
	require.Contains(t, up, "revoke execute on function public.billing_sync_tenant_plan(uuid, jsonb) from public")
	require.Contains(t, up, "grant execute on function public.billing_sync_tenant_plan(uuid, jsonb) to openoms")

	// Rollback restores the shallow-merge body.
	require.Contains(t, down, "|| p_settings")
	require.NotContains(t, down, "drop function")
	require.NotContains(t, down, "jsonb_set(")
}

func readMigrationSQL(t *testing.T, file string) string {
	t.Helper()

	switch file {
	case "000015_add_user_language.up.sql",
		"000015_add_user_language.down.sql",
		"000009_used_license_tokens.down.sql",
		"000010_tenant_plan_guard.down.sql",
		"000025_auth_function_openoms_grants.up.sql",
		"000025_auth_function_openoms_grants.down.sql",
		"000027_redact_find_tenant_by_slug_settings.up.sql",
		"000027_redact_find_tenant_by_slug_settings.down.sql",
		"000030_billing_sync_tenant_plan_targeted_key.up.sql",
		"000030_billing_sync_tenant_plan_targeted_key.down.sql":
	default:
		t.Fatalf("unexpected migration file %q", file)
	}

	// #nosec G304 -- test input is constrained by the allowlist above.
	content, err := os.ReadFile(filepath.Join("..", "..", "migrations", file))
	require.NoError(t, err)

	return string(content)
}

func normalizedSQL(sql string) string {
	return strings.Join(strings.Fields(strings.ToLower(sql)), " ")
}

func topLevelOptionalRoleRevokes(sql string) []string {
	var revokes []string
	for line := range strings.SplitSeq(sql, "\n") {
		trimmed := strings.TrimSpace(strings.ToLower(line))
		if !strings.HasPrefix(trimmed, "revoke execute on function") {
			continue
		}
		if strings.Contains(trimmed, " from openoms") {
			revokes = append(revokes, strings.TrimSpace(line))
		}
	}

	return revokes
}
