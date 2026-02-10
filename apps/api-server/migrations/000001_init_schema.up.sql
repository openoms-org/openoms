-- OpenOMS — Database schema
-- Migration 000001: Full schema with enums, tables, indexes, RLS, triggers
--
-- Row-Level Security (RLS) ensures every tenant sees ONLY its own data.
-- The Go middleware sets: SET app.current_tenant_id = '<uuid>'
-- PostgreSQL then auto-filters every query with: WHERE tenant_id = '<uuid>'

-- ============================================================
-- Extensions
-- ============================================================
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";   -- UUID v4 generation
CREATE EXTENSION IF NOT EXISTS "pgcrypto";    -- Cryptographic functions

-- ============================================================
-- Application role (non-superuser, RLS-enforced)
-- ============================================================
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'openoms_app') THEN
        CREATE ROLE openoms_app WITH LOGIN PASSWORD 'openoms-dev-password';
    END IF;
END
$$;

GRANT CONNECT ON DATABASE openoms TO openoms_app;
GRANT USAGE ON SCHEMA public TO openoms_app;

-- ============================================================
-- Enum types
-- ============================================================
CREATE TYPE plan_type AS ENUM ('free', 'lite', 'pro', 'enterprise');
CREATE TYPE user_role AS ENUM ('owner', 'admin', 'member');
CREATE TYPE integration_provider AS ENUM ('allegro', 'inpost', 'dhl', 'dpd', 'woocommerce');
CREATE TYPE integration_status AS ENUM ('active', 'inactive', 'error');
CREATE TYPE order_source AS ENUM ('allegro', 'woocommerce', 'manual');
CREATE TYPE order_status AS ENUM (
    'new', 'confirmed', 'processing', 'ready_to_ship', 'shipped',
    'in_transit', 'delivered', 'completed', 'cancelled', 'refunded', 'on_hold'
);
CREATE TYPE shipment_provider AS ENUM ('inpost', 'dhl', 'dpd', 'manual');
CREATE TYPE shipment_status AS ENUM (
    'created', 'label_ready', 'picked_up', 'in_transit',
    'out_for_delivery', 'delivered', 'returned', 'failed'
);
CREATE TYPE webhook_status AS ENUM ('received', 'processing', 'processed', 'failed');

-- ============================================================
-- Table: tenants
-- ============================================================
CREATE TABLE tenants (
    id          UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    name        VARCHAR(255) NOT NULL,
    slug        VARCHAR(100) NOT NULL UNIQUE,
    plan        plan_type   NOT NULL DEFAULT 'free',
    settings    JSONB       NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================
-- Table: users
-- ============================================================
CREATE TABLE users (
    id            UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id     UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    email         VARCHAR(255) NOT NULL,
    name          VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role          user_role   NOT NULL DEFAULT 'member',
    last_login_at TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(tenant_id, email)
);

-- ============================================================
-- Table: integrations
-- ============================================================
CREATE TABLE integrations (
    id           UUID               PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id    UUID               NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    provider     integration_provider NOT NULL,
    status       integration_status NOT NULL DEFAULT 'inactive',
    credentials  JSONB              NOT NULL DEFAULT '{}',   -- encrypted at app level
    settings     JSONB              NOT NULL DEFAULT '{}',
    last_sync_at TIMESTAMPTZ,
    created_at   TIMESTAMPTZ        NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ        NOT NULL DEFAULT NOW(),

    UNIQUE(tenant_id, provider)
);

-- ============================================================
-- Table: orders
-- ============================================================
CREATE TABLE orders (
    id               UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id        UUID         NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    external_id      VARCHAR(255),
    source           order_source NOT NULL DEFAULT 'manual',
    integration_id   UUID         REFERENCES integrations(id) ON DELETE SET NULL,
    status           order_status NOT NULL DEFAULT 'new',
    customer_name    VARCHAR(255) NOT NULL,
    customer_email   VARCHAR(255),
    customer_phone   VARCHAR(50),
    shipping_address JSONB        NOT NULL DEFAULT '{}',
    billing_address  JSONB,
    items            JSONB        NOT NULL DEFAULT '[]',
    total_amount     DECIMAL(12,2) NOT NULL DEFAULT 0,
    currency         VARCHAR(3)   NOT NULL DEFAULT 'PLN',
    notes            TEXT,
    metadata         JSONB        NOT NULL DEFAULT '{}',
    ordered_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    shipped_at       TIMESTAMPTZ,
    delivered_at     TIMESTAMPTZ,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- ============================================================
-- Table: shipments
-- ============================================================
CREATE TABLE shipments (
    id              UUID             PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id       UUID             NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    order_id        UUID             NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    provider        shipment_provider NOT NULL,
    integration_id  UUID             REFERENCES integrations(id) ON DELETE SET NULL,
    tracking_number VARCHAR(255),
    status          shipment_status  NOT NULL DEFAULT 'created',
    label_url       VARCHAR(1024),
    carrier_data    JSONB            NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ      NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ      NOT NULL DEFAULT NOW()
);

-- ============================================================
-- Table: products
-- ============================================================
CREATE TABLE products (
    id             UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id      UUID         NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    external_id    VARCHAR(255),
    source         order_source NOT NULL DEFAULT 'manual',
    name           VARCHAR(500) NOT NULL,
    sku            VARCHAR(100),
    ean            VARCHAR(20),
    price          DECIMAL(12,2) NOT NULL DEFAULT 0,
    stock_quantity INTEGER      NOT NULL DEFAULT 0,
    metadata       JSONB        NOT NULL DEFAULT '{}',
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- ============================================================
-- Table: webhook_events
-- ============================================================
CREATE TABLE webhook_events (
    id           UUID                 PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id    UUID                 NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    provider     integration_provider NOT NULL,
    event_type   VARCHAR(100)         NOT NULL,
    payload      JSONB                NOT NULL,
    status       webhook_status       NOT NULL DEFAULT 'received',
    processed_at TIMESTAMPTZ,
    error        TEXT,
    created_at   TIMESTAMPTZ          NOT NULL DEFAULT NOW()
);

-- ============================================================
-- Table: audit_log
-- ============================================================
CREATE TABLE audit_log (
    id          BIGSERIAL   PRIMARY KEY,
    tenant_id   UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id     UUID        REFERENCES users(id) ON DELETE SET NULL,
    action      VARCHAR(100) NOT NULL,
    entity_type VARCHAR(50)  NOT NULL,
    entity_id   UUID,
    changes     JSONB       NOT NULL DEFAULT '{}',
    ip_address  INET,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================
-- Indexes (composite, tenant_id first for RLS performance)
-- ============================================================
CREATE INDEX idx_users_tenant ON users(tenant_id);
CREATE INDEX idx_integrations_tenant ON integrations(tenant_id, provider);
CREATE INDEX idx_orders_tenant_status ON orders(tenant_id, status, created_at DESC);
CREATE INDEX idx_orders_tenant_source ON orders(tenant_id, source);
CREATE INDEX idx_orders_external ON orders(tenant_id, source, external_id);
CREATE INDEX idx_shipments_tenant ON shipments(tenant_id, status);
CREATE INDEX idx_shipments_order ON shipments(order_id);
CREATE INDEX idx_shipments_tracking ON shipments(tracking_number) WHERE tracking_number IS NOT NULL;
CREATE INDEX idx_products_tenant ON products(tenant_id);
CREATE INDEX idx_products_sku ON products(tenant_id, sku) WHERE sku IS NOT NULL;
CREATE INDEX idx_products_ean ON products(tenant_id, ean) WHERE ean IS NOT NULL;
CREATE INDEX idx_webhook_queue ON webhook_events(status, created_at) WHERE status IN ('received', 'processing');
CREATE INDEX idx_audit_entity ON audit_log(tenant_id, entity_type, entity_id);

-- ============================================================
-- Row-Level Security (RLS)
-- ============================================================

-- Enable and force RLS on all tenant-scoped tables
ALTER TABLE tenants        ENABLE ROW LEVEL SECURITY;
ALTER TABLE users          ENABLE ROW LEVEL SECURITY;
ALTER TABLE integrations   ENABLE ROW LEVEL SECURITY;
ALTER TABLE orders         ENABLE ROW LEVEL SECURITY;
ALTER TABLE shipments      ENABLE ROW LEVEL SECURITY;
ALTER TABLE products       ENABLE ROW LEVEL SECURITY;
ALTER TABLE webhook_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_log      ENABLE ROW LEVEL SECURITY;

ALTER TABLE tenants        FORCE ROW LEVEL SECURITY;
ALTER TABLE users          FORCE ROW LEVEL SECURITY;
ALTER TABLE integrations   FORCE ROW LEVEL SECURITY;
ALTER TABLE orders         FORCE ROW LEVEL SECURITY;
ALTER TABLE shipments      FORCE ROW LEVEL SECURITY;
ALTER TABLE products       FORCE ROW LEVEL SECURITY;
ALTER TABLE webhook_events FORCE ROW LEVEL SECURITY;
ALTER TABLE audit_log      FORCE ROW LEVEL SECURITY;

-- Tenant isolation policies
CREATE POLICY tenant_isolation ON tenants
    USING (id = current_setting('app.current_tenant_id', true)::uuid);

CREATE POLICY tenant_isolation ON users
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

CREATE POLICY tenant_isolation ON integrations
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

CREATE POLICY tenant_isolation ON orders
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

CREATE POLICY tenant_isolation ON shipments
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

CREATE POLICY tenant_isolation ON products
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

CREATE POLICY tenant_isolation ON webhook_events
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

CREATE POLICY tenant_isolation ON audit_log
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- ============================================================
-- Grants for application role
-- ============================================================
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO openoms_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO openoms_app;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO openoms_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT USAGE, SELECT ON SEQUENCES TO openoms_app;

-- ============================================================
-- Trigger: auto-update updated_at on row modification
-- ============================================================
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_tenants_updated_at
    BEFORE UPDATE ON tenants
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER trigger_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER trigger_integrations_updated_at
    BEFORE UPDATE ON integrations
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER trigger_orders_updated_at
    BEFORE UPDATE ON orders
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER trigger_shipments_updated_at
    BEFORE UPDATE ON shipments
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER trigger_products_updated_at
    BEFORE UPDATE ON products
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();
