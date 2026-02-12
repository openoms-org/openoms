-- Table for delayed/scheduled automation actions
CREATE TABLE IF NOT EXISTS automation_delayed_actions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    rule_id UUID NOT NULL REFERENCES automation_rules(id) ON DELETE CASCADE,
    action_index INT NOT NULL,
    order_id UUID REFERENCES orders(id),
    execute_at TIMESTAMPTZ NOT NULL,
    executed BOOLEAN NOT NULL DEFAULT FALSE,
    executed_at TIMESTAMPTZ,
    error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- Store the full action + event data needed to execute later
    action_data JSONB NOT NULL DEFAULT '{}',
    event_data JSONB NOT NULL DEFAULT '{}'
);

CREATE INDEX IF NOT EXISTS idx_delayed_actions_execute
    ON automation_delayed_actions(execute_at) WHERE NOT executed;

CREATE INDEX IF NOT EXISTS idx_delayed_actions_tenant
    ON automation_delayed_actions(tenant_id);

-- RLS
ALTER TABLE automation_delayed_actions ENABLE ROW LEVEL SECURITY;
ALTER TABLE automation_delayed_actions FORCE ROW LEVEL SECURITY;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_policies WHERE tablename = 'automation_delayed_actions' AND policyname = 'tenant_isolation'
    ) THEN
        CREATE POLICY tenant_isolation ON automation_delayed_actions
            USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
    END IF;
END
$$;

GRANT SELECT, INSERT, UPDATE ON automation_delayed_actions TO openoms_app;
