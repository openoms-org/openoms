-- OPE-477: billing_sync_tenant_plan used to concatenate the caller's whole JSONB blob onto
-- tenants.settings (a shallow merge via the jsonb concatenation operator) that replaces whole
-- top-level subtrees with whatever the caller passes. As a SECURITY DEFINER function this lets a caller clobber
-- arbitrary settings keys (limits, encrypted credential blobs, webhook config) and, for any
-- nested object, drop sibling keys it didn't include. The function's only job is to record
-- the subscription status, so update ONLY that key via jsonb_set and leave every other key
-- untouched. The signature is unchanged for zero-downtime compatibility with running pods,
-- so no application code changes are required.

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
    SET settings = CASE
            WHEN p_settings ? 'subscription_status' THEN
                jsonb_set(
                    COALESCE(settings, '{}'::jsonb),
                    '{subscription_status}',
                    p_settings -> 'subscription_status',
                    true
                )
            ELSE settings
        END,
        updated_at = now()
    WHERE id = p_tenant_id;
$$;

-- CREATE OR REPLACE preserves existing privileges; re-assert them defensively so the
-- function carries a complete, self-documenting grant set regardless of prior state.
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
