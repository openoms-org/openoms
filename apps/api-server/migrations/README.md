# Database migrations

golang-migrate, applied as a Helm pre-install/pre-upgrade hook. The hygiene rules
below are enforced by the **Migration Safety** CI job (`.github/workflows/ci.yml`);
the ones CI cannot check mechanically are review responsibilities.

## Rules

### 1. Migrations are frozen once merged (forward-only)
Never edit, delete, or rename a migration file after it has merged to `main`.
Production has already run the old file content, so changing it makes fresh
installs diverge from production. To change schema, add a **new** migration.

CI (`Freeze applied migrations`) blocks any modification to an already-merged
`*.sql` file. Override label (rare — e.g. fixing a migration that was never
deployed anywhere): `migration:allow-edit`.

### 2. Backward-compatible only (blue-green safe)
Old and new pods run side by side during a rollout, so a migration must not break
the currently-deployed code. No `DROP COLUMN/TABLE/FUNCTION`, `RENAME`, type
changes, `SET NOT NULL`, or `ADD ... NOT NULL` without a default in a single
deploy — split destructive changes across two deploys (deploy code that stops
using the column first, drop it later). Override label: `migration:destructive`.

### 3. Index builds must not lock tables
`CREATE INDEX` without `CONCURRENTLY` takes an `ACCESS EXCLUSIVE` lock and blocks
all writes to the table while the index builds. Use `CREATE INDEX CONCURRENTLY`.
A brand-new/empty table created in the same migration is exempt (the lock is
harmless) — mark the line with a trailing `-- migrate:index-lock-ok` comment, or
add the `migration:index-lock-ok` PR label.

CI (`Enforce CONCURRENTLY for new indexes`) blocks new non-concurrent indexes.

### 4. Down migrations must not re-open closed gaps
A `*.down.sql` that restores a previous definition can silently re-introduce a
fixed security or correctness problem (e.g. restoring a SECURITY DEFINER function
that leaked encrypted settings, or a fail-open default). Treat security/correctness
migrations as **forward-only**: the `.down.sql` should carry a `WARNING:` comment
stating what it re-opens and exists only to recover a broken deploy — it is not a
routine rollback. This is a **review check** (not mechanically enforced): a
reviewer must confirm no down migration re-opens a previously closed gap.

## schema_migrations
golang-migrate stores exactly **one row** (`version`, `dirty`). Never insert
multiple rows. Migrations apply in ascending version order; gaps are tolerated but
merge migrations in version order so production applies them in sequence.

## Roles (grants)
Production (Supabase) app role is `openoms_app`; self-hosted is `openoms`; an
`openoms_auth` role is used during registration. SECURITY DEFINER functions must
`REVOKE EXECUTE ... FROM PUBLIC` and re-grant to whichever of those roles exist,
guarded by `pg_roles` checks so the same migration runs on both environments.
