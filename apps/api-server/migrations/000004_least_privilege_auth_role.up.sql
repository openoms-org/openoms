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

-- 2. Grant schema usage
GRANT USAGE ON SCHEMA public TO openoms_auth;

-- 3. Grant minimal table-level permissions
GRANT SELECT ON public.tenants TO openoms_auth;
GRANT SELECT ON public.users TO openoms_auth;
GRANT SELECT, UPDATE ON public.invitations TO openoms_auth;
GRANT SELECT ON public.returns TO openoms_auth;
GRANT SELECT ON public.orders TO openoms_auth;

-- 4. Add role-specific RLS policies so openoms_auth can see all rows
-- (these functions need cross-tenant access by design)
CREATE POLICY auth_role_tenants ON public.tenants FOR SELECT TO openoms_auth USING (true);
CREATE POLICY auth_role_users ON public.users FOR SELECT TO openoms_auth USING (true);
CREATE POLICY auth_role_invitations_select ON public.invitations FOR SELECT TO openoms_auth USING (true);
CREATE POLICY auth_role_invitations_update ON public.invitations FOR UPDATE TO openoms_auth USING (true);
CREATE POLICY auth_role_returns ON public.returns FOR SELECT TO openoms_auth USING (true);
CREATE POLICY auth_role_orders ON public.orders FOR SELECT TO openoms_auth USING (true);

-- 5. Transfer ownership (preserves existing function definitions exactly)
ALTER FUNCTION public.find_tenant_by_slug(text) OWNER TO openoms_auth;
ALTER FUNCTION public.find_user_for_auth(text, uuid) OWNER TO openoms_auth;
ALTER FUNCTION public.find_invitation_by_token(text) OWNER TO openoms_auth;
ALTER FUNCTION public.find_return_by_token(text) OWNER TO openoms_auth;
ALTER FUNCTION public.find_order_tenant_id(uuid) OWNER TO openoms_auth;
ALTER FUNCTION public.use_invitation(text) OWNER TO openoms_auth;

-- 6. Lock down execution: only openoms_app can call these functions
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
