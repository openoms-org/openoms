ALTER TABLE products DROP COLUMN IF EXISTS dropship_supplier_id;
ALTER TABLE products DROP COLUMN IF EXISTS is_dropship;
DROP TABLE IF EXISTS dropship_order_items;
DROP TABLE IF EXISTS dropship_orders;
