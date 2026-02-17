-- Fix any existing negative stock values before adding constraints
UPDATE warehouse_stock SET quantity = 0 WHERE quantity < 0;
UPDATE warehouse_stock SET reserved = 0 WHERE reserved < 0;

-- Prevent negative stock quantities at the database level (IF NOT EXISTS via DO block)
DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_quantity_non_negative') THEN
    ALTER TABLE warehouse_stock ADD CONSTRAINT chk_quantity_non_negative CHECK (quantity >= 0);
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_reserved_non_negative') THEN
    ALTER TABLE warehouse_stock ADD CONSTRAINT chk_reserved_non_negative CHECK (reserved >= 0);
  END IF;
END $$;
