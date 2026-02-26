-- Track used license tokens to prevent replay attacks.
-- Each JTI (JWT ID) can only be used once.
CREATE TABLE public.used_license_tokens (
    jti       uuid PRIMARY KEY,
    tenant_id uuid NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    email     text NOT NULL,
    plan      text NOT NULL,
    used_at   timestamptz NOT NULL DEFAULT now()
);

-- Index for auditing: find all tokens used by a tenant
CREATE INDEX idx_used_license_tokens_tenant ON public.used_license_tokens (tenant_id);

-- SECURITY DEFINER: check if a license token JTI has been used.
-- Needed because registration happens before tenant context is set (no RLS).
CREATE OR REPLACE FUNCTION public.check_license_token_used(p_jti uuid)
RETURNS boolean
LANGUAGE sql
SECURITY DEFINER
SET search_path = 'public'
AS $$
    SELECT EXISTS(SELECT 1 FROM used_license_tokens WHERE jti = p_jti);
$$;

-- SECURITY DEFINER: mark a license token JTI as used after successful registration.
CREATE OR REPLACE FUNCTION public.mark_license_token_used(
    p_jti uuid,
    p_tenant_id uuid,
    p_email text,
    p_plan text
)
RETURNS void
LANGUAGE sql
SECURITY DEFINER
SET search_path = 'public'
AS $$
    INSERT INTO used_license_tokens (jti, tenant_id, email, plan)
    VALUES (p_jti, p_tenant_id, p_email, p_plan)
    ON CONFLICT (jti) DO NOTHING;
$$;

-- Grant minimal permissions to the auth role (used during registration)
GRANT EXECUTE ON FUNCTION public.check_license_token_used(uuid) TO openoms_auth;
GRANT EXECUTE ON FUNCTION public.mark_license_token_used(uuid, uuid, text, text) TO openoms_auth;
