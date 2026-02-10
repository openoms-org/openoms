-- Migration 000016: Enhance integrations for multi-account support
-- Drop old unique constraint (tenant_id, provider) to allow multiple accounts per provider
ALTER TABLE integrations DROP CONSTRAINT IF EXISTS integrations_tenant_id_provider_key;

-- Add new columns
ALTER TABLE integrations
    ADD COLUMN label TEXT,
    ADD COLUMN sync_cursor TEXT,
    ADD COLUMN error_message TEXT;

-- New unique: one label per provider per tenant (NULL label counts as '')
CREATE UNIQUE INDEX idx_integrations_tenant_provider_label
    ON integrations(tenant_id, provider, COALESCE(label, ''));
