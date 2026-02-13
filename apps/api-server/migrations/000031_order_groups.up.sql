CREATE TABLE order_groups (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    group_type TEXT NOT NULL,
    source_order_ids UUID[] NOT NULL,
    target_order_ids UUID[] NOT NULL,
    notes TEXT,
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE order_groups ENABLE ROW LEVEL SECURITY;
ALTER TABLE order_groups FORCE ROW LEVEL SECURITY;
CREATE POLICY order_groups_tenant_isolation ON order_groups
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE INDEX idx_order_groups_tenant ON order_groups(tenant_id);
GRANT SELECT, INSERT, UPDATE, DELETE ON order_groups TO openoms_app;

ALTER TABLE orders ADD COLUMN merged_into UUID REFERENCES orders(id);
ALTER TABLE orders ADD COLUMN split_from UUID REFERENCES orders(id);
