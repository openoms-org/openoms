-- Create a least-privilege role for SECURITY DEFINER auth functions.
-- Instead of running as postgres (superuser), these functions will run as
-- openoms_auth which has only the minimal SELECT/UPDATE grants needed.
-- This reduces the blast radius: a SQL injection in these functions can only
-- read the 5 specific tables, not the entire database.

-- 1. Create the dedicated role (NOLOGIN — only used via function ownership)
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'openoms_auth') THEN
    CREATE ROLE openoms_auth NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOBYPASSRLS;
  END IF;
END;
$$;

-- 2. Grant role membership so postgres can SET ROLE (needed for Supabase where
--    postgres is not a true superuser and ALTER FUNCTION OWNER is restricted)
GRANT openoms_auth TO postgres;

-- 3. Grant schema usage + CREATE (needed to create functions as openoms_auth)
DO $$
BEGIN
  -- On Supabase, public schema is owned by pg_database_owner
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'pg_database_owner') THEN
    EXECUTE 'SET ROLE pg_database_owner';
    GRANT CREATE ON SCHEMA public TO openoms_auth;
    RESET ROLE;
  ELSE
    GRANT CREATE ON SCHEMA public TO openoms_auth;
  END IF;
END;
$$;
GRANT USAGE ON SCHEMA public TO openoms_auth;

-- 4. Grant minimal table-level permissions
GRANT SELECT ON public.tenants TO openoms_auth;
GRANT SELECT ON public.users TO openoms_auth;
GRANT SELECT, UPDATE ON public.invitations TO openoms_auth;
GRANT SELECT ON public.returns TO openoms_auth;
GRANT SELECT ON public.orders TO openoms_auth;

-- 5. Add role-specific RLS policies so openoms_auth can see all rows
-- (these functions need cross-tenant access by design)
CREATE POLICY auth_role_tenants ON public.tenants FOR SELECT TO openoms_auth USING (true);
CREATE POLICY auth_role_users ON public.users FOR SELECT TO openoms_auth USING (true);
CREATE POLICY auth_role_invitations_select ON public.invitations FOR SELECT TO openoms_auth USING (true);
CREATE POLICY auth_role_invitations_update ON public.invitations FOR UPDATE TO openoms_auth USING (true);
CREATE POLICY auth_role_returns ON public.returns FOR SELECT TO openoms_auth USING (true);
CREATE POLICY auth_role_orders ON public.orders FOR SELECT TO openoms_auth USING (true);

-- 6. Drop functions (owned by postgres) and recreate as openoms_auth
--    Using DROP+CREATE+SET ROLE because ALTER FUNCTION OWNER TO is blocked
--    on Supabase's managed postgres (not a true superuser).
DROP FUNCTION IF EXISTS public.find_tenant_by_slug(text);
DROP FUNCTION IF EXISTS public.find_user_for_auth(text, uuid);
DROP FUNCTION IF EXISTS public.find_invitation_by_token(text);
DROP FUNCTION IF EXISTS public.find_return_by_token(text);
DROP FUNCTION IF EXISTS public.find_order_tenant_id(uuid);
DROP FUNCTION IF EXISTS public.use_invitation(text);

SET ROLE openoms_auth;

CREATE FUNCTION public.find_tenant_by_slug(p_slug text)
 RETURNS TABLE(id uuid, name character varying, slug character varying, plan text, settings jsonb, created_at timestamp with time zone, updated_at timestamp with time zone)
 LANGUAGE sql STABLE SECURITY DEFINER
 SET search_path TO 'public'
AS $$
    SELECT t.id, t.name, t.slug, t.plan, t.settings, t.created_at, t.updated_at
    FROM tenants t
    WHERE t.slug = p_slug;
$$;

CREATE FUNCTION public.find_user_for_auth(p_email text, p_tenant_id uuid)
 RETURNS TABLE(id uuid, tenant_id uuid, email text, name text, password_hash text, role text, role_id uuid, created_at timestamp with time zone, updated_at timestamp with time zone, totp_secret text, totp_enabled boolean)
 LANGUAGE sql SECURITY DEFINER
 SET search_path TO 'public'
AS $$
    SELECT u.id, u.tenant_id, u.email, u.name, u.password_hash,
           u.role, u.role_id, u.created_at, u.updated_at,
           u.totp_secret, u.totp_enabled
    FROM users u
    WHERE u.email = p_email AND u.tenant_id = p_tenant_id;
$$;

CREATE FUNCTION public.find_invitation_by_token(p_token text)
 RETURNS TABLE(id uuid, tenant_id uuid, email text, role text, expires_at timestamp with time zone, used_at timestamp with time zone)
 LANGUAGE sql SECURITY DEFINER
 SET search_path TO 'public'
AS $$
    SELECT i.id, i.tenant_id, i.email, i.role, i.expires_at, i.used_at
    FROM invitations i
    WHERE i.token = p_token
    LIMIT 1;
$$;

CREATE FUNCTION public.find_return_by_token(p_token text)
 RETURNS TABLE(id uuid, tenant_id uuid, order_id uuid, status character varying, reason text, items jsonb, refund_amount numeric, notes text, return_token text, customer_email text, customer_notes text, created_at timestamp with time zone, updated_at timestamp with time zone)
 LANGUAGE sql STABLE SECURITY DEFINER
 SET search_path TO 'public'
AS $$
    SELECT r.id, r.tenant_id, r.order_id, r.status,
           r.reason, r.items, r.refund_amount,
           r.notes, r.return_token, r.customer_email,
           r.customer_notes, r.created_at, r.updated_at
    FROM returns r
    WHERE r.return_token = p_token;
$$;

CREATE FUNCTION public.find_order_tenant_id(p_order_id uuid)
 RETURNS TABLE(tenant_id uuid, customer_email text)
 LANGUAGE sql STABLE SECURITY DEFINER
 SET search_path TO 'public'
AS $$
    SELECT o.tenant_id, o.customer_email
    FROM orders o
    WHERE o.id = p_order_id;
$$;

CREATE FUNCTION public.use_invitation(p_token text)
 RETURNS void
 LANGUAGE sql SECURITY DEFINER
 SET search_path TO 'public'
AS $$
    UPDATE invitations SET used_at = now() WHERE token = p_token AND used_at IS NULL;
$$;

RESET ROLE;

-- 7. Lock down execution: only openoms_app can call these functions
REVOKE EXECUTE ON FUNCTION public.find_tenant_by_slug(text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION public.find_tenant_by_slug(text) TO openoms_app;

REVOKE EXECUTE ON FUNCTION public.find_user_for_auth(text, uuid) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION public.find_user_for_auth(text, uuid) TO openoms_app;

REVOKE EXECUTE ON FUNCTION public.find_invitation_by_token(text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION public.find_invitation_by_token(text) TO openoms_app;

REVOKE EXECUTE ON FUNCTION public.find_return_by_token(text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION public.find_return_by_token(text) TO openoms_app;

REVOKE EXECUTE ON FUNCTION public.find_order_tenant_id(uuid) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION public.find_order_tenant_id(uuid) TO openoms_app;

REVOKE EXECUTE ON FUNCTION public.use_invitation(text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION public.use_invitation(text) TO openoms_app;

-- 8. Revoke CREATE on schema from openoms_auth (was only needed for function creation)
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
