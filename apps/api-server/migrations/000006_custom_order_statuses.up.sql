-- Migration 000006: Convert order_status ENUM to TEXT for custom statuses
ALTER TABLE orders ALTER COLUMN status TYPE TEXT USING status::TEXT;
ALTER TABLE orders ALTER COLUMN status SET DEFAULT 'new';
DROP TYPE IF EXISTS order_status;
