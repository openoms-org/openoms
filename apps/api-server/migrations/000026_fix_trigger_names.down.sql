-- Revert to old trigger function names
DROP TRIGGER IF EXISTS update_product_variants_updated_at ON product_variants;
CREATE TRIGGER update_product_variants_updated_at
    BEFORE UPDATE ON product_variants
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_warehouses_updated_at ON warehouses;
CREATE TRIGGER update_warehouses_updated_at
    BEFORE UPDATE ON warehouses
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_warehouse_stock_updated_at ON warehouse_stock;
CREATE TRIGGER update_warehouse_stock_updated_at
    BEFORE UPDATE ON warehouse_stock
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_customers_updated_at ON customers;
CREATE TRIGGER update_customers_updated_at
    BEFORE UPDATE ON customers
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Revert FORCE ROW LEVEL SECURITY
ALTER TABLE automation_rules NO FORCE ROW LEVEL SECURITY;
ALTER TABLE automation_rule_logs NO FORCE ROW LEVEL SECURITY;
ALTER TABLE product_variants NO FORCE ROW LEVEL SECURITY;
ALTER TABLE warehouses NO FORCE ROW LEVEL SECURITY;
ALTER TABLE warehouse_stock NO FORCE ROW LEVEL SECURITY;
ALTER TABLE customers NO FORCE ROW LEVEL SECURITY;

-- Revert RLS policies to without true parameter
DROP POLICY IF EXISTS automation_rules_tenant_isolation ON automation_rules;
CREATE POLICY automation_rules_tenant_isolation ON automation_rules
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid);

DROP POLICY IF EXISTS automation_rule_logs_tenant_isolation ON automation_rule_logs;
CREATE POLICY automation_rule_logs_tenant_isolation ON automation_rule_logs
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid);

DROP POLICY IF EXISTS product_variants_tenant_isolation ON product_variants;
CREATE POLICY product_variants_tenant_isolation ON product_variants
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid);

DROP POLICY IF EXISTS warehouses_tenant_isolation ON warehouses;
CREATE POLICY warehouses_tenant_isolation ON warehouses
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid);

DROP POLICY IF EXISTS warehouse_stock_tenant_isolation ON warehouse_stock;
CREATE POLICY warehouse_stock_tenant_isolation ON warehouse_stock
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid);

DROP POLICY IF EXISTS customers_tenant_isolation ON customers;
CREATE POLICY customers_tenant_isolation ON customers
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid);

-- Revoke DELETE on webhook_deliveries
REVOKE DELETE ON webhook_deliveries FROM openoms_app;
