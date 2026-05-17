DROP FUNCTION IF EXISTS public.billing_get_checkout_session_with_refs(text);
DROP FUNCTION IF EXISTS public.billing_complete_checkout_session_with_refs(text, text, text, text, text, timestamptz, timestamptz, timestamptz);

DROP INDEX IF EXISTS public.idx_billing_checkout_sessions_stripe_subscription_id;
DROP INDEX IF EXISTS public.idx_billing_checkout_sessions_stripe_customer_id;

ALTER TABLE public.billing_checkout_sessions
    DROP CONSTRAINT IF EXISTS billing_checkout_sessions_subscription_status_check,
    DROP COLUMN IF EXISTS current_period_end,
    DROP COLUMN IF EXISTS current_period_start,
    DROP COLUMN IF EXISTS trial_end,
    DROP COLUMN IF EXISTS subscription_status,
    DROP COLUMN IF EXISTS stripe_subscription_id,
    DROP COLUMN IF EXISTS stripe_customer_id;
