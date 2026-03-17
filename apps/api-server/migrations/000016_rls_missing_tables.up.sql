-- Enable RLS on tenant-scoped tables that were missing it.
-- These tables have tenant_id FK but had no database-level isolation.

-- payment_settlements
ALTER TABLE payment_settlements ENABLE ROW LEVEL SECURITY;
ALTER TABLE payment_settlements FORCE ROW LEVEL SECURITY;
CREATE POLICY payment_settlements_tenant ON payment_settlements
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- payment_transactions
ALTER TABLE payment_transactions ENABLE ROW LEVEL SECURITY;
ALTER TABLE payment_transactions FORCE ROW LEVEL SECURITY;
CREATE POLICY payment_transactions_tenant ON payment_transactions
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- pick_pack_sessions
ALTER TABLE pick_pack_sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE pick_pack_sessions FORCE ROW LEVEL SECURITY;
CREATE POLICY pick_pack_sessions_tenant ON pick_pack_sessions
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- pick_pack_items
ALTER TABLE pick_pack_items ENABLE ROW LEVEL SECURITY;
ALTER TABLE pick_pack_items FORCE ROW LEVEL SECURITY;
CREATE POLICY pick_pack_items_tenant ON pick_pack_items
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- dropship_orders
ALTER TABLE dropship_orders ENABLE ROW LEVEL SECURITY;
ALTER TABLE dropship_orders FORCE ROW LEVEL SECURITY;
CREATE POLICY dropship_orders_tenant ON dropship_orders
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- dropship_order_items
ALTER TABLE dropship_order_items ENABLE ROW LEVEL SECURITY;
ALTER TABLE dropship_order_items FORCE ROW LEVEL SECURITY;
CREATE POLICY dropship_order_items_tenant ON dropship_order_items
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- supplier_portal_tokens
ALTER TABLE supplier_portal_tokens ENABLE ROW LEVEL SECURITY;
ALTER TABLE supplier_portal_tokens FORCE ROW LEVEL SECURITY;
CREATE POLICY supplier_portal_tokens_tenant ON supplier_portal_tokens
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- supplier_messages
ALTER TABLE supplier_messages ENABLE ROW LEVEL SECURITY;
ALTER TABLE supplier_messages FORCE ROW LEVEL SECURITY;
CREATE POLICY supplier_messages_tenant ON supplier_messages
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- recurring_orders
ALTER TABLE recurring_orders ENABLE ROW LEVEL SECURITY;
ALTER TABLE recurring_orders FORCE ROW LEVEL SECURITY;
CREATE POLICY recurring_orders_tenant ON recurring_orders
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- recurring_order_items
ALTER TABLE recurring_order_items ENABLE ROW LEVEL SECURITY;
ALTER TABLE recurring_order_items FORCE ROW LEVEL SECURITY;
CREATE POLICY recurring_order_items_tenant ON recurring_order_items
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- repricing_rules
ALTER TABLE repricing_rules ENABLE ROW LEVEL SECURITY;
ALTER TABLE repricing_rules FORCE ROW LEVEL SECURITY;
CREATE POLICY repricing_rules_tenant ON repricing_rules
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- repricing_log
ALTER TABLE repricing_log ENABLE ROW LEVEL SECURITY;
ALTER TABLE repricing_log FORCE ROW LEVEL SECURITY;
CREATE POLICY repricing_log_tenant ON repricing_log
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- customer_segments
ALTER TABLE customer_segments ENABLE ROW LEVEL SECURITY;
ALTER TABLE customer_segments FORCE ROW LEVEL SECURITY;
CREATE POLICY customer_segments_tenant ON customer_segments
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- customer_segment_members
ALTER TABLE customer_segment_members ENABLE ROW LEVEL SECURITY;
ALTER TABLE customer_segment_members FORCE ROW LEVEL SECURITY;
CREATE POLICY customer_segment_members_tenant ON customer_segment_members
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- loyalty_programs
ALTER TABLE loyalty_programs ENABLE ROW LEVEL SECURITY;
ALTER TABLE loyalty_programs FORCE ROW LEVEL SECURITY;
CREATE POLICY loyalty_programs_tenant ON loyalty_programs
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- customer_loyalty
ALTER TABLE customer_loyalty ENABLE ROW LEVEL SECURITY;
ALTER TABLE customer_loyalty FORCE ROW LEVEL SECURITY;
CREATE POLICY customer_loyalty_tenant ON customer_loyalty
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- listing_sync_configs
ALTER TABLE listing_sync_configs ENABLE ROW LEVEL SECURITY;
ALTER TABLE listing_sync_configs FORCE ROW LEVEL SECURITY;
CREATE POLICY listing_sync_configs_tenant ON listing_sync_configs
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- listing_sync_log
ALTER TABLE listing_sync_log ENABLE ROW LEVEL SECURITY;
ALTER TABLE listing_sync_log FORCE ROW LEVEL SECURITY;
CREATE POLICY listing_sync_log_tenant ON listing_sync_log
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- stock_sync_channels
ALTER TABLE stock_sync_channels ENABLE ROW LEVEL SECURITY;
ALTER TABLE stock_sync_channels FORCE ROW LEVEL SECURITY;
CREATE POLICY stock_sync_channels_tenant ON stock_sync_channels
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- stock_sync_events
ALTER TABLE stock_sync_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE stock_sync_events FORCE ROW LEVEL SECURITY;
CREATE POLICY stock_sync_events_tenant ON stock_sync_events
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- Fix broken RLS on stocktakes (OPE-87): COALESCE fallback was always-true
DROP POLICY IF EXISTS stocktakes_tenant ON stocktakes;
CREATE POLICY stocktakes_tenant ON stocktakes
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

DROP POLICY IF EXISTS stocktake_items_tenant ON stocktake_items;
CREATE POLICY stocktake_items_tenant ON stocktake_items
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- Grant permissions to app role (handles both Supabase and self-hosted)
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'openoms_app') THEN
        EXECUTE 'GRANT ALL ON ALL TABLES IN SCHEMA public TO openoms_app';
    END IF;
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'openoms') THEN
        EXECUTE 'GRANT ALL ON ALL TABLES IN SCHEMA public TO openoms';
    END IF;
END $$;
