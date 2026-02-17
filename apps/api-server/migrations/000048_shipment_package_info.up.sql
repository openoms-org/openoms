-- Add multi-package support columns to shipments table.
-- Allows multiple packages per order with ordering, weight, dimensions, and notes.
ALTER TABLE shipments ADD COLUMN package_number INT NOT NULL DEFAULT 1;
ALTER TABLE shipments ADD COLUMN weight DECIMAL(10,3);
ALTER TABLE shipments ADD COLUMN dimensions_length DECIMAL(10,2);
ALTER TABLE shipments ADD COLUMN dimensions_width DECIMAL(10,2);
ALTER TABLE shipments ADD COLUMN dimensions_height DECIMAL(10,2);
ALTER TABLE shipments ADD COLUMN notes TEXT;

-- Index for efficient lookup of all shipments for an order, sorted by package number
CREATE INDEX idx_shipments_order_package ON shipments(order_id, package_number);
