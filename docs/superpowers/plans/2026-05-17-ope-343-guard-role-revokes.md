# OPE-343 Guard Role Revokes Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make rollback migrations safe when optional database roles such as `openoms` or `openoms_auth` do not exist.

**Architecture:** Keep the SQL behavior identical when roles exist, but wrap role-specific `REVOKE EXECUTE` statements in PostgreSQL `DO $$` blocks that check `pg_roles` first. Add a static Go regression test in the database package so future down migrations do not reintroduce unguarded revokes for optional OpenOMS roles.

**Tech Stack:** Go tests with `testing` and `testify/require`, PostgreSQL SQL migrations, golang-migrate conventions.

---

### Task 1: Regression Test

**Files:**
- Modify: `apps/api-server/internal/database/migration_auth_grants_test.go`

- [ ] **Step 1: Add a failing static test**

Add a test that scans rollback migrations for top-level `REVOKE EXECUTE ... FROM openoms` / `openoms_auth` statements and requires the affected migration to contain a role-existence guard:

```go
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
```

- [ ] **Step 2: Run the test and verify RED**

Run:

```bash
cd /Users/rafs/praca/openoms-dev/public/apps/api-server
go test ./internal/database -run TestRollbackMigrationsGuardOptionalRoleRevokes -count=1
```

Expected before the fix: FAIL because `000009_used_license_tokens.down.sql` and `000010_tenant_plan_guard.down.sql` contain unguarded role revokes.

### Task 2: Guard Down Migrations

**Files:**
- Modify: `apps/api-server/migrations/000009_used_license_tokens.down.sql`
- Modify: `apps/api-server/migrations/000010_tenant_plan_guard.down.sql`

- [ ] **Step 1: Guard `000009` revokes**

Replace top-level `REVOKE` statements with:

```sql
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'openoms_auth') THEN
    EXECUTE 'REVOKE EXECUTE ON FUNCTION public.update_license_token_tenant(uuid, uuid) FROM openoms_auth';
    EXECUTE 'REVOKE EXECUTE ON FUNCTION public.mark_license_token_used(uuid, uuid, text, text) FROM openoms_auth';
    EXECUTE 'REVOKE EXECUTE ON FUNCTION public.check_license_token_used(uuid) FROM openoms_auth';
  END IF;
END;
$$;
```

- [ ] **Step 2: Guard `000010` revokes**

Replace top-level `REVOKE` statements with separate checks for the optional roles used by the up migration:

```sql
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'openoms_auth') THEN
    EXECUTE 'REVOKE EXECUTE ON FUNCTION public.get_tenant_plan(uuid) FROM openoms_auth';
  END IF;
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'openoms') THEN
    EXECUTE 'REVOKE EXECUTE ON FUNCTION public.get_tenant_plan(uuid) FROM openoms';
  END IF;
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'openoms_app') THEN
    EXECUTE 'REVOKE EXECUTE ON FUNCTION public.get_tenant_plan(uuid) FROM openoms_app';
  END IF;
END;
$$;
```

### Task 3: Validation

**Files:**
- Test: `apps/api-server/internal/database/migration_auth_grants_test.go`

- [ ] **Step 1: Run the targeted migration test**

```bash
cd /Users/rafs/praca/openoms-dev/public/apps/api-server
go test ./internal/database -run TestRollbackMigrationsGuardOptionalRoleRevokes -count=1
```

Expected: PASS.

- [ ] **Step 2: Run broader database tests**

```bash
cd /Users/rafs/praca/openoms-dev/public/apps/api-server
go test ./internal/database -count=1
```

Expected: PASS.

- [ ] **Step 3: Run migration timeout checker**

```bash
cd /Users/rafs/praca/openoms-dev/public
./scripts/check-migration-timeouts.sh apps/api-server/migrations
```

Expected: PASS/no errors.

- [ ] **Step 4: Run repository validation before push**

```bash
cd /Users/rafs/praca/openoms-dev/public
git diff --check
./scripts/local-ci.sh
```

Expected: PASS. Commit only after the working tree contains the intended migration/test/plan changes.

### Risk And Rollback

Risk is low: the up migration grant semantics are unchanged, and the down migrations still revoke when the roles exist. The only behavior change is that rollback no longer fails when optional roles are absent.

Rollback is the PR revert. If the guard itself caused an unexpected migration parser issue, reverting restores the previous SQL, but that also restores the original missing-role rollback failure.
