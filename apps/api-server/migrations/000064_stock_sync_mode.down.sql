DROP INDEX IF EXISTS idx_product_listings_auto_sync;
ALTER TABLE product_listings DROP COLUMN IF EXISTS stock_sync_mode;
