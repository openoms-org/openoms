DROP INDEX IF EXISTS idx_suppliers_integration_id;
ALTER TABLE suppliers DROP COLUMN IF EXISTS integration_id;
