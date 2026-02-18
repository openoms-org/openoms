-- Add per-listing stock sync mode (auto vs manual) to product_listings.
ALTER TABLE product_listings
    ADD COLUMN stock_sync_mode TEXT NOT NULL DEFAULT 'auto';

-- Partial index for fast lookup of auto-sync listings by product.
CREATE INDEX idx_product_listings_auto_sync
    ON product_listings(tenant_id, product_id)
    WHERE status = 'active' AND external_id IS NOT NULL AND stock_sync_mode = 'auto';
