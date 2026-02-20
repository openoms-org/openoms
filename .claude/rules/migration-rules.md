---
paths:
  - "apps/api-server/migrations/**"
---

# Database Migration Rules — OpenOMS

## Safety Rules

1. **Backward-compatible ONLY**. Migrations run as Helm pre-install/pre-upgrade hook — old code may still be running when migration executes.

2. **Expand first, contract later**:
   - Adding column: OK (nullable or with DEFAULT)
   - Renaming column: NO — add new, migrate data, drop old in separate migration
   - Dropping column: Only after all code no longer references it
   - Changing type: Add new column, migrate, drop old

3. **Always provide down migration** (matching `_down.sql` file).

4. **Naming**: `000NNN_description.{up,down}.sql` — sequential numbering, lowercase snake_case.

## Multi-Tenant (RLS) Requirements

Every new tenant-scoped table MUST have:

```sql
-- Enable RLS
ALTER TABLE new_table ENABLE ROW LEVEL SECURITY;

-- Tenant isolation policy
CREATE POLICY new_table_tenant ON new_table
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- Grant to app user
GRANT ALL ON new_table TO openoms;
```

## SECURITY DEFINER Functions

Only create SECURITY DEFINER functions for operations that MUST bypass RLS:
- Authentication (login, register)
- Public access (return forms, tracking pages)

Always use `SECURITY INVOKER` (default) otherwise.

## PostgreSQL Roles

- `postgres`: Superuser, runs migrations
- `openoms`: App user, RLS-scoped

After migration: `GRANT ALL ON ALL TABLES IN SCHEMA public TO openoms;`

## Supabase Considerations

- Migration runs against direct connection (session mode, port 5432), NOT pooler
- `simple_protocol` not used for migrations (direct connection uses extended protocol)
- Test migrations locally with `task migrate` before pushing
