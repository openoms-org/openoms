ALTER TABLE warehouse_stock DROP CONSTRAINT IF EXISTS chk_quantity_non_negative;
ALTER TABLE warehouse_stock DROP CONSTRAINT IF EXISTS chk_reserved_non_negative;
