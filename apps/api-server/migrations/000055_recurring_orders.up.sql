CREATE TABLE recurring_orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    customer_id UUID REFERENCES customers(id),
    customer_name TEXT NOT NULL,
    customer_email TEXT,
    status TEXT NOT NULL DEFAULT 'active',
    frequency TEXT NOT NULL,
    interval_days INT NOT NULL,
    next_order_date DATE NOT NULL,
    last_order_date DATE,
    end_date DATE,
    total_orders_created INT NOT NULL DEFAULT 0,
    max_orders INT,
    shipping_address JSONB,
    notes TEXT,
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE recurring_order_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    recurring_order_id UUID NOT NULL REFERENCES recurring_orders(id) ON DELETE CASCADE,
    product_id UUID REFERENCES products(id),
    sku TEXT NOT NULL,
    product_name TEXT NOT NULL,
    quantity INT NOT NULL,
    unit_price DECIMAL(12,2) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE recurring_orders ENABLE ROW LEVEL SECURITY;
ALTER TABLE recurring_order_items ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON recurring_orders USING (tenant_id = current_setting('app.current_tenant_id')::uuid);
CREATE POLICY tenant_isolation ON recurring_order_items USING (tenant_id = current_setting('app.current_tenant_id')::uuid);

CREATE INDEX idx_recurring_orders_tenant ON recurring_orders(tenant_id);
CREATE INDEX idx_recurring_orders_customer ON recurring_orders(customer_id);
CREATE INDEX idx_recurring_orders_status ON recurring_orders(status);
CREATE INDEX idx_recurring_orders_next_date ON recurring_orders(next_order_date);
CREATE INDEX idx_recurring_order_items_recurring ON recurring_order_items(recurring_order_id);

GRANT SELECT, INSERT, UPDATE, DELETE ON recurring_orders TO openoms_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON recurring_order_items TO openoms_app;
