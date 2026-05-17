-- apps/api-server/migrations/000010_tenant_plan_guard.down.sql

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'openoms_auth') THEN
    EXECUTE 'REVOKE EXECUTE ON FUNCTION public.get_tenant_plan(uuid) FROM openoms_auth';
  END IF;
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'openoms') THEN
    EXECUTE 'REVOKE EXECUTE ON FUNCTION public.get_tenant_plan(uuid) FROM openoms';
  END IF;
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'openoms_app') THEN
    EXECUTE 'REVOKE EXECUTE ON FUNCTION public.get_tenant_plan(uuid) FROM openoms_app';
  END IF;
END;
$$;

DROP FUNCTION IF EXISTS public.get_tenant_plan(uuid);
