DROP INDEX IF EXISTS idx_shipments_order_package;
ALTER TABLE shipments DROP COLUMN IF EXISTS notes;
ALTER TABLE shipments DROP COLUMN IF EXISTS dimensions_height;
ALTER TABLE shipments DROP COLUMN IF EXISTS dimensions_width;
ALTER TABLE shipments DROP COLUMN IF EXISTS dimensions_length;
ALTER TABLE shipments DROP COLUMN IF EXISTS weight;
ALTER TABLE shipments DROP COLUMN IF EXISTS package_number;
