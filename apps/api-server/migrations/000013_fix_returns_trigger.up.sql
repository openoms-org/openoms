-- Add missing updated_at trigger for returns table
CREATE TRIGGER trigger_returns_updated_at
    BEFORE UPDATE ON returns
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- Fix RLS policy inconsistencies
ALTER TABLE webhook_deliveries FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS webhook_deliveries_tenant_isolation ON webhook_deliveries;
CREATE POLICY webhook_deliveries_tenant_isolation ON webhook_deliveries
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

ALTER TABLE returns FORCE ROW LEVEL SECURITY;
