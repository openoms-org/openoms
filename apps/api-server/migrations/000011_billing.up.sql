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
-- SECURITY DEFINER functions for pre-registration checkout operations
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

-- Atomically claim a checkout session for registration.
-- Returns true if claimed (status changed to 'registered'), false if already claimed.
CREATE OR REPLACE FUNCTION public.billing_claim_checkout_session(
    p_stripe_session_id text,
    p_tenant_id uuid
)
RETURNS boolean
LANGUAGE sql
SECURITY DEFINER
SET search_path = 'public'
AS $$
    WITH upd AS (
        UPDATE billing_checkout_sessions
        SET status = 'registered', tenant_id = p_tenant_id, updated_at = now()
        WHERE stripe_session_id = p_stripe_session_id AND status = 'completed'
        RETURNING id
    )
    SELECT EXISTS(SELECT 1 FROM upd);
$$;

-- ============================================================================
-- Grant permissions to app roles (self-hosted: openoms, Supabase: openoms_app, auth: openoms_auth)
-- ============================================================================
DO $$
BEGIN
  -- openoms (self-hosted app role)
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'openoms') THEN
    EXECUTE 'GRANT EXECUTE ON FUNCTION public.billing_create_checkout_session(text, text, text) TO openoms';
    EXECUTE 'GRANT EXECUTE ON FUNCTION public.billing_complete_checkout_session(text, text) TO openoms';
    EXECUTE 'GRANT EXECUTE ON FUNCTION public.billing_get_checkout_session(text) TO openoms';
    EXECUTE 'GRANT EXECUTE ON FUNCTION public.billing_claim_checkout_session(text, uuid) TO openoms';
    EXECUTE 'GRANT ALL ON TABLE public.billing_checkout_sessions TO openoms';
    EXECUTE 'GRANT ALL ON TABLE public.billing_customers TO openoms';
    EXECUTE 'GRANT ALL ON TABLE public.billing_subscriptions TO openoms';
  END IF;

  -- openoms_app (Supabase app role)
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'openoms_app') THEN
    EXECUTE 'GRANT EXECUTE ON FUNCTION public.billing_create_checkout_session(text, text, text) TO openoms_app';
    EXECUTE 'GRANT EXECUTE ON FUNCTION public.billing_complete_checkout_session(text, text) TO openoms_app';
    EXECUTE 'GRANT EXECUTE ON FUNCTION public.billing_get_checkout_session(text) TO openoms_app';
    EXECUTE 'GRANT EXECUTE ON FUNCTION public.billing_claim_checkout_session(text, uuid) TO openoms_app';
    EXECUTE 'GRANT ALL ON TABLE public.billing_checkout_sessions TO openoms_app';
    EXECUTE 'GRANT ALL ON TABLE public.billing_customers TO openoms_app';
    EXECUTE 'GRANT ALL ON TABLE public.billing_subscriptions TO openoms_app';
  END IF;

  -- openoms_auth (auth role, used during registration)
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'openoms_auth') THEN
    EXECUTE 'GRANT EXECUTE ON FUNCTION public.billing_create_checkout_session(text, text, text) TO openoms_auth';
    EXECUTE 'GRANT EXECUTE ON FUNCTION public.billing_complete_checkout_session(text, text) TO openoms_auth';
    EXECUTE 'GRANT EXECUTE ON FUNCTION public.billing_get_checkout_session(text) TO openoms_auth';
    EXECUTE 'GRANT EXECUTE ON FUNCTION public.billing_claim_checkout_session(text, uuid) TO openoms_auth';
  END IF;
END;
$$;
