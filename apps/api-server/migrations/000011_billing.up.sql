-- apps/api-server/migrations/000011_billing.up.sql
-- Stripe billing integration: checkout sessions, customers, subscriptions.

-- ============================================================================
-- Table: billing_checkout_sessions
-- Tracks Stripe Checkout Sessions for registration flow.
-- Pre-registration: no tenant context, accessed via SECURITY DEFINER functions.
-- ============================================================================
CREATE TABLE IF NOT EXISTS public.billing_checkout_sessions (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    stripe_session_id text NOT NULL UNIQUE,
    plan              text NOT NULL,
    billing_interval  text NOT NULL CHECK (billing_interval IN ('month', 'year')),
    email             text,
    status            text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'completed', 'registered')),
    tenant_id         uuid REFERENCES public.tenants(id) ON DELETE SET NULL,
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_billing_checkout_sessions_status ON public.billing_checkout_sessions (status);

ALTER TABLE public.billing_checkout_sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.billing_checkout_sessions FORCE ROW LEVEL SECURITY;

-- ============================================================================
-- Table: billing_customers
-- Links a tenant to a Stripe customer.
-- ============================================================================
CREATE TABLE IF NOT EXISTS public.billing_customers (
    id                 uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id          uuid NOT NULL UNIQUE REFERENCES public.tenants(id) ON DELETE CASCADE,
    stripe_customer_id text NOT NULL UNIQUE,
    created_at         timestamptz NOT NULL DEFAULT now()
);

ALTER TABLE public.billing_customers ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.billing_customers FORCE ROW LEVEL SECURITY;

-- RLS policy: tenant can only see their own billing customer record
CREATE POLICY billing_customers_tenant_isolation ON public.billing_customers
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- ============================================================================
-- Table: billing_subscriptions
-- Tracks Stripe subscriptions per tenant.
-- ============================================================================
CREATE TABLE IF NOT EXISTS public.billing_subscriptions (
    id                     uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id              uuid NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    stripe_subscription_id text NOT NULL UNIQUE,
    stripe_customer_id     text NOT NULL,
    plan                   text NOT NULL,
    billing_interval       text NOT NULL CHECK (billing_interval IN ('month', 'year')),
    status                 text NOT NULL DEFAULT 'trialing' CHECK (status IN ('trialing', 'active', 'past_due', 'canceled', 'unpaid')),
    trial_end              timestamptz,
    current_period_start   timestamptz,
    current_period_end     timestamptz,
    canceled_at            timestamptz,
    created_at             timestamptz NOT NULL DEFAULT now(),
    updated_at             timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_billing_subscriptions_tenant ON public.billing_subscriptions (tenant_id);
CREATE INDEX IF NOT EXISTS idx_billing_subscriptions_status ON public.billing_subscriptions (status);

ALTER TABLE public.billing_subscriptions ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.billing_subscriptions FORCE ROW LEVEL SECURITY;

-- RLS policy: tenant can only see their own subscriptions
CREATE POLICY billing_subscriptions_tenant_isolation ON public.billing_subscriptions
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- ============================================================================
-- SECURITY DEFINER functions — checkout session lifecycle
-- (no RLS context available during checkout/registration)
-- ============================================================================

-- Create a checkout session record
CREATE OR REPLACE FUNCTION public.billing_create_checkout_session(
    p_stripe_session_id text,
    p_plan text,
    p_billing_interval text
)
RETURNS uuid
LANGUAGE sql
SECURITY DEFINER
SET search_path = 'public'
AS $$
    INSERT INTO billing_checkout_sessions (stripe_session_id, plan, billing_interval)
    VALUES (p_stripe_session_id, p_plan, p_billing_interval)
    RETURNING id;
$$;

-- Mark a checkout session as completed (Stripe webhook or API verify)
CREATE OR REPLACE FUNCTION public.billing_complete_checkout_session(
    p_stripe_session_id text,
    p_email text
)
RETURNS boolean
LANGUAGE sql
SECURITY DEFINER
SET search_path = 'public'
AS $$
    WITH upd AS (
        UPDATE billing_checkout_sessions
        SET status = 'completed', email = p_email, updated_at = now()
        WHERE stripe_session_id = p_stripe_session_id AND status = 'pending'
        RETURNING id
    )
    SELECT EXISTS(SELECT 1 FROM upd);
$$;

-- Get a checkout session by Stripe session ID
CREATE OR REPLACE FUNCTION public.billing_get_checkout_session(p_stripe_session_id text)
RETURNS TABLE(
    id uuid,
    stripe_session_id text,
    plan text,
    billing_interval text,
    email text,
    status text,
    tenant_id uuid,
    created_at timestamptz
)
LANGUAGE sql
SECURITY DEFINER
SET search_path = 'public'
AS $$
    SELECT cs.id, cs.stripe_session_id, cs.plan, cs.billing_interval,
           cs.email, cs.status, cs.tenant_id, cs.created_at
    FROM billing_checkout_sessions cs
    WHERE cs.stripe_session_id = p_stripe_session_id;
$$;

-- Atomically claim a checkout session (pre-registration, no tenant_id yet).
-- Returns true if claimed, false if already claimed or not completed.
CREATE OR REPLACE FUNCTION public.billing_claim_checkout_session(p_stripe_session_id text)
RETURNS boolean
LANGUAGE sql
SECURITY DEFINER
SET search_path = 'public'
AS $$
    WITH upd AS (
        UPDATE billing_checkout_sessions
        SET status = 'registered', updated_at = now()
        WHERE stripe_session_id = p_stripe_session_id AND status = 'completed'
        RETURNING id
    )
    SELECT EXISTS(SELECT 1 FROM upd);
$$;

-- Set tenant_id on a claimed checkout session (post-registration).
CREATE OR REPLACE FUNCTION public.billing_update_checkout_tenant(
    p_stripe_session_id text,
    p_tenant_id uuid
)
RETURNS void
LANGUAGE sql
SECURITY DEFINER
SET search_path = 'public'
AS $$
    UPDATE billing_checkout_sessions
    SET tenant_id = p_tenant_id, updated_at = now()
    WHERE stripe_session_id = p_stripe_session_id AND status = 'registered';
$$;

-- ============================================================================
-- SECURITY DEFINER functions — billing customers (webhook + registration context)
-- ============================================================================

-- Create a billing customer record (bypasses RLS for registration flow)
CREATE OR REPLACE FUNCTION public.billing_create_customer(
    p_tenant_id uuid,
    p_stripe_customer_id text
)
RETURNS uuid
LANGUAGE sql
SECURITY DEFINER
SET search_path = 'public'
AS $$
    INSERT INTO billing_customers (tenant_id, stripe_customer_id)
    VALUES (p_tenant_id, p_stripe_customer_id)
    ON CONFLICT (tenant_id) DO UPDATE SET stripe_customer_id = EXCLUDED.stripe_customer_id
    RETURNING id;
$$;

-- Look up a billing customer by Stripe customer ID (webhook context, no RLS)
CREATE OR REPLACE FUNCTION public.billing_get_customer_by_stripe_id(p_stripe_customer_id text)
RETURNS TABLE(
    id uuid,
    tenant_id uuid,
    stripe_customer_id text,
    created_at timestamptz
)
LANGUAGE sql
SECURITY DEFINER
SET search_path = 'public'
AS $$
    SELECT bc.id, bc.tenant_id, bc.stripe_customer_id, bc.created_at
    FROM billing_customers bc
    WHERE bc.stripe_customer_id = p_stripe_customer_id;
$$;

-- ============================================================================
-- SECURITY DEFINER functions — billing subscriptions (webhook context)
-- ============================================================================

-- Upsert a subscription (create or update by stripe_subscription_id)
CREATE OR REPLACE FUNCTION public.billing_upsert_subscription(
    p_tenant_id uuid,
    p_stripe_subscription_id text,
    p_stripe_customer_id text,
    p_plan text,
    p_billing_interval text,
    p_status text,
    p_trial_end timestamptz,
    p_current_period_start timestamptz,
    p_current_period_end timestamptz
)
RETURNS uuid
LANGUAGE sql
SECURITY DEFINER
SET search_path = 'public'
AS $$
    INSERT INTO billing_subscriptions
        (tenant_id, stripe_subscription_id, stripe_customer_id, plan, billing_interval, status,
         trial_end, current_period_start, current_period_end)
    VALUES (p_tenant_id, p_stripe_subscription_id, p_stripe_customer_id, p_plan, p_billing_interval, p_status,
            p_trial_end, p_current_period_start, p_current_period_end)
    ON CONFLICT (stripe_subscription_id) DO UPDATE SET
        status = EXCLUDED.status,
        current_period_start = COALESCE(EXCLUDED.current_period_start, billing_subscriptions.current_period_start),
        current_period_end = COALESCE(EXCLUDED.current_period_end, billing_subscriptions.current_period_end),
        updated_at = now()
    RETURNING id;
$$;

-- Update subscription status by Stripe subscription ID (webhook context)
CREATE OR REPLACE FUNCTION public.billing_update_subscription_by_stripe_id(
    p_stripe_sub_id text,
    p_status text,
    p_period_start timestamptz,
    p_period_end timestamptz,
    p_canceled_at timestamptz
)
RETURNS boolean
LANGUAGE sql
SECURITY DEFINER
SET search_path = 'public'
AS $$
    WITH upd AS (
        UPDATE billing_subscriptions
        SET status = p_status,
            current_period_start = COALESCE(p_period_start, current_period_start),
            current_period_end = COALESCE(p_period_end, current_period_end),
            canceled_at = COALESCE(p_canceled_at, canceled_at),
            updated_at = now()
        WHERE stripe_subscription_id = p_stripe_sub_id
        RETURNING id
    )
    SELECT EXISTS(SELECT 1 FROM upd);
$$;

-- ============================================================================
-- SECURITY DEFINER function — tenant plan sync (webhook context)
-- ============================================================================

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
    SET settings = COALESCE(settings, '{}'::jsonb) || p_settings,
        updated_at = now()
    WHERE id = p_tenant_id;
$$;

-- ============================================================================
-- Grant permissions to app roles
-- ============================================================================
DO $$
BEGIN
  -- openoms (self-hosted app role)
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
    EXECUTE 'GRANT ALL ON TABLE public.billing_checkout_sessions TO openoms';
    EXECUTE 'GRANT ALL ON TABLE public.billing_customers TO openoms';
    EXECUTE 'GRANT ALL ON TABLE public.billing_subscriptions TO openoms';
  END IF;

  -- openoms_app (Supabase app role)
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
    EXECUTE 'GRANT ALL ON TABLE public.billing_checkout_sessions TO openoms_app';
    EXECUTE 'GRANT ALL ON TABLE public.billing_customers TO openoms_app';
    EXECUTE 'GRANT ALL ON TABLE public.billing_subscriptions TO openoms_app';
  END IF;

  -- openoms_auth (auth role, used during registration)
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
