CREATE TABLE dropship_orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    order_id UUID NOT NULL REFERENCES orders(id),
    supplier_id UUID NOT NULL REFERENCES suppliers(id),
    supplier_name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    -- Status: pending, sent, confirmed, shipped, delivered, cancelled
    supplier_reference TEXT,
    tracking_number TEXT,
    carrier TEXT,
    notes TEXT,
    total_cost DECIMAL(12,2) NOT NULL DEFAULT 0,
    currency TEXT NOT NULL DEFAULT 'PLN',
    sent_at TIMESTAMPTZ,
    confirmed_at TIMESTAMPTZ,
    shipped_at TIMESTAMPTZ,
    delivered_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE dropship_order_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    dropship_order_id UUID NOT NULL REFERENCES dropship_orders(id) ON DELETE CASCADE,
    product_id UUID REFERENCES products(id),
    sku TEXT NOT NULL,
    product_name TEXT NOT NULL,
    quantity INT NOT NULL,
    unit_cost DECIMAL(12,2) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE dropship_orders ENABLE ROW LEVEL SECURITY;
ALTER TABLE dropship_order_items ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON dropship_orders USING (tenant_id = current_setting('app.current_tenant_id')::uuid);
CREATE POLICY tenant_isolation ON dropship_order_items USING (tenant_id = current_setting('app.current_tenant_id')::uuid);

CREATE INDEX idx_dropship_orders_tenant ON dropship_orders(tenant_id);
CREATE INDEX idx_dropship_orders_order ON dropship_orders(order_id);
CREATE INDEX idx_dropship_orders_supplier ON dropship_orders(supplier_id);
CREATE INDEX idx_dropship_orders_status ON dropship_orders(status);
CREATE INDEX idx_dropship_items_order ON dropship_order_items(dropship_order_id);

-- Add dropship flag to products
ALTER TABLE products ADD COLUMN is_dropship BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE products ADD COLUMN dropship_supplier_id UUID REFERENCES suppliers(id);

GRANT SELECT, INSERT, UPDATE, DELETE ON dropship_orders TO openoms_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON dropship_order_items TO openoms_app;
