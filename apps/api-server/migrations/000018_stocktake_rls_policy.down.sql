-- Down migration intentionally keeps the strict stocktake RLS policies.
-- Reintroducing the old COALESCE(..., tenant_id) policy would reopen a
-- tenant-isolation vulnerability.
SELECT 1;
