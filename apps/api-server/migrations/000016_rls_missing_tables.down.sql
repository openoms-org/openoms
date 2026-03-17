-- Revert RLS policies added in 000016

-- Drop policies on newly protected tables
DROP POLICY IF EXISTS payment_settlements_tenant ON payment_settlements;
DROP POLICY IF EXISTS payment_transactions_tenant ON payment_transactions;
DROP POLICY IF EXISTS pick_pack_sessions_tenant ON pick_pack_sessions;
DROP POLICY IF EXISTS pick_pack_items_tenant ON pick_pack_items;
DROP POLICY IF EXISTS dropship_orders_tenant ON dropship_orders;
DROP POLICY IF EXISTS dropship_order_items_tenant ON dropship_order_items;
DROP POLICY IF EXISTS supplier_portal_tokens_tenant ON supplier_portal_tokens;
DROP POLICY IF EXISTS supplier_messages_tenant ON supplier_messages;
DROP POLICY IF EXISTS recurring_orders_tenant ON recurring_orders;
DROP POLICY IF EXISTS recurring_order_items_tenant ON recurring_order_items;
DROP POLICY IF EXISTS repricing_rules_tenant ON repricing_rules;
DROP POLICY IF EXISTS repricing_log_tenant ON repricing_log;
DROP POLICY IF EXISTS customer_segments_tenant ON customer_segments;
DROP POLICY IF EXISTS customer_segment_members_tenant ON customer_segment_members;
DROP POLICY IF EXISTS loyalty_programs_tenant ON loyalty_programs;
DROP POLICY IF EXISTS customer_loyalty_tenant ON customer_loyalty;
DROP POLICY IF EXISTS listing_sync_configs_tenant ON listing_sync_configs;
DROP POLICY IF EXISTS listing_sync_log_tenant ON listing_sync_log;
DROP POLICY IF EXISTS stock_sync_channels_tenant ON stock_sync_channels;
DROP POLICY IF EXISTS stock_sync_events_tenant ON stock_sync_events;

-- Disable RLS on those tables
ALTER TABLE payment_settlements DISABLE ROW LEVEL SECURITY;
ALTER TABLE payment_transactions DISABLE ROW LEVEL SECURITY;
ALTER TABLE pick_pack_sessions DISABLE ROW LEVEL SECURITY;
ALTER TABLE pick_pack_items DISABLE ROW LEVEL SECURITY;
ALTER TABLE dropship_orders DISABLE ROW LEVEL SECURITY;
ALTER TABLE dropship_order_items DISABLE ROW LEVEL SECURITY;
ALTER TABLE supplier_portal_tokens DISABLE ROW LEVEL SECURITY;
ALTER TABLE supplier_messages DISABLE ROW LEVEL SECURITY;
ALTER TABLE recurring_orders DISABLE ROW LEVEL SECURITY;
ALTER TABLE recurring_order_items DISABLE ROW LEVEL SECURITY;
ALTER TABLE repricing_rules DISABLE ROW LEVEL SECURITY;
ALTER TABLE repricing_log DISABLE ROW LEVEL SECURITY;
ALTER TABLE customer_segments DISABLE ROW LEVEL SECURITY;
ALTER TABLE customer_segment_members DISABLE ROW LEVEL SECURITY;
ALTER TABLE loyalty_programs DISABLE ROW LEVEL SECURITY;
ALTER TABLE customer_loyalty DISABLE ROW LEVEL SECURITY;
ALTER TABLE listing_sync_configs DISABLE ROW LEVEL SECURITY;
ALTER TABLE listing_sync_log DISABLE ROW LEVEL SECURITY;
ALTER TABLE stock_sync_channels DISABLE ROW LEVEL SECURITY;
ALTER TABLE stock_sync_events DISABLE ROW LEVEL SECURITY;

-- Restore broken COALESCE policy on stocktakes (revert to original)
DROP POLICY IF EXISTS stocktakes_tenant ON stocktakes;
CREATE POLICY stocktakes_tenant ON stocktakes
    USING (tenant_id = COALESCE(NULLIF(current_setting('app.current_tenant_id', true), ''), tenant_id)::uuid);

DROP POLICY IF EXISTS stocktake_items_tenant ON stocktake_items;
CREATE POLICY stocktake_items_tenant ON stocktake_items
    USING (tenant_id = COALESCE(NULLIF(current_setting('app.current_tenant_id', true), ''), tenant_id)::uuid);
