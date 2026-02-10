-- Migration 000004: Add payment columns to orders
ALTER TABLE orders
  ADD COLUMN payment_status VARCHAR(20) NOT NULL DEFAULT 'pending',
  ADD COLUMN payment_method VARCHAR(50),
  ADD COLUMN paid_at TIMESTAMPTZ;

-- Set realistic payment data on existing seed orders
UPDATE orders SET payment_status = 'refunded', payment_method = 'przelew', paid_at = ordered_at + INTERVAL '1 hour' WHERE status = 'refunded';
UPDATE orders SET payment_status = 'paid', payment_method = 'PayU', paid_at = ordered_at + INTERVAL '30 minutes' WHERE status IN ('delivered', 'completed', 'shipped', 'in_transit');
UPDATE orders SET payment_status = 'paid', payment_method = 'BLIK', paid_at = ordered_at + INTERVAL '15 minutes' WHERE status IN ('processing', 'ready_to_ship', 'confirmed');
UPDATE orders SET payment_status = 'pending' WHERE status = 'new';
UPDATE orders SET payment_status = 'failed' WHERE status = 'cancelled';
