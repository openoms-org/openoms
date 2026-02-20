-- Revert: restore function ownership to postgres and drop openoms_auth role

-- 1. Drop functions owned by openoms_auth
DROP FUNCTION IF EXISTS public.find_tenant_by_slug(text);
DROP FUNCTION IF EXISTS public.find_user_for_auth(text, uuid);
DROP FUNCTION IF EXISTS public.find_invitation_by_token(text);
DROP FUNCTION IF EXISTS public.find_return_by_token(text);
DROP FUNCTION IF EXISTS public.find_order_tenant_id(uuid);
DROP FUNCTION IF EXISTS public.use_invitation(text);

-- 2. Recreate as postgres (default owner)
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

-- 3. Drop role-specific RLS policies
DROP POLICY IF EXISTS auth_role_tenants ON public.tenants;
DROP POLICY IF EXISTS auth_role_users ON public.users;
DROP POLICY IF EXISTS auth_role_invitations_select ON public.invitations;
DROP POLICY IF EXISTS auth_role_invitations_update ON public.invitations;
DROP POLICY IF EXISTS auth_role_returns ON public.returns;
DROP POLICY IF EXISTS auth_role_orders ON public.orders;

-- 4. Revoke grants and drop role
REVOKE ALL ON ALL TABLES IN SCHEMA public FROM openoms_auth;
REVOKE ALL ON SCHEMA public FROM openoms_auth;
REVOKE openoms_auth FROM postgres;
DROP ROLE IF EXISTS openoms_auth;
