-- Marketplace category mappings: external marketplace category → internal OMS category.
CREATE TABLE IF NOT EXISTS marketplace_category_mappings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    integration_id UUID NOT NULL REFERENCES integrations(id) ON DELETE CASCADE,
    external_category_id TEXT NOT NULL,
    external_category_name TEXT NOT NULL DEFAULT '',
    category_id UUID REFERENCES product_categories(id) ON DELETE SET NULL,
    auto_created BOOLEAN NOT NULL DEFAULT false,
    confirmed BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, integration_id, external_category_id)
);

CREATE INDEX idx_marketplace_cat_mappings_integration
    ON marketplace_category_mappings(integration_id);

CREATE INDEX idx_marketplace_cat_mappings_category
    ON marketplace_category_mappings(category_id);

ALTER TABLE marketplace_category_mappings ENABLE ROW LEVEL SECURITY;
ALTER TABLE marketplace_category_mappings FORCE ROW LEVEL SECURITY;

CREATE POLICY marketplace_category_mappings_tenant_isolation
    ON marketplace_category_mappings
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- Add default_category_id to integrations for fallback category on import.
ALTER TABLE integrations
    ADD COLUMN IF NOT EXISTS default_category_id UUID REFERENCES product_categories(id) ON DELETE SET NULL;
