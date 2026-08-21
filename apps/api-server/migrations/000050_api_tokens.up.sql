-- Long-lived owner API tokens. Hash at rest (token_hash), never plaintext.
-- Lookup before tenant context uses SECURITY DEFINER find_api_token_for_auth,
-- the same pattern as find_user_for_auth.

CREATE TABLE public.api_tokens (
    id           uuid PRIMARY KEY DEFAULT public.uuid_generate_v4(),
    tenant_id    uuid NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    user_id      uuid NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    name         text NOT NULL,
    token_hash   text NOT NULL,
    last_used_at timestamptz,
    revoked_at   timestamptz,
    created_at   timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX uq_api_tokens_hash ON public.api_tokens (token_hash); -- migrate:index-lock-ok
CREATE INDEX idx_api_tokens_user ON public.api_tokens (tenant_id, user_id) WHERE revoked_at IS NULL; -- migrate:index-lock-ok

ALTER TABLE public.api_tokens ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.api_tokens FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON public.api_tokens USING ((tenant_id = (current_setting('app.current_tenant_id'::text, true))::uuid));

DO $$
DECLARE
    app_role text;
BEGIN
    FOREACH app_role IN ARRAY ARRAY['openoms_app', 'openoms'] LOOP
        IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = app_role) THEN
            EXECUTE format('GRANT SELECT, INSERT, UPDATE, DELETE ON public.api_tokens TO %I', app_role);
        END IF;
    END LOOP;
END;
$$;

-- Cross-tenant auth lookup: openoms_auth can SELECT every row (like users/tenants).
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'openoms_auth') THEN
        GRANT SELECT ON public.api_tokens TO openoms_auth;
        CREATE POLICY auth_role_api_tokens ON public.api_tokens FOR SELECT TO openoms_auth USING (true);
    END IF;
END;
$$;

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'pg_database_owner') THEN
    EXECUTE 'SET ROLE pg_database_owner';
    GRANT CREATE ON SCHEMA public TO openoms_auth;
    RESET ROLE;
  ELSE
    GRANT CREATE ON SCHEMA public TO openoms_auth;
  END IF;
END;
$$;

SET ROLE openoms_auth;

CREATE FUNCTION public.find_api_token_for_auth(p_token_hash text)
 RETURNS TABLE(id uuid, tenant_id uuid, user_id uuid, name text, token_hash text, last_used_at timestamptz, revoked_at timestamptz, created_at timestamptz)
 LANGUAGE sql STABLE SECURITY DEFINER
 SET search_path TO 'public'
AS $$
    SELECT t.id, t.tenant_id, t.user_id, t.name, t.token_hash, t.last_used_at, t.revoked_at, t.created_at
    FROM api_tokens t
    WHERE t.token_hash = p_token_hash
      AND t.revoked_at IS NULL;
$$;

RESET ROLE;

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'pg_database_owner') THEN
    EXECUTE 'SET ROLE pg_database_owner';
    REVOKE CREATE ON SCHEMA public FROM openoms_auth;
    RESET ROLE;
  ELSE
    REVOKE CREATE ON SCHEMA public FROM openoms_auth;
  END IF;
END;
$$;

REVOKE EXECUTE ON FUNCTION public.find_api_token_for_auth(text) FROM PUBLIC;

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'openoms_app') THEN
    GRANT EXECUTE ON FUNCTION public.find_api_token_for_auth(text) TO openoms_app;
  END IF;
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'openoms') THEN
    GRANT EXECUTE ON FUNCTION public.find_api_token_for_auth(text) TO openoms;
  END IF;
END;
$$;
