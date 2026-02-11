-- Product variants (sizes, colors, etc.)
CREATE TABLE product_variants (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    sku TEXT,
    ean TEXT,
    name TEXT NOT NULL,
    attributes JSONB NOT NULL DEFAULT '{}',
    price_override DECIMAL(12,2),
    stock_quantity INTEGER NOT NULL DEFAULT 0,
    weight DECIMAL(8,3),
    image_url TEXT,
    position INTEGER NOT NULL DEFAULT 0,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE products ADD COLUMN has_variants BOOLEAN NOT NULL DEFAULT false;

-- RLS
ALTER TABLE product_variants ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON product_variants
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid);

-- Indexes
CREATE INDEX idx_product_variants_product ON product_variants(product_id);
CREATE INDEX idx_product_variants_tenant ON product_variants(tenant_id);
CREATE INDEX idx_product_variants_sku ON product_variants(sku) WHERE sku IS NOT NULL;
CREATE INDEX idx_product_variants_ean ON product_variants(ean) WHERE ean IS NOT NULL;

-- Updated_at trigger
CREATE TRIGGER update_product_variants_updated_at
    BEFORE UPDATE ON product_variants
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Grants
GRANT SELECT, INSERT, UPDATE, DELETE ON product_variants TO openoms_app;
