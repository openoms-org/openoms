-- ============================================================
-- OpenOMS — Rollback migration 000001
-- Drops everything in reverse dependency order.
-- ============================================================

-- ============================================================
-- Revoke grants from application role
-- ============================================================

REVOKE ALL ON ALL TABLES IN SCHEMA public FROM openoms;
REVOKE USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public FROM openoms;
REVOKE EXECUTE ON ALL FUNCTIONS IN SCHEMA public FROM openoms;

-- ============================================================
-- Drop triggers
-- ============================================================

DROP TRIGGER IF EXISTS update_stock_sync_channels_updated_at ON stock_sync_channels;
DROP TRIGGER IF EXISTS update_stocktakes_updated_at ON stocktakes;
DROP TRIGGER IF EXISTS update_warehouse_documents_updated_at ON warehouse_documents;
DROP TRIGGER IF EXISTS update_warehouse_stock_updated_at ON warehouse_stock;
DROP TRIGGER IF EXISTS update_warehouses_updated_at ON warehouses;
DROP TRIGGER IF EXISTS update_roles_updated_at ON roles;
DROP TRIGGER IF EXISTS update_product_variants_updated_at ON product_variants;
DROP TRIGGER IF EXISTS update_product_bundles_updated_at ON product_bundles;
DROP TRIGGER IF EXISTS update_price_list_items_updated_at ON price_list_items;
DROP TRIGGER IF EXISTS update_price_lists_updated_at ON price_lists;
DROP TRIGGER IF EXISTS update_customers_updated_at ON customers;
DROP TRIGGER IF EXISTS set_updated_at ON automation_rules;
DROP TRIGGER IF EXISTS trigger_supplier_products_updated_at ON supplier_products;
DROP TRIGGER IF EXISTS trigger_suppliers_updated_at ON suppliers;
DROP TRIGGER IF EXISTS trigger_invoices_updated_at ON invoices;
DROP TRIGGER IF EXISTS trigger_returns_updated_at ON returns;
DROP TRIGGER IF EXISTS trigger_shipments_updated_at ON shipments;
DROP TRIGGER IF EXISTS trigger_product_categories_updated_at ON product_categories;
DROP TRIGGER IF EXISTS trigger_product_listings_updated_at ON product_listings;
DROP TRIGGER IF EXISTS trigger_products_updated_at ON products;
DROP TRIGGER IF EXISTS trigger_orders_updated_at ON orders;
DROP TRIGGER IF EXISTS trigger_integrations_updated_at ON integrations;
DROP TRIGGER IF EXISTS trigger_users_updated_at ON users;
DROP TRIGGER IF EXISTS trigger_tenants_updated_at ON tenants;

-- ============================================================
-- Drop tables (reverse dependency order)
-- ============================================================

-- Stock sync
DROP TABLE IF EXISTS stock_sync_events CASCADE;
DROP TABLE IF EXISTS stock_sync_channels CASCADE;

-- Listing sync
DROP TABLE IF EXISTS listing_sync_log CASCADE;
DROP TABLE IF EXISTS listing_sync_configs CASCADE;

-- Customer loyalty & segments
DROP TABLE IF EXISTS customer_loyalty CASCADE;
DROP TABLE IF EXISTS loyalty_programs CASCADE;
DROP TABLE IF EXISTS customer_segment_members CASCADE;
DROP TABLE IF EXISTS customer_segments CASCADE;

-- Repricing
DROP TABLE IF EXISTS repricing_log CASCADE;
DROP TABLE IF EXISTS repricing_rules CASCADE;

-- Recurring orders
DROP TABLE IF EXISTS recurring_order_items CASCADE;
DROP TABLE IF EXISTS recurring_orders CASCADE;

-- Supplier portal & messages
DROP TABLE IF EXISTS supplier_messages CASCADE;
DROP TABLE IF EXISTS supplier_portal_tokens CASCADE;

-- Dropship
DROP TABLE IF EXISTS dropship_order_items CASCADE;
DROP TABLE IF EXISTS dropship_orders CASCADE;

-- Pick & pack
DROP TABLE IF EXISTS pick_pack_items CASCADE;
DROP TABLE IF EXISTS pick_pack_sessions CASCADE;

-- Payment reconciliation
DROP TABLE IF EXISTS payment_transactions CASCADE;
DROP TABLE IF EXISTS payment_settlements CASCADE;

-- Purchase orders
DROP TABLE IF EXISTS purchase_order_items CASCADE;
DROP TABLE IF EXISTS purchase_orders CASCADE;

-- Invitations
DROP TABLE IF EXISTS invitations CASCADE;

-- Webhook deliveries & events
DROP TABLE IF EXISTS webhook_deliveries CASCADE;
DROP TABLE IF EXISTS webhook_events CASCADE;

-- Audit log (sequence dropped via CASCADE)
DROP TABLE IF EXISTS audit_log CASCADE;

-- Supplier category mappings & products
DROP TABLE IF EXISTS supplier_category_mappings CASCADE;
DROP TABLE IF EXISTS supplier_products CASCADE;

-- Sync jobs
DROP TABLE IF EXISTS sync_jobs CASCADE;

-- Exchange rates
DROP TABLE IF EXISTS exchange_rates CASCADE;

-- Automation
DROP TABLE IF EXISTS automation_delayed_actions CASCADE;
DROP TABLE IF EXISTS automation_rule_logs CASCADE;
DROP TABLE IF EXISTS automation_rules CASCADE;

-- Stocktakes
DROP TABLE IF EXISTS stocktake_items CASCADE;
DROP TABLE IF EXISTS stocktakes CASCADE;

-- Warehouse documents
DROP TABLE IF EXISTS warehouse_document_items CASCADE;
DROP TABLE IF EXISTS warehouse_documents CASCADE;

-- Warehouse stock
DROP TABLE IF EXISTS warehouse_stock CASCADE;

-- Warehouses
DROP TABLE IF EXISTS warehouses CASCADE;

-- Returns
DROP TABLE IF EXISTS returns CASCADE;

-- Invoices
DROP TABLE IF EXISTS invoices CASCADE;

-- Shipments
DROP TABLE IF EXISTS shipments CASCADE;

-- Order groups
DROP TABLE IF EXISTS order_groups CASCADE;

-- Orders
DROP TABLE IF EXISTS orders CASCADE;

-- Price list items
DROP TABLE IF EXISTS price_list_items CASCADE;

-- Product listings
DROP TABLE IF EXISTS product_listings CASCADE;

-- Product bundles
DROP TABLE IF EXISTS product_bundles CASCADE;

-- Product variants
DROP TABLE IF EXISTS product_variants CASCADE;

-- Products
DROP TABLE IF EXISTS products CASCADE;

-- Suppliers
DROP TABLE IF EXISTS suppliers CASCADE;

-- Customers
DROP TABLE IF EXISTS customers CASCADE;

-- Price lists
DROP TABLE IF EXISTS price_lists CASCADE;

-- Product categories
DROP TABLE IF EXISTS product_categories CASCADE;

-- Integrations
DROP TABLE IF EXISTS integrations CASCADE;

-- Users
DROP TABLE IF EXISTS users CASCADE;

-- Roles
DROP TABLE IF EXISTS roles CASCADE;

-- Tenants
DROP TABLE IF EXISTS tenants CASCADE;

-- ============================================================
-- Drop functions
-- ============================================================

DROP FUNCTION IF EXISTS public.use_invitation(text);
DROP FUNCTION IF EXISTS public.find_invitation_by_token(text);
DROP FUNCTION IF EXISTS public.find_return_by_token(text);
DROP FUNCTION IF EXISTS public.find_order_tenant_id(uuid);
DROP FUNCTION IF EXISTS public.find_user_for_auth(text, uuid);
DROP FUNCTION IF EXISTS public.find_tenant_by_slug(text);
DROP FUNCTION IF EXISTS public.update_updated_at_column();
DROP FUNCTION IF EXISTS public.update_updated_at();

-- ============================================================
-- Drop extensions
-- ============================================================

DROP EXTENSION IF EXISTS "uuid-ossp";
DROP EXTENSION IF EXISTS pgcrypto;
DROP EXTENSION IF EXISTS pg_trgm;
