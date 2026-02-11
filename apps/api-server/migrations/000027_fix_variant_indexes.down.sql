-- Revert to non-tenant-scoped indexes
DROP INDEX IF EXISTS idx_product_variants_tenant_sku;
DROP INDEX IF EXISTS idx_product_variants_tenant_ean;
CREATE INDEX idx_product_variants_sku ON product_variants(sku) WHERE sku IS NOT NULL;
CREATE INDEX idx_product_variants_ean ON product_variants(ean) WHERE ean IS NOT NULL;
