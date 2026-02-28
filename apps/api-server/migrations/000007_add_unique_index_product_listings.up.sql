-- Prevent duplicate listings per integration (same external_id + integration_id).
CREATE UNIQUE INDEX IF NOT EXISTS idx_product_listings_external_integration
    ON product_listings(external_id, integration_id)
    WHERE external_id IS NOT NULL;

-- Prevent duplicate listings per product per integration (one listing per product+integration).
CREATE UNIQUE INDEX IF NOT EXISTS idx_product_listings_product_integration
    ON product_listings(tenant_id, product_id, integration_id);
