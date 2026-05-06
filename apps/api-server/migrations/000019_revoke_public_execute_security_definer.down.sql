GRANT EXECUTE ON FUNCTION public.check_license_token_used(uuid) TO PUBLIC;
GRANT EXECUTE ON FUNCTION public.mark_license_token_used(uuid, uuid, text, text) TO PUBLIC;
GRANT EXECUTE ON FUNCTION public.update_license_token_tenant(uuid, uuid) TO PUBLIC;

GRANT EXECUTE ON FUNCTION public.get_tenant_plan(uuid) TO PUBLIC;

GRANT EXECUTE ON FUNCTION public.billing_create_checkout_session(text, text, text) TO PUBLIC;
GRANT EXECUTE ON FUNCTION public.billing_complete_checkout_session(text, text) TO PUBLIC;
GRANT EXECUTE ON FUNCTION public.billing_get_checkout_session(text) TO PUBLIC;
GRANT EXECUTE ON FUNCTION public.billing_claim_checkout_session(text) TO PUBLIC;
GRANT EXECUTE ON FUNCTION public.billing_update_checkout_tenant(text, uuid) TO PUBLIC;
GRANT EXECUTE ON FUNCTION public.billing_create_customer(uuid, text) TO PUBLIC;
GRANT EXECUTE ON FUNCTION public.billing_get_customer_by_stripe_id(text) TO PUBLIC;
GRANT EXECUTE ON FUNCTION public.billing_upsert_subscription(uuid, text, text, text, text, text, timestamptz, timestamptz, timestamptz) TO PUBLIC;
GRANT EXECUTE ON FUNCTION public.billing_update_subscription_by_stripe_id(text, text, timestamptz, timestamptz, timestamptz) TO PUBLIC;
GRANT EXECUTE ON FUNCTION public.billing_sync_tenant_plan(uuid, jsonb) TO PUBLIC;
