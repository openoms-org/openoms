-- Migration 000017: Product listings table
CREATE TABLE product_listings (
    id              UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id       UUID         NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    product_id      UUID         NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    integration_id  UUID         NOT NULL REFERENCES integrations(id) ON DELETE CASCADE,
    external_id     TEXT,
    status          TEXT         NOT NULL DEFAULT 'pending',
    url             TEXT,
    price_override  DECIMAL(12,2),
    stock_override  INTEGER,
    sync_status     TEXT         NOT NULL DEFAULT 'pending',
    last_synced_at  TIMESTAMPTZ,
    error_message   TEXT,
    metadata        JSONB        NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_product_listings_tenant ON product_listings(tenant_id);
CREATE INDEX idx_product_listings_product ON product_listings(tenant_id, product_id);
CREATE INDEX idx_product_listings_integration ON product_listings(tenant_id, integration_id);
CREATE UNIQUE INDEX idx_product_listings_ext ON product_listings(tenant_id, integration_id, external_id)
    WHERE external_id IS NOT NULL;

ALTER TABLE product_listings ENABLE ROW LEVEL SECURITY;
ALTER TABLE product_listings FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON product_listings
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

GRANT SELECT, INSERT, UPDATE, DELETE ON product_listings TO openoms_app;

CREATE TRIGGER trigger_product_listings_updated_at
    BEFORE UPDATE ON product_listings
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();
