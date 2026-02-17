-- Pick & Pack workflow tables
CREATE TABLE pick_pack_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    session_type TEXT NOT NULL DEFAULT 'single', -- 'single' (one order) or 'batch' (multiple orders)
    status TEXT NOT NULL DEFAULT 'picking', -- picking, packing, completed, cancelled
    assigned_to UUID REFERENCES users(id),
    started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at TIMESTAMPTZ,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE pick_pack_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    session_id UUID NOT NULL REFERENCES pick_pack_sessions(id) ON DELETE CASCADE,
    order_id UUID NOT NULL REFERENCES orders(id),
    product_id UUID REFERENCES products(id),
    sku TEXT NOT NULL,
    product_name TEXT NOT NULL,
    quantity_required INT NOT NULL,
    quantity_picked INT NOT NULL DEFAULT 0,
    quantity_packed INT NOT NULL DEFAULT 0,
    pick_location TEXT,
    picked_at TIMESTAMPTZ,
    packed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- RLS
ALTER TABLE pick_pack_sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE pick_pack_items ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON pick_pack_sessions USING (tenant_id = current_setting('app.current_tenant_id')::uuid);
CREATE POLICY tenant_isolation ON pick_pack_items USING (tenant_id = current_setting('app.current_tenant_id')::uuid);

-- Indexes
CREATE INDEX idx_pick_pack_sessions_tenant ON pick_pack_sessions(tenant_id);
CREATE INDEX idx_pick_pack_sessions_status ON pick_pack_sessions(status);
CREATE INDEX idx_pick_pack_sessions_assigned ON pick_pack_sessions(assigned_to);
CREATE INDEX idx_pick_pack_items_session ON pick_pack_items(session_id);
CREATE INDEX idx_pick_pack_items_order ON pick_pack_items(order_id);

GRANT SELECT, INSERT, UPDATE, DELETE ON pick_pack_sessions TO openoms_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON pick_pack_items TO openoms_app;
