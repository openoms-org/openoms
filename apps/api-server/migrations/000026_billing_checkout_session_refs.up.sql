-- OPE-319: Persist Stripe checkout references before post-registration finalization.
--
-- The previous flow needed a live Stripe session fetch after registration to
-- create billing_customers and billing_subscriptions. If that fetch failed, the
-- tenant was registered but billing records were missing. These nullable fields
-- let webhook/status completion store enough data for finalization fallback.

ALTER TABLE public.billing_checkout_sessions
    ADD COLUMN IF NOT EXISTS stripe_customer_id text,
    ADD COLUMN IF NOT EXISTS stripe_subscription_id text,
    ADD COLUMN IF NOT EXISTS subscription_status text,
    ADD COLUMN IF NOT EXISTS trial_end timestamptz,
    ADD COLUMN IF NOT EXISTS current_period_start timestamptz,
    ADD COLUMN IF NOT EXISTS current_period_end timestamptz;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'billing_checkout_sessions_subscription_status_check'
      AND conrelid = 'public.billing_checkout_sessions'::regclass
  ) THEN
    ALTER TABLE public.billing_checkout_sessions
      ADD CONSTRAINT billing_checkout_sessions_subscription_status_check
      CHECK (
        subscription_status IS NULL OR
        subscription_status IN ('incomplete', 'incomplete_expired', 'trialing', 'active', 'past_due', 'canceled', 'unpaid', 'paused')
      );
  END IF;
END;
$$;

CREATE INDEX IF NOT EXISTS idx_billing_checkout_sessions_stripe_customer_id
    ON public.billing_checkout_sessions (stripe_customer_id)
    WHERE stripe_customer_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_billing_checkout_sessions_stripe_subscription_id
    ON public.billing_checkout_sessions (stripe_subscription_id)
    WHERE stripe_subscription_id IS NOT NULL;

CREATE OR REPLACE FUNCTION public.billing_complete_checkout_session_with_refs(
    p_stripe_session_id text,
    p_email text,
    p_stripe_customer_id text,
    p_stripe_subscription_id text,
    p_subscription_status text,
    p_trial_end timestamptz,
    p_current_period_start timestamptz,
    p_current_period_end timestamptz
)
RETURNS boolean
LANGUAGE sql
SECURITY DEFINER
SET search_path = 'public'
AS $$
    WITH upd AS (
        UPDATE billing_checkout_sessions
        SET status = CASE WHEN status = 'pending' THEN 'completed' ELSE status END,
            email = COALESCE(NULLIF(p_email, ''), email),
            stripe_customer_id = COALESCE(NULLIF(p_stripe_customer_id, ''), stripe_customer_id),
            stripe_subscription_id = COALESCE(NULLIF(p_stripe_subscription_id, ''), stripe_subscription_id),
            subscription_status = COALESCE(NULLIF(p_subscription_status, ''), subscription_status),
            trial_end = COALESCE(p_trial_end, trial_end),
            current_period_start = COALESCE(p_current_period_start, current_period_start),
            current_period_end = COALESCE(p_current_period_end, current_period_end),
            updated_at = now()
        WHERE stripe_session_id = p_stripe_session_id
          AND status IN ('pending', 'completed', 'registered')
        RETURNING id
    )
    SELECT EXISTS(SELECT 1 FROM upd);
$$;

CREATE OR REPLACE FUNCTION public.billing_get_checkout_session_with_refs(p_stripe_session_id text)
RETURNS TABLE(
    id uuid,
    stripe_session_id text,
    plan text,
    billing_interval text,
    email text,
    status text,
    tenant_id uuid,
    stripe_customer_id text,
    stripe_subscription_id text,
    subscription_status text,
    trial_end timestamptz,
    current_period_start timestamptz,
    current_period_end timestamptz,
    created_at timestamptz
)
LANGUAGE sql
SECURITY DEFINER
SET search_path = 'public'
AS $$
    SELECT cs.id, cs.stripe_session_id, cs.plan, cs.billing_interval,
           cs.email, cs.status, cs.tenant_id,
           cs.stripe_customer_id, cs.stripe_subscription_id, cs.subscription_status,
           cs.trial_end, cs.current_period_start, cs.current_period_end,
           cs.created_at
    FROM billing_checkout_sessions cs
    WHERE cs.stripe_session_id = p_stripe_session_id;
$$;

REVOKE EXECUTE ON FUNCTION public.billing_complete_checkout_session_with_refs(text, text, text, text, text, timestamptz, timestamptz, timestamptz) FROM PUBLIC;
REVOKE EXECUTE ON FUNCTION public.billing_get_checkout_session_with_refs(text) FROM PUBLIC;

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'openoms') THEN
    EXECUTE 'GRANT EXECUTE ON FUNCTION public.billing_complete_checkout_session_with_refs(text, text, text, text, text, timestamptz, timestamptz, timestamptz) TO openoms';
    EXECUTE 'GRANT EXECUTE ON FUNCTION public.billing_get_checkout_session_with_refs(text) TO openoms';
  END IF;

  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'openoms_app') THEN
    EXECUTE 'GRANT EXECUTE ON FUNCTION public.billing_complete_checkout_session_with_refs(text, text, text, text, text, timestamptz, timestamptz, timestamptz) TO openoms_app';
    EXECUTE 'GRANT EXECUTE ON FUNCTION public.billing_get_checkout_session_with_refs(text) TO openoms_app';
  END IF;

  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'openoms_auth') THEN
    EXECUTE 'GRANT EXECUTE ON FUNCTION public.billing_complete_checkout_session_with_refs(text, text, text, text, text, timestamptz, timestamptz, timestamptz) TO openoms_auth';
    EXECUTE 'GRANT EXECUTE ON FUNCTION public.billing_get_checkout_session_with_refs(text) TO openoms_auth';
  END IF;
END;
$$;
