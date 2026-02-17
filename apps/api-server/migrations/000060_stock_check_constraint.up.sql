-- Prevent negative stock quantities at the database level
ALTER TABLE warehouse_stock ADD CONSTRAINT chk_quantity_non_negative CHECK (quantity >= 0);
ALTER TABLE warehouse_stock ADD CONSTRAINT chk_reserved_non_negative CHECK (reserved >= 0);
