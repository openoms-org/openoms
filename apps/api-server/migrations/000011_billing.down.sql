-- apps/api-server/migrations/000011_billing.down.sql

DROP FUNCTION IF EXISTS public.billing_sync_tenant_plan(uuid, jsonb);
DROP FUNCTION IF EXISTS public.billing_update_subscription_by_stripe_id(text, text, timestamptz, timestamptz, timestamptz);
DROP FUNCTION IF EXISTS public.billing_upsert_subscription(uuid, text, text, text, text, text, timestamptz, timestamptz, timestamptz);
DROP FUNCTION IF EXISTS public.billing_get_customer_by_stripe_id(text);
DROP FUNCTION IF EXISTS public.billing_create_customer(uuid, text);
DROP FUNCTION IF EXISTS public.billing_update_checkout_tenant(text, uuid);
DROP FUNCTION IF EXISTS public.billing_claim_checkout_session(text);
DROP FUNCTION IF EXISTS public.billing_get_checkout_session(text);
DROP FUNCTION IF EXISTS public.billing_complete_checkout_session(text, text);
DROP FUNCTION IF EXISTS public.billing_create_checkout_session(text, text, text);

DROP TABLE IF EXISTS public.billing_subscriptions;
DROP TABLE IF EXISTS public.billing_customers;
DROP TABLE IF EXISTS public.billing_checkout_sessions;
