-- Revert get_tenant_plan to tenant plan/settings only.

CREATE OR REPLACE FUNCTION public.get_tenant_plan(p_tenant_id uuid)
RETURNS TABLE(plan text, settings jsonb)
LANGUAGE sql
SECURITY DEFINER
SET search_path = 'public'
AS $$
    SELECT t.plan, t.settings FROM tenants t WHERE t.id = p_tenant_id;
$$;

REVOKE EXECUTE ON FUNCTION public.get_tenant_plan(uuid) FROM PUBLIC;

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'openoms') THEN
    EXECUTE 'GRANT EXECUTE ON FUNCTION public.get_tenant_plan(uuid) TO openoms';
  END IF;
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'openoms_app') THEN
    EXECUTE 'GRANT EXECUTE ON FUNCTION public.get_tenant_plan(uuid) TO openoms_app';
  END IF;
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'openoms_auth') THEN
    EXECUTE 'GRANT EXECUTE ON FUNCTION public.get_tenant_plan(uuid) TO openoms_auth';
  END IF;
END;
$$;
