ALTER TABLE products DROP COLUMN IF EXISTS is_bundle;
DROP TRIGGER IF EXISTS update_product_bundles_updated_at ON product_bundles;
DROP TABLE IF EXISTS product_bundles;
