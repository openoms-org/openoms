-- Keep the self-hosted least-privilege app role usable when it is created
-- before migrations by docker-compose.prod.yml, and repair existing local
-- volumes where earlier guarded grants were skipped because the role did not
-- exist yet.

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'openoms_app') THEN
    EXECUTE 'GRANT USAGE ON SCHEMA public TO openoms_app';
    EXECUTE 'GRANT ALL ON ALL TABLES IN SCHEMA public TO openoms_app';
    EXECUTE 'GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO openoms_app';
    EXECUTE 'ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO openoms_app';
    EXECUTE 'ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT USAGE, SELECT ON SEQUENCES TO openoms_app';

    IF to_regprocedure('public.find_tenant_by_slug(text)') IS NOT NULL THEN
      EXECUTE 'GRANT EXECUTE ON FUNCTION public.find_tenant_by_slug(text) TO openoms_app';
    END IF;
    IF to_regprocedure('public.find_user_for_auth(text, uuid)') IS NOT NULL THEN
      EXECUTE 'GRANT EXECUTE ON FUNCTION public.find_user_for_auth(text, uuid) TO openoms_app';
    END IF;
    IF to_regprocedure('public.find_invitation_by_token(text)') IS NOT NULL THEN
      EXECUTE 'GRANT EXECUTE ON FUNCTION public.find_invitation_by_token(text) TO openoms_app';
    END IF;
    IF to_regprocedure('public.find_return_by_token(text)') IS NOT NULL THEN
      EXECUTE 'GRANT EXECUTE ON FUNCTION public.find_return_by_token(text) TO openoms_app';
    END IF;
    IF to_regprocedure('public.find_order_tenant_id(uuid)') IS NOT NULL THEN
      EXECUTE 'GRANT EXECUTE ON FUNCTION public.find_order_tenant_id(uuid) TO openoms_app';
    END IF;
    IF to_regprocedure('public.use_invitation(text)') IS NOT NULL THEN
      EXECUTE 'GRANT EXECUTE ON FUNCTION public.use_invitation(text) TO openoms_app';
    END IF;

    IF to_regprocedure('public.check_license_token_used(uuid)') IS NOT NULL THEN
      EXECUTE 'GRANT EXECUTE ON FUNCTION public.check_license_token_used(uuid) TO openoms_app';
    END IF;
    IF to_regprocedure('public.mark_license_token_used(uuid, uuid, text, text)') IS NOT NULL THEN
      EXECUTE 'GRANT EXECUTE ON FUNCTION public.mark_license_token_used(uuid, uuid, text, text) TO openoms_app';
    END IF;
    IF to_regprocedure('public.update_license_token_tenant(uuid, uuid)') IS NOT NULL THEN
      EXECUTE 'GRANT EXECUTE ON FUNCTION public.update_license_token_tenant(uuid, uuid) TO openoms_app';
    END IF;
    IF to_regprocedure('public.get_tenant_plan(uuid)') IS NOT NULL THEN
      EXECUTE 'GRANT EXECUTE ON FUNCTION public.get_tenant_plan(uuid) TO openoms_app';
    END IF;

    IF to_regprocedure('public.billing_create_checkout_session(text, text, text)') IS NOT NULL THEN
      EXECUTE 'GRANT EXECUTE ON FUNCTION public.billing_create_checkout_session(text, text, text) TO openoms_app';
    END IF;
    IF to_regprocedure('public.billing_complete_checkout_session(text, text)') IS NOT NULL THEN
      EXECUTE 'GRANT EXECUTE ON FUNCTION public.billing_complete_checkout_session(text, text) TO openoms_app';
    END IF;
    IF to_regprocedure('public.billing_get_checkout_session(text)') IS NOT NULL THEN
      EXECUTE 'GRANT EXECUTE ON FUNCTION public.billing_get_checkout_session(text) TO openoms_app';
    END IF;
    IF to_regprocedure('public.billing_claim_checkout_session(text)') IS NOT NULL THEN
      EXECUTE 'GRANT EXECUTE ON FUNCTION public.billing_claim_checkout_session(text) TO openoms_app';
    END IF;
    IF to_regprocedure('public.billing_update_checkout_tenant(text, uuid)') IS NOT NULL THEN
      EXECUTE 'GRANT EXECUTE ON FUNCTION public.billing_update_checkout_tenant(text, uuid) TO openoms_app';
    END IF;
    IF to_regprocedure('public.billing_create_customer(uuid, text)') IS NOT NULL THEN
      EXECUTE 'GRANT EXECUTE ON FUNCTION public.billing_create_customer(uuid, text) TO openoms_app';
    END IF;
    IF to_regprocedure('public.billing_get_customer_by_stripe_id(text)') IS NOT NULL THEN
      EXECUTE 'GRANT EXECUTE ON FUNCTION public.billing_get_customer_by_stripe_id(text) TO openoms_app';
    END IF;
    IF to_regprocedure('public.billing_upsert_subscription(uuid, text, text, text, text, text, timestamptz, timestamptz, timestamptz)') IS NOT NULL THEN
      EXECUTE 'GRANT EXECUTE ON FUNCTION public.billing_upsert_subscription(uuid, text, text, text, text, text, timestamptz, timestamptz, timestamptz) TO openoms_app';
    END IF;
    IF to_regprocedure('public.billing_update_subscription_by_stripe_id(text, text, timestamptz, timestamptz, timestamptz)') IS NOT NULL THEN
      EXECUTE 'GRANT EXECUTE ON FUNCTION public.billing_update_subscription_by_stripe_id(text, text, timestamptz, timestamptz, timestamptz) TO openoms_app';
    END IF;
    IF to_regprocedure('public.billing_sync_tenant_plan(uuid, jsonb)') IS NOT NULL THEN
      EXECUTE 'GRANT EXECUTE ON FUNCTION public.billing_sync_tenant_plan(uuid, jsonb) TO openoms_app';
    END IF;
    IF to_regprocedure('public.billing_complete_checkout_session_with_refs(text, text, text, text, text, timestamptz, timestamptz, timestamptz)') IS NOT NULL THEN
      EXECUTE 'GRANT EXECUTE ON FUNCTION public.billing_complete_checkout_session_with_refs(text, text, text, text, text, timestamptz, timestamptz, timestamptz) TO openoms_app';
    END IF;
    IF to_regprocedure('public.billing_get_checkout_session_with_refs(text)') IS NOT NULL THEN
      EXECUTE 'GRANT EXECUTE ON FUNCTION public.billing_get_checkout_session_with_refs(text) TO openoms_app';
    END IF;
  END IF;
END;
$$;
