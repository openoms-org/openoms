-- Migration 000015 down: Remove delivery fields from orders
DROP INDEX IF EXISTS idx_orders_delivery_method;
ALTER TABLE orders
    DROP COLUMN IF EXISTS delivery_method,
    DROP COLUMN IF EXISTS pickup_point_id;
