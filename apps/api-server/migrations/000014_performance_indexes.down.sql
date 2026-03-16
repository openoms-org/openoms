-- Note: cannot use CONCURRENTLY here — golang-migrate wraps in a transaction
DROP INDEX IF EXISTS idx_warehouse_stock_product_variant;
DROP INDEX IF EXISTS idx_audit_entity_created;
DROP INDEX IF EXISTS idx_suppliers_sync_active;
