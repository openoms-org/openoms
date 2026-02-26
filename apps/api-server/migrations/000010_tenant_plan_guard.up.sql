-- apps/api-server/migrations/000010_tenant_plan_guard.up.sql

-- SECURITY DEFINER: get tenant plan and settings without RLS context.
-- Used by plan enforcement middleware (runs before WithTenant transaction).
CREATE OR REPLACE FUNCTION public.get_tenant_plan(p_tenant_id uuid)
RETURNS TABLE(plan text, settings jsonb)
LANGUAGE sql
SECURITY DEFINER
SET search_path = 'public'
AS $$
    SELECT t.plan, t.settings FROM tenants t WHERE t.id = p_tenant_id;
$$;

-- Grant to the app role (not just auth role — this runs for authenticated requests)
GRANT EXECUTE ON FUNCTION public.get_tenant_plan(uuid) TO openoms;
GRANT EXECUTE ON FUNCTION public.get_tenant_plan(uuid) TO openoms_auth;
