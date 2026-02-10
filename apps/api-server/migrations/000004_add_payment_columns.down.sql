-- Migration 000004 down: Remove payment columns from orders
ALTER TABLE orders DROP COLUMN paid_at, DROP COLUMN payment_method, DROP COLUMN payment_status;
