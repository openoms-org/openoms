-- Migration 000014: Convert all remaining ENUMs to TEXT
-- order_status was already converted in migration 000006

-- tenants.plan
ALTER TABLE tenants ALTER COLUMN plan TYPE TEXT USING plan::TEXT;
ALTER TABLE tenants ALTER COLUMN plan SET DEFAULT 'free';
DROP TYPE IF EXISTS plan_type;

-- users.role
ALTER TABLE users ALTER COLUMN role TYPE TEXT USING role::TEXT;
ALTER TABLE users ALTER COLUMN role SET DEFAULT 'member';
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
