-- Migration 000015: Add delivery fields to orders
ALTER TABLE orders
    ADD COLUMN delivery_method TEXT,
    ADD COLUMN pickup_point_id TEXT;

CREATE INDEX idx_orders_delivery_method ON orders(tenant_id, delivery_method)
    WHERE delivery_method IS NOT NULL;
