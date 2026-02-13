CREATE TABLE stocktakes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    warehouse_id UUID NOT NULL REFERENCES warehouses(id),
    name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'draft', -- draft, in_progress, completed, cancelled
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    notes TEXT,
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE stocktake_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    stocktake_id UUID NOT NULL REFERENCES stocktakes(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id),
    expected_quantity INT NOT NULL DEFAULT 0,
    counted_quantity INT,
    difference INT GENERATED ALWAYS AS (COALESCE(counted_quantity, 0) - expected_quantity) STORED,
    notes TEXT,
    counted_at TIMESTAMPTZ,
    counted_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- RLS
ALTER TABLE stocktakes ENABLE ROW LEVEL SECURITY;
ALTER TABLE stocktakes FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON stocktakes
    USING (tenant_id = COALESCE(NULLIF(current_setting('app.current_tenant_id', true), '')::uuid, tenant_id));

ALTER TABLE stocktake_items ENABLE ROW LEVEL SECURITY;
ALTER TABLE stocktake_items FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON stocktake_items
    USING (tenant_id = COALESCE(NULLIF(current_setting('app.current_tenant_id', true), '')::uuid, tenant_id));

-- Indexes
CREATE INDEX idx_stocktakes_warehouse ON stocktakes(warehouse_id);
CREATE INDEX idx_stocktakes_tenant ON stocktakes(tenant_id);
CREATE INDEX idx_stocktakes_status ON stocktakes(status);
CREATE INDEX idx_stocktake_items_stocktake ON stocktake_items(stocktake_id);
CREATE INDEX idx_stocktake_items_product ON stocktake_items(product_id);

-- Triggers
CREATE TRIGGER update_stocktakes_updated_at BEFORE UPDATE ON stocktakes FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Grants
GRANT SELECT, INSERT, UPDATE, DELETE ON stocktakes TO openoms_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON stocktake_items TO openoms_app;
