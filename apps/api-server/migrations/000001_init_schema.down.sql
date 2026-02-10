-- OpenOMS — Rollback migration 000001
-- Drops everything in reverse dependency order.

-- ============================================================
-- Drop triggers
-- ============================================================
DROP TRIGGER IF EXISTS trigger_products_updated_at ON products;
DROP TRIGGER IF EXISTS trigger_shipments_updated_at ON shipments;
DROP TRIGGER IF EXISTS trigger_orders_updated_at ON orders;
DROP TRIGGER IF EXISTS trigger_integrations_updated_at ON integrations;
DROP TRIGGER IF EXISTS trigger_users_updated_at ON users;
DROP TRIGGER IF EXISTS trigger_tenants_updated_at ON tenants;

-- ============================================================
-- Drop trigger function
-- ============================================================
DROP FUNCTION IF EXISTS update_updated_at();

-- ============================================================
-- Drop tables (reverse dependency order)
-- ============================================================
DROP TABLE IF EXISTS audit_log CASCADE;
DROP TABLE IF EXISTS webhook_events CASCADE;
DROP TABLE IF EXISTS products CASCADE;
DROP TABLE IF EXISTS shipments CASCADE;
DROP TABLE IF EXISTS orders CASCADE;
DROP TABLE IF EXISTS integrations CASCADE;
DROP TABLE IF EXISTS users CASCADE;
DROP TABLE IF EXISTS tenants CASCADE;

-- ============================================================
-- Drop enum types
-- ============================================================
DROP TYPE IF EXISTS webhook_status;
DROP TYPE IF EXISTS shipment_status;
DROP TYPE IF EXISTS shipment_provider;
DROP TYPE IF EXISTS order_status;
DROP TYPE IF EXISTS order_source;
DROP TYPE IF EXISTS integration_status;
DROP TYPE IF EXISTS integration_provider;
DROP TYPE IF EXISTS user_role;
DROP TYPE IF EXISTS plan_type;

-- ============================================================
-- Revoke grants from application role
-- ============================================================
ALTER DEFAULT PRIVILEGES IN SCHEMA public REVOKE SELECT, INSERT, UPDATE, DELETE ON TABLES FROM openoms_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public REVOKE USAGE, SELECT ON SEQUENCES FROM openoms_app;
REVOKE ALL PRIVILEGES ON SCHEMA public FROM openoms_app;
REVOKE CONNECT ON DATABASE openoms FROM openoms_app;
