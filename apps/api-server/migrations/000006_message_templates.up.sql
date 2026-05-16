CREATE TABLE IF NOT EXISTS message_templates (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name text NOT NULL,
    channel text NOT NULL DEFAULT 'allegro',
    subject text,
    body text NOT NULL,
    variables text[] DEFAULT '{}',
    is_autoresponder boolean DEFAULT false,
    trigger_event text,
    enabled boolean DEFAULT true,
    created_at timestamptz DEFAULT NOW(),
    updated_at timestamptz DEFAULT NOW()
);

ALTER TABLE message_templates ENABLE ROW LEVEL SECURITY;

CREATE POLICY message_templates_tenant ON message_templates
    USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

ALTER TABLE message_templates FORCE ROW LEVEL SECURITY;

-- Speed up RLS-filtered queries on message_templates.
CREATE INDEX IF NOT EXISTS idx_message_templates_tenant_id
    ON message_templates(tenant_id);

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'openoms_app') THEN
        GRANT SELECT, INSERT, UPDATE, DELETE ON message_templates TO openoms_app;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'openoms') THEN
        GRANT SELECT, INSERT, UPDATE, DELETE ON message_templates TO openoms;
    END IF;
END $$;
