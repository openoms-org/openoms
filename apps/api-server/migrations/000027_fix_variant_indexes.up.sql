-- Fix product_variants indexes to be tenant-scoped
DROP INDEX IF EXISTS idx_product_variants_sku;
DROP INDEX IF EXISTS idx_product_variants_ean;
CREATE INDEX idx_product_variants_tenant_sku ON product_variants(tenant_id, sku) WHERE sku IS NOT NULL;
CREATE INDEX idx_product_variants_tenant_ean ON product_variants(tenant_id, ean) WHERE ean IS NOT NULL;
