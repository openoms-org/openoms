-- Listing sync configs — per-integration sync configuration
CREATE TABLE listing_sync_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    integration_id UUID NOT NULL REFERENCES integrations(id) ON DELETE CASCADE,
    sync_direction TEXT NOT NULL DEFAULT 'bidirectional',
    auto_sync BOOLEAN NOT NULL DEFAULT false,
    sync_interval_minutes INT NOT NULL DEFAULT 30,
    field_mapping JSONB NOT NULL DEFAULT '{}',
    price_rule TEXT NOT NULL DEFAULT 'same',
    price_modifier DECIMAL(10,2) NOT NULL DEFAULT 0,
    stock_buffer INT NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'active',
    last_sync_at TIMESTAMPTZ,
    last_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Listing sync log — individual sync operation records
CREATE TABLE listing_sync_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    config_id UUID NOT NULL REFERENCES listing_sync_configs(id) ON DELETE CASCADE,
    direction TEXT NOT NULL,
    entity_type TEXT NOT NULL,
    product_id UUID,
    external_id TEXT,
    status TEXT NOT NULL DEFAULT 'success',
    changes JSONB,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- RLS
ALTER TABLE listing_sync_configs ENABLE ROW LEVEL SECURITY;
ALTER TABLE listing_sync_log ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON listing_sync_configs USING (tenant_id = current_setting('app.current_tenant_id')::uuid);
CREATE POLICY tenant_isolation ON listing_sync_log USING (tenant_id = current_setting('app.current_tenant_id')::uuid);

-- Indexes
CREATE INDEX idx_listing_sync_configs_tenant ON listing_sync_configs(tenant_id);
CREATE INDEX idx_listing_sync_configs_integration ON listing_sync_configs(integration_id);
CREATE INDEX idx_listing_sync_log_config_created ON listing_sync_log(tenant_id, config_id, created_at DESC);

-- Grants
GRANT SELECT, INSERT, UPDATE, DELETE ON listing_sync_configs TO openoms_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON listing_sync_log TO openoms_app;
