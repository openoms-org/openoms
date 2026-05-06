-- OPE-213: SECURITY DEFINER functions must not be executable by PUBLIC.
--
-- PostgreSQL grants EXECUTE on new functions to PUBLIC by default. These
-- functions intentionally bypass RLS for pre-registration, webhook, billing,
-- license-token, and plan-cache flows, so only application roles should call
-- them.

REVOKE EXECUTE ON FUNCTION public.check_license_token_used(uuid) FROM PUBLIC;
REVOKE EXECUTE ON FUNCTION public.mark_license_token_used(uuid, uuid, text, text) FROM PUBLIC;
REVOKE EXECUTE ON FUNCTION public.update_license_token_tenant(uuid, uuid) FROM PUBLIC;

REVOKE EXECUTE ON FUNCTION public.get_tenant_plan(uuid) FROM PUBLIC;

REVOKE EXECUTE ON FUNCTION public.billing_create_checkout_session(text, text, text) FROM PUBLIC;
REVOKE EXECUTE ON FUNCTION public.billing_complete_checkout_session(text, text) FROM PUBLIC;
REVOKE EXECUTE ON FUNCTION public.billing_get_checkout_session(text) FROM PUBLIC;
REVOKE EXECUTE ON FUNCTION public.billing_claim_checkout_session(text) FROM PUBLIC;
REVOKE EXECUTE ON FUNCTION public.billing_update_checkout_tenant(text, uuid) FROM PUBLIC;
REVOKE EXECUTE ON FUNCTION public.billing_create_customer(uuid, text) FROM PUBLIC;
REVOKE EXECUTE ON FUNCTION public.billing_get_customer_by_stripe_id(text) FROM PUBLIC;
REVOKE EXECUTE ON FUNCTION public.billing_upsert_subscription(uuid, text, text, text, text, text, timestamptz, timestamptz, timestamptz) FROM PUBLIC;
REVOKE EXECUTE ON FUNCTION public.billing_update_subscription_by_stripe_id(text, text, timestamptz, timestamptz, timestamptz) FROM PUBLIC;
REVOKE EXECUTE ON FUNCTION public.billing_sync_tenant_plan(uuid, jsonb) FROM PUBLIC;

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'openoms') THEN
    EXECUTE 'GRANT EXECUTE ON FUNCTION public.check_license_token_used(uuid) TO openoms';
    EXECUTE 'GRANT EXECUTE ON FUNCTION public.mark_license_token_used(uuid, uuid, text, text) TO openoms';
    EXECUTE 'GRANT EXECUTE ON FUNCTION public.update_license_token_tenant(uuid, uuid) TO openoms';
    EXECUTE 'GRANT EXECUTE ON FUNCTION public.get_tenant_plan(uuid) TO openoms';
  END IF;

  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'openoms_app') THEN
    EXECUTE 'GRANT EXECUTE ON FUNCTION public.check_license_token_used(uuid) TO openoms_app';
    EXECUTE 'GRANT EXECUTE ON FUNCTION public.mark_license_token_used(uuid, uuid, text, text) TO openoms_app';
    EXECUTE 'GRANT EXECUTE ON FUNCTION public.update_license_token_tenant(uuid, uuid) TO openoms_app';
    EXECUTE 'GRANT EXECUTE ON FUNCTION public.get_tenant_plan(uuid) TO openoms_app';
  END IF;

  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'openoms_auth') THEN
    EXECUTE 'GRANT EXECUTE ON FUNCTION public.check_license_token_used(uuid) TO openoms_auth';
    EXECUTE 'GRANT EXECUTE ON FUNCTION public.mark_license_token_used(uuid, uuid, text, text) TO openoms_auth';
    EXECUTE 'GRANT EXECUTE ON FUNCTION public.update_license_token_tenant(uuid, uuid) TO openoms_auth';
    EXECUTE 'GRANT EXECUTE ON FUNCTION public.get_tenant_plan(uuid) TO openoms_auth';
  END IF;

  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'openoms') THEN
    EXECUTE 'GRANT EXECUTE ON FUNCTION public.billing_create_checkout_session(text, text, text) TO openoms';
    EXECUTE 'GRANT EXECUTE ON FUNCTION public.billing_complete_checkout_session(text, text) TO openoms';
    EXECUTE 'GRANT EXECUTE ON FUNCTION public.billing_get_checkout_session(text) TO openoms';
    EXECUTE 'GRANT EXECUTE ON FUNCTION public.billing_claim_checkout_session(text) TO openoms';
    EXECUTE 'GRANT EXECUTE ON FUNCTION public.billing_update_checkout_tenant(text, uuid) TO openoms';
    EXECUTE 'GRANT EXECUTE ON FUNCTION public.billing_create_customer(uuid, text) TO openoms';
    EXECUTE 'GRANT EXECUTE ON FUNCTION public.billing_get_customer_by_stripe_id(text) TO openoms';
    EXECUTE 'GRANT EXECUTE ON FUNCTION public.billing_upsert_subscription(uuid, text, text, text, text, text, timestamptz, timestamptz, timestamptz) TO openoms';
    EXECUTE 'GRANT EXECUTE ON FUNCTION public.billing_update_subscription_by_stripe_id(text, text, timestamptz, timestamptz, timestamptz) TO openoms';
    EXECUTE 'GRANT EXECUTE ON FUNCTION public.billing_sync_tenant_plan(uuid, jsonb) TO openoms';
  END IF;

  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'openoms_app') THEN
    EXECUTE 'GRANT EXECUTE ON FUNCTION public.billing_create_checkout_session(text, text, text) TO openoms_app';
    EXECUTE 'GRANT EXECUTE ON FUNCTION public.billing_complete_checkout_session(text, text) TO openoms_app';
    EXECUTE 'GRANT EXECUTE ON FUNCTION public.billing_get_checkout_session(text) TO openoms_app';
    EXECUTE 'GRANT EXECUTE ON FUNCTION public.billing_claim_checkout_session(text) TO openoms_app';
    EXECUTE 'GRANT EXECUTE ON FUNCTION public.billing_update_checkout_tenant(text, uuid) TO openoms_app';
    EXECUTE 'GRANT EXECUTE ON FUNCTION public.billing_create_customer(uuid, text) TO openoms_app';
    EXECUTE 'GRANT EXECUTE ON FUNCTION public.billing_get_customer_by_stripe_id(text) TO openoms_app';
    EXECUTE 'GRANT EXECUTE ON FUNCTION public.billing_upsert_subscription(uuid, text, text, text, text, text, timestamptz, timestamptz, timestamptz) TO openoms_app';
    EXECUTE 'GRANT EXECUTE ON FUNCTION public.billing_update_subscription_by_stripe_id(text, text, timestamptz, timestamptz, timestamptz) TO openoms_app';
    EXECUTE 'GRANT EXECUTE ON FUNCTION public.billing_sync_tenant_plan(uuid, jsonb) TO openoms_app';
  END IF;

  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'openoms_auth') THEN
    EXECUTE 'GRANT EXECUTE ON FUNCTION public.billing_create_checkout_session(text, text, text) TO openoms_auth';
    EXECUTE 'GRANT EXECUTE ON FUNCTION public.billing_complete_checkout_session(text, text) TO openoms_auth';
    EXECUTE 'GRANT EXECUTE ON FUNCTION public.billing_get_checkout_session(text) TO openoms_auth';
    EXECUTE 'GRANT EXECUTE ON FUNCTION public.billing_claim_checkout_session(text) TO openoms_auth';
    EXECUTE 'GRANT EXECUTE ON FUNCTION public.billing_update_checkout_tenant(text, uuid) TO openoms_auth';
    EXECUTE 'GRANT EXECUTE ON FUNCTION public.billing_create_customer(uuid, text) TO openoms_auth';
  END IF;
END;
$$;
