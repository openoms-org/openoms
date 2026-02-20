-- Fix SECURITY DEFINER function owners.
-- These functions must be owned by postgres (superuser) to bypass
-- FORCE ROW LEVEL SECURITY on tenant-scoped tables.
-- When owned by 'openoms', FORCE RLS blocks them from reading rows
-- without a tenant context, breaking authentication.

ALTER FUNCTION find_tenant_by_slug(text) OWNER TO postgres;
ALTER FUNCTION find_user_for_auth(text, uuid) OWNER TO postgres;
ALTER FUNCTION find_order_tenant_id(uuid) OWNER TO postgres;
ALTER FUNCTION find_return_by_token(text) OWNER TO postgres;
