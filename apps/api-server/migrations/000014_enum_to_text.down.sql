-- Migration 000014 down: Restore ENUM types

CREATE TYPE plan_type AS ENUM ('free', 'lite', 'pro', 'enterprise');
ALTER TABLE tenants ALTER COLUMN plan TYPE plan_type USING plan::plan_type;
ALTER TABLE tenants ALTER COLUMN plan SET DEFAULT 'free';

CREATE TYPE user_role AS ENUM ('owner', 'admin', 'member');
ALTER TABLE users ALTER COLUMN role TYPE user_role USING role::user_role;
ALTER TABLE users ALTER COLUMN role SET DEFAULT 'member';

CREATE TYPE integration_provider AS ENUM ('allegro', 'inpost', 'dhl', 'dpd', 'woocommerce');
ALTER TABLE integrations ALTER COLUMN provider TYPE integration_provider USING provider::integration_provider;
ALTER TABLE webhook_events ALTER COLUMN provider TYPE integration_provider USING provider::integration_provider;

CREATE TYPE integration_status AS ENUM ('active', 'inactive', 'error');
ALTER TABLE integrations ALTER COLUMN status TYPE integration_status USING status::integration_status;
ALTER TABLE integrations ALTER COLUMN status SET DEFAULT 'inactive';

CREATE TYPE order_source AS ENUM ('allegro', 'woocommerce', 'manual');
ALTER TABLE orders ALTER COLUMN source TYPE order_source USING source::order_source;
ALTER TABLE orders ALTER COLUMN source SET DEFAULT 'manual';
ALTER TABLE products ALTER COLUMN source TYPE order_source USING source::order_source;
ALTER TABLE products ALTER COLUMN source SET DEFAULT 'manual';

CREATE TYPE webhook_status AS ENUM ('received', 'processing', 'processed', 'failed');
ALTER TABLE webhook_events ALTER COLUMN status TYPE webhook_status USING status::webhook_status;
ALTER TABLE webhook_events ALTER COLUMN status SET DEFAULT 'received';

CREATE TYPE shipment_provider AS ENUM ('inpost', 'dhl', 'dpd', 'manual');
ALTER TABLE shipments ALTER COLUMN provider TYPE shipment_provider USING provider::shipment_provider;

CREATE TYPE shipment_status AS ENUM (
    'created', 'label_ready', 'picked_up', 'in_transit',
    'out_for_delivery', 'delivered', 'returned', 'failed'
);
ALTER TABLE shipments ALTER COLUMN status TYPE shipment_status USING status::shipment_status;
ALTER TABLE shipments ALTER COLUMN status SET DEFAULT 'created';
