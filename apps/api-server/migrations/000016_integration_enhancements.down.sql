-- Migration 000016 down: Revert integration enhancements
DROP INDEX IF EXISTS idx_integrations_tenant_provider_label;

ALTER TABLE integrations
    DROP COLUMN IF EXISTS label,
    DROP COLUMN IF EXISTS sync_cursor,
    DROP COLUMN IF EXISTS error_message;

-- Restore original unique constraint
ALTER TABLE integrations ADD CONSTRAINT integrations_tenant_id_provider_key UNIQUE (tenant_id, provider);
