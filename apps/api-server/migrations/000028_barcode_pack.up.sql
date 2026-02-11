-- Phase 25: No new tables needed — barcode lookup uses existing products/product_variants tables.
-- We just need indexes for efficient SKU/EAN lookups.

CREATE INDEX IF NOT EXISTS idx_products_sku ON products(sku) WHERE sku IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_products_ean ON products(ean) WHERE ean IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_product_variants_sku ON product_variants(sku) WHERE sku IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_product_variants_ean ON product_variants(ean) WHERE ean IS NOT NULL;
