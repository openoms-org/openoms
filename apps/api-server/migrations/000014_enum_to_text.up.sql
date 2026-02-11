-- Migration 000014: Convert all remaining ENUMs to TEXT
-- order_status was already converted in migration 000006

-- tenants.plan
ALTER TABLE tenants ALTER COLUMN plan TYPE TEXT USING plan::TEXT;
ALTER TABLE tenants ALTER COLUMN plan SET DEFAULT 'free';
-- Recreate find_tenant_by_slug: the old version returns plan_type enum which we
-- are about to drop. PG cannot DROP TYPE with dependents, so rebuild with TEXT.
DROP FUNCTION IF EXISTS public.find_tenant_by_slug(TEXT);
CREATE OR REPLACE FUNCTION public.find_tenant_by_slug(p_slug TEXT)
RETURNS TABLE(id UUID, name VARCHAR, slug VARCHAR, plan TEXT, settings JSONB, created_at TIMESTAMPTZ, updated_at TIMESTAMPTZ)
SECURITY DEFINER
SET search_path = public
AS $$
    SELECT t.id, t.name, t.slug, t.plan, t.settings, t.created_at, t.updated_at
    FROM tenants t
    WHERE t.slug = p_slug;
$$ LANGUAGE sql STABLE;
GRANT EXECUTE ON FUNCTION public.find_tenant_by_slug(TEXT) TO openoms_app;
DROP TYPE IF EXISTS plan_type;

-- users.role
ALTER TABLE users ALTER COLUMN role TYPE TEXT USING role::TEXT;
ALTER TABLE users ALTER COLUMN role SET DEFAULT 'member';
-- Recreate find_user_for_auth: the old version returns user_role enum which we
-- are about to drop. PG cannot DROP TYPE with dependents, so rebuild with TEXT.
DROP FUNCTION IF EXISTS public.find_user_for_auth(TEXT, UUID);
CREATE OR REPLACE FUNCTION public.find_user_for_auth(p_email TEXT, p_tenant_id UUID)
RETURNS TABLE(id UUID, tenant_id UUID, email VARCHAR, name VARCHAR, password_hash VARCHAR, role TEXT, created_at TIMESTAMPTZ, updated_at TIMESTAMPTZ)
SECURITY DEFINER
SET search_path = public
AS $$
    SELECT u.id, u.tenant_id, u.email, u.name, u.password_hash, u.role, u.created_at, u.updated_at
    FROM users u
    WHERE u.email = p_email AND u.tenant_id = p_tenant_id;
$$ LANGUAGE sql STABLE;
GRANT EXECUTE ON FUNCTION public.find_user_for_auth(TEXT, UUID) TO openoms_app;
DROP TYPE IF EXISTS user_role;

-- integrations.provider
ALTER TABLE integrations ALTER COLUMN provider TYPE TEXT USING provider::TEXT;

-- integrations.status
ALTER TABLE integrations ALTER COLUMN status TYPE TEXT USING status::TEXT;
ALTER TABLE integrations ALTER COLUMN status SET DEFAULT 'inactive';
DROP TYPE IF EXISTS integration_status;

-- orders.source
ALTER TABLE orders ALTER COLUMN source TYPE TEXT USING source::TEXT;
ALTER TABLE orders ALTER COLUMN source SET DEFAULT 'manual';

-- products.source
ALTER TABLE products ALTER COLUMN source TYPE TEXT USING source::TEXT;
ALTER TABLE products ALTER COLUMN source SET DEFAULT 'manual';

DROP TYPE IF EXISTS order_source;

-- webhook_events.provider
ALTER TABLE webhook_events ALTER COLUMN provider TYPE TEXT USING provider::TEXT;

DROP TYPE IF EXISTS integration_provider;

-- webhook_events.status
ALTER TABLE webhook_events ALTER COLUMN status TYPE TEXT USING status::TEXT;
ALTER TABLE webhook_events ALTER COLUMN status SET DEFAULT 'received';
DROP TYPE IF EXISTS webhook_status;

-- shipments.provider
ALTER TABLE shipments ALTER COLUMN provider TYPE TEXT USING provider::TEXT;

-- shipments.status
ALTER TABLE shipments ALTER COLUMN status TYPE TEXT USING status::TEXT;
ALTER TABLE shipments ALTER COLUMN status SET DEFAULT 'created';

DROP TYPE IF EXISTS shipment_provider;
DROP TYPE IF EXISTS shipment_status;
