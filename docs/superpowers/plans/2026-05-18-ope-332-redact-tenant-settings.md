# OPE-332 Redact Tenant Settings From Slug Lookup Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Stop `find_tenant_by_slug` from returning tenant `settings` JSONB secrets on login/tracking lookup paths.

**Architecture:** Keep the existing function signature for one-step, zero-downtime compatibility with currently running pods, but recreate the function so the `settings` column is always returned as `{}`. This removes the sensitive data flow immediately while preserving callers that still scan seven columns. A later cleanup can remove the compatibility column once all deployed code no longer needs it.

**Tech Stack:** PostgreSQL migrations via golang-migrate, Go static regression tests, OpenOMS public API server repository, Markdown system documentation.

---

## Files

- Modify: `apps/api-server/internal/database/migration_auth_grants_test.go`
  - Add a focused static regression test proving the new migration redacts `settings`, preserves SECURITY DEFINER hardening, and rollback restores the previous function body.
- Create: `apps/api-server/migrations/000027_redact_find_tenant_by_slug_settings.up.sql`
  - Recreate `public.find_tenant_by_slug(text)` under `openoms_auth` with the same return signature and `settings` as `'{}'::jsonb`.
- Create: `apps/api-server/migrations/000027_redact_find_tenant_by_slug_settings.down.sql`
  - Restore the previous function body that returns `t.settings` for rollback compatibility.
- Modify: `docs/system-documentation.md`
  - Document that the login slug lookup intentionally returns redacted settings and that full tenant settings must be read only through tenant-scoped repository paths.

## Task 1: Regression Test

**Files:**
- Modify: `apps/api-server/internal/database/migration_auth_grants_test.go`

- [ ] **Step 1: Add failing static migration test**

Add a test that expects migration `000027` to exist and to redact settings:

```go
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
```

Extend the `readMigrationSQL` allowlist with:

```go
"000027_redact_find_tenant_by_slug_settings.up.sql",
"000027_redact_find_tenant_by_slug_settings.down.sql",
```

- [ ] **Step 2: Run RED**

Run:

```bash
cd apps/api-server
go test ./internal/database -run TestFindTenantBySlugMigrationRedactsSettings -count=1
```

Expected: FAIL because the `000027` migration files do not exist yet.

## Task 2: Migration

**Files:**
- Create: `apps/api-server/migrations/000027_redact_find_tenant_by_slug_settings.up.sql`
- Create: `apps/api-server/migrations/000027_redact_find_tenant_by_slug_settings.down.sql`

- [ ] **Step 1: Add the up migration**

Create `000027_redact_find_tenant_by_slug_settings.up.sql`:

```sql
-- OPE-332: avoid exposing tenant settings through the login slug lookup.
--
-- Keep the existing result shape for zero-downtime deploy compatibility with
-- already-running pods, but return a redacted JSON object instead of
-- tenants.settings. Full settings must be read through tenant-scoped paths.

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'pg_database_owner') THEN
    EXECUTE 'SET ROLE pg_database_owner';
    GRANT CREATE ON SCHEMA public TO openoms_auth;
    RESET ROLE;
  ELSE
    GRANT CREATE ON SCHEMA public TO openoms_auth;
  END IF;
END;
$$;

SET ROLE openoms_auth;

CREATE OR REPLACE FUNCTION public.find_tenant_by_slug(p_slug text)
 RETURNS TABLE(id uuid, name character varying, slug character varying, plan text, settings jsonb, created_at timestamp with time zone, updated_at timestamp with time zone)
 LANGUAGE sql STABLE SECURITY DEFINER
 SET search_path TO 'public'
AS $$
    SELECT t.id, t.name, t.slug, t.plan, '{}'::jsonb AS settings, t.created_at, t.updated_at
    FROM tenants t
    WHERE t.slug = p_slug;
$$;

RESET ROLE;

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'pg_database_owner') THEN
    EXECUTE 'SET ROLE pg_database_owner';
    REVOKE CREATE ON SCHEMA public FROM openoms_auth;
    RESET ROLE;
  ELSE
    REVOKE CREATE ON SCHEMA public FROM openoms_auth;
  END IF;
END;
$$;

REVOKE EXECUTE ON FUNCTION public.find_tenant_by_slug(text) FROM PUBLIC;

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'openoms_app') THEN
    GRANT EXECUTE ON FUNCTION public.find_tenant_by_slug(text) TO openoms_app;
  END IF;
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'openoms') THEN
    GRANT EXECUTE ON FUNCTION public.find_tenant_by_slug(text) TO openoms;
  END IF;
END;
$$;
```

- [ ] **Step 2: Add the down migration**

Create `000027_redact_find_tenant_by_slug_settings.down.sql`:

```sql
-- Restore the pre-OPE-332 function body for rollback compatibility.

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'pg_database_owner') THEN
    EXECUTE 'SET ROLE pg_database_owner';
    GRANT CREATE ON SCHEMA public TO openoms_auth;
    RESET ROLE;
  ELSE
    GRANT CREATE ON SCHEMA public TO openoms_auth;
  END IF;
END;
$$;

SET ROLE openoms_auth;

CREATE OR REPLACE FUNCTION public.find_tenant_by_slug(p_slug text)
 RETURNS TABLE(id uuid, name character varying, slug character varying, plan text, settings jsonb, created_at timestamp with time zone, updated_at timestamp with time zone)
 LANGUAGE sql STABLE SECURITY DEFINER
 SET search_path TO 'public'
AS $$
    SELECT t.id, t.name, t.slug, t.plan, t.settings, t.created_at, t.updated_at
    FROM tenants t
    WHERE t.slug = p_slug;
$$;

RESET ROLE;

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'pg_database_owner') THEN
    EXECUTE 'SET ROLE pg_database_owner';
    REVOKE CREATE ON SCHEMA public FROM openoms_auth;
    RESET ROLE;
  ELSE
    REVOKE CREATE ON SCHEMA public FROM openoms_auth;
  END IF;
END;
$$;

REVOKE EXECUTE ON FUNCTION public.find_tenant_by_slug(text) FROM PUBLIC;

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'openoms_app') THEN
    GRANT EXECUTE ON FUNCTION public.find_tenant_by_slug(text) TO openoms_app;
  END IF;
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'openoms') THEN
    GRANT EXECUTE ON FUNCTION public.find_tenant_by_slug(text) TO openoms;
  END IF;
END;
$$;
```

- [ ] **Step 3: Run GREEN**

Run:

```bash
cd apps/api-server
go test ./internal/database -run TestFindTenantBySlugMigrationRedactsSettings -count=1
```

Expected: PASS.

## Task 3: Documentation

**Files:**
- Modify: `docs/system-documentation.md`

- [ ] **Step 1: Update SECURITY DEFINER table**

Change the `find_tenant_by_slug(slug)` row to:

```markdown
| `find_tenant_by_slug(slug)` | Login: znalezienie tenanta po slug; zwraca zredagowane `{}` w polu `settings` dla kompatybilnosci sygnatury |
```

- [ ] **Step 2: Add one sentence under the table**

Add:

```markdown
Pelne `tenants.settings` sa odczytywane tylko przez tenant-scoped sciezki repozytorium, nie przez publiczny/loginowy lookup sluga.
```

## Task 4: Validation And PR

**Files:**
- All touched files.

- [ ] **Step 1: Run focused tests**

Run:

```bash
cd apps/api-server
go test ./internal/database -count=1
```

Expected: PASS.

- [ ] **Step 2: Run migration and diff checks**

Run:

```bash
cd .
git diff --check
./scripts/local-ci.sh
```

Expected: both PASS.

- [ ] **Step 3: Commit**

Run:

```bash
cd .
git add apps/api-server/internal/database/migration_auth_grants_test.go apps/api-server/migrations/000027_redact_find_tenant_by_slug_settings.up.sql apps/api-server/migrations/000027_redact_find_tenant_by_slug_settings.down.sql docs/system-documentation.md docs/superpowers/plans/2026-05-18-ope-332-redact-tenant-settings.md
git commit -m "OPE-332: redact tenant settings from slug lookup"
```

- [ ] **Step 4: Push and open PR**

Run:

```bash
git push -u origin fix/OPE-332-redact-tenant-settings
```

Open PR:

```markdown
OPE-332: redact tenant settings from auth slug lookup
```

PR body must include:

```markdown
## Summary
- Redacts `settings` from `find_tenant_by_slug` while preserving the existing function signature for zero-downtime compatibility.
- Keeps explicit EXECUTE grants for application roles and PUBLIC revoke.
- Documents the login lookup behavior.

## Test plan
- `cd apps/api-server && go test ./internal/database -count=1`
- `git diff --check`
- `./scripts/local-ci.sh`

## Docs updated
- [x] `docs/system-documentation.md` — documented redacted `find_tenant_by_slug` settings behavior
```

## Risk And Rollback

- Risk: replacing `find_tenant_by_slug` takes a short function DDL lock. Because the signature is unchanged, existing pods continue to scan the same seven columns after commit.
- Risk: rollback restores the previous behavior, including full `settings`; if rollback is needed, open a follow-up task to re-apply the redaction once the deploy issue is fixed.
- No API shape, dashboard behavior, or public Helm chart changes are expected.
