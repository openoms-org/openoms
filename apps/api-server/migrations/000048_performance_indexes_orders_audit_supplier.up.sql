-- Wave-0 additive performance indexes (performance audit 2026-06-12, OPE-541).
-- All three lead with tenant_id (the RLS predicate column) so they are RLS-correct
-- and cannot leak across tenants. Pure-additive; no existing index serves these paths.

-- Bound the lock wait and build time so an index build on the large orders table can
-- never hang production: fail fast instead (required for any orders-touching migration).
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '2min';

-- Unfiltered orders list (ORDER BY created_at DESC), the 30-day dashboard windows and
-- keyset pagination. idx_orders_tenant_status leads with status, so it cannot serve an
-- unfiltered created_at ordering; the trailing id is a stable keyset tiebreaker.
CREATE INDEX IF NOT EXISTS idx_orders_tenant_created ON public.orders USING btree (tenant_id, created_at DESC, id); -- migrate:index-lock-ok

-- Audit log default list page (ORDER BY created_at DESC) on the fastest-growing table.
-- idx_audit_entity leads with entity_type, so it cannot serve the unfiltered list sort.
CREATE INDEX IF NOT EXISTS idx_audit_log_tenant_created ON public.audit_log USING btree (tenant_id, created_at DESC); -- migrate:index-lock-ok

-- Products-list correlated supplier subquery (sp.product_id = p.id) and the reverse
-- product->supplier lookups. No existing supplier_products index includes product_id.
CREATE INDEX IF NOT EXISTS idx_supplier_products_product ON public.supplier_products USING btree (tenant_id, product_id); -- migrate:index-lock-ok
