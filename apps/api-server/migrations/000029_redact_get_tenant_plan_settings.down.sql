-- OPE-475 rollback. WARNING: this restores the pre-000029 definition of
-- get_tenant_plan, which returns the FULL tenants.settings JSONB blob — including
-- AES-256-GCM encrypted fields — to the plan cache/middleware. This re-opens the
-- settings-leak the up migration closed. The migration is intended to be
-- forward-only; only run this down to recover a broken deploy, not as routine.

CREATE OR REPLACE FUNCTION public.get_tenant_plan(p_tenant_id uuid)
RETURNS TABLE(plan text, settings jsonb)
LANGUAGE sql
SECURITY DEFINER
SET search_path = 'public'
AS $$
    SELECT
        t.plan,
        CASE
            WHEN latest_sub.status IS NULL THEN t.settings
            ELSE COALESCE(t.settings, '{}'::jsonb)
                || jsonb_build_object('subscription_status', latest_sub.status)
        END AS settings
    FROM tenants t
    LEFT JOIN LATERAL (
        SELECT bs.status
        FROM billing_subscriptions bs
        WHERE bs.tenant_id = t.id
        ORDER BY bs.created_at DESC, bs.updated_at DESC
        LIMIT 1
    ) AS latest_sub ON true
    WHERE t.id = p_tenant_id;
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
