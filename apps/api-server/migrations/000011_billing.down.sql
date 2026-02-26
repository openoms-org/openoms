-- apps/api-server/migrations/000011_billing.down.sql

DROP FUNCTION IF EXISTS public.billing_claim_checkout_session(text, uuid);
DROP FUNCTION IF EXISTS public.billing_get_checkout_session(text);
DROP FUNCTION IF EXISTS public.billing_complete_checkout_session(text, text);
DROP FUNCTION IF EXISTS public.billing_create_checkout_session(text, text, text);

DROP TABLE IF EXISTS public.billing_subscriptions;
DROP TABLE IF EXISTS public.billing_customers;
DROP TABLE IF EXISTS public.billing_checkout_sessions;
