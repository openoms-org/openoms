CREATE TABLE price_lists (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT,
    currency TEXT NOT NULL DEFAULT 'PLN',
    is_default BOOLEAN NOT NULL DEFAULT false,
    discount_type TEXT NOT NULL DEFAULT 'percentage', -- percentage, fixed, override
    active BOOLEAN NOT NULL DEFAULT true,
    valid_from TIMESTAMPTZ,
    valid_to TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE price_list_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    price_list_id UUID NOT NULL REFERENCES price_lists(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    variant_id UUID REFERENCES product_variants(id),
    price DECIMAL(12,2), -- override price (when discount_type = 'override')
    discount DECIMAL(5,2), -- percentage or fixed amount
    min_quantity INTEGER DEFAULT 1, -- quantity threshold for this price
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(price_list_id, product_id, variant_id, min_quantity)
);

-- RLS on both tables
ALTER TABLE price_lists ENABLE ROW LEVEL SECURITY;
ALTER TABLE price_lists FORCE ROW LEVEL SECURITY;
CREATE POLICY price_lists_tenant_isolation ON price_lists
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

ALTER TABLE price_list_items ENABLE ROW LEVEL SECURITY;
ALTER TABLE price_list_items FORCE ROW LEVEL SECURITY;
CREATE POLICY price_list_items_tenant_isolation ON price_list_items
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

CREATE INDEX idx_price_lists_tenant ON price_lists(tenant_id);
CREATE INDEX idx_price_list_items_list ON price_list_items(price_list_id);
CREATE INDEX idx_price_list_items_product ON price_list_items(product_id);

GRANT SELECT, INSERT, UPDATE, DELETE ON price_lists TO openoms_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON price_list_items TO openoms_app;

CREATE TRIGGER update_price_lists_updated_at BEFORE UPDATE ON price_lists FOR EACH ROW EXECUTE FUNCTION update_updated_at();
CREATE TRIGGER update_price_list_items_updated_at BEFORE UPDATE ON price_list_items FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- Add price_list_id to customers for B2B pricing
ALTER TABLE customers ADD COLUMN IF NOT EXISTS price_list_id UUID REFERENCES price_lists(id);
