-- Link products and suppliers to hierarchical categories.
-- Creates supplier_category_mappings for feed-to-OMS category resolution.

-- Add category_id FK to products (nullable, coexists with old category TEXT column)
ALTER TABLE products ADD COLUMN category_id UUID REFERENCES product_categories(id) ON DELETE SET NULL;
CREATE INDEX idx_products_category_id ON products(tenant_id, category_id);

-- Add default_category_id FK to suppliers
ALTER TABLE suppliers ADD COLUMN default_category_id UUID REFERENCES product_categories(id) ON DELETE SET NULL;

-- Supplier category mappings (feed category name → OMS category)
CREATE TABLE supplier_category_mappings (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    supplier_id     UUID NOT NULL REFERENCES suppliers(id) ON DELETE CASCADE,
    source_category TEXT NOT NULL,
    category_id     UUID REFERENCES product_categories(id) ON DELETE SET NULL,
    auto_matched    BOOLEAN NOT NULL DEFAULT false,
    confirmed       BOOLEAN NOT NULL DEFAULT false,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_supplier_category_mapping_unique
    ON supplier_category_mappings(tenant_id, supplier_id, source_category);
CREATE INDEX idx_supplier_category_mappings_supplier
    ON supplier_category_mappings(supplier_id);

ALTER TABLE supplier_category_mappings ENABLE ROW LEVEL SECURITY;
ALTER TABLE supplier_category_mappings FORCE ROW LEVEL SECURITY;
CREATE POLICY supplier_category_mappings_tenant_isolation ON supplier_category_mappings
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

GRANT SELECT, INSERT, UPDATE, DELETE ON supplier_category_mappings TO openoms_app;

-- Persist source category from feed in supplier_products for future re-mapping
ALTER TABLE supplier_products ADD COLUMN source_category TEXT;
