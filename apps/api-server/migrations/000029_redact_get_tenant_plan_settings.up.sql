-- OPE-475: get_tenant_plan returned the entire tenants.settings JSONB blob, which
-- contains AES-256-GCM encrypted fields (email.smtp_pass, sms.api_token, ksef.token,
-- invoicing.credentials.*, webhooks.endpoints[].secret). Those encrypted values were
-- reaching the plan cache and the plan-guard middleware. The plan guard only reads
-- `limits` (plaintext integers) and the injected `subscription_status`, so return ONLY
-- those — encrypted blobs must never leave this SECURITY DEFINER boundary. The function
-- signature is unchanged for zero-downtime compatibility with already-running pods.

CREATE OR REPLACE FUNCTION public.get_tenant_plan(p_tenant_id uuid)
RETURNS TABLE(plan text, settings jsonb)
LANGUAGE sql
SECURITY DEFINER
SET search_path = 'public'
AS $$
    SELECT
        t.plan,
        CASE
            WHEN latest_sub.status IS NULL THEN
                jsonb_build_object('limits', t.settings -> 'limits')
            ELSE
                jsonb_build_object('limits', t.settings -> 'limits')
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
