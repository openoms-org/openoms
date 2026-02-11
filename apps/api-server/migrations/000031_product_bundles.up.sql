CREATE TABLE product_bundles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    bundle_product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    component_product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    component_variant_id UUID REFERENCES product_variants(id),
    quantity INTEGER NOT NULL DEFAULT 1,
    position INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(bundle_product_id, component_product_id, component_variant_id)
);

ALTER TABLE product_bundles ENABLE ROW LEVEL SECURITY;
ALTER TABLE product_bundles FORCE ROW LEVEL SECURITY;
CREATE POLICY product_bundles_tenant_isolation ON product_bundles
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE INDEX idx_product_bundles_bundle ON product_bundles(bundle_product_id);
CREATE INDEX idx_product_bundles_component ON product_bundles(component_product_id);
GRANT SELECT, INSERT, UPDATE, DELETE ON product_bundles TO openoms_app;
CREATE TRIGGER update_product_bundles_updated_at
    BEFORE UPDATE ON product_bundles
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

ALTER TABLE products ADD COLUMN is_bundle BOOLEAN NOT NULL DEFAULT false;
