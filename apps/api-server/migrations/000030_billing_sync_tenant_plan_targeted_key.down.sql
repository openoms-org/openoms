-- OPE-477 rollback. WARNING: this restores the pre-000030 definition of
-- billing_sync_tenant_plan, which shallow-merges the caller's whole JSONB blob into
-- tenants.settings (`settings || p_settings`) and can therefore clobber arbitrary
-- settings keys. The migration is intended to be forward-only; only run this down to
-- recover a broken deploy, not as routine.

CREATE OR REPLACE FUNCTION public.billing_sync_tenant_plan(
    p_tenant_id uuid,
    p_settings jsonb
)
RETURNS void
LANGUAGE sql
SECURITY DEFINER
SET search_path = 'public'
AS $$
    UPDATE tenants
    SET settings = COALESCE(settings, '{}'::jsonb) || p_settings,
        updated_at = now()
    WHERE id = p_tenant_id;
$$;

REVOKE EXECUTE ON FUNCTION public.billing_sync_tenant_plan(uuid, jsonb) FROM PUBLIC;

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'openoms') THEN
    EXECUTE 'GRANT EXECUTE ON FUNCTION public.billing_sync_tenant_plan(uuid, jsonb) TO openoms';
  END IF;
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'openoms_app') THEN
    EXECUTE 'GRANT EXECUTE ON FUNCTION public.billing_sync_tenant_plan(uuid, jsonb) TO openoms_app';
  END IF;
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'openoms_auth') THEN
    EXECUTE 'GRANT EXECUTE ON FUNCTION public.billing_sync_tenant_plan(uuid, jsonb) TO openoms_auth';
  END IF;
END;
$$;
