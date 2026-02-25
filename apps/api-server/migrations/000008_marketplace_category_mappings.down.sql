ALTER TABLE integrations DROP COLUMN IF EXISTS default_category_id;
DROP POLICY IF EXISTS marketplace_category_mappings_tenant_isolation ON marketplace_category_mappings;
DROP TABLE IF EXISTS marketplace_category_mappings;
