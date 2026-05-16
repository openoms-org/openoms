# OPE-293 Order Migration Timeouts Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the `000021_unique_order_external_id` migration fail fast instead of holding long locks on `orders`, and add a regression guard for future high-risk order-table migrations.

**Architecture:** Keep the deployed schema shape unchanged. Update the historical migration because it is the source of truth for fresh databases and staging rebuilds, and add a CI guard that blocks non-baseline `orders` DDL/DML migrations unless they set both `lock_timeout` and `statement_timeout`.

**Tech Stack:** PostgreSQL SQL migrations, Bash CI guard, GitHub Actions, OpenOMS public repo documentation.

---

## Scope

In scope:

- Add transaction-local PostgreSQL timeouts to `000021_unique_order_external_id.up.sql`.
- Narrow the duplicate cleanup query so it only ranks keys that actually have duplicates.
- Add a migration safety script that fails when future non-baseline `orders` migrations miss timeouts.
- Wire the guard into public CI.
- Update migration policy docs.

Out of scope:

- No new production data migration. Existing production has already applied `000021`.
- No switch to `CREATE INDEX CONCURRENTLY`; golang-migrate runs SQL migrations in a transaction.
- No data-model change beyond the already-existing unique index behavior.

## Files

- Modify: `apps/api-server/migrations/000021_unique_order_external_id.up.sql`
  - Add `SET LOCAL lock_timeout`.
  - Add `SET LOCAL statement_timeout`.
  - Keep duplicate dedupe semantics and unique index unchanged.
- Modify: `apps/api-server/migrations/000017_missing_fk_indexes.up.sql`
  - Add the same transaction-local timeouts because this migration also creates an `orders` index.
- Create: `scripts/check-migration-timeouts.sh`
  - Scan non-baseline `*.up.sql` migrations for risky `orders` DDL/DML.
  - Require `SET LOCAL lock_timeout` and `SET LOCAL statement_timeout`.
- Modify: `.github/workflows/ci.yml`
  - Run `scripts/check-migration-timeouts.sh` in the test job after the existing SQL guards.
- Modify: `docs/migration-policy.md`
  - Document the timeout requirement for large-table migrations.

## Task 1: Add The Regression Guard First

**Files:**

- Create: `scripts/check-migration-timeouts.sh`

- [ ] Step 1: Create the guard script.

```bash
#!/usr/bin/env bash
set -euo pipefail

root="${1:-apps/api-server/migrations}"
missing=0

while IFS= read -r -d '' file; do
  base="$(basename "$file")"

  # 000001 is the clean baseline schema and intentionally creates all base indexes.
  if [[ "$base" == 000001_* ]]; then
    continue
  fi

  normalized="$(tr '\n' ' ' < "$file" | tr '\t' ' ')"
  if [[ "$normalized" =~ (UPDATE|DELETE[[:space:]]+FROM|ALTER[[:space:]]+TABLE)[[:space:]]+(public\.)?orders ]] ||
     [[ "$normalized" =~ CREATE[[:space:]]+(UNIQUE[[:space:]]+)?INDEX[^;]+ON[[:space:]]+(public\.)?orders ]] ||
     [[ "$normalized" =~ DROP[[:space:]]+INDEX[^;]+idx_orders ]]; then
    if ! grep -Eiq "SET[[:space:]]+LOCAL[[:space:]]+lock_timeout" "$file"; then
      echo "Missing SET LOCAL lock_timeout in $file"
      missing=1
    fi
    if ! grep -Eiq "SET[[:space:]]+LOCAL[[:space:]]+statement_timeout" "$file"; then
      echo "Missing SET LOCAL statement_timeout in $file"
      missing=1
    fi
  fi
done < <(find "$root" -type f -name '*.up.sql' -print0 | sort -z)

if [[ "$missing" -ne 0 ]]; then
  echo "High-risk orders migrations must set transaction-local lock and statement timeouts."
  exit 1
fi

echo "Migration timeout guard passed"
```

- [ ] Step 2: Make the script executable.

Run:

```bash
chmod +x scripts/check-migration-timeouts.sh
```

- [ ] Step 3: Run the guard and confirm RED against current `000021`.

Run:

```bash
scripts/check-migration-timeouts.sh
```

Expected:

```text
Missing SET LOCAL lock_timeout in apps/api-server/migrations/000017_missing_fk_indexes.up.sql
Missing SET LOCAL statement_timeout in apps/api-server/migrations/000017_missing_fk_indexes.up.sql
Missing SET LOCAL lock_timeout in apps/api-server/migrations/000021_unique_order_external_id.up.sql
Missing SET LOCAL statement_timeout in apps/api-server/migrations/000021_unique_order_external_id.up.sql
High-risk orders migrations must set transaction-local lock and statement timeouts.
```

## Task 2: Harden Migration 000021

**Files:**

- Modify: `apps/api-server/migrations/000017_missing_fk_indexes.up.sql`
- Modify: `apps/api-server/migrations/000021_unique_order_external_id.up.sql`

- [ ] Step 1: Add timeouts at the top of both migrations.

Add:

```sql
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '2min';
```

- [ ] Step 2: Replace the duplicate CTE with a narrower two-stage query.

Use:

```sql
WITH duplicate_keys AS (
    SELECT tenant_id, source, external_id
    FROM public.orders
    WHERE external_id IS NOT NULL
      AND external_id <> ''
    GROUP BY tenant_id, source, external_id
    HAVING COUNT(*) > 1
),
duplicate_orders AS (
    SELECT
        o.id,
        o.external_id,
        ROW_NUMBER() OVER (
            PARTITION BY o.tenant_id, o.source, o.external_id
            ORDER BY o.created_at ASC, o.id ASC
        ) AS row_num
    FROM public.orders AS o
    JOIN duplicate_keys AS d
      ON d.tenant_id = o.tenant_id
     AND d.source = o.source
     AND d.external_id = o.external_id
)
UPDATE public.orders AS o
SET
    metadata = COALESCE(o.metadata, '{}'::jsonb) || jsonb_build_object(
        'dedup_original_external_id', o.external_id,
        'dedup_migration', '000021_unique_order_external_id',
        'dedup_at', now()
    ),
    external_id = NULL,
    updated_at = now()
FROM duplicate_orders AS d
WHERE o.id = d.id
  AND d.row_num > 1;
```

- [ ] Step 3: Keep existing index replacement semantics.

The migration must still end with:

```sql
DROP INDEX IF EXISTS public.idx_orders_external;

CREATE UNIQUE INDEX idx_orders_tenant_source_external_id_unique
    ON public.orders USING btree (tenant_id, source, external_id)
    WHERE external_id IS NOT NULL AND external_id <> '';
```

- [ ] Step 4: Run the guard and confirm GREEN.

Run:

```bash
scripts/check-migration-timeouts.sh
```

Expected:

```text
Migration timeout guard passed
```

## Task 3: Validate SQL Behavior

**Files:**

- Test: `apps/api-server/migrations/000021_unique_order_external_id.up.sql`

- [ ] Step 1: Run a fresh PostgreSQL migration pass.

Run:

```bash
container="openoms-ope-293-$(date +%s)"
docker run -d --rm --name "$container" -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=openoms_test postgres:16-alpine >/dev/null
trap 'docker rm -f "$container" >/dev/null 2>&1 || true' EXIT
for i in $(seq 1 30); do
  docker exec "$container" pg_isready -U postgres -d openoms_test >/dev/null 2>&1 && break
  sleep 1
done
docker exec -i "$container" psql -v ON_ERROR_STOP=1 -U postgres -d openoms_test >/dev/null <<'SQL'
CREATE ROLE openoms LOGIN;
CREATE ROLE openoms_app LOGIN;
CREATE ROLE openoms_auth LOGIN;
SQL
for f in apps/api-server/migrations/*.up.sql; do
  docker exec -i "$container" psql -v ON_ERROR_STOP=1 -U postgres -d openoms_test < "$f" >/dev/null
done
docker exec "$container" psql -v ON_ERROR_STOP=1 -U postgres -d openoms_test -c "\d public.orders" >/dev/null
```

Expected: every migration applies successfully and `orders` exists.

- [ ] Step 2: Run a focused duplicate cleanup scenario.

Run a temporary database that applies migrations through `000020`, inserts duplicate marketplace orders, then applies `000021` and verifies exactly one external ID survives for each duplicate key.

Expected SQL assertion:

```sql
SELECT COUNT(*)
FROM public.orders
WHERE source = 'allegro'
  AND external_id = 'A-1';
```

Expected: `1`.

## Task 4: Wire Guard Into CI And Update Docs

**Files:**

- Modify: `.github/workflows/ci.yml`
- Modify: `docs/migration-policy.md`

- [ ] Step 1: Add the CI step after `Check RLS policies`.

```yaml
      - name: Check migration timeouts
        run: scripts/check-migration-timeouts.sh
```

- [ ] Step 2: Document the rule in `docs/migration-policy.md`.

Add:

````markdown
## Large-table migration timeouts

Any non-baseline migration that writes to `orders` or changes `orders` indexes must set transaction-local limits before the DDL/DML:

```sql
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '2min';
```

The goal is to fail deploy safely instead of blocking live writes during a blue-green rollout. CI enforces this with `scripts/check-migration-timeouts.sh`.
````

## Task 5: Final Validation And PR

**Files:**

- All changed files.

- [ ] Step 1: Run diff self-review.

Run:

```bash
git diff --check
git diff --stat
git diff
```

- [ ] Step 2: Run targeted guards.

Run:

```bash
scripts/check-migration-timeouts.sh
scripts/check-rls-policies.sql
```

Note: `scripts/check-rls-policies.sql` must be run through `psql`; do not execute it as shell.

- [ ] Step 3: Run full local CI before push.

Run:

```bash
./scripts/local-ci.sh
```

Expected: all checks pass.

- [ ] Step 4: Commit.

```bash
git add apps/api-server/migrations/000017_missing_fk_indexes.up.sql apps/api-server/migrations/000021_unique_order_external_id.up.sql scripts/check-migration-timeouts.sh .github/workflows/ci.yml docs/migration-policy.md docs/superpowers/plans/2026-05-16-ope-293-order-migration-timeouts.md
git commit -m "OPE-293: add order migration timeouts"
```

- [ ] Step 5: Push and open PR.

Branch: `fix/OPE-293-order-migration-timeouts`

PR title:

```text
OPE-293: add order migration timeouts
```

PR body must include:

```markdown
## Docs updated
- [x] `docs/migration-policy.md` — documented timeout rule for high-risk `orders` migrations
- [x] `docs/superpowers/plans/2026-05-16-ope-293-order-migration-timeouts.md` — implementation plan
```

## Risk And Rollback

Risk:

- Fresh or not-yet-migrated databases with very large `orders` tables can fail `000021` if dedupe or index creation exceeds `statement_timeout`.
- That failure is intentional: it prevents long write blocking during deploy.

Rollback:

- No new forward migration is added, so current production data is not changed by this PR beyond normal redeploy.
- If a not-yet-migrated environment fails on `000021`, rerun in a planned maintenance window with a higher timeout or split dedupe/index work into an operator-run migration.

## Self-Review Checklist

- [ ] Historical migration preserves the unique index contract.
- [ ] Guard fails on the original unsafe migration and passes after the fix.
- [ ] CI runs the guard automatically.
- [ ] Docs explain why the timeout is required.
- [ ] No production secrets or environment-specific values are touched.
