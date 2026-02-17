-- Stock sync channels: per-channel configuration for real-time stock synchronization
CREATE TABLE stock_sync_channels (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    integration_id UUID REFERENCES integrations(id) ON DELETE SET NULL,
    channel_type TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT true,
    stock_buffer INTEGER NOT NULL DEFAULT 0,
    sync_mode TEXT NOT NULL DEFAULT 'realtime',
    priority INTEGER NOT NULL DEFAULT 0,
    last_sync_at TIMESTAMPTZ,
    last_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Stock sync events: audit trail of every stock change propagation
CREATE TABLE stock_sync_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    product_id UUID REFERENCES products(id) ON DELETE SET NULL,
    sku TEXT,
    trigger_type TEXT NOT NULL,
    old_quantity INTEGER NOT NULL DEFAULT 0,
    new_quantity INTEGER NOT NULL DEFAULT 0,
    available_quantity INTEGER NOT NULL DEFAULT 0,
    channels_notified INTEGER NOT NULL DEFAULT 0,
    channels_failed INTEGER NOT NULL DEFAULT 0,
    details JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- RLS
ALTER TABLE stock_sync_channels ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON stock_sync_channels
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid);

ALTER TABLE stock_sync_events ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON stock_sync_events
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid);

-- Indexes
CREATE INDEX idx_stock_sync_channels_tenant ON stock_sync_channels(tenant_id);
CREATE INDEX idx_stock_sync_channels_integration ON stock_sync_channels(integration_id);
CREATE INDEX idx_stock_sync_events_tenant_product ON stock_sync_events(tenant_id, product_id, created_at DESC);
CREATE INDEX idx_stock_sync_events_tenant_created ON stock_sync_events(tenant_id, created_at DESC);

-- Triggers
CREATE TRIGGER update_stock_sync_channels_updated_at
    BEFORE UPDATE ON stock_sync_channels
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Grants
GRANT SELECT, INSERT, UPDATE, DELETE ON stock_sync_channels TO openoms_app;
GRANT SELECT, INSERT ON stock_sync_events TO openoms_app;
