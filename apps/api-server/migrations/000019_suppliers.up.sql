-- Suppliers table
CREATE TABLE suppliers (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name        VARCHAR(255) NOT NULL,
    code        VARCHAR(100),
    feed_url    TEXT,
    feed_format TEXT NOT NULL DEFAULT 'iof',
    status      TEXT NOT NULL DEFAULT 'active',
    settings    JSONB NOT NULL DEFAULT '{}',
    last_sync_at TIMESTAMPTZ,
    error_message TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_suppliers_tenant ON suppliers(tenant_id);
ALTER TABLE suppliers ENABLE ROW LEVEL SECURITY;
ALTER TABLE suppliers FORCE ROW LEVEL SECURITY;
CREATE POLICY suppliers_tenant_isolation ON suppliers
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
GRANT SELECT, INSERT, UPDATE, DELETE ON suppliers TO openoms_app;

CREATE TRIGGER trigger_suppliers_updated_at
    BEFORE UPDATE ON suppliers FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- Supplier products table (maps supplier items to OMS products)
CREATE TABLE supplier_products (
    id             UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id      UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    supplier_id    UUID NOT NULL REFERENCES suppliers(id) ON DELETE CASCADE,
    product_id     UUID REFERENCES products(id) ON DELETE SET NULL,
    external_id    TEXT NOT NULL,
    name           TEXT NOT NULL,
    ean            TEXT,
    sku            TEXT,
    price          DECIMAL(12,2),
    stock_quantity INTEGER NOT NULL DEFAULT 0,
    metadata       JSONB NOT NULL DEFAULT '{}',
    last_synced_at TIMESTAMPTZ,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_supplier_products_tenant ON supplier_products(tenant_id);
CREATE INDEX idx_supplier_products_supplier ON supplier_products(supplier_id);
CREATE INDEX idx_supplier_products_ean ON supplier_products(tenant_id, ean) WHERE ean IS NOT NULL;
CREATE UNIQUE INDEX idx_supplier_products_unique ON supplier_products(tenant_id, supplier_id, external_id);
ALTER TABLE supplier_products ENABLE ROW LEVEL SECURITY;
ALTER TABLE supplier_products FORCE ROW LEVEL SECURITY;
CREATE POLICY supplier_products_tenant_isolation ON supplier_products
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
GRANT SELECT, INSERT, UPDATE, DELETE ON supplier_products TO openoms_app;

CREATE TRIGGER trigger_supplier_products_updated_at
    BEFORE UPDATE ON supplier_products FOR EACH ROW EXECUTE FUNCTION update_updated_at();
