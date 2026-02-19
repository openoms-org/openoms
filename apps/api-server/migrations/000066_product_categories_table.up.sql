-- Hierarchical product categories with adjacency list model.
-- Supports tree queries via recursive CTE. Max depth enforced in application (6 levels).

CREATE TABLE product_categories (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id  UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    parent_id  UUID REFERENCES product_categories(id) ON DELETE SET NULL,
    name       VARCHAR(200) NOT NULL,
    slug       VARCHAR(200) NOT NULL,
    color      VARCHAR(7) DEFAULT '#6b7280',
    icon       VARCHAR(50),
    position   INT NOT NULL DEFAULT 0,
    depth      INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_product_categories_tenant ON product_categories(tenant_id);
CREATE INDEX idx_product_categories_parent ON product_categories(tenant_id, parent_id);
CREATE UNIQUE INDEX idx_product_categories_slug ON product_categories(tenant_id, slug);

ALTER TABLE product_categories ENABLE ROW LEVEL SECURITY;
ALTER TABLE product_categories FORCE ROW LEVEL SECURITY;
CREATE POLICY product_categories_tenant_isolation ON product_categories
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

GRANT SELECT, INSERT, UPDATE, DELETE ON product_categories TO openoms_app;

CREATE TRIGGER trigger_product_categories_updated_at
    BEFORE UPDATE ON product_categories FOR EACH ROW EXECUTE FUNCTION update_updated_at();
