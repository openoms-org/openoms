# OPE-292 RLS Missing OK Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix late RLS policies that call `current_setting('app.current_tenant_id')` without `missing_ok=true`, and ensure the affected tenant-scoped tables enforce RLS for table owners.

**Architecture:** Keep the historical migrations correct for fresh databases, and add a new forward-only migration for already-deployed databases. The runtime policy expression should match the stricter modern pattern: `tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid`, with explicit `WITH CHECK` for writes.

**Tech Stack:** PostgreSQL RLS, golang-migrate SQL migrations, OpenOMS public repo CI.

---

## Scope

Linear: `OPE-292`

Repo: `public`

In scope:
- Fix `000005_allegro_parameter_mappings.up.sql`.
- Fix `000006_message_templates.up.sql`.
- Add a new migration `000024_fix_late_rls_missing_ok` for existing databases.
- Add a regression guard that fails when RLS policies for the affected tables do not use `missing_ok=true` or when `FORCE ROW LEVEL SECURITY` is missing.
- Update security/domain context docs.

Out of scope:
- Broader RLS cleanup for all historical policies.
- `OPE-293` migration lock-timeout work.
- SBOM / Dependency-Track token lifecycle.

## Files

- Modify: `apps/api-server/migrations/000005_allegro_parameter_mappings.up.sql`
  - Add `FORCE ROW LEVEL SECURITY`.
  - Replace unsafe `current_setting('app.current_tenant_id')::uuid` with `NULLIF(current_setting('app.current_tenant_id', true), '')::uuid`.
  - Add explicit `WITH CHECK`.

- Modify: `apps/api-server/migrations/000006_message_templates.up.sql`
  - Replace unsafe `current_setting('app.current_tenant_id')::uuid`.
  - Add explicit `WITH CHECK`.

- Create: `apps/api-server/migrations/000024_fix_late_rls_missing_ok.up.sql`
  - Patch already-deployed databases.
  - Recreate policies on `public.allegro_parameter_mappings` and `public.message_templates`.
  - Enforce RLS on both tables.
  - Include a migration sanity check.

- Create: `apps/api-server/migrations/000024_fix_late_rls_missing_ok.down.sql`
  - Forward-only/no-op rollback that intentionally keeps stricter policies.

- Create: `scripts/check-rls-policies.sql`
  - Validate migrated database policy posture.

- Modify: `.github/workflows/ci.yml`
  - Run `scripts/check-rls-policies.sql` after migrations in the `Test` job.

- Modify: `.claude/context/SECURITY_POSTURE.md`
  - Record OPE-292 fix.

- Modify: `.claude/context/DOMAIN_MODEL.md`
  - Clarify RLS policy pattern uses `NULLIF(current_setting(..., true), '')::uuid` and `FORCE ROW LEVEL SECURITY`.

## Implementation Tasks

### Task 1: Update Historical Fresh-Database Migrations

**Files:**
- Modify: `apps/api-server/migrations/000005_allegro_parameter_mappings.up.sql`
- Modify: `apps/api-server/migrations/000006_message_templates.up.sql`

- [ ] Step 1: Update `000005` RLS policy to the strict pattern.

Expected final block:

```sql
ALTER TABLE allegro_parameter_mappings ENABLE ROW LEVEL SECURITY;
ALTER TABLE allegro_parameter_mappings FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON allegro_parameter_mappings
    USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);
```

- [ ] Step 2: Update `000006` RLS policy to the strict pattern.

Expected final block:

```sql
ALTER TABLE message_templates ENABLE ROW LEVEL SECURITY;

CREATE POLICY message_templates_tenant ON message_templates
    USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

ALTER TABLE message_templates FORCE ROW LEVEL SECURITY;
```

- [ ] Step 3: Run a text check for unsafe policy expressions.

Run:

```bash
rg -n "current_setting\\('app\\.current_tenant_id'\\)::uuid" apps/api-server/migrations/000005_*.up.sql apps/api-server/migrations/000006_*.up.sql
```

Expected: no output.

### Task 2: Add Forward Migration for Existing Databases

**Files:**
- Create: `apps/api-server/migrations/000024_fix_late_rls_missing_ok.up.sql`
- Create: `apps/api-server/migrations/000024_fix_late_rls_missing_ok.down.sql`

- [ ] Step 1: Create the `up` migration.

Expected file:

```sql
-- Fix late RLS policies that were added after 000003 and reintroduced
-- current_setting(... ) without missing_ok=true.

ALTER TABLE public.allegro_parameter_mappings ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.allegro_parameter_mappings FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation ON public.allegro_parameter_mappings;

CREATE POLICY tenant_isolation ON public.allegro_parameter_mappings
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

ALTER TABLE public.message_templates ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.message_templates FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS message_templates_tenant ON public.message_templates;

CREATE POLICY message_templates_tenant ON public.message_templates
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DO $$
DECLARE
    bad_policy_count integer;
    missing_force_count integer;
BEGIN
    SELECT COUNT(*)
    INTO bad_policy_count
    FROM pg_policies
    WHERE schemaname = 'public'
      AND tablename IN ('allegro_parameter_mappings', 'message_templates')
      AND (
          qual NOT LIKE '%current_setting(%true%'
          OR with_check NOT LIKE '%current_setting(%true%'
      );

    IF bad_policy_count <> 0 THEN
        RAISE EXCEPTION 'OPE-292: unsafe RLS policies remain on allegro_parameter_mappings/message_templates';
    END IF;

    SELECT COUNT(*)
    INTO missing_force_count
    FROM pg_class c
    JOIN pg_namespace n ON n.oid = c.relnamespace
    WHERE n.nspname = 'public'
      AND c.relname IN ('allegro_parameter_mappings', 'message_templates')
      AND c.relforcerowsecurity IS DISTINCT FROM true;

    IF missing_force_count <> 0 THEN
        RAISE EXCEPTION 'OPE-292: FORCE ROW LEVEL SECURITY missing on allegro_parameter_mappings/message_templates';
    END IF;
END $$;
```

- [ ] Step 2: Create the `down` migration as intentionally forward-only.

Expected file:

```sql
-- Down migration intentionally keeps the stricter OPE-292 RLS policies.
-- Reintroducing current_setting(... ) without missing_ok=true would break
-- Supabase transaction-mode pooler compatibility and weaken tenant isolation.
```

### Task 3: Add a CI Regression Guard

**Files:**
- Create: `scripts/check-rls-policies.sql`
- Modify: `.github/workflows/ci.yml`

- [ ] Step 1: Create the SQL guard.

Expected file:

```sql
DO $$
DECLARE
    bad_policy text;
    missing_force text;
BEGIN
    SELECT string_agg(format('%I.%I policy %I', schemaname, tablename, policyname), E'\n')
    INTO bad_policy
    FROM pg_policies
    WHERE schemaname = 'public'
      AND tablename IN ('allegro_parameter_mappings', 'message_templates')
      AND (
          qual NOT LIKE '%current_setting(%true%'
          OR with_check NOT LIKE '%current_setting(%true%'
      );

    IF bad_policy IS NOT NULL THEN
        RAISE EXCEPTION 'RLS policies without missing_ok=true:%', E'\n' || bad_policy;
    END IF;

    SELECT string_agg(format('%I.%I', n.nspname, c.relname), E'\n')
    INTO missing_force
    FROM pg_class c
    JOIN pg_namespace n ON n.oid = c.relnamespace
    WHERE n.nspname = 'public'
      AND c.relname IN ('allegro_parameter_mappings', 'message_templates')
      AND c.relforcerowsecurity IS DISTINCT FROM true;

    IF missing_force IS NOT NULL THEN
        RAISE EXCEPTION 'Tenant tables without FORCE ROW LEVEL SECURITY:%', E'\n' || missing_force;
    END IF;
END $$;
```

- [ ] Step 2: Wire the guard after `Check SECURITY DEFINER grants` in `.github/workflows/ci.yml`.

Expected workflow step:

```yaml
      - name: Check RLS policies
        run: |
          docker exec openoms-test-db psql -U postgres -d openoms_test -f /workspace/scripts/check-rls-policies.sql
```

If the CI container does not mount the repository at `/workspace`, use the same copy/mount pattern already used by neighboring migration checks.

### Task 4: Validate Locally

**Files:**
- No new files.

- [ ] Step 1: Run diff whitespace check.

Run:

```bash
git diff --check
```

Expected: no output.

- [ ] Step 2: Run targeted migration SQL syntax validation through the same Postgres path available locally.

Preferred command if the dev DB is running:

```bash
cd apps/api-server
for f in migrations/*.up.sql; do
  psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f "$f"
done
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f ../../scripts/check-rls-policies.sql
```

Expected: all migrations apply, and `check-rls-policies.sql` exits 0.

Fallback if no local DB is running:

```bash
git diff --check
rg -n "current_setting\\('app\\.current_tenant_id'\\)::uuid" apps/api-server/migrations/000005_*.up.sql apps/api-server/migrations/000006_*.up.sql apps/api-server/migrations/000024_*.up.sql
```

Expected: `git diff --check` passes and the unsafe `rg` check has no output.

- [ ] Step 3: Run full public local CI before push.

Run:

```bash
./scripts/local-ci.sh
```

Expected: all checks pass.

### Task 5: Docs and PR

**Files:**
- Modify: `.claude/context/SECURITY_POSTURE.md`
- Modify: `.claude/context/DOMAIN_MODEL.md`

- [ ] Step 1: Add a 2026-05-16 security update bullet.

Suggested text:

```markdown
- OPE-292: late RLS migrations for `allegro_parameter_mappings` and `message_templates` now use `current_setting('app.current_tenant_id', true)` with `NULLIF(..., '')::uuid`, explicit write checks, and `FORCE ROW LEVEL SECURITY`; CI checks the migrated database for these regressions.
```

- [ ] Step 2: Update the RLS pattern in `DOMAIN_MODEL.md`.

Suggested SQL snippet:

```sql
CREATE POLICY xxx_tenant ON xxx
    USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);
```

- [ ] Step 3: Create branch and mark Linear in progress.

Run:

```bash
git checkout -b fix/OPE-292-rls-missing-ok
```

Linear:
- Move `OPE-292` to `In Progress`.

- [ ] Step 4: Commit and PR.

Commit:

```bash
git add apps/api-server/migrations scripts/check-rls-policies.sql .github/workflows/ci.yml .claude/context/SECURITY_POSTURE.md .claude/context/DOMAIN_MODEL.md
git commit -m "OPE-292: fix late RLS missing_ok policies"
```

PR title:

```text
OPE-292: fix late RLS missing_ok policies
```

PR docs section:

```markdown
## Docs updated
- [x] SECURITY_POSTURE.md — documented stricter late RLS policy fix
- [x] DOMAIN_MODEL.md — clarified canonical RLS policy pattern
```

## Risk and Rollback

Risk:
- Low data risk: policy recreation is metadata-only and does not rewrite tenant data.
- Moderate operational risk: stricter `FORCE ROW LEVEL SECURITY` can expose code paths that accidentally query tenant tables without `database.WithTenant`; that is the intended fail-closed behavior.
- Fresh and existing databases both need coverage; old migration edits alone are insufficient.

Rollback:
- Do not roll back to unsafe policies. If production uncovers a missing tenant context, hotfix the offending code path to use `database.WithTenant`.
- If the migration itself fails, Helm atomic deploy should stop before promoting the new release; inspect the migration job logs and fix the SQL.

## Self-Review Checklist

- [ ] `000005` and `000006` no longer contain `current_setting('app.current_tenant_id')::uuid`.
- [ ] Existing DB migration `000024` patches both affected tables.
- [ ] Both tables have `FORCE ROW LEVEL SECURITY`.
- [ ] Policies contain both `USING` and `WITH CHECK`.
- [ ] CI has a regression guard.
- [ ] Docs updated in the same PR.
