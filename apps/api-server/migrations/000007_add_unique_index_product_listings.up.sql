-- Prevent duplicate listings per integration (same external_id + integration_id).
-- Uses CONCURRENTLY to avoid table-level locks in production.
CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS idx_product_listings_external_integration
    ON product_listings(external_id, integration_id)
    WHERE external_id IS NOT NULL;

-- Speed up RLS-filtered queries on message_templates.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_message_templates_tenant_id
    ON message_templates(tenant_id);
