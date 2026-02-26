-- apps/api-server/migrations/000010_tenant_plan_guard.down.sql

REVOKE EXECUTE ON FUNCTION public.get_tenant_plan(uuid) FROM openoms_auth;
REVOKE EXECUTE ON FUNCTION public.get_tenant_plan(uuid) FROM openoms;
DROP FUNCTION IF EXISTS public.get_tenant_plan(uuid);
