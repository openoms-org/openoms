CREATE TABLE automation_rules (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT,
    enabled BOOLEAN NOT NULL DEFAULT true,
    priority INTEGER NOT NULL DEFAULT 0,
    trigger_event TEXT NOT NULL,
    conditions JSONB NOT NULL DEFAULT '[]',
    actions JSONB NOT NULL DEFAULT '[]',
    last_fired_at TIMESTAMPTZ,
    fire_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE automation_rules ENABLE ROW LEVEL SECURITY;
CREATE POLICY automation_rules_tenant_isolation ON automation_rules
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid);
CREATE INDEX idx_automation_rules_tenant_enabled ON automation_rules(tenant_id, enabled);
CREATE TRIGGER set_updated_at BEFORE UPDATE ON automation_rules
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();
GRANT SELECT, INSERT, UPDATE, DELETE ON automation_rules TO openoms_app;

CREATE TABLE automation_rule_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    rule_id UUID NOT NULL REFERENCES automation_rules(id) ON DELETE CASCADE,
    trigger_event TEXT NOT NULL,
    entity_type TEXT NOT NULL,
    entity_id UUID NOT NULL,
    conditions_met BOOLEAN NOT NULL,
    actions_executed JSONB NOT NULL DEFAULT '[]',
    error_message TEXT,
    executed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE automation_rule_logs ENABLE ROW LEVEL SECURITY;
CREATE POLICY automation_rule_logs_tenant_isolation ON automation_rule_logs
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid);
CREATE INDEX idx_automation_rule_logs_rule_id ON automation_rule_logs(rule_id);
CREATE INDEX idx_automation_rule_logs_executed_at ON automation_rule_logs(executed_at DESC);
GRANT SELECT, INSERT ON automation_rule_logs TO openoms_app;
