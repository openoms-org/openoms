-- OpenOMS — Auth helper functions
-- SECURITY DEFINER functions bypass RLS for authentication lookups.
-- These run as the function owner (openoms superuser), not the caller (openoms_app).

CREATE OR REPLACE FUNCTION public.find_tenant_by_slug(p_slug TEXT)
RETURNS TABLE(id UUID, name VARCHAR, slug VARCHAR, plan plan_type, settings JSONB, created_at TIMESTAMPTZ, updated_at TIMESTAMPTZ)
SECURITY DEFINER
SET search_path = public
AS $$
    SELECT t.id, t.name, t.slug, t.plan, t.settings, t.created_at, t.updated_at
    FROM tenants t
    WHERE t.slug = p_slug;
$$ LANGUAGE sql STABLE;

CREATE OR REPLACE FUNCTION public.find_user_for_auth(p_email TEXT, p_tenant_id UUID)
RETURNS TABLE(id UUID, tenant_id UUID, email VARCHAR, name VARCHAR, password_hash VARCHAR, role user_role, created_at TIMESTAMPTZ, updated_at TIMESTAMPTZ)
SECURITY DEFINER
SET search_path = public
AS $$
    SELECT u.id, u.tenant_id, u.email, u.name, u.password_hash, u.role, u.created_at, u.updated_at
    FROM users u
    WHERE u.email = p_email AND u.tenant_id = p_tenant_id;
$$ LANGUAGE sql STABLE;

GRANT EXECUTE ON FUNCTION public.find_tenant_by_slug(TEXT) TO openoms_app;
GRANT EXECUTE ON FUNCTION public.find_user_for_auth(TEXT, UUID) TO openoms_app;
