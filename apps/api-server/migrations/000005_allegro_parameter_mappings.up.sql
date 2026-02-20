CREATE TABLE allegro_parameter_mappings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    supplier_id UUID NOT NULL REFERENCES suppliers(id) ON DELETE CASCADE,
    allegro_category_id TEXT NOT NULL,
    allegro_param_id TEXT NOT NULL,
    allegro_param_name TEXT NOT NULL,
    source_type TEXT NOT NULL CHECK (source_type IN ('attribute', 'field', 'static')),
    source_key TEXT NOT NULL,
    value_mapping JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, supplier_id, allegro_category_id, allegro_param_id)
);

CREATE INDEX idx_allegro_param_mappings_lookup
    ON allegro_parameter_mappings (tenant_id, supplier_id, allegro_category_id);

ALTER TABLE allegro_parameter_mappings ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON allegro_parameter_mappings
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid);

GRANT ALL ON allegro_parameter_mappings TO openoms;
