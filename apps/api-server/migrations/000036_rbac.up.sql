CREATE TABLE roles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT,
    is_system BOOLEAN NOT NULL DEFAULT false,
    permissions TEXT[] NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, name)
);

ALTER TABLE roles ENABLE ROW LEVEL SECURITY;
ALTER TABLE roles FORCE ROW LEVEL SECURITY;
CREATE POLICY roles_tenant_isolation ON roles
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE INDEX idx_roles_tenant ON roles(tenant_id);
GRANT SELECT, INSERT, UPDATE, DELETE ON roles TO openoms_app;
CREATE TRIGGER update_roles_updated_at BEFORE UPDATE ON roles FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- Add role_id to users (nullable — falls back to legacy role field)
ALTER TABLE users ADD COLUMN role_id UUID REFERENCES roles(id);

-- Recreate find_user_for_auth to include the new role_id column.
-- The previous version (updated in migration 000014) does not include role_id
-- because the column didn't exist yet. The return type changes (new column), so
-- we must DROP first — CREATE OR REPLACE cannot change the return type.
DROP FUNCTION IF EXISTS public.find_user_for_auth(TEXT, UUID);
CREATE FUNCTION public.find_user_for_auth(p_email TEXT, p_tenant_id UUID)
RETURNS TABLE(id UUID, tenant_id UUID, email VARCHAR, name VARCHAR, password_hash VARCHAR, role TEXT, role_id UUID, created_at TIMESTAMPTZ, updated_at TIMESTAMPTZ)
SECURITY DEFINER
SET search_path = public
AS $$
    SELECT u.id, u.tenant_id, u.email, u.name, u.password_hash, u.role, u.role_id, u.created_at, u.updated_at
    FROM users u
    WHERE u.email = p_email AND u.tenant_id = p_tenant_id;
$$ LANGUAGE sql STABLE;

GRANT EXECUTE ON FUNCTION public.find_user_for_auth(TEXT, UUID) TO openoms_app;
