DROP INDEX IF EXISTS idx_shipments_external;
ALTER TABLE shipments DROP COLUMN IF EXISTS external_id;
