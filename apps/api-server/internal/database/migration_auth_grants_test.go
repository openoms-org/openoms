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

func readMigrationSQL(t *testing.T, file string) string {
	t.Helper()

	switch file {
	case "000015_add_user_language.up.sql",
		"000015_add_user_language.down.sql",
		"000025_auth_function_openoms_grants.up.sql",
		"000025_auth_function_openoms_grants.down.sql":
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
