-- Revert: restore function ownership to postgres and drop openoms_auth role

-- 1. Restore ownership to postgres
ALTER FUNCTION public.find_tenant_by_slug(text) OWNER TO postgres;
ALTER FUNCTION public.find_user_for_auth(text, uuid) OWNER TO postgres;
ALTER FUNCTION public.find_invitation_by_token(text) OWNER TO postgres;
ALTER FUNCTION public.find_return_by_token(text) OWNER TO postgres;
ALTER FUNCTION public.find_order_tenant_id(uuid) OWNER TO postgres;
ALTER FUNCTION public.use_invitation(text) OWNER TO postgres;

-- 2. Restore execute to PUBLIC
GRANT EXECUTE ON FUNCTION public.find_tenant_by_slug(text) TO PUBLIC;
GRANT EXECUTE ON FUNCTION public.find_user_for_auth(text, uuid) TO PUBLIC;
GRANT EXECUTE ON FUNCTION public.find_invitation_by_token(text) TO PUBLIC;
GRANT EXECUTE ON FUNCTION public.find_return_by_token(text) TO PUBLIC;
GRANT EXECUTE ON FUNCTION public.find_order_tenant_id(uuid) TO PUBLIC;
GRANT EXECUTE ON FUNCTION public.use_invitation(text) TO PUBLIC;

-- 3. Drop role-specific RLS policies
DROP POLICY IF EXISTS auth_role_tenants ON public.tenants;
DROP POLICY IF EXISTS auth_role_users ON public.users;
DROP POLICY IF EXISTS auth_role_invitations_select ON public.invitations;
DROP POLICY IF EXISTS auth_role_invitations_update ON public.invitations;
DROP POLICY IF EXISTS auth_role_returns ON public.returns;
DROP POLICY IF EXISTS auth_role_orders ON public.orders;

-- 4. Revoke grants and drop role
REVOKE ALL ON ALL TABLES IN SCHEMA public FROM openoms_auth;
REVOKE USAGE ON SCHEMA public FROM openoms_auth;
DROP ROLE IF EXISTS openoms_auth;
