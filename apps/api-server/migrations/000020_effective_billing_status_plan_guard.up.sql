-- OPE-207: Plan guard must enforce current Stripe subscription status.
--
-- Keep the public function signature stable for PlanCache, but merge the latest
-- billing_subscriptions.status into returned tenant settings as
-- subscription_status. This lets middleware enforce billing state before tenant
-- RLS context exists, while preserving tenants.plan as the commercial plan ID.

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
