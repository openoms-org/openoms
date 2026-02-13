-- Drop the old non-unique index
DROP INDEX IF EXISTS idx_products_sku;
-- Create unique index (only when SKU is not empty)
CREATE UNIQUE INDEX idx_products_tenant_sku ON products(tenant_id, sku) WHERE sku IS NOT NULL AND sku != '';
