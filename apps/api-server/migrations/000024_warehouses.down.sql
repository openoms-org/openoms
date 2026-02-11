ALTER TABLE shipments DROP COLUMN IF EXISTS warehouse_id;
DROP TABLE IF EXISTS warehouse_stock;
DROP TABLE IF EXISTS warehouses;
