# OPE-318 Auth Function OpenOMS Grants Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ensure self-hosted deployments using the `openoms` application role keep EXECUTE privileges on auth SECURITY DEFINER functions after migration 15 and on already-migrated databases.

**Architecture:** Keep the fix in public database migrations. Patch the future up/down path in `000015_add_user_language` and add an append-only repair migration for installations already past migration 15.

**Tech Stack:** PostgreSQL migrations, Go static migration tests, golang-migrate-compatible SQL.

---

## Files And Scope

- Modify: `apps/api-server/migrations/000015_add_user_language.up.sql`
  - Add a guarded `GRANT EXECUTE ... TO openoms` for `public.find_user_for_auth(text, uuid)`.
- Modify: `apps/api-server/migrations/000015_add_user_language.down.sql`
  - Add the same guarded grant after the rollback function is recreated.
- Create: `apps/api-server/migrations/000025_auth_function_openoms_grants.up.sql`
  - Repair current databases by granting `openoms` EXECUTE on auth SECURITY DEFINER functions if the role exists.
- Create: `apps/api-server/migrations/000025_auth_function_openoms_grants.down.sql`
  - Revoke only the explicit repair grants from `openoms` if the role exists.
- Create: `apps/api-server/internal/database/migration_auth_grants_test.go`
  - Static regression tests for the migration grant contract.
- No enterprise changes.
- No UI/API route changes.

## Root Cause

`000015_add_user_language` recreates `public.find_user_for_auth(text, uuid)`, revokes PUBLIC execute, then grants execute only to `openoms_app`. In self-hosted deployments the app role is `openoms`, so login can lose execute permission after migration 15 or after rolling back migration 15.

## Implementation Tasks

### Task 1: RED Test

- [ ] Add `apps/api-server/internal/database/migration_auth_grants_test.go`.
- [ ] Test that `000015_add_user_language.up.sql` contains a guarded grant to `openoms` for `find_user_for_auth`.
- [ ] Test that `000015_add_user_language.down.sql` contains the same guarded grant.
- [ ] Test that the latest repair migration grants `openoms` execute on these auth helper functions:
  - `find_tenant_by_slug(text)`
  - `find_user_for_auth(text, uuid)`
  - `find_invitation_by_token(text)`
  - `find_return_by_token(text)`
  - `find_order_tenant_id(uuid)`
  - `use_invitation(text)`
- [ ] Run:

```bash
cd apps/api-server
go test ./internal/database -run TestAuthFunctionMigrationsGrantOpenOMSRole -count=1
```

Expected before implementation: FAIL because the `openoms` grant is missing.

### Task 2: Migration Fix

- [ ] Patch both `000015_add_user_language.up.sql` and `.down.sql` with:

```sql
IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'openoms') THEN
  GRANT EXECUTE ON FUNCTION public.find_user_for_auth(text, uuid) TO openoms;
END IF;
```

- [ ] Add `000025_auth_function_openoms_grants.up.sql` with guarded grants for the six auth helper functions.
- [ ] Add `000025_auth_function_openoms_grants.down.sql` with guarded revokes for only those explicit `openoms` grants.

### Task 3: GREEN Verification

- [ ] Run the targeted database test again:

```bash
cd apps/api-server
go test ./internal/database -run TestAuthFunctionMigrationsGrantOpenOMSRole -count=1
```

Expected: PASS.

- [ ] Run migration-adjacent checks:

```bash
cd apps/api-server
go test ./internal/database ./internal/service -count=1
```

Expected: PASS.

### Task 4: Repository Validation

- [ ] Run:

```bash
git diff --check
```

Expected: no output.

- [ ] Run full local CI before push:

```bash
./scripts/local-ci.sh
```

Expected: all checks pass.

### Task 5: PR And Post-Merge

- [ ] Commit with Linear ID:

```bash
git add apps/api-server/migrations/000015_add_user_language.up.sql \
  apps/api-server/migrations/000015_add_user_language.down.sql \
  apps/api-server/migrations/000025_auth_function_openoms_grants.up.sql \
  apps/api-server/migrations/000025_auth_function_openoms_grants.down.sql \
  apps/api-server/internal/database/migration_auth_grants_test.go \
  docs/superpowers/plans/2026-05-17-ope-318-auth-function-openoms-grants.md
git commit -m "OPE-318: grant auth functions to self-hosted app role"
```

- [ ] Push branch and open PR titled `OPE-318: grant auth functions to self-hosted app role`.
- [ ] Inspect CI and CodeRabbit comments.
- [ ] Merge only when checks and actionable review comments are clear.
- [ ] Verify public main CI/Release and enterprise deploy/SBOM dispatch after merge.

## Risk And Rollback

- Risk: granting too broadly could expose auth helper functions to unexpected roles. Mitigation: grant only to existing `openoms`, the documented self-hosted app role, and only for existing auth helper functions already intended for application use.
- Rollback: `000025_auth_function_openoms_grants.down.sql` revokes explicit grants from `openoms`. The `000015` down path keeps rollback login usable for self-hosted deployments.
- Existing production: append-only `000025` repairs already-migrated databases; editing `000015` alone would not.

