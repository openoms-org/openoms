-- Migration 000006 down: Restore order_status ENUM
CREATE TYPE order_status AS ENUM (
    'new', 'confirmed', 'processing', 'ready_to_ship', 'shipped',
    'in_transit', 'delivered', 'completed', 'cancelled', 'refunded', 'on_hold'
);
ALTER TABLE orders ALTER COLUMN status TYPE order_status USING status::order_status;
ALTER TABLE orders ALTER COLUMN status SET DEFAULT 'new';
