CREATE TABLE repricing_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    strategy TEXT NOT NULL,
    priority INT NOT NULL DEFAULT 0,
    scope_type TEXT NOT NULL DEFAULT 'all',
    scope_value JSONB,
    params JSONB NOT NULL DEFAULT '{}',
    min_price DECIMAL(12,2),
    max_price DECIMAL(12,2),
    channels JSONB DEFAULT '["internal"]',
    last_applied_at TIMESTAMPTZ,
    products_affected INT NOT NULL DEFAULT 0,
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE repricing_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    rule_id UUID NOT NULL REFERENCES repricing_rules(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id),
    old_price DECIMAL(12,2) NOT NULL,
    new_price DECIMAL(12,2) NOT NULL,
    reason TEXT,
    channel TEXT NOT NULL DEFAULT 'internal',
    applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE repricing_rules ENABLE ROW LEVEL SECURITY;
ALTER TABLE repricing_log ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON repricing_rules USING (tenant_id = current_setting('app.current_tenant_id')::uuid);
CREATE POLICY tenant_isolation ON repricing_log USING (tenant_id = current_setting('app.current_tenant_id')::uuid);

CREATE INDEX idx_repricing_rules_tenant ON repricing_rules(tenant_id);
CREATE INDEX idx_repricing_rules_status ON repricing_rules(status);
CREATE INDEX idx_repricing_log_rule ON repricing_log(rule_id);
CREATE INDEX idx_repricing_log_product ON repricing_log(product_id);
CREATE INDEX idx_repricing_log_applied ON repricing_log(applied_at);

GRANT SELECT, INSERT, UPDATE, DELETE ON repricing_rules TO openoms_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON repricing_log TO openoms_app;
