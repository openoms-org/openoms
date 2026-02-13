-- Drop the unique index
DROP INDEX IF EXISTS idx_products_tenant_sku;
-- Recreate the original non-unique index
CREATE INDEX idx_products_sku ON products(tenant_id, sku);
