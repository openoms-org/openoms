-- OPE-332: avoid exposing tenant settings through the login slug lookup.
--
-- Keep the existing result shape for zero-downtime deploy compatibility with
-- already-running pods, but return a redacted JSON object instead of
-- tenants.settings. Full settings must be read through tenant-scoped paths.

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

CREATE OR REPLACE FUNCTION public.find_tenant_by_slug(p_slug text)
 RETURNS TABLE(id uuid, name character varying, slug character varying, plan text, settings jsonb, created_at timestamp with time zone, updated_at timestamp with time zone)
 LANGUAGE sql STABLE SECURITY DEFINER
 SET search_path TO 'public'
AS $$
    SELECT t.id, t.name, t.slug, t.plan, '{}'::jsonb AS settings, t.created_at, t.updated_at
    FROM tenants t
    WHERE t.slug = p_slug;
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

REVOKE EXECUTE ON FUNCTION public.find_tenant_by_slug(text) FROM PUBLIC;

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'openoms_app') THEN
    GRANT EXECUTE ON FUNCTION public.find_tenant_by_slug(text) TO openoms_app;
  END IF;
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'openoms') THEN
    GRANT EXECUTE ON FUNCTION public.find_tenant_by_slug(text) TO openoms;
  END IF;
END;
$$;
