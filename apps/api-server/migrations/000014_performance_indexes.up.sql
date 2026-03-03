-- Performance indexes identified by code audit (2026-03-03)

-- Composite index for stock sync JOIN: warehouse_stock queried by (product_id, variant_id)
-- Supports stock_sync_worker.go and order_service.go stock operations
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_warehouse_stock_product_variant
ON warehouse_stock(product_id, variant_id);

-- Composite index for audit log entity queries with created_at ordering
-- Supports audit_repository.go ListByEntity with ORDER BY created_at DESC LIMIT 50
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_audit_entity_created
ON audit_log(tenant_id, entity_type, entity_id, created_at DESC);

-- Partial index for supplier sync worker: finds active suppliers due for sync
-- Supports supplier_sync_worker.go Run() query
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_suppliers_sync_active
ON suppliers(status, last_sync_at) WHERE status = 'active';
