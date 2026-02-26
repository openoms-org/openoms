-- Track used license tokens to prevent replay attacks.
-- Each JTI (JWT ID) can only be used once.
CREATE TABLE IF NOT EXISTS public.used_license_tokens (
    jti       uuid PRIMARY KEY,
    tenant_id uuid REFERENCES public.tenants(id) ON DELETE CASCADE,
    email     text NOT NULL,
    plan      text NOT NULL,
    used_at   timestamptz NOT NULL DEFAULT now()
);

-- Index for auditing: find all tokens used by a tenant
CREATE INDEX IF NOT EXISTS idx_used_license_tokens_tenant ON public.used_license_tokens (tenant_id);

-- RLS: defense-in-depth — direct access denied; all access via SECURITY DEFINER functions.
ALTER TABLE public.used_license_tokens ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.used_license_tokens FORCE ROW LEVEL SECURITY;

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

-- SECURITY DEFINER: atomically claim a license token JTI before registration.
-- tenant_id is initially NULL (tenant doesn't exist yet) and set after registration.
-- Returns true if claimed (inserted), false if already used (conflict on PK).
CREATE OR REPLACE FUNCTION public.mark_license_token_used(
    p_jti uuid,
    p_tenant_id uuid,
    p_email text,
    p_plan text
)
RETURNS boolean
LANGUAGE sql
SECURITY DEFINER
SET search_path = 'public'
AS $$
    WITH ins AS (
        INSERT INTO used_license_tokens (jti, tenant_id, email, plan)
        VALUES (p_jti, p_tenant_id, p_email, p_plan)
        ON CONFLICT (jti) DO NOTHING
        RETURNING jti
    )
    SELECT EXISTS(SELECT 1 FROM ins);
$$;

-- SECURITY DEFINER: update claimed token with actual tenant_id after registration.
CREATE OR REPLACE FUNCTION public.update_license_token_tenant(p_jti uuid, p_tenant_id uuid)
RETURNS void
LANGUAGE sql
SECURITY DEFINER
SET search_path = 'public'
AS $$
    UPDATE used_license_tokens SET tenant_id = p_tenant_id WHERE jti = p_jti;
$$;

-- Grant minimal permissions to the auth role (used during registration)
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'openoms_auth') THEN
    EXECUTE 'GRANT EXECUTE ON FUNCTION public.check_license_token_used(uuid) TO openoms_auth';
    EXECUTE 'GRANT EXECUTE ON FUNCTION public.mark_license_token_used(uuid, uuid, text, text) TO openoms_auth';
    EXECUTE 'GRANT EXECUTE ON FUNCTION public.update_license_token_tenant(uuid, uuid) TO openoms_auth';
  END IF;
END;
$$;
