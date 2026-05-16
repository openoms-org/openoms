# Migration Policy (Blue-Green Compatible)

## Rule

Every migration MUST be backward-compatible with the previous code version. During blue-green deployment, old and new application code coexist against the same database.

## Forbidden in a single deploy

- `DROP COLUMN` / `DROP TABLE`
- `RENAME COLUMN`
- `ALTER COLUMN TYPE` (narrowing conversions)
- `ADD COLUMN ... NOT NULL` without `DEFAULT`
- `DROP INDEX` (may break queries in old code)

## Pattern: 2-deploy destructive change

**Deploy 1 (code change):** New code handles both old and new schema.

**Deploy 2 (schema cleanup):** Migration removes old schema elements.

### Examples

| Change | Deploy 1 | Deploy 2 |
|--------|----------|----------|
| Remove column `x` | Code stops reading/writing `x` | `ALTER TABLE DROP COLUMN x` |
| Rename `old_col` → `new_col` | Add `new_col`, code writes to both | Drop `old_col` |
| Change type `text` → `int` | Add `col_v2 int`, code reads both | Drop `col` |
| Add NOT NULL column | `ADD COLUMN x DEFAULT 'val'` | — (single deploy OK with DEFAULT) |

## Large-table migration timeouts

Any non-baseline migration that writes to `orders` or changes `orders` indexes must set transaction-local limits before the DDL/DML:

```sql
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '2min';
```

The goal is to fail deploy safely instead of blocking live writes during a blue-green rollout. CI enforces this with `scripts/check-migration-timeouts.sh`.

## CI enforcement

New migrations in PRs are scanned for destructive SQL operations. The check blocks merge if found.

**Override:** Add the `migration:destructive` label to the PR. Use this only when:
- The destructive change has been split into 2 deploys (this is deploy 2)
- You have confirmed old code is no longer running
